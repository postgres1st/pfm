// Copyright (C) 2026 Postgre First
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

package supervisord

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"
	"time"

	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/percona/pmm/managed/models"
)

func TestSystemdUnitName(t *testing.T) {
	t.Parallel()

	// supervisord program names map to native pfm-*.service units; the
	// redundant "pmm-" prefix on the control-plane services is collapsed.
	assert.Equal(t, "pfm-agent.service", systemdUnitName("pmm-agent"))
	assert.Equal(t, "pfm-managed.service", systemdUnitName("pmm-managed"))
	assert.Equal(t, "pfm-victoriametrics.service", systemdUnitName("victoriametrics"))
}

func TestSelectProcessManager(t *testing.T) {
	t.Parallel()

	for _, tc := range []struct {
		name             string
		env              string
		hasSupervisorctl bool
		hasSystemctl     bool
		hasPfmUnits      bool
		want             processManagerKind
	}{
		// explicit env wins over detection (operator asserts intent)
		{"env forces systemd", "systemd", true, false, false, pmSystemd},
		{"env forces supervisord", "supervisord", false, true, true, pmSupervisord},
		// unknown env value falls through to auto-detect
		{"garbage env -> auto", "wat", false, true, true, pmSystemd},
		// auto-detect needs POSITIVE evidence: supervisorctl absent, systemctl
		// present, AND the pfm unit set actually installed.
		{"auto systemd (units present)", "", false, true, true, pmSystemd},
		{"auto declines without pfm units", "", false, true, false, pmSupervisord},
		{"auto supervisord (both present)", "", true, true, true, pmSupervisord},
		{"auto default (neither)", "", false, false, false, pmSupervisord},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tc.want, selectProcessManager(tc.env, tc.hasSupervisorctl, tc.hasSystemctl, tc.hasPfmUnits))
		})
	}
}

func TestMarshalEnvConfig(t *testing.T) {
	t.Parallel()

	vmParams, err := models.NewVictoriaMetricsParams(models.BasePrometheusConfigPath, models.VMBaseURL)
	require.NoError(t, err)
	pgParams := &models.PGParams{Addr: "127.0.0.1:5432", DBName: "postgres", DBUsername: "db_username", DBPassword: "db_password", SSLMode: "disable"}
	s := New("/run/pfm", &models.Params{VMParams: vmParams, PGParams: pgParams, HAParams: &models.HAParams{}})
	settings := &models.Settings{DataRetention: 30 * 24 * time.Hour, PMMPublicAddress: "192.168.0.42:8443"}
	settings.VictoriaMetrics.CacheEnabled = new(false)

	render := func(t *testing.T, name string) string {
		t.Helper()
		b, err := s.marshalEnvConfig(name, settings)
		require.NoError(t, err)
		out := string(b)
		assert.Contains(t, out, "# Managed by pmm-managed. DO NOT EDIT.")
		return out
	}

	t.Run("victoriametrics", func(t *testing.T) {
		t.Parallel()
		env := render(t, "victoriametrics")
		assert.Contains(t, env, "VM_retentionPeriod=30d")
		assert.Contains(t, env, "VM_search_disableCache=true") // cache disabled -> disableCache true
	})
	t.Run("vmalert", func(t *testing.T) {
		t.Parallel()
		env := render(t, "vmalert")
		assert.Contains(t, env, "VMALERT_DATASOURCE_URL="+vmParams.URL())
		assert.Contains(t, env, "VMALERT_EXTRA_FLAGS=")
	})
	t.Run("vmproxy", func(t *testing.T) {
		t.Parallel()
		env := render(t, "vmproxy")
		assert.Contains(t, env, "VMPROXY_TARGET_URL="+vmParams.URL())
		assert.Contains(t, env, "PMM_BIND_ADDRESS=127.0.0.1") // InterfaceToBind default
	})
	t.Run("qan-api2", func(t *testing.T) {
		t.Parallel()
		env := render(t, "qan-api2")
		assert.Contains(t, env, "QANAPI_DATA_RETENTION=30")
		assert.Contains(t, env, "PMM_CLICKHOUSE_ADDR=127.0.0.1:9000")
		assert.Contains(t, env, "PMM_CLICKHOUSE_DATABASE=pmm")
	})
	t.Run("grafana", func(t *testing.T) {
		t.Parallel()
		env := render(t, "grafana")
		assert.Contains(t, env, "PMM_POSTGRES_ADDR=127.0.0.1:5432")
		assert.Contains(t, env, "GF_SERVER_DOMAIN=192.168.0.42:8443")
	})
	t.Run("unknown service errors", func(t *testing.T) {
		t.Parallel()
		_, err := s.marshalEnvConfig("does-not-exist", settings)
		assert.Error(t, err)
	})
}

