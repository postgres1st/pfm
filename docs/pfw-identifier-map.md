# PGF WatchTower identifier map (task 1.2)

Authority for the mechanical rename in tasks 1.3–1.7. Compiled 29 Aug 2026 against `main`
at `830777e39`, by inspecting the tree — not by transcribing report §15.

**Status:** decisions taken 29 Aug 2026 (see "Decisions" at the end). This is the authority
for the rename; work the "Execution order" section at the end, in that order.

**Status as of 5 Sep 2026 — Tier A steps 1 and 2 and all of Tier B are complete.** The
Tier A and Tier B tables below are a **record of what was changed**, not pending work:
the packages, units, binaries, install root and service account all carry `pfw-` today.
Re-read a table before treating any row as a task.

**A.5 (CLI help and messages) is now done too**, along with the support-funnel half of
the documentation problem. What that took is recorded in A.5 and A.6 below.

**Env vars were misfiled and are now corrected.** `PMM_AGENT_*` sat under tier C on the
grounds that env vars are internal wire. They are not: our own install guides tell
customers to type them in `docker run -e` lines and Kubernetes manifests, which is
tier B by this document's own definition. They are now `PFW_AGENT_*` with **no
fallback** — beta1 was never distributed, so there was nothing to stay compatible with,
and beta2 was the last release where the rename was free. The **server-side** `PMM_*`
variables are genuinely tier C and stay: the units, `pfw-init.sh` and the supervisord
env files agree on them, and no customer types them.

**Still outstanding:**

- **Upstream release notes.** `documentation/docs/release-notes/3.0.0…3.8.1` are
  Percona's release notes for *PMM*, describing releases this product never had. They
  carry ~220 of the ~247 `perconadev.atlassian.net` ticket links and most of the
  remaining `percona.com` references. Rewriting the links would falsify someone else's
  record; the real question is whether we ship these files at all. Not decided.
- **`docs.percona.com` / `www.percona.com` references in live docs.** Legitimate where
  a component is unmodified from upstream and the reference is *attributed* to Percona.
  Worth a pass to confirm each one is, but not a defect on its face.

**Naming convention:** `PGF WatchTower` is the *display* name; `pfw-` is the *identifier*
prefix for every package, unit and binary — not the report's longer `pfw-watchtower-*`.
`pfw` expands to **P**ostgres**F**irst **W**atchtower. Note the display name and the
identifier prefix use different abbreviations of the same product; that was a deliberate
call, not an oversight.

> Evidence in this document was gathered against `830777e39`. Tier assignments are stable, but
> the counts (occurrences, path totals, cross-unit references) drift as the tree changes —
> re-measure before relying on a number, rather than quoting it.
>
> **When re-measuring, exclude `docs/`.** This file names the very strings it counts, so a
> plain `git grep` includes the map itself and overstates every figure. `/run/pfw` reads 30
> tree-wide but 28 excluding `docs/`; `/srv/pfw` 20 vs 18; `/usr/lib/pfw` 7 vs 6. The figures
> below all exclude this document.

---

## The classification rule

Report §10.2 and §15 both say to keep deep internal names. That is only actionable with a
sharper rule, because "internal" hides two very different risks:

| Tier | Definition | Rename policy |
|---|---|---|
| **A — user-visible** | A human reads it: UI text, docs, dashboard titles, `systemctl` descriptions, RPM summaries, CLI help | Rename freely |
| **B — contract, human-typed** | A human types it or pins it: package names, unit names, binary names, install paths | Rename **only with a back-compat alias**, one release of overlap |
| **C — wire / stored** | Two components agree on it, or it is persisted: env vars, plugin IDs, proto/API paths, auth usernames, DB values | **Do not rename** in this workstream |

Getting B and C wrong is what turns a rebrand into an outage. Everything below is assigned a
tier with evidence.

---

## Tier A — user-visible (rename freely)

### A.1 Product and component names (report §15)

| Current | Target |
|---|---|
| PFMM / PFMM Server | **PGF WatchTower Server** |
| PFMM Client | **PGF WatchTower Agent** |
| PFMM Advisors | **PostgreSQL Advisors** |
| PostgreSQL dashboard section | **PostgreSQL Health** |
| Query Analytics (QAN) | **unchanged — report §15 rejected** (decision 4) |

