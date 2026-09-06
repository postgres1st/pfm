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
	"io"
	"net/url"
)

// Errors returned by ResolveServerPassword.
var (
	// ErrServerPasswordTwice is returned when the password is supplied both in
	// --server-url and on stdin. Refused rather than silently preferring one: an
	// operator who passes both has a wrong belief about where their credential went.
	ErrServerPasswordTwice = errors.New("--server-password-stdin cannot be combined with a password in --server-url")

	// ErrNoServerURL is returned when --server-password-stdin is given with no
	// --server-url to attach the password to.
	ErrNoServerURL = errors.New("--server-password-stdin requires --server-url")

	// ErrNoServerUsername is returned when --server-url carries no username. There is
	// nothing to authenticate as, and url.UserPassword("", pw) would send an empty one.
	ErrNoServerUsername = errors.New("--server-url must include a username, e.g. https://admin@server-host/")

	// ErrEmptyServerPassword is returned when nothing arrives on stdin.
	ErrEmptyServerPassword = errors.New("no server password received on stdin")
)

// ResolveServerPassword moves the server credential off the command line.
//
// --server-url takes https://username:password@host/, so the Postgres1st WatchTower admin
// password appears in pfw-admin's own argv on every invocation -- readable by any
// local user through /proc/<pid>/cmdline, and written to shell history. With
// --server-password-stdin the URL carries only the username and the password arrives
// on stdin, so neither happens.
//
// It mutates ServerURL in place because every consumer reads the credential from
// there; keeping the password in a second field would mean auditing each one.
func ResolveServerPassword(g *GlobalFlags, stdin io.Reader) error {
	if !g.ServerPasswordStdin {
		return nil
	}

	if g.ServerURL == nil {
		return ErrNoServerURL
	}

	if g.ServerURL.User != nil {
		if _, ok := g.ServerURL.User.Password(); ok {
			return ErrServerPasswordTwice
		}
	}

	username := ""
	if g.ServerURL.User != nil {
		username = g.ServerURL.User.Username()
	}
	if username == "" {
		return ErrNoServerUsername
	}

	// Read only the first line and strip exactly one terminator, so `echo pw |` and
	// `printf '%s' pw |` behave identically; see commands.ReadSecretFrom, which this
	// mirrors deliberately rather than importing, to keep pkg/flags free of a
	// dependency on the commands package.
	data, err := io.ReadAll(io.LimitReader(stdin, 1<<16))
	if err != nil {
		return err
	}
	password := string(data)
	for i, r := range password {
		if r == '\r' || r == '\n' {
			password = password[:i]
			break
		}
	}
	if password == "" {
		return ErrEmptyServerPassword
	}

	g.ServerURL.User = url.UserPassword(username, password)

	return nil
}
