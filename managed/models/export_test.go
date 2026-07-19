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

package models

// SetSupportedServiceTypesForTests narrows or widens the service type allowlist and returns
// a function restoring the previous value. Tests using it must not run in parallel with each
// other, as the allowlist is package-level state.
func SetSupportedServiceTypesForTests(types ...ServiceType) func() {
	previous := supportedServiceTypes
	supportedServiceTypes = serviceTypeSet(types)
	return func() { supportedServiceTypes = previous }
}
