# PFMM 3.9.0 BETA1 → GA: P0 blocker handoff

Working document for the five GA blockers in the readiness report (§16.1), plus one the
report missed. Written to be picked up cold in a fresh session.

**Baseline:** `main` at `668772c1c`. Every file:line below was re-verified against that
commit — but re-check before editing, because line numbers drift.

**Source:** `PFW_3.9.0_BETA1_Detailed_Testing_Report_Final.pdf`, assessment dated
17 Aug 2026. Report scores: beta readiness 8/10, **GA readiness 6/10 — NOT YET**.

> **Read this first.** The report is right about most things and wrong about one. P0.4 as
> written will send you to build the wrong fix. Each entry below states what was verified in
> the tree versus what the report asserts.

---

## Status at a glance

| ID | Item | Report says | Tree says |
|---|---|---|---|
| P0.1 | SELinux release contradiction | Blocker | **Note corrected; enforcing automation still open** |
| P0.2 | Secrets in `argv` | Nothing exists | **Mostly built already** |
| P0.3 | Universal `admin/admin` | Blocker | **Addressed on `fix/p0-bootstrap-credentials`** |
| P0.3b | Hardcoded DB password | *not in the report* | **Addressed on `fix/p0-bootstrap-credentials`** |
| P0.3c | Hardcoded ClickHouse credential | *not in the report* | **Addressed on `fix/p0-3c-clickhouse`** |
| P0.4 | Exporters "unauthenticated" | Blocker | **Misreported — real defect is different** |
| P0.5 | Native host certification | Blocker | **Landed in PR #33; not automated** |

Suggested order:

1. **P0.1's documentation half** — settled below: the release note is stale on both of its
   claims. Small and unblocked; do it first to stop shipping a contradiction.
2. **P0.2** — small, half-built already.
3. **P0.4** — needs a design call on the per-agent secret and the bind address.
4. **P0.3c** — scoped out of the P0.3/P0.3b branch deliberately: it needs a `users.d` drop-in
   and two Go defaults changed, not the `/srv` secrets-file pattern that branch established.
5. **Automating the enforcing check, and P0.5's remaining certification** — both need the
   native VM, so do them in one sitting.

**P0.3 and P0.3b are done** on `fix/p0-bootstrap-credentials` — they were the first two items
of this list. Their sections below record what landed, and what the native VM still has to
confirm before either can be called closed.

---

## P0.1 — SELinux release contradiction

**Confirmed open — and both sides of the contradiction are partly wrong.** This was
originally recorded as "determine which claim is true". It has since been settled from the
tree, so start from here rather than re-deriving it.

`documentation/docs/release-notes/pfmm-3.9.0-beta1.md:80-81` makes **two** claims:

1. "This beta has not been validated on a host with SELinux in enforcing mode"
2. "we do not yet ship an SELinux policy module"

**Claim 2 is false.** The product ships a policy module today:

```
build/packages/selinux/pfw_nginx.te , .fc        policy source, in-tree
pfw-server.spec:117-118                          installs pfw_nginx.pp to
                                                 %{_datadir}/selinux/packages/
pfw-server.spec:338-339                          %post loads it: semodule -n -i
pfw-server.spec:498-500                          %postun removes it
build/scripts/test-pfw-airgap:262               asserts the .pp is in the bundle
```

It arrived in a run of native fixes — `c2d08383b build(selinux): ship a policy module`,
`77ef62a62 feat(selinux): policy rules for the /srv layout`, plus ClickHouse fcaps and nginx
relabel fixes. So the annotated tag is right on this point and the in-repo note is stale.

**Claim 1 is also stale**, though less obviously. Session notes record that enforcing mode
*was* validated by hand on RHEL 9.8 — ten units active, readyz 200, zero denials with
`dontaudit` disabled — after the assessment was written. So the tag is right and the release
note is simply out of date on both counts.

What remains true, and is the real gap: **no automated enforcing validation exists in the
repo.** `test-pfw-airgap:205` is a *presence* check on the `.pp` file, not a functional test;
no script calls `setenforce`; and the spec's own comments (lines 63-65, 333-337) say the Rocky 9
test containers have neither `semodule` nor a policy store, so the container suite
structurally cannot cover it. The validation is therefore real but not reproducible from the
tree, and not re-run per release.

