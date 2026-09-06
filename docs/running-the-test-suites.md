# Running the test suites on a fresh machine

Every suite in this repo, in the order that gets you from a clean checkout to a full
run, using Docker for anything that needs a toolchain or a service. Nothing here
installs a compiler, database or build tool on the host.

Written against aarch64 (Apple-silicon-class Linux). x86_64 should work but has not
been exercised; the only architecture-specific notes are called out below.

`build/scripts/README.md` covers the bundle pipeline itself — what each build stage
does and why. This document is about *running the tests*.

---

## Run order

The layers are not independent: two of the Go suites need packages that only exist
after the bundle is built and installed. Read this before following the sections in
page order, which is grouped by *kind* of suite rather than by sequence.

```
Layer 0   guards                      nothing            seconds
              |
Layer 1a  agent/…                     PostgreSQL         container you start
          qan-api2/…                  ClickHouse         container you start
              |
Layer 2   build the bundle            Docker + ~30 GB    25-40 min cold
              |
Layer 3   test-pfw-airgap             the bundle         leaves a server running
              |
Layer 1b  managed/… , api-tests/…     that server        <- needs Layers 2 and 3
Layer 3   client, upgrade, negative-control, qan

Aside     test-docs-site              Docker + network    ~2 min, NOT a release gate
```

So the actual sequence on a fresh machine is:

```bash
build/scripts/test-frozen-identifiers && build/scripts/check-release-claims   # Layer 0
#   ... start PostgreSQL and ClickHouse, run agent/… and qan-api2/…           # Layer 1a
build/scripts/build-pfw-airgap                                                # Layer 2
build/scripts/test-pfw-airgap                                                 # Layer 3
#   ... now run managed/… and api-tests/… against the server it left up       # Layer 1b
build/scripts/test-pfw-client
build/scripts/test-pfw-upgrade
docker rm -f pfw-airgap-test && build/scripts/test-pfw-airgap                 # fresh host
build/scripts/test-pfw-negative-control
build/scripts/test-pfw-qan
```

Two dependencies are easy to miss:

- **`managed/utils/validators` needs `pfw-vmalert` on `PATH`**, which only exists once
  the bundle is built. Extract it from `build/.pfw-airgap-<arch>/pfw-repo` or accept
  that one package failing.
- **`test-pfw-negative-control` needs a *freshly installed* host.** It mutates real
  state, so run `test-pfw-airgap` again immediately before it.

If you only want a quick signal, Layer 0 alone catches most mistakes and takes seconds.

`build/scripts/test-docs-site` sits outside this order on purpose — see *The
documentation site* below.

---

## What you need

| | |
|---|---|
| Docker | the only host dependency; no Go, rpm, protoc or Node required |
| Disk | **~40 GB free.** The bundle build alone wants ~30 GB |
| RAM | **16 GB.** Grafana's webpack runs a 10 GB heap and OOMs below ~12 GB |
| Ports | 5432, 6379, 3306, 18123, 19000 must be free, or see *Port already taken* |

Do **not** build from a second checkout of this repo. `build-pfw-airgap` writes to
`build/.pfw-airgap-<arch>` inside the working tree, which is where it belongs; a
duplicate clone drifts from the real tree and eats the disk the build needs.

Two scratch paths are used throughout. Keeping caches out of `$HOME` is deliberate —
nothing this document runs should leave toolchain state on the host:

```bash
export CACHE=/tmp/pfw-cache          # Go build + module cache
export SCRATCH=/tmp/pfw-scratch      # extracted fixtures, copied secrets
mkdir -p "$CACHE" "$SCRATCH"
```

A shorthand for "run Go in a container", used by every Go command below:

