# Postgres1st WatchTower 3.9.0 beta2 — release notes

**Postgres1st WatchTower** — PostgreSQL monitoring and query analytics, delivered as a
signed package repository you install on your own RHEL, Rocky Linux or AlmaLinux 9
server. No internet access is required on that server, and no container runtime is
involved.

This is a **beta**. It is complete and tested for the workflow below, and the limitations
at the end are stated plainly rather than left to be discovered.

!!! warning "There is no upgrade path from beta1"

    beta1 shipped as `pfm-*` under the earlier name PFMM. Everything a package manager
    keys on changed in beta2 — package names, the systemd units, the service account and
    the install root — so `dnf upgrade` cannot carry a beta1 host forward. **Install
    beta2 fresh.** From beta2 onward the path is normal: `3.9.0~beta2` sorts below
    `3.9.0~rc` and `3.9.0`, so later releases upgrade in place.

---

## What changed since beta1

**The product is Postgres1st WatchTower.** Packages are `pfw-*`, units are
`pfw-*.service`, the service account is `pfw`, and everything installs under
`/opt/postgres1st/watchtower`. The client agent is `pfw-agent`; its configuration lives
at `/opt/postgres1st/watchtower/config/pfw-agent.yaml`.

**Every server component is now built from source.** beta1 fetched four of them prebuilt
from Percona's public build cache. Those artifacts carried Percona's branding regardless
of what our specs said, the cache rotates its keys so the pinned versions expired, and it
publishes x86_64 only. Building them ourselves fixed all three at once, and is what makes
an **aarch64 bundle** possible — beta1 could not produce one.

**The server now starts on a host with SELinux enforcing.** beta1 shipped a policy module
but was never installed on an enforcing host by an automated gate. Doing that for beta2
found a defect that would have stopped the product dead: the credential store under
`/srv` carried the generic `var_t` type, and systemd reads `EnvironmentFile=` as PID 1 —
a *confined* domain — so three services could not read their configuration and restarted
forever with `Permission denied`, with `readyz` stuck at 500. The store now has its own
type, readable by PID 1 alone, applied at first boot by a step that verifies its own
result rather than failing silently. See *Known limitations* for exactly which
architecture this was certified on.

**Zero egress by design.** The server makes no outbound requests: Grafana's news feed is
off, and the upstream blog-feed proxy and its dashboard panel are gone.

**Security fixes carried forward from the beta1 line**, all present here: bootstrap
credentials generated per install rather than a shipped constant, the internal PostgreSQL
password no longer written to a world-readable file, per-agent exporter credentials
generated on every agent-creation path, and exporters binding loopback by default.

---

## What you get

A monitoring server you install with `dnf` and start with `systemctl`, providing:

| | |
|---|---|
| **Dashboards** | provisioned Grafana dashboards for PostgreSQL and host metrics |
| **Query Analytics** | per-query statistics from `pg_stat_statements` or `pg_stat_monitor` |
| **Advisors** | PostgreSQL configuration and health checks |
| **Metrics storage** | VictoriaMetrics, 30-day retention by default |
| **Agents** | `pfw-agent` on the monitored hosts, installed from the same bundle |

The server runs as the unprivileged `pfw` account under full systemd hardening. Only
port 8443 (HTTPS) needs to be reachable.

## PostgreSQL only — by design

WatchTower monitors PostgreSQL. Adding a MySQL, MongoDB, ProxySQL or Valkey service is
refused at registration:

```
$ pfw-admin add mysql ...
Service type "mysql" is not supported by this deployment.
```

This is deliberate, controlled by `PFW_DB_TYPES`, and it extends to what the interface
offers. Upstream PMM ships advisor checks for database families this build refuses; those
can never produce a result, so they are not listed. If you are comparing against PMM and
expect to see more, that is why — nothing is disabled or broken.

## Packages

| Package | Contents |
|---|---|
| `pfw-server` | the assembly: units, nginx config, first-boot provisioning |
| `pfw-managed` | the control plane |
| `pfw-agent` | the monitoring agent, for the server and for monitored hosts |
| `pfw-qan-api2` | Query Analytics API |
| `pfw-victoriametrics` | metrics storage and vmalert |
| `pfw-grafana`, `pfw-dashboards` | the UI and its provisioned dashboards |
| `pfw-dump`, `vmproxy` | support-bundle export, metrics proxy |

## Installing

See `INSTALL.md` inside the bundle. In outline:

1. Unpack the tarball and import the signing key
2. Point `dnf` at the bundled repository
3. `sudo dnf --enablerepo=pfw install pfw-server`
4. `sudo systemctl start pfw.target`
5. Open `https://<host>:8443/` and sign in as `admin`. The password is generated during
   first boot; read it with the command under *Bootstrap credentials* below.

The bundle carries WatchTower, PostgreSQL 18 and ClickHouse: everything an EL9 repository
does not provide. Your distribution's own packages — `nginx`, `perl`, `polkit`,
`openssl` — come from your normal repositories, as they would for anything else you
install. If the server has no repository of its own, `fetch-os-dependencies.sh` collects
them on a connected machine for you to copy across.

## Verifying what you received

Every package is signed by Postgres1st, including the third-party ones, so a single
`rpm --import` covers the whole set and `gpgcheck=1` applies throughout. Check the
tarball's `.sha256` as well: the checksum proves the download arrived intact, the
signature proves who built it.

