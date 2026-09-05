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
	"net"
	"regexp"
	"testing"
	"time"
)

// dialTimeout is deliberately short. A missing service refuses the connection
// immediately; this bound only matters for an address that blackholes packets,
// where we would rather skip quickly than stall the suite.
const dialTimeout = 2 * time.Second

// hostPortRE matches the first host:port in a DSN, in any of the shapes the
// agent's tests use: tcp(127.0.0.1:3306), mongodb://127.0.0.1:27017 and bare
// 127.0.0.1:6379. It deliberately does not parse the DSN properly -- the goal is
// only to find something to dial.
var hostPortRE = regexp.MustCompile(`(\d{1,3}(?:\.\d{1,3}){3}|localhost):(\d{2,5})`)

// SkipIfUnreachable skips the test when nothing is listening on addr.
//
// PGF WatchTower monitors PostgreSQL. The MySQL, MongoDB and Valkey suites cover
// code paths the postgres-only gate refuses to reach in production, and they need
// service containers that `make -C agent env-up-db` starts. Without those, they
// failed with a wall of "connection refused" that buried real regressions: 35
// packages red, none of them a defect. A skip says "not exercised"; a failure
// claims "broken", and only one of those was ever true here.
//
// PostgreSQL is deliberately NOT covered by this helper. It is the database this
// product exists to monitor, so a missing PostgreSQL is a broken environment worth
// failing on, not a suite to wave through.
func SkipIfUnreachable(tb testing.TB, addr, service string) {
	tb.Helper()

	conn, err := net.DialTimeout("tcp", addr, dialTimeout)
	if err != nil {
		tb.Skipf("%s is not reachable at %s (%v); start it with `make -C agent env-up-db`", service, addr, err)
		return
	}
	conn.Close() //nolint:errcheck
}

// SkipIfDSNUnreachable extracts the first host:port from dsn and skips when nothing
// is listening there. Use it for table-driven cases that carry a DSN rather than a
// bare address. A dsn with no recognisable host:port is left alone: the test may not
// need a service at all, and skipping it would hide a real failure.
func SkipIfDSNUnreachable(tb testing.TB, dsn, service string) {
	tb.Helper()

	if m := hostPortRE.FindString(dsn); m != "" {
		SkipIfUnreachable(tb, m, service)
	}
}