```bash
gorun() {   # gorun [extra docker args...] -- <shell command>
  local args=(); while [ "$1" != "--" ]; do args+=("$1"); shift; done; shift
  docker run --rm -v "$PWD":/src -w /src -v "$CACHE":/gocache \
    --user "$(id -u):$(id -g)" -e HOME=/tmp -e GOFLAGS=-mod=mod \
    -e GOCACHE=/gocache/build -e GOMODCACHE=/gocache/mod \
    "${args[@]}" golang:1.26 \
    sh -c "git config --global --add safe.directory '*'; $*"
}
```

---

## Layer 0 — source-tree guards (seconds, no environment)

Run these first. They need no container, no network and no services, and they catch
the class of mistake that otherwise surfaces an hour later inside an RPM build.

```bash
build/scripts/test-frozen-identifiers     # expect: 68 passed, 0 failed
build/scripts/check-release-claims        # expect:  8 passed, 0 failed
```

If either is red, stop and fix it before building anything.

`check-release-claims` guards the half of the rebrand that nothing compiles: the prose a
customer follows. A missed rename in a spec breaks a build and we find it; a missed
rename in an install guide ships an instruction that fails in the customer's terminal,
so they meet it instead of us. It asserts three negatives — no `systemctl` instruction
naming a unit the packaging does not ship, no `pmm-*` command or unit in a customer
instruction, and no bundled document linking to a Percona or PMM URL. All three derive
their truth from the artifacts, not from a list, because a stale allowlist inside a
rename check is indistinguishable from no check.

It is deliberately blind to the bare word "PMM", so the guides can go on saying this
product derives from upstream PMM. That attribution is true and worth keeping, and a
blanket brand grep would force someone to delete it to get a green run.

---

## Layer 1 — Go suites

No single container can host all of these: `agent` wants a PostgreSQL on
`localhost:5432` with its own credentials, while `managed` wants a fully installed
server. Run them in the phases below.

### Compile and vet everything

```bash
gorun -- 'go build ./... && go vet ./...'
```

### Layer 1a: agent/… — needs PostgreSQL

MySQL, MongoDB and Valkey tests **skip** when their service is absent, and each skip
prints an explicit `--- SKIP:` line naming the service. PostgreSQL deliberately does
**not** skip: it is the database this product monitors, so a missing one is a broken
environment, not a suite to wave through.

The tests need the `world` sample database. Its fixture image is amd64-only, so
extract it rather than run it — `docker create` never executes the image:

```bash
docker create --name pfw-seed aleksi/test_db:1.1.0 >/dev/null
mkdir -p "$SCRATCH/test_db" && docker cp pfw-seed:/test_db/. "$SCRATCH/test_db/"
docker rm pfw-seed >/dev/null
```

Start PostgreSQL. Percona's distribution is required for the `pgstatmonitor` package
(`pg_stat_monitor` is not in stock PostgreSQL). Note it is **not** published on a host
port — see *Port already taken*:

```bash
docker run -d --name pfw-test-pg \
  -e POSTGRES_USER=pmm-agent -e POSTGRES_PASSWORD=pmm-agent-password \
  -v "$SCRATCH/test_db/postgresql/world":/docker-entrypoint-initdb.d:ro \
  percona/percona-distribution-postgresql:18 \
  -c shared_preload_libraries='pg_stat_statements,pg_stat_monitor' \
  -c track_activity_query_size=2048 -c pg_stat_statements.max=10000 \
  -c pg_stat_monitor.pgsm_query_max_len=10000 -c pg_stat_statements.track=all \
  -c pg_stat_statements.save=off -c track_io_timing=on
sleep 25
```

The official image trusts loopback, which makes "wrong password" tests pass when they
should fail. Require a password, then reload:

The path differs by image, so ask the server rather than hardcoding it -- Percona's
PG18 uses /data/db, the official image /var/lib/postgresql/data. A wrong path makes
`sed` fail silently, loopback stays trusted, and the wrong-password tests then fail in
a way that looks like a code defect:

