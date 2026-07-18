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
	"bytes"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"text/template"

	"golang.org/x/sys/unix"

	"github.com/percona/pmm/managed/models"
	"github.com/percona/pmm/utils/pdeathsig"
)

// pfmEnvDir holds the per-service EnvironmentFiles pmm-managed renders for the
// systemd units. It is a pfm-owned tmpfs dir (created by tmpfiles.d), so the
// non-root pfm process can write it without touching root-owned systemd paths.
const pfmEnvDir = "/run/pfm"

// pfmUnitPaths are the locations the RPM installs pfm.target; their presence is
// the positive signal that the native systemd stack is actually deployed.
var pfmUnitPaths = []string{
	"/usr/lib/systemd/system/pfm.target",
	"/etc/systemd/system/pfm.target",
}

// processManagerKind selects which backend launches/controls the service set.
type processManagerKind string

const (
	pmSupervisord processManagerKind = "supervisord"
	pmSystemd     processManagerKind = "systemd"

	// processManagerEnv, when set to a valid kind, forces the backend; any
	// other value falls through to auto-detection.
	processManagerEnv = "PMM_PROCESS_MANAGER"
)

// selectProcessManager decides the backend: an explicit, valid env value wins
// (the operator asserts intent). Otherwise auto-detect requires POSITIVE
// evidence that the native stack is deployed — supervisorctl absent, systemctl
// present, AND the pfm unit set installed. Gating auto on the mere absence of
// supervisorctl would misfire on any dev/CI box or half-provisioned host,
// silently selecting a backend that can't control most services.
func selectProcessManager(env string, hasSupervisorctl, hasSystemctl, hasPfmUnits bool) processManagerKind {
	switch processManagerKind(env) {
	case pmSystemd:
		return pmSystemd
	case pmSupervisord:
		return pmSupervisord
	}
	if !hasSupervisorctl && hasSystemctl && hasPfmUnits {
		return pmSystemd
	}
	return pmSupervisord
}

// pfmUnitsInstalled reports whether pfm.target is present on disk — the
// positive signal used to gate auto-detection of the systemd backend.
func pfmUnitsInstalled() bool {
	for _, p := range pfmUnitPaths {
		if _, err := os.Stat(p); err == nil {
			return true
		}
	}
	return false
}

// systemdUnitName maps a supervisord program name to the native pfm systemd
// unit that replaces it: "victoriametrics" -> pfm-victoriametrics.service. The
// redundant "pmm-" prefix is dropped so the control-plane services read
// pfm-managed / pfm-agent (not pfm-pmm-managed), consistent with pfm-init.
func systemdUnitName(name string) string {
	return "pfm-" + strings.TrimPrefix(name, "pmm-") + ".service"
}

