# Contributing to Postgres1st WatchTower

Postgres1st WatchTower is a PostgreSQL-only monitoring server
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
| `managed/` | `pfw-managed` — the server API, inventory and service registration |
| `qan-api2/` | Query Analytics API, backed by ClickHouse |
| `agent/` | `pfw-agent` — runs on monitored hosts and supervises exporters |
| `admin/` | `pfw-admin` — the CLI |
| `ui/` | the React interface and its Grafana compatibility plugin |
| `dashboards/` | provisioned Grafana dashboards |
| `vmproxy/` | VictoriaMetrics proxy |
| `api-tests/` | API-level integration tests |
| `build/` | packaging, the air-gap bundle pipeline and its acceptance suites |
| `documentation/` | product documentation and release notes |

The Grafana interface is a separate fork: [postgres1st/grafana](https://github.com/postgres1st/grafana).

Exporters and other vendored components are built from pinned upstream sources. The
authoritative list — repositories and exact commits — is
[`build/scripts/pfw-airgap-vars`](build/scripts/pfw-airgap-vars); it is not duplicated
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
make run-managed     # pfw-managed
make run-agent       # pfw-agent
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
exporters present on the system — installing `pfw-agent` from a bundle is the simplest
way to get them.

## Tests

**Unit tests** — `make test` in any component directory. Most of these need live
databases; without them a clean tree still reports dozens of failing packages, which is
an absent environment rather than a regression.

**`agent/` tests** — `make env-up-db` from `agent/`, then `make test`. That brings up
MySQL, MongoDB (five variants: plain, TLS, no-auth, and two replica sets), PostgreSQL and
Valkey, published on the host because the tests dial `localhost` directly. With it up the
suite is fully green; without it eleven packages fail.

Use `env-up-db` rather than `env-up` unless you also need a server: `env-up` additionally
starts `perconalab/pmm-server:3-dev-latest`, which is amd64-only *and* upstream's image,
so our renamed paths are absent from it — the same reason the `managed.yml` unit-test job
is disabled.

**On arm64** (Apple silicon, Graviton) two images need care, and `agent/Makefile` handles
both: `MYSQL_IMAGE` defaults to `percona/percona-server:8.0`, because the compose default
`percona:5.7` publishes amd64 only; and `aleksi/test_db` is amd64-only with no
alternative, but it is data-only and Docker populates its named volumes at container
*create*, before anything executes — so it seeds `world.sql` correctly and then exits 255
with an exec-format error. **That exit code is expected.** Check the volumes, not the
container.

Always run these with `-p 1` (which `make test` does). In parallel, the collector suite's
`test_collector` database leaks into the profiler suite's count and looks like a defect.

**`qan-api2` tests** — `make -C qan-api2 test-env-up`, then `make -C qan-api2 test`. That
target starts ClickHouse and imports the fixtures, and both halves matter: the tests
`docker exec` into a container named `pmm-clickhouse-test` by that exact name, and they
query a `pmm_test` database that only the import step creates. With it, all four packages
pass. Do not hand-roll the container — the name and the fixture load are both contracts.
Tear down with `test-env-down`.

Because those tests shell out to `docker`, run them on a host that has the CLI, not
inside a plain toolchain container.

**`managed/` + `qan-api2` together: 48 packages, 0 failures.** That needs the product
installed, not just its dependencies, in a systemd container:

```bash
docker run -d --name pfw-test --privileged --cgroupns=host --network host \
  -e container=docker --tmpfs /run --tmpfs /run/lock -v /sys/fs/cgroup:/sys/fs/cgroup:rw \
  -v "$PWD:/src" -v /var/run/docker.sock:/var/run/docker.sock <rocky9+systemd+go image>
# then, inside: install the RPMs the build produces
dnf -y --nogpgcheck install pfw-managed pfw-agent      # advisors, pfw-agent.yaml
rpm -i --nodeps pfw-server pfw-grafana pfw-dashboards  # nginx confs, grafana, pmm-app
```

Why each piece: `--network host` so the tests' hardcoded `127.0.0.1` reaches the
services; `--privileged --cgroupns=host` or systemd exits 255 silently; the docker socket
because `qan-api2` shells out to the CLI. `supervisor` comes from **EPEL**, not Rocky
base. `make` must be present — the starlark test runs `make release-starlark`.

Services: PostgreSQL on 5432 with **trust** auth; ClickHouse via
`make -C qan-api2 test-env-up`; VictoriaMetrics started with
`-http.pathPrefix=/prometheus`, because that is the path the health probe uses;
`grafana-server` from **our** `pfw-grafana` — stock Grafana 404s on
`/api/auth/serviceaccount`, which is a fork endpoint; and `pfw-managed` itself on `:7773`
for the inventory metrics test, which needs `/srv/.postgres_password` present and the
`pmm-managed` role owning its database.

Two traps worth knowing. Running `pfw-managed` makes `managed/services/inventory` pass but
rewrites `/etc/supervisord.d`; and installing `pfw-server` creates
`/srv/.pfw-secrets/clickhouse.env`, which `managed/utils/dbsecret` asserts is *absent*. The
two groups want opposite states, so run them in separate passes if you need both green in
one command.

**`managed/` tests cannot share a PostgreSQL with the agent tests**:
they connect to `127.0.0.1:5432` as `postgres` with the password in
`/srv/.postgres_password` — absent on a dev host, so **trust auth** — while the agent
tests need `pmm-agent` with password auth on the same port. Run one suite or the other.
They also want ClickHouse on `9000`, Grafana on `3000`, VictoriaMetrics on `9090` and
vmalert on `8880`. They also expect a writable `/srv`, which a dev host does not
give you; running the suite in a throwaway toolchain container clears that:

```bash
docker run --rm --network host --tmpfs /srv:rw,mode=1777 \
  --user "$(id -u):$(id -g)" -e HOME=/tmp -v "$PWD:/src" -w /src \
  golang:1.26 go test -p 1 ./managed/...
```

`--network host` is what lets the tests' hardcoded `127.0.0.1` reach the published service
ports, and `--user` keeps build output from landing root-owned in your worktree. A residue
still fails: those tests exec `pfw-managed`, `supervisorctl` and `vmalert`, and read
`/srv/prometheus/rules/pmm`, so they want the product installed rather than just its
dependencies running. That is what the devcontainer is for.

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
build/scripts/test-pfw-airgap            # install the bundle in an air-gapped container
build/scripts/test-pfw-negative-control  # prove the suite's assertions can actually fail
build/scripts/test-pfw-upgrade           # upgrade path
```

Run the negative control whenever you touch the airgap suite. A green suite that cannot
fail proves nothing, and this has already caught a false pass.

Note that container suites cannot substitute for a native install: SELinux transitions and
file capabilities are both inert under `docker run --privileged`, and have hidden real
blockers that only appeared on a real RHEL host.

## Building the product

```bash
build/scripts/build-pfw-airgap             # every stage in order
build/scripts/build-pfw-airgap rpms bundle # resume from a named stage
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
