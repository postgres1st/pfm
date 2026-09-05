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

package agents

import (
	"strings"
	"testing"

	"github.com/AlekSi/pointer"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	agentv1 "github.com/percona/pmm/api/agent/v1"
	"github.com/percona/pmm/managed/models"
	"github.com/percona/pmm/version"
)

func requireNoDuplicateFlags(t *testing.T, flags []string) {
	t.Helper()
	s := make(map[string]struct{})
	for _, f := range flags {
		name := strings.Split(f, "=")[0]
		if after, ok := strings.CutPrefix(name, "--no-"); ok { // kingpin's --no-<name> disables --<name>
			name = "--" + after
		}
		if _, present := s[name]; present {
			assert.Failf(t, "flag (or --no- form) is already present", "%q", name)
		}
		s[name] = struct{}{}
	}
}

func TestPathsBaseForDifferentVersions(t *testing.T) {
	left := "{{"
	right := "}}"
	assert.Equal(t, "/opt/postgres1st/watchtower", pathsBase(version.MustParse("2.22.01"), left, right))
	assert.Equal(t, "{{ .paths_base }}", pathsBase(version.MustParse("2.23.0"), left, right))
	assert.Equal(t, "{{ .paths_base }}", pathsBase(version.MustParse("2.23.0-3-g7aa417c"), left, right))
	assert.Equal(t, "{{ .paths_base }}", pathsBase(version.MustParse("2.23.0-beta4"), left, right))
	assert.Equal(t, "{{ .paths_base }}", pathsBase(version.MustParse("2.23.0-rc1"), left, right))
}

func TestGetExporterListenAddress(t *testing.T) {
	// Hermetic: without this the suite's result depends on whether
	// PFW_EXPOSE_EXPORTERS happens to be set in the caller's environment, which
	// silently flips the default these subtests assert.
	exposeAllExportersByDefault = func() bool { return false }
	t.Cleanup(func() { exposeAllExportersByDefault = func() bool { return false } })

	t.Run("uses 127.0.0.1 in push mode", func(t *testing.T) {
		node := &models.Node{
			Address: "1.2.3.4",
		}
		exporter := &models.Agent{
			ExporterOptions: models.ExporterOptions{
				PushMetrics: true,
			},
		}

		assert.Equal(t, "127.0.0.1", getExporterListenAddress(node, exporter))
	})
	t.Run("exposes exporter address when enabled in push mode", func(t *testing.T) {
		node := &models.Node{
			Address: "1.2.3.4",
		}
		exporter := &models.Agent{
			ExporterOptions: models.ExporterOptions{
				ExposeExporter: true,
				PushMetrics:    true,
			},
		}

		assert.Equal(t, "0.0.0.0", getExporterListenAddress(node, exporter))
	})
	t.Run("exposes exporter address when enabled in pull mode", func(t *testing.T) {
		node := &models.Node{
			Address: "1.2.3.4",
		}
		exporter := &models.Agent{
			ExporterOptions: models.ExporterOptions{
				ExposeExporter: true,
				PushMetrics:    false,
			},
		}

		assert.Equal(t, "0.0.0.0", getExporterListenAddress(node, exporter))
	})
	// The default is loopback. Pull-mode scraping reaches the exporter at
	// node.Address:port (models.Agent.ExporterURL), so an operator who scrapes
	// across hosts must opt in with --expose-exporter, or set the fleet-wide
	// escape hatch below, or use push mode. This is deliberate: the previous
	// default published every exporter on every interface.
	t.Run("binds loopback by default in pull mode", func(t *testing.T) {
		node := &models.Node{
			Address: "1.2.3.4",
		}
		exporter := &models.Agent{
			ExporterOptions: models.ExporterOptions{
				PushMetrics: false,
			},
		}

		assert.Equal(t, "127.0.0.1", getExporterListenAddress(node, exporter))
	})
	t.Run("binds loopback by default when the node is unavailable", func(t *testing.T) {
		exporter := &models.Agent{
			ExporterOptions: models.ExporterOptions{
				PushMetrics: false,
			},
		}

		assert.Equal(t, "127.0.0.1", getExporterListenAddress(nil, exporter))
	})
	// The escape hatch exists because the failure mode is silent: metrics simply
	// stop arriving. A fleet already dark cannot be recovered with a per-service
	// flag, since that needs a working connection to re-register each service.
	t.Run("escape hatch restores the previous fleet-wide default", func(t *testing.T) {
		exposeAllExportersByDefault = func() bool { return true }
		t.Cleanup(func() { exposeAllExportersByDefault = func() bool { return false } })

		exporter := &models.Agent{
			ExporterOptions: models.ExporterOptions{
				PushMetrics: false,
			},
		}

		assert.Equal(t, "0.0.0.0", getExporterListenAddress(nil, exporter))
	})
	t.Run("push mode still wins over the escape hatch", func(t *testing.T) {
		exposeAllExportersByDefault = func() bool { return true }
		t.Cleanup(func() { exposeAllExportersByDefault = func() bool { return false } })

		exporter := &models.Agent{
			ExporterOptions: models.ExporterOptions{
				PushMetrics: true,
			},
		}

		assert.Equal(t, "127.0.0.1", getExporterListenAddress(nil, exporter))
	})
}

