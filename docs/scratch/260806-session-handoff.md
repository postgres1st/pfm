# HANDOFF — 2026-08-06

For a fresh session. Covers what landed today, what is sitting uncommitted, and what
is genuinely unknown. Companion notes: `260805-upgrade-test-and-gaps-handoff.md`
(test suites, older gaps), `260806-ci-failure-triage.md` (CI work, other session).

---

## STATE

`main` = **`1b7a409bb`**, local and remote in sync. Merged today:

| | |
|---|---|
| `3f20fbc8b` | `fix(deps)` — repoint the `go-openapi/spec` replace at `v0.22.9-percona` |
| `682c1fea8` | `fix(build)` — build `dashboards/pmm-app` in the air-gap pipeline |
| `2dcf348d2` | `fix(ci)` — unbreak CI jobs after the pfm rename |
| `ba2219a47` | `docs` — correct references that contradicted the tree; document the test suites |
| `854df1fce` | `feat(advisors)` — expose only checks this build can run (109 → 26) |
| `1b7a409bb` | `docs` — cover the advisor gate; stop hardcoding test counts |

PRs #29 and #30 merged. Earlier: PR #24 (the whole PFMM bundle workstream) merged as
`9ce84ca19`.

### Landed after the merges, from a parallel session — NOT tested by me

Committed as `3f0f827a3`, `1d7c2dc8a`, `03c219151`, `fb6427765` while this handoff was
being written (and rebased afterwards, so earlier drafts of this file name SHAs that no
longer exist).
They read correctly but I have run none of them:

1. **`.github/dependabot.yml`** — ignore `github.com/go-openapi/*`. A `replace`
   directive is invisible to Dependabot, so it bumped the family until `analysis`
   v0.25.5 called `spec` APIs the pinned Percona-Lab fork predates, and **main stopped
   compiling**. Ignoring the whole family (not just `analysis`, which is indirect and
   rides along) is the right call.
2. **`build/scripts/pfmm-airgap-vars`** — `${PFMM_SIGN_KEY-default}` instead of `:-`.
   With `:-`, `PFMM_SIGN_KEY= build-pfmm-airgap` substituted the default anyway, so the
   documented unsigned-build path was unreachable from the environment.
3. **`build/scripts/build-pfmm-airgap`** — new `write_install_md()`. Previously an
   unsigned bundle still shipped an INSTALL.md instructing `gpgcheck=1`, which would
   make dnf refuse every package in it. Now it rewrites step 2, the repo stanza and
   the tamper blockquote. Keyed off the presence of the exported key file, i.e. the
   same signal `test-pfmm-airgap`'s `p_bundle_signed` uses.

5. **`fb6427765`** — guard `write_install_md` against INSTALL.md drift.
4. **`03c219151`** — warn when building from a dirty tree. This closes a doubt recorded
   below: the NEVRA carries `HEAD` while the tarball is built from the working tree, so a
   dirty tree ships uncommitted code under a clean commit label. It is not hypothetical —
   the 2026-08-06 x86_64 bundle went out stamped `89d2fd8`, a commit that does not even
   compile, because the fixes it actually contained were uncommitted when the stage ran.
   A warning rather than a hard failure, because dirty-tree builds are how the pipeline
   gets iterated on and a `.dirty` NEVRA suffix would break the ordering
   `test-pfmm-upgrade` asserts on.

**Nothing exercises the unsigned path.** Both suites assume a signed bundle. If (2) and
(3) are to be relied on, build once with `PFMM_SIGN_KEY=` and confirm the emitted
INSTALL.md installs cleanly.

---

## THE THREE SUITES (read `build/scripts/README.md` first)

```
test-pfmm-airgap             fresh install, offline
test-pfmm-upgrade            in-place upgrade
test-pfmm-negative-control   proves those assertions can actually fail
```

The rule that matters: **a green suite is not evidence** until each assertion has been
shown to go red when the thing it guards is broken. That is not theory — the first
control run found an assertion passing for the wrong reason (readyz stays 200 with the
ClickHouse ownership repair disabled, because ClickHouse limps rather than dying).

Re-run the negative control after touching any assertion. Every assertion is a `p_*`
predicate and the harness **sources** them from the suites; controlling a copy proves
nothing about the code that runs.

---

## OPEN DOUBTS — things genuinely unknown

- **Do advisor checks actually fire?** The gate now shows 26 PostgreSQL checks instead
  of 109. But `checks:start` returns 200 on a healthy server and yields *no* failed
  checks, which is indistinguishable from checks that never execute. If they do not
  run, that change removed dead entries from a dead feature. **Cheapest test: register
  a deliberately misconfigured PostgreSQL instance and see whether anything fires.**
  ~15 minutes, and it settles the largest unknown in what merged today.

  **Someone already started this and the result was not written down.** A container
  `pfm-pgtest` (`perconalab/percona-distribution-postgresql:17`, with
  `pg_stat_statements` and `pg_stat_monitor` preloaded) was created around 13:43 by a
  parallel session and has since been removed. No `pgtest` service is registered on any
  of the running PFM servers, and no scratch note mentions advisors. So the outcome is
  unknown — ask that session before redoing it, and **write the answer down this time**,
  here or in a commit message. An experiment whose result nobody recorded costs the same
  as one never run.