### A.2 Grafana dashboard folders and titles

Measured from `dashboards/**/*.json`. Note these are already **PFMM**, not PMM — an earlier
rebrand pass landed:

| Current string | Occurrences | Target |
|---|---|---|
| `PFMM Annotations` | 27 | PGF WatchTower Annotations |
| `PFMM Health` | 2 | **WatchTower Health** (report §15) |
| `PFMM Query Analytics` | 2 | PGF WatchTower Query Analytics |
| `PFMM HA Health Overview` | 2 | WatchTower HA Health Overview |
| `PFMM Upgrade` | 2 | WatchTower Upgrade |
| `PFMM` | 3 | PGF WatchTower |
| `PMM UI`, `PMM suffix regex` | 1 each | leftovers — sweep |

Folder titles are display strings, but a **dashboard's folder is also how provisioning finds
it**. Confirm provisioning does not key off the folder title before renaming.

### A.3 systemd unit descriptions

Shown by `systemctl status`. All **14** units (see B.2 for the full set) used lowercase
`pfw`, e.g.:

```
pfw-managed.service        Description=pfw control plane (pfw-managed)
pfw-qan-api2.service       Description=pfw qan-api2 (Query Analytics API)
pfw-agent.service          Description=pfw-agent (Postgres1st monitoring agent)
```

Target: `PGF WatchTower control plane`, etc. **Descriptions only — not unit filenames** (Tier B).

### A.5 CLI help and messages — **done**

The tier definition names "CLI help" as tier A, and no pass had touched it. The `help:`
tags were cleared by an earlier CLI pass; what remained, and is now fixed:

| Surface | Was | Now |
|---|---|---|
| `pfw-admin status` headings | `PMM Server:` / `PMM Client:` | `PGF WatchTower Server:` / `PGF WatchTower Agent:` |
| `pfw-admin register` output | `pmm-agent registered.` | `pfw-agent registered.` |
| flag help, fatal messages | "PMM Server", "PMM Agent" | "PGF WatchTower Server/Agent" |
| diagnostic archive members | `client/pmm-agent-version.txt`, … | `client/pfw-*` |

The status headings were the most visible old branding left in the product — the first
thing a user sees from the command they run to check the install worked.

The two *correctness* bugs this section used to flag — instructions naming binaries that
do not exist on an installed system — were fixed earlier and are guarded:
`admin/commands/base/setup.go` now says `pfw-admin config`, and
`admin/agentlocal/agentlocal.go` says `pfw-agent`.

Three assertion sites move with this text and are the reason it is not a blind sweep:
`admin/commands/status_test.go` pins the status output verbatim, and both
`api-tests/server/logs_test.go` and `managed/services/server/logs_test.go` pin the
diagnostic archive member list.

### A.6 Documentation that pointed at Percona — **support funnel done**

Not a naming tier, but the same defect: the rebrand renamed link LABELS and kept the
HREFs, so the docs offered Percona's property as ours.

Fixed: `get-help.md` was Percona's sales page with a live Typeform lead-capture form;
the same form was on **107** advisor check pages; the docs site loaded a kapa.ai widget
branded "Percona AI Assistant" with Percona's website id. A "PGF WatchTower Community
Forum" pointed at `forums.percona.com`, a "PGF WatchTower Demo" at `pmmdemo.percona.com`,
and "our Trust Center" at `trust.percona.com` — two of those behind a `per.co.na`
shortener that a `percona.com` grep does not match.

Kept deliberately: `repo.percona.com` and `check.percona.com` are functional endpoints,
and `docs.percona.com` is legitimate upstream reference where it is attributed.

### A.4 RPM `Summary:` / `%description`

| Spec | Current summary |
|---|---|
| `pfw-managed.spec` | Postgres1st Monitoring and Management management… |
| `pfw-client.spec` | Postgres1st Monitoring and Management Client |
| `pfw-server.spec` | Native systemd assembly of the pfw (PMM-derived) |

"Postgres1st Monitoring and Management" is the PFMM expansion and must become PGF WatchTower
wording. `pfw-server.spec` also leaks "(PMM-derived)" into customer-visible metadata.

