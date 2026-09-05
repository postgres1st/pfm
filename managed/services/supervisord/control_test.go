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

package supervisord

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

// The case that matters is "deactivating". An earlier version of this mapping treated
// anything other than "active" as stopped, which reports a unit still shutting down as
// already down -- and the caller uses exactly that to decide when it is safe to start
// rewriting the database. Declaring the server stopped while it is still running is the
// race that loses data, so it is asserted explicitly rather than left to parseIsActive.
func TestStatusFromIsActive(t *testing.T) {
	t.Parallel()

	for _, tc := range []struct {
		out              string
		running, stopped bool
		why              string
	}{
		{"active\n", true, false, "plainly up"},
		{"inactive\n", false, true, "plainly down"},
		{"failed\n", false, true, "failed is down, not undeterminable — a caller waiting to stop must not hang"},
		{"activating\n", true, false, "coming up still counts as running"},
		{"deactivating\n", true, false, "STILL RUNNING: reporting this as stopped would race the shutdown"},
		{"reloading\n", true, false, "reloading is up"},
		{"", false, false, "no output is undeterminable: neither question is confirmed"},
		{"something-new\n", false, false, "an unrecognised state must not be guessed either way"},
	} {
		t.Run(tc.out+"/"+tc.why, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tc.running, statusFromIsActive(tc.out, StatusRunning),
				"is-active %q asked for RUNNING", tc.out)
			assert.Equal(t, tc.stopped, statusFromIsActive(tc.out, StatusStopped),
				"is-active %q asked for STOPPED", tc.out)
		})
	}
}

// An undeterminable state must answer false to BOTH questions, so a polling caller keeps
// waiting and times out rather than proceeding on a guess. Asserted separately because it
// is the property that makes the failure mode safe, and a future edit could satisfy the
// table above while breaking it.
func TestUndeterminableStateIsNeverConfirmed(t *testing.T) {
	t.Parallel()

	for _, out := range []string{"", "unknown", "garbage", "Failed to get properties"} {
		assert.False(t, statusFromIsActive(out, StatusRunning), "%q must not confirm running", out)
		assert.False(t, statusFromIsActive(out, StatusStopped), "%q must not confirm stopped", out)
	}
}

// currentProcessManager must reach the same verdict as the running server's New(), or a
// standalone tool and the server disagree about which manager owns the services. Both
// call selectProcessManager; this pins the env override, which is the path an operator
// uses and the one a rename would silently break.
func TestCurrentProcessManagerHonoursEnvOverride(t *testing.T) {
	t.Setenv(processManagerEnv, string(pmSystemd))
	pm, _, _ := currentProcessManager()
	assert.Equal(t, pmSystemd, pm)

	t.Setenv(processManagerEnv, string(pmSupervisord))
	pm, _, _ = currentProcessManager()
	assert.Equal(t, pmSupervisord, pm)
}
