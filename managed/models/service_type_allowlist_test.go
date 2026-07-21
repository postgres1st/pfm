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

package models

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func TestParseSupportedServiceTypes(t *testing.T) {
	t.Parallel()

	t.Run("single type", func(t *testing.T) {
		t.Parallel()

		res, err := parseSupportedServiceTypes("postgresql")
		require.NoError(t, err)
		assert.Equal(t, map[ServiceType]struct{}{PostgreSQLServiceType: {}}, res)
	})

	t.Run("multiple types", func(t *testing.T) {
		t.Parallel()

		res, err := parseSupportedServiceTypes("postgresql,proxysql")
		require.NoError(t, err)
		assert.Equal(t, map[ServiceType]struct{}{
			PostgreSQLServiceType: {},
			ProxySQLServiceType:   {},
		}, res)
	})

	t.Run("surrounding whitespace is tolerated", func(t *testing.T) {
		t.Parallel()

		res, err := parseSupportedServiceTypes(" postgresql , proxysql ")
		require.NoError(t, err)
		assert.Equal(t, map[ServiceType]struct{}{
			PostgreSQLServiceType: {},
			ProxySQLServiceType:   {},
		}, res)
	})

	t.Run("duplicates collapse", func(t *testing.T) {
		t.Parallel()

		res, err := parseSupportedServiceTypes("postgresql,postgresql")
		require.NoError(t, err)
		assert.Equal(t, map[ServiceType]struct{}{PostgreSQLServiceType: {}}, res)
	})

	// An operator who sets the variable at all is making a deliberate statement.
	// Anything that does not resolve to at least one real type is a misconfiguration
	// we refuse rather than paper over, so a typo cannot silently ship the default.
	t.Run("unknown type is rejected", func(t *testing.T) {
		t.Parallel()

		_, err := parseSupportedServiceTypes("postgres")
		require.Error(t, err)
		assert.Contains(t, err.Error(), "postgres")
	})

	t.Run("one bad entry rejects the whole list", func(t *testing.T) {
		t.Parallel()

		_, err := parseSupportedServiceTypes("postgresql,mysqll")
		require.Error(t, err)
		assert.Contains(t, err.Error(), "mysqll")
	})

	t.Run("empty value is rejected", func(t *testing.T) {
		t.Parallel()

		_, err := parseSupportedServiceTypes("")
		require.Error(t, err)
	})

	t.Run("separators only is rejected", func(t *testing.T) {
		t.Parallel()

		_, err := parseSupportedServiceTypes(" , ")
		require.Error(t, err)
	})
}

// Not parallel: the subtests swap the package-level allowlist.
func TestIsServiceTypeSupported(t *testing.T) {
	// PostgreSQL is the product. External carries Patroni, which exposes its own /metrics
	// endpoint and has no exporter binary, and HAProxy fronts Patroni clusters and is scraped
	// natively — dropping either would remove a PostgreSQL HA capability, not a MySQL one.
	t.Run("default allowlist is PostgreSQL, external and HAProxy", func(t *testing.T) {
		assert.Equal(t, []ServiceType{
			PostgreSQLServiceType,
			HAProxyServiceType,
			ExternalServiceType,
		}, defaultSupportedServiceTypes)
	})

	t.Run("permitted type is supported", func(t *testing.T) {
		restore := SetSupportedServiceTypesForTests(PostgreSQLServiceType)
		defer restore()

		assert.True(t, IsServiceTypeSupported(PostgreSQLServiceType))
	})

	t.Run("forbidden type is not supported", func(t *testing.T) {
		restore := SetSupportedServiceTypesForTests(PostgreSQLServiceType)
		defer restore()

		assert.False(t, IsServiceTypeSupported(MySQLServiceType))
	})

	t.Run("allowlist is honoured when widened", func(t *testing.T) {
		restore := SetSupportedServiceTypesForTests(PostgreSQLServiceType, MySQLServiceType)
		defer restore()

		assert.True(t, IsServiceTypeSupported(MySQLServiceType))
	})
}

