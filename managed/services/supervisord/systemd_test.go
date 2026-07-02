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
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestSystemdUnitName(t *testing.T) {
	t.Parallel()

	// supervisord program names map to native pfm-*.service units.
	assert.Equal(t, "pfm-pmm-agent.service", systemdUnitName("pmm-agent"))
	assert.Equal(t, "pfm-victoriametrics.service", systemdUnitName("victoriametrics"))
}

func TestSelectProcessManager(t *testing.T) {
	t.Parallel()

	for _, tc := range []struct {
		name             string
		env              string
		hasSupervisorctl bool
		hasSystemctl     bool
		want             processManagerKind
	}{
		// explicit env wins over detection
		{"env forces systemd", "systemd", true, false, pmSystemd},
		{"env forces supervisord", "supervisord", false, true, pmSupervisord},
		// unknown env value falls through to auto-detect
		{"garbage env -> auto", "wat", false, true, pmSystemd},
		// auto-detect: systemd only when supervisorctl absent and systemctl present
		{"auto systemd", "", false, true, pmSystemd},
		{"auto supervisord (both present)", "", true, true, pmSupervisord},
		{"auto default (neither)", "", false, false, pmSupervisord},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tc.want, selectProcessManager(tc.env, tc.hasSupervisorctl, tc.hasSystemctl))
		})
	}
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
