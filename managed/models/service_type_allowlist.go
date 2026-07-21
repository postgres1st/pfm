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
	"fmt"
	"os"
	"slices"
	"strings"
)

// SupportedServiceTypesEnvVar overrides the set of Service types this build accepts.
// Value is a comma-separated list of ServiceType values, e.g. "postgresql,proxysql".
// Intended for debugging; the shipped default needs no override.
const SupportedServiceTypesEnvVar = "PFM_DB_TYPES"

// defaultSupportedServiceTypes is what this distribution ships with.
//
// PostgreSQL is the product. The other two are kept because removing them would cut a
// PostgreSQL capability rather than a MySQL one: Patroni exposes its own /metrics endpoint
// and is scraped as an external service (there is no Patroni exporter binary), and HAProxy
// commonly fronts a Patroni cluster and is likewise scraped natively. Neither ships an
// exporter or a DSN builder, so both cost little beyond an inventory entry.
var defaultSupportedServiceTypes = []ServiceType{
	PostgreSQLServiceType,
	HAProxyServiceType,
	ExternalServiceType,
}

// allServiceTypes is every type the models package knows about, in the same order as
// service_model.go. A type absent here is rejected as unknown rather than silently
// allowed, so a service type added upstream stays disabled until listed deliberately.
var allServiceTypes = []ServiceType{
	MySQLServiceType,
	MongoDBServiceType,
	PostgreSQLServiceType,
	ProxySQLServiceType,
	HAProxyServiceType,
	ExternalServiceType,
	ValkeyServiceType,
}

// supportedServiceTypes is resolved once at process start. Resolving per-request would let
// the allowlist change between a validation check and the write that follows it.
var supportedServiceTypes = resolveSupportedServiceTypes()

func resolveSupportedServiceTypes() map[ServiceType]struct{} {
	raw, ok := os.LookupEnv(SupportedServiceTypesEnvVar)
	if !ok {
		return serviceTypeSet(defaultSupportedServiceTypes)
	}

	// The variable is set, so the operator meant something by it. Refuse to start rather
	// than fall back to the default, which would discard their intent without a signal.
	res, err := parseSupportedServiceTypes(raw)
	if err != nil {
		panic(fmt.Sprintf("invalid %s: %s", SupportedServiceTypesEnvVar, err))
	}
	return res
}

func parseSupportedServiceTypes(raw string) (map[ServiceType]struct{}, error) {
	res := make(map[ServiceType]struct{})
	for _, field := range strings.Split(raw, ",") {
		field = strings.TrimSpace(field)
		if field == "" {
			continue
		}
		serviceType := ServiceType(field)
		if !slices.Contains(allServiceTypes, serviceType) {
			return nil, fmt.Errorf("unknown service type %q", field)
		}
		res[serviceType] = struct{}{}
	}
	if len(res) == 0 {
		return nil, fmt.Errorf("no service types listed in %q", raw)
	}
	return res, nil
}

func serviceTypeSet(types []ServiceType) map[ServiceType]struct{} {
	res := make(map[ServiceType]struct{}, len(types))
	for _, t := range types {
		res[t] = struct{}{}
	}
	return res
}

// IsServiceTypeSupported reports whether Services of the given type may be created.
func IsServiceTypeSupported(serviceType ServiceType) bool {
	_, ok := supportedServiceTypes[serviceType]
	return ok
}

// IsAgentTypeSupported reports whether Agents of the given type may be created.
//
// Agent types not bound to a Service are always supported: they attach to a Node or to PMM
// itself and carry no database technology. A Service-bound type survives if at least one of
// the Service types it serves is permitted, which keeps the multi-database cloud exporters
// available for their PostgreSQL half.
func IsAgentTypeSupported(agentType AgentType) bool {
	serviceTypes, ok := serviceTypesByAgentType[agentType]
	if !ok {
		return true
	}

	for _, serviceType := range serviceTypes {
		if IsServiceTypeSupported(serviceType) {
			return true
		}
	}
	return false
}

// SupportedServiceTypes returns the permitted types in service_model.go declaration order,
// for the API to expose so that clients do not hardcode their own copy of the allowlist.
func SupportedServiceTypes() []ServiceType {
	res := make([]ServiceType, 0, len(supportedServiceTypes))
	for _, serviceType := range allServiceTypes {
		if _, ok := supportedServiceTypes[serviceType]; ok {
			res = append(res, serviceType)
		}
	}
	return res
}
