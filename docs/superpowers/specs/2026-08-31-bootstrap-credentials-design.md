# Design: generated bootstrap credentials for the native stack (P0.3 + P0.3b)

**Status:** proposed
**Branch:** `fix/p0-bootstrap-credentials`
**Baseline:** `main` at `67104cdd6`
**Source:** `docs/p0-blockers-handoff.md` §P0.3, §P0.3b

---

## 1. Problem

The native RPM stack ships three constant credentials. Two are database roles created at
first boot with their password equal to their name; the third is Grafana's `admin`/`admin`,
which applies because nothing ever sets it.

| Credential | Where it is fixed | Mode on disk |
|---|---|---|
| `pmm-managed` PostgreSQL role | `pfw-init.sh:126`, `pfw-managed.service:60`, `defaults/grafana.env:11` | **0644** on the unit |
| `grafana` PostgreSQL role | `pfw-init.sh:119`, `grafana.ini:11` → `/etc/grafana/pfw.ini` | 0640 |
| Grafana `admin` login | nowhere — `grafana.ini` has no `[security] admin_password` | n/a |

The 0644 unit file is the only one that is a disclosure: `pfw-server.spec:131` installs every
unit `0644`, and `%files` (line 322) adds no `%attr`, so any local unprivileged user can read
`Environment=PMM_POSTGRES_DBPASSWORD=pmm-managed` and connect to the monitoring database.
PostgreSQL never sets `listen_addresses`, so reachability is loopback-only and the
*superuser* password is already generated properly (`openssl rand -hex 16`, 0600). The
exposure is therefore local-disclosure → monitoring-database access, not remote compromise.

The Grafana admin credential is a separate and larger problem: it is the product's front
door, reachable over the network, and it is `admin`/`admin` on every install.

---

## 2. Corrections to the handoff

Three claims in `docs/p0-blockers-handoff.md` did not survive verification. They change what
the fix has to touch, so they are recorded here rather than left implicit.

**`PMM_ADMIN_PASSWORD` is not honoured.** The handoff cites
`managed/utils/envvars/parser.go:223` as evidence it is respected. That line is a `continue`
inside a switch commented *"skip various HA-related variables"* — the value is explicitly
discarded. No other code in this repository reads it; the remaining hits are Helm
documentation for an external chart. The native default comes from `grafana.ini` having no
`[security] admin_password`, so Grafana's own built-in `admin`/`admin` applies.

**Fixing `.env.example` fixes nothing shipped.** Both `.env*.example` files are
docker-compose inputs. The RPM path never reads them: `pfw-grafana.service:45` takes
`EnvironmentFile=/run/pfw/grafana.env`, seeded from `/usr/lib/pfw/defaults/grafana.env`,
which contains no `GF_SECURITY_ADMIN_*` key at all.

**There is a third hardcoded role, and a fourth credential out of scope.** The handoff lists
only `pmm-managed`; `pfw-init.sh:119` provisions `grafana`/`grafana` the same way. Separately,
ClickHouse's `default` user has the fixed password `clickhouse` — `default-users.xml:57` and
`low-memory-users.xml:62` ship `password_sha256_hex 7e099f39…`, verified as
`sha256("clickhouse")` — mirrored in `defaults/grafana.env:19`, `defaults/qan-api2.env:6`,
`supervisord.go:50` and `pfw-managed/main.go:755`.

---

## 3. Scope

**In scope.** The `pmm-managed` role password, the `grafana` role password, and the Grafana
`admin` bootstrap password, on the native systemd/RPM stack — plus the shipped operator tool
that hardcodes the same constant (§5.5).

**Out of scope, tracked as P0.3c.** The ClickHouse `default` credential. It is a different
mechanism — a SHA256 inside a root-owned `/etc/clickhouse-server` XML that `%post` rewrites
on every transaction, plus two Go defaults — so it wants a `users.d` drop-in rather than the
`/srv` secrets-file pattern used here. Folding it in would double the change and widen the
VM test matrix for no shared machinery. This design adds it to the handoff with the evidence
above.