- [x] Correct `pfmm-3.9.0-beta1.md` — done. Claim 2 was verifiably false and is
      corrected: the module ships, is staged, packaged and loaded. Claim 1 is now stated
      as *evidenced* rather than asserted or omitted. The tree contains falsifiable
      artifacts from a native enforcing run — `pfw-nginx.service:49` and
      `pfw-test-lib:181` both record measurements taken on RHEL 9.8, and the unit ships
      an `ExecStartPre` relabel that exists only because of what that run found. So the
      note says enforcing mode was exercised by hand on RHEL 9.8 (x86_64) and names what
      it produced, while being explicit that it is not reproducible and not re-run.
      **New gap recorded: the hand testing was x86_64 only; aarch64 has never been
      enforcing-tested**, and the product ships both.
- [x] Add a CI check that fails when documents and shipped artifacts disagree —
      `build/scripts/check-release-claims`, in its own workflow
      (`.github/workflows/release-claims.yml`). It is NOT in `main.yml`: that workflow
      carries `paths-ignore: documentation/**`, so a pull request editing only the
      release note — exactly the case this guards — would never have triggered it. The
      new workflow fires on both the documents and the packaging they are checked
      against, on pull requests and on pushes to `main`.
      Five assertions, each looking for a *false* statement rather than requiring
      particular wording, so an honest reword does not fail the build. Ten negative
      controls: eight defects go red, and two honest edits (rewording the disclaimer, a
      comment mentioning `setenforce`) stay green.
- [ ] Automate the enforcing check so the claim is reproducible rather than a one-off
      run, and cover aarch64. Still VM-only: the container images have neither
      `semodule` nor a policy store. The check fires when a `setenforce` caller appears
      under `build/` or `.github/workflows/`, so the documents cannot keep disclaiming
      coverage that has since been built.

> When checking denials on that host, remember `dontaudit` rules suppress them:
> `setenforce 0`, then `semodule -DB`, then reproduce.

**Acceptance:** release notes, tagged commit, RPM contents and the support matrix all state
the same thing — and the enforcing claim is backed by a run on a native host, not a container.

---

## P0.2 — Secrets in `argv`

**Mostly built already.** The report reads as though nothing exists. In fact a file-based
credential path works today:

```
admin/commands/management/add_postgresql.go:64
    CredentialsSource string `type:"existingfile" help:"Credentials provider"`
admin/commands/base.go:90
    func ReadFromSource(src string) (*Credentials, error)
```

It carries both `Password` and `AgentPassword`, and the same flag exists on `add_external`,
`add_haproxy` and others. **Verified: zero implementations of `--password-stdin`.**

Remaining work is therefore smaller than the report implies:

- [x] Add `--password-stdin` — on `add_postgresql`, `add_haproxy`, `add_external` and
      `add_external_serverless`. Deliberately NOT on `add_mysql`, `add_valkey`,
      `add_mongodb` or `add_proxysql`: those service types are rejected by the
      allowlist, so a second credential path there is surface on commands that cannot
      register anything. Backlog, not a gap.
- [x] Deprecate bare `--password` — warns on use, naming `/proc/<pid>/cmdline` and
      shell history so the warning says why rather than just scolding.
      `--credentials-source` already existed and is unchanged; supplying both
      `--password` and `--password-stdin` is refused rather than silently resolved.
- [x] Confirm `redactWords` covers every new path — satisfied incidentally by P0.4:
      `redactWords` already reads `AgentPassword`, which is now always set.
- [x] **The larger argv leak, not in the original list.** `pfw-admin config` passed the
      PGF WatchTower admin credential to `pfw-agent` as `--server-password` on its
      command line, so it sat in a second process's world-readable
      `/proc/<pid>/cmdline` for the duration of setup. It now travels in the child's
      environment (`PMM_AGENT_SERVER_PASSWORD`, which pfw-agent already accepted),
      which is 0400 owner-only.
- [x] `--server-password-stdin` — `--server-url` may now be given as
      `https://admin@host/` with the password on stdin, so the admin credential is out
      of pfw-admin's own argv too. Supplying it both ways is refused, and a URL with no
      username is refused rather than authenticating as nobody. Note both stdin flags
      read the same stdin: combining `--password-stdin` with `--server-password-stdin`
      fails, and the error says so.
- [ ] Single-use agent registration tokens (report §14) — still the genuinely unbuilt
      piece, and the proper replacement for putting admin credentials on the command
      line at all. Scoped out deliberately: it needs a tokens table and migration,
      create/list/revoke API, TTL and single-use semantics, and its own security
      review.

**Acceptance:** no credential reaches `/proc/<pid>/cmdline` or shell history on any documented
onboarding path.

