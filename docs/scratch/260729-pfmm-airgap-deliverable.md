# PFMM air-gapped customer RPM bundle — BUILT + VALIDATED

**Session:** 2026-07-28/29. Supersedes the plan in `260728-customer-rpm-handoff.md`
(that doc's blockers are cleared; keep it only for background).

## RESULT

`pfmm-server-el9-aarch64.tar.gz` — **548 MB, 192 packages**, full dependency closure.
Installs and boots on a **network-isolated** Rocky 9 host with zero internet.

- sha256 `67fa183615e0e87823495fd60b156007d6a12ea363d3b635f49869fdcd15923e`
- Scratchpad (session-local, **will be lost**): `.../scratchpad/pfmm-server-el9-aarch64.tar.gz`
- **ACTION: copy the tarball somewhere durable before this session ends.**

### Acceptance test — all passed, on a `--internal` docker network (NIC + IP, no internet)
| Check | Result |
|---|---|
| Offline install (`--disablerepo='*'`) | ✅ per INSTALL.md verbatim |
| `pfm.target` boot | ✅ all 12 pfm units active |
| `/v1/readyz` | ✅ **200** |
| Gate: mysql / mongodb | ✅ rejected — `Service type "mysql" is not supported by this deployment.` |
| Gate: postgresql | ✅ accepted |
| Branding (grafana `/graph/login`, `/pfm-ui`) | ✅ `Postgres1st Monitoring and Management`, `pfm-logo.svg` |
| Metrics | ✅ all scrape targets up, `pg_up=1` |
| Dashboards / datasources | ✅ 33 dashboards, 3 datasources |
| qan-api2 stability | ✅ 0 restarts (was 576 before the fix below) |

## PACKAGE SET
| Package | Version | Source |
|---|---|---|
| `pfm-server` | 3.0.0 noarch | ours |
| `pfm-managed` | 3.9.0 | ours, gated, from source |
| `pfm-grafana` | **12.4.5** | our rebrand fork, rebased |
| `pfm-client` | 3.9.0 | ours + all exporters built from source |
| `percona-dashboards` | 3.9.0 | ours (trimmed, 33 dashboards) |
| `percona-victoriametrics` / `percona-qan-api2` / `vmproxy` / `pmm-dump` | 3.9.0 / 1.145.0 | Percona S3 build cache |
| PostgreSQL | **18.4** | **PGDG** (not Percona ppg) |
| ClickHouse | 26.7.1 | clickhouse-stable |

## CHANGES MADE THIS SESSION (all uncommitted on `feat/pmm-3.9.0-forward-port`)

1. **grafana 12.4.5** — rebased 6 rebrand commits (127 files) from the 12.4.4 base onto
   `upstream/PMM-15190-grafana-12.4.5` (tip `fd3a0fba5b9`). **Zero conflicts.**
   Fork HEAD now `140c02f0334`. Safety branch: `backup/pre-12.4.5-rebase`.
   The old spec claimed 12.4.5 while pointing at a commit absent from our fork.
2. **`pmm-managed` → `pfm-managed`** — package, spec file, 4 binaries, cmd dirs,
   starlark exec-by-name, ExecStart, supervisord template + testdata, nginx swagger
   docroot (`/usr/share/pfm-managed`), `ProjectName` ldflag.
3. **`pmm-client` → `pfm-client`** — `pfm-server` Requires; binaries `pfm-agent`,
   `pfm-admin`, `pfm-agent-entrypoint`; CLI program names; service account;
   config `pfm-agent.yaml`; state dir `/srv/pfm-agent`.
4. **Unit-name split** (deliberate, user decision): the **client** owns
   `pfm-agent.service`; the server's self-monitoring unit is
   `pfm-server-agent.service`. Both packages land on a server host, so they cannot
   share the name. `systemdUnitName()` special-cases `pmm-agent` accordingly.
5. **`/usr/local/percona` → `/opt/postgres1st/pfmm`** — advisors, checks,
   alerting-templates (3 Go constants + spec + dev/CI). Dashboards moved to
   `/opt/postgres1st/pfmm/dashboards` (9 consumer files incl. grafana.ini, provisioning,
   pfm-init.sh, entrypoint).
6. **PostgreSQL 14 → 18 (PGDG)** — 18 files. Package names `postgresql18*` (PGDG),
   not `percona-postgresql18*`. Data dir `/srv/postgres18`, `PG_BIN=/usr/pgsql-18/bin`.
7. **Ownership fixes** — no spec creates a `pmm` user (only `pfm`), so `%attr(pmm,…)`
   silently fell back to root, and `install -d -o 1000` shipped dirs owned by an
   arbitrary local uid. Now `pfm` throughout, with a `%pre` in `pfm-managed`.
8. `AlertsPage.messages.ts`: `'Percona Alerting'` → `'PFMM Alerting'` (UI dist now has
   **zero** Percona strings).

## BUGS FOUND + FIXED BY THE ACCEPTANCE TEST (would all have shipped broken)

1. **PG15+ `public` schema** — *the* PG18 blocker. Since PG15, CREATE on schema
   `public` is revoked from PUBLIC and the schema is owned by `pg_database_owner`.
   Both grafana's and pfm-managed's migrators died on their first `CREATE TABLE`
   (`permission denied for schema public`, 42501); grafana crash-looped 27×.
   **Fix:** `provision_app_db()` in `pfm-init.sh` now makes each app role OWN its
   database (so `pg_database_owner` resolves to it), and pre-creates the
   `pmm-managed` DB/role before pfm-managed first starts.
2. **qan-api2 dirty ClickHouse migration** — `After=pfm-clickhouse.service` orders
   against `Type=exec`, satisfied when the binary is exec'd, *not* when port 9000
   listens. qan-api2 began migrating against a not-ready server, died partway, marked
   `schema_migrations` dirty, and **can never self-recover** → 576 restarts, readyz 504.
   **Fix:** `ExecStartPre` bash `/dev/tcp` gate on 127.0.0.1:9000.
3. **`systemdUnitName("pmm-agent")`** returned `pfm-agent.service` — which after the
   unit split is the *client's* unit, masked by `pfm-server`'s `%post`. The server's
   agent never started. **Fix:** map to `pfm-server-agent.service`.
4. **`pfm-agent setup` node-address** — required only when no routable address is
   autodetected; on an isolated host it hard-failed. **Fix:** detect first
   global-scope IPv4, fall back to 127.0.0.1. (First attempt used `ip route get 1`,
   which exits 2 with no default route and tripped `errexit` — use `addr show`.)
5. **Stale unit `Description=`** — still said `pmm-agent`/`pmm-managed`/`PostgreSQL 14`
   (user-visible in `systemctl status`).
6. `pfm-managed.spec` was missing `%global debug_package %{nil}` (both sibling specs
   have it) — build failed on an empty debugsource for CGO_ENABLED=0 Go binaries.

## DELIBERATELY LEFT AS `pmm-*` (do not "fix" without a migration)
- `pmm_managed` Prometheus metric namespace — dashboards query it.
- `pmm-managed` Postgres role/DB name.
- `PMM_*` environment-variable contract.
- Go module path `github.com/percona/pmm`.
- supervisord *program* names (internal keys, mapped to `pfm-*` units).

## UPGRADE-PATH DEFECTS (found 2026-08-04, by upgrading a RUNNING server)

The from-scratch acceptance test cannot see any of these: they only appear on the
**second** install. All three shipped in the first bundle. Fixed in `1cf4583c1` +
`c4bcfe629`, with structural assertions added in `49cd5072b` (suite now 19 checks).

1. **Dashboards could never update.** `grafana.ini` set `paths.plugins` to
   `/srv/grafana/plugins` and `pfm-init` copied the packaged plugins there on first
   boot — but that provisioning is guarded by `srv_provisioned` and runs exactly once.
   Upgrading `percona-dashboards` (or `pfm-managed`, which adds `pmm-compat-app` to the
   same dir) replaced the packaged files while the running server kept serving the
   first-boot copy **forever**. Fix: `paths.plugins` points at the owning package dir;
   the copy is deleted. `/srv/grafana` itself is still created — it is `paths.data`
   (dropping it broke first-boot on `PERCONA_DASHBOARDS_VERSION`).
2. **Upgrading `pfm-grafana` took the whole server down.** `%post` installed our config
   over `/etc/grafana/grafana.ini`, a path `pfm-grafana` owns, so its upgrade restored
   the stock sqlite-in-homepath config and grafana died with
   `mkdir /usr/share/grafana/data: read-only file system`. Fix: ship
   `/etc/grafana/pfm.ini` (0640 root:pfm — it carries the DB password) selected via
   `grafana server --config`, mirroring `pfm-nginx`'s existing `/etc/nginx/pfm.conf`.
   Provisioning likewise read in place from `/usr/share/pfm/grafana/provisioning`.
3. **`pfm-server` could never be upgraded at all.** `Release: 1%{?dist}` was hardcoded,
   so every build produced the identical NEVRA and `dnf upgrade` silently skipped the
   package that owns every unit, `pfm-init.sh`, and the grafana/nginx/polkit config.
   Fix: build-timestamp + commit release, matching the sibling specs.

**Restart semantics** (verified, not assumed): dashboards need **no restart** —
`updateIntervalSeconds: 60` on the dashboard provider makes grafana rescan within a
minute. Only a change to `pfm.ini` itself needs `systemctl restart pfm-grafana`.

**Lesson:** never write into a directory another package owns, and never copy packaged
files elsewhere at first boot — both guarantee drift on upgrade.

## KNOWN GAPS / FOLLOW-UPS
- Grafana plugin metadata still carries `keywords:["percona",…]` in plugin.json —
  metadata only, not rendered. Cosmetic.
- Grafana `OPERATOR_LABELS`/`OPERATOR_FULL_LABELS` ("Percona Operator for X") remain in
  `constants.ts` — verified **zero consumers** (dead DBaaS code), so not a visible surface.
- Exporters are built from `percona/*` source: module paths and internal branding are
  still Percona. Separate workstream ([[pfm-exporter-fork-branding]]).
- Grafana fork is still **unpushed** (`postgres1st/grafana` 404), so `grafana.spec`
  Source0 can't fetch — builds require the local fork at `~/Projects/postgres1st-grafana`.
- Tests: `checks`/`alerting` need a live PG on :5432; `TestDevContainer` needs
  `supervisorctl`. Both environmental, unrelated to these changes.
  `TestSavePMMConfig` now derives the path from `models.AgentConfigFilePath` instead of
  a duplicated literal (it had silently gone stale).
- **Committed** on `feat/pmm-3.9.0-forward-port`, **not pushed**: `7cf084593` (rebrand +
  `/opt/postgres1st/pfmm` + PG18), `54ff49343` (reproducible pipeline + acceptance test),
  `dea55c47d` (PostgreSQL-only dashboards, square sidebar mark, `/pfm-ui` redirects),
  `1cf4583c1` + `c4bcfe629` (the three upgrade defects above), `49cd5072b` (their tests).
  Grafana fork `~/Projects/postgres1st-grafana` has its own 3 commits on
  `postgres1st-rebrand`, also unpushed (HEAD `9721ae9`).
- Pre-push attestation `.git/pgf-review-feat-pmm-3.9.0-forward-port.md` is pinned to
  `54ff49343` — **re-pin to the current HEAD** before pushing.
- `PFMM_SERVER_VERSION` was removed from `pfmm-airgap-vars`: it was never passed to
  rpmbuild, so it looked like it set the version while doing nothing. `pfm-server`'s
  version lives in `%define full_pfm_version` in its own spec.

## HOW TO REBUILD
Builder image `pfm-rpmbuilder:9` (rocky9 + rpm-build, createrepo_c, fontconfig, unzip,
systemd-rpm-macros, systemd-devel). Base test image `pfm-rocky9-systemd`.
systemd-in-docker needs `--privileged --cgroupns=host -e container=docker`
`--tmpfs /run --tmpfs /run/lock -v /sys/fs/cgroup:/sys/fs/cgroup:rw` (without
`--cgroupns=host` systemd exits 255).
Air-gap test: `docker network create --internal` — do **not** just disconnect the
bridge, that removes the NIC entirely and is stricter than a real air-gapped host.
