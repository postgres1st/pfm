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

package commands

import (
	"net/url"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/percona/pmm/admin/pkg/flags"
)

func TestConfigCommandArgs(t *testing.T) {
	cmd := &ConfigCommand{
		NodeAddress: "1.2.3.4",
		NodeType:    "generic",
		NodeName:    "node1",
	}

	t.Run("SwitchToTLS1", func(t *testing.T) {
		u, err := url.Parse("http://127.0.0.1:80")
		require.NoError(t, err)
		args, switchedToTLS := cmd.args(&flags.GlobalFlags{ServerURL: u})
		expected := []string{
			"--server-address=127.0.0.1:443",
			"--server-insecure-tls",
			"setup", "1.2.3.4", "generic", "node1",
		}
		assert.Equal(t, expected, args)
		assert.True(t, switchedToTLS)
	})

	t.Run("SwitchToTLS2", func(t *testing.T) {
		cmd := &ConfigCommand{
			NodeAddress: "1.2.3.4",
			NodeType:    "generic",
			NodeName:    "node1",
		}
		u, err := url.Parse("http://admin:admin@127.0.0.1")
		require.NoError(t, err)
		args, switchedToTLS := cmd.args(&flags.GlobalFlags{ServerURL: u})
		expected := []string{
			"--server-address=127.0.0.1:443",
			"--server-username=admin",
			"--server-insecure-tls",
			"setup", "1.2.3.4", "generic", "node1",
		}
		assert.Equal(t, expected, args)
		assert.True(t, switchedToTLS)
	})
	t.Run("DisableCollectors", func(t *testing.T) {
		cmd := &ConfigCommand{
			NodeAddress:       "1.2.3.4",
			NodeType:          "generic",
			NodeName:          "node1",
			DisableCollectors: []string{"cpu", "diskstats"},
		}
		u, err := url.Parse("http://admin:admin@127.0.0.1")
		require.NoError(t, err)
		args, switchedToTLS := cmd.args(&flags.GlobalFlags{ServerURL: u})
		expected := []string{
			"--server-address=127.0.0.1:443",
			"--server-username=admin",
			"--server-insecure-tls",
			"setup",
			"--disable-collectors=cpu,diskstats",
			"1.2.3.4", "generic", "node1",
		}
		assert.Equal(t, expected, args)
		assert.True(t, switchedToTLS)
	})

	t.Run("LoggingLevel", func(t *testing.T) {
		cmd := &ConfigCommand{
			NodeAddress: "1.2.3.4",
			NodeType:    "generic",
			NodeName:    "node1",
			LogLevelFatalFlags: flags.LogLevelFatalFlags{
				LogLevel: "info",
			},
		}

		u, err := url.Parse("http://admin:admin@127.0.0.1")
		require.NoError(t, err)
		args, switchedToTLS := cmd.args(&flags.GlobalFlags{
			ServerURL:   u,
			EnableDebug: true,
			EnableTrace: true,
		})
		expected := []string{
			"--server-address=127.0.0.1:443",
			"--server-username=admin",
			"--server-insecure-tls",
			"--log-level=info",
			"--debug",
			"--trace",
			"setup",
			"1.2.3.4",
			"generic",
			"node1",
		}
		assert.Equal(t, expected, args)
		assert.True(t, switchedToTLS)
	})
}

// The PGF WatchTower admin credential used to be passed to pfw-agent as
// --server-password on its command line, so it was readable by any local user via
// /proc/<pid>/cmdline (0444) for the lifetime of the setup call. It now travels in
// the child's environment, which is 0400 owner-only.
func TestConfigCommandDoesNotPutTheServerPasswordInArgv(t *testing.T) {
	t.Parallel()

	u, err := url.Parse("https://admin:s3cr3t@1.2.3.4/")
	require.NoError(t, err)
	globals := &flags.GlobalFlags{ServerURL: u}

	cmd := &ConfigCommand{}
	args, _ := cmd.args(globals)

	for _, a := range args {
		assert.NotContains(t, a, "s3cr3t", "the credential must not appear in any argument")
		assert.NotContains(t, a, "--server-password", "the flag itself must be gone, not merely empty")
	}
	// The username is not a secret and stays, so the agent still knows who to be.
	assert.Contains(t, args, "--server-username=admin")

	env := serverPasswordEnv(globals)
	assert.Equal(t, []string{"PFW_AGENT_SERVER_PASSWORD=s3cr3t"}, env,
		"the credential must still reach pfw-agent, via its environment")
}

func TestServerPasswordEnvHandlesAbsentCredentials(t *testing.T) {
	t.Parallel()

	t.Run("no user info at all", func(t *testing.T) {
		t.Parallel()
		u, err := url.Parse("https://1.2.3.4/")
		require.NoError(t, err)
		assert.Nil(t, serverPasswordEnv(&flags.GlobalFlags{ServerURL: u}))
	})

	// A username with no password: emitting PFW_AGENT_SERVER_PASSWORD= would override
	// any value the agent already has in its own environment with an empty one.
	t.Run("username but no password", func(t *testing.T) {
		t.Parallel()
		u, err := url.Parse("https://admin@1.2.3.4/")
		require.NoError(t, err)
		assert.Nil(t, serverPasswordEnv(&flags.GlobalFlags{ServerURL: u}))
	})

	t.Run("nil ServerURL", func(t *testing.T) {
		t.Parallel()
		assert.Nil(t, serverPasswordEnv(&flags.GlobalFlags{}))
	})
}