```bash
HBA=$(docker exec pfw-test-pg psql -U pmm-agent -d pmm-agent -tAc 'SHOW hba_file' | tr -d '\r')
docker exec pfw-test-pg sh -c \
  "sed -i 's|127.0.0.1/32 *trust|127.0.0.1/32 scram-sha-256|; s|::1/128 *trust|::1/128 scram-sha-256|' $HBA"
docker exec pfw-test-pg psql -U pmm-agent -d pmm-agent -c 'SELECT pg_reload_conf()'
docker exec pfw-test-pg sh -c "grep -E '^host +all' $HBA"   # verify: no 'trust' on loopback
```

Some agent tests exec binaries by name, so build them onto a `PATH` the container sees:

```bash
mkdir -p "$SCRATCH/bin"
gorun -v "$SCRATCH/bin":/out -- 'for t in \
    "pfw-admin ./admin/cmd/pfw-admin" "pfw-agent ./agent" \
    "pfw-managed ./managed/cmd/pfw-managed" \
    "pfw-managed-init ./managed/cmd/pfw-managed-init" \
    "pfw-managed-starlark ./managed/cmd/pfw-managed-starlark" \
    "pfw-qan-api2 ./qan-api2"; do
      set -- $t; go build -o /out/$1 $2 || echo "FAILED $1"; done'
```

Two more binaries are needed by `managed/` and exist only after Layer 2 installs the
bundle. Copy them out of the running server rather than building them:

```bash
docker cp pfw-airgap-test:/usr/sbin/pfw-vmalert         "$SCRATCH/bin/"
docker cp pfw-airgap-test:/usr/sbin/pfw-victoriametrics "$SCRATCH/bin/"
```

`managed/utils/validators` execs `pfw-vmalert`, and `managed/services/victoriametrics`
execs `pfw-victoriametrics`; without them those packages fail with "executable file not
found", which reads like a defect and is not one.

Run the suite by joining the database's network namespace, so `localhost:5432` resolves
without publishing a port:

```bash
docker run --rm --network container:pfw-test-pg \
  -v "$PWD":/src -w /src -v "$CACHE":/gocache -v "$SCRATCH/bin":/testbin \
  --user "$(id -u):$(id -g)" -e HOME=/tmp -e GOFLAGS=-mod=mod \
  -e GOCACHE=/gocache/build -e GOMODCACHE=/gocache/mod \
  -e PATH=/testbin:/usr/local/go/bin:/usr/local/bin:/usr/bin:/bin \
  golang:1.26 sh -c "git config --global --add safe.directory '*'; go test -p 1 -count=1 ./agent/..."
```

**Expect 43 ok / 0 fail.** `-p 1` is not optional: in parallel the collector suite's
`test_collector` database leaks into the profiler suite's count and reads as a defect.

### Layer 1a: qan-api2/… — needs ClickHouse

The tests shell out to `docker exec` against a container named **exactly**
`pmm-clickhouse-test`, so the container running them needs the Docker socket.

```bash
make -C qan-api2 start-clickhouse
sleep 15
docker exec pmm-clickhouse-test clickhouse client --password=clickhouse \
  --query="CREATE DATABASE IF NOT EXISTS pmm_test"
gorun -w /src/qan-api2 -- 'go run ./cmd/render-migrations' > "$SCRATCH/ch.sql"
docker exec -i pmm-clickhouse-test clickhouse client --password=clickhouse \
  -d pmm_test --multiline --multiquery < "$SCRATCH/ch.sql"
cat qan-api2/fixture/metrics.part_*.json | docker exec -i pmm-clickhouse-test \
  clickhouse client --password=clickhouse -d pmm_test \
  --query="INSERT INTO metrics FORMAT JSONEachRow"

docker run --rm --network host -v "$PWD":/src -w /src -v "$CACHE":/gocache \
  -v /var/run/docker.sock:/var/run/docker.sock --user 0:0 -e HOME=/tmp \
  -e GOFLAGS=-mod=mod -e GOCACHE=/gocache/build -e GOMODCACHE=/gocache/mod \
  golang:1.26 sh -c 'git config --global --add safe.directory "*";
    command -v docker >/dev/null || { apt-get update -qq && apt-get install -y -qq docker.io; } >/dev/null 2>&1
    go test -p 1 -count=1 ./qan-api2/...'
```

