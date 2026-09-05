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

package agents

import (
	"context"
	"errors"
	"fmt"
	"os"
	"strings"
	"sync"

	"github.com/AlekSi/pointer"
	"github.com/sirupsen/logrus"

	agentv1 "github.com/percona/pmm/api/agent/v1"
	"github.com/percona/pmm/managed/models"
	"github.com/percona/pmm/version"
)

// AgentService handles generic communication with the Agent.
type AgentService struct {
	r *Registry
}

// NewAgentService returns new agent service.
func NewAgentService(r *Registry) *AgentService {
	return &AgentService{
		r: r,
	}
}

// Logs by Agent ID.
func (a *AgentService) Logs(_ context.Context, pmmAgentID, agentID string, limit uint32) ([]string, uint32, error) {
	agent, err := a.r.get(pmmAgentID)
	if err != nil {
		return nil, 0, err
	}

	resp, err := agent.channel.SendAndWaitResponse(&agentv1.AgentLogsRequest{
		AgentId: agentID,
		Limit:   limit,
	})
	if err != nil {
		return nil, 0, err
	}

	agentLogsResponse, ok := resp.(*agentv1.AgentLogsResponse)
	if !ok {
		return nil, 0, errors.New("wrong response from agent (not AgentLogsResponse model)")
	}

	return agentLogsResponse.GetLogs(), agentLogsResponse.GetAgentConfigLogLinesCount(), nil
}

// PBMSwitchPITR switches Point-in-Time Recovery feature for pbm on the pmm-agent.
func (a *AgentService) PBMSwitchPITR(pmmAgentID, dsn string, files map[string]string, tdp *models.DelimiterPair, enabled bool) error {
	agent, err := a.r.get(pmmAgentID)
	if err != nil {
		return err
	}

	req := &agentv1.PBMSwitchPITRRequest{
		Dsn: dsn,
		TextFiles: &agentv1.TextFiles{
			Files:              files,
			TemplateLeftDelim:  tdp.Left,
			TemplateRightDelim: tdp.Right,
		},
		Enabled: enabled,
	}

	_, err = agent.channel.SendAndWaitResponse(req)
	return err
}

type redactMode int

const (
	redactSecrets redactMode = iota
	exposeSecrets
)

var pmmAgentPathsBaseSupport = version.MustParse("2.22.99")

// redactWords returns words that should be redacted from given Agent logs/output.
func redactWords(agent *models.Agent) []string {
	var words []string
	if s := pointer.GetString(agent.Password); s != "" {
		words = append(words, s)
	}
	if s := pointer.GetString(agent.AgentPassword); s != "" {
		words = append(words, s)
	}
	if s := agent.AWSOptions.AWSSecretKey; s != "" {
		words = append(words, s)
	}
	if s := agent.AzureOptions.ClientSecret; s != "" {
		words = append(words, s)
	}
	if s := agent.MongoDBOptions.TLSCertificateKey; s != "" {
		words = append(words, s)
	}
	if s := agent.MongoDBOptions.TLSCertificateKeyFilePassword; s != "" {
		words = append(words, s)
	}
	if s := agent.MySQLOptions.TLSKey; s != "" {
		words = append(words, s)
	}
	if s := agent.PostgreSQLOptions.SSLKey; s != "" {
		words = append(words, s)
	}

	return words
}

// pathsBase returns paths base and in case of unsupported PMM client a hardcoded value.
func pathsBase(agentVersion *version.Parsed, tdpLeft, tdpRight string) string {
	if agentVersion == nil || agentVersion.Less(pmmAgentPathsBaseSupport) {
		return "/opt/postgres1st/watchtower"
	}

	return tdpLeft + " .paths_base " + tdpRight
}

// ensureAuthParams updates agent start parameters to contain prometheus webconfig.
func ensureAuthParams(exporter *models.Agent, params *agentv1.SetStateRequest_AgentProcess,
	agentVersion *version.Parsed, minAuthVersion *version.Parsed, useNewTLSConfig bool,
) error {
	if agentVersion.Less(minAuthVersion) {
		auth, err := httpAuthEnv(exporter)
		if err != nil {
			return err
		}
		params.Env = append(params.Env, auth)
		return nil
	}

	if params.TextFiles == nil {
		params.TextFiles = make(map[string]string)
	}

	wcf, err := exporter.BuildWebConfigFile()
	if err != nil {
		return err
	}
	params.TextFiles["webConfig"] = wcf
	// see https://github.com/prometheus/exporter-toolkit/blob/master/docs/web-configuration.md
	if useNewTLSConfig {
		params.Args = append(params.Args, "--web.config.file="+params.TemplateLeftDelim+" .TextFiles.webConfig "+params.TemplateRightDelim)
	} else {
		params.Args = append(params.Args, "--web.config="+params.TemplateLeftDelim+" .TextFiles.webConfig "+params.TemplateRightDelim)
	}

	return nil
}

