# PGF WatchTower Development Guide for AI Agents

## Maintaining This Document

This file is read by every AI agent at session start. **You are responsible for keeping it accurate.** After completing work, check whether any of these apply:

- Added, removed, or renamed a top-level directory or component
- Added or removed a per-component `AGENTS.md`
- Changed the tech stack (new dependency in `go.mod`, new tool, removed technology)
- Changed build targets in `Makefile` / `Makefile.include`
- Changed global conventions (code style, error handling, testing patterns)
- Changed architecture or data-flow (new pipeline, changed communication protocol)
- Changed the development environment (`docker-compose.yml`, `.devcontainer/`)

If any apply, update the relevant sections of this file. Also update the matching per-component `AGENTS.md` if one exists for the affected area.

Do **not** update this file for routine code changes (bug fixes, minor feature implementation) that don't alter the repo's structure or conventions.

## How This Documentation Is Organized

This file is the **single authoritative entry point** for AI agents working with PGF WatchTower. It provides the product-wide overview, architecture, domain model, conventions, and cross-links to component-specific guides.

### Component Guides

Each component has a dedicated guide with architecture, directory structure, domain model, patterns, testing, and key files. When working on a specific component, read the relevant guide:

| Component | Guide | Scope |
|-----------|-------|-------|
| **pmm-managed** (server backend) | [managed/AGENTS.md](managed/AGENTS.md) | `managed/**` |
| **pmm-agent** (client agent) | [agent/AGENTS.md](agent/AGENTS.md) | `agent/**` |
| **pmm-admin** (CLI) | [admin/AGENTS.md](admin/AGENTS.md) | `admin/**` |
| **APIs** (protobuf definitions) | [api/AGENTS.md](api/AGENTS.md) | `api/**` |
| **qan-api2** (query analytics) | [qan-api2/AGENTS.md](qan-api2/AGENTS.md) | `qan-api2/**` |
| **vmproxy** (VictoriaMetrics proxy) | [vmproxy/AGENTS.md](vmproxy/AGENTS.md) | `vmproxy/**` |
| **UI** (React frontend) | [ui/AGENTS.md](ui/AGENTS.md) | `ui/**` |
| **Dashboards** (Grafana dashboard definitions) | [dashboards/dashboards/AGENTS.md](dashboards/dashboards/AGENTS.md) | `dashboards/dashboards/**` |
| **QAN App** (Grafana plugin & QAN panel) | [dashboards/pmm-app/AGENTS.md](dashboards/pmm-app/AGENTS.md) | `dashboards/pmm-app/**` |
| **API Tests** (integration tests) | [api-tests/AGENTS.md](api-tests/AGENTS.md) | `api-tests/**` |
| **Build & Packaging** | [build/AGENTS.md](build/AGENTS.md) | `build/**` |

---

## Naming: pmm-* vs pfw-* (read before "fixing" a name)

This fork ships as **PGF WatchTower**, and the rename is **deliberately partial**. The
authority for what may and may not be renamed is
[`docs/pfw-identifier-map.md`](docs/pfw-identifier-map.md) — read its tier table before
changing any name.

The short version: *Go identifiers and module paths* kept the upstream `pmm` names, and
should stay that way. But anything a **user reads or types** now says `pfw-*` — the installed
binaries are `pfw-admin` and `pfw-agent`, so user-facing text naming `pmm-admin` or
`pmm-agent` is a bug, not a deliberate remnant. It names commands that do not exist.

| Thing | State | Example |
|---|---|---|
| Go module path | **unchanged** | `github.com/percona/pmm` |
| Source directories | **unchanged** | `managed/`, `agent/`, `admin/` |
| `cmd/` dirs and binaries | **renamed** | `managed/cmd/pfw-managed`, `pfw-admin` |
| systemd units | **renamed** | `pfw-managed.service`, `pfw.target` |
| Server data directory | **moved** | `/opt/postgres1st/watchtower` |
| UI base path | **changed** | `/pfw-ui` |
| `PMM_*` environment variables | **unchanged** | `PMM_CLICKHOUSE_ADDR` |
| Metric namespace (`pmm_*`) | **unchanged** | `pmm_agent_id` |
| PostgreSQL database + role | **unchanged** | `pmm-managed` (not `pmm`) |
| ClickHouse database | **unchanged** | `pmm` |
| OS service accounts | **unchanged** | `pfw` (server), `pfw-agent` (client) |