**Explicit non-goal: forced rotation on first login.** P0.3's third bullet asks for it.
Grafana applies `[security] admin_password` only when it creates the initial admin user, and
exposes no supported "must change password" flag on that user; implementing this means a
post-start API call against an internal field. Once the bootstrap password is random and
per-install, the threat forced rotation defends against — a *known* password — is already
gone. Dropping it is a deliberate YAGNI call, not an oversight.

---

## 4. The relocation / generation split

The decision taken was *fresh installs only*. That cannot mean "skip upgraded hosts": the RPM
installs the new unit files on upgrade too, so if `pfw-managed.service` stops carrying
`Environment=PMM_POSTGRES_DBPASSWORD=…` and starts reading an `EnvironmentFile`, an upgraded
host has no such file and `pfw-managed` starts with an empty password and never connects.

The fix therefore separates two things the handoff bundles:

- **Relocation** — moving the secret out of the world-readable unit into a 0600 file. Applies
  to **every** host, fresh and upgraded. This is what closes the disclosure.
- **Generation** — replacing the constant with `openssl rand -hex 16`. Applies to **fresh
  installs only**. This is what makes the credential unguessable.

On an already-provisioned host (`/srv/pmm-distribution` present) with no secrets file, the
*historical literal* is written into the 0600 file. No `ALTER ROLE`, no PostgreSQL startup, no
database traffic on the upgrade path — the migration is one `printf`.

### 4.1 Why relocation cannot live in `pfw-init.sh` alone

`pfw-init.service` is `Type=oneshot` with `RemainAfterExit=yes`, and the spec uses
`%systemd_postun pfw.target` (line 296), **not** `%systemd_postun_with_restart`. So an RPM
upgrade performs a `daemon-reload` and nothing else: the new unit files land on disk, running
services keep their old environment, and `pfw-init.service` stays `active (exited)` — it does
not re-run. `Requires=pfw-init.service` does not re-trigger an already-active unit, and
`pfw-init.service` has no `PartOf=pfw.target`, so a target restart does not sweep it up either.

The failure this produces is sharp. An operator runs `dnf upgrade` and then
`systemctl restart pfw-managed`; systemd reads the new unit, finds a non-optional
`EnvironmentFile` that nothing has created, and refuses to start the service. The stack
returns only after a reboot, and only because a reboot happens to re-run `pfw-init`.

**Relocation therefore happens in `%post`, as root, on upgrade transactions only:**

```
if [ $1 -ge 2 ] && [ -f /srv/pmm-distribution ]; then
    for each secret file that does not exist:
        install -m 0600 -o pfw -g pfw  <legacy literal>
fi
```

The `$1 -ge 2` guard keeps it off fresh installs, where `/srv/pmm-distribution` does not exist
yet and `pfw-init.sh` must *generate* rather than relocate. This mirrors the root-side
first-boot fixes already in `%post` (line 243), which exist for exactly this class of problem:
work `pfw-init.sh` cannot do because it runs unprivileged and sandboxed.

`pfw-init.sh` keeps its own idempotent copy of the same logic. Belt and braces: `%post` covers
the upgrade-then-restart window, `pfw-init.sh` covers reboots, re-provisioning, and any host
whose `%post` was interrupted.

One mitigating behaviour, found during review and worth recording because it bears on the risk
judgement: `models.initWithRoot` (`managed/models/database.go:1388`) self-heals. When
pmm-managed hits a `28000`/`28P01` auth error it reconnects as the superuser using
`/srv/.postgres_password` and runs `ALTER USER … WITH PASSWORD '<configured>'`
(`database.go:1441`). A password *mismatch* on the pmm-managed role therefore repairs itself on
the next start rather than wedging — meaning rotation-on-upgrade would have been cheaper than
the handoff implies. It does **not** rescue a *missing* `EnvironmentFile` (§4.1), and Grafana
has no equivalent, so the split below stands as decided.

