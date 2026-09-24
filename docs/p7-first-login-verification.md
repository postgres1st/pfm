# P7 — Install and first-login validation

Roadmap outcome (as written): "Correct first-login instructions and validate
installation, initial authentication, service startup, credential storage and
agent registration on each supported platform."

Shares its platform matrix with P17 (`docs/p17-native-platform-certification.md`)
— both require "on each supported platform," so the same native AWS instances
cover both, just with different checks run against them.

---

## Supported platforms

Same matrix as P17. Only RHEL 9 has been touched at all; Rocky and Alma are
untested for P7 exactly as they are for P17.

| OS | Arch | P17 platform status | P7 testing status |
|---|---|---|---|
| RHEL 9 | x86_64 | ✅ PASS (native, SELinux enforcing) | ✅ **PASS — all 6 outcome steps** |
| RHEL 9 | aarch64 | Install done; full P17 checklist not yet finished | ✅ **PASS — all 6 outcome steps** |
| Rocky Linux 9 | x86_64 | Not yet run | Not yet run |
| Rocky Linux 9 | aarch64 | Not yet run | Not yet run |
| AlmaLinux 9 | x86_64 | Not yet run | Not yet run |
| AlmaLinux 9 | aarch64 | Not yet run | Not yet run |

---

## Verification steps — run per platform combination

The six commands/checks behind each row in the results table below. Assumes a
freshly installed server per `build/packages/pfw-airgap-INSTALL.md` (and, for
step 6, a second instance with `pfw-agent` installed per `INSTALL-CLIENT.md`).

### 1. Correct first-login instructions

Follow `build/packages/pfw-airgap-INSTALL.md` step 7 **exactly as written**,
verbatim — not from memory:
```bash
sudo sed -n 's/^GF_SECURITY_ADMIN_PASSWORD=//p' /srv/.pfw-secrets/grafana.env
```
**Pass**: a real password prints, no missing command, no need to jump to another
section to find it.

### 2. Validate installation

```bash
sha256sum -c pfw-server-el9-${ARCH}.tar.gz.sha256
gpg --show-keys --with-fingerprint pfw-repo/RPM-GPG-KEY-postgres1st   # matches 5F62 9B10 E9EA 318F 0D10 42E9 20E4 C7CB 9663 03F6
sudo dnf --enablerepo=pfw install -y pfw-server
rpm -qa | grep ^pfw-           # every expected package landed
```
**Pass**: checksum OK, fingerprint matches, install completes with no errors.

### 3. Initial authentication

```bash
curl -sk -o /dev/null -w '%{http_code}\n' https://127.0.0.1:8443/v1/readyz   # expect 200
```
Then actually log into the UI (`https://<host>:8443/`) with `admin` + the
password from step 1.
**Pass**: login succeeds first try, no password mismatch, no "user does not
exist."

### 4. Service startup

```bash
systemctl list-units 'pfw*' --all      # every unit active
systemctl is-enabled pfw.target        # enabled, survives reboot
journalctl -u pfw-managed -u pfw-grafana -u pfw-clickhouse --no-pager | grep -i "error\|fatal"
```
**Pass**: nothing failed. Note from experience: this grep throws false
positives (Grafana migration log lines whose *column names* contain the word
"error") — read every hit, don't just count them. See bugs #3-#5 for what
this actually surfaced.

### 5. Credential storage

```bash
stat -c '%a %U:%G' /srv/.pfw-secrets              # expect 700 pfw:pfw
stat -c '%a %U:%G' /srv/.pfw-secrets/grafana.env  # expect 600 pfw:pfw
sudo -u nobody cat /srv/.pfw-secrets/grafana.env  # expect Permission denied
```
**Pass**: correct permissions, and confirmed *unreadable* by a non-owner, not
just documented as such.

### 6. Agent registration

