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

package models_test

import (
	"testing"
	"time"

	"github.com/AlekSi/pointer"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"gopkg.in/reform.v1"
	"gopkg.in/reform.v1/dialects/postgresql"

	"github.com/percona/pmm/managed/models"
	"github.com/percona/pmm/managed/utils/testdb"
	"github.com/percona/pmm/managed/utils/tests"
)

// skipIfServiceTypeUnsupported skips tests covering a service type this build does not
// accept. Gated on the allowlist rather than commented out, so widening PFM_DB_TYPES
// restores the coverage without editing tests.
func skipIfServiceTypeUnsupported(t *testing.T, serviceType models.ServiceType) {
	t.Helper()

	if !models.IsServiceTypeSupported(serviceType) {
		t.Skipf("Service type %q is not supported by this deployment.", serviceType)
	}
}

func TestServiceHelpers(t *testing.T) {
	now, origNowF := models.Now(), models.Now
	models.Now = func() time.Time {
		return now
	}
	sqlDB := testdb.Open(t, models.SkipFixtures, nil)
	defer func() {
		models.Now = origNowF
		require.NoError(t, sqlDB.Close())
	}()

	setup := func(t *testing.T) (*reform.Querier, func(t *testing.T)) {
		t.Helper()
		db := reform.NewDB(sqlDB, postgresql.Dialect, reform.NewPrintfLogger(t.Logf))
		tx, err := db.Begin()
		require.NoError(t, err)
		q := tx.Querier

		for _, str := range []reform.Struct{
			&models.Node{
				NodeID:   "N1",
				NodeType: models.GenericNodeType,
				NodeName: "Node",
			},
			&models.Node{
				NodeID:   "N2",
				NodeType: models.GenericNodeType,
				NodeName: "Node 2",
			},

			&models.Service{
				ServiceID:   "S1",
				ServiceType: models.MongoDBServiceType,
				ServiceName: "Service without Agents",
				NodeID:      "N1",
				Address:     new("127.0.0.1"),
				Port:        pointer.ToUint16OrNil(27017),
			},
			&models.Service{
				ServiceID:   "S2",
				ServiceType: models.MySQLServiceType,
				ServiceName: "Service with Agents",
				NodeID:      "N1",
				Address:     new("127.0.0.1"),
				Port:        pointer.ToUint16OrNil(3306),
			},
			&models.Service{
				ServiceID:   "S21",
				ServiceType: models.ValkeyServiceType,
				ServiceName: "Standalone Valkey Service",
				NodeID:      "N1",
				Address:     new("127.0.0.1"),
				Port:        pointer.ToUint16OrNil(6379),
			},
			&models.Service{
				ServiceID:   "S3",
				ServiceType: models.MySQLServiceType,
				ServiceName: "Third service",
				NodeID:      "N2",
				Socket:      pointer.ToStringOrNil("/var/run/mysqld/mysqld.sock"),
			},
			&models.Service{
				ServiceID:     "S4",
				ServiceType:   models.ExternalServiceType,
				ExternalGroup: "external",
				ServiceName:   "Fourth service",
				NodeID:        "N2",
			},
			&models.Service{
				ServiceID:   "S5",
				ServiceType: models.ProxySQLServiceType,
				ServiceName: "Fifth service",
				NodeID:      "N1",
				Address:     new("127.0.0.1"),
				Port:        pointer.ToUint16OrNil(6032),
			},
			&models.Service{
				ServiceID:   "S6",
				ServiceType: models.ProxySQLServiceType,
				ServiceName: "Sixth service",
				NodeID:      "N2",
				Socket:      pointer.ToStringOrNil("/tmp/proxysql_admin.sock"),
			},
			&models.Service{
				ServiceID:     "S7",
				ServiceType:   models.ExternalServiceType,
				ServiceName:   "Seventh service",
				NodeID:        "N2",
				Address:       new("127.0.0.1"),
				Port:          pointer.ToUint16OrNil(6379),
				ExternalGroup: "redis",
				CreatedAt:     now,
				UpdatedAt:     now,
			},
			&models.Service{
				ServiceID:   "S8",
				ServiceType: models.HAProxyServiceType,
				ServiceName: "Eighth service",
				NodeID:      "N2",
			},

			&models.Agent{
				AgentID:      "A1",
				AgentType:    models.PMMAgentType,
				RunsOnNodeID: new("N1"),
			},
			&models.Agent{
				AgentID:    "A2",
				AgentType:  models.MySQLdExporterType,
				PMMAgentID: new("A1"),
				ServiceID:  new("S2"),
			},
		} {
			require.NoError(t, q.Insert(str))
		}

		teardown := func(t *testing.T) {
			t.Helper()
			require.NoError(t, tx.Rollback())
		}
		return q, teardown
	}

	t.Run("FindServices", func(t *testing.T) {
		q, teardown := setup(t)
		defer teardown(t)

		services, err := models.FindServices(q, models.ServiceFilters{})
		require.NoError(t, err)
		assert.Len(t, services, 9)

		services, err = models.FindServices(q, models.ServiceFilters{NodeID: "N1"})
		require.NoError(t, err)
		assert.Len(t, services, 4)
		assert.Equal(t, []*models.Service{{
			ServiceID:   "S1",
			ServiceType: models.MongoDBServiceType,
			ServiceName: "Service without Agents",
			NodeID:      "N1",
			Address:     new("127.0.0.1"),
			Port:        pointer.ToUint16OrNil(27017),
			CreatedAt:   now,
			UpdatedAt:   now,
		}, {
			ServiceID:   "S2",
			ServiceType: models.MySQLServiceType,
			ServiceName: "Service with Agents",
			NodeID:      "N1",
			Address:     new("127.0.0.1"),
			Port:        pointer.ToUint16OrNil(3306),
			CreatedAt:   now,
			UpdatedAt:   now,
		}, {
			ServiceID:   "S21",
			ServiceType: models.ValkeyServiceType,
			ServiceName: "Standalone Valkey Service",
			NodeID:      "N1",
			Address:     new("127.0.0.1"),
			Port:        pointer.ToUint16OrNil(6379),
			CreatedAt:   now,
			UpdatedAt:   now,
		}, {
			ServiceID:   "S5",
			ServiceType: models.ProxySQLServiceType,
			ServiceName: "Fifth service",
			NodeID:      "N1",
			Address:     new("127.0.0.1"),
			Port:        pointer.ToUint16OrNil(6032),
			CreatedAt:   now,
			UpdatedAt:   now,
		}}, services)

		services, err = models.FindServices(q, models.ServiceFilters{NodeID: "N1", ServiceType: new(models.MySQLServiceType)})
		require.NoError(t, err)
		assert.Len(t, services, 1)
		assert.Equal(t, []*models.Service{{
			ServiceID:   "S2",
			ServiceType: models.MySQLServiceType,
			ServiceName: "Service with Agents",
			NodeID:      "N1",
			Address:     new("127.0.0.1"),
			Port:        pointer.ToUint16OrNil(3306),
			CreatedAt:   now,
			UpdatedAt:   now,
		}}, services)

		services, err = models.FindServices(q, models.ServiceFilters{NodeID: "N2", ServiceType: new(models.ExternalServiceType)})
		require.NoError(t, err)
		assert.Len(t, services, 2)
		assert.Equal(t, []*models.Service{
			{
				ServiceID:     "S4",
				ServiceType:   models.ExternalServiceType,
				ServiceName:   "Fourth service",
				ExternalGroup: "external",
				NodeID:        "N2",
				CreatedAt:     now,
				UpdatedAt:     now,
			},
			{
				ServiceID:     "S7",
				ServiceType:   models.ExternalServiceType,
				ServiceName:   "Seventh service",
				NodeID:        "N2",
				Address:       new("127.0.0.1"),
				Port:          pointer.ToUint16OrNil(6379),
				ExternalGroup: "redis",
				CreatedAt:     now,
				UpdatedAt:     now,
			},
		}, services)

		services, err = models.FindServices(q, models.ServiceFilters{NodeID: "N2", ServiceType: new(models.ProxySQLServiceType)})
		require.NoError(t, err)
		assert.Len(t, services, 1)
		assert.Equal(t, []*models.Service{{
			ServiceID:   "S6",
			ServiceType: models.ProxySQLServiceType,
			ServiceName: "Sixth service",
			Socket:      pointer.ToStringOrNil("/tmp/proxysql_admin.sock"),
			NodeID:      "N2",
			CreatedAt:   now,
			UpdatedAt:   now,
		}}, services)

		services, err = models.FindServices(q, models.ServiceFilters{ExternalGroup: "redis"})
		require.NoError(t, err)
		assert.Len(t, services, 1)
		assert.Equal(t, []*models.Service{{
			ServiceID:     "S7",
			ServiceType:   models.ExternalServiceType,
			ServiceName:   "Seventh service",
			NodeID:        "N2",
			Address:       new("127.0.0.1"),
			Port:          pointer.ToUint16OrNil(6379),
			ExternalGroup: "redis",
			CreatedAt:     now,
			UpdatedAt:     now,
		}}, services)

		services, err = models.FindServices(q, models.ServiceFilters{NodeID: "N2", ServiceType: new(models.HAProxyServiceType)})
		require.NoError(t, err)
		assert.Len(t, services, 1)
		assert.Equal(t, []*models.Service{{
			ServiceID:   "S8",
			ServiceType: models.HAProxyServiceType,
			ServiceName: "Eighth service",
			NodeID:      "N2",
			CreatedAt:   now,
			UpdatedAt:   now,
		}}, services)
	})

	t.Run("FindActiveServiceTypes", func(t *testing.T) {
		q, teardown := setup(t)
		defer teardown(t)

		types, err := models.FindActiveServiceTypes(q)
		require.NoError(t, err)
		assert.Len(t, types, 6)
	})

	t.Run("RemoveService", func(t *testing.T) {
		q, teardown := setup(t)
		defer teardown(t)

		err := models.RemoveService(q, "", models.RemoveRestrict)
		tests.AssertGRPCError(t, status.New(codes.InvalidArgument, `Empty Service ID.`), err)

		err = models.RemoveService(q, "S0", models.RemoveRestrict)
		tests.AssertGRPCError(t, status.New(codes.NotFound, `Service with ID "S0" not found.`), err)

		_, err = models.FindServiceByID(q, "S1")
		require.NoError(t, err)
		err = models.RemoveService(q, "S1", models.RemoveRestrict)
		require.NoError(t, err)
		_, err = models.FindServiceByID(q, "S1")
		tests.AssertGRPCError(t, status.New(codes.NotFound, `Service with ID "S1" not found.`), err)

		err = models.RemoveService(q, "S2", models.RemoveRestrict)
		tests.AssertGRPCError(t, status.New(codes.FailedPrecondition, `Service with ID "S2" has agents.`), err)

		_, err = models.FindServiceByID(q, "S2")
		require.NoError(t, err)
		err = models.RemoveService(q, "S2", models.RemoveCascade)
		require.NoError(t, err)
		_, err = models.FindServiceByID(q, "S2")
		tests.AssertGRPCError(t, status.New(codes.NotFound, `Service with ID "S2" not found.`), err)
	})

	t.Run("MySQL Conflict socket and address", func(t *testing.T) {
		skipIfServiceTypeUnsupported(t, models.MySQLServiceType)

		q, teardown := setup(t)
		defer teardown(t)

		_, err := models.AddNewService(q, models.MySQLServiceType, &models.AddDBMSServiceParams{
			ServiceName: "test-mysql-socket-address",
			NodeID:      "N1",
			Address:     new("127.0.0.1"),
			Port:        new(uint16(3306)),
			Socket:      new("/var/run/mysqld/mysqld.sock"),
		})
		tests.AssertGRPCError(t, status.New(codes.InvalidArgument, `Socket and address cannot be specified together.`), err)
	})

	t.Run("MySQL empty connection", func(t *testing.T) {
		skipIfServiceTypeUnsupported(t, models.MySQLServiceType)

		q, teardown := setup(t)
		defer teardown(t)

		_, err := models.AddNewService(q, models.MySQLServiceType, &models.AddDBMSServiceParams{
			ServiceName: "test-mysql-socket-address",
			NodeID:      "N1",
		})
		tests.AssertGRPCError(t, status.New(codes.InvalidArgument, `Neither socket nor address passed.`), err)
	})

	t.Run("PostgreSQL conflict socket and address", func(t *testing.T) {
		q, teardown := setup(t)
		defer teardown(t)

		_, err := models.AddNewService(q, models.PostgreSQLServiceType, &models.AddDBMSServiceParams{
			ServiceName: "test-postgresql-socket-address",
			NodeID:      "N1",
			Address:     new("127.0.0.1"),
			Port:        new(uint16(5432)),
			Socket:      new("/var/run/postgresql"),
		})
		tests.AssertGRPCError(t, status.New(codes.InvalidArgument, `Socket and address cannot be specified together.`), err)
	})

	t.Run("PostgreSQL empty connection", func(t *testing.T) {
		q, teardown := setup(t)

		defer teardown(t)
		_, err := models.AddNewService(q, models.PostgreSQLServiceType, &models.AddDBMSServiceParams{
			ServiceName: "test-postgresql-socket-address",
			NodeID:      "N1",
		})
		tests.AssertGRPCError(t, status.New(codes.InvalidArgument, `Neither socket nor address passed.`), err)
	})

	t.Run("MongoDB conflict socket and address", func(t *testing.T) {
		skipIfServiceTypeUnsupported(t, models.MongoDBServiceType)

		q, teardown := setup(t)
		defer teardown(t)

		_, err := models.AddNewService(q, models.MongoDBServiceType, &models.AddDBMSServiceParams{
			ServiceName: "test-mongodb-socket-address",
			NodeID:      "N1",
			Address:     new("127.0.0.1"),
			Port:        new(uint16(27017)),
			Socket:      new("/tmp/mongodb-27017.sock"),
		})
		tests.AssertGRPCError(t, status.New(codes.InvalidArgument, `Socket and address cannot be specified together.`), err)
	})

	t.Run("MongoDB empty connection", func(t *testing.T) {
		skipIfServiceTypeUnsupported(t, models.MongoDBServiceType)

		q, teardown := setup(t)
		defer teardown(t)

		_, err := models.AddNewService(q, models.MongoDBServiceType, &models.AddDBMSServiceParams{
			ServiceName: "test-mongodb-socket-address",
		})
		tests.AssertGRPCError(t, status.New(codes.InvalidArgument, `Neither socket nor address passed.`), err)
	})

	t.Run("ProxySQL empty connection", func(t *testing.T) {
		skipIfServiceTypeUnsupported(t, models.ProxySQLServiceType)

		q, teardown := setup(t)
		defer teardown(t)

		_, err := models.AddNewService(q, models.ProxySQLServiceType, &models.AddDBMSServiceParams{
			ServiceName: "test-proxysql-socket-address",
			NodeID:      "N1",
		})
		tests.AssertGRPCError(t, status.New(codes.InvalidArgument, `Neither socket nor address passed.`), err)
	})

	t.Run("ProxySQL conflict socket and address", func(t *testing.T) {
		skipIfServiceTypeUnsupported(t, models.ProxySQLServiceType)

		q, teardown := setup(t)
		defer teardown(t)

		_, err := models.AddNewService(q, models.ProxySQLServiceType, &models.AddDBMSServiceParams{
			ServiceName: "test-proxysql-socket-address",
			NodeID:      "N1",
			Address:     new("127.0.0.1"),
			Port:        new(uint16(6032)),
			Socket:      new("/tmp/proxysql_admin.sock"),
		})
		tests.AssertGRPCError(t, status.New(codes.InvalidArgument, `Socket and address cannot be specified together.`), err)
	})

	// The cluster filter is service-type agnostic; PostgreSQL exercises it as well as MongoDB did.
	t.Run("PostgreSQL find services in the same cluster", func(t *testing.T) {
		q, teardown := setup(t)
		defer teardown(t)
		s1, err := models.AddNewService(q, models.PostgreSQLServiceType, &models.AddDBMSServiceParams{
			ServiceName: "pgrs1",
			NodeID:      "N1",
			Cluster:     "cluster0",
			Address:     new("127.0.0.1"),
			Port:        pointer.ToUint16OrNil(5432),
		})
		require.NoError(t, err)

		s2, err := models.AddNewService(q, models.PostgreSQLServiceType, &models.AddDBMSServiceParams{
			ServiceName: "pgrs2",
			NodeID:      "N1",
			Cluster:     "cluster0",
			Address:     new("127.0.0.1"),
			Port:        pointer.ToUint16OrNil(5432),
		})
		require.NoError(t, err)
		_, err = models.AddNewService(q, models.PostgreSQLServiceType, &models.AddDBMSServiceParams{
			ServiceName: "pgrs3",
			NodeID:      "N1",
			Cluster:     "cluster1",
			Address:     new("127.0.0.1"),
			Port:        pointer.ToUint16OrNil(5432),
		})
		require.NoError(t, err)

		services, err := models.FindServices(q, models.ServiceFilters{
			ServiceType: new(models.PostgreSQLServiceType),
			Cluster:     "cluster0",
		})
		require.NoError(t, err)
		assert.NotNil(t, services)
		assert.ElementsMatch(t, []*models.Service{s1, s2}, services)
	})

	t.Run("Change standard labels", func(t *testing.T) {
		q, teardown := setup(t)
		defer teardown(t)
		s, err := models.AddNewService(q, models.ExternalServiceType, &models.AddDBMSServiceParams{
			ServiceName:   "mongors1",
			NodeID:        "N1",
			Cluster:       "cluster0",
			ExternalGroup: "ext",
			Address:       new("127.0.0.1"),
			Port:          pointer.ToUint16OrNil(27017),
		})
		require.NoError(t, err)

		err = models.ChangeStandardLabels(q, s.ServiceID, models.ServiceStandardLabelsParams{
			Cluster:        new("cluster"),
			Environment:    new("env"),
			ReplicationSet: new("rs"),
			ExternalGroup:  new("external"),
		})
		require.NoError(t, err)

		ns, err := models.FindServiceByID(q, s.ServiceID)
		require.NoError(t, err)

		assert.Equal(t, "cluster", ns.Cluster)
		assert.Equal(t, "env", ns.Environment)
		assert.Equal(t, "rs", ns.ReplicationSet)
		assert.Equal(t, "external", ns.ExternalGroup)
	})

	t.Run("Software versions record created when adding a service", func(t *testing.T) {
		q, teardown := setup(t)
		defer teardown(t)

		emptyVersionsCreatedByServiceType := map[models.ServiceType]bool{
			models.MySQLServiceType:      true,
			models.MongoDBServiceType:    true,
			models.PostgreSQLServiceType: false,
		}

		specs := []struct {
			serviceType models.ServiceType
			name        string
			port        uint16
		}{
			{models.MySQLServiceType, "mysql", 3306},
			{models.MongoDBServiceType, "mongo", 27017},
			{models.PostgreSQLServiceType, "postgres", 5432},
		}

		var services []*models.Service
		for _, spec := range specs {
			if !models.IsServiceTypeSupported(spec.serviceType) {
				continue
			}

			service, err := models.AddNewService(q, spec.serviceType, &models.AddDBMSServiceParams{
				ServiceName: spec.name,
				NodeID:      "N1",
				Address:     new("127.0.0.1"),
				Port:        pointer.ToUint16OrNil(spec.port),
			})
			require.NoError(t, err)
			services = append(services, service)
		}
		require.NotEmpty(t, services)

		for _, service := range services {
			swVersions, err := models.FindServiceSoftwareVersionsByServiceID(q, service.ServiceID)

			// Was "return", which exited the subtest on the first service and left the
			// remaining types unchecked. The loop only became meaningful once the service
			// list was filtered by the allowlist, so the fix belongs with that change.
			if emptyVersionsCreatedByServiceType[service.ServiceType] {
				require.NoError(t, err)
				assert.NotNil(t, swVersions)
				continue
			}

			require.ErrorIs(t, err, models.ErrNotFound)
			assert.Nil(t, swVersions)
		}
	})
}