`make test-env-up` does all of this in one target, but its import step races the
container's startup and fails on a cold machine. The steps above are the same thing
with the wait made explicit.

**Expect 4 ok / 0 fail.**

### Layer 1b: managed/… and api-tests/… — need an installed server (do Layers 2 and 3 first)

There are two ways to point these at a server, and they are not equivalent.

**Inside the host (44 ok / 0 fail).** Build a systemd image that also carries Go,
install onto it, and run the tests *in* it. The tests then see the real systemd, the real
`/srv`, and provisioned Grafana — not just the server's network.

```bash
# one-off: a systemd image with the Go toolchain and make
docker create --name pfw-go-src golang:1.26 >/dev/null
mkdir -p "$SCRATCH/goroot" && docker cp pfw-go-src:/usr/local/go "$SCRATCH/goroot/"
docker rm pfw-go-src >/dev/null
cat > "$SCRATCH/goroot/Dockerfile" <<'DOCKER'
FROM pfw-rocky9-systemd:latest
RUN dnf -y install make && dnf clean all
COPY go /usr/local/go
ENV PATH=/usr/local/go/bin:/usr/local/sbin:/usr/local/bin:/usr/sbin:/usr/bin:/sbin:/bin
DOCKER
docker build -t pfw-rocky9-systemd-go:latest "$SCRATCH/goroot"

# install onto it — PFW_SYSTEMD_IMAGE is honoured by the suite
docker rm -f pfw-airgap-test
PFW_SYSTEMD_IMAGE=pfw-rocky9-systemd-go:latest KEEP=1 build/scripts/test-pfw-airgap

# source and module cache in; .git and the build workspace are not needed
tar --exclude='./.git' --exclude='./build/.pfw-airgap-*' --exclude='./.worktree' \
    --exclude='./ui/node_modules' -cf - . \
  | docker exec -i pfw-airgap-test sh -c 'mkdir -p /src && tar -xf - -C /src'
docker cp "$CACHE" pfw-airgap-test:/gocache

docker exec -e HOME=/tmp -e GOFLAGS=-mod=mod -e GOCACHE=/gocache/build \
  -e GOMODCACHE=/gocache/mod -e PMM_RELEASE_PATH=/usr/sbin -w /src pfw-airgap-test \
  /usr/local/go/bin/go test -p 1 -count=1 ./managed/...
```

`make` is in that image on purpose: `cmd/pfw-managed-starlark` shells out to it, and the
stock Rocky image has no compiler toolchain.

One test gets *worse* inside, and it is the test's fault rather than the environment's:
`managed/utils/dbsecret` asserts that ClickhousePassword() falls back to the shipped
constant, which only holds when `/srv/.pfw-secrets/clickhouse.env` does not exist. On a
provisioned host it does exist, so the fallback is not taken. The test asserts a default
without controlling for the real file.

**Alongside the host (simpler: 35 ok / 9 fail).** Join the server's network namespace
only. Quicker to set up and enough for most work, but every test needing a process
manager or a file under `/srv` fails. Use this when iterating; use the above before
believing a count.


These want ClickHouse, Grafana, VictoriaMetrics, PostgreSQL and `pfw-managed` all
running and talking to each other. Assembling that from separate images is a losing
game; **install our own bundle instead** and point the tests at it. That means doing
Layer 2 and the airgap suite first — come back here afterwards.

Once `pfw-airgap-test` is running (the airgap suite leaves it up):

