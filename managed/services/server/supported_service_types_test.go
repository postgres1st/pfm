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

package server

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/percona/pmm/managed/models"
)

// expectedSupportedServiceTypeStrings derives the wanted value from the allowlist itself so
// the assertion tracks the shipped default rather than duplicating the literal list.
func expectedSupportedServiceTypeStrings() []string {
	types := models.SupportedServiceTypes()
	res := make([]string, len(types))
	for i, t := range types {
		res[i] = string(t)
	}
	return res
}

func TestReadOnlySettingsExposesSupportedServiceTypes(t *testing.T) {
	t.Parallel()

	// convertReadOnlySettings reads only its settings argument, so a zero-value Server is enough.
	var s Server
	res := s.convertReadOnlySettings(&models.Settings{})

	assert.NotEmpty(t, res.SupportedServiceTypes, "the allowlist must reach the all-roles endpoint")
	assert.Equal(t, expectedSupportedServiceTypeStrings(), res.SupportedServiceTypes)
}
