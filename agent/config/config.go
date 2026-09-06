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

// Package config provides access to pfw-agent configuration.
package config

import (
	"errors"
	"fmt"
	"io/fs"
	"net"
	"net/url"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/alecthomas/kingpin/v2"
	"github.com/sirupsen/logrus"
	"golang.org/x/sys/unix"
	"gopkg.in/yaml.v3"

	"github.com/percona/pmm/utils/nodeinfo"
	"github.com/percona/pmm/version"
)

const (
	pathBaseDefault = "/opt/postgres1st/watchtower"
	agentTmpPath    = "tmp" // temporary directory to keep exporters' config files, relative to pathBase
	agentDataPath   = "data"
	agentPrefix     = "/agent_id/"
)

// Server represents Postgres1st WatchTower Server configuration.
type Server struct {
	Address     string `yaml:"address"`
	Username    string `yaml:"username"`
	Password    string `yaml:"password"`
	InsecureTLS bool   `yaml:"insecure-tls"`

	WithoutTLS bool `yaml:"without-tls,omitempty"` // for development and testing
}

// URL returns base Postgres1st WatchTower Server URL for JSON APIs.
func (s *Server) URL() *url.URL {
	if s.Address == "" {
		return nil
	}

	var user *url.Userinfo
	switch {
	case s.Password != "":
		user = url.UserPassword(s.Username, s.Password)
	case s.Username != "":
		user = url.User(s.Username)
	}
	return &url.URL{
		Scheme: "https",
		User:   user,
		Host:   s.Address,
		Path:   "/",
	}
}

// FilteredURL returns URL with redacted password.
func (s *Server) FilteredURL() string {
	u := s.URL()
	if u == nil {
		return ""
	}

	if _, ps := u.User.Password(); ps {
		u.User = url.UserPassword(u.User.Username(), "***")
	}

	// unescape ***; url.unescape and url.encodeUserPassword are not exported, so use strings.Replace
	return strings.ReplaceAll(u.String(), ":%2A%2A%2A@", ":***@")
}

// Paths represents binary paths configuration.
type Paths struct {
	PathsBase        string `yaml:"paths_base"`
	ExportersBase    string `yaml:"exporters_base"`
	NodeExporter     string `yaml:"node_exporter"`
	MySQLdExporter   string `yaml:"mysqld_exporter"`
	MongoDBExporter  string `yaml:"mongodb_exporter"`
	PostgresExporter string `yaml:"postgres_exporter"`
	ProxySQLExporter string `yaml:"proxysql_exporter"`
	RDSExporter      string `yaml:"rds_exporter"`
	AzureExporter    string `yaml:"azure_exporter"`
	ValkeyExporter   string `yaml:"valkey_exporter"`

	VMAgent string `yaml:"vmagent"`
	Nomad   string `yaml:"nomad"`

	TempDir      string `yaml:"tempdir"`
	NomadDataDir string `yaml:"nomad_data_dir"`

	PTSummary        string `yaml:"pt_summary"`
	PTPGSummary      string `yaml:"pt_pg_summary"`
	PTMySQLSummary   string `yaml:"pt_mysql_summary"`
	PTMongoDBSummary string `yaml:"pt_mongodb_summary"`

	SlowLogFilePrefix string `yaml:"slowlog_file_prefix,omitempty"` // for development and testing
}

// Ports represents ports configuration.
type Ports struct {
	Min uint16 `yaml:"min"`
	Max uint16 `yaml:"max"`
}

// Setup contains `pfw-agent setup` flag and argument values.
// It is never stored in configuration file.
type Setup struct {
	NodeType          string
	NodeName          string
	MachineID         string
	Distro            string
	ContainerID       string
	ContainerName     string
	NodeModel         string
	Region            string
	Az                string
	Address           string
	MetricsMode       string
	DisableCollectors string
	CustomLabels      string
	AgentPassword     string
	ProcMountsPath    string

	Force            bool
	SkipRegistration bool
	ExposeExporter   bool
}