---

## Tier B — contract identifiers (rename only with an alias)

### B.1 RPM package names

| Spec file | `Name:` today | Target |
|---|---|---|
| `pfw-server.spec` | `pfw-server` | `pfw-server` |
| `pfw-client.spec` | `pfw-client` | `pfw-agent` |
| `pfw-managed.spec` | `pfw-managed` | `pfw-managed` |
| `grafana.spec` | `pfw-grafana` | `pfw-grafana` |
| `percona-dashboards.spec` | `percona-dashboards` | `pfw-dashboards` |
| `pfw-qan-api2.spec` | `pfw-qan-api2` | `pfw-qan-api2` |
| `victoriametrics.spec` | `pfw-victoriametrics` | `pfw-victoriametrics` |
| `pfw-dump.spec` | ~~`pmm-dump`~~ | **`pfw-dump`** — done |
| `vmproxy.spec` | ~~`vmproxy`~~ | **`pfw-vmproxy`** — done |

Decision 1 (corrected): only the **five packages the shipping pipeline builds** take the
`pfw-` prefix — `pfw-server`, `pfw-managed`, `pfw-agent`, `pfw-grafana`, `pfw-dashboards`.
**No `Provides:` or `Obsoletes:` — beta1 was never distributed**, so there is no upgrade
path from `pfw-*` to `pfw-*` and nothing to obsolete. The same reasoning removes the unit
`Alias=` stanzas, the binary symlinks and the install-root aliases: they existed only to
carry a beta1 host forward.

> **The forward path is what matters now.** beta2 → rc → GA are all `pfw-*` → `pfw-*`, same
> names, so they depend entirely on a rising NEVRA rather than on any aliasing. The static
> release that used to make this a live defect is **fixed**: `build-pfw-airgap` now builds
> with `--define "release 1.<utc>.<shortcommit>"`, and `test-pfw-airgap`'s
> `p_release_stamped` asserts the release is not `1.el9` for every package.

**`pfw-victoriametrics` and `pmm-dump` must NOT be renamed** — but the rule is about
*who builds them*, not the names themselves. A downloaded package ships under its upstream
name whatever our spec says, so pointing `pfw-server`'s `Requires:` at a name nothing
provides makes `dnf install pfw-server` fail on unresolved dependencies.

That is why `percona-qan-api2` **was** renamed to `pfw-qan-api2` once the pipeline started
building it from source (its source is in this repo), and why `vmproxy` — also built here
now — carries our summary and URL.

**That precondition has since been met for everything.** `stage_s3` is retired and every
server component is built from source, so nothing is fetched prebuilt any more. The
freeze above therefore no longer applies, and every row is now done: `pmm-dump` →
`pfw-dump`, `victoriametrics` → `pfw-victoriametrics`, `vmproxy` → `pfw-vmproxy`.

### Installed binary names are namespaced too — not only the packages

`/usr/sbin` is shared with every other package on the host, so an unqualified binary
name is a latent install conflict: RPM refuses to install two packages that own the
same path. `vmproxy`, `vmalert` and `victoriametrics` were all bare, and `vmalert` in
particular is the name upstream VictoriaMetrics packaging uses. All three now install
as `pfw-*`:

| Was | Now |
|---|---|
| `/usr/sbin/vmproxy` | `/usr/sbin/pfw-vmproxy` |
| `/usr/sbin/vmalert` | `/usr/sbin/pfw-vmalert` |
| `/usr/sbin/victoriametrics` | `/usr/sbin/pfw-victoriametrics` |

The **grafana fork is the deliberate exception** — it replaces upstream grafana and has
to own `/usr/sbin/grafana`, `grafana-server` and `/usr/bin/grafana-cli` for the drop-in
to work. `test-frozen-identifiers` asserts every other installed binary is prefixed.

One coupling this created is worth naming, because it fails only at runtime:
`ValidateAlertingRules` execs its binary by **bare name from PATH**, so it and
`victoriametrics.spec` must agree or alerting-rule validation dies with "executable file
not found" on the customer's server. That pair is asserted.

### supervisord program names stay `pmm-*` — deliberately

