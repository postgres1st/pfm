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

package tests

import (
	"encoding/base64"
	"os"
	"os/user"
	"strings"
	"testing"
	"time"

	"github.com/brianvoe/gofakeit/v6"
	"github.com/stretchr/testify/require"
)

// GenEmail generates test user email.
func GenEmail(tb testing.TB) string {
	tb.Helper()
	u, err := user.Current()
	require.NoError(tb, err)

	hostname, err := os.Hostname()
	require.NoError(tb, err)

	return strings.Join([]string{u.Username, hostname, gofakeit.Email(), "test"}, ".")
}

// GenCredentials generates test user email and password.
func GenCredentials(tb testing.TB) (string, string) {
	tb.Helper()
	email := GenEmail(tb)
	password := gofakeit.Password(true, true, true, false, false, 14)
	return email, password
}

//nolint:gochecknoinits
func init() {
	gofakeit.Seed(time.Now().UnixNano())
}

// GrafanaAdminFile is the EnvironmentFile the RPM %post writes the generated Grafana
// admin password into. It is 0600 pfw:pfw, so a test only reads it when running as a
// user that can -- on any other host the fallback below applies.
const (
	GrafanaAdminFile = "/srv/.pfw-secrets/grafana.env" //nolint:gosec
	GrafanaAdminKey  = "GF_SECURITY_ADMIN_PASSWORD"
	grafanaAdminUser = "admin"
	grafanaAdminSeed = "admin"
)

// GrafanaAdminCredentials returns the credentials that actually authenticate against
// Grafana on THIS host.
//
// Hardcoding admin/admin was correct before credentials were generated at first boot,
// and is wrong on any provisioned install: the password is random per host, so the tests
// got 401 and read as broken while the server was working exactly as designed. Falls
// back to the seed password for the container image and for dev boxes with no store.
func GrafanaAdminCredentials(tb testing.TB) (string, string) {
	tb.Helper()

	data, err := os.ReadFile(GrafanaAdminFile)
	if err != nil {
		return grafanaAdminUser, grafanaAdminSeed
	}
	for _, line := range strings.Split(string(data), "\n") {
		if v, ok := strings.CutPrefix(strings.TrimSpace(line), GrafanaAdminKey+"="); ok && v != "" {
			return grafanaAdminUser, v
		}
	}
	return grafanaAdminUser, grafanaAdminSeed
}

// GrafanaAdminBasicAuth returns the value for an Authorization header carrying the
// credentials that authenticate on THIS host. A base64 literal of admin:admin is the
// same hardcoding as SetBasicAuth("admin","admin"), only harder to grep for.
func GrafanaAdminBasicAuth(tb testing.TB) string {
	tb.Helper()

	user, pass := GrafanaAdminCredentials(tb)
	return "Basic " + base64.StdEncoding.EncodeToString([]byte(user+":"+pass))
}
