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

// Package dbsecret resolves the pmm-managed PostgreSQL role password from the
// per-install secret store that pfw-init.sh writes on a native install.
//
// The password used to be a constant compiled into flag defaults, unit files and
// packaging. It is now generated per install into a 0600 EnvironmentFile, so the
// binaries that connect as that role have to be able to find it when the
// environment does not carry it. Both pfw-managed and pfw-encryption-rotation
// need exactly this, which is why it lives here rather than in either main.
package dbsecret

import (
	"os"
	"strings"
)

const (
	// File is the systemd EnvironmentFile that pfw-init.sh writes the generated
	// pmm-managed role password into. It is 0600 pfw:pfw inside a 0700 directory.
	File = "/srv/.pfw-secrets/managed-db.env" //nolint:gosec

	// Key is the variable name inside File. It matches the flag's Envar so the
	// same name works whether systemd exports it or we read the file ourselves.
	Key = "PMM_POSTGRES_DBPASSWORD"

	// Legacy is the constant shipped before credentials were generated. A host
	// provisioned by an older build still authenticates with it, and the container
	// image still uses it, so it stays as the last-resort fallback.
	Legacy = "pmm-managed"

	// ClickhouseFile is the EnvironmentFile the RPM %post writes the generated
	// ClickHouse credential into, at 0600 pfw:pfw. It is separate from File so
	// pfw-managed is the only unit holding both: pfw-qan-api2 and pfw-grafana
	// receive PMM_CLICKHOUSE_PASSWORD through pmm-managed's own render of
	// /run/pfw/*.env rather than mounting a secret of their own.
	ClickhouseFile = "/srv/.pfw-secrets/clickhouse.env" //nolint:gosec

	// ClickhouseKey is the variable name inside ClickhouseFile, matching the flag's
	// Envar so the same name works whether systemd exports it or we read the file.
	ClickhouseKey = "PMM_CLICKHOUSE_PASSWORD"

	// ClickhouseLegacy is the password whose sha256 the packaged users XML shipped
	// before this release. Kept as the fallback for the container image and for
	// hosts provisioned by an older build.
	ClickhouseLegacy = "clickhouse"
)

// ParseEnv returns the value of key in a systemd EnvironmentFile, or "" when the
// key is absent. Deliberately a parser rather than a shell `source`: the file is
// data, not code.
func ParseEnv(data []byte, key string) string {
	for _, line := range strings.Split(string(data), "\n") {
		v, ok := strings.CutPrefix(strings.TrimSpace(line), key+"=")
		if !ok {
			continue
		}
		// Exactly one matched pair, the way systemd's own parser treats it.
		// strings.Trim(v, `"`) would strip EVERY leading and trailing quote, so a
		// value that legitimately starts or ends with one comes back mangled and
		// then silently fails to authenticate.
		if len(v) >= 2 && strings.HasPrefix(v, `"`) && strings.HasSuffix(v, `"`) {
			v = v[1 : len(v)-1]
		}
		return v
	}
	return ""
}

// FromStore returns the password recorded in File, or "" when the file is absent,
// unreadable, or carries no value for Key. Callers that must have a password use
// ManagedDBPassword instead.
func FromStore() string {
	return readValue(File, Key)
}

// readValue reads one key out of one EnvironmentFile, returning "" for every
// failure mode. A missing store is the normal case in the container image, so it
// is not an error worth propagating.
func readValue(path, key string) string {
	data, err := os.ReadFile(path)
	if err != nil {
		return ""
	}
	return ParseEnv(data, key)
}

// ClickhousePassword returns the generated ClickHouse credential, falling back to
// the historical constant for the container image and for hosts with no store.
func ClickhousePassword() string {
	if v := readValue(ClickhouseFile, ClickhouseKey); v != "" {
		return v
	}
	return ClickhouseLegacy
}

// ManagedDBPassword returns the password from the store, falling back to the
// historical constant for hosts (and containers) that have no store.
func ManagedDBPassword() string {
	if v := FromStore(); v != "" {
		return v
	}
	return Legacy
}
