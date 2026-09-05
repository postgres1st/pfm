// Copyright (C) 2023 Percona LLC
//
// This program is free software: you can redistribute it and/or modify
// it under the terms of the GNU Affero General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// This program is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
// GNU Affero General Public License for more details.
//
// You should have received a copy of the GNU Affero General Public License
// along with this program.  If not, see <https://www.gnu.org/licenses/>.

package models

import (
	"regexp"
	"testing"

	"github.com/AlekSi/pointer"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Internal test package: GenerateAgentPassword is unexported, and
// agent_model_test.go is package models_test, which cannot reach it.

func TestGenerateAgentPassword(t *testing.T) {
	t.Parallel()

	hex32 := regexp.MustCompile(`^[0-9a-f]{32}$`)

	t.Run("is 128 bits of hex", func(t *testing.T) {
		t.Parallel()
		pw, err := generateAgentPassword()
		require.NoError(t, err)
		assert.Regexp(t, hex32, pw)
	})

	// Hex rather than base64 so the value survives the exporter web-config YAML,
	// the VictoriaMetrics scrape config, and a systemd EnvironmentFile unescaped.
	t.Run("is unique per call", func(t *testing.T) {
		t.Parallel()
		seen := make(map[string]struct{}, 100)
		for range 100 {
			pw, err := generateAgentPassword()
			require.NoError(t, err)
			_, dup := seen[pw]
			require.False(t, dup, "generateAgentPassword returned a duplicate: %s", pw)
			seen[pw] = struct{}{}
		}
	})
}

func TestGetAgentPasswordDoesNotFallBackToAgentID(t *testing.T) {
	t.Parallel()

	// The whole point of P0.4: the agent ID is stored in inventory, returned by the
	// API and written to logs, so deriving the exporter password from it is
	// authentication in name only. A row with no stored password must yield "",
	// which callers treat as fail-closed -- never the agent ID.
	t.Run("empty when unset", func(t *testing.T) {
		t.Parallel()
		a := &Agent{AgentID: "/agent_id/00000000-0000-4000-8000-000000000001"}
		got := a.GetAgentPassword()
		assert.Empty(t, got)
		assert.NotEqual(t, a.AgentID, got)
	})

	t.Run("returns the stored password when set", func(t *testing.T) {
		t.Parallel()
		a := &Agent{
			AgentID:       "/agent_id/00000000-0000-4000-8000-000000000002",
			AgentPassword: pointer.ToString("2f8a1c0b9d4e6f7a0b1c2d3e4f5a6b7c"),
		}
		assert.Equal(t, "2f8a1c0b9d4e6f7a0b1c2d3e4f5a6b7c", a.GetAgentPassword())
	})
}

func TestBuildWebConfigFileFailsClosedOnEmptyPassword(t *testing.T) {
	t.Parallel()

	// Without this, an un-backfilled row would hash the empty string and the
	// exporter would come up accepting `pmm:` with no password at all -- strictly
	// worse than the agent-ID fallback this change removes.
	t.Run("empty password is an error", func(t *testing.T) {
		t.Parallel()
		a := &Agent{AgentID: "/agent_id/00000000-0000-4000-8000-000000000003"}
		_, err := a.BuildWebConfigFile()
		require.Error(t, err)
		assert.Contains(t, err.Error(), "no agent password")
	})

	t.Run("a set password still builds", func(t *testing.T) {
		t.Parallel()
		a := &Agent{
			AgentID:       "/agent_id/00000000-0000-4000-8000-000000000004",
			AgentPassword: pointer.ToString("2f8a1c0b9d4e6f7a0b1c2d3e4f5a6b7c"),
		}
		cfg, err := a.BuildWebConfigFile()
		require.NoError(t, err)
		assert.Contains(t, cfg, "basic_auth_users:")
		assert.Contains(t, cfg, "pmm: ")
		// The hash, never the plaintext.
		assert.NotContains(t, cfg, "2f8a1c0b9d4e6f7a0b1c2d3e4f5a6b7c")
	})
}

func TestEnsureAgentPassword(t *testing.T) {
	t.Parallel()

	hex32 := regexp.MustCompile(`^[0-9a-f]{32}$`)

	t.Run("generates when the caller supplied none", func(t *testing.T) {
		t.Parallel()
		row := &Agent{AgentID: "/agent_id/x"}
		require.NoError(t, ensureAgentPassword(row))
		require.NotNil(t, row.AgentPassword)
		assert.Regexp(t, hex32, *row.AgentPassword)
	})

	t.Run("preserves an operator-supplied password", func(t *testing.T) {
		t.Parallel()
		row := &Agent{AgentID: "/agent_id/x", AgentPassword: pointer.ToString("operator-chosen")}
		require.NoError(t, ensureAgentPassword(row))
		assert.Equal(t, "operator-chosen", *row.AgentPassword)
	})

	// An empty-string pointer is not the same as nil, and CreateNodeExporter's
	// signature takes *string straight from the API, so both reach here.
	t.Run("treats an empty pointer as unset", func(t *testing.T) {
		t.Parallel()
		row := &Agent{AgentID: "/agent_id/x", AgentPassword: pointer.ToString("")}
		require.NoError(t, ensureAgentPassword(row))
		assert.Regexp(t, hex32, *row.AgentPassword)
	})
}
