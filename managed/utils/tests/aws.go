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
	"net"
	"os"
	"testing"
	"time"
)

// awsProbeAddr is the endpoint the RDS tests ultimately talk to. Dialling it is a
// cheap proxy for "can this host reach AWS at all"; the tests themselves then fail or
// pass on the API's answer.
const (
	awsProbeAddr    = "rds.us-east-1.amazonaws.com:443"
	awsProbeTimeout = 3 * time.Second
)

// GetAWSKeys returns testing AWS keys.
func GetAWSKeys(tb testing.TB) (string, string) {
	tb.Helper()

	accessKey, secretKey := os.Getenv("AWS_ACCESS_KEY"), os.Getenv("AWS_SECRET_KEY")
	if accessKey == "" || secretKey == "" {
		tb.Skip("Environment variables AWS_ACCESS_KEY / AWS_SECRET_KEY are not defined, skipping test")
	}
	return accessKey, secretKey
}

// SkipIfHostUnreachable skips when addr ("host:port") cannot be dialled.
//
// The general form of SkipIfAWSUnreachable, for tests that need a live answer from a
// remote service. On the airgapped test host nothing off-box is reachable by design, so
// such a test can only fail there, and a red result says nothing a reader should act on.
// The skip names the endpoint and the dial error so an unexercised test reads as
// unexercised rather than as a pass.
func SkipIfHostUnreachable(tb testing.TB, addr, what string) {
	tb.Helper()

	conn, err := net.DialTimeout("tcp", addr, awsProbeTimeout)
	if err != nil {
		tb.Skipf("%s is not reachable at %s (%v); skipping a test that needs a live response", what, addr, err)
		return
	}
	conn.Close() //nolint:errcheck
}

// SkipIfAWSUnreachable skips when this host cannot reach AWS.
//
// Some RDS tests pass DELIBERATELY INVALID credentials and assert that AWS rejects them
// with InvalidClientTokenId -- which only happens if the request reaches AWS. With no
// egress the SDK instead retries to exhaustion and reports "exceeded maximum number of
// attempts", so the test fails having proven nothing about our code.
//
// That is the normal state for two environments we care about: the airgapped test host,
// whose whole purpose is to have no route off-box, and any developer machine behind a
// proxy. GetAWSKeys already skips when credentials are absent; this covers the tests that
// supply their own bad ones on purpose.
func SkipIfAWSUnreachable(tb testing.TB) {
	tb.Helper()

	conn, err := net.DialTimeout("tcp", awsProbeAddr, awsProbeTimeout)
	if err != nil {
		tb.Skipf("AWS is not reachable at %s (%v); skipping a test that needs a real API response", awsProbeAddr, err)
		return
	}
	conn.Close() //nolint:errcheck
}
