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

package commands

import (
	"errors"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestReadSecretFrom(t *testing.T) {
	t.Parallel()

	for _, tc := range []struct {
		name  string
		in    string
		want  string
		why   string
		isErr error
	}{
		{
			name: "bare value",
			in:   "s3cret",
			want: "s3cret",
			why:  "printf '%s' produces no trailing newline",
		},
		{
			name: "one trailing newline is stripped",
			in:   "s3cret\n",
			want: "s3cret",
			why:  "echo is what people reach for first; authenticating with a trailing \\n looks like a wrong password",
		},
		{
			name: "CRLF is stripped",
			in:   "s3cret\r\n",
			want: "s3cret",
			why:  "a password file authored on Windows must not authenticate with a stray \\r",
		},
		{
			name: "only the first line is taken",
			in:   "s3cret\nnot-part-of-it\n",
			want: "s3cret",
			why:  "a here-doc or trailing blank line must not smuggle more into the credential",
		},
		{
			name: "interior spaces are preserved",
			in:   "two words\n",
			want: "two words",
			why:  "a password may legitimately contain spaces; trimming would corrupt it",
		},
		{
			name: "leading whitespace is preserved",
			in:   "  padded\n",
			want: "  padded",
			why:  "same reason: only the line terminator is ours to remove",
		},
		{
			name:  "empty input is an error",
			in:    "",
			isErr: ErrEmptyStdinSecret,
			why:   "silently registering with no credential is the failure this flag exists to avoid",
		},
		{
			name:  "a lone newline is an error",
			in:    "\n",
			isErr: ErrEmptyStdinSecret,
			why:   "`echo | pfw-admin ...` must not be read as an empty password",
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			got, err := ReadSecretFrom(strings.NewReader(tc.in))

			if tc.isErr != nil {
				require.Error(t, err, tc.why)
				assert.True(t, errors.Is(err, tc.isErr), "expected %v, got %v", tc.isErr, err)
				assert.Empty(t, got, "no secret may be returned alongside an error")
				return
			}

			require.NoError(t, err, tc.why)
			assert.Equal(t, tc.want, got, tc.why)
		})
	}

	// Unbounded reads turn `--password-stdin < /dev/zero` into an OOM.
	t.Run("input is bounded", func(t *testing.T) {
		t.Parallel()

		got, err := ReadSecretFrom(strings.NewReader(strings.Repeat("a", maxStdinSecret*2)))

		require.NoError(t, err)
		assert.Len(t, got, maxStdinSecret, "read must stop at the limit rather than consuming everything")
	})
}
