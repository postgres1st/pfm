# HANDOFF — upgrade test, open gaps, and honest doubts

**Written:** 2026-08-05. Read this before touching the air-gapped bundle work.
Companion: `260729-pfmm-airgap-deliverable.md` (component map, earlier findings).

---

## STATE

| | |
|---|---|
| Branch | `feat/pmm-3.9.0-forward-port` @ `7bf26e1d0` — **pushed** to `postgres1st/pfm` |
| Grafana fork | `postgres1st-rebrand` @ `f4d9e0d87d3` — **pushed** to `postgres1st/grafana` (private) |
| Bundle | 192 packages, 549 MB, all signed, `sha256 f92c7bfb93dbcc528b6c100907f2b2568b48c87c900b0ce361593b2ee088f067` |
| Suite | `build/scripts/test-pfmm-airgap` — **27 passed, 0 failed** |
| Versions | pfm-server / pfm-managed / pfm-client **3.9.0**, pfm-grafana **12.4.5**, PostgreSQL **18.4** (PGDG), ClickHouse **26.7** |
| Signing key | `contact@postgresfirst.com`, fpr `5F629B10E9EA318F0D1042E920E4C7CB966303F6` — **placeholder**, passphrase-less |
| Release | **not tagged** — deliberately waiting on x86_64 |

Ten commits on the 3.9.0 merge (`b25e19a21`). Artifacts staged in `~/pfmm-release-3.9.0/`
(tarball, `.sha256`, public key, `RELEASE-NOTES.md`). Build output lives in
`build/.pfmm-airgap/` (gitignored).

Two containers may still be running: `pfmm-test` (long-lived demo with real metric
history) and `pfmm-bind` (last test run). Neither is needed; `pfmm-test` is the only one
with data worth keeping.

---

## THE MAIN TASK — an actual upgrade test — **DONE (2026-08-05)**

Built as `build/scripts/test-pfmm-upgrade`; **30 assertions, 0 failures, two consecutive
clean runs**. The shared
container harness moved to `build/scripts/pfmm-test-lib`, and `test-pfmm-airgap` was
refactored onto it (re-run: 28 passed, 0 failed — nothing lost). What follows is the
reasoning that led there, kept because the design questions it settles are not obvious
from reading the script.

**How the "previous release" problem was actually solved** — none of the three options
below. It is built from *today's* sources with a forced-old build stamp (`2501010000`),
so its NEVRAs sort below the shipped bundle and `dnf` genuinely has to upgrade. No
artifact to archive, no drift. The stamp must be rewritten in the generated spec: a
`%define` inside a spec beats a `--define` on the rpmbuild command line.

**Two findings from building it:**

- **The ClickHouse defect is exercised, not simulated.** `clickhouse-server`'s `%post`
  runs `clickhouse install --user clickhouse`, which chowns its directories back — and
  it runs identically on `dnf reinstall`. So the test reinstalls the package and asserts
  the ownership *was* reverted (a canary: if a future ClickHouse stops doing this, the
  test says so rather than silently testing nothing), then asserts the repair holds.
- **A negative control corrected an assertion that passed for the wrong reason.** With
  `pfm-clickhouse-perms` masked, readyz **still returns 200** — ClickHouse limps rather
  than dying, serving queries while ConfigReloader and AsynchronousMetrics fail every
  second with `Permission denied ["/etc/clickhouse-server/config.xml"]`. The 504 seen
  when this defect first bit was one outcome, not the only one. The discriminating
  assertion is whether the `pfm` user can read ClickHouse's config; that one fails with
  the repair masked and passes with it active, verified both ways.

### Both suites are now negative-controlled — `test-pfmm-negative-control [airgap|upgrade|both]`

**0 not discriminating, 3 skipped with reasons, 0 failed.** (Exact counts move whenever an
assertion is added — as they did hours later when the advisor gate landed — so read them
from the run rather than from here.) The two suites
describe different hosts (fresh
install vs upgraded), so the harness runs a phase per host and points `${container}` at
each. Shared predicates live in `pfmm-test-lib` and are controlled once, in the upgrade
phase.

**The two evidence classes are counted and printed separately, and must stay that way.**
A *state break* changes the host so the guarded property is genuinely false — proof the
assertion is wired to real state and discriminates. An *input substitution* (`[input]`)
feeds the same function a known-bad argument, which proves only that the predicate
discriminates, not that the suite aims it at the right target. Seven assertions can only be
controlled that way: a read-only mount, the PostgreSQL major, RPM signature checks, RPM
ownership, the release stamp, and the metrics timestamp. Do not let one stand in for a
state break where a state break is possible.

