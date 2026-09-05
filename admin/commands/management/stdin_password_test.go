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

package management

import (
	"errors"
	"strings"
	"testing"

	"github.com/sirupsen/logrus"
	logrustest "github.com/sirupsen/logrus/hooks/test"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/percona/pmm/admin/commands"
)

func TestResolvePassword(t *testing.T) {
	t.Run("stdin supplies the password", func(t *testing.T) {
		got, err := resolvePassword(true, "", strings.NewReader("from-stdin\n"))

		require.NoError(t, err)
		assert.Equal(t, "from-stdin", got)
	})

	// Supplying both is a mistake worth refusing. Silently preferring one would
	// leave an operator believing the credential never reached argv when it did.
	t.Run("both sources is an error, not a silent preference", func(t *testing.T) {
		got, err := resolvePassword(true, "from-flag", strings.NewReader("from-stdin\n"))

		require.Error(t, err)
		assert.True(t, errors.Is(err, commands.ErrPasswordFlagAndStdin))
		assert.Empty(t, got)
	})

	t.Run("an empty stdin is an error rather than an empty password", func(t *testing.T) {
		got, err := resolvePassword(true, "", strings.NewReader(""))

		require.Error(t, err)
		assert.True(t, errors.Is(err, commands.ErrEmptyStdinSecret))
		assert.Empty(t, got)
	})

	t.Run("the flag still works, and warns", func(t *testing.T) {
		hook := logrustest.NewGlobal()
		t.Cleanup(hook.Reset)

		got, err := resolvePassword(false, "from-flag", strings.NewReader(""))

		require.NoError(t, err)
		assert.Equal(t, "from-flag", got, "--password must keep working; this is a deprecation, not a removal")

		require.Len(t, hook.Entries, 1, "using --password must warn")
		assert.Equal(t, logrus.WarnLevel, hook.LastEntry().Level)
		// The warning has to say WHY, or it reads as noise and gets ignored.
		assert.Contains(t, hook.LastEntry().Message, "/proc/")
		assert.Contains(t, hook.LastEntry().Message, "--password-stdin")
	})

	// The common case: neither flag given, because the password came from
	// --credentials-source, which the caller resolved first. Warning here would fire
	// on the very path we are steering people towards.
	t.Run("no password given warns about nothing", func(t *testing.T) {
		hook := logrustest.NewGlobal()
		t.Cleanup(hook.Reset)

		got, err := resolvePassword(false, "", strings.NewReader(""))

		require.NoError(t, err)
		assert.Empty(t, got)
		assert.Empty(t, hook.Entries, "an unset --password must not warn")
	})
}
