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
	"fmt"
	"io"
	"strings"
)

// ErrEmptyStdinSecret is returned when --password-stdin is given but nothing arrives
// on stdin. Treated as an error rather than an empty password: silently registering a
// service with no credential is the failure mode this flag exists to avoid.
var ErrEmptyStdinSecret = errors.New("no password received on stdin")

// ErrPasswordFlagAndStdin is returned when a password is supplied twice.
var ErrPasswordFlagAndStdin = errors.New("--password and --password-stdin are mutually exclusive")

// ErrBothStdinFlags is returned when two flags each want to read stdin. There is only
// one, so the second would read nothing and report an empty password rather than the
// real mistake.
var ErrBothStdinFlags = errors.New("--password-stdin and --server-password-stdin cannot be used together: " +
	"both read stdin. Use --credentials-source for the service password, or put the server password in --server-url")

// maxStdinSecret bounds how much is read before giving up. A password is not
// megabytes; without a bound, `pfw-admin add postgresql --password-stdin < /dev/zero`
// consumes memory until the process dies.
const maxStdinSecret = 1 << 16

// ReadSecretFrom reads a single secret from r.
//
// Exactly one trailing newline is stripped, so both `printf '%s' pw |` and
// `echo pw |` produce the same value -- `echo` is what people reach for first, and
// silently authenticating with "pw\n" would fail in a way that looks like a wrong
// password. Interior whitespace is preserved: a password may legitimately contain
// spaces, and trimming them would corrupt it.
//
// Only the first line is taken. A here-doc or a file with a trailing blank line would
// otherwise smuggle the remainder into the credential.
func ReadSecretFrom(r io.Reader) (string, error) {
	data, err := io.ReadAll(io.LimitReader(r, maxStdinSecret))
	if err != nil {
		return "", fmt.Errorf("failed to read password from stdin: %w", err)
	}

	secret := string(data)
	if i := strings.IndexAny(secret, "\r\n"); i >= 0 {
		secret = secret[:i]
	}

	if secret == "" {
		return "", ErrEmptyStdinSecret
	}

	return secret, nil
}
