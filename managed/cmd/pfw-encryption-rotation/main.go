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
// Package main is the main package for encryption keys rotation.
package main

import (
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/alecthomas/kong"
	"github.com/sirupsen/logrus"

	"github.com/percona/pmm/managed/models"
	encryptionService "github.com/percona/pmm/managed/services/encryption"
	"github.com/percona/pmm/managed/utils/dbsecret"
	"github.com/percona/pmm/managed/utils/encryption"
	"github.com/percona/pmm/utils/logger"
	"github.com/percona/pmm/version"
)

const codeDBConnectionFailed = 1

func main() {
	signal.Ignore(syscall.SIGINT, syscall.SIGTERM) // to prevent any interuptions during process

	logger.SetupGlobalLogger()

	var opts flags
	kong.Parse(
		&opts,
		kong.Name("encryption-rotation"),
		kong.Description("Version "+version.Version),
		kong.UsageOnError(),
		kong.ConfigureHelp(kong.HelpOptions{
			Compact:             true,
			NoExpandSubcommands: true,
		}),
		kong.Vars{
			"address":             models.DefaultPostgreSQLAddr,
			"disable_sslmode":     models.DisableSSLMode,
			"require_sslmode":     models.RequireSSLMode,
			"verify_sslmode":      models.VerifyCaSSLMode,
			"verify_full_sslmode": models.VerifyFullSSLMode,
		},
	)

	if opts.GenerateKey {
		e := &encryption.Encryption{}
		key, err := e.GenerateKey()
		if err != nil {
			logrus.Errorf("Failed to generate key: %v", err)
			os.Exit(1)
		}
		fmt.Print(key) //nolint:forbidigo
		os.Exit(0)
	}

	sqlDB, err := models.OpenDB(setupParams(opts))
	if err != nil {
		logrus.Error(err)
		os.Exit(codeDBConnectionFailed)
	}

	statusCode, err := encryptionService.RotateEncryptionKey(sqlDB, "pmm-managed")
	sqlDB.Close() //nolint:errcheck
	if err != nil {
		logrus.Error(err)
		os.Exit(statusCode)
	}
}

//nolint:lll
type flags struct {
	Address     string `name:"postgres-addr" default:"${address}" help:"PostgreSQL address with port"`
	DBName      string `name:"postgres-name" default:"pmm-managed" help:"PostgreSQL database name"`
	DBUsername  string `name:"postgres-username" default:"pmm-managed" help:"PostgreSQL database username name"`
	DBPassword  string `name:"postgres-password" help:"PostgreSQL database password. If omitted OR set to an empty string, it is read from /srv/.pfw-secrets/managed-db.env, falling back to \"pmm-managed\"."`
	SSLMode     string `name:"postgres-ssl-mode" default:"${disable_sslmode}" help:"PostgreSQL SSL mode" enum:"${disable_sslmode}, ${require_sslmode},${verify_sslmode}, ${verify_full_sslmode}"`
	SSLCAPath   string `name:"postgres-ssl-ca-path" help:"PostgreSQL SSL CA root certificate path" type:"path"`
	SSLKeyPath  string `name:"postgres-ssl-key-path" help:"PostgreSQL SSL key path" type:"path"`
	SSLCertPath string `name:"postgres-ssl-cert-path" help:"PostgreSQL SSL certificate path" type:"path"`
	GenerateKey bool   `name:"generate-key" help:"Only generate a new encryption key and print to stdout"`
}

// The pmm-managed role password is generated at first boot on a native install, so
// this tool can no longer carry it as a flag default -- the documented invocation
// in documentation/docs/admin/security/data_encryption.md passes no --postgres-password.
// Resolving it through dbsecret keeps that invocation working; the same package
// backs pfw-managed's own resolution, so the two cannot drift.
func setupParams(opts flags) models.SetupDBParams {
	password := opts.DBPassword
	if password == "" {
		password = dbsecret.ManagedDBPassword()
	}
	return models.SetupDBParams{
		Address:     opts.Address,
		Name:        opts.DBName,
		Username:    opts.DBUsername,
		Password:    password,
		SSLMode:     opts.SSLMode,
		SSLCAPath:   opts.SSLCAPath,
		SSLKeyPath:  opts.SSLKeyPath,
		SSLCertPath: opts.SSLCertPath,
	}
}