// envQuote renders v as a systemd EnvironmentFile value that survives the
// line-oriented parser verbatim: the value is wrapped in double quotes with
// backslash and double-quote escaped, and CR/LF stripped. Without this, a value
// ending in `\` is read as a line continuation and swallows the following
// KEY=value line (e.g. a password ending in `\` silently drops the SSL_* lines
// rendered after it), and a value beginning with a quote is truncated at the
// unbalanced quote. systemd strips the wrapping quotes, so the consumer receives
// the exact original bytes. Verified against systemd's EnvironmentFile parser.
// Apply only to free-form, settings-derived values that occupy a whole line;
// never to values interpolated mid-line (an address with a `:port` suffix, a
// numeric retention with a `d` suffix), where a wrapping quote would break the
// surrounding syntax.
func envQuote(v string) string {
	v = strings.NewReplacer("\r", "", "\n", "").Replace(v)
	v = strings.ReplaceAll(v, `\`, `\\`)
	v = strings.ReplaceAll(v, `"`, `\"`)
	return `"` + v + `"`
}

// envTemplates renders the systemd EnvironmentFile (KEY=value) for each dynamic
// service, using the same params as the supervisord templates. Only the
// settings-driven values live here; stable defaults are static args in the unit
// (see build/packages/config/pfm/*.service). The pfm-* units read these from
// /run/pfm/<name>.env. Keep the KEY names in lockstep with those units.
//
// Free-form settings values on their own line (passwords, names, SSL paths,
// server host) go through `envq` so a hostile value cannot break out of its
// line; see envQuote. Structured values (addresses, ports, retention, HA
// advertise) are left bare because they are interpolated mid-line.
var envTemplates = template.Must(template.New("").Funcs(template.FuncMap{"envq": envQuote}).Option("missingkey=error").Parse(`
{{define "victoriametrics"}}VM_retentionPeriod={{ .DataRetentionDays }}d
VM_search_disableCache={{ .VMSearchDisableCache }}
PMM_BIND_ADDRESS={{ .InterfaceToBind }}
{{end}}

{{define "vmalert"}}VMALERT_DATASOURCE_URL={{ .VMURL }}
VMALERT_EXTRA_FLAGS={{ range .VMAlertFlags }}{{ . }} {{ end }}
PMM_BIND_ADDRESS={{ .InterfaceToBind }}
{{end}}

{{define "vmproxy"}}VMPROXY_TARGET_URL={{ .VMURL }}
PMM_BIND_ADDRESS={{ .InterfaceToBind }}
{{end}}

{{define "qan-api2"}}QANAPI_DATA_RETENTION={{ .DataRetentionDays }}
PMM_CLICKHOUSE_ADDR={{ .ClickhouseAddr }}
PMM_CLICKHOUSE_DATABASE={{ .ClickhouseDatabase | envq }}
PMM_CLICKHOUSE_USER={{ .ClickhouseUser | envq }}
PMM_CLICKHOUSE_PASSWORD={{ .ClickhousePassword | envq }}
{{end}}

{{define "grafana"}}GF_ANALYTICS_REPORTING_ENABLED=false
GF_ANALYTICS_CHECK_FOR_UPDATES=false
GF_ANALYTICS_CHECK_FOR_PLUGIN_UPDATES=false
GF_NEWS_NEWS_FEED_ENABLED=false
GF_SNAPSHOTS_EXTERNAL_ENABLED=false
GF_PLUGINS_PLUGIN_ADMIN_ENABLED=false
{{ if .PMMServerHost }}GF_SERVER_DOMAIN={{ .PMMServerHost | envq }}
{{ end }}PMM_POSTGRES_ADDR={{ .PostgresAddr }}
PMM_POSTGRES_DBNAME={{ .PostgresDBName | envq }}
PMM_POSTGRES_USERNAME={{ .PostgresDBUsername | envq }}
PMM_POSTGRES_DBPASSWORD={{ .PostgresDBPassword | envq }}
PMM_POSTGRES_SSL_MODE={{ .PostgresSSLMode }}
PMM_POSTGRES_SSL_CA_PATH={{ .PostgresSSLCAPath | envq }}
PMM_POSTGRES_SSL_KEY_PATH={{ .PostgresSSLKeyPath | envq }}
PMM_POSTGRES_SSL_CERT_PATH={{ .PostgresSSLCertPath | envq }}
PMM_CLICKHOUSE_HOST={{ .ClickhouseHost }}
PMM_CLICKHOUSE_PORT={{ .ClickhousePort }}
PMM_CLICKHOUSE_USER={{ .ClickhouseUser | envq }}
PMM_CLICKHOUSE_PASSWORD={{ .ClickhousePassword | envq }}
{{- if .HAEnabled }}
GF_UNIFIED_ALERTING_HA_LISTEN_ADDRESS=0.0.0.0:{{ .GrafanaGossipPort }}
GF_UNIFIED_ALERTING_HA_ADVERTISE_ADDRESS={{ .HAAdvertiseAddress }}:{{ .GrafanaGossipPort }}
GF_UNIFIED_ALERTING_HA_PEERS={{ .HANodes }}
{{- end }}
{{end}}
`))

// marshalEnvConfig renders the systemd EnvironmentFile for a dynamic service,
// reusing the shared config params. The pfm-<name>.service unit reads the
// result from /run/pfm/<name>.env.
func (s *Service) marshalEnvConfig(name string, settings *models.Settings) ([]byte, error) {
	tmpl := envTemplates.Lookup(name)
	if tmpl == nil {
		return nil, fmt.Errorf("no systemd env template for service %q", name)
	}
	params, err := s.configParams(settings)
	if err != nil {
		return nil, err
	}
	// EnvironmentFile is line-oriented (KEY=value), so a newline in a rendered
	// value would inject an extra line (e.g. overriding PMM_BIND_ADDRESS). Strip
	// CR/LF from values before rendering. Safe to mutate: configParams returns a
	// fresh map per call.
	sanitizeEnvValues(params)

	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, params); err != nil {
		return nil, fmt.Errorf("failed to render env template %q: %w", name, err)
	}
	return append([]byte("# Managed by pmm-managed. DO NOT EDIT.\n"), buf.Bytes()...), nil
}

