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

// Package server contains PMM server API tests.
package server

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc/codes"

	pmmapitests "github.com/percona/pmm/api-tests"
	advisorsv1 "github.com/percona/pmm/api/advisors/v1"
	advisorClient "github.com/percona/pmm/api/advisors/v1/json/client"
	advisor "github.com/percona/pmm/api/advisors/v1/json/client/advisor_service"
	serverClient "github.com/percona/pmm/api/server/v1/json/client"
	server "github.com/percona/pmm/api/server/v1/json/client/server_service"
)

// RestoreSettingsDefaults restores PMM Server settings to their default values.
func RestoreSettingsDefaults(t *testing.T) {
	t.Helper()

	// Some deployments configure individual settings through the ENVIRONMENT --
	// PMM_ENABLE_TELEMETRY and PMM_ENABLE_UPDATES in the native install -- and the server
	// then rejects the whole ChangeSettings call naming the variable. Because this helper
	// is the deferred restore for most of the settings suite, one such field meant
	// NOTHING was restored: settings leaked between tests and later runs failed asserting
	// defaults the server no longer held. A dozen unrelated-looking failures, one cause.
	//
	// So it drops whichever field the server objects to and retries, rather than hardcoding
	// a list that would be wrong on the other deployment. Everything else still gets
	// restored, which is the point.
	body := server.ChangeSettingsBody{
		EnableAdvisor:  new(true),
		EnableAlerting: new(true),
		// Telemetry and updates are deliberately NOT requested. This product ships them
		// force-disabled -- telemetry reports to a third party, and there is no
		// auto-update or version-broadcast feature -- so a test helper must never ask
		// for them to be ON. Asking and backing off when refused still leaves the
		// request on any deployment that does not happen to refuse it.
		MetricsResolutions: &server.ChangeSettingsParamsBodyMetricsResolutions{
			Hr: "5s",
			Mr: "10s",
			Lr: "60s",
		},
		AdvisorRunIntervals: &server.ChangeSettingsParamsBodyAdvisorRunIntervals{
			FrequentInterval: "14400s",
			StandardInterval: "86400s",
			RareInterval:     "280800s",
		},
		DataRetention: "2592000s",
		AWSPartitions: &server.ChangeSettingsParamsBodyAWSPartitions{
			Values: []string{"aws"},
		},
	}

	var res *server.ChangeSettingsOK
	var err error
	// At most one retry per env-managed field, plus one final attempt.
	for range 4 {
		res, err = serverClient.Default.ServerService.ChangeSettings(&server.ChangeSettingsParams{
			Body:    body,
			Context: pmmapitests.Context,
		})
		if err == nil {
			break
		}
		msg := err.Error()
		switch {
		case strings.Contains(msg, "PMM_ENABLE_ALERTING") && body.EnableAlerting != nil:
			body.EnableAlerting = nil
		case strings.Contains(msg, "PMM_ENABLE_ADVISOR") && body.EnableAdvisor != nil:
			body.EnableAdvisor = nil
		default:
			// Not an environment-managed field: a real failure, report it.
			require.NoError(t, err)
		}
	}
	require.NoError(t, err)
	// Telemetry is deliberately not requested above, so it must not be asserted here
	// either: this product ships it force-disabled, and demanding it be ON would make
	// the restore fail on every install that honours that.
	assert.True(t, res.Payload.Settings.AdvisorEnabled)
	expectedResolutions := &server.ChangeSettingsOKBodySettingsMetricsResolutions{
		Hr: "5s",
		Mr: "10s",
		Lr: "60s",
	}
	assert.Equal(t, expectedResolutions, res.Payload.Settings.MetricsResolutions)
	expectedAdvisorRunIntervals := &server.ChangeSettingsOKBodySettingsAdvisorRunIntervals{
		FrequentInterval: "14400s",
		StandardInterval: "86400s",
		RareInterval:     "280800s",
	}
	assert.Equal(t, expectedAdvisorRunIntervals, res.Payload.Settings.AdvisorRunIntervals)
	assert.Equal(t, "2592000s", res.Payload.Settings.DataRetention)
	assert.Equal(t, []string{"aws"}, res.Payload.Settings.AWSPartitions)
}

func restoreCheckIntervalDefaults(t *testing.T) {
	t.Helper()

	resp, err := advisorClient.Default.AdvisorService.ListAdvisorChecks(nil)
	require.NoError(t, err)
	require.NotEmpty(t, resp.Payload.Checks)

	var params *advisor.ChangeAdvisorChecksParams

	for _, check := range resp.Payload.Checks {
		params = &advisor.ChangeAdvisorChecksParams{
			Body: advisor.ChangeAdvisorChecksBody{
				Params: []*advisor.ChangeAdvisorChecksParamsBodyParamsItems0{
					{
						Name:     check.Name,
						Interval: new(advisorsv1.AdvisorCheckInterval_ADVISOR_CHECK_INTERVAL_STANDARD.String()),
					},
				},
			},
			Context: pmmapitests.Context,
		}

		_, err = advisorClient.Default.AdvisorService.ChangeAdvisorChecks(params)
		require.NoError(t, err)
	}
}

// AssertEnvOwnedChangeIsRefused sends body -- which must try to switch telemetry or
// updates ON -- and asserts the server REFUSES it with exactly wantMsg.
//
// This is deliberately an assertion and not a skip. The product ships both
// force-disabled: telemetry reports to a third party, and there is no auto-update or
// version-broadcast feature. "The API cannot switch these back on" is a guarantee the
// product makes, so the upstream tests that proved the capability now prove its removal.
//
// The distinction matters for a privacy guarantee. A skip records that we could not
// check. This records that we checked and telemetry is unreachable -- and if it ever
// fails, including on a deployment that forgot to set the variable, that is a real
// finding about that deployment rather than a quirk of the test.
//
// It also asserts the refusal is ATOMIC. A body that switches telemetry on AND changes
// advisors must change NOTHING: a server that applied the rest of the body and silently
// dropped only the telemetry field would leave the caller believing the whole write
// landed.
func AssertEnvOwnedChangeIsRefused(t *testing.T, wantMsg string, body server.ChangeSettingsBody) {
	t.Helper()

	before, err := serverClient.Default.ServerService.GetSettings(nil)
	require.NoError(t, err)

	_, err = serverClient.Default.ServerService.ChangeSettings(&server.ChangeSettingsParams{
		Body:    body,
		Context: pmmapitests.Context,
	})
	// Pinned to the exact code and message, not merely "an error happened": a 401 from a
	// lost session or a 500 from a broken server would otherwise read as the guarantee
	// holding, which is the one way this assertion could reassure us while being blind.
	pmmapitests.AssertAPIErrorf(t, err, 400, codes.FailedPrecondition, "%s", wantMsg)

	after, err := serverClient.Default.ServerService.GetSettings(nil)
	require.NoError(t, err)
	assert.False(t, after.Payload.Settings.TelemetryEnabled, "telemetry must stay off")
	assert.False(t, after.Payload.Settings.UpdatesEnabled, "updates must stay off")
	assert.Equal(t, before.Payload.Settings.AdvisorEnabled, after.Payload.Settings.AdvisorEnabled,
		"a refused change must be atomic: no other field in the body may be applied")
}
