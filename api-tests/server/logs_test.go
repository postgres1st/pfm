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

package server

import (
	"archive/zip"
	"bytes"
	"slices"
	"sort"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	pmmapitests "github.com/percona/pmm/api-tests"
	serverClient "github.com/percona/pmm/api/server/v1/json/client"
	"github.com/percona/pmm/api/server/v1/json/client/server_service"
)

// The diagnostic archive's contents depend on which process manager the server runs.
// The two backends keep service logs and configuration in entirely different places:
// supervisord writes .log files under /srv/logs and ini files under /etc/supervisord.d,
// while a systemd host has neither and the archive collects the journal and the unit
// files instead. A single hardcoded list can only be right on one of them -- it used to
// be the supervisord one, so on the platform we actually ship this test failed while the
// archive was correct.
//
// The backend is read off the archive itself rather than off this machine's filesystem,
// because the server under test may not be this machine.
func TestDownloadLogs(t *testing.T) {
	t.Parallel()

	var buf bytes.Buffer
	res, err := serverClient.Default.ServerService.Logs(&server_service.LogsParams{
		Context: pmmapitests.Context,
	}, &buf)
	require.NoError(t, err)
	require.NotNil(t, res)

	r := bytes.NewReader(buf.Bytes())
	zipR, err := zip.NewReader(r, r.Size())
	require.NoError(t, err)

	actual := make([]string, 0, len(zipR.File))
	for _, file := range zipR.File {
		// Per-agent logs are named with a random agent ID, so only the fixed entry in
		// that directory can be asserted by name.
		if strings.HasPrefix(file.Name, "client/pfw-agent/") && file.Name != "client/pfw-agent/pmm-agent.log" {
			continue
		}
		actual = append(actual, file.Name)
	}
	sort.Strings(actual)

	// Collected the same way regardless of backend.
	common := []string{
		"client/list.txt",
		"client/pfw-admin-version.txt",
		"client/pfw-agent-config.yaml",
		"client/pfw-agent-version.txt",
		"client/pfw-agent/pmm-agent.log",
		"client/status.json",
		"installed.json",
		"nginx.conf",
		"pfw-agent.yaml",
		"pfw-ssl.conf",
		"pfw.conf",
		"pmm-version.txt",
		"prometheus.base.yml",
		"victoriametrics-promscrape.yml",
		"victoriametrics_targets.json",
	}
	assert.Subset(t, actual, common, "these entries must be present whatever the backend")

	hasSystemd := slices.Contains(actual, "systemctl_status.log")
	hasSupervisord := slices.Contains(actual, "supervisorctl_status.log")
	require.NotEqual(t, hasSystemd, hasSupervisord,
		"the archive must report exactly one process manager's status, got systemd=%v supervisord=%v", hasSystemd, hasSupervisord)

	if hasSystemd {
		// Pinned exactly. On systemd every entry is code-derived -- journalUnits and the
		// unit list in processManagerConfigFiles are both constants -- so the full set is
		// deterministic, and equality is what catches an entry silently dropping out.
		expected := append([]string{}, common...)
		expected = append(expected,
			"pfw-clickhouse.log",
			"pfw-grafana.log",
			"pfw-init.log",
			"pfw-managed.log",
			"pfw-managed.service",
			"pfw-nginx.log",
			"pfw-postgresql.log",
			"pfw-qan-api2.log",
			"pfw-qan-api2.service",
			"pfw-server-agent.log",
			"pfw-victoriametrics.log",
			"pfw-victoriametrics.service",
			"pfw-vmalert.log",
			"pfw-vmalert.service",
			"pfw-vmproxy.log",
			"pfw-vmproxy.service",
			"pfw.target",
			"systemctl_status.log",
		)
		sort.Strings(expected)
		assert.Equal(t, expected, actual)
		return
	}

	// Supervisord is deliberately NOT pinned to an exact set. Its service logs come from
	// a filepath.Glob over /srv/logs, so the set is a property of the deployment rather
	// than of the code, and an equality here would assert something this suite cannot
	// know. What IS guaranteed is the configuration the archive must carry, plus the
	// presence of some service logs.
	assert.Subset(t, actual, []string{
		"supervisorctl_status.log",
		"supervisord.conf",
		"pmm.ini",
		"qan-api2.ini",
		"victoriametrics.ini",
		"vmalert.ini",
		"vmproxy.ini",
	}, "the supervisord archive must carry the configuration that governs the services")

	var serviceLogs int
	for _, name := range actual {
		if strings.HasSuffix(name, ".log") && !strings.HasPrefix(name, "client/") {
			serviceLogs++
		}
	}
	assert.NotZero(t, serviceLogs, "the archive must carry service logs; that is what it is for")
}