The unchanged items are the contract shared with upstream code and with already-deployed
agents. Renaming them means forking behaviour we still want to rebase onto `percona/pmm`,
so they stay as they are on purpose.

**Practical rule:** a *command example* or a *path* using `pmm-` is a bug -- those moved.
A *component name* in prose is fine. Verify before changing either: `ls managed/cmd`
settles it in one command.

Note also that this build is **PostgreSQL-only**. `pfw-admin add mysql` (or mongodb,
proxysql, valkey) is rejected with "not supported by this deployment", so examples using
those service types are wrong regardless of the binary name.

## Product Overview

PGF WatchTower is an open-source **PostgreSQL** monitoring
solution, forked from Percona Monitoring and Management. It uses a **client-server
architecture** where lightweight agents on monitored hosts collect metrics and query
analytics data, sending them to a central server for storage, alerting, and visualization.

The accepted service types are `postgresql`, `haproxy` and `external` — see
`managed/models/service_type_allowlist.go`. HAProxy and external exist to serve PostgreSQL
deployments (HAProxy commonly fronts Patroni; Patroni's own `/metrics` is scraped as an
external service), not as products in their own right.

**Inherited code you will still find in the tree.** MySQL and MongoDB collectors
(`agent/agents/mysql`, `agent/agents/mongodb`), their QAN sources, and the backup
subsystem (`managed/services/backup`, `agent/runner/jobs/mysql_backup_job.go`,
`agent/client/pbm.go`) are all still present. They are unreachable in this build: the
allowlist rejects the service types that would drive them. Do not treat their existence as
evidence that a feature is supported — and do not delete them casually either, because
keeping the diff against upstream small is what makes rebasing onto `percona/pmm`
tractable.

This is a **monorepository** containing every component, the APIs, documentation, and
build scripts. Every backend component is written in Go; the UI is TypeScript/React.

## Architecture and Data Flow

### Metrics Pipeline

```
Exporters (node, postgres, rds, azure -- the only ones this build ships)
  → VMAgent (scrapes exporters)
    → VictoriaMetrics (time-series storage on the server)
      → Grafana (visualization)
      → VMAlert → Alertmanager (alerting)
```

### Query Analytics (QAN) Pipeline

```
QAN Agents (built into pmm-agent: pg_stat_statements, pg_stat_monitor;
            perfschema, slowlog and the MongoDB profiler remain in code but are unreachable)
  → pmm-managed (gRPC receiver)
    → qan-api2 (gRPC collector)
      → ClickHouse (query analytics storage)
        → UI / Grafana (visualization)
```

### Agent Communication

```
pmm-agent ←→ pmm-managed (bidirectional gRPC stream)
  - Server sends: SetStateRequest, StartAction, StartJob, Ping
  - Agent sends: StateChanged, QanCollect, ActionResult, JobResult, Pong
```

### Backup Pipeline

```
pmm-managed (orchestrator)
  → pmm-agent jobs (PBM for MongoDB, mysqldump/xtrabackup for MySQL)
    → S3/MinIO/local storage
```

Inherited and intact, but unreachable: every job it can schedule targets a service type
the allowlist rejects. PostgreSQL backup is not implemented upstream, so it is not
available here either.

## Domain Model

The core inventory model is **Node → Service → Agent**:

- **Node**: a physical or virtual host (generic, container, remote, RDS, Azure)
- **Service**: a database or application running on a node (MySQL, MongoDB, PostgreSQL, ProxySQL, HAProxy, Valkey, external)
- **Agent**: a monitoring agent associated with a node or service (pmm-agent, exporters, QAN agents, VMAgent)

Relationships:
- A Node has many Services
- A Service belongs to one Node
- An Agent runs on a Node (`runs_on_node_id`) and optionally monitors a Service (`service_id`)
- A child Agent belongs to a parent PMM Agent (`pmm_agent_id`)

## Repository Map

### Core Components

