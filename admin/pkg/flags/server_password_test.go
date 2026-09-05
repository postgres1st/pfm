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

package flags

import (
	"errors"
	"net/url"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func mustURL(t *testing.T, raw string) *url.URL {
	t.Helper()
	u, err := url.Parse(raw)
	require.NoError(t, err)
	return u
}

func TestResolveServerPassword(t *testing.T) {
	t.Parallel()

	t.Run("stdin supplies the password and the URL keeps the username", func(t *testing.T) {
		t.Parallel()
		g := &GlobalFlags{ServerURL: mustURL(t, "https://admin@1.2.3.4/"), ServerPasswordStdin: true}

		require.NoError(t, ResolveServerPassword(g, strings.NewReader("s3cr3t\n")))

		require.NotNil(t, g.ServerURL.User)
		assert.Equal(t, "admin", g.ServerURL.User.Username())
		pw, ok := g.ServerURL.User.Password()
		assert.True(t, ok)
		assert.Equal(t, "s3cr3t", pw)
	})

	// Supplying it twice means the operator is wrong about where their credential
	// went. Silently preferring one would leave that belief intact.
	t.Run("a password in the URL as well is refused", func(t *testing.T) {
		t.Parallel()
		g := &GlobalFlags{ServerURL: mustURL(t, "https://admin:inurl@1.2.3.4/"), ServerPasswordStdin: true}

		err := ResolveServerPassword(g, strings.NewReader("s3cr3t\n"))

		require.Error(t, err)
		assert.True(t, errors.Is(err, ErrServerPasswordTwice))
	})

	// url.UserPassword("", pw) would happily produce ":pw@host" and authenticate as
	// nobody, which fails in a way that looks like a wrong password.
	t.Run("a URL with no username is refused", func(t *testing.T) {
		t.Parallel()
		g := &GlobalFlags{ServerURL: mustURL(t, "https://1.2.3.4/"), ServerPasswordStdin: true}

		err := ResolveServerPassword(g, strings.NewReader("s3cr3t\n"))

		require.Error(t, err)
		assert.True(t, errors.Is(err, ErrNoServerUsername))
	})

	t.Run("empty stdin is an error, not an empty password", func(t *testing.T) {
		t.Parallel()
		g := &GlobalFlags{ServerURL: mustURL(t, "https://admin@1.2.3.4/"), ServerPasswordStdin: true}

		err := ResolveServerPassword(g, strings.NewReader(""))

		require.Error(t, err)
		assert.True(t, errors.Is(err, ErrEmptyServerPassword))
	})

	t.Run("no --server-url to attach to", func(t *testing.T) {
		t.Parallel()
		g := &GlobalFlags{ServerPasswordStdin: true}

		err := ResolveServerPassword(g, strings.NewReader("s3cr3t\n"))

		require.Error(t, err)
		assert.True(t, errors.Is(err, ErrNoServerURL))
	})

	// The overwhelmingly common path: the flag is not used at all. It must not read
	// stdin, or it would block any command run without a pipe.
	t.Run("without the flag nothing is read and the URL is untouched", func(t *testing.T) {
		t.Parallel()
		g := &GlobalFlags{ServerURL: mustURL(t, "https://admin:inurl@1.2.3.4/")}

		require.NoError(t, ResolveServerPassword(g, strings.NewReader("ignored")))

		pw, _ := g.ServerURL.User.Password()
		assert.Equal(t, "inurl", pw)
	})
}