Consequence, to be stated in the release notes: **existing BETA1 hosts keep a known database
password.** It is no longer world-readable, but it is still `pmm-managed`. Operators who want
it rotated must reinstall or rotate by hand. This is a documented gap, not a silent one.

### 4.2 The relocation inference expires — the generation marker

**Added after the whole-branch review.** Both copies of the relocation logic infer *"this host
predates the secret store"* from *"`/srv` is provisioned AND the secret file is absent"*. That
inference is sound exactly once. After this release ships, every host it provisions is
provisioned, so the same condition now also describes a host whose secret store was **lost** —
and `/srv/.pfw-secrets` is easy to lose: `0700` and hidden, so `cp -r /srv/*`, a `tar` or
`rsync` without `.[!.]*`, and most operator cleanups carry the visible contents of `/srv`
(including the PostgreSQL cluster) and silently drop it.

Writing the legacy literal in that case is worse than a broken connection. `initWithRoot`
self-heals a mismatch by running `ALTER USER … WITH PASSWORD '<configured>'` as the superuser,
so it would *rotate a generated credential back to the shipped public constant* and report
success. Grafana, with no such path, would wedge permanently.

The fix is a marker, `/srv/pfw-secrets-generated`:

- **Location.** Outside `/srv/.pfw-secrets` and **not hidden**. A marker inside the directory
  disappears with the thing it describes; a dotfile beside it disappears for the very same
  reason the directory does. A plain file at the top of `/srv` travels with `/srv/postgres18`,
  which is what makes the mismatch — cluster present, secrets gone — observable at all. It is
  deliberately *not* a line in `/srv/pmm-distribution`: `managed/utils/distribution` parses that
  file's contents.
- **Derivation, not a side effect.** `sync_generated_marker()` writes it at the end of every
  `ensure_secrets()` run, based on whether the values in the files differ from the two legacy
  constants — never as a side effect of the generating branch. A crash between the write and the
  marker, or a marker deleted by hand, therefore self-heals on the next boot, and the marker can
  never claim a credential that was never written. An operator who rotates a legacy host's
  passwords by hand and updates the files gets the protection for free, which is correct.
- **Semantics.** Marker present + a secret file missing ⇒ the store was LOST ⇒ `pfw-init.sh`
  fails loudly with recovery instructions, and `%post` prints a warning and writes nothing.
  Marker absent + provisioned ⇒ the genuine legacy case, unchanged.
- **Whole-`/srv` loss** needs no special case: `/srv/pmm-distribution` is gone too, so the host
  reads as a fresh install and generates, which is right.

---

## 5. Design

### 5.1 Secret store

A single directory, `${SRV}/.pfw-secrets`, mode `0700 pfw:pfw`, created by `pfw-init.sh`. It
is runtime state, not packaged content, so it needs no `%files` entry — `/srv` is already
created `0770 pfw:pfw` by `%post` (spec line 318), and `pfw-init.service` runs `UMask=0077`
(line 39), so everything below is owner-only from the first syscall.

Two files, one per consuming unit, each `0600`:

```
/srv/.pfw-secrets/managed-db.env     PMM_POSTGRES_DBPASSWORD=<value>
/srv/.pfw-secrets/grafana.env        GF_DATABASE_PASSWORD=<value>
                                     GF_SECURITY_ADMIN_PASSWORD=<value>   # fresh installs only
```

One file per consumer rather than one shared file: `EnvironmentFile=` has no way to scope
keys, so a shared file would inject every key into every unit's environment. **This is
organisational, not enforcement** — both units run `User=pfw` and the directory is `0700
pfw:pfw`, so either process can read the other's file; `ProtectSystem=strict` restricts writes,
not reads. Actual isolation would need systemd's `LoadCredential=`.