func TestSaveEnvAndReloadRollback(t *testing.T) {
	t.Parallel()

	// A reload failure must not leave the new file committed: otherwise
	// change-detection sees it as "unchanged" next render and never retries.
	// Force reload to fail by pointing systemctl at `false` (exits non-zero).
	falsePath, err := exec.LookPath("false")
	require.NoError(t, err)

	newSvc := func(dir string) *Service {
		return &Service{pm: pmSystemd, systemctlPath: falsePath, envDir: dir, l: logrus.WithField("component", "test")}
	}

	t.Run("restores previous file", func(t *testing.T) {
		t.Parallel()
		dir := t.TempDir()
		s := newSvc(dir)
		path := filepath.Join(dir, "vmalert.env")
		require.NoError(t, os.WriteFile(path, []byte("OLD=1\n"), 0o600))

		changed, err := s.saveEnvAndReload("vmalert", []byte("NEW=2\n"))
		require.Error(t, err)    // reload failed
		assert.False(t, changed) // not successfully applied
		b, readErr := os.ReadFile(path)
		require.NoError(t, readErr)
		assert.Equal(t, "OLD=1\n", string(b)) // rolled back -> next render will retry
	})

	t.Run("removes first-render file", func(t *testing.T) {
		t.Parallel()
		dir := t.TempDir()
		s := newSvc(dir)

		_, err := s.saveEnvAndReload("vmproxy", []byte("NEW=2\n"))
		require.Error(t, err)
		_, statErr := os.Stat(filepath.Join(dir, "vmproxy.env"))
		assert.True(t, os.IsNotExist(statErr)) // removed -> next render will retry
	})
}

func TestProcessControlAvailable(t *testing.T) {
	t.Parallel()

	// Guards graceful degradation: each backend reports available only when its
	// own binary was found.
	assert.True(t, (&Service{pm: pmSystemd, systemctlPath: "/usr/bin/systemctl"}).processControlAvailable())
	assert.False(t, (&Service{pm: pmSystemd, systemctlPath: ""}).processControlAvailable())
	assert.True(t, (&Service{pm: pmSupervisord, supervisorctlPath: "/usr/bin/supervisorctl"}).processControlAvailable())
	assert.False(t, (&Service{pm: pmSupervisord, supervisorctlPath: ""}).processControlAvailable())
}

func TestParseIsActive(t *testing.T) {
	t.Parallel()

	// `systemctl is-active <unit>` prints one state word (plus trailing newline).
	// Mirrors parseStatus semantics: running -> true, definitely-not -> false,
	// undeterminable -> nil.
	for _, tc := range []struct {
		out  string
		want *bool
	}{
		{"active\n", new(true)},
		{"activating\n", new(true)},
		{"reloading\n", new(true)},
		{"deactivating\n", new(true)}, // still up, like supervisord STOPPING
		{"inactive\n", new(false)},
		{"failed\n", new(false)},
		{"unknown\n", nil},
		{"", nil},
	} {
		assert.Equal(t, tc.want, parseIsActive(tc.out), "%q", tc.out)
	}
}