// ExposeExportersEnvVar restores the previous behaviour of binding every exporter
// to 0.0.0.0 regardless of the per-service --expose-exporter flag.
//
// It exists because the new loopback default fails silently: an exporter that is
// no longer reachable simply stops being scraped, and a fleet that has already
// gone dark cannot be recovered with a per-service flag, since re-registering a
// service needs a working connection to it. Set to a strconv.ParseBool-accepted
// truthy value to restore the old default while re-registering, then unset it.
const ExposeExportersEnvVar = "PFW_EXPOSE_EXPORTERS"

// Resolved lazily and cached, NOT in a package-level initializer. An initializer
// runs before main() and before logging is configured, so a bad value there
// produces a bare stack trace and a crash loop.
//
// And it must never panic at all: this variable is the documented recovery path
// from a fleet-wide metrics blackout, so a typo while recovering must not take the
// server down on top of it. An unparseable value is reported and treated as unset.
// (models.resolveSupportedServiceTypes does panic on a bad value, but that variable
// is documented as debugging-only; this one is production recovery.)
//
// Tests in this package override the resolved value directly.
var exposeAllExportersByDefault = sync.OnceValue(resolveExposeExporters)

func resolveExposeExporters() bool {
	// Trimmed and lower-cased before parsing: a systemd EnvironmentFile line, a
	// compose `environment:` entry and a shell export all produce stray whitespace
	// or capitalisation, and none of them mean anything different by it. An EMPTY
	// value counts as unset -- os.LookupEnv reports it as present, and `FOO=` in a
	// unit file is far more likely to mean "left blank" than "invalid".
	raw := strings.ToLower(strings.TrimSpace(os.Getenv(ExposeExportersEnvVar)))
	if raw == "" {
		return false
	}

	switch raw {
	case "1", "t", "true", "y", "yes", "on":
		return true
	case "0", "f", "false", "n", "no", "off":
		return false
	}

	logrus.Warnf("Ignoring %s=%q: expected a boolean (true/false, yes/no, on/off, 1/0). "+
		"Exporters will use the default loopback bind; set a valid value to expose them.",
		ExposeExportersEnvVar, raw)

	return false
}

// httpAuthEnv builds the HTTP_AUTH environment entry for exporters too old for a
// prometheus web config, failing closed when no credential is stored.
//
// It exists as a helper because three call sites produce this string --
// ensureAuthParams, mysqldExporterConfig and proxysqlExporterConfig -- and two of
// them built it inline. An empty password there yields `HTTP_AUTH=pmm:`, which the
// exporter accepts, so the endpoint would end up with no authentication at all:
// strictly worse than the agent-ID fallback removed in this change. Every producer
// of this string must go through here.
func httpAuthEnv(exporter *models.Agent) (string, error) {
	password := exporter.GetAgentPassword()
	if password == "" {
		return "", fmt.Errorf("no agent password set for agent %s", exporter.AgentID)
	}

	return "HTTP_AUTH=pmm:" + password, nil
}

// getExporterListenAddress returns the appropriate listen address to use for a given exporter.
//
// Loopback is the default. Pull-mode scraping reaches an exporter at
// node.Address:port (models.Agent.ExporterURL), so binding loopback means a
// remote scrape no longer succeeds unless the operator opts in per service with
// --expose-exporter, sets ExposeExportersEnvVar fleet-wide, or switches the
// service to push mode. That is the intended trade: the previous default
// published every exporter on every interface, and the credential travels in
// cleartext because these endpoints carry no TLS.
func getExporterListenAddress(_ *models.Node, exporter *models.Agent) string {
	switch {
	// Order matters. An explicit per-service --expose-exporter still wins over
	// push mode, as it did before this change. The fleet-wide escape hatch is
	// checked only after push mode, so turning it on does not start publishing
	// listeners for services that need no reachable listener at all.
	case exporter.ExporterOptions.ExposeExporter:
		return "0.0.0.0"
	case exporter.ExporterOptions.PushMetrics:
		return "127.0.0.1"
	case exposeAllExportersByDefault():
		return "0.0.0.0"
	}

	return "127.0.0.1"
}
