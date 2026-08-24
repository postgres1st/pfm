# Contributing to PFMM

Postgres1st Monitoring and Management (PFMM) is a PostgreSQL-only monitoring server
delivered as signed RPMs. See the [README](README.md) for what it is and how it installs.

## Table of contents

1. [Repository structure](#repository-structure)
2. [Prerequisites](#prerequisites)
3. [Setting up a development environment](#setting-up-a-development-environment)
4. [Tests](#tests)
5. [Building the product](#building-the-product)
6. [Submitting a pull request](#submitting-a-pull-request)

## Repository structure

Unlike upstream, everything is in this one repository:

| Path | What it is |
|---|---|
| `api/` | gRPC/protobuf definitions and the generated clients |
| `managed/` | `pfm-managed` — the server API, inventory and service registration |
| `qan-api2/` | Query Analytics API, backed by ClickHouse |
| `agent/` | `pfm-agent` — runs on monitored hosts and supervises exporters |
| `admin/` | `pfm-admin` — the CLI |
| `ui/` | the React interface and its Grafana compatibility plugin |
| `dashboards/` | provisioned Grafana dashboards |
| `vmproxy/` | VictoriaMetrics proxy |
| `api-tests/` | API-level integration tests |
| `build/` | packaging, the air-gap bundle pipeline and its acceptance suites |
| `documentation/` | product documentation and release notes |

The Grafana interface is a separate fork: [postgres1st/grafana](https://github.com/postgres1st/grafana).

Exporters and other vendored components are built from pinned upstream sources. The
authoritative list — repositories and exact commits — is
[`build/scripts/pfmm-airgap-vars`](build/scripts/pfmm-airgap-vars); it is not duplicated
here, because a copied list goes stale silently.

## Prerequisites

Read the [Code of Conduct](CODE_OF_CONDUCT.md) before contributing.

Everything else is picked up by the devcontainer, so a local Go, Node or protoc toolchain
is optional. To work outside the container you need Go, Node 22, Yarn and `make`.

## Setting up a development environment

### The devcontainer (recommended)

From the repository root on the host:

```bash
make env-up          # start the devcontainer; slow the first time, cached afterwards
make env             # shell into it
make env-down        # stop it
```

`make env-up-rebuild` refreshes the image when it has moved on.

Inside the container, rebuild and restart individual services — each replaces the running
binary and restarts it under supervisor:

```bash
make run-managed     # pfm-managed
make run-agent       # pfm-agent
make run-qan         # qan-api2
make run-vmproxy     # vmproxy
make run             # all of the above
```

These targets live in [`Makefile.devcontainer`](Makefile.devcontainer); `make help` inside
the container lists them.

### UI

```bash
make run-ui          # Vite dev server with hot reload for the main UI
make run-qan-ui      # webpack + livereload for the QAN Grafana plugin
```

See [`ui/README.md`](ui/README.md) for the full flow, including running Vite on the host
instead, and [`dashboards/CONTRIBUTING.md`](dashboards/CONTRIBUTING.md) for the QAN plugin
and dashboards.

### Agent

From `agent/`, `make setup-dev` builds the agent, registers it with a running server and
writes its configuration; `make run` then runs it. The agent needs vmagent and the
exporters present on the system — installing `pfm-client` from a bundle is the simplest
way to get them.

## Tests

**Unit tests** — `make test` in any component directory.

**Linters** — `make check-all` runs the linters and the licence-header check. Run it
before opening a pull request.

**API tests** — in `api-tests/`, against a running server:

```bash
PMM_SERVER_URL=https://admin:admin@127.0.0.1 make test
```

The variable keeps its `PMM_` name because the Makefile guards on it.

**Acceptance suites** — these exercise the shipped artifact rather than the source tree,
and are the ones that matter before a release:

```bash
build/scripts/test-pfmm-airgap            # install the bundle in an air-gapped container
build/scripts/test-pfmm-negative-control  # prove the suite's assertions can actually fail
build/scripts/test-pfmm-upgrade           # upgrade path
```

Run the negative control whenever you touch the airgap suite. A green suite that cannot
fail proves nothing, and this has already caught a false pass.

Note that container suites cannot substitute for a native install: SELinux transitions and
file capabilities are both inert under `docker run --privileged`, and have hidden real
blockers that only appeared on a real RHEL host.

## Building the product

```bash
build/scripts/build-pfmm-airgap             # every stage in order
build/scripts/build-pfmm-airgap rpms bundle # resume from a named stage
```

This produces the signed, air-gapped tarball. It needs docker, network access, ≥16 GB RAM
and ≥30 GB of free disk; everything else runs in containers. See the script's header for
the stage list.

Do **not** build client packages from `Percona-Lab/pmm-submodules`. That path produces
upstream `pmm-agent` and `pmm-admin` binaries, which carry neither the PostgreSQL
service-type gate nor our packaging paths — the result looks correct and silently is not.

## Submitting a pull request

Worth reading first:

- [Working with Git and GitHub](dev/docs/process/GIT_AND_GITHUB.md)
- [Tech stack](dev/docs/process/tech_stack.md)
- [Best practices](dev/docs/process/best_practices.md)

Then:

- run `make check-all` and the relevant tests, and make sure they pass
- keep the change and its commit message self-explanatory — say why, not just what
- if you changed packaging, unit files or the bundle, run the acceptance suites too