// sanitizeEnvValues strips CR/LF from string and []string params so a rendered
// value cannot break out of its KEY=value line in an EnvironmentFile.
func sanitizeEnvValues(params map[string]any) {
	r := strings.NewReplacer("\n", "", "\r", "")
	for k, v := range params {
		switch val := v.(type) {
		case string:
			params[k] = r.Replace(val)
		case []string:
			out := make([]string, len(val))
			for i, s := range val {
				out[i] = r.Replace(s)
			}
			params[k] = out
		}
	}
}

// processControlAvailable reports whether the active backend can control
// services — the systemd analogue of the old `supervisorctlPath == ""` guard,
// used to degrade gracefully when the backend binary is absent.
//
// Privilege model (committed target): pmm-managed stays non-root (User=pfm);
// authorization to run `systemctl` against pfm-*.service is granted by a
// shipped polkit rule, implemented alongside the SELinux policy (T6). Until
// then this only proves the binary exists, not that the caller is authorized —
// TODO(T6): replace with a real authorization probe (a benign systemctl/D-Bus
// call) so an unauthorized process degrades instead of erroring per call.
func (s *Service) processControlAvailable() bool {
	if s.pm == pmSystemd {
		return s.systemctlPath != ""
	}
	return s.supervisorctlPath != ""
}

// systemctl runs `systemctl <args...>`, mirroring the supervisorctl shim.
func (s *Service) systemctl(args ...string) ([]byte, error) {
	if s.systemctlPath == "" {
		return nil, errors.New("systemctl not found")
	}

	cmd := exec.Command(s.systemctlPath, args...) //nolint:gosec,noctx
	cmdLine := strings.Join(cmd.Args, " ")
	s.l.Debugf("Running %q...", cmdLine)
	pdeathsig.Set(cmd, unix.SIGKILL)
	b, err := cmd.Output()
	if err != nil {
		// systemd writes the actual reason (e.g. polkit "Access denied", "Unit
		// not loaded") to stderr, which Output() captures into ExitError.Stderr
		// but the default error string drops. Surface it — pre-polkit (T6)
		// authorization failures are otherwise undiagnosable.
		var exitErr *exec.ExitError
		if errors.As(err, &exitErr) && len(exitErr.Stderr) > 0 {
			return b, fmt.Errorf("%s failed: %w: %s", cmdLine, err, bytes.TrimSpace(exitErr.Stderr))
		}
		return b, fmt.Errorf("%s failed: %w", cmdLine, err)
	}
	return b, nil
}