// Config represents pfw-agent's configuration.
type Config struct {
	// no config file there

	ID                             string `yaml:"id"`
	ListenAddress                  string `yaml:"listen-address"`
	ListenPort                     uint16 `yaml:"listen-port"`
	RunnerCapacity                 uint16 `yaml:"runner-capacity,omitempty"`
	RunnerMaxConnectionsPerService uint16 `yaml:"runner-max-connections-per-service,omitempty"`
	ProcMountsPath                 string `yaml:"proc-mounts-path,omitempty"`

	Server Server `yaml:"server"`
	Paths  Paths  `yaml:"paths"`
	Ports  Ports  `yaml:"ports"`

	LogLevel string `yaml:"log-level"`
	Debug    bool   `yaml:"debug"`
	Trace    bool   `yaml:"trace"`

	LogLinesCount         uint   `json:"log-lines-count"`
	PerfschemaRefreshRate uint16 `yaml:"perfschema-refresh-rate,omitempty"`

	WindowConnectedTime time.Duration `yaml:"window-connected-time"`

	Setup      Setup      `yaml:"-"`
	Encryption Encryption `yaml:"-"`
}

// ConfigFileDoesNotExistError error is returned from Get method if configuration file is expected,
// but does not exist.
type ConfigFileDoesNotExistError string //nolint:revive

func (e ConfigFileDoesNotExistError) Error() string {
	return fmt.Sprintf("configuration file %s does not exist", string(e))
}

// getFromCmdLine parses command-line flags, environment variables and configuration file
// (if --config-file/PFW_AGENT_CONFIG_FILE is defined).
// It returns configuration, configuration file path (value of -config-file/PFW_AGENT_CONFIG_FILE, may be empty),
// and any encountered error. That error may be ConfigFileDoesNotExistError if configuration file path is not empty,
// but file itself does not exist. Configuration from command-line flags and environment variables
// is still returned in this case.
func getFromCmdLine(cfg *Config, l *logrus.Entry) (string, error) {
	return get(os.Args[1:], cfg, l)
}