- ~~A dirty working tree produces RPMs stamped with a clean commit.~~ **Addressed** by
  `03c219151`, which warns (deliberately not fails) at the `rpms` stage. Worth knowing it
  already bit once: the x86_64 bundle was stamped with a commit that does not compile.
- **`checks:start`, `checks:batchChange`, `GetFailedChecks` remain ungated.** A client
  can still start or toggle a filtered check by name. Harmless today — no MySQL or
  MongoDB service can exist — but inconsistent.

---

## GAPS

**Blocking a release**

- **x86_64 is still unbuilt.** Everything that blocked it is now merged. The VM is
  provisioned and ready: `ssh pfmm-build`, m7i.4xlarge, **RHEL 9.8 with SELinux
  Enforcing**, docker 29.7.1, repos cloned, 132 GB free. Run under `tmux` — never in an
  SSH foreground.
- **Running the suites there does NOT close the SELinux gap.** I said earlier that it
  would; that was wrong. Both suites do `docker run --privileged`, which disables SELinux
  confinement for the container outright, and even unprivileged the units inside run as
  `container_t` rather than under the host's targeted policy for those services. Closing
  the gap needs a **native** install on the VM — `dnf install pfm-server` on the host
  itself, with `getenforce` reporting Enforcing. That is a different exercise from the
  acceptance suite and nothing currently automates it.
- **No release tag.** Deferred until x86_64 passes.

**Real**

- **Signing key is a passphrase-less placeholder** (`contact@postgresfirst.com`) on two
  machines, custody undecided. Anyone holding it can sign packages a customer's
  `gpgcheck=1` accepts.
- **CI** — work in flight in another session; see `260806-ci-failure-triage.md`.
- **`documentation/`** — 710 files, still upstream Percona's, describing MySQL/MongoDB
  and published to docs.percona.com. Our bundle never builds or ships it. The only
  PFMM user documentation is the `INSTALL.md` inside the tarball.
- **Upgrades tested forward across small deltas only.** A multi-release jump is untested.
- **Storage sizing in the operator guide is reasoned, not measured.**
- **`pg_monitor` documented as sufficient** against upstream's `SUPERUSER` advice,
  without every collector exercised against it.

**Cosmetic**

- Exporters build from `percona/*`; module paths and internal strings stay Percona.
- 107 `pmm-*` references remain in our docs as *component names in prose*. That is
  deliberate — source dirs, Go module path, `PMM_*` env contract and `pmm_*` metric
  namespace are all unchanged so upstream rebases keep working. `AGENTS.md` has the
  authoritative table. **The rule: a command example or a path using `pmm-` is a bug;
  a component name in prose is not.**

---

## EVERY OPEN ITEM, including the small ones

Grouped by whether someone must decide, verify, or merely remember.

### Needs a decision

1. **Signing-key custody.** Placeholder identity, no passphrase, on two machines. Blocks
   nothing technically; blocks a real customer delivery ethically.
2. **Do the ~80 MySQL/MongoDB advisor YAMLs keep shipping?** They are filtered from the
   API but still installed to `/opt/postgres1st/pfmm/checks`. Kept deliberately so
   upstream rebases stay clean; the alternative is dropping them in `pfm-managed.spec`.
3. **Is `documentation/` (710 upstream files) ever going to be PFMM's?** Today the only
   user documentation is the `INSTALL.md` inside the tarball. Fork-and-rebrand, write a
   small PFMM set, or state that the bundle guide is the product's documentation.
4. **Does ignoring `github.com/go-openapi/*` in Dependabot cost us security updates?**
   It stops main breaking again, but the family now only moves by hand. Someone should
   own checking it periodically.
5. **Release tag string / version scheme** once x86_64 passes.
6. **Which bundle is "the" release?** `~/pfmm-release-3.9.0/` holds the old artifact
   (`sha f92c7bfb…`); the tree now builds a different one (`sha 022b5c2e…`, includes the
   advisor gate). The staged release directory is stale.

### Needs verifying — cheap, high information

7. **Do advisor checks actually fire?** The single largest unknown in what merged today.
8. **Does the unsigned build path work end to end?** Two of the three uncommitted changes
   exist to support it and nothing exercises it.
9. **Native install on the VM with SELinux Enforcing** — the only way to close that gap.
10. **Does the Grafana Advisors page render sensibly** with 26 checks? I verified the API,
    never the page. Advisors reach users through Grafana (`NAV_ADVISORS`), not the SPA.
11. **`read_more_url` on a failed check** — only appears on failures, so never observed.
    Values are per-check and often third-party (one points at mongodb.com), not uniformly
    Percona.