// updateEnvConfig regenerates one dynamic service's EnvironmentFile under the
// systemd backend, mirroring the per-service special cases of UpdateConfiguration.
func (s *Service) updateEnvConfig(name string, settings *models.Settings) error {
	reenabled := false
	switch {
	case name == "nomad-server":
		// No pfm-nomad-server unit exists under the systemd backend yet. Surface
		// the gap rather than silently ignoring an enabled setting.
		if settings.IsNomadEnabled() {
			s.l.Warnf("Nomad is enabled but has no native systemd unit; it will not run under the systemd backend.")
		}
		return nil
	case name == "victoriametrics":
		if s.vmParams.ExternalVM() {
			// External VM: the embedded VM must not run.
			return s.disableDynamicService(name)
		}
		// Embedded VM selected: clear the disable marker a previous external-VM
		// config may have left, so the unit's ConditionPathExists no longer skips it.
		var err error
		if reenabled, err = removeIfExists(disabledMarkerPath(name)); err != nil {
			return fmt.Errorf("failed to clear disable marker for %s: %w", name, err)
		}
	}

	cfg, err := s.marshalEnvConfig(name, settings)
	if err != nil {
		return err
	}
	applied, err := s.saveEnvAndReload(name, cfg)
	if err != nil {
		return err
	}
	if reenabled && !applied {
		// The unit was disabled (ConditionPathExists-skipped, hence down) and the env
		// render came out byte-identical, so saveEnvAndReload issued no restart. Force
		// it up — clearing the marker alone only takes effect on the next (re)start.
		return s.reload(name)
	}
	return nil
}

// removeIfExists deletes path, reporting whether it was there. Absence is not an error.
func removeIfExists(path string) (bool, error) {
	switch err := os.Remove(path); {
	case err == nil:
		return true, nil
	case errors.Is(err, fs.ErrNotExist):
		return false, nil
	default:
		return false, err
	}
}

// pfmDataDir is the persistent data root the units live under (/srv). A package
// var so tests can redirect it; the disable marker must be here (persistent), not
// in the tmpfs env dir, or the disable would not survive a reboot.
var pfmDataDir = "/srv"

// disabledMarkerPath is the persistent flag whose PRESENCE makes the unit's
// `ConditionPathExists=!<marker>` skip it on every (re)start. It lives in a
// dedicated control dir — NOT inside the service's own datadir — so wiping a
// service's data (e.g. resetting metrics) can't silently drop the disable flag.
// Keep this in lockstep with the unit's ConditionPathExists= path.
func disabledMarkerPath(name string) string {
	return filepath.Join(pfmDataDir, ".pfm-disabled", name)
}

// disableDynamicService stops a dynamic unit and durably keeps it down when an
// embedded service is replaced by an external one (e.g. an external VM). A plain
// `stop` is not durable — the unit stays in pfm.target's static Wants= and
// pfm-tmpfiles re-seeds its EnvironmentFile every boot, so it would restart on
// reboot. Durability is provided by a persistent marker file the unit's
// `ConditionPathExists=!<marker>` checks, NOT by `systemctl mask`: masking needs
// the coarse manage-unit-files polkit action, which systemd exposes with no
// per-unit detail and so cannot be scoped to pfm-* — granting it would let a
// compromised pfm process `link` an attacker unit and start it as root. This path
// needs only a file write (pfm owns /srv) plus a scoped `stop` (manage-units).
//
// The marker is written BEFORE the stop so a crash in between still leaves the
// unit condition-blocked from restarting, never running-without-marker.
func (s *Service) disableDynamicService(name string) error {
	marker := disabledMarkerPath(name)
	if err := os.MkdirAll(filepath.Dir(marker), 0o750); err != nil { //nolint:mnd
		return err
	}
	if err := os.WriteFile(marker, nil, 0o600); err != nil { //nolint:mnd
		return err
	}
	if _, err := s.systemctl("stop", systemdUnitName(name)); err != nil {
		// Surface it rather than silently deleting the env file while the unit keeps
		// running; the marker already blocks the next restart and the next render retries.
		return fmt.Errorf("failed to stop %s: %w", systemdUnitName(name), err)
	}
	if err := os.Remove(filepath.Join(s.envDir, name+".env")); err != nil && !errors.Is(err, fs.ErrNotExist) {
		return err
	}
	return nil
}

