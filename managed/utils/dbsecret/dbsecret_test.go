// Copyright (C) 2023 Percona LLC
//
// This program is free software: you can redistribute it and/or modify
// it under the terms of the GNU Affero General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// This program is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
// GNU Affero General Public License for more details.
//
// You should have received a copy of the GNU Affero General Public License
// along with this program. If not, see <https://www.gnu.org/licenses/>.

package dbsecret

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParseEnv(t *testing.T) {
	t.Parallel()

	t.Run("reads the value", func(t *testing.T) {
		t.Parallel()
		assert.Equal(t, "deadbeef", ParseEnv([]byte("PMM_POSTGRES_DBPASSWORD=deadbeef\n"), Key))
	})

	t.Run("ignores other keys and blank lines", func(t *testing.T) {
		t.Parallel()
		in := []byte("\nGF_DATABASE_PASSWORD=other\nPMM_POSTGRES_DBPASSWORD=wanted\n")
		assert.Equal(t, "wanted", ParseEnv(in, Key))
	})

	// pmm-managed's own renderer wraps free-form values in double quotes
	// (systemd.go envQuote), so a hand-copied file may carry them.
	t.Run("strips surrounding double quotes", func(t *testing.T) {
		t.Parallel()
		assert.Equal(t, "abc", ParseEnv([]byte(`PMM_POSTGRES_DBPASSWORD="abc"`), Key))
	})

	// The regression M7 fixes: strings.Trim would eat both of these.
	t.Run("strips only one matched pair", func(t *testing.T) {
		t.Parallel()
		assert.Equal(t, `"abc"`, ParseEnv([]byte(`PMM_POSTGRES_DBPASSWORD=""abc""`), Key))
	})

	t.Run("keeps an unmatched quote", func(t *testing.T) {
		t.Parallel()
		assert.Equal(t, `"abc`, ParseEnv([]byte(`PMM_POSTGRES_DBPASSWORD="abc`), Key))
		assert.Equal(t, `abc"`, ParseEnv([]byte(`PMM_POSTGRES_DBPASSWORD=abc"`), Key))
	})

	// A lone `"` is one character, so there is no pair to strip.
	t.Run("single quote character is not a pair", func(t *testing.T) {
		t.Parallel()
		assert.Equal(t, `"`, ParseEnv([]byte(`PMM_POSTGRES_DBPASSWORD="`), Key))
	})

	t.Run("empty value stays empty", func(t *testing.T) {
		t.Parallel()
		assert.Empty(t, ParseEnv([]byte("PMM_POSTGRES_DBPASSWORD=\n"), Key))
	})

	t.Run("absent key returns empty", func(t *testing.T) {
		t.Parallel()
		assert.Empty(t, ParseEnv([]byte("OTHER=1\n"), Key))
	})
}

func TestClickhousePassword(t *testing.T) {
	t.Parallel()

	// The ClickHouse credential lives in its own file so pfw-grafana and
	// pfw-qan-api2 inherit it through pmm-managed's render rather than each
	// mounting the pmm-managed database password.
	t.Run("constants match what the packaging writes", func(t *testing.T) {
		t.Parallel()
		assert.Equal(t, "/srv/.pfw-secrets/clickhouse.env", ClickhouseFile)
		assert.Equal(t, "PMM_CLICKHOUSE_PASSWORD", ClickhouseKey)
		assert.Equal(t, "clickhouse", ClickhouseLegacy)
	})

	t.Run("parses the file the packaging writes", func(t *testing.T) {
		t.Parallel()
		assert.Equal(t, "deadbeef",
			ParseEnv([]byte("PMM_CLICKHOUSE_PASSWORD=deadbeef\n"), ClickhouseKey))
	})

	// Falls back rather than returning empty: the container image and any host
	// predating this release still use the shipped constant.
	//
	// The expectation is derived from the filesystem rather than assuming the store is
	// absent. Asserting the fallback unconditionally passes on a dev box and fails on
	// any PROVISIONED host, where the file exists and the fallback is correctly NOT
	// taken -- so the test failed precisely where the code was working.
	t.Run("returns the stored credential, or the legacy constant when there is none", func(t *testing.T) {
		t.Parallel()

		data, err := os.ReadFile(ClickhouseFile)
		if err != nil {
			assert.Equal(t, ClickhouseLegacy, ClickhousePassword(),
				"no store at %s, so the shipped constant must be used", ClickhouseFile)
			return
		}
		stored := ParseEnv(data, ClickhouseKey)
		require.NotEmpty(t, stored, "%s exists but holds no %s", ClickhouseFile, ClickhouseKey)
		assert.Equal(t, stored, ClickhousePassword(),
			"a store exists, so its value must win over the legacy constant")
		assert.NotEqual(t, ClickhouseLegacy, ClickhousePassword(),
			"a provisioned host must not fall back to the shipped constant")
	})
}