`systemd` reads `EnvironmentFile=` as PID 1, before the unit's sandbox applies, so `0600
pfw:pfw` is readable by systemd and by nothing else unprivileged.

**Value format.** `openssl rand -hex 16` — 128 bits as `[0-9a-f]{32}`. Hex is chosen over
base64 because the value has to survive three parsers unescaped: `provision_app_db` rejects
anything outside `^[A-Za-z0-9_-]+$` (`pfw-init.sh:87`), systemd's line-oriented
`EnvironmentFile` parser treats a trailing `\` as a line continuation, and the value is
interpolated into a SQL string literal. Hex is safe in all three with no quoting.

**Read-back.** Values are `[0-9a-f]{32}` or the legacy literals `pmm-managed` / `grafana`,
none of which need quoting, so `sed -n 's/^KEY=//p'` is sufficient and avoids `source`-ing
attacker-influenced content.

### 5.2 `pfw-init.sh`

Add `ensure_secrets()`, called from `main()` **before** `provision_srv`, because
`provision_databases` needs the generated values and because it must run on upgraded hosts
where `provision_srv` returns early at its sentinel guard (line 132).

```
ensure_secrets()
  install -d -m 0700 "${SECRETS_DIR}"
  for each secret file that does not exist:
      if srv_provisioned; then  value=<legacy literal>     # relocate, do not rotate
      else                      value=$(openssl rand -hex 16)
      write 0600
  log the path of the admin credential (NOT its value) when freshly generated
```

`provision_app_db` calls at lines 119 and 126 change from literal passwords to the values
read back from the secret files. Its identifier and password validation stays as-is; the
generated values satisfy it.

`GF_SECURITY_ADMIN_PASSWORD` is written only on a fresh install. On an upgraded host the
admin user already exists and Grafana would ignore the key, so writing it would be
misleading.

The journal line records the *path*, not the value. Report §14 asks for the credential to be
"displayed once on first startup", but a headless RPM install has no console to display it
on: the only "display" is journald, which persists it to `/var/log/journal` for anything in
`adm` or `systemd-journal` to read — reintroducing a weaker form of the disclosure this
change exists to close. Pointing at a 0600 file gives the operator the same access with a
tighter audience.

### 5.3 Units

`pfw-managed.service`: delete `Environment=PMM_POSTGRES_DBPASSWORD=pmm-managed` (line 60) and
add `EnvironmentFile=/srv/.pfw-secrets/managed-db.env`. Deletion is required because the line
*is* the disclosure — not because it would win. `man systemd.exec` is explicit that
*"settings from these files override settings made with `Environment=`"*, so a forgotten line
would leak the constant while the file still supplied the real value: a silent leak, not a
visible break. Add `-/srv/.pfw-secrets` to `ReadOnlyPaths` (line 47), which currently
grants `ReadWritePaths=/srv`, so a compromised pmm-managed could otherwise rewrite the
secrets it reads.

`pfw-grafana.service`: add `EnvironmentFile=/srv/.pfw-secrets/grafana.env` **after** the
existing `/run/pfw/grafana.env` line (45). Later files win, so the generated value overrides
the packaged seed and survives pmm-managed's re-render — which never emits `GF_DATABASE_*` or
`GF_SECURITY_*` (`systemd.go:157-183`; verified no `GF_DATABASE_*` or `GF_SECURITY_*` key in that span).

Neither file is optional (no leading `-`). A missing secret must fail the unit closed, matching
the rationale already written at `pfw-grafana.service:41-44` for the seed file.

### 5.4 Package content

`defaults/grafana.env:11`: `PMM_POSTGRES_DBPASSWORD` becomes empty. On a fresh install the
seed's old literal would simply be *wrong* until pmm-managed's first render replaces it
(`systemd.go:167`), and the unit already documents indefinite retry until that render lands.
Empty is strictly better than wrong, and removes the last shipped copy of the constant.

`grafana.ini:11`: `password = grafana` becomes empty **in the RPM only**.
`/etc/grafana/pfw.ini` is `%config(noreplace)`, so upgraded hosts keep the old file containing
`grafana` — consistent, because their relocated env file carries the same legacy value.

**Corrected after the whole-branch review:** the first implementation emptied the line in the
ansible source, which is NOT RPM-only. `build/ansible/roles/grafana/tasks/main.yml:22-25` copies
that same file to `/etc/grafana/grafana.ini` for the container image;
`managed/services/supervisord/supervisord.go:393` runs the container's Grafana against it, that
supervisord block exports **no** `GF_DATABASE_PASSWORD`, and
`build/ansible/roles/initialization/tasks/main.yml:76` creates the container's `grafana` role
*with* the password `grafana`. Emptying the shared source therefore left container Grafana with
no credential against a role that has one. The literal stays in the ansible file for the
container, and `pfw-server.spec`'s `%install` blanks it in the buildroot copy of `pfw.ini`
(guarded by a `grep` so a renamed or reworded line fails the build rather than silently shipping
the constant).

`pfw-server.spec`: **no permission change.** Once the unit carries no secret, `0644` is the
correct mode for a unit file; the fix is removing the secret, not hiding the unit. The spec
comment at line 346 explaining the `0640` on `pfw.ini` should be updated to say the file no
longer carries a password.

`.env.example` / `.env.dev.example` line 20: replace `admin` with an obviously-invalid
placeholder and a comment showing `openssl rand -hex 16`. These are docker-compose inputs and
not the shipped artifact; this is hygiene, not the P0 fix.

### 5.5 Operator tooling

`pfw-encryption-rotation` is shipped to `%{_sbindir}` (`pfw-managed.spec:57`) and documented
as an operator step (`documentation/docs/admin/security/data_encryption.md:48`). Its
`--postgres-password` flag carries `default:"pmm-managed"`
(`managed/cmd/pfw-encryption-rotation/main.go:90`). On a fresh install with a generated
password, running it as documented will fail to connect.

The tool reads the secret file itself when `--postgres-password` is not given, falling back
to the current default only when the file is absent. Doing it in the tool rather than in the
documentation keeps the documented invocation working unchanged, which matters because the
alternative is asking operators to paste a `sed` expression into a command they run while
rotating encryption keys.

That documentation page also names the binary `pmm-encryption-rotation`, which the RPM does
not ship — the same defect class as `668772c1c`. Fixed in passing.

---

## 6. Testing

### 6.1 The suites currently assert the bug

`build/scripts/pfw-test-lib:119` defaults `SRV` to
`--server-url=https://admin:admin@127.0.0.1:8443`, and `test-pfw-airgap:61,68`,
`test-pfw-upgrade:80` and `test-pfw-negative-control:259` all curl with `-u admin:admin`.