The harness now **asserts its own tally reconciles** (`state + input == controlled`). It
did not, at first: two bespoke controls incremented the total without recording a class,
which inflated the headline number while under-reporting how the evidence was obtained —
the very distinction the split exists to make.

Making two predicates controllable at all exposed real defects in them:
`p_postgresql_accepted` used a fixed service name and `pfm-admin add` is not idempotent,
so it passed once and would fail forever after — nothing could restore it. And
`p_postgres_18` could not discriminate its own version match; stopping PostgreSQL only
proves it reads live state, so it is now `p_postgres_major [expected]`.

The airgap suite also gained an assertion: the tamper check now verifies the *untampered*
copy still passes, because otherwise "tampering was detected" could just mean `rpm -K`
never succeeds in that environment. 0 failures.

### The earlier, upgrade-only pass

**20 controlled, 0 not discriminating, 4 skipped with reasons.** The harness breaks the
thing each assertion guards and requires the assertion to notice: baseline green →
broken red → restored green. It **sources the predicates out of `test-pfmm-upgrade`**
(`PFMM_LIB_ONLY=1`) rather than reimplementing them, because controlling a copy of an
assertion proves nothing about the assertion that runs. That is why the upgrade test's
checks were refactored into `p_*` predicates — 30 assertions reduce to 21 distinct ones.

Not controlled, and why: `p_bundle_signed` (reads a file on a read-only mount; breaking
it means building an unsigned bundle), "dnf upgrade completed" (it *is* the exit status
of the operation under test), and two pairs that reuse predicates controlled elsewhere.
`p_marker_served` needs no staged control — the upgrade run asserts it green before and
red after, on the same predicate, which is a stronger two-sided control than anything the
harness could stage.

**The harness's own failure mode is the thing to watch.** In its first run all three
failures were bugs in the *controls*, not weak assertions: `docker network connect`
installs the default route asynchronously, so a fixed sleep raced it; and
`pkill -f 'http.server 3000'` matched its own `bash -c` command line, killing the shell
before `systemctl start pfm-grafana` ran, which took grafana down and cascaded into every
control after it. Reporting a good assertion as decorative is the most damaging wrong
answer this thing can give, so `ctl` now **polls** for the state change within a budget
and always runs the restore — "could not stage the break" and "the assertion is
decorative" stay separate verdicts. Use the `[h]ttp\.server` bracket spelling with
`pkill -f`, always.

The refactor also found a latent false pass on its own: `curl` exits 0 on an HTTP 500, so
the old ClickHouse helper reported a *rejected* INSERT as success. It now uses
`--fail-with-body`.

It also turns an INSTALL.md promise into an assertion: the doc says dashboards need no
restart because Grafana rescans every 60s. The test checks propagation *before*
restarting anything, so a wrong doc fails the build.

### Why this was the top item

Four HIGH defects surfaced this session. **Every one was found by upgrading a running
server by hand, and none was visible to the from-scratch suite:**

1. `pfm-grafana` upgrade reverted our config → server dead (`mkdir /usr/share/grafana/data: read-only`)
2. Dashboards frozen at first boot → no dashboard update could ever reach an install
3. `pfm-server` / `pfm-client` had static NEVRAs → `dnf upgrade` silently skipped them
4. A ClickHouse upgrade reset `/etc/clickhouse-server` ownership → readyz 504

The suite now has structural assertions for those four *specific* mechanisms. That is a
proxy, not a test: a fifth upgrade defect of a different shape would pass cleanly. The
product ships as a bundle customers upgrade **in place**, so this is the scenario most
likely to break in the field and the least covered.

### Design (as built)

```
1. build the previous release: rebuild the 5 pfm packages into a scratch topdir with
   the stamp forced old, and mark one dashboard title with [PREV-RELEASE-CANARY];
   assemble repo A by hardlinking the 187 unchanged third-party RPMs from the bundle
2. install it offline, boot, wait for readyz 200
3. seed all four data stores: a registered postgresql service (pfm-managed's DB), a
   Grafana folder (Grafana's DB), a ClickHouse row, live metrics + a timestamp;
   plus an operator edit to a %config(noreplace) file
4. repoint baseurl at the shipped bundle, dnf upgrade with gpgcheck=1, daemon-reload,
   restart pfm.target -- exactly the sequence INSTALL.md gives the operator
5. assert data survived, config was not reverted, the gate still holds, every package
   actually moved, and the marked dashboard was replaced (before any restart)
6. dnf reinstall clickhouse-server, then assert the ownership repair holds
```