// Not parallel: the subtests swap the package-level allowlist.
//
// A nil Querier is passed deliberately. The allowlist check must happen before AddNewService
// touches the database, so a rejected type cannot reach validation, uniqueness checks or the
// DSN builders — the latter panic on types they do not handle.
func TestAddNewServiceRejectsUnsupportedType(t *testing.T) {
	t.Run("forbidden type is rejected before any database access", func(t *testing.T) {
		restore := SetSupportedServiceTypesForTests(PostgreSQLServiceType)
		defer restore()

		service, err := AddNewService(nil, MySQLServiceType, &AddDBMSServiceParams{
			ServiceName: "test-mysql",
			NodeID:      "test-node",
		})

		require.Error(t, err)
		assert.Nil(t, service)
		assert.Equal(t, codes.InvalidArgument, status.Code(err))
		assert.Contains(t, err.Error(), "mysql")
	})

	t.Run("every non-permitted type is rejected", func(t *testing.T) {
		restore := SetSupportedServiceTypesForTests(PostgreSQLServiceType)
		defer restore()

		for _, serviceType := range allServiceTypes {
			if serviceType == PostgreSQLServiceType {
				continue
			}

			_, err := AddNewService(nil, serviceType, &AddDBMSServiceParams{ServiceName: "test", NodeID: "node"})
			require.Errorf(t, err, "service type %q was not rejected", serviceType)
			assert.Equalf(t, codes.InvalidArgument, status.Code(err), "service type %q", serviceType)
		}
	})
}

// Not parallel: the subtests swap the package-level allowlist.
func TestIsAgentTypeSupported(t *testing.T) {
	t.Run("exporter for a permitted service type is supported", func(t *testing.T) {
		restore := SetSupportedServiceTypesForTests(PostgreSQLServiceType)
		defer restore()

		assert.True(t, IsAgentTypeSupported(PostgresExporterType))
		assert.True(t, IsAgentTypeSupported(QANPostgreSQLPgStatementsAgentType))
	})

	t.Run("exporter for a forbidden service type is not supported", func(t *testing.T) {
		restore := SetSupportedServiceTypesForTests(PostgreSQLServiceType)
		defer restore()

		assert.False(t, IsAgentTypeSupported(MySQLdExporterType))
		assert.False(t, IsAgentTypeSupported(MongoDBExporterType))
		assert.False(t, IsAgentTypeSupported(ProxySQLExporterType))
		assert.False(t, IsAgentTypeSupported(ValkeyExporterType))
		assert.False(t, IsAgentTypeSupported(QANMySQLSlowlogAgentType))
		assert.False(t, IsAgentTypeSupported(RTAMongoDBAgentType))
	})

	// Agents not bound to a Service at all — they attach to Nodes or to PMM itself — must
	// stay available regardless of which database types are permitted.
	t.Run("agents not bound to a service are always supported", func(t *testing.T) {
		restore := SetSupportedServiceTypesForTests(PostgreSQLServiceType)
		defer restore()

		assert.True(t, IsAgentTypeSupported(NodeExporterType))
		assert.True(t, IsAgentTypeSupported(VMAgentType))
		assert.True(t, IsAgentTypeSupported(PMMAgentType))
	})

	// These serve several service types; one permitted type is enough to keep them.
	t.Run("multi-service exporters survive if any of their types is permitted", func(t *testing.T) {
		restore := SetSupportedServiceTypesForTests(PostgreSQLServiceType)
		defer restore()

		assert.True(t, IsAgentTypeSupported(RDSExporterType))
		assert.True(t, IsAgentTypeSupported(AzureDatabaseExporterType))
	})

	t.Run("external exporter follows the external service type", func(t *testing.T) {
		restore := SetSupportedServiceTypesForTests(PostgreSQLServiceType, ExternalServiceType)
		defer restore()
		assert.True(t, IsAgentTypeSupported(ExternalExporterType))

		restore2 := SetSupportedServiceTypesForTests(PostgreSQLServiceType)
		defer restore2()
		assert.False(t, IsAgentTypeSupported(ExternalExporterType))
	})
}

// Not parallel: the subtests swap the package-level allowlist.
//
// A nil Querier is passed deliberately: the check must happen before CreateAgent touches
// the database. The ServiceID is empty because that is the path the compatibility matrix
// does not cover — with a ServiceID set, an unsupported exporter is already rejected by
// compatibleServiceAndAgent, since no Service of its type can exist.
func TestCreateAgentRejectsUnsupportedType(t *testing.T) {
	t.Run("forbidden agent type is rejected before any database access", func(t *testing.T) {
		restore := SetSupportedServiceTypesForTests(PostgreSQLServiceType)
		defer restore()

		agent, err := CreateAgent(nil, MySQLdExporterType, &CreateAgentParams{})

		require.Error(t, err)
		assert.Nil(t, agent)
		assert.Equal(t, codes.FailedPrecondition, status.Code(err))
		assert.Contains(t, err.Error(), string(MySQLdExporterType))
	})
}

// Not parallel: the subtests swap the package-level allowlist.
func TestSupportedServiceTypes(t *testing.T) {
	t.Run("returns types in service_model declaration order", func(t *testing.T) {
		restore := SetSupportedServiceTypesForTests(ValkeyServiceType, MySQLServiceType, PostgreSQLServiceType)
		defer restore()

		assert.Equal(t, []ServiceType{MySQLServiceType, PostgreSQLServiceType, ValkeyServiceType}, SupportedServiceTypes())
	})
}
