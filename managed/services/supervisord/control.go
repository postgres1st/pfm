// Copyright (C) 2023 Percona LLC
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//  http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package supervisord

import (
	"os"
	"os/exec"
	"strings"
)

// Status values callers pass to SupervisedServiceHasStatus. They are the
// supervisorctl spellings; the systemd branch maps them to is-active states.
const (
	StatusRunning = "RUNNING"
	StatusStopped = "STOPPED"
)

// This file exists for callers that need to control a supervised service WITHOUT
// constructing a full Service — standalone tools, chiefly pfw-encryption-rotation.
//
// They used to shell out to `supervisorctl` directly, which is correct for the
// container image and broken everywhere else: the native RPM install runs on
// systemd and ships no supervisorctl at all. pfw-encryption-rotation is shipped by
// pfw-managed.spec, so on a native host the key-rotation tool failed at its first
// step with "executable file not found". It failed BEFORE touching any data, so
// nothing was corrupted, but the feature simply did not work on the only platform
// beta2 ships.
//
// The backend choice deliberately reuses selectProcessManager rather than
// re-deriving it, so a standalone tool and the running server can never disagree
// about which process manager owns the services.

// currentProcessManager resolves the backend the same way New() does.
func currentProcessManager() (kind processManagerKind, supervisorctlPath, systemctlPath string) {
	supervisorctlPath, _ = exec.LookPath("supervisorctl")
	systemctlPath, _ = exec.LookPath("systemctl")
	kind = selectProcessManager(
		os.Getenv(processManagerEnv),
		supervisorctlPath != "",
		systemctlPath != "",
		pfwUnitsInstalled(),
	)
	return kind, supervisorctlPath, systemctlPath
}

// ControlSupervisedService runs "start" or "stop" against serviceName using whichever
// process manager this host actually runs. serviceName is the supervisord program name
// (for example "pmm-managed"); on systemd it is translated by systemdUnitName, which is
// what keeps the two backends addressing the same service.
func ControlSupervisedService(action, serviceName string) ([]byte, error) {
	pm, supervisorctlPath, systemctlPath := currentProcessManager()
	if pm == pmSystemd {
		//nolint:gosec,noctx // path comes from LookPath; unit name is derived, not user input
		return exec.Command(systemctlPath, action, systemdUnitName(serviceName)).CombinedOutput()
	}
	//nolint:gosec,noctx
	return exec.Command(supervisorctlPath, action, serviceName).CombinedOutput()
}

// SupervisedServiceHasStatus reports whether serviceName is CONFIRMED to be in the given
// state, where status is StatusRunning or StatusStopped.
//
// "Confirmed" is the important word. The systemd branch defers to parseIsActive, which
// already encodes the states carefully: activating, reloading and deactivating all count
// as still running, and anything unrecognised is undeterminable rather than a guess. An
// undeterminable state returns false for BOTH questions, so a caller polling for "stopped"
// keeps waiting and eventually times out instead of proceeding.
//
// That direction matters more than it looks. The first version of this function treated
// anything other than "active" as stopped, which reports a unit that is still
// DEACTIVATING as already down -- and the one caller uses this to decide when it is safe
// to start rewriting the database. Declaring the server stopped while it is still running
// is exactly the race that corrupts data.
//
// `systemctl is-active` exits non-zero for anything not active, so the exit status is
// deliberately ignored: it is information, not an error.
func SupervisedServiceHasStatus(serviceName, status string) bool {
	pm, supervisorctlPath, systemctlPath := currentProcessManager()
	if pm == pmSystemd {
		//nolint:gosec,noctx
		out, _ := exec.Command(systemctlPath, "is-active", systemdUnitName(serviceName)).CombinedOutput()
		return statusFromIsActive(string(out), status)
	}
	//nolint:gosec,noctx
	out, _ := exec.Command(supervisorctlPath, "status", serviceName).CombinedOutput()
	return strings.Contains(string(out), strings.ToUpper(status))
}

// statusFromIsActive maps `systemctl is-active` output to a confirmed yes/no for the
// asked-about status. Split out so it is testable without a live systemctl -- and so the
// test exercises THIS code rather than a copy of it, which would pass happily while the
// real path diverged.
func statusFromIsActive(isActiveOutput, status string) bool {
	running := parseIsActive(isActiveOutput)
	if running == nil {
		return false
	}
	if strings.EqualFold(status, StatusRunning) {
		return *running
	}
	return !*running
}