## Upgrading

Upgrades from beta2 onward are ordinary `dnf` upgrades against a newer bundle:

```bash
sudo dnf --enablerepo=pfw upgrade 'pfw-*' vmproxy
```

`/srv` is left alone, so metrics history, dashboards and the Grafana database survive.
Dashboards refresh without a restart; other components need one. `INSTALL.md` has the
detail. There is no path from beta1 — see the warning at the top.

---

## Known limitations

Please read these before deploying anything you depend on.

**SELinux — certified on x86_64, not on aarch64.** The bundle installs a policy module
(`pfw_nginx`) covering nginx's TLS material under `/srv/nginx` and the credential store
under `/srv/.pfw-secrets`, and loads it on install where `semodule` is available.

For beta2 the x86_64 bundle was installed on a clean RHEL 9.8 host with SELinux
**enforcing**, by the documented procedure with `gpgcheck=1`: the server reached ready in
about ten seconds with no manual intervention, no service restarted even once, and there
were no SELinux denials — checked with `dontaudit` rules disabled as well as enabled,
since `dontaudit` suppresses exactly the denials worth seeing. A cold reboot was also
exercised: the stack came back automatically with its labels intact.

**The aarch64 bundle has not had that test.** The fix is architecture-independent policy,
so it should behave identically, but "should" is not "was measured", and this note will
not claim otherwise. If you deploy aarch64 with SELinux enforcing, review denials before
relying on it. When checking, run `setenforce 0`, then `semodule -DB`, then reproduce —
otherwise `dontaudit` will hide the denials from you.

The systemd hardening directives and the polkit rule the server depends on are shipped
and installed, but no test asserts either. Treat both as configured rather than verified.

**Scale is not characterised.** No fleet-scale figure is published for this release: the
number of monitored instances a server sustains has not been measured. Size from your own
trial rather than from a claim here.

**Low-disk, low-memory and interrupted-transaction behaviour are untested.** The server
has not been exercised against a full filesystem, memory pressure, or an install
interrupted part-way.

**Container and Helm deployment are not part of this release.** The bundle installs
natively; the Docker path is not rebranded and is not tested here.

**Architecture.** A bundle installs only on the architecture it was built for. The other
architecture is a separate download.

**Upgrades across large gaps.** Upgrading has been tested between closely spaced builds.
A jump across several releases is untested. PostgreSQL major versions do not upgrade in
place — the data directory carries the major version, and a bundle built against a newer
major will stop rather than start against an existing cluster.

**Storage sizing.** The guidance of roughly 1 GB per monitored instance per month at
30-day retention is reasoned from the retention settings, not measured across a fleet.
Watch `/srv` on your first instances.

**Monitoring account.** `pg_monitor` is documented as sufficient, against upstream's
advice to use a superuser. Not every collector has been exercised with only that role.

**Exporter endpoints.** Exporters bind `127.0.0.1` by default and carry a per-agent
credential generated at registration. Two caveats:

- Exposing one deliberately — `--expose-exporter` per service, or `PFW_EXPOSE_EXPORTERS`
  fleet-wide — publishes it on all interfaces, and these endpoints carry **no TLS**, so
  the credential travels in cleartext. Firewall them to the WatchTower server.
- **The RDS and Azure exporters are not covered by either protection.** They bind all
  interfaces and carry no authentication at all: one process serves many monitored
  instances, so no single agent's credential or `--expose-exporter` setting applies.
  Closing that needs a separate change. Firewall those ports if you use either.

**Bootstrap credentials are generated per install.** The Grafana admin password and the
internal PostgreSQL role passwords are generated at first boot and stored in
`/srv/.pfw-secrets` (directory mode 0700, files 0600). Read the initial admin password
with:

    sudo sed -n 's/^GF_SECURITY_ADMIN_PASSWORD=//p' /srv/.pfw-secrets/grafana.env

**Back up `/srv/.pfw-secrets`.** On a fresh install it is the only copy of passwords the
PostgreSQL roles already hold. The directory is hidden and mode 0700, so `cp -r /srv/*`
and `tar`/`rsync` invocations that do not include `.[!.]*` skip it silently. A marker at
`/srv/pfw-secrets-generated` records that the credentials were generated; if that marker
is present and the secret files are gone, the server refuses to start rather than writing
a known constant over a generated password. Restore the directory from a backup, or reset
the roles by hand as the PostgreSQL superuser and write the new values back.

**Signing key.** This beta is signed with a key whose custody is not yet finalised. The
key used for the general-availability release may differ; its release notes will say so,
and you would import the new key at that point.

---

## Licensing

WatchTower is built on Percona Monitoring and Management and Grafana, and is distributed
under the **GNU Affero General Public License, version 3**. Everything Postgres1st builds
is AGPLv3, as is the Grafana it embeds. The remainder are Apache 2.0 (ClickHouse,
VictoriaMetrics) and the PostgreSQL License (PostgreSQL). Each package declares its own
licence — `rpm -qi <package>` reports it.

The AGPL entitles you to the corresponding source for the AGPL components. Contact
Postgres1st to obtain it.

## Support

Report problems to Postgres1st with the output of:

```bash
curl -sk -o /dev/null -w '%{http_code}\n' https://127.0.0.1:8443/v1/readyz
systemctl --failed
journalctl -u pfw-managed -n 200
```
