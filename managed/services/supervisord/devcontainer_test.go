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

package supervisord

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/percona/pmm/managed/models"
)

// TODO move tests to other files and remove this one.
func TestDevContainer(t *testing.T) {
	// This exercises the SUPERVISORD backend specifically -- it drives
	// /etc/supervisord.d and asserts supervisorctl is present. A systemd-native
	// install ships neither, so there is nothing here for it to test and a failure
	// says only "this is not a devcontainer". Skip rather than fail, so an
	// unexercised test reads as unexercised.
	if _, err := exec.LookPath("supervisorctl"); err != nil {
		t.Skip("supervisorctl is not installed; this test covers the supervisord backend only")
	}

	t.Run("UpdateConfiguration", func(t *testing.T) {
		// logrus.SetLevel(logrus.DebugLevel)
		vmParams, err := models.NewVictoriaMetricsParams(models.BasePrometheusConfigPath, models.VMBaseURL)
		require.NoError(t, err)

		s := New("/etc/supervisord.d", &models.Params{VMParams: vmParams, PGParams: &models.PGParams{}, HAParams: &models.HAParams{}})
		require.NotEmpty(t, s.supervisorctlPath)

		ctx, cancel := context.WithCancel(t.Context())
		defer cancel()
		go s.Run(ctx)

		// restore original files after test
		originals := make(map[string][]byte)
		matches, err := filepath.Glob("/etc/supervisord.d/*.ini")
		require.NoError(t, err)
		for _, m := range matches {
			b, err := os.ReadFile(m) //nolint:gosec
			require.NoError(t, err)
			originals[m] = b
		}
		defer func() {
			for name, b := range originals {
				err = os.WriteFile(name, b, 0)
				require.NoError(t, err)
			}
			// Remove the ones this test created, not just restore the ones it found.
			// saveConfigAndReload below asserts changed==true on the first write and
			// false on the second, so a leftover victoriametrics.ini makes the FIRST
			// write report unchanged and the test fails on its second run -- passing
			// once on a clean tree and never again.
			after, err := filepath.Glob("/etc/supervisord.d/*.ini")
			require.NoError(t, err)
			for _, m := range after {
				if _, existed := originals[m]; !existed {
					require.NoError(t, os.Remove(m))
				}
			}
			// force update supervisor config
			err = s.supervisorctl("update")
			require.NoError(t, err)
		}()

		settings := &models.Settings{
			DataRetention: 3600 * time.Hour,
		}

		b, err := s.marshalConfig(templates.Lookup("victoriametrics"), settings)
		require.NoError(t, err)
		changed, err := s.saveConfigAndReload("victoriametrics", b)
		require.NoError(t, err)
		assert.True(t, changed)
		changed, err = s.saveConfigAndReload("victoriametrics", b)
		require.NoError(t, err)
		assert.False(t, changed)

		err = s.UpdateConfiguration(settings)
		require.NoError(t, err)
	})
}
