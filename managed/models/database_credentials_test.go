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

// This file is package models (internal test), not models_test, because
// createUserSQL/alterUserPasswordSQL/grantAllOnDatabaseSQL are package-private
// pure functions unit-tested without a database. database_test.go next to this
// file is package models_test and cannot see unexported identifiers; the split
// mirrors the existing service_model_test.go (package models) alongside
// dsn_helpers_test.go (package models_test) in this same directory.
package models

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestCredentialSQLQuoting(t *testing.T) {
	t.Parallel()

	// A single quote closes the password literal when interpolated raw, which let
	// an operator-supplied PMM_POSTGRES_DBPASSWORD run arbitrary SQL as the
	// superuser -- initWithRoot connects as postgres.
	t.Run("password with a single quote is escaped", func(t *testing.T) {
		t.Parallel()
		assert.Equal(t,
			`CREATE USER "pmm-managed" LOGIN PASSWORD 'a''; DROP DATABASE postgres; --'`,
			createUserSQL("pmm-managed", "a'; DROP DATABASE postgres; --"))
	})

	// pq.QuoteLiteral switches to a C-style escape when the value contains a
	// backslash, and emits a LEADING SPACE before the E. Asserted so a future
	// refactor that hand-rolls the quoting has to reproduce it.
	t.Run("password with a backslash uses the E escape", func(t *testing.T) {
		t.Parallel()
		assert.Equal(t, `CREATE USER "u" LOGIN PASSWORD  E'a\\b'`, createUserSQL("u", `a\b`))
	})

	t.Run("identifier with a double quote is escaped", func(t *testing.T) {
		t.Parallel()
		assert.Equal(t, `CREATE USER "we""ird" LOGIN PASSWORD 'p'`, createUserSQL(`we"ird`, "p"))
	})

	// GRANT is DDL: PostgreSQL does not accept bind parameters in place of
	// identifiers, so the previous `GRANT ... ON DATABASE $1 TO $2` was a syntax
	// error every time it was reached.
	t.Run("grant interpolates identifiers, not placeholders", func(t *testing.T) {
		t.Parallel()
		got := grantAllOnDatabaseSQL("pmm-managed", "pmm-managed")
		assert.Equal(t, `GRANT ALL PRIVILEGES ON DATABASE "pmm-managed" TO "pmm-managed"`, got)
		assert.NotContains(t, got, "$1")
	})

	t.Run("alter user escapes the password", func(t *testing.T) {
		t.Parallel()
		assert.Equal(t, `ALTER USER "u" WITH PASSWORD 'a''b'`, alterUserPasswordSQL("u", "a'b"))
	})

	// params.Name is operator-controlled (--postgres-name / PMM_POSTGRES_DBNAME,
	// main.go:697). db.Exec on lib/pq sends a simple-query message, and the simple
	// query protocol allows ';'-separated statements, so a '"' in the name escapes
	// the identifier and injects SQL as the postgres superuser initWithRoot connects
	// as. pfw-init.sh:84-86 guards the identical call with an allowlist regex.
	t.Run("database name with a double quote is escaped", func(t *testing.T) {
		t.Parallel()
		assert.Equal(t, `CREATE DATABASE "ev""il"`, createDatabaseSQL(`ev"il`))
	})
}