Only the **fresh-install** callers change. `test-pfw-airgap` installs from scratch, so its
two sites and the shared `SRV` must read the generated credential out of
`/srv/.pfw-secrets/grafana.env` — and an airgap run that still passes with `admin:admin` is
proof the fix did not land. `test-pfw-upgrade:80` and `test-pfw-negative-control:259` run
against an *upgraded* host, where relocation deliberately leaves the existing Grafana admin
user alone.

**Corrected after the whole-branch review:** that is true of a real legacy host but NOT of this
harness. `build_previous_release` builds from current sources, so `install_previous_release` is
a genuine first boot that *generates* an admin password; `downgrade_to_legacy_credential_shape`
must therefore also reset the admin user to `admin` for `admin:admin` to be correct. Without
that reset the upgrade suite 401s and exits before performing the upgrade at all. (Line 259
sits inside `phase_upgrade`, line 185, not `phase_airgap`, line 393.)

### 6.2 New assertions

Added to `test-pfw-airgap` (fresh install):

- `/srv/.pfw-secrets` is `0700 pfw:pfw`; both `.env` files are `0600`
- no unit under `/usr/lib/systemd/system/pfw-*.service` matches `PMM_POSTGRES_DBPASSWORD=`
- the provisioned password is not `pmm-managed`, and not `grafana` for the grafana role
- **negative control:** `psql` as `pmm-managed`/`pmm-managed` is *rejected*
- `pfw-managed.service` and `pfw-grafana.service` reach `active`
- login with `admin`/`admin` is rejected; login with the generated password succeeds

