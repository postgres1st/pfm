# Postgres1st Monitoring and Management (PFMM)

PostgreSQL monitoring and query analytics, delivered as a **signed package repository**
you install on your own RHEL, Rocky Linux or AlmaLinux 9 server. No internet access is
required on that server, and no container runtime is involved.

> **Status: beta.** The current release is `3.9.0~beta1`. It is complete and tested for
> the workflow described in the installation guides; the known limitations are stated
> plainly in the [release notes](documentation/docs/release-notes/pfmm-3.9.0-beta1.md)
> rather than left to be discovered.

## What this is

PFMM is a fork of [Percona Monitoring and Management](https://github.com/percona/pmm)
(PMM) that does one thing instead of five. Where PMM monitors MySQL, MongoDB, PostgreSQL,
Valkey and Redis, PFMM accepts PostgreSQL and refuses the rest — not by hiding them in the
interface, but with an allowlist enforced in the API at service and agent registration
(`managed/models/service_type_allowlist.go`).

Three service types are accepted:

| Type | Why |
|---|---|
| `postgresql` | the product |
| `haproxy` | commonly fronts a Patroni cluster; scraped natively, no exporter binary |
| `external` | how Patroni's own `/metrics` endpoint is scraped |

Anything else is rejected as unsupported. A service type added upstream stays disabled
until it is listed deliberately, so the gate does not quietly widen on a rebase.

The other substantive difference is delivery. PMM ships primarily as a container image;
PFMM ships as RPMs in a GPG-signed yum repository, built for air-gapped installation on a
host you control.

## Installation

Two guides, one bundle. Both ship inside the release tarball:

- **[Server](build/packages/pfmm-airgap-INSTALL.md)** — the monitoring server, on its own
  host.
- **[Client](build/packages/pfmm-airgap-INSTALL-CLIENT.md)** — the agent, on each
  PostgreSQL host you want to monitor.

In outline: unpack the tarball, import the signing key, point yum at the unpacked
directory, and `dnf install pfm-server` (or `pfm-client`). Operating-system dependencies
come from your own repositories — PFMM ships only what no EL9 repository provides
(PostgreSQL and ClickHouse), and the bundle includes `fetch-os-dependencies.sh` for hosts
with no repository of their own.

Requirements are in the server guide; briefly, RHEL/Rocky/AlmaLinux 9 with systemd,
2 vCPU and 4 GB RAM to start, and metric storage under `/srv`.

## Building

```bash
build/scripts/build-pfmm-airgap            # every stage, in order
build/scripts/build-pfmm-airgap rpms bundle  # resume from a named stage
```

Produces `pfmm-server-el9-<arch>.tar.gz` and its `.sha256`. Everything except docker runs
in containers, so the host needs no rpmbuild, Go or Node toolchain — but it does need
**≥16 GB RAM** (the Grafana webpack build OOMs below ~12 GB), ≥30 GB free disk, and
network access. The *result* is what installs without a network.

Acceptance suites live alongside it:

```bash
build/scripts/test-pfmm-airgap           # installs the bundle in an air-gapped container
build/scripts/test-pfmm-negative-control # verifies the suite's assertions can actually fail
build/scripts/test-pfmm-upgrade          # upgrade path
```

The negative-control suite exists because a green run proves nothing on its own. Note that
container suites cannot substitute for a native install: SELinux transitions and file
capabilities are both inert under `docker run --privileged`, and have hidden real blockers.

## Components

Server side, packaged as `pfm-server` and its dependencies: `pfm-managed` (the API and
inventory), a [Grafana fork](https://github.com/postgres1st/grafana) for the interface,
VictoriaMetrics and `vmproxy` for metrics, ClickHouse and `qan-api2` for query analytics,
and the provisioned dashboards. Client side, `pfm-client` installs `pfm-agent` and
`pfm-admin`.

Some inherited components still carry upstream names in their spec files and Go module
paths. That rebrand is deliberate remaining work, not an oversight.

## Contributing

See [CONTRIBUTING.md](CONTRIBUTING.md). Issues and pull requests go to
[postgres1st/pfm](https://github.com/postgres1st/pfm).

## Licensing

PFMM is built on Percona Monitoring and Management and Grafana, and everything Postgres1st
builds is licensed under the **GNU Affero General Public License, version 3**.

| Component | Licence |
|---|---|
| Server | [GNU AGPLv3](LICENSE) |
| Client agent | [Apache 2.0](agent/LICENSE) |
| Documentation | [GNU AGPLv3](documentation/LICENSE) |

Bundled third-party components keep their own licences — Apache 2.0 for ClickHouse and
VictoriaMetrics, the PostgreSQL Licence for PostgreSQL. Each package declares its own;
`rpm -qi <package>` reports it.

## Upstream

PFMM is derived from [percona/pmm](https://github.com/percona/pmm) and keeps its version
number: PFMM 3.9.0 corresponds to PMM 3.9.0. Copyright in the inherited code remains with
Percona LLC and the other original authors, and the AGPL headers are preserved throughout.
