// Copyright (C) 2023 Percona LLC
//
// This program is free software: you can redistribute it and/or modify
// it under the terms of the GNU Affero General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// This program is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
// GNU Affero General Public License for more details.
//
// You should have received a copy of the GNU Affero General Public License
// along with this program.  If not, see <https://www.gnu.org/licenses/>.

package models

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
)

// agentPasswordBytes is 128 bits. The value is a bearer credential for the
// exporter's basic auth, so it only has to resist guessing, not be memorable.
const agentPasswordBytes = 16

// generateAgentPassword returns a fresh random exporter credential as lowercase
// hex.
//
// Hex rather than base64 because the value has to survive three consumers
// unescaped: the exporter web-config YAML (models.BuildWebConfigFile), the
// VictoriaMetrics scrape config that carries the plaintext to the scraper, and
// the `HTTP_AUTH=pmm:<password>` environment variable still used for agents
// older than the web-config support (see agents.ensureAuthParams). Base64's
// `+/=` and any shell- or YAML-significant byte would need escaping in at least
// one of those.
//
// crypto/rand.Read never returns a short read without an error, so a nil error
// means len(b) bytes were filled.
func generateAgentPassword() (string, error) {
	b := make([]byte, agentPasswordBytes)
	if _, err := rand.Read(b); err != nil {
		return "", fmt.Errorf("failed to generate agent password: %w", err)
	}

	return hex.EncodeToString(b), nil
}