// get is Get for unit tests: it parses args instead of command-line.
func get(args []string, cfg *Config, l *logrus.Entry) (string, error) { //nolint:gocognit,cyclop
	var configFileF string
	var err error
	// tweak configuration on exit to cover all return points
	defer func() {
		if cfg == nil {
			return
		}

		// set default values
		if strings.HasPrefix(cfg.ID, agentPrefix) {
			l.Warnf("The agent ID '%s' contains a legacy prefix '%s'. It will be used without it.", cfg.ID, agentPrefix)
			cfg.ID, _ = strings.CutPrefix(cfg.ID, agentPrefix)
		}
		if cfg.ListenAddress == "" {
			cfg.ListenAddress = "127.0.0.1"
		}
		if cfg.ListenPort == 0 {
			cfg.ListenPort = 7777
		}
		if cfg.Ports.Min == 0 {
			cfg.Ports.Min = 42000 // for minimal compatibility with PMM Client 1.x firewall rules and documentation
		}
		if cfg.Ports.Max == 0 {
			cfg.Ports.Max = 51999
		}
		if cfg.WindowConnectedTime == 0 {
			cfg.WindowConnectedTime = time.Hour
		}
		if cfg.PerfschemaRefreshRate == 0 {
			cfg.PerfschemaRefreshRate = 5
		}

		for sp, v := range map[*string]string{
			&cfg.Paths.NodeExporter:     "node_exporter",
			&cfg.Paths.MySQLdExporter:   "mysqld_exporter",
			&cfg.Paths.MongoDBExporter:  "mongodb_exporter",
			&cfg.Paths.PostgresExporter: "postgres_exporter",
			&cfg.Paths.ValkeyExporter:   "valkey_exporter",
			&cfg.Paths.ProxySQLExporter: "proxysql_exporter",
			&cfg.Paths.RDSExporter:      "rds_exporter",
			&cfg.Paths.AzureExporter:    "azure_exporter",
			&cfg.Paths.VMAgent:          "vmagent",
			&cfg.Paths.PTSummary:        "tools/pt-summary",
			&cfg.Paths.PTPGSummary:      "tools/pt-pg-summary",
			&cfg.Paths.PTMongoDBSummary: "tools/pt-mongodb-summary",
			&cfg.Paths.PTMySQLSummary:   "tools/pt-mysql-summary",
			&cfg.Paths.Nomad:            "tools/nomad",
		} {
			if *sp == "" {
				*sp = v
			}
		}

		if cfg.Paths.PathsBase == "" {
			cfg.Paths.PathsBase = pathBaseDefault
		}
		if cfg.Paths.ExportersBase == "" {
			cfg.Paths.ExportersBase = filepath.Join(cfg.Paths.PathsBase, "exporters")
		}

		if abs, _ := filepath.Abs(cfg.Paths.PathsBase); abs != "" {
			cfg.Paths.PathsBase = abs
		}
		if abs, _ := filepath.Abs(cfg.Paths.ExportersBase); abs != "" {
			cfg.Paths.ExportersBase = abs
		}

		if cfg.Paths.TempDir == "" {
			cfg.Paths.TempDir = filepath.Join(cfg.Paths.PathsBase, agentTmpPath)
			l.Infof("Temporary directory will default to %s", cfg.Paths.TempDir)
		}

		if cfg.Paths.NomadDataDir == "" {
			cfg.Paths.NomadDataDir = filepath.Join(cfg.Paths.PathsBase, agentDataPath, "nomad")
			l.Infof("Nomad data directory will default to %s", cfg.Paths.NomadDataDir)
		}

		if !filepath.IsAbs(cfg.Paths.TempDir) {
			cfg.Paths.TempDir = filepath.Join(cfg.Paths.PathsBase, cfg.Paths.TempDir)
			l.Debugf("Temporary directory is configured as %s", cfg.Paths.TempDir)
		}

		for n, sp := range map[string]*string{
			"Percona Toolkit pt-summary":         &cfg.Paths.PTSummary,
			"Percona Toolkit pt-pg-summary":      &cfg.Paths.PTPGSummary,
			"Percona Toolkit pt-mongodb-summary": &cfg.Paths.PTMongoDBSummary,
			"Percona Toolkit pt-mysql-summary":   &cfg.Paths.PTMySQLSummary,
			"Nomad binary":                       &cfg.Paths.Nomad,
		} {
			if !filepath.IsAbs(*sp) {
				*sp = filepath.Join(cfg.Paths.PathsBase, *sp)
				l.Infof("Using %s as a path to %s", *sp, n)
			}
		}

		for n, sp := range map[string]*string{
			"node_exporter":     &cfg.Paths.NodeExporter,
			"mysqld_exporter":   &cfg.Paths.MySQLdExporter,
			"mongodb_exporter":  &cfg.Paths.MongoDBExporter,
			"postgres_exporter": &cfg.Paths.PostgresExporter,
			"valkey_exporter":   &cfg.Paths.ValkeyExporter,
			"proxysql_exporter": &cfg.Paths.ProxySQLExporter,
			"rds_exporter":      &cfg.Paths.RDSExporter,
			"azure_exporter":    &cfg.Paths.AzureExporter,
			"vmagent":           &cfg.Paths.VMAgent,
		} {
			if cfg.Paths.ExportersBase != "" && !filepath.IsAbs(*sp) {
				*sp = filepath.Join(cfg.Paths.ExportersBase, *sp)
			}
			l.Infof("Using %s as a path to %s", *sp, n)
		}

		if cfg.Server.Address != "" {
			_, _, e := net.SplitHostPort(cfg.Server.Address)
			if e != nil {
				host := cfg.Server.Address
				cfg.Server.Address = net.JoinHostPort(host, "443")
				l.Infof("Updating Postgres1st WatchTower Server address from %q to %q.", host, cfg.Server.Address)
			}
		}

		// enabled cross-component PMM_DEBUG and PMM_TRACE take priority
		if b, _ := strconv.ParseBool(os.Getenv("PMM_DEBUG")); b {
			cfg.Debug = true
		}
		if b, _ := strconv.ParseBool(os.Getenv("PMM_TRACE")); b {
			cfg.Trace = true
		}
	}()

	// parse command-line flags and environment variables
	app, cfgFileF := Application(cfg)
	_, err = app.Parse(args)
	if err != nil {
		return configFileF, err
	}
	if *cfgFileF == "" {
		return configFileF, err
	}

	configFileF, err = filepath.Abs(*cfgFileF)
	if err != nil {
		return configFileF, err
	}
	l.Infof("Loading configuration file %s.", configFileF)
	fileCfg, err := loadFromFile(configFileF, &cfg.Encryption)
	if err != nil {
		return configFileF, err
	}

	// re-parse flags into configuration from file
	app, _ = Application(fileCfg)
	_, err = app.Parse(args)
	if err != nil {
		return configFileF, err
	}

	*cfg = *fileCfg
	return configFileF, nil
}

