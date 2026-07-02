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
	"errors"
	"fmt"
	"os"
	"os/exec"
	"strings"

	"golang.org/x/sys/unix"

	"github.com/percona/pmm/utils/pdeathsig"
)

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

// systemdUnitName maps a supervisord program name (e.g. "pmm-agent",
// "victoriametrics") to the native pfm systemd unit that replaces it.
func systemdUnitName(name string) string {
	return "pfm-" + name + ".service"
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
		return b, fmt.Errorf("%s failed: %w", cmdLine, err)
	}
	return b, nil
}

// reloadViaSystemd re-applies a regenerated config for a dynamic service.
// The static units are installed by the RPM, so no `daemon-reload` is needed
// per config change; reload-or-restart re-reads the unit's EnvironmentFile/
// config. NOTE: making pmm-managed's regenerated config land where the units
// read it is T3 — here we only swap the reload *trigger*.
func (s *Service) reloadViaSystemd(name string) error {
	_, err := s.systemctl("reload-or-restart", systemdUnitName(name))
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