**Warm the module cache first.** The airgap container is deliberately isolated —
"no route to the internet" is one of the suite's own assertions — so a test container
joining its network namespace cannot reach proxy.golang.org. On a warm cache this is
invisible; on a fresh machine every package dies with `[setup failed]` while downloading
a test-only dependency:

```bash
docker run --rm -v "$PWD":/src -w /src -v "$CACHE":/gocache \
  --user "$(id -u):$(id -g)" -e HOME=/tmp -e GOFLAGS=-mod=mod \
  -e GOCACHE=/gocache/build -e GOMODCACHE=/gocache/mod golang:1.26 \
  sh -c 'git config --global --add safe.directory "*"; go mod download all'
```

```bash
# /srv is plain container filesystem, not a volume, so --volumes-from will NOT
# bring this across. Without it every managed test fails on "password
# authentication failed for user postgres".
mkdir -p "$SCRATCH/srv"
docker cp pfw-airgap-test:/srv/.postgres_password "$SCRATCH/srv/.postgres_password"
chmod 644 "$SCRATCH/srv/.postgres_password"

docker run --rm --network container:pfw-airgap-test \
  -v "$PWD":/src -w /src -v "$CACHE":/gocache -v "$SCRATCH/bin":/testbin \
  -v "$SCRATCH/srv/.postgres_password":/srv/.postgres_password:ro \
  --user 0:0 -e HOME=/tmp -e GOFLAGS=-mod=mod \
  -e GOCACHE=/gocache/build -e GOMODCACHE=/gocache/mod -e PMM_RELEASE_PATH=/testbin \
  -e PATH=/testbin:/usr/local/go/bin:/usr/local/bin:/usr/bin:/bin \
  golang:1.26 sh -c "git config --global --add safe.directory '*'; go test -p 1 -count=1 ./managed/..."
```

**Expect 44 ok / 0 fail.** Anything else is a finding.

This was 33 ok / 11 fail for a long time, recorded as "the environmental ceiling". It was
not a ceiling. All eleven were tests asserting a product that no longer exists —
credentials from before they were generated per install, the container image's file
layout, a live network from a deliberately airgapped host. Fixing them turned up three
real product bugs the tests had been tolerating: encryption rotation could not run
natively, the diagnostic archive shipped with no service logs at all, and pfw-managed
could not validate its VictoriaMetrics config after a binary rename.

If a package fails, the first question is which product it is asserting, not what the
environment is missing.

Two operational notes:

- This step runs as `--user 0:0`, so it writes root-owned files into the shared Go cache
  and into `managed/services/grafana/fuzzdata`. A later non-root step then fails with
  "permission denied". Reclaim them from a container, never with sudo:
  `docker run --rm -v "$PWD":/src -v "$CACHE":/gocache golang:1.26 chown -R $(id -u):$(id -g) /src /gocache`
- Clear Grafana's login lockout before running, or authentication fails on a correct
  password (see *Gotchas*).

### The supervisord backend needs its own container

`TestDevContainer` skips when supervisorctl is absent, which is correct here — but the
supervisord backend is live code, so skipping it everywhere would leave half of
`selectProcessManager` untested. It runs on a container that has supervisorctl instead:

```bash
build/scripts/test-supervisord-backend
```

The script runs the test with `-v` and **treats a skip as a failure**, because `ok` alone
does not distinguish the two: a skipped test and a passed test both end in `ok`, and the
whole reason this script exists is that the test must not skip. If supervisorctl went
missing from the image, or the socket never came up, the test would quietly opt out and a
naive script would report success while covering nothing. Expect:

```
=== RUN   TestDevContainer
=== RUN   TestDevContainer/UpdateConfiguration
--- PASS: TestDevContainer
```

If it skips instead, the script says so and exits 1. Do not "fix" that by editing the
script with `sed` to change the test command — it lives inside an `sh -c '...'` string, so
anything you add there must contain no single quotes, and mangling the quoting yields
`[setup failed]`, which reads like a cold-cache failure and is not one.