Needs a **second** instance with `pfw-agent` installed (`INSTALL-CLIENT.md`).
Before registering, ensure `pg_hba.conf` uses password auth, not the EL9
default `ident` (see gotcha #6 below) — check first:
```bash
grep -n "127.0.0.1\|::1" /var/lib/pgsql/data/pg_hba.conf
```
Then, on the client:
```bash
sudo pfw-admin config --server-url=https://admin:<password>@<server-host>:8443 \
     --server-insecure-tls <this-host-address> generic <node-name>

echo -n '<pg-password>' | sudo pfw-admin add postgresql --host=127.0.0.1 --port=5432 \
     --username=pfw --password-stdin \
     --environment=production --cluster=<cluster-name> <service-name>
```
Back on the server:
```bash
pfw-admin inventory list services --server-url=https://admin:<password>@127.0.0.1:8443 --server-insecure-tls
```
**Pass**: node + service both appear in server inventory, metrics start
flowing within about a minute.

---

## Results per outcome requirement

Being precise about what was actually *confirmed with command output* versus
*implied but not directly verified* — the two are not the same thing.

| Requirement | RHEL9/x86_64 | RHEL9/aarch64 | Evidence / notes |
|---|---|---|---|
| Correct first-login instructions | ✅ Fixed | ✅ Confirmed (user-reported) | Bug found and fixed in `build/packages/pfw-airgap-INSTALL.md` step 7 (was missing the password-read command entirely) — merged PR #73. |
| Validate installation | ✅ Confirmed | ✅ Confirmed (user-reported) | aarch64: full install completed, including the wrong-architecture rejection angle exercised along the way. |
| Initial authentication | ✅ Confirmed | ✅ Confirmed (user-reported) | Password retrieved via the fixed documented command; login confirmed working. |
| Service startup | ✅ Confirmed | ✅ Confirmed (user-reported), with known non-fatal bugs | `journalctl` checked on aarch64 — found bugs #3 and #4 below, both assessed as non-fatal (server runs fine despite them). Bug #5 (possible benign first-boot races) was never explicitly re-confirmed as resolved. |
| Credential storage | ✅ Confirmed (user-reported) | ✅ Confirmed (user-reported) | |
| Agent registration | ✅ Confirmed (user-reported) | ✅ Confirmed (user-reported), after debugging | Gotchas #6 and #7 (`pg_hba.conf` ident default, the `\password`-vs-`-c` password-reset issue) were hit and resolved on aarch64; not separately noted as encountered on x86_64, but likely arch-independent — worth assuming they'd recur there too under the same quick-test setup. |

---

## Bugs found during this work

| # | Bug | Found on | Severity | Status |
|---|---|---|---|---|
| 1 | `INSTALL.md` step 7's first-login password instruction was broken — the actual read command was missing entirely, only a dangling sentence remained | RHEL9/x86_64 | Customer-facing docs, high (blocks a first-time installer) | **Fixed, merged** — PR #73 |
| 2 | `grafana.spec`'s `full_pmm_version` placeholder was never substituted by `build-pfw-airgap` (unlike the other three specs with the same pattern) — shipped the literal `3.0.0` forever | Static analysis during P5 guard work | Build/version-identity, low (cosmetic — Grafana's actual package `Version:` is unaffected, only an internal build-provenance string) | **Fixed on branch, PR closed unmerged** — fix exists on `fix/p5-release-claims-version-guards`, not in `main`. P5's PR was closed because it didn't address P5's actual GA-alignment outcome; this specific fix was never cherry-picked out separately. |
| 3 | `pfw-server.spec`'s Grafana-provisioning-directory loop creates `datasources`, `dashboards`, `plugins` but not `alerting` — Grafana logs a `level=error` on every first boot trying to read a directory that doesn't exist | RHEL9/aarch64, this session | Packaging, low (non-fatal — server boots and runs fine, just an unnecessary error-level log line) | **Found, not fixed** — offered to fix, awaiting go-ahead |
| 4 | `PostgreSQL_Instance_Summary.json` has a ClickHouse-datasource panel (`select round($uptime, 2)`) with no guard against its hidden `$uptime` template variable being empty — on a fresh install, before `postgres_exporter` completes its first scrape, this renders as `select round(, 2)` and ClickHouse rejects it with `SYNTAX_ERROR` | RHEL9/aarch64, this session | Dashboard, low (one panel fails silently on fresh installs, no server-wide impact) | **Found, not fixed** — offered to fix, awaiting go-ahead |
| 5 | (Unconfirmed) `pfw-managed` logged `FailedPrecondition: pmm-agent ... is not currently connected`, and ClickHouse logged `UNKNOWN_DATABASE: pmm does not exist` — both at the same first-boot timestamp | RHEL9/aarch64, this session | Unknown — likely benign startup-ordering race | **Needs confirmation** — command given to check whether it recurs after boot settles (`journalctl --since "5 minutes ago"`); result never reported back |
| 6 | **Testing-setup gotcha, not a WatchTower bug.** `postgresql-setup --initdb` on EL9 defaults `pg_hba.conf`'s `host` entries for `127.0.0.1`/`::1` to `ident` auth, not password auth — so `pfw-admin add postgresql --username=... --password=...` fails with `Ident authentication failed` regardless of correct credentials, on any freshly-initialized test Postgres. Hit independently by two different people following the same quick-test setup instructions. | RHEL9/aarch64 (this session) and a teammate's separate AWS instance | Docs/guidance gap in the quick-test setup steps given during this work, not the product | **Root-caused and fixed on-host** — change `ident` to `scram-sha-256` for the relevant `pg_hba.conf` lines, restart `postgresql`. Worth folding into whatever official "quick test Postgres" guidance exists so the next person doesn't rediscover it. |
| 7 | Password reset via `psql -c "ALTER USER pfw WITH PASSWORD '...'"` appeared to succeed (`ALTER ROLE` printed) but the password still didn't authenticate — resolved by resetting it interactively via `\password` instead. Root cause never fully confirmed (leading theory: a smart-quote substitution during copy-paste of the single-quoted password into the terminal, corrupting the literal string passed to SQL). | RHEL9/aarch64, this session | Unconfirmed root cause — likely a copy-paste/terminal artifact, not a real bug | **Worked around, not root-caused** — resetting passwords interactively (`\password`) rather than via a pasted `-c` string avoided it. Worth using `\password` as the default recommendation going forward regardless. |

---

## Next steps

- **RHEL 9 is done for both architectures now** (x86_64 from the earlier P17 session, aarch64 confirmed this session, all 6 outcome steps). Rocky Linux 9 and AlmaLinux 9 — both architectures — remain completely untested for P7, same as they are for P17.
- Fix bugs #3 and #4 (both scoped, both low-risk, still open).
- Decide on bug #2: cherry-pick the grafana fix out of the closed P5 branch into its own PR, since it's independent of P5's GA-alignment blocker.
- Confirm bug #5 one way or the other — still unconfirmed.
- Fold gotchas #6 and #7 into the client-install testing guidance so the next person (or the next platform combination) doesn't rediscover them from scratch.
