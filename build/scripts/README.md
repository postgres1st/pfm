# Air-gapped Postgres1st WatchTower bundle: build and test

The scripts that produce and validate the offline bundle a customer installs on a
RHEL/Rocky 9 host with no internet. Everything here is independent of the upstream
Jenkins/`pmm-submodules` path — it builds from *this* checkout.

> Setting up a fresh machine, or running the Go, ClickHouse and server-backed suites
> as well? See [docs/running-the-test-suites.md](../../docs/running-the-test-suites.md).
> This file covers the bundle pipeline only.

## Build

```bash
build/scripts/build-pfw-airgap [stage ...]     # no args = every stage, in order
```

Stages are resumable so a failure does not repeat the expensive ones
(`grafana-fe` is the long pole):

`grafana-fe` → `grafana-be` → `pmm-bin` → `exporters` → `s3` → `rpms` → `closure` → `bundle`

Output lands in `build/.pfw-airgap-<arch>/` (gitignored): a `pfw-repo/` yum repo, an
`INSTALL.md`, and a tarball plus `.sha256`.

Every input is pinned to an exact commit in **`pfw-airgap-vars`**. Do not replace those
with branch names — "main" moves, and the bundle stops being reproducible. That file also
holds the signing identity; the bundle stage re-signs *every* RPM, including third-party
ones, so a customer's `gpgcheck=1` works after a single `rpm --import`.

Network is required to *build*. The **result** is what installs without network.

## Test — three suites, and they do not overlap

```bash
build/scripts/test-pfw-airgap            # fresh install, offline
build/scripts/test-pfw-upgrade           # in-place upgrade
build/scripts/test-pfw-negative-control  # proves those assertions can fail
build/scripts/test-pfw-qan               # end-to-end Query Analytics
```

Each run prints its own tally, which is why none is quoted here: the counts in an
earlier draft of this file were stale within hours of writing it, because adding one
assertion changes three numbers.

**`test-pfw-airgap`** installs the bundle into a throwaway systemd container with no
route off-host and asserts what the customer is paying for: offline install under
`gpgcheck=1`, readyz, the PostgreSQL-only gate, branding, live metrics, that a tampered
package is refused, and that only nginx answers off-host.

The gate is asserted in two places, because it is enforced in two: services of an
unsupported type are rejected at registration, and advisor checks of an unsupported
family are filtered out of the API listings. The upstream check set is 49 MySQL and 34
MongoDB against 26 PostgreSQL, so without the second the server advertises far more
coverage than it has. Both defer to `models.IsServiceTypeSupported`, so `PFW_DB_TYPES`
moves them together -- and the negative control proves that by widening it and watching
the MySQL checks reappear.

**`test-pfw-upgrade`** exists because **a fresh install cannot observe an upgrade
defect**. All four HIGH defects fixed during this workstream — Grafana config reverted on
upgrade, dashboards frozen at first boot, static NEVRAs that made `dnf upgrade` skip our
packages silently, and a `clickhouse-server` transaction resetting `/etc/clickhouse-server`
ownership — were found by upgrading a running server by hand, and every one of them
shipped past a green from-scratch run. It synthesises a previous release from current
sources with the build stamp forced old, so NEVRAs genuinely sort below the shipped bundle
and `dnf` really has to upgrade.

**`test-pfw-negative-control`** is the reason to believe the other two. Run it after
changing any assertion:

```bash
build/scripts/test-pfw-negative-control [airgap|upgrade|both]   # default: both
```

It needs the containers the other two leave behind (`PFW_TEST_KEEP=1`, the default).

## The rule that matters

**A green suite is not evidence.** It becomes evidence once each assertion has been shown
to go RED when the thing it guards is actually broken. That is not theoretical here: the
first control ever run found an assertion passing for the wrong reason — `readyz` stays
200 with the ClickHouse ownership repair disabled, because ClickHouse limps rather than
dying. The assertion proved nothing and was replaced.

This is why every assertion is a `p_*` predicate and why the harness **sources** them from
the suites rather than reimplementing them. Controlling a copy of an assertion says
nothing about the assertion that runs.

Two classes of control, counted separately in the output because they are not equally
strong:

- **state break** — change the host so the guarded property is genuinely false. Proves the
  assertion is wired to reality *and* discriminates.
- **`[input]`** — call the same predicate with a known-bad argument. Proves it
  discriminates, but **not** that the suite aims it at the right target. Only for
  properties that cannot be broken safely (a read-only mount, the PostgreSQL major).
  Never let one stand in for a state break that was possible.

## Traps already paid for

- **Poll for state changes; never sleep-then-check** — including the *baseline*. Five
  broken controls during this work, every one from judging state that had not settled. A
  racing control reports a sound assertion as decorative, which is worse than no control.
- `readyz` returning 200 does **not** mean pfw-managed will accept a *write*.
- `pkill -f foo` matches its own `bash -c` command line. Use `[f]oo`.
- systemd-in-docker needs `--cgroupns=host`, or it exits 255 with no output.
- An air gap must be a `--internal` network, **not** a disconnected bridge — removing the
  interface is stricter than a real host and produces failures customers never see.
- `curl` exits 0 on an HTTP 500. Use `--fail-with-body` when a service's error body would
  otherwise read as success.
- Anything `rpmbuild` writes inside a container is root-owned; `chown` it back or later
  non-root runs fail with EACCES.
- The bundle stage *empties* `pfw-repo/` rather than replacing it, so a bind-mounted test
  container keeps working. Do not reintroduce `rm -rf` there.

### `check-release-claims`

Fails when the customer-facing documents and the shipped artifacts disagree -- the
release notes and the two bundled INSTALL guides, checked against the spec, `pfw-init.sh`
and the packaged env seeds. Needs no container, no root and no built bundle, so it runs
in CI (`.github/workflows/release-claims.yml`). Each assertion looks for a *false*
statement rather than requiring particular wording, so rewording a sentence honestly
does not fail the build.

### `test-pfw-init-secrets`

Host-history tests for `pfw-init.sh`'s bootstrap credential logic: which credential a
given host should end up with on a fresh install, an upgrade from before the secret
store existed, and a host whose secret store was lost. Sources the script as a library
with `SRV` redirected into a temp directory, so it needs no container and no root.

**`test-pfw-qan`** asks the one question the other suites never did: does a query run on a
monitored database show up in Query Analytics? They assert QAN's scaffolding -- the
service runs, readyz includes it, ClickHouse accepts a row, a service can be registered --
and every one of those can be green while the product collects nothing.

It starts a second container as the monitored host, on the same `--internal` network as
the server, and installs `pg_stat_monitor` there. That extension is **not** in our bundle
and is not meant to be: it belongs on the database being monitored, which is the
operator's to install. `pfw-admin add postgresql` defaults to
`--query-source=pgstatmonitor`, so this is also the only test that exercises the default
path end to end.

Run it after `test-pfw-airgap`, which leaves the server it needs. The marker table and
service name carry a timestamp on purpose -- with fixed names the report query matches
rows an earlier run left in ClickHouse and the test passes without collecting anything,
which is exactly what happened the first time it ran.

Negative-controlled by stopping `pfw-server-agent`: the QAN assertion goes red while the
fixture assertions stay green, so a failure points at collection rather than at the
harness. Verified green, red with the agent stopped, green again once it is back.