That distinction matters. The MySQL and MongoDB skips cover paths the postgres-only gate
refuses in production, so not running them costs nothing. This covers code a customer
can run, so it is skipped *here* and exercised *there*.

**Known gap.** This suite covers `managed/services/supervisord`. It does *not* cover
`TestDownloadLogs`'s supervisord branch, which needs a live supervisord-*backed server* —
the container image, which beta2 does not build. That branch's expectations are derived
from reading `processManagerConfigFiles`, not from a run. It is deliberately a subset
assertion rather than an exact one for that reason, but treat it as unexercised.

For `api-tests`, pass the server URL with the generated admin password:

```bash
PW=$(docker exec pfw-airgap-test sh -c \
  "sed -n 's/^GF_SECURITY_ADMIN_PASSWORD=//p' /srv/.pfw-secrets/grafana.env" | tr -d '\r\n')
docker run --rm --network container:pfw-airgap-test \
  -v "$PWD":/src -w /src -v "$CACHE":/gocache --user 0:0 -e HOME=/tmp \
  -e GOFLAGS=-mod=mod -e GOCACHE=/gocache/build -e GOMODCACHE=/gocache/mod \
  -e PMM_SERVER_URL="https://admin:${PW}@127.0.0.1:8443/" -e PMM_SERVER_INSECURE_TLS=1 \
  golang:1.26 sh -c "git config --global --add safe.directory '*'; go test -p 1 -count=1 ./api-tests/..."
```

**Expect 9 ok / 0 fail.**

Getting there meant deciding, for each inherited failure, whether it was our bug or an
upstream assumption we deliberately broke. Three kinds turned up, and they are worth
telling apart because only one of them is allowed to grow:

*Tests of a capability the postgres-only gate removes.* Roughly 18 subtests add MySQL,
MongoDB or ProxySQL services and the gate refuses them. These call
`pmmapitests.SkipIfServiceTypeUnsupported(t, "mysql")` and report `not supported`, which
is the gate working. A **drop** in the skip count is the finding here, not the skips.

*Tests that asserted telemetry or auto-update could be switched on.* These are not
skipped. The product forces both off in `pfw-managed.service`, and the server answers a
request to enable either with `FailedPrecondition`, so the tests now assert **the
refusal** via `serverTest.AssertEnvOwnedChangeIsRefused`. The distinction matters for a
privacy guarantee: a skip would record that we could not check, whereas this records that
we checked and telemetry is unreachable. If it ever fails — including on a deployment
that forgot to set the variable — that is a real finding about that deployment. The
helper also asserts the refusal is *atomic*, because a body that asks for telemetry and
advisors together must change neither.

`build/scripts/test-frozen-identifiers` holds the line: a test may write
`EnableTelemetry: new(true)` only inside an `AssertEnvOwnedChangeIsRefused` call, and the
guard additionally pins the helper itself, since a gutted helper would silently re-permit
every site leaning on it.

*Tests that hardcoded the container image's layout.* `TestDownloadLogs` asserted an exact
list of diagnostic-archive entries taken from a supervisord host. On systemd the archive
legitimately differs — journal entries and unit files instead of `/srv/logs/*.log` and
`/etc/supervisord.d/*.ini` — so the test failed while the archive was correct. It now
reads the backend off the archive itself (not off the machine running the test, which may
not be the server) and asserts accordingly: an exact set for systemd, where every entry is
code-derived from constants, and configuration-plus-non-empty for supervisord, where the
service logs come from a `filepath.Glob` and are a property of the deployment rather than
of the code.

---

## Layer 2 — build the bundle

```bash
PFW_GRAFANA_FORK=$HOME/Projects/postgres1st-grafana \
  build/scripts/build-pfw-airgap 2>&1 | tee /tmp/bundle-build.log
```

`PFW_GRAFANA_FORK` is optional. Leave it unset and the build clones
`postgres1st/grafana` at the pinned commit into the workspace; point it at an existing
clone to skip a large fetch and reuse `node_modules` between runs.