// saveEnvAndReload writes a dynamic service's EnvironmentFile and, if it
// changed, asks systemd to re-apply it. Returns true if it was applied.
//
// On reload failure it rolls the file back to its previous content (mirroring
// saveConfigAndReload): otherwise the new file stays committed and the next
// render sees it as "unchanged" and never retries, leaving the unit stuck.
func (s *Service) saveEnvAndReload(name string, cfg []byte) (bool, error) {
	path := filepath.Join(s.envDir, name+".env")
	oldCfg, err := os.ReadFile(path) //nolint:gosec
	if errors.Is(err, fs.ErrNotExist) {
		oldCfg, err = nil, nil
	}
	if err != nil {
		return false, err
	}
	if bytes.Equal(cfg, oldCfg) {
		s.l.Infof("%s.env not changed, doing nothing.", name)
		return false, nil
	}
	if err := s.writeEnvFile(path, cfg); err != nil {
		return false, err
	}
	if err := s.reload(name); err != nil {
		// Roll back so the next render is detected as changed and retried.
		if restoreErr := s.restoreEnvFile(path, oldCfg); restoreErr != nil {
			s.l.Errorf("Failed to roll back %s.env: %s.", name, restoreErr)
		}
		return false, err
	}
	s.l.Infof("%s.env updated and reloaded.", name)
	return true, nil
}

// writeEnvFile writes cfg atomically at 0600. These files carry DB/ClickHouse
// passwords: a fresh temp file + rename means the secret content is never
// visible at a looser mode (WriteFile does not tighten an existing file) and
// readers never see a partial file. systemd's manager reads EnvironmentFile as
// root before dropping to User=pfm, so owner-only (pfm) is sufficient.
func (s *Service) writeEnvFile(path string, cfg []byte) error {
	tmp := path + ".tmp"
	_ = os.Remove(tmp)                                    // clear any stale temp so the create below starts fresh
	if err := os.WriteFile(tmp, cfg, 0o600); err != nil { //nolint:mnd
		return err
	}
	if err := os.Chmod(tmp, 0o600); err != nil { //nolint:mnd // guard a pre-existing looser temp
		_ = os.Remove(tmp)
		return err
	}
	if err := os.Rename(tmp, path); err != nil {
		_ = os.Remove(tmp)
		return err
	}
	return nil
}

// restoreEnvFile puts back the previous EnvironmentFile after a failed reload;
// if there was none, it removes the file so the next render is treated as new.
func (s *Service) restoreEnvFile(path string, oldCfg []byte) error {
	if oldCfg == nil {
		if err := os.Remove(path); err != nil && !errors.Is(err, fs.ErrNotExist) {
			return err
		}
		return nil
	}
	return s.writeEnvFile(path, oldCfg)
}

// reloadViaSystemd re-applies a regenerated config for a dynamic service.
// The static units are installed by the RPM, so no `daemon-reload` is needed
// per config change; reload-or-restart re-reads the unit's EnvironmentFile.
// reset-failed first: a unit that hit its start limit rejects reload-or-restart
// until reset (no-op when healthy).
func (s *Service) reloadViaSystemd(name string) error {
	unit := systemdUnitName(name)
	if _, err := s.systemctl("reset-failed", unit); err != nil {
		s.l.Debugf("reset-failed %s: %s", unit, err)
	}
	_, err := s.systemctl("reload-or-restart", unit)
	return err
}

// parseIsActive interprets `systemctl is-active <unit>` output as a running
// state, mirroring parseStatus: true if the unit is up (or coming up/going
// down), false if it is definitely not running, nil if undeterminable.
func parseIsActive(output string) *bool {
	switch strings.TrimSpace(output) {
	case "active", "activating", "reloading", "deactivating":
		return new(true)
	case "inactive", "failed":
		return new(false)
	default:
		return nil
	}
}