// This is the coverage whose absence let two call sites build `HTTP_AUTH=pmm:`
// with an empty password: the fixtures in the per-exporter tests all set a
// credential, so no test exercised the unset case on those paths.
func TestHTTPAuthFailsClosedWithoutACredential(t *testing.T) {
	t.Run("helper refuses an agent with no password", func(t *testing.T) {
		exporter := &models.Agent{AgentID: "agent-id"}

		got, err := httpAuthEnv(exporter)

		require.Error(t, err)
		require.ErrorContains(t, err, "no agent password")
		// Never emit the bare prefix: the exporter accepts `pmm:` with an empty
		// password, which is no authentication at all.
		require.NotEqual(t, "HTTP_AUTH=pmm:", got)
		require.Empty(t, got)
	})

	t.Run("helper builds the entry when a password is set", func(t *testing.T) {
		exporter := &models.Agent{
			AgentID:       "agent-id",
			AgentPassword: pointer.ToString("2f8a1c0b9d4e6f7a0b1c2d3e4f5a6b7c"),
		}

		got, err := httpAuthEnv(exporter)

		require.NoError(t, err)
		require.Equal(t, "HTTP_AUTH=pmm:2f8a1c0b9d4e6f7a0b1c2d3e4f5a6b7c", got)
		// The agent ID must never appear as the credential again.
		require.NotContains(t, got, exporter.AgentID)
	})
}

// resolveExposeExporters had no coverage at all: the tests overrode the resolved
// value, so the variable name, the parsing and the failure behaviour were never
// exercised. It is the documented recovery path from a fleet-wide metrics blackout,
// so the value an operator most plausibly typos must not take the server down.
func TestResolveExposeExporters(t *testing.T) {
	for _, tc := range []struct {
		raw  string
		want bool
		why  string
	}{
		{"", false, "unset-equivalent: `FOO=` in a unit file means blank, not invalid"},
		{"true", true, ""},
		{"TRUE", true, "case-insensitive"},
		{" true ", true, "EnvironmentFile lines carry stray whitespace"},
		{"1", true, ""},
		{"yes", true, "plausible spelling; used to panic"},
		{"on", true, "plausible spelling; used to panic"},
		{"false", false, ""},
		{"no", false, ""},
		{"off", false, ""},
		{"0", false, ""},
		{"bogus", false, "unparseable is reported and ignored, never fatal"},
	} {
		t.Run(tc.raw+" "+tc.why, func(t *testing.T) {
			t.Setenv(ExposeExportersEnvVar, tc.raw)
			// Must not panic for any input -- that is the whole point.
			require.NotPanics(t, func() {
				assert.Equal(t, tc.want, resolveExposeExporters(), tc.why)
			})
		})
	}
}

// TestExporterConfigsFailClosedWithoutACredential covers the CALL SITES, not the
// helper. Its absence is what let two builders construct `HTTP_AUTH=pmm:` inline and
// bypass the guard entirely: TestHTTPAuthFailsClosedWithoutACredential proves
// httpAuthEnv is correct and proves nothing about who calls it, and every
// per-exporter fixture sets a password, so the unset case was never reached.
//
// Driven as a table over every builder so a new exporter is a visibly missing row
// rather than silent absence of coverage.
func TestExporterConfigsFailClosedWithoutACredential(t *testing.T) {
	node := &models.Node{Address: "1.2.3.4"}
	// Old enough to take the legacy HTTP_AUTH branch in every builder that has one;
	// that branch is the one that used to emit an empty credential.
	legacy := version.MustParse("2.20.0")

	// No AgentPassword: the state a row predating generated credentials is in, and
	// the state backfillAgentPasswords exists to eliminate.
	newExporter := func(t models.AgentType) *models.Agent {
		return &models.Agent{
			AgentID:    "agent-id",
			AgentType:  t,
			ListenPort: pointer.ToUint16(12345),
			Username:   pointer.ToString("username"),
			Password:   pointer.ToString("password"),
		}
	}

	for _, tc := range []struct {
		name      string
		agentType models.AgentType
		call      func(*models.Agent) (*agentv1.SetStateRequest_AgentProcess, error)
	}{
		{"node_exporter", models.NodeExporterType, func(a *models.Agent) (*agentv1.SetStateRequest_AgentProcess, error) {
			return nodeExporterConfig(node, a, legacy)
		}},
		{"mysqld_exporter", models.MySQLdExporterType, func(a *models.Agent) (*agentv1.SetStateRequest_AgentProcess, error) {
			svc := &models.Service{ServiceID: "s", ServiceType: models.MySQLServiceType, Address: pointer.ToString("1.2.3.4"), Port: pointer.ToUint16(3306)}
			return mysqldExporterConfig(node, svc, a, redactSecrets, legacy)
		}},
		{"proxysql_exporter", models.ProxySQLExporterType, func(a *models.Agent) (*agentv1.SetStateRequest_AgentProcess, error) {
			svc := &models.Service{ServiceID: "s", ServiceType: models.ProxySQLServiceType, Address: pointer.ToString("1.2.3.4"), Port: pointer.ToUint16(6032)}
			return proxysqlExporterConfig(node, svc, a, redactSecrets, legacy)
		}},
		{"postgres_exporter", models.PostgresExporterType, func(a *models.Agent) (*agentv1.SetStateRequest_AgentProcess, error) {
			svc := &models.Service{ServiceID: "s", ServiceType: models.PostgreSQLServiceType, Address: pointer.ToString("1.2.3.4"), Port: pointer.ToUint16(5432), DatabaseName: "postgres"}
			return postgresExporterConfig(node, svc, a, redactSecrets, legacy)
		}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			cfg, err := tc.call(newExporter(tc.agentType))

			require.Error(t, err, "%s must refuse to build a config for an agent with no credential", tc.name)
			require.ErrorContains(t, err, "no agent password")
			// Nothing may be handed back: a partially built config could still be sent.
			require.Nil(t, cfg)
		})
	}
}
