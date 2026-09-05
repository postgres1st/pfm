// Copyright (C) 2023 Percona LLC
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//  http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package tests

import (
	"database/sql"
	"net"
	"net/url"
	"strconv"
	"strings"
	"testing"

	_ "github.com/lib/pq" // register SQL driver
	"github.com/stretchr/testify/require"

	"github.com/percona/pmm/agent/utils/version"
)

const (
	defaultPostgresPort = 5432
	pgMaxIdleConns      = 10
)

// GetTestPostgreSQLDSN returns DNS for PostgreSQL test database.
func GetTestPostgreSQLDSN(tb testing.TB) string {
	tb.Helper()

	if testing.Short() {
		tb.Skip("-short flag is passed, skipping test with real database.")
	}
	q := make(url.Values)
	q.Set("sslmode", "disable") // TODO: make it configurable

	u := &url.URL{
		Scheme:   "postgres",
		Host:     net.JoinHostPort("localhost", strconv.Itoa(defaultPostgresPort)),
		Path:     "pmm-agent",
		User:     url.UserPassword("pmm-agent", "pmm-agent-password"),
		RawQuery: q.Encode(),
	}

	return u.String()
}

// OpenTestPostgreSQL opens connection to PostgreSQL test database.
func OpenTestPostgreSQL(tb testing.TB) *sql.DB {
	tb.Helper()

	db, err := sql.Open("postgres", GetTestPostgreSQLDSN(tb))
	require.NoError(tb, err)

	db.SetMaxIdleConns(pgMaxIdleConns)
	db.SetMaxOpenConns(10)
	db.SetConnMaxLifetime(0)

	waitForTestDataLoad(tb, db)

	return db
}

// PostgreSQLVersion returns PostgreSQL version components as major and minor/patch.
// For versions before 10, this is major and minor (e.g. "9" and "6").
// For versions 10 and above, this is major and patch level according to PostgreSQL's versioning scheme
// IsPerconaPostgreSQL reports whether the server is a Percona distribution rather than
// a stock PostgreSQL.
//
// It matters because Percona's build reports a different shared-buffer hit count for the
// same query -- 32 where stock reports 33 -- so tests asserting that metric need to know
// which they are talking to.
//
// This asks the SERVER, deliberately. The previous check read the POSTGRES_IMAGE env var
// and matched the string "perconalab", which fails two ways that both look like a code
// defect: the variable is unset unless the runner threads it through, and Percona also
// publishes under the "percona" org, which "perconalab" does not match. Either way the
// test expected 33, got 32, and blamed the wrong thing.
func IsPerconaPostgreSQL(tb testing.TB, db *sql.DB) bool {
	tb.Helper()

	var v string
	err := db.QueryRow("SELECT /* pmm-agent-tests:IsPerconaPostgreSQL */ version()").Scan(&v)
	require.NoError(tb, err)

	return strings.Contains(v, "Percona")
}

// (e.g. "10" and "", "18" and "2" for PostgreSQL 18.2).
func PostgreSQLVersion(tb testing.TB, db *sql.DB) (string, string) {
	tb.Helper()

	var v string
	err := db.QueryRow("SELECT /* pmm-agent-tests:PostgreSQLVersion */ version()").Scan(&v)
	require.NoError(tb, err)

	major, minor := version.ParsePostgreSQLVersion(v)
	require.NotEmpty(tb, major, "Failed to parse PostgreSQL version from %q.", v)
	tb.Logf("version = %q (major = %q, minor = %q)", v, major, minor)

	return major, minor
}
