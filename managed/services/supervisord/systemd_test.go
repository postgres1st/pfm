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
	"fmt"
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
	// pfm-agent.service is the client package's unit (masked on a server), so the
	// server's self-monitoring agent maps to pfm-server-agent.service instead.
	assert.Equal(t, "pfm-server-agent.service", systemdUnitName("pmm-agent"))
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
		assert.Contains(t, env, "PMM_CLICKHOUSE_ADDR=127.0.0.1:9000") // address: bare (mid-line style)
		assert.Contains(t, env, `PMM_CLICKHOUSE_DATABASE="pmm"`)      // free-form value: envq-quoted
	})
	t.Run("grafana", func(t *testing.T) {
		t.Parallel()
		env := render(t, "grafana")
		assert.Contains(t, env, "PMM_POSTGRES_ADDR=127.0.0.1:5432") // address: bare
		assert.Contains(t, env, `GF_SERVER_DOMAIN="192.168.0.42:8443"`)
		// zero-egress: grafana phone-homes disabled via env (holds regardless
		// of the on-disk grafana.ini).
		assert.Contains(t, env, "GF_ANALYTICS_REPORTING_ENABLED=false")
		assert.Contains(t, env, "GF_ANALYTICS_CHECK_FOR_UPDATES=false")
		assert.Contains(t, env, "GF_SNAPSHOTS_EXTERNAL_ENABLED=false")
		assert.Contains(t, env, "GF_PLUGINS_PLUGIN_ADMIN_ENABLED=false")
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

func TestMarshalEnvConfigGrafanaHA(t *testing.T) {
	t.Parallel()

	vmParams, err := models.NewVictoriaMetricsParams(models.BasePrometheusConfigPath, models.VMBaseURL)
	require.NoError(t, err)
	pgParams := &models.PGParams{Addr: "127.0.0.1:5432", DBName: "postgres", DBUsername: "u", DBPassword: "p", SSLMode: "disable"}
	haParams := &models.HAParams{Enabled: true, GrafanaGossipPort: 9095, AdvertiseAddress: "10.0.0.5", Nodes: []string{"n1", "n2"}}
	s := New("/run/pfm", &models.Params{VMParams: vmParams, PGParams: pgParams, HAParams: haParams})
	settings := &models.Settings{DataRetention: 30 * 24 * time.Hour}

	env, err := s.marshalEnvConfig("grafana", settings)
	require.NoError(t, err)
	out := string(env)
	assert.Contains(t, out, "GF_UNIFIED_ALERTING_HA_LISTEN_ADDRESS=0.0.0.0:9095")
	assert.Contains(t, out, "GF_UNIFIED_ALERTING_HA_ADVERTISE_ADDRESS=10.0.0.5:9095")
	assert.Contains(t, out, "GF_UNIFIED_ALERTING_HA_PEERS=n1:9095,n2:9095")
}

func TestSanitizeEnvValues(t *testing.T) {
	t.Parallel()

	params := map[string]any{
		"S": "a\nPMM_BIND_ADDRESS=0.0.0.0",
		"L": []string{"--flag=x\r\ninjected", "--ok"},
		"I": 42,
	}
	sanitizeEnvValues(params)
	assert.Equal(t, "aPMM_BIND_ADDRESS=0.0.0.0", params["S"])
	assert.Equal(t, []string{"--flag=xinjected", "--ok"}, params["L"])
	assert.Equal(t, 42, params["I"]) // non-string values untouched
}

// fakeSystemctl writes a stub systemctl that appends its args to a log file and
// exits with exitCode. Returns the binary path and the log path.
func fakeSystemctl(t *testing.T, exitCode int) (bin, logPath string) {
	t.Helper()
	dir := t.TempDir()
	logPath = filepath.Join(dir, "calls.log")
	bin = filepath.Join(dir, "systemctl")
	script := fmt.Sprintf("#!/bin/sh\necho \"$@\" >> %q\nexit %d\n", logPath, exitCode)
	require.NoError(t, os.WriteFile(bin, []byte(script), 0o755))
	return bin, logPath
}

func newSystemdService(bin, envDir string) *Service {
	return &Service{pm: pmSystemd, systemctlPath: bin, envDir: envDir, l: logrus.WithField("component", "test")}
}

func TestReloadViaSystemd(t *testing.T) {
	t.Parallel()
	bin, logPath := fakeSystemctl(t, 0)
	s := newSystemdService(bin, t.TempDir())

	require.NoError(t, s.reloadViaSystemd("victoriametrics"))
	calls, err := os.ReadFile(logPath)
	require.NoError(t, err)
	// reset-failed must precede reload-or-restart, and use the mapped unit name.
	assert.Contains(t, string(calls), "reset-failed pfm-victoriametrics.service")
	assert.Contains(t, string(calls), "reload-or-restart pfm-victoriametrics.service")
}

func TestDisableDynamicService(t *testing.T) {
	// not parallel: overrides package-level pfmDataDir.
	saved := pfmDataDir
	t.Cleanup(func() { pfmDataDir = saved })

	t.Run("stop fails -> error, env kept, but marker already written", func(t *testing.T) {
		pfmDataDir = t.TempDir()
		bin, _ := fakeSystemctl(t, 1) // stop fails
		dir := t.TempDir()
		path := filepath.Join(dir, "victoriametrics.env")
		require.NoError(t, os.WriteFile(path, []byte("x"), 0o600))
		s := newSystemdService(bin, dir)
		require.Error(t, s.disableDynamicService("victoriametrics"))
		_, statErr := os.Stat(path)
		assert.NoError(t, statErr) // env not deleted when stop failed
		// The marker is written BEFORE the stop, so the unit stays down on the next
		// boot (ConditionPathExists) even though the immediate stop failed.
		_, mErr := os.Stat(disabledMarkerPath("victoriametrics"))
		assert.NoError(t, mErr)
	})

	t.Run("ok -> marker written, scoped stop (never mask), env removed", func(t *testing.T) {
		pfmDataDir = t.TempDir()
		bin, logPath := fakeSystemctl(t, 0)
		dir := t.TempDir()
		path := filepath.Join(dir, "victoriametrics.env")
		require.NoError(t, os.WriteFile(path, []byte("x"), 0o600))
		s := newSystemdService(bin, dir)
		require.NoError(t, s.disableDynamicService("victoriametrics"))
		calls, err := os.ReadFile(logPath)
		require.NoError(t, err)
		// Durable disable is a persistent marker + a SCOPED `stop` — never `mask`,
		// which would need the unscopeable manage-unit-files polkit action (a local
		// root-escalation vector via `link`).
		assert.Contains(t, string(calls), "stop pfm-victoriametrics.service")
		assert.NotContains(t, string(calls), "mask")
		_, mErr := os.Stat(disabledMarkerPath("victoriametrics"))
		assert.NoError(t, mErr) // marker present -> ConditionPathExists keeps it down
		_, statErr := os.Stat(path)
		assert.True(t, os.IsNotExist(statErr))
	})
}

func TestUpdateEnvConfigEmbeddedClearsMarker(t *testing.T) {
	// not parallel: overrides package-level pfmDataDir.
	saved := pfmDataDir
	t.Cleanup(func() { pfmDataDir = saved })
	pfmDataDir = t.TempDir()

	// A disable marker left by a prior external-VM config.
	marker := disabledMarkerPath("victoriametrics")
	require.NoError(t, os.MkdirAll(filepath.Dir(marker), 0o750))
	require.NoError(t, os.WriteFile(marker, nil, 0o644))

	vmParams, err := models.NewVictoriaMetricsParams(models.BasePrometheusConfigPath, models.VMBaseURL)
	require.NoError(t, err) // embedded VM (ExternalVM() == false)
	pgParams := &models.PGParams{Addr: "127.0.0.1:5432", DBName: "postgres", DBUsername: "u", DBPassword: "p", SSLMode: "disable"}
	dir := t.TempDir()
	bin, logPath := fakeSystemctl(t, 0)
	s := New(dir, &models.Params{VMParams: vmParams, PGParams: pgParams, HAParams: &models.HAParams{}})
	s.pm, s.systemctlPath, s.envDir = pmSystemd, bin, dir
	settings := &models.Settings{DataRetention: 30 * 24 * time.Hour}
	settings.VictoriaMetrics.CacheEnabled = new(false)

	require.NoError(t, s.updateEnvConfig("victoriametrics", settings))
	// Marker cleared so ConditionPathExists no longer blocks the embedded VM...
	_, mErr := os.Stat(marker)
	assert.True(t, os.IsNotExist(mErr))
	// ...the env is re-rendered, and it is started via a scoped reload (no unmask).
	_, eErr := os.Stat(filepath.Join(dir, "victoriametrics.env"))
	assert.NoError(t, eErr)
	calls, _ := os.ReadFile(logPath)
	assert.NotContains(t, string(calls), "unmask")
}

func TestUpdateEnvConfigReenableForcesRestartWhenEnvUnchanged(t *testing.T) {
	// not parallel: overrides package-level pfmDataDir.
	saved := pfmDataDir
	t.Cleanup(func() { pfmDataDir = saved })
	pfmDataDir = t.TempDir()

	vmParams, err := models.NewVictoriaMetricsParams(models.BasePrometheusConfigPath, models.VMBaseURL)
	require.NoError(t, err)
	pgParams := &models.PGParams{Addr: "127.0.0.1:5432", DBName: "postgres", DBUsername: "u", DBPassword: "p", SSLMode: "disable"}
	dir := t.TempDir()
	bin, logPath := fakeSystemctl(t, 0)
	s := New(dir, &models.Params{VMParams: vmParams, PGParams: pgParams, HAParams: &models.HAParams{}})
	s.pm, s.systemctlPath, s.envDir = pmSystemd, bin, dir
	settings := &models.Settings{DataRetention: 30 * 24 * time.Hour}
	settings.VictoriaMetrics.CacheEnabled = new(false)

	// Pre-seed the env file with exactly what the render will produce, so
	// saveEnvAndReload sees no change and issues no restart on its own...
	cfg, err := s.marshalEnvConfig("victoriametrics", settings)
	require.NoError(t, err)
	require.NoError(t, os.WriteFile(filepath.Join(dir, "victoriametrics.env"), cfg, 0o600))
	// ...and mark it disabled, as a prior external-VM config would.
	marker := disabledMarkerPath("victoriametrics")
	require.NoError(t, os.MkdirAll(filepath.Dir(marker), 0o750))
	require.NoError(t, os.WriteFile(marker, nil, 0o600))

	require.NoError(t, s.updateEnvConfig("victoriametrics", settings))
	_, mErr := os.Stat(marker)
	assert.True(t, os.IsNotExist(mErr)) // marker cleared
	// The unit was disabled and the env is byte-identical, so a restart must be
	// FORCED — otherwise re-enabling the embedded VM leaves it down until reboot.
	calls, _ := os.ReadFile(logPath)
	assert.Contains(t, string(calls), "reload-or-restart pfm-victoriametrics.service")
}

func TestEnvQuote(t *testing.T) {
	t.Parallel()

	// A rendered EnvironmentFile value must survive systemd's line-oriented
	// parser: a trailing backslash would otherwise continue onto (swallow) the
	// next KEY=value line, and a leading quote would truncate the value.
	for _, tc := range []struct {
		in, want string
	}{
		{"plain", `"plain"`},
		{`secret\`, `"secret\\"`}, // trailing backslash -> escaped, no line continuation
		{`pa"ss`, `"pa\"ss"`},     // embedded double quote -> escaped
		{`"lead`, `"\"lead"`},     // leading quote -> escaped, not a quote-span opener
		{"a\nb\r\nc", `"abc"`},    // CR/LF stripped (cannot appear inside a single line)
		{`c:\path\to\x`, `"c:\\path\\to\\x"`},
	} {
		assert.Equal(t, tc.want, envQuote(tc.in), "%q", tc.in)
	}
}

func TestMarshalEnvConfigHostilePassword(t *testing.T) {
	t.Parallel()

	// A password ending in a backslash (or containing a quote) must not break
	// out of its line and swallow the security-relevant SSL_* lines rendered
	// after it in the grafana template.
	vmParams, err := models.NewVictoriaMetricsParams(models.BasePrometheusConfigPath, models.VMBaseURL)
	require.NoError(t, err)
	pgParams := &models.PGParams{
		Addr: "127.0.0.1:5432", DBName: "postgres", DBUsername: "u",
		DBPassword: `p\`, SSLMode: "require", // trailing backslash
	}
	s := New("/run/pfm", &models.Params{VMParams: vmParams, PGParams: pgParams, HAParams: &models.HAParams{}})
	settings := &models.Settings{DataRetention: 30 * 24 * time.Hour}

	env, err := s.marshalEnvConfig("grafana", settings)
	require.NoError(t, err)
	out := string(env)
	// Password rendered as a self-contained, escaped, quoted value.
	assert.Contains(t, out, `PMM_POSTGRES_DBPASSWORD="p\\"`)
	// The line that follows it in the template is still present on its own line,
	// i.e. not consumed by a line continuation.
	assert.Contains(t, out, "\nPMM_POSTGRES_SSL_MODE=require\n")
}

func TestSystemctlSurfacesStderr(t *testing.T) {
	t.Parallel()

	// systemd's real failure reason goes to stderr; the error must include it.
	dir := t.TempDir()
	bin := filepath.Join(dir, "systemctl")
	script := "#!/bin/sh\necho 'Access denied' >&2\nexit 1\n"
	require.NoError(t, os.WriteFile(bin, []byte(script), 0o755))
	s := newSystemdService(bin, dir)

	_, err := s.systemctl("start", "pfm-victoriametrics.service")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "Access denied")
}

func TestSaveEnvAndReloadUnchanged(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	path := filepath.Join(dir, "vmproxy.env")
	require.NoError(t, os.WriteFile(path, []byte("SAME=1\n"), 0o600))
	// bogus systemctl path: if reload were attempted it would error.
	s := newSystemdService(filepath.Join(dir, "nonexistent-systemctl"), dir)

	changed, err := s.saveEnvAndReload("vmproxy", []byte("SAME=1\n"))
	require.NoError(t, err) // no reload attempted on the unchanged fast-path
	assert.False(t, changed)
}

func TestWriteEnvFileMode(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	path := filepath.Join(dir, "x.env")
	require.NoError(t, (&Service{}).writeEnvFile(path, []byte("K=v\n")))
	fi, err := os.Stat(path)
	require.NoError(t, err)
	assert.Equal(t, os.FileMode(0o600), fi.Mode().Perm())
}

func TestPfmUnitsInstalled(t *testing.T) {
	// not parallel: mutates the package-level pfmUnitPaths.
	saved := pfmUnitPaths
	t.Cleanup(func() { pfmUnitPaths = saved })

	target := filepath.Join(t.TempDir(), "pfm.target")
	pfmUnitPaths = []string{target}
	assert.False(t, pfmUnitsInstalled())
	require.NoError(t, os.WriteFile(target, []byte("x"), 0o644))
	assert.True(t, pfmUnitsInstalled())
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