Takes roughly 25–40 minutes cold, most of it Grafana's frontend. Output:

```
build/.pfw-airgap-aarch64/pfw-server-el9-aarch64.tar.gz     ~486 MB
build/.pfw-airgap-aarch64/pfw-repo/                          15 RPMs, signed
```

To iterate after a code change, skip the Grafana stages:

```bash
build/scripts/build-pfw-airgap pmm-bin rpms closure bundle
```

---

## Layer 3 — the install suites

Run in this order. Each leaves its container up for inspection; `KEEP=0` removes it.

```bash
build/scripts/test-pfw-airgap             # expect: 46 passed, 0 failed
build/scripts/test-pfw-client             # expect:  9 passed, 0 failed
build/scripts/test-pfw-upgrade            # expect: 34 passed, 0 failed
build/scripts/test-pfw-negative-control   # expect: 58 passed, 0 failed
build/scripts/test-pfw-qan                # expect:  5 passed, 0 failed
```

`test-pfw-client` is the only suite that executes the documented CLIENT procedure. Every
other one runs `pfw-admin` on the server and passes `--server-url`, which bypasses the
agent config entirely — so nothing else ever reads `pfw-agent.yaml`, runs `pfw-admin
config`, installs `pfw-agent` on a host that is not the server, or registers a database
that is not `127.0.0.1`. It deliberately passes **no** `--server-url` to `add`, so the
credentials must come from the config that `pfw-admin config` just wrote.