12. **`pfm-submodules` pins are stale again.** `pmm` is pinned at `1f60ece58`; main is
    `1b7a409bb`. Predicted and now confirmed — a pinned submodule does not follow
    `branch =`. Needs re-pinning after every merge, or automating.

### Needs remembering — housekeeping and latent risk

13. **`.worktree/ui-rebranding` still on disk** — 3.1 GB, ~250k root-owned files from
    container builds. Git no longer tracks it. Needs
    `sudo rm -rf /home/pgfirst/Projects/pmm/.worktree/ui-rebranding`. Disk was at 80%.
14. **`feat/advisor-gate-and-docs` still on the remote** after merging.
15. **`260806-ci-failure-triage.md` is untracked** — it exists only in this working copy,
    so a clean checkout elsewhere will not have it. Commit it if the CI findings matter.
16. **`docs/scratch/` now holds 11 files**, five of them predating this work and untracked.
    Convention says clean up when work concludes; the test-suite knowledge they held has
    moved into `build/scripts/README.md`.
17. **Four containers still running**: `pfmm-test` (28h, real metric history — the only one
    worth keeping), `pfmm-bind`, `pfm-airgap-test`, `pfm-upgrade-test`. The last two are
    reused by the negative control and are mutated-then-restored, so not pristine.
18. **`%post` still writes into `/etc/clickhouse-server`.** The symlinks survived a
    26.7.1 → 26.7.2 upgrade, so not an active defect, but it is the same shape as two that
    did bite.
19. **Exporters bind `0.0.0.0:42000-42010`** unauthenticated by upstream design. Documented
    as a firewall requirement; push-metrics mode would bind loopback instead.
20. **`pmm-dump` is pinned to a different commit for x86_64** than aarch64, because no
    x86_64 build exists at the aarch64 commit. Expected asymmetry, easy to misread as an
    error.
21. **The grafana fork is private**, so `grafana.spec`'s `Source0` does not resolve
    anonymously. The pipeline reads the local clone instead and needs no credentials.
22. **Three assertions remain uncontrolled**, with reasons printed by the run: `dnf upgrade
    completed` (it *is* the operation's exit status), and two pairs that reuse predicates
    controlled elsewhere.
23. **`[input]`-class controls are weaker evidence** than state breaks and are counted
    separately on purpose. Seven assertions can only be controlled that way. Never let one
    stand in for a state break that was possible.
24. **`api/MIGRATION_EXAMPLES.md` deliberately keeps `pmm-admin` examples** — it annotates
    upstream's v2→v3 API migration with the v2 CLI of the time.

## TRAPS — paid for, do not rediscover

**The recurring one: measuring something adjacent to the question.** It happened five
or more times today, and every instance produced a confident, wrong statement.

- `cmd | tail; echo $?` reports `tail`'s status. `go build` "passed" while failing.
- `git branch -r --contains` reads *local* tracking refs. I reported a pushed branch as
  unpushed and called it at-risk. `git ls-remote` asks the server.
- `endswith(("X",""))` is always true — `endswith("")` matches everything.
- Comparing a container's bind-mount against the directory it mounts: identical by
  construction, proves nothing.
- Grepping `^  family:` when the field is nested four spaces deep: "absent" was
  mis-indented, not missing.

**Test-harness specifics**

- Poll for state changes, never sleep-then-check — **including the baseline**. A racing
  control reports a sound assertion as decorative, the most damaging wrong answer it
  can give.
- `readyz` 200 does **not** mean pfm-managed will accept a *write*.
- `pkill -f foo` matches its own `bash -c`. Use `[f]oo`.
- `curl` exits 0 on HTTP 500 — use `--fail-with-body`.
- `pfm-admin add` is not idempotent; predicates that add a service need unique names.

**Build/packaging**

- systemd-in-docker needs `--cgroupns=host`, or it exits 255 silently.
- An air gap must be a `--internal` network, not a disconnected bridge.
- Anything `rpmbuild` writes in a container is root-owned; `chown` it back.
- The bundle stage *empties* `pfmm-repo/` rather than replacing it, so bind-mounted
  containers keep working.
- A pinned submodule does **not** follow its `branch =` setting. Re-check
  `postgres1st/pfm-submodules` pins after every merge to main; both were stale for
  two weeks.

---

## SUGGESTED ORDER

1. **Build once with `PFMM_SIGN_KEY=`** and install from the emitted INSTALL.md. The
   four build-script commits are already in (`3f0f827a3`, `1d7c2dc8a`, `03c219151`,
   `fb6427765`) — what is missing is that nothing exercises the unsigned path they add,
   and both suites assume a signed bundle.
2. Settle the advisor doubt (~15 min). It is the largest unknown in what just merged.
3. Run the x86_64 build on `pfmm-build` under tmux, then all three suites there. This
   is the release blocker *and* the SELinux gap in one run.
4. Decide signing-key custody before any real customer delivery.
