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

import "strings"

// systemdUnitName maps a supervisord program name (e.g. "pmm-agent",
// "victoriametrics") to the native pfm systemd unit that replaces it.
func systemdUnitName(name string) string {
	return "pfm-" + name + ".service"
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