---

## P0.3 — Universal `admin/admin` bootstrap

**Addressed on `fix/p0-bootstrap-credentials`.** The original diagnosis below cited two things
that did not survive verification while building the fix; corrected here rather than left
implicit:

- **`PMM_ADMIN_PASSWORD` is not honoured.** `managed/utils/envvars/parser.go:223` is a
  `continue` inside a switch commented *"skip various HA-related variables"* — the value is
  explicitly discarded, not applied as an override. No other code in this repository reads it;
  the remaining hits are Helm documentation for an external chart. The native default comes
  from `grafana.ini` having no `[security] admin_password`, so Grafana's own built-in
  `admin`/`admin` applies.
- **Fixing `.env.example` fixes nothing shipped.** Both `.env*.example` files are
  docker-compose inputs; the RPM path never reads them — `pfw-grafana.service:45` takes
  `EnvironmentFile=/run/pfw/grafana.env`, seeded from `/usr/lib/pfw/defaults/grafana.env`,
  which contains no `GF_SECURITY_ADMIN_*` key at all.

Fixed by generating the Grafana admin password at first boot and storing it in
`/srv/.pfw-secrets/grafana.env` (mode 0600); see the release notes for the operator-facing
read-it-back command. `.env*.example` were separately corrected to a `CHANGE_ME` placeholder
for the docker-compose path, which is unrelated to the native fix above.

- [x] Generate a cryptographically random bootstrap password at install time
- [ ] Force rotation on first login — explicit non-goal (design doc §3): once the password is
      random and per-install, forced rotation defends against a threat that no longer exists,
      and Grafana exposes no supported "must change password" flag to hang it on
- [ ] Display the generated credential once on first startup (report §14) — not done; the
      operator reads it from `/srv/.pfw-secrets/grafana.env` (see release notes)

---

## P0.3b — Hardcoded internal PostgreSQL credential — **not in the report**

**Addressed on `fix/p0-bootstrap-credentials`.** Found during review. Same class as P0.3: a
fixed credential shipped in the package.

The defect, at `5b7ca79b1` (the commit this branch was cut from):

```
build/packages/config/pfw/pfw-init.sh:126          provision_app_db pmm-managed pmm-managed pmm-managed
build/packages/config/pfw/pfw-managed.service:60   Environment=PMM_POSTGRES_DBPASSWORD=pmm-managed
build/packages/config/pfw/defaults/grafana.env:9-11
```

Those line numbers are historical -- the fix moved every one of them. The same places
on `fix/p0-bootstrap-credentials`:

```
build/packages/config/pfw/pfw-init.sh:276          provision_app_db pmm-managed pmm-managed \
build/packages/config/pfw/pfw-init.sh:277              "$(read_secret "${MANAGED_DB_SECRET}" PMM_POSTGRES_DBPASSWORD)"
build/packages/config/pfw/pfw-managed.service:62   EnvironmentFile=/srv/.pfw-secrets/managed-db.env
build/packages/config/pfw/defaults/grafana.env:11  PMM_POSTGRES_DBPASSWORD=""
```

**Correction found while building the fix: there is a third hardcoded role, and a fourth
credential that is out of scope here.** `pfw-init.sh:119` (now `:268`) provisions `grafana`/`grafana`
the same way as `pmm-managed` above — both are covered by this branch. Separately, ClickHouse's
`default` user has the fixed password `clickhouse`; it is a different mechanism (a SHA256
inside a root-owned XML file, not a `/srv` secrets file) and is tracked below as P0.3c rather
than folded into this branch.

**Severity — measured, not assumed.** Bounded but real:

- The PostgreSQL *superuser* password **is** generated properly (`openssl rand -hex 16`,
  stored 0600) and auth is `scram-sha-256`, not `trust`. Good.
- `listen_addresses` is never set, so PostgreSQL binds localhost only. Not remotely reachable.
- **But `pfw-managed.service` installs mode 0644** (`pfw-server.spec:131`) and carries the
  password in plain text. The `defaults/*.env` copies are 0640.

Net: any **local unprivileged user** can read the unit, take `pmm-managed:pmm-managed`, and
connect to the monitoring database. Local disclosure → database access.

- [x] Generate the role password at install; have the unit, `pfw-init.sh` and `grafana.env`
      read it from one 0600 file rather than embedding it

