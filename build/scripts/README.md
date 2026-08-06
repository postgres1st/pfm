# Air-gapped PFMM bundle: build and test

The scripts that produce and validate the offline bundle a customer installs on a
RHEL/Rocky 9 host with no internet. Everything here is independent of the upstream
Jenkins/`pmm-submodules` path — it builds from *this* checkout.

## Build

```bash
build/scripts/build-pfmm-airgap [stage ...]     # no args = every stage, in order
```

Stages are resumable so a failure does not repeat the expensive ones
(`grafana-fe` is the long pole):

`grafana-fe` → `grafana-be` → `pmm-bin` → `exporters` → `s3` → `rpms` → `closure` → `bundle`

Output lands in `build/.pfmm-airgap/` (gitignored): a `pfmm-repo/` yum repo, an
`INSTALL.md`, and a tarball plus `.sha256`.

Every input is pinned to an exact commit in **`pfmm-airgap-vars`**. Do not replace those
with branch names — "main" moves, and the bundle stops being reproducible. That file also
holds the signing identity; the bundle stage re-signs *every* RPM, including third-party
ones, so a customer's `gpgcheck=1` works after a single `rpm --import`.

Network is required to *build*. The **result** is what installs without network.

## Test — three suites, and they do not overlap

```bash
build/scripts/test-pfmm-airgap            # fresh install, offline          (29 assertions)
build/scripts/test-pfmm-upgrade           # in-place upgrade                (30 assertions)
build/scripts/test-pfmm-negative-control  # proves the assertions can fail  (37 controls)
```

**`test-pfmm-airgap`** installs the bundle into a throwaway systemd container with no
route off-host and asserts what the customer is paying for: offline install under
`gpgcheck=1`, readyz, the PostgreSQL-only gate, branding, live metrics, that a tampered
package is refused, and that only nginx answers off-host.

**`test-pfmm-upgrade`** exists because **a fresh install cannot observe an upgrade
defect**. All four HIGH defects fixed during this workstream — Grafana config reverted on
upgrade, dashboards frozen at first boot, static NEVRAs that made `dnf upgrade` skip our
packages silently, and a `clickhouse-server` transaction resetting `/etc/clickhouse-server`
ownership — were found by upgrading a running server by hand, and every one of them
shipped past a green from-scratch run. It synthesises a previous release from current
sources with the build stamp forced old, so NEVRAs genuinely sort below the shipped bundle
and `dnf` really has to upgrade.

**`test-pfmm-negative-control`** is the reason to believe the other two. Run it after
changing any assertion:

```bash
build/scripts/test-pfmm-negative-control [airgap|upgrade|both]   # default: both
```

It needs the containers the other two leave behind (`PFMM_TEST_KEEP=1`, the default).

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
- `readyz` returning 200 does **not** mean pfm-managed will accept a *write*.
- `pkill -f foo` matches its own `bash -c` command line. Use `[f]oo`.
- systemd-in-docker needs `--cgroupns=host`, or it exits 255 with no output.
- An air gap must be a `--internal` network, **not** a disconnected bridge — removing the
  interface is stricter than a real host and produces failures customers never see.
- `curl` exits 0 on an HTTP 500. Use `--fail-with-body` when a service's error body would
  otherwise read as success.
- Anything `rpmbuild` writes inside a container is root-owned; `chown` it back or later
  non-root runs fail with EACCES.
- The bundle stage *empties* `pfmm-repo/` rather than replacing it, so a bind-mounted test
  container keeps working. Do not reintroduce `rm -rf` there.