Added to `test-pfw-upgrade` (relocation path), which already builds and installs a previous
release (`build_previous_release`, line 119):

- after upgrade the secret files exist and contain the legacy literals
- the unit no longer carries the literal
- both services are still `active` and the API still answers
- **the §4.1 hole specifically:** upgrade, then `systemctl restart pfw-managed` *without*
  restarting `pfw-init` or rebooting, and assert the service reaches `active`. This is the
  scenario the `%post` relocation exists for; without it the restart fails on a missing
  `EnvironmentFile`, and it is the one assertion that would have caught the original design.

### 6.3 What the container cannot prove

Per prior sessions: SELinux transitions and fcaps are inert under Docker. The container suite
*can* prove file modes, `EnvironmentFile` wiring, unit start, and the upgrade relocation —
which is most of this change. It cannot prove that `/srv/.pfw-secrets` gets a usable SELinux
label under enforcing. Note that `%post` (spec line 308) claims *"pfw-init restorecons what it
creates itself"*, but `pfw-init.sh` contains no `restorecon` call — the new directory relies on
label inheritance from `/srv`. **This must be checked on the native VM**, with `setenforce 0`
then `semodule -DB` first, because `dontaudit` rules suppress the denials.

**Corrected after the whole-branch review:** the upgrade path had a second, concrete gap here.
The relocation block sat *below* `%post`'s own `restorecon -R /srv`, so the directory and files
it creates were made after the only relabel in the transaction and were never labelled by the
call that exists for exactly that. The block now runs first in `%post`.