Two traps if you extend it. `pfw-admin list` is **node-scoped** ("Show Services and Agents
running on this Node"), so asking the server about a client's service always comes back
empty against a product that is working; use the inventory API. And that API is
`GET /v1/inventory/services`, not a `:list` POST. Both mistakes were made the first time
this suite ran, and both looked like product failures.

`test-pfw-negative-control` reads as the most valuable of the four: it breaks each
condition the airgap suite asserts and confirms the assertion goes red. Read its
summary line, not just the pass count — **`0 not discriminating`** is the number that
matters. An assertion that stays green while its condition is broken is worse than a
missing test.

Run it **against a freshly installed host**. It mutates real state, and a previous
interrupted run leaves the host in a state that makes the next run fail for reasons
that have nothing to do with your change:

```bash
docker rm -f pfw-airgap-test
build/scripts/test-pfw-airgap && build/scripts/test-pfw-negative-control
```

`test-pfw-upgrade` builds a second, older-stamped bundle before it can test anything,
so it takes ~15 minutes even when everything is cached.

**Interrupting it is safe, but it was not always.** `rpmbuild` runs as root inside a
container, so a run that is killed mid-build — Ctrl-C, a closed terminal, a reboot —
leaves root-owned files under `build/.pfw-airgap-<arch>/work/upgrade`. The next run's
cleanup then dies in thousands of `Permission denied` lines: a failure that surfaces one
run *later* than its cause and looks nothing like it. The suite now reclaims ownership
from a container before cleaning, so this self-heals. If you meet the same shape
elsewhere, reclaim it the same way and never with `sudo`:

```bash
docker run --rm -v "$PWD/build/.pfw-airgap-$(uname -m)/work":/w golang:1.26 \
  chown -R "$(id -u):$(id -g)" /w
```

---

## The documentation site (not a release gate)

```bash
build/scripts/test-docs-site              # expect: 4 passed, 0 failed  (~2 min)
```

**A red result here must never block a bundle build.** The documentation site is not a
beta2 deliverable: `build-pfw-airgap` takes exactly one file from that tree, the current
release notes. The site still carries pre-rename names, Percona links and an information
architecture describing five delivery paths we do not ship — all deliberately deferred to
a rewrite before GA, and recorded in `docs/deferred-documentation-rebrand.md`. This suite
does not assert any of that, and would be permanently red if it did.

It asserts the two things that must not regress while the rewrite is pending: the site
builds without dead links or missing files, and it ships no trackers.

The tracker check is the reason the suite exists, and its scope is deliberate: it searches
**the whole built site**, not the sources and not just the rendered HTML. Both narrower
checks were tried on this tree and both reported clean while it was not:

- Grepping `*.md` and `*.yml` missed PostHog, an Osano CMP and a Google Form, because
  those were hand-written into the Jinja overrides rather than configured in `mkdocs.yml`.
- Grepping the rendered HTML then missed the *assets*. mkdocs copies everything under
  `docs/` into the build whether or not a page references it, so `js/rating.js` — carrying
  the live Google Form URL it POSTed reader feedback to — was still published and
  downloadable after the `<script>` tag was gone. **Unloaded is not unshipped.**

It also fails on any mkdocs `WARNING`. mkdocs reports a dead internal link or a missing
image as a warning and still exits 0, which is how a broken front-page link survived: it
pointed at `release-notes/{{release}}.md` while `variables.yml` still said `3.8.1`, a
Percona release whose notes are deliberately not reproduced here.

Negative-controlled in all three directions — a tracker script added to a template, an
**unreferenced** tracker asset added to `docs/js/`, and a dead internal link each turn
exactly one assertion red.

---

## Gotchas that cost real time

**Port already taken.** Another container may hold 5432. Rather than fight it, none of
the commands above publish host ports — they use `--network container:<name>`, which
puts the test container in the database's network namespace so `localhost:5432`
resolves. Check with `docker ps --format '{{.Names}} {{.Ports}}'` before assuming a
port is free, and never stop a container you did not start; it may belong to another
session.

**Loopback is trusted.** The side effect of the namespace trick is that connections
appear to come from 127.0.0.1, which the official PostgreSQL image trusts. Tests that
assert a *wrong* password is rejected then fail, because it was accepted. The `pg_hba`
edit above is what fixes it.

**amd64-only images on arm64.** `aleksi/test_db` cannot be executed here — extract it
with `docker create` + `docker cp`, never `docker run`. If you reach for a throwaway
`alpine` container, it will fail the same way; use `golang:1.26` or `rockylinux:9`,
which are multi-arch.

**Root-owned files after a failed build.** `rpmbuild` runs as root inside its
container. A suite that dies mid-build leaves thousands of root-owned files, and the
next run fails on `rm: cannot remove`. Reclaim them from a container, not with `sudo`:

```bash
docker run --rm -v "$PWD/build/.pfw-airgap-aarch64/work":/w golang:1.26 \
  chown -R "$(id -u):$(id -g)" /w
```

**Grafana locks the admin account.** Five failed logins lock it for a window, and
every later probe returns 401 on a password that is correct — including `api-tests`
run against the same server. `test-pfw-negative-control` deliberately authenticates
with a wrong password, so it clears `login_attempt` as part of its restore. If you
authenticate by hand and get an inexplicable 401:

```bash
docker exec pfw-airgap-test sh -c \
  "PGPASSWORD=\$(cat /srv/.postgres_password) psql -U postgres -h /run/postgresql \
   -d grafana -c 'DELETE FROM login_attempt'"
```

**Never `pkill -f` a script by name.** The pattern matches the `pkill` command's own
argument list and any shell waiting on that script, so it kills the monitoring you are
running alongside it. Use `docker rm -f <container>`, or match on a pid from
`ps -eo pid,args | grep '[t]est-pfw-...'` — the bracket keeps the grep from matching
itself.

---

## Cleanup

```bash
docker rm -f pfw-airgap-test pfw-upgrade-test pfw-test-pg pmm-clickhouse-test
rm -rf build/.pfw-airgap-aarch64/work "$SCRATCH"
```

Keep `build/.pfw-airgap-*/pfw-repo` and the tarball if you intend to re-run the
install suites; rebuilding them is the expensive part.