// Application returns kingpin application that will parse command-line flags and environment variables
// (but not configuration file) into cfg except --config-file/PFW_AGENT_CONFIG_FILE that is returned separately.
func Application(cfg *Config) (*kingpin.Application, *string) {
	app := kingpin.New("pfw-agent", "Version "+version.Version)
	app.HelpFlag.Short('h')

	app.Command("run", "Run pfw-agent (default command)").Default()

	// All `app` flags should be optional and should not have non-zero default values for:
	// * `pfw-agent setup` to work;
	// * correct configuration file loading.
	// See `get` above for the actual default values.

	configFileF := app.Flag("config-file", "Configuration file path [PFW_AGENT_CONFIG_FILE]").
		Envar("PFW_AGENT_CONFIG_FILE").PlaceHolder("</path/to/pfw-agent.yaml>").String()
	app.Flag("config-file-key-file", "Path to the key file used to encrypt/decrypt the configuration file").
		Envar("PFW_AGENT_CONFIG_FILE_KEY_FILE").StringVar(&cfg.Encryption.KeyFile)
	app.Flag("config-file-key-password", "Password for the key file (if required)").
		Envar("PFW_AGENT_CONFIG_FILE_KEY_PASSWORD").StringVar(&cfg.Encryption.KeyFilePassword)

	app.Flag("id", "ID of this pfw-agent [PFW_AGENT_ID]").
		Envar("PFW_AGENT_ID").StringVar(&cfg.ID)
	app.Flag("listen-address", "Agent local API address [PFW_AGENT_LISTEN_ADDRESS]").
		Envar("PFW_AGENT_LISTEN_ADDRESS").StringVar(&cfg.ListenAddress)
	app.Flag("listen-port", "Agent local API port [PFW_AGENT_LISTEN_PORT]").
		Envar("PFW_AGENT_LISTEN_PORT").Uint16Var(&cfg.ListenPort)
	app.Flag("runner-capacity", "Agent internal actions/jobs runner capacity [PFW_AGENT_RUNNER_CAPACITY]").
		Envar("PFW_AGENT_RUNNER_CAPACITY").Uint16Var(&cfg.RunnerCapacity)
	app.Flag("runner-max-connections-per-service", "Agent internal action/job runner connection limit per DB instance").
		Envar("PFW_AGENT_RUNNER_MAX_CONNECTIONS_PER_SERVICE").Uint16Var(&cfg.RunnerMaxConnectionsPerService)

	app.Flag("server-address", "Postgres1st WatchTower Server address [PFW_AGENT_SERVER_ADDRESS]").
		Envar("PFW_AGENT_SERVER_ADDRESS").PlaceHolder("<host:port>").StringVar(&cfg.Server.Address)
	app.Flag("server-username", "Username to connect to Postgres1st WatchTower Server [PFW_AGENT_SERVER_USERNAME]").
		Envar("PFW_AGENT_SERVER_USERNAME").StringVar(&cfg.Server.Username)
	app.Flag("server-password", "Password to connect to Postgres1st WatchTower Server [PFW_AGENT_SERVER_PASSWORD]").
		Envar("PFW_AGENT_SERVER_PASSWORD").StringVar(&cfg.Server.Password)
	app.Flag("server-insecure-tls", "Skip Postgres1st WatchTower Server TLS certificate validation [PFW_AGENT_SERVER_INSECURE_TLS]").
		Envar("PFW_AGENT_SERVER_INSECURE_TLS").BoolVar(&cfg.Server.InsecureTLS)
	// no flag for WithoutTLS - it is only for development and testing

	app.Flag("paths-base", "Base path for exporters/collectors/tools to use [PFW_AGENT_PATHS_BASE]").
		Envar("PFW_AGENT_PATHS_BASE").StringVar(&cfg.Paths.PathsBase)
	app.Flag("paths-exporters_base", "Base path for exporters to use [PFW_AGENT_PATHS_EXPORTERS_BASE]").
		Envar("PFW_AGENT_PATHS_EXPORTERS_BASE").StringVar(&cfg.Paths.ExportersBase)
	app.Flag("paths-node_exporter", "Path to node_exporter to use [PFW_AGENT_PATHS_NODE_EXPORTER]").
		Envar("PFW_AGENT_PATHS_NODE_EXPORTER").StringVar(&cfg.Paths.NodeExporter)
	app.Flag("paths-mysqld_exporter", "Path to mysqld_exporter to use [PFW_AGENT_PATHS_MYSQLD_EXPORTER]").
		Envar("PFW_AGENT_PATHS_MYSQLD_EXPORTER").StringVar(&cfg.Paths.MySQLdExporter)
	app.Flag("paths-mongodb_exporter", "Path to mongodb_exporter to use [PFW_AGENT_PATHS_MONGODB_EXPORTER]").
		Envar("PFW_AGENT_PATHS_MONGODB_EXPORTER").StringVar(&cfg.Paths.MongoDBExporter)
	app.Flag("paths-postgres_exporter", "Path to postgres_exporter to use [PFW_AGENT_PATHS_POSTGRES_EXPORTER]").
		Envar("PFW_AGENT_PATHS_POSTGRES_EXPORTER").StringVar(&cfg.Paths.PostgresExporter)
	app.Flag("paths-proxysql_exporter", "Path to proxysql_exporter to use [PFW_AGENT_PATHS_PROXYSQL_EXPORTER]").
		Envar("PFW_AGENT_PATHS_PROXYSQL_EXPORTER").StringVar(&cfg.Paths.ProxySQLExporter)
	app.Flag("paths-azure_exporter", "Path to azure_exporter to use [PFW_AGENT_PATHS_AZURE_EXPORTER]").
		Envar("PFW_AGENT_PATHS_AZURE_EXPORTER").StringVar(&cfg.Paths.AzureExporter)
	app.Flag("paths-valkey-exporter", "Path to valkey_exporter to use [PFW_AGENT_PATHS_VALKEY_EXPORTER]").
		Envar("PFW_AGENT_PATHS_VALKEY_EXPORTER").StringVar(&cfg.Paths.ValkeyExporter)
	app.Flag("paths-pt-summary", "Path to pt summary to use [PFW_AGENT_PATHS_PT_SUMMARY]").
		Envar("PFW_AGENT_PATHS_PT_SUMMARY").StringVar(&cfg.Paths.PTSummary)
	app.Flag("paths-pt-pg-summary", "Path to pt-pg-summary to use [PFW_AGENT_PATHS_PT_PG_SUMMARY]").
		Envar("PFW_AGENT_PATHS_PT_PG_SUMMARY").StringVar(&cfg.Paths.PTPGSummary)
	app.Flag("paths-pt-mongodb-summary", "Path to pt mongodb summary to use [PFW_AGENT_PATHS_PT_MONGODB_SUMMARY]").
		Envar("PFW_AGENT_PATHS_PT_MONGODB_SUMMARY").StringVar(&cfg.Paths.PTMongoDBSummary)
	app.Flag("paths-pt-mysql-summary", "Path to pt my sql summary to use [PFW_AGENT_PATHS_PT_MYSQL_SUMMARY]").
		Envar("PFW_AGENT_PATHS_PT_MYSQL_SUMMARY").StringVar(&cfg.Paths.PTMySQLSummary)
	app.Flag("paths-nomad", "Path to nomad binary. Can be overridden using [PFW_AGENT_PATHS_NOMAD]").
		Envar("PFW_AGENT_PATHS_NOMAD").StringVar(&cfg.Paths.Nomad)
	app.Flag("paths-nomad-data-dir", "Nomad data directory [PFW_AGENT_PATHS_NOMAD_DATA_DIR]").
		Envar("PFW_AGENT_PATHS_NOMAD_DATA_DIR").StringVar(&cfg.Paths.NomadDataDir)
	app.Flag("paths-tempdir", "Temporary directory for exporters [PFW_AGENT_PATHS_TEMPDIR]").
		Envar("PFW_AGENT_PATHS_TEMPDIR").StringVar(&cfg.Paths.TempDir)
	// no flag for SlowLogFilePrefix - it is only for development and testing

	app.Flag("ports-min", "Minimal allowed port number for listening sockets [PFW_AGENT_PORTS_MIN]").
		Envar("PFW_AGENT_PORTS_MIN").Uint16Var(&cfg.Ports.Min)
	app.Flag("ports-max", "Maximal allowed port number for listening sockets [PFW_AGENT_PORTS_MAX]").
		Envar("PFW_AGENT_PORTS_MAX").Uint16Var(&cfg.Ports.Max)
	app.Flag("window-connected-time", "Window time for which we track the status of connection between agent and server").
		Envar("PFW_AGENT_WINDOW_CONNECTED_TIME").DurationVar(&cfg.WindowConnectedTime)

	app.Flag("log-level", "Set logging level [PFW_AGENT_LOG_LEVEL]").
		Envar("PFW_AGENT_LOG_LEVEL").EnumVar(&cfg.LogLevel, "debug", "info", "warn", "error", "fatal")
	app.Flag("debug", "Enable debug output [PFW_AGENT_DEBUG]").
		Envar("PFW_AGENT_DEBUG").BoolVar(&cfg.Debug)
	app.Flag("trace", "Enable trace output (implies debug) [PFW_AGENT_TRACE]").
		Envar("PFW_AGENT_TRACE").BoolVar(&cfg.Trace)
	app.Flag("log-lines-count",
		"Take and return N most recent log lines in logs.zip for each: server, every configured exporters and agents [PFW_AGENT_LOG_LINES_COUNT]").
		Envar("PFW_AGENT_LOG_LINES_COUNT").Default("1024").UintVar(&cfg.LogLinesCount)
	app.Flag("perfschema-refresh-rate",
		"Change how often PMM scrapes data from Performance Schema (in seconds) [PFW_AGENT_PERFSCHEMA_REFRESH_RATE]").
		Envar("PFW_AGENT_PERFSCHEMA_REFRESH_RATE").Uint16Var(&cfg.PerfschemaRefreshRate)
	jsonF := app.Flag("json", "Enable JSON output").Action(func(*kingpin.ParseContext) error {
		logrus.SetFormatter(&logrus.JSONFormatter{}) // with levels and timestamps always present
		return nil
	}).Bool()

	app.Flag("version", "Show application version").Short('v').Action(func(*kingpin.ParseContext) error {
		// We use fmt instead of log package to provide proper output for --json flag.
		if *jsonF {
			fmt.Println(version.FullInfoJSON()) //nolint:forbidigo
		} else {
			fmt.Println(version.FullInfo()) //nolint:forbidigo
		}
		os.Exit(0)

		return nil
	}).Bool()

	setupCmd := app.Command("setup", "Configure local pfw-agent")
	nodeinfo := nodeinfo.Get()

	if nodeinfo.PublicAddress == "" {
		help := "Node address [PFW_AGENT_SETUP_NODE_ADDRESS]"
		setupCmd.Arg("node-address", help).Required().
			Envar("PFW_AGENT_SETUP_NODE_ADDRESS").StringVar(&cfg.Setup.Address)
	} else {
		help := fmt.Sprintf("Node address (autodetected default: %s) [PFW_AGENT_SETUP_NODE_ADDRESS]", nodeinfo.PublicAddress)
		setupCmd.Arg("node-address", help).Default(nodeinfo.PublicAddress).
			Envar("PFW_AGENT_SETUP_NODE_ADDRESS").StringVar(&cfg.Setup.Address)
	}

	nodeTypeKeys := []string{"generic", "container"}
	nodeTypeDefault := "generic"
	if nodeinfo.Container {
		nodeTypeDefault = "container"
	}
	nodeTypeHelp := fmt.Sprintf("Node type, one of: %s (default: %s) [PFW_AGENT_SETUP_NODE_TYPE]", strings.Join(nodeTypeKeys, ", "), nodeTypeDefault)
	setupCmd.Arg("node-type", nodeTypeHelp).Default(nodeTypeDefault).
		Envar("PFW_AGENT_SETUP_NODE_TYPE").EnumVar(&cfg.Setup.NodeType, nodeTypeKeys...)

	hostname, _ := os.Hostname()
	nodeNameHelp := fmt.Sprintf("Node name (autodetected default: %s) [PFW_AGENT_SETUP_NODE_NAME]", hostname)
	setupCmd.Arg("node-name", nodeNameHelp).Default(hostname).
		Envar("PFW_AGENT_SETUP_NODE_NAME").StringVar(&cfg.Setup.NodeName)

	var defaultMachineID string
	if nodeinfo.MachineID != "" {
		defaultMachineID = nodeinfo.MachineID
	}
	setupCmd.Flag("machine-id", "Node machine-id (default is autodetected) [PFW_AGENT_SETUP_MACHINE_ID]").Default(defaultMachineID).
		Envar("PFW_AGENT_SETUP_MACHINE_ID").StringVar(&cfg.Setup.MachineID)
	setupCmd.Flag("distro", "Node OS distribution (default is autodetected) [PFW_AGENT_SETUP_DISTRO]").Default(nodeinfo.Distro).
		Envar("PFW_AGENT_SETUP_DISTRO").StringVar(&cfg.Setup.Distro)
	setupCmd.Flag("container-id", "Container ID [PFW_AGENT_SETUP_CONTAINER_ID]").
		Envar("PFW_AGENT_SETUP_CONTAINER_ID").StringVar(&cfg.Setup.ContainerID)
	setupCmd.Flag("container-name", "Container name [PFW_AGENT_SETUP_CONTAINER_NAME]").
		Envar("PFW_AGENT_SETUP_CONTAINER_NAME").StringVar(&cfg.Setup.ContainerName)
	setupCmd.Flag("node-model", "Node model [PFW_AGENT_SETUP_NODE_MODEL]").
		Envar("PFW_AGENT_SETUP_NODE_MODEL").StringVar(&cfg.Setup.NodeModel)
	setupCmd.Flag("region", "Node region [PFW_AGENT_SETUP_REGION]").
		Envar("PFW_AGENT_SETUP_REGION").StringVar(&cfg.Setup.Region)
	setupCmd.Flag("az", "Node availability zone [PFW_AGENT_SETUP_AZ]").
		Envar("PFW_AGENT_SETUP_AZ").StringVar(&cfg.Setup.Az)

	setupCmd.Flag("force", "Remove Node with that name with all dependent Services and Agents if one exist [PFW_AGENT_SETUP_FORCE]").
		Envar("PFW_AGENT_SETUP_FORCE").BoolVar(&cfg.Setup.Force)
	setupCmd.Flag("skip-registration", "Skip registration on Postgres1st WatchTower Server [PFW_AGENT_SETUP_SKIP_REGISTRATION]").
		Envar("PFW_AGENT_SETUP_SKIP_REGISTRATION").BoolVar(&cfg.Setup.SkipRegistration)
	setupCmd.Flag("metrics-mode", "Metrics flow mode for agents node-exporter, can be push - agent will push metrics,"+
		"pull - server scrape metrics from agent  or auto - chosen by server. [PFW_AGENT_SETUP_METRICS_MODE]").
		Envar("PFW_AGENT_SETUP_METRICS_MODE").Default("auto").EnumVar(&cfg.Setup.MetricsMode, "auto", "push", "pull")
	setupCmd.Flag("disable-collectors", "Comma-separated list of collector names to exclude from exporter. [PFW_AGENT_SETUP_DISABLE_COLLECTORS]").
		Envar("PFW_AGENT_SETUP_DISABLE_COLLECTORS").Default("").StringVar(&cfg.Setup.DisableCollectors)
	setupCmd.Flag("custom-labels", "Custom labels [PFW_AGENT_SETUP_CUSTOM_LABELS]").
		Envar("PFW_AGENT_SETUP_CUSTOM_LABELS").StringVar(&cfg.Setup.CustomLabels)
	setupCmd.Flag("agent-password", "Custom password for /metrics endpoint [PFW_AGENT_SETUP_NODE_PASSWORD]").
		Envar("PFW_AGENT_SETUP_NODE_PASSWORD").StringVar(&cfg.Setup.AgentPassword)
	setupCmd.Flag("expose-exporter", "Expose the address of the agent's node-exporter publicly on 0.0.0.0").
		Envar("PFW_AGENT_EXPOSE_EXPORTER").BoolVar(&cfg.Setup.ExposeExporter)
	setupCmd.Flag("proc-mounts-path", "Path to /proc/mounts file for the filesystem collector [PFW_AGENT_SETUP_PROC_MOUNTS_PATH]").
		Envar("PFW_AGENT_SETUP_PROC_MOUNTS_PATH").StringVar(&cfg.Setup.ProcMountsPath)

	return app, configFileF
}