> **This cannot be verified from a checkout.** It changes first-boot ordering, and breaking
> first boot is worse than the disclosure it fixes. Test on the native VM before landing.
> It also interacts with the Tier C entry for the database *name* — renaming the role and
> rotating its password are one migration, so plan them together.
>
> **The native VM run is still pending** — this branch's final verification checklist
> requires a fresh install and an upgrade-from-BETA1 run on both arches with SELinux
> Enforcing before it can be treated as closed; see that branch's task list for the exact
> steps. Nothing here rotates or renames the database name, consistent with Tier C above.

---

## P0.3c — Hardcoded ClickHouse credential — found while fixing P0.3b

The `default` user's password is the literal `clickhouse`. Verified:
`sha256("clickhouse")` is `7e099f39b84ea79559b3e85ea046804e63725fd1f46b37f281276aae20f86dc3`,
the value shipped as `password_sha256_hex` in both
`build/ansible/roles/clickhouse/files/default-users.xml:57` and
`low-memory-users.xml:62`. The plaintext is mirrored in
`build/packages/config/pfw/defaults/grafana.env:19`, `defaults/qan-api2.env:6`,
and hardcoded in Go at `managed/services/supervisord/supervisord.go:50` and
`managed/cmd/pfw-managed/main.go:755`.

Reachability is loopback-only (`default-config.xml:218-219` sets `listen_host` to
`::1` and `127.0.0.1`), so this is local disclosure, as P0.3b was.

**Addressed on `fix/p0-3c-clickhouse`.** Two things recorded here when it was filed
turned out to be wrong, and the plan changed accordingly:

- *"`%post` rewrites the XML on every transaction"* — it does not. That whole
  ClickHouse block is inside `if [ $1 -eq 1 ]`, so it runs on first install only and
  an upgrade never re-copies those files. A one-time substitution therefore survives.
- *"so it needs a `users.d` drop-in"* — not needed, and it was the weaker option.
  ClickHouse's `.d` merge semantics against a symlinked users file were never
  confirmed, whereas substituting over the known hash in the deployed XML is
  deterministic and self-evidencing: the file states which credential the server
  accepts, which is what relocation on upgrade is now gated on.

- [x] Generate at first boot — done in `%post`, first install only
- [x] Feed the value through `PMM_CLICKHOUSE_PASSWORD` — but read from the secret
      file by each consuming unit, NOT rendered into `/run/pfw/*.env`. Routing it
      through pmm-managed's render broke the byte-identity invariant that keeps
      qan-api2's first-boot ClickHouse migration from being restarted mid-flight
      (see the note atop `pfw-qan-api2.service`); a test now asserts that invariant.
- [x] Remove the Go constants — resolved via `managed/utils/dbsecret`, with the
      historical constant kept as the fallback for the container image

Still open, and the same trade-off P0.3b made: **an upgraded host keeps the published
`clickhouse` password.** Relocation moves it into the 0600 store but does not rotate
it, because rotating means restarting ClickHouse mid-upgrade. Rotation on an existing
host is a separate change.

Verification is VM-gated like the rest: nothing here has run against a built RPM.
`build/scripts/test-pfw-init-secrets` covers the host-history decision table with no
container, but `%post` itself, SELinux labelling, and whether ClickHouse accepts the
generated password all need the native run.

---

## P0.4 — Exporter endpoint security — **the report is wrong here**

The report says ports 42000–42010 are "unauthenticated by default". **They are not.**

`ensureAuthParams` (`managed/services/agents/agents.go:135`) is called on **every** exporter
path — `postgresql.go:155`, `node.go:142`, `mysql.go:169`, `mongodb.go:86` — and writes a
prometheus webconfig with basic auth. Building "add authentication" is the wrong fix.

The two real defects:

**1. The password is not a secret.** `managed/models/agent_model.go:476-477`:

```go
func (a *Agent) GetAgentPassword() string {
    password := a.AgentID          // ← falls back to the agent ID
```

The agent ID is stored in inventory, returned by the API, and appears in logs. Authentication
in name only. (Corroborated by tests asserting `HTTP_AUTH=pmm:agent-id`.)

**2. Default bind is `0.0.0.0`.** `getExporterListenAddress`
(`managed/services/agents/agents.go:163`) returns `0.0.0.0` in its default branch; only
`PushMetrics` yields `127.0.0.1`. Port range starts at `agent/config/config.go:217`
(`Ports.Min = 42000`).

- [ ] Generate a real random per-agent secret instead of falling back to the agent ID
- [ ] Default to localhost binding, or bind + firewall
- [ ] Consider mTLS or a single agent tunnel (report §9) as the stronger option

