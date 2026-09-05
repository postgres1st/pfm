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
	"io"
	"os"

	"github.com/sirupsen/logrus"

	"github.com/percona/pmm/admin/commands"
)

// StdinPasswordHelp is shared so every command describes the flag identically, and
// is registered as a kong variable in admin/cmd/bootstrap.go.
const StdinPasswordHelp = "Read the password from stdin instead of the command line, " +
	"where it is visible to any local user via /proc/<pid>/cmdline and is recorded in shell history"

// resolvePassword applies the precedence between the three ways a password can arrive,
// and warns when the credential was passed somewhere other people can read it.
//
// Precedence, and why: --password-stdin wins over --password because supplying both is
// a mistake worth refusing rather than silently resolving; --credentials-source is
// applied by the caller beforehand and is left alone, since a file the operator points
// at deliberately is not a leak.
//
// The warning is not merely advisory. On a default Linux host /proc/<pid>/cmdline is
// world-readable, so any local user can read the password of any registration in
// flight, and the value lands in the invoking shell's history file as well. That is
// the defect this flag exists to close, and an operator who keeps using --password
// should be told rather than left to assume the flag made it safe.
func resolvePassword(passwordStdin bool, password string, stdin io.Reader) (string, error) {
	if passwordStdin {
		if password != "" {
			return "", commands.ErrPasswordFlagAndStdin
		}

		return commands.ReadSecretFrom(stdin)
	}

	if password != "" {
		logrus.Warn("--password puts the credential in this process's command line, " +
			"where any local user can read it from /proc/<pid>/cmdline, and in your shell history. " +
			"Use --password-stdin or --credentials-source instead.")
	}

	return password, nil
}

// resolvePasswordStdin is the production entry point; tests call resolvePassword with
// their own reader.
func resolvePasswordStdin(passwordStdin bool, password string) (string, error) {
	return resolvePassword(passwordStdin, password, os.Stdin)
}