// loadFromFile loads configuration from file.
// As a special case, if file does not exist, it returns ConfigFileDoesNotExistError.
// Other errors are returned if file exists, but configuration can't be loaded due to permission problems,
// YAML parsing problems, etc.
func loadFromFile(path string, enc *Encryption) (*Config, error) {
	_, err := os.Stat(path)
	if errors.Is(err, fs.ErrNotExist) {
		return nil, ConfigFileDoesNotExistError(path)
	}

	b, err := os.ReadFile(path) //nolint:gosec
	if err != nil {
		return nil, err
	}

	if enc != nil && len(enc.KeyFile) != 0 && len(b) != 0 {
		b, err = enc.Decrypt(b)
		if err != nil {
			return nil, err
		}
	}

	cfg := &Config{}
	err = yaml.Unmarshal(b, cfg) //nolint:musttag // false positive
	if err != nil {
		return nil, err
	}

	return cfg, nil
}

// SaveToFile saves configuration to file.
// No special cases.
func SaveToFile(path string, cfg *Config, comment string) error {
	b, err := yaml.Marshal(cfg) //nolint:musttag // false positive
	if err != nil {
		return err
	}

	var res []byte
	if comment != "" {
		res = []byte("# " + comment + "\n")
	}

	res = append(res, "---\n"...)
	res = append(res, b...)
	if cfg != nil && len(cfg.Encryption.KeyFile) != 0 {
		res, err = cfg.Encryption.Encrypt(res)
		if err != nil {
			return err
		}
	}

	return os.WriteFile(path, res, 0o640) //nolint:gosec,mnd
}

// IsWritable checks if specified path is writable.
func IsWritable(path string) error {
	_, err := os.Stat(path)
	if err != nil {
		// File doesn't exist, check if folder is writable.
		if errors.Is(err, fs.ErrNotExist) {
			return unix.Access(filepath.Dir(path), unix.W_OK)
		}
		return err
	}

	return unix.Access(path, unix.W_OK)
}