Two things reduce the risk. `EnvironmentFile=` is read by PID 1, not by the confined service
(`man systemd.exec`: *"read from the file system of the service manager, before any file
system changes like bind mounts take place"*), so the service's own domain never opens the
file. And if the label is wrong, the remedy already exists in-tree: `pfw-nginx.service:63`
carries `ExecStartPre=-+/usr/sbin/restorecon -R /srv/nginx`, where `+` escapes the sandbox to
run as root and `-` keeps it non-fatal on a host without `restorecon`. The same line pointed at
`/srv/.pfw-secrets` is the fix if the VM run needs one.

### 6.4 Negative control

Per project practice, each new assertion is verified by deliberately breaking the fix and
confirming the assertion goes red. A green suite here has produced a false pass before.

---

## 7. Risks

| Risk | Mitigation |
|---|---|
| First boot breaks — worse than the disclosure it fixes | `ensure_secrets` touches no database and starts no daemon; it only writes files. Full airgap suite plus native VM before merge. |
| Upgraded host left without a secret file → `pfw-managed` dead | Relocation runs outside the sentinel guard, on every boot, before `provision_srv`. Covered by `test-pfw-upgrade`. |
| Grafana's PMM plugin misbehaves with an empty seed password until the first render | Unknown consumer of `PMM_POSTGRES_*` in `grafana.env`; the unit already documents retry-until-render. Verify on the VM. |
| `GF_DATABASE_PASSWORD` is not honoured by the Grafana fork | Standard `GF_<SECTION>_<KEY>` override; high confidence but **unverified against `pfw-grafana`** — confirm before merge. |
| New `/srv/.pfw-secrets` mislabelled under SELinux enforcing | §6.3; native VM check. |
| A present-but-empty secret file silently falls back to the old constant | `pfw-managed/main.go:711` sets `Default("pmm-managed")` on the flag, so an unset env var resolves to the constant rather than erroring. A *missing* file still fails the unit closed (§5.3); only a truncated one degrades quietly. Asserted against directly in §6.2. |

---

## 8. Adjacent defects found during review

Pre-existing, not introduced here, but all three sit on the code path this change starts
feeding *generated* values through. **All three are in scope for this branch.** Shipping a
design that routes freshly generated passwords into an unescaped SQL interpolation while
labelling it "pre-existing" would be indefensible. The cost is honest and worth stating: the
first two are Go changes, so this branch now also needs the `managed/` suite and a live
PostgreSQL on `127.0.0.1:5432`, which the shell-only change did not.

**CRITICAL — password interpolated into SQL without escaping.** `database.go:1426` builds
`CREATE USER "%s" LOGIN PASSWORD '%s'` and `:1441` builds `ALTER USER "%s" WITH PASSWORD '%s'`
via `fmt.Sprintf`. A password containing a single quote closes the literal and injects SQL as
the *superuser*. `pfw-init.sh:87` already defends against precisely this in the shell path, and
its comment explains why quoting alone is insufficient; the Go path has no such guard. The
generated hex is safe, but the value is operator-settable via `PMM_POSTGRES_DBPASSWORD`.

**HIGH — `GRANT` uses bind parameters for identifiers.** `database.go:1432` runs
`GRANT ALL PRIVILEGES ON DATABASE $1 TO $2` with arguments. PostgreSQL does not accept
placeholders for identifiers in DDL, so this fails with a syntax error whenever reached, and
`SetupDB` returns the error rather than tolerating it (`database.go:1285`). It is latent only
because the branch requires the role to be absent and `pfw-init.sh:126` pre-creates it on
native. Any host where pre-creation is skipped or fails gets a hard start failure here.

**LOW — documented binary does not exist.** `data_encryption.md:48` says
`pmm-encryption-rotation`; the RPM ships `pfw-encryption-rotation` (`pfw-managed.spec:57`).
Same class as `668772c1c`.

---

## 9. Files changed

| File | Change |
|---|---|
| `build/packages/config/pfw/pfw-init.sh` | `ensure_secrets()`; generated values passed to `provision_app_db` |
| `build/packages/config/pfw/pfw-managed.service` | drop `Environment=`, add `EnvironmentFile=`, extend `ReadOnlyPaths` |
| `build/packages/config/pfw/pfw-grafana.service` | add second `EnvironmentFile=` |
| `build/packages/config/pfw/defaults/grafana.env` | empty the `PMM_POSTGRES_DBPASSWORD` seed |
| `build/ansible/roles/grafana/files/grafana.ini` | keeps `password = grafana` for the container; `pfw-server.spec` `%install` blanks the RPM's copy |
| `build/packages/rpm/server/SPECS/pfw-server.spec` | `%post` upgrade relocation (§4.1); comment correction |
| `managed/utils/dbsecret` | shared resolution of the `pmm-managed` password from the secret store |
| `managed/cmd/pfw-encryption-rotation/main.go` | read the secret file when the flag is unset |
| `managed/cmd/pfw-managed/main.go` | resolve a set-but-EMPTY `PMM_POSTGRES_DBPASSWORD` from the store instead of kingpin's `Default` |
| `build/scripts/pfw-test-lib` + 3 suites | stop asserting `admin:admin`; new assertions |
| `.env.example`, `.env.dev.example` | placeholder instead of `admin` |
| `documentation/.../data_encryption.md` | correct the binary name |
| `documentation/docs/release-notes/…` | upgraded hosts retain their password |
| `docs/p0-blockers-handoff.md` | record P0.3c, the §2 corrections, and §8 |
| `managed/models/database.go` | §8 CRITICAL and HIGH |

---

## 10. Acceptance

1. No shipped file contains a usable database or admin password for a fresh install.
2. No world-readable file contains a database password on any host, fresh or upgraded.
3. A fresh install rejects `admin`/`admin` and rejects `pmm-managed`/`pmm-managed`.
4. An upgraded install still starts, with its credentials relocated to 0600.
5. Both container suites pass, each new assertion negative-controlled.
6. The native VM run confirms SELinux labelling and `EnvironmentFile` reads under enforcing.
7. Release notes state that upgraded hosts retain their existing database password.