Repo A is built **unsigned** on purpose: it is a synthetic stand-in that is never
shipped, and signing it would make running a test require the release key. The upgrade
transaction — the half that matters — still runs against the signed bundle with
`gpgcheck=1`.

### Gotchas that will bite

- The bundle stage **empties** `pfmm-repo/` rather than replacing it, specifically so
  bind-mounted containers keep working. Do not reintroduce `rm -rf` there.
- Two bundles means two mounts, or repointing `baseurl` between them. INSTALL.md
  documents the repoint flow — the test should mirror it, not invent its own.
- `readyz` can take ~2 minutes after a restart. Poll, don't sample.
- A `Type=oneshot` unit that has run is `inactive` and that is correct — the "all units
  active" assertion already had to be reworded once for this.
- Grafana's dashboard JSON has a `"title"` on every panel and the dashboard's own is not
  the first. Use the search API (`/graph/api/search?query=…`, returns `[]` on no match),
  not `grep title` — that returns `"title":"Service"`.
- ClickHouse's HTTP interface needs credentials; take them from `/run/pfm/qan-api2.env`
  rather than hardcoding, so a changed password fails loudly instead of silently.
- A control that races the state it is breaking reports a **good assertion as decorative**
  — the most misleading answer available. Poll for the change; never sleep-then-check.
  **Poll the baseline too.** Four separate bugs this session came from judging state that
  had not settled, and the last one was the baseline check — the only step still sampling
  once. `readyz` returning 200 does not mean pfm-managed will accept a *write*: the cheap
  gate rejections answer immediately while registering a service still fails.
- `pkill -f <pattern>` matches its own `bash -c` command line. Use `[p]attern`.
- `pfm-admin add` is not idempotent and pmm-managed does **not** dedupe on
  node+address+port — a second add with a different name on the same host and port
  succeeds. Predicates that add a service need a unique name to be re-runnable.

---

## Added after this handoff was written

**Advisor family gate** (`23f4b617c`). The API listed 109 advisor checks of which 83
targeted MySQL or MongoDB — databases the gate refuses — so the product advertised
coverage it does not have. `ListAdvisorChecks`/`ListAdvisors` now filter by family via the
same `models.IsServiceTypeSupported` the Service gate uses, and an advisor left with no
checks is dropped. 109 → 26. Covered by a new acceptance assertion and a negative control
that widens `PFM_DB_TYPES` and watches the MySQL checks return.

**Still open on that change:** the 26 surviving checks have never been shown to actually
*fire*. `checks:start` returns 200 and `advisor_enabled` is true, but a healthy server
yields no failed checks, which is indistinguishable from checks that do not execute at
all. Registering a deliberately misconfigured PostgreSQL instance would settle it in about
fifteen minutes, and if they never run, that is a more interesting finding than the
filtering was.

## GAPS

### Blocking the release

- **x86_64 is unbuilt and untested.** Nothing cross-compiles. Procedure and signing-key
  transfer are in `~/pfmm-x86_64-build.md`. All arch-specific inputs verified reachable
  (S3 uses a different layout for x86_64; `pmm-dump` is pinned to a *different commit*
  because no x86_64 build exists at the aarch64 one).
- **No release tag.** Version scheme settled (track upstream PMM), tag string not chosen.

### Real, unresolved

- **Signing key is a placeholder.** `contact@postgresfirst.com`, no passphrase, and after
  the x64 transfer it exists on two machines. Anyone holding it can sign packages that
  customers' `gpgcheck=1` accepts. Custody needs deciding before a real delivery; a
  hardware token or CI secret changes the picture. Swapping it is one variable in
  `pfmm-airgap-vars` plus a rebuild.
- **No CI.** Nothing runs the suite automatically. Every defect found this session was
  found because someone ran it by hand.
- **Branch not merged to `main`.** A pre-push/PR hook enforces the attestation
  (`.git/pgf-review-feat-pmm-3.9.0-forward-port.md`) and will block until it names the
  current HEAD.
- **Full repo test suite never run.** `checks`/`alerting` need a live PostgreSQL on 5432;
  `TestDevContainer` needs `supervisorctl`. Only targeted packages were run.

### Latent, not currently broken

