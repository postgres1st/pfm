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
	"os/exec"
	"strings"

	"golang.org/x/sys/unix"

	"github.com/percona/pmm/utils/pdeathsig"
)

// processManagerKind selects which backend launches/controls the service set.
type processManagerKind string

const (
	pmSupervisord processManagerKind = "supervisord"
	pmSystemd     processManagerKind = "systemd"

	// processManagerEnv, when set to a valid kind, forces the backend; any
	// other value falls through to auto-detection.
	processManagerEnv = "PMM_PROCESS_MANAGER"
)

// selectProcessManager decides the backend: an explicit, valid env value wins;
// otherwise auto-detect systemd only when supervisorctl is absent and systemctl
// is present (the shipped-RPM shape), defaulting to supervisord everywhere else.
func selectProcessManager(env string, hasSupervisorctl, hasSystemctl bool) processManagerKind {
	switch processManagerKind(env) {
	case pmSystemd:
		return pmSystemd
	case pmSupervisord:
		return pmSupervisord
	}
	if !hasSupervisorctl && hasSystemctl {
		return pmSystemd
	}
	return pmSupervisord
}

// systemdUnitName maps a supervisord program name (e.g. "pmm-agent",
// "victoriametrics") to the native pfm systemd unit that replaces it.
func systemdUnitName(name string) string {
	return "pfm-" + name + ".service"
}

// processControlAvailable reports whether the active backend can control
// services — the systemd analogue of the old `supervisorctlPath == ""` guard,
// used to degrade gracefully when the backend binary is absent.
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