`build/ansible/roles/supervisord/files/pmm.ini` and `managed/services/supervisord/pmm_config.go`
declare `[program:pmm-init]`, `[program:pmm-managed]`, `[program:pmm-agent]`, and
`managed/services/encryption` drives them by program name. **These are not a missed sweep.** They are keys into `systemdUnitName()`
(`managed/services/supervisord/systemd.go`), which translates a supervisord program name into
the native unit that replaces it:

| supervisord program | systemd unit |
|---|---|
| `pmm-managed` | `pfw-managed.service` |
| `pmm-agent` | `pfw-server-agent.service` (special-cased; the client's `pfw-agent.service` is masked) |
| `victoriametrics` | `pfw-victoriametrics.service` |

The function is `"pfw-" + strings.TrimPrefix(name, "pmm-") + ".service"`, so renaming the
program names to `pfw-*` produces `pfw-pfw-managed.service`. The **commands** beside them were
rebranded — they must name real binaries, e.g. `[program:pmm-agent]` runs `/usr/sbin/pfw-agent`
— which is what makes the file look half-swept when it is not. Revisit only when the Docker
image path is rebranded, since that is where the program names originate. Distinct from the
frozen `pmm-managed` **database and role**, which is Tier C for an unrelated reason.

The native upgrade path is `dnf upgrade 'pfw-*'`
(`build/packages/pfw-airgap-INSTALL.md`). It used to need `vmproxy` named separately, which
made the glob a lockstep hazard; now that every package carries the `pfw-` prefix the glob
covers all of them and that risk is gone.

### The process manager is selected, never doubled

`selectProcessManager` picks ONE backend: `PFW_PROCESS_MANAGER` forces it, otherwise
auto-detect chooses systemd only on positive evidence — systemctl present AND `pfw.target`
installed — so it cannot misfire on a half-provisioned host. Running both managers at once
would mean two supervisors owning the same services and fighting over their state; that is
not a supported configuration and should not be made one.

That env var was `PMM_PROCESS_MANAGER` and is now `PFW_PROCESS_MANAGER`. It is set in two
places that must agree — the constant in `systemd.go` and `Environment=` in
`pfw-managed.service` — and a mismatch is silent, because the server just falls back to
auto-detection and still resolves to systemd on a native host. Asserted.

**Two call sites used to bypass the selection entirely**, hardcoding `supervisorctl`:
`managed/services/encryption/encryption_rotation.go` and `managed/services/server/logs.go`.
The first mattered: `pfw-encryption-rotation` is shipped by `pfw-managed.spec`, and on a
native install it died at its first step with "executable file not found" — fail-safe, since
it stops before touching data, but the key-rotation feature did not work on the only platform
beta2 ships. The rotation path now routes through `supervisord.ControlSupervisedService`.

### B.2 systemd unit filenames

**14 units**, all `pfw-*` → `pfw-*`:

- 12 `.service` files in `build/packages/config/pfw/`
- `build/packages/config/pfw/pfw.target`
- `build/packages/config/pfw-agent.service` — **outside** that directory, and the one
  installed on every monitored host by the client package. Easiest to miss; costliest to.

Decision 2: rename now, **with `Alias=` on every unit** so the old `pfw-*` names keep working
for operator runbooks and existing automation. Two things `Alias=` does not cover, and both
must be handled in the same change:

- `Alias=` only takes effect on `systemctl enable`, so upgrades must re-enable units or ship
  the compat symlinks directly in the RPM.
- **63 cross-unit references** (`After=`, `Requires=`, `WantedBy=`, `PartOf=`) inside the 14
  units point at the old names — dominated by `pfw-init.service` (23) and `pfw.target` (22).
  They resolve through aliases, but leaving them stale means a dependency graph that reads
  entirely in old names. Update them with the filenames, in one commit.
- No unit currently declares `Alias=`, so this is new machinery, not an edit to existing
  aliasing.

The unit rename is therefore **14 filenames + 63 internal references + 14 new `Alias=`
stanzas**, not 14 renames.

### B.2a Non-unit `pfw-` packaging files

The same directory ships four more `pfw-`prefixed files that rename with the units and are
easy to miss because they are not `.service` files:

| Source | Installed as |
|---|---|
| `pfw-sysusers.conf` | `/usr/lib/sysusers.d/pfw.conf` |
| `pfw-tmpfiles.conf` | `/usr/lib/tmpfiles.d/pfw.conf` |
| `pfw-polkit.rules` | polkit rule (see T6 in the native-packaging notes) |
| `pfw-init.sh` | first-boot provisioning script |

### B.2b Where the compat symlinks actually live

`build/packages/rpm/client/pfw-client.spec` `%post` creates them, and `%postun` removes them:

```
%post    for file in pfw-admin pfw-agent
         ln -s /opt/postgres1st/pfw/bin/$file /usr/bin/$file
         ln -s /opt/postgres1st/pfw/bin/$file /usr/sbin/$file
```

Both loops must gain the old names when the binaries rename. **No `pmm-*` symlinks are
shipped today**, so the earlier `pmm-*` → `pfw-*` rename left no compatibility behind — which
is why the user-facing strings that still say `pmm-admin` name a command that does not exist
(see A.5).

### B.5b The client's *second* service account

B.5 covers the server's `pfw` account. The **client package creates a separate one**:
`pfw-agent` user and group, via `groupadd`/`useradd` in `%pre` (`pfw-client.spec:94-96`),
not via sysusers.d. It owns `/opt/postgres1st/pfw` and the agent config, and `%postun`
deletes it on uninstall. Two accounts, two mechanisms — both need a decision.

### B.3 Binaries

Decision 3: `pfw-` prefix, keeping the existing short component words:

| Today | Target |
|---|---|
| `pfw-admin` | `pfw-admin` |
| `pfw-agent` | `pfw-agent` |
| `pfw-managed` | `pfw-managed` |
| `pfw-agent-entrypoint` | `pfw-agent-entrypoint` |
| `pfw-encryption-rotation` | `pfw-encryption-rotation` |
| `pfw-managed-init` | `pfw-managed-init` |
| `pfw-managed-starlark` | `pfw-managed-starlark` |

Ship `pfw-*` symlinks for one release — `pfw-admin add postgresql` is in every runbook and
customer script.

> The binary names above come from the `Makefile`, not from `bin/`. **`bin/` is gitignored
> and holds zero tracked files** — any stale `pmm-*` artifacts there are local build output
> on one machine, not repo state, and a fresh clone will not have them.

**The user-facing strings rename with the binaries — same commit.** The "stop telling
users to run binaries that do not exist" fix corrected 91
occurrences across 31 files that still named `pmm-agent`/`pmm-admin`, binaries the earlier
rename had already removed. Two were instructions ("Please run `pmm-admin config`") that
produced *command not found*.

Those strings now say `pfw-agent`/`pfw-admin`, matching what is installed **today**. When the
binaries become `pfw-*`, they must move again in the same commit, or the identical bug
reappears pointing forward instead of backward. `admin/commands/status.go:48-49` and
`admin/README.md`'s sample output are part of that set, and `TestStatus` asserts on it, so a
missed update fails the build rather than shipping silently.

The only reason they were not written as `pfw-*` up front is that a message must name a
binary that exists at the time it is printed.

**Collision check:** `pfw-agent` is both a binary (B.3) and the `pfw-client` package's new
name (B.1). That is intentional and matches how `pfw-agent`/`pfw-client` relate today, but
worth stating so nobody "fixes" it later.

### B.4 Install path

Decision 5 (amended): **unify both roots into `/opt/postgres1st/watchtower`.**

The leaf names the *product*, not the org: `/opt/postgres1st/` already carries the vendor
identity, so a prefix repeating it would encode the org twice. Paths are read far more
often than typed — in `ls`, journal output, unit files and docs — and operators reach the
binaries through `/usr/sbin` symlinks rather than the full path.

Under `/opt` there are two roots, split by content type — an artifact of an earlier partial
rename, not a design:

| Root | Holds | Scope |
|---|---|---|
| `/opt/postgres1st/pfw` | `bin`, `collectors`, `config`, `data`, `exporters`, `tmp`, `tools` | 31 files, 216 occurrences |
| `/opt/postgres1st/pfw` | `advisors`, `alerting-templates`, `checks`, `dashboards` | 16 files, 49 occurrences |

**Collision check on directory names: clean.** The two subtrees share no directory name.

**But that check was insufficient.** It compared paths, not *package ownership*. The client
claimed its whole root recursively (`%attr(-,pfw-agent,pfw-agent) /opt/postgres1st/pfw`),
which was safe only while the roots were separate. Merged, that claim swallows the server's
`advisors/`, `checks/` and `dashboards/` — files owned by a different package and a different
user. The client now claims only the five subtrees it installs, and all three packages
declare the shared parents with identical attributes.

**But `/opt` is not the whole story.** Three further `pfw` path families exist, and decision 5
as originally written ("unify *both* roots") would leave them behind:

| Path | Occurrences | What it is |
|---|---|---|
| `/run/pfw` | 28 | tmpfs runtime dir, recreated each boot by `tmpfiles.d` |
| `/srv/pfw` | 18 | persistent data under the `/srv` root |
| `/usr/lib/pfw` | 6 | `defaults/*.env` seeds referenced by `tmpfiles.d` |
| `/usr/share/pmm-server` | 12 | nginx maintenance page and static assets |
| `/srv/pmm-distribution` | 4 | distribution marker file written at first boot |

The last two still carry the **`pmm`** name, not `pfw` — the earlier rename never reached
them.

Renaming these is mostly mechanical, but `/srv/pfw` holds **persistent data**, so moving it
needs a migration step on upgrade, not just a path change.

### B.5 The `pfw` service account — previously untiered

`pfw-sysusers.conf` creates a system user and group named `pfw`, and it is load-bearing:

- **24** `User=pfw` / `Group=pfw` entries across the units (11 units each)
- `tmpfiles.d` creates `/run/pfw`, `/run/postgresql`, `/run/clickhouse-server`, `/run/nginx`
  owned `pfw:pfw`
- RPM `%attr(-,pfw,root)` file ownership
- 4 files perform a `pfw:pfw` chown

**Treat as Tier C for now.** Unlike a path, the account name is baked into *file ownership on
disk*: renaming it means chowning every persistent file under `/srv` during upgrade, and an
interrupted upgrade leaves data owned by an account that no longer exists. The sysusers
comment notes units reference `pfw` by name and the numeric UID is deliberately unpinned —
which makes the *name* the contract.

- [ ] Decide explicitly whether the account renames. Not renaming it leaves `pfw` visible in
  `ps`, `systemctl status` and file listings after the rebrand.

This is the highest-blast-radius item in the map, because the binary root is **agent-visible**.
Four hardcodes must change together, and the third one is the dangerous one:

```
agent/config/config.go:40             pathBaseDefault
managed/models/agent_model.go:103     AgentConfigFilePath
managed/services/agents/agents.go:128 paths_base handed to agents older than 2.22.99
managed/services/nomad/nomad.go:38    pathToNomad
```

Required alongside the rename:

- **Compat symlinks** `/opt/postgres1st/pfw` → `watchtower` and `/opt/postgres1st/pfw` → `watchtower`,
  shipped by the RPM. Without them, an older agent handed the old `paths_base` finds nothing.
- Keep `agents.go:128` returning the **old** path for pre-2.22.99 agents, or verify the
  symlink covers it — decide explicitly rather than by omission.
- RPM `%files`/`%dir`, the 12 systemd units, and the Grafana provisioning `path:` in
  `build/ansible/roles/grafana/files/dashboards.yml` all move together.

---

## Tier C — do not rename in this workstream

| Surface | Evidence | Why it is frozen |
|---|---|---|
| **`PMM_*` environment variables** | **153 distinct** across `managed/`, `agent/`, `admin/` | Every deployment, compose file, and systemd env file sets these. Renaming needs a dual-read deprecation cycle, which is its own project. |
| **Grafana plugin IDs** | `pmm-app`, `pmm-qan-app-panel`, `pmm-compat-app` | Referenced by dashboard JSON and Grafana's loader; renaming invalidates stored dashboards. |
| **Exporter auth username** | `HTTP_AUTH=pmm:` — 3 production sites (`agents.go:139`, `mysql.go:176`, `proxysql.go:73`) **and 17 more in tests** | Agreed between server and every running exporter. Changing it is an agent-compat break. See P0.4 — this line is being touched for security reasons anyway; do not also rename it. |
| **Internal Go module paths** | 1951 `PMM` occurrences in non-test Go | Report §10.2 explicitly says retain and document. |

The `pmm_annotation` tag was previously listed here and has since been renamed — see
decision 6. It is the exception that shows the rule: moving a stored identifier needed a
migration shape (match both tags), not a substitution.

### Open naming inconsistencies, deliberately not changed

| Surface | Current | Why it was left |
|---|---|---|
| Fork-added env var | `PFW_DB_TYPES` | The only `PFW_`-prefixed variable. Nothing has shipped, so renaming to `PFW_` is free — but it is read by `api-tests/helpers.go` and the negative-control suite, and the frozen-identifier guard's "no `PFW_*` env var appeared" assertion would fire on it. A decision, not a defect. |
| Grafana page title | "Postgres1st Monitoring and Management" | Served by the `postgres1st/grafana` fork, a separate repository. Until that is rebranded, `/graph/login` and `/pfw-ui` show different product names. |
| Config filenames | `pfw-agent.yaml`, `/etc/nginx/pfw.conf`, `/etc/grafana/pfw.ini` | Renaming a config file orphans the existing one on upgrade. `pfw-agent.yaml` in particular carries the agent ID and server URL. |
| Data directories | `/usr/share/pfw`, `/usr/share/pfw-managed`, `/srv/pfw-agent` | `%{_datadir}` paths are pinned deliberately (see B.1); `/srv` is persistent data needing a migration. |

### C.1 Surfaces found in review pass 5 — previously uninventoried

| Surface | Evidence | Tier and why |
|---|---|---|
| **Internal PostgreSQL database, role and password** — all literally `pmm-managed` | `pfw-init.sh:126` (`provision_app_db pmm-managed pmm-managed pmm-managed`), `pfw-managed.service:60,64,65`, `defaults/grafana.env:9-11` | **C.** Stored in the database cluster. Renaming needs `ALTER DATABASE`/`ALTER ROLE` on upgrade, coordinated across the unit, the init script and Grafana's env file. |
| **nginx-served URL paths** | ~~`/pmm-static` and `/percona-blog/feed`~~ | **Done — removed, not renamed.** Both were dead or wrong natively: `/pmm-static` aliased `/usr/share/pmm-server/static`, which the RPM never creates (that tree is a Docker-image artifact), and `/percona-blog/feed` proxied percona.com from a product documented as needing no internet. Nothing in the UI called either. |
| **nginx config filename** | ~~`nginx/conf.d/pmm.conf`~~ → `pfw.conf`, `pmm-ssl.conf` → `pfw-ssl.conf` | **Done.** The `include` glob in `nginx.conf` moved with them; a mismatch there is silent (nginx starts and serves nothing), so `test-frozen-identifiers` now asserts the glob matches every conf.d file the spec installs. |

The map previously tiered env vars, plugin IDs, protos, API paths and the exporter auth
username, but had **no entry at all** for the internal database identity or for
nginx-served URL paths. The `/v1/...` neutrality claim covers the API, not these.

**Already neutral — no work needed** (verified, not assumed). Note this is a claim about
*package declarations and URL paths*, not filenames: `api/` still has 14 tracked paths
containing `pmm` (generated clients such as `pmm_roles_api_client.go`), which are internal
and correctly left alone.

- **Proto packages** — `accesscontrol.v1beta1`, `actions.v1`, `advisors.v1`, … no `pmm` in any
  `package` declaration under `api/`.
- **HTTP API paths** — `/v1/accesscontrol/roles`, `/v1/advisors`, … already product-neutral.

This is the good news in the map: the rename does **not** touch the API contract.

---

## Scope by area (tracked paths containing `pmm`)

| Area | Paths | Dominant tier |
|---|---|---|
| `ui/` | 577 | A (strings) + C (plugin IDs) |
| `dashboards/` | 313 | A (titles) + C (plugin IDs) |
| `documentation/` | 113 | A |
| `build/` | 15 | B |
| `api/` | 14 | C — leave |
| `managed/` | 11 | C — leave |
| other | 6 | mixed |

Path counts are a poor proxy for effort: `api/` and `managed/` are Tier C and should barely be
touched, while `ui/` and `dashboards/` carry nearly all the user-visible strings.

---

## Decisions (taken 29 Aug 2026)

1. **All packages take `pfw-`**, including the four still on Percona/PMM names. Each carries
   `Provides:` + `Obsoletes:` for its old name.
2. **systemd units rename now, with `Alias=`** for the old `pfw-*` names.
3. **Binaries take `pfw-`** — `pfw-admin`, `pfw-agent`, … with `pfw-*` symlinks for one
   release. `PGF` = PostgreSQL First.
4. **"Query Analytics" stays.** Report §15's rename to "Query Intelligence" is **rejected**:
   QAN carries no Percona branding, is well understood by DBAs, and the surfaces that would
   make the rename coherent — `/v1/qan:*`, the `pmm-qan-app-panel` plugin id, and the
   ClickHouse database `pmm` — are all Tier C and cannot move. A display-only rename would
   leave users seeing "Query Intelligence" in the UI and `qan` everywhere else.

5. **Install roots unify into `/opt/postgres1st/watchtower`** — replacing the two
   separate roots that existed before, `/opt/postgres1st/pfm` (binaries) and
   `/opt/postgres1st/pfmm` (content). This supersedes the earlier
   "leave it" recommendation. See B.4 for the agent-compat requirement.

The identifier prefix is `pfw-`, **not** the report's `pgf-watchtower-*`. "PGF WatchTower"
remains the display name, so the prefix and the display name use different abbreviations of
the same product — deliberate, not an oversight. The install root takes no prefix at all: it
is `/opt/postgres1st/watchtower`, a directory rather than a package name, so it reads as a
word.

6. **`pmm_annotation` → `pfw_annotation`** — done; see the `pfw_annotation` rename
   commit. Initially recommended
   against and then requested anyway; implemented so no history is lost. The writer emits the
   new tag, and dashboards match **both** tags rather than swapping, because annotations
   already stored in Grafana carry the old one. 50 tag arrays across 26 dashboards carry
   both; none carry only the old tag.

   The Query Analytics dashboard takes the new tag alone: its query omits `matchAny`, so it
   uses AND semantics and a second tag would match nothing.

   This is the one Tier C item that has moved. It stays in Tier C for anyone reading the
   table above — the point is that moving it required a migration shape, not a substitution.

---

## Suggested execution order for 1.3–1.7

1. ~~**Tier A strings** in `dashboards/` and `ui/`~~ — **done**.
   **1b. Repo and component docs — not yet done, and originally missing from this map.**
   21 files, 83 occurrences outside `documentation/`: `README.md` (whose H1 still reads
   "Postgres1st Monitoring and Management (PFMM)"), `CONTRIBUTING.md`, `AGENTS.md`,
   `CODE_OF_CONDUCT.md`, the per-component READMEs/AGENTS files, and — most visibly —
   `build/packages/pfw-airgap-INSTALL.md` and `-INSTALL-CLIENT.md`, which ship to
   customers. Unblocked: none of this depends on the `documentation/` prune decision.
   Leaving it makes the rebrand incoherent at the repo's front door.
   `documentation/` itself (175 files, 1115 occurrences) remains blocked — see the prune
   question before spending effort there.
   Skip anything containing "Query Analytics" (decision 4).
2. ~~**Tier A** in RPM summaries and unit descriptions.~~ — **done**.
3. **Tier B packages** — `pfw-*` names, `Provides:`/`Obsoletes:`, **and the
   `dnf upgrade 'pfw-*' vmproxy` glob, all in one commit.** Splitting
   these is how upgrades silently break.
4. **Tier B units** — 14 filenames + 63 cross-references + 14 `Alias=` stanzas, plus the
   four non-unit `pfw-` files in B.2a. One commit.
5. **Tier B binaries** — `pfw-*` with `pfw-*` symlinks.
6. Leave Tier C entirely. Add a regression test asserting the 153 `PMM_*` env var names, the
   three plugin IDs, and the ClickHouse database name `pmm` are unchanged, so a later sweep
   cannot quietly break them.

Steps 3 and 4 are the ones that can break a running install; 1, 2 and 5 are low risk.