| Directory | Component | Purpose | Guide |
|-----------|-----------|---------|-------|
| `/managed` | pmm-managed | Server backend: inventory, APIs, VictoriaMetrics, Grafana, backup, alerting, HA | [managed/AGENTS.md](managed/AGENTS.md) |
| `/agent` | pmm-agent | Client agent: exporters, QAN/RTA collectors, actions, backup/restore jobs | [agent/AGENTS.md](agent/AGENTS.md) |
| `/admin` | pmm-admin | CLI for managing monitored services | [admin/AGENTS.md](admin/AGENTS.md) |
| `/api` | APIs | Protobuf definitions and generated gRPC/REST/Swagger clients | [api/AGENTS.md](api/AGENTS.md) |
| `/qan-api2` | qan-api2 | Query Analytics API: ClickHouse ingestion and analytics | [qan-api2/AGENTS.md](qan-api2/AGENTS.md) |
| `/vmproxy` | vmproxy | VictoriaMetrics reverse proxy with LBAC filtering | [vmproxy/AGENTS.md](vmproxy/AGENTS.md) |
| `/ui` | UI | React/TypeScript frontend (Vite, MUI, TanStack Query) | [ui/AGENTS.md](ui/AGENTS.md) |
| `/dashboards/dashboards` | Grafana Dashboards | Grafana dashboard JSON definitions for MySQL, MongoDB, PostgreSQL, OS, and more | [dashboards/dashboards/AGENTS.md](dashboards/dashboards/AGENTS.md) |
| `/dashboards/pmm-app` | QAN App | Grafana application plugin bundling dashboards and the Query Analytics panel | [dashboards/pmm-app/AGENTS.md](dashboards/pmm-app/AGENTS.md) |
| `/api-tests` | API Tests | Integration tests against a live server | [api-tests/AGENTS.md](api-tests/AGENTS.md) |
| `/build` | Build & Packaging | Docker, RPM/DEB, Packer, Ansible | [build/AGENTS.md](build/AGENTS.md) |

### Supporting Directories

| Directory | Purpose |
|-----------|---------|
| `/docs` | API documentation and process docs (tech stack, best practices, git workflow) |
| `/documentation` | User-facing documentation (MkDocs) |
| `/version` | Version info and feature flags |
| `/dev` | Development utilities (e.g., mongo-rs-backups) |
| `/.devcontainer` | Devcontainer setup for local development |

### External Repositories