> `HTTP_AUTH=pmm:` appears at `agents.go:139`, `mysql.go:176`, `proxysql.go:73` plus **17
> occurrences in tests**. The `pmm:` username is Tier C in the identifier map — do **not**
> rename it while fixing this.

---

## P0.5 — Native host certification

The report's status is stale: a run of native fixes landed after the 17 Aug assessment as
**PR #33**, rebase-merged — which is why no commit message mentions it and `git log --grep`
finds nothing. It is the seven-commit range `6fafedf33..e2daf510d`:

```
c2d08383b  build(selinux): ship a policy module, packaging path first
77ef62a62  feat(selinux): policy rules for the /srv layout, and drop worker_rlimit_nofile
095da4568  fix(build): compile the SELinux module in the pipeline, not in %build
06a292a16  fix(nginx): relabel /srv/nginx before start so a replaced cert still loads
91a92be51  fix(clickhouse): create /run/clickhouse-server instead of assuming it
6fafedf33  fix(clickhouse): cover the binary's fcaps in CapabilityBoundingSet
```

All are ancestors of current `main`. What remains unproven is not the packaging work but the
**validation** — see P0.1.

- [ ] Confirm which of the above are actually exercised by a native run
- [ ] Fresh native install on RHEL 9.x / Rocky 9.x / Alma 9.x, SELinux **Enforcing**, both arches
- [ ] Re-run wrong-architecture rejection and `fetch-os-dependencies.sh` (report §4: RETEST)
- [ ] Certify low-disk, low-memory and interrupted-package-transaction scenarios (all uncertified)
- [ ] Reboot / auto-start persistence

> **Container suites cannot substitute.** SELinux transitions and fcaps/bounding-set EPERM are
> both inert under Docker. Report §2 makes the same point from a ClickHouse file-capability
> defect that passed containerised CI and broke on native RHEL. When checking SELinux denials,
> `dontaudit` rules suppress them — `setenforce 0` first, then `semodule -DB`.

---

## Things that will bite you

- **`docs/pfw-identifier-map.md` is the authority for naming.** P0.4 and P0.3b both touch
  identifiers it freezes (`HTTP_AUTH=pmm:`, the `pmm-managed` database name). Read its Tier C
  section before renaming anything while fixing a P0.
- **The Go toolchain needs a pin.** `go.mod` requires ≥1.26.5:
  `GOTOOLCHAIN=go1.26.5 GOPATH=/tmp/... GOCACHE=/tmp/...`
- **`managed/` tests need a live PostgreSQL** on `127.0.0.1:5432`. A throwaway container works:
  `docker run -d -e POSTGRES_HOST_AUTH_METHOD=trust -p 127.0.0.1:5432:5432 postgres:18-alpine`
  (`postgresPassword()` falls back to empty when `/srv/.postgres_password` is absent).
- **Four tests fail on a clean tree** — `TestPackages`, `TestVersionPlain`, `TestVersionJson`,
  `TestNoCommandPrintsUsage` — all needing `pfw-admin` on `$PATH`. Baseline, not regressions.
- **Negative-control anything you fix.** Break the fix deliberately and confirm a test goes
  red. A green suite here has already hidden a false pass once.
- **UI work needs Node 22 in a container** — host Node 24 breaks the install. Pass
  `--user $(id -u):$(id -g)`, or the container leaves root-owned `node_modules` that blocks
  both `git worktree remove` and `rm`.

---

## Not P0, but adjacent and already scoped

- **N.1** — `documentation/` publishes Docker/Podman/Helm/AWS install pages for a product that
  ships as signed RPMs, and `pmm-upgrade/index.md` omits the native `dnf` path entirely.
  175 files, 1115 occurrences, blocked on a prune decision.
- **N.2** — the UI upgrade feature has no server-side implementation: `PMM_WATCHTOWER_*` is
  read nowhere, and `updater.go` only *checks* versions.
- **N.3** — `updater.go:112` builds `LatestNewsURL` as `https://per.co.na/pmm/...`, and
  `updater.go:193` queries a version service. Both matter for air-gapped installs.
- **N.4** — the QAN annotation query is unreachable through the normal flow (AND semantics
  against a tag the server only adds to *unscoped* annotations).
- **P1.8** — `.github/workflows/sbom.yml` covers only `pmm` and `vmproxy`, and names its
  artifact `pmm.spdx.json`.