- **`%post` still writes into `/etc/clickhouse-server`** (config files plus the
  `config.xml`/`users.xml` symlinks) at install time only. Verified those symlinks
  *survived* the 26.7.1 → 26.7.2 upgrade, so this is not an active defect — but it is the
  same shape as two that did bite, and a future ClickHouse packaging change could revert
  it. The ownership half is now handled by `pfm-clickhouse-perms.service`.
- **Exporters bind `0.0.0.0:42000-42010` on every monitored host** and serve
  unauthenticated database and OS statistics. This is upstream's pull-model design and is
  documented as a firewall requirement. Push-metrics mode would bind loopback instead —
  a deployment choice nobody has made.

### Cosmetic

- Exporters are built from `percona/*` source; module paths and internal strings remain
  Percona ([[pfm-exporter-fork-branding]]).
- Grafana `plugin.json` files still carry `keywords: ["percona", …]` — metadata, not
  rendered.
- `OPERATOR_LABELS` / `OPERATOR_FULL_LABELS` remain in the fork's `constants.ts` with
  **zero consumers** (dead DBaaS code).
- 13 dangling dashboard `refId`s across two dashboards — confirmed **pre-existing
  upstream**, untouched deliberately.

---

## DOUBTS — things I am genuinely unsure about

Stated plainly so nobody inherits false confidence.

- **Signing every third-party package is a judgement call, not an obvious right answer.**
  It was necessary (ClickHouse ships unsigned RPMs; PGDG/Rocky use keys the customer
  would otherwise import individually) and it means one import covers all 192. But it
  also means Postgres1st is *vouching for* packages it did not build. If that is not the
  intended trust posture, the alternative is shipping several vendor keys and accepting a
  more complex install. Nobody has explicitly agreed to the current posture.
- **Storage sizing in the docs (~1 GB per instance per month at 30-day retention) is an
  estimate, not measured.** It came from reasoning about VictoriaMetrics and ClickHouse
  behaviour, not from observation. Worth measuring before a customer plans capacity on it.
- **`pg_monitor` is documented as sufficient for the monitoring role**, against upstream
  docs that say `SUPERUSER`. I believe upstream is over-broad, but I did not exercise
  every collector against a `pg_monitor`-only account. If a dashboard is unexpectedly
  empty, this is the first thing to suspect.
- **The 27 assertions pass consistently now, but qan-api2's dirty-migration failure was
  intermittent and load-sensitive.** The root cause (pfm-managed restarting it mid-
  migration) is understood and fixed by making seeded env byte-identical to a render, and
  restarts went 988 → 0. I am fairly confident, not certain, that no other trigger exists.
- **The ClickHouse ownership repair runs `chown -R` as root.** Verified GNU `chown -R`
  does not dereference symlinks, so it cannot be redirected at arbitrary files. Still, a
  recursive root chown is a thing to keep an eye on if those directories ever become
  writable by something less trusted.
- **`pfm-server`'s spec still carries a header comment saying it was never validated on a
  real host.** That is now stale — it has been installed and booted many times — but only
  ever in containers, never on a real Rocky/RHEL VM. SELinux in particular has never been
  exercised; every test host runs with it effectively disabled.
- **Upgrades have only been tested forward across small deltas** built minutes apart.
  A customer upgrading after months, across a larger version jump, is untested.

---

## HARD-WON KNOWLEDGE (do not rediscover)

- The from-scratch test **cannot** see upgrade defects. Four HIGH bugs proved it.
- Writing into another package's directories is the recurring root cause. Grafana config,
  dashboards, ClickHouse ownership — all the same shape. Prefer reading from the owning
  package over copying.
- `Type=exec` ordering is not readiness ordering. Both the ClickHouse race and a failed
  `After=pfm-managed` fix fell into this; the note is preserved in
  `pfm-qan-api2.service` so it is not retried.
- systemd-in-docker needs `--cgroupns=host` or it exits 255 silently. The air-gap must be
  a `--internal` network, **not** a disconnected bridge — removing the interface is
  stricter than a real host and produces failures customers never see.
- The attestation hook is real and blocks pushes. Update the note with **Python, not
  shell heredocs** — two silent write failures happened that way, and only the hook
  caught them.
- `git diff --name-only` output split on whitespace corrupts paths containing spaces
  (`dashboards/dashboards/PMM Health/…`). Use `-z`.

## MEMORY POINTERS

[[customer-rpm-deliverable]], [[systemd-in-docker-and-airgap-testing]],
[[postgres-only-gate]], [[grafana-fork-rebrand]], [[pfm-exporter-fork-branding]],
[[ui-path-rename-three-artifacts]]