| Repository | Purpose |
|------------|---------|
| [postgres1st/grafana](https://github.com/postgres1st/grafana) | Our Grafana fork — the interface |
| `percona/node_exporter` | Machine-level metrics exporter |
| `percona/postgres_exporter` | PostgreSQL metrics exporter |
| `percona/rds_exporter` | AWS RDS metrics exporter |
| `percona/azure_metrics_exporter` | Azure database metrics exporter |
| `percona/percona-toolkit` | Command-line tools bundled with the client |
| `hashicorp/nomad` | Orchestrates client components on monitored nodes |

Those four exporters are the only ones this build ships; the mysqld, mongodb and proxysql
exporters are not built. The pinned repository URLs and exact commits live in
[`build/scripts/pfw-airgap-vars`](build/scripts/pfw-airgap-vars), which is authoritative
— this table is orientation, not a source of truth.

Two upstream repositories are deliberately **not** listed as usable here:

- **`percona/pmm-qa`** — upstream's end-to-end suite. It targets MySQL and MongoDB
  scenarios and does not run against this build. Our acceptance suites are
  `build/scripts/test-pfw-*`.
- **`Percona-Lab/pmm-submodules`** — upstream's feature-build orchestration. Building
  client packages through it yields *upstream* `pmm-agent` and `pmm-admin` binaries with
  neither the service-type gate nor our packaging paths. The result looks correct and is
  not. Build with `build/scripts/build-pfw-airgap`.

## Tech Stack

| Technology | Role |
|------------|------|
| **Go** | All backend components |
| **TypeScript / React** | the UI (`/ui`) |
| **Protobuf v3 / gRPC** | API definitions and inter-component communication |
| **grpc-gateway** | HTTP/JSON REST API generated from gRPC definitions |
| **PostgreSQL** | Primary data store for pmm-managed (inventory, settings, backups) |
| **ClickHouse** | Query analytics data store (qan-api2) |
| **VictoriaMetrics** | Time-series metrics storage |
| **VMAlert** | Alerting rules evaluation |
| **Grafana** | Dashboards and visualization |
| **reform** | Go ORM for PostgreSQL (used in pmm-managed only — NOT gorm) |
| **logrus** | Structured logging |
| **testify** | Test assertions (`assert`, `require` packages only — NOT suites) |
| **mockery** | Mock generation for Go interfaces |
| **golangci-lint** | Static analysis and linting |
| **Kong** | CLI framework for pmm-admin |
| **Docker Compose** | Development environment |
| **Ansible** | Server provisioning and configuration |
| **Packer** | Machine image builds (AMI) |

## Global Development Conventions

### Code Style
- Format with `gofumpt -s`; run `make format`
- Follow [Effective Go](https://golang.org/doc/effective_go.html) and [CodeReviewComments](https://github.com/golang/go/wiki/CodeReviewComments)
- Import grouping: stdlib, then external (`github.com/percona`, third-party), then internal (this repo)
- Use `any` instead of `interface{}`
- Use modern slice helpers (`slices.Contains`), range loops
- Don't use named return values
- Don't inline comments (`code // comment`); put comments on separate lines
- Don't add obvious/redundant comments; only comment non-obvious intent

### Error Handling
- Use `status.Error()` with proper gRPC codes for API errors
- Wrap errors with context: `fmt.Errorf("descriptive context: %w", err)`
- Return early on errors to avoid deep nesting
- Use `errors.Is()`, `errors.As()` or `errors.AsType()` for error inspection
- Use standard `errors` package, not `github.com/pkg/errors`
- Check `reform.ErrNoRows` for "not found" scenarios in pmm-managed

### Logging
- Use `logrus` with structured fields
- Pass `*logrus.Entry` (not `*logrus.Logger`) to maintain context
- Format: `s.l.WithField("key", value).Error("message")`
- Log to unbuffered stderr; let the process supervisor handle the rest

### Environment Variables
- `PMM_DEV_*` — development/test only, never for end users
- `PMM_TEST_*` — not part of GA functionality
- `PMM_*` — GA functionality
- Use sub-prefixes for component groups (e.g., `PMM_HA_*`)

### Testing
- Use `testify/assert` and `testify/require` (not testify suites)
- Mock generation via `mockery` (config in `.mockery.yaml`)
- Unit tests: `*_test.go` next to implementation
- Integration tests: `/api-tests/`, run against a live server
- Acceptance suites: `build/scripts/test-pfw-airgap`, `-negative-control`, `-upgrade`
  (upstream's `pmm-qa` E2E suite does not apply to this build)

### Code Generation
- Protobuf/gRPC: `make gen` from repo root
- reform ORM: `//go:generate go tool reform` (pmm-managed only)
- Mocks: `mockery` per `.mockery.yaml`
- **Never edit generated files** (`.pb.go`, `.pb.gw.go`, `*_reform.go`, `*.pb.validate.go`, swagger specs, `json/client/`)

### Graceful Shutdown
- Handle `SIGTERM` and `SIGINT` by canceling parent context
- Stop handling signals after first receipt so second signal terminates immediately
- Startup errors are fatal; runtime errors are handled, logged, and communicated

### Debug Endpoints
All long-running daemons expose on `127.0.0.1`:
- `/debug/metrics` — Prometheus metrics
- `/debug/vars` — expvar (command line, memory stats)
- `/debug/requests`, `/debug/events` — trace facility
- `/debug/pprof` — profiling

## Key Make Targets

| Target | Purpose |
|--------|---------|
| `make env-up` | Start development container (the server) |
| `make env-up-rebuild` | Rebuild development container from scratch |
| `make run-ui` | Inside devcontainer: Vite HMR for the main UI |
| `make run-qan-ui` | Inside devcontainer: webpack + livereload for the QAN Grafana plugin |
| `make gen` | Generate all code (protobuf, reform, mocks, format) |
| `make check` | Run linters (buf, golangci-lint, go-sumtype) |
| `make format` | Format code (gofumpt, goimports, gci) |
| `make release` | Build all binaries (agent, admin, managed, qan-api2) |
| `make test-common` | Run common unit tests |
| `make api-test` | Run API integration tests |
| `make prepare-pr` | Full pre-PR pipeline: gen + check-all + go mod tidy |

## Key Files to Reference

- `Makefile`, `Makefile.include` — build and development targets
- `docker-compose.dev.yml` — development environment (server, renderer)
- `docker-compose.yml` — community/quickstart compose (stable image, minimal config)
- `go.mod` — Go module definition
- `.golangci.yml` — linter configuration
- `.mockery.yaml` — mock generation configuration
- `dev/docs/process/tech_stack.md` — technology choices and rationale
- `dev/docs/process/best_practices.md` — coding best practices
- `dev/docs/process/GIT_AND_GITHUB.md` — git workflow
