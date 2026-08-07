# PFMM 3.9.0 beta1 — release notes

**Postgres1st Monitoring and Management (PFMM)** — PostgreSQL monitoring and query
analytics, delivered as a signed package repository you install on your own RHEL,
Rocky Linux or AlmaLinux 9 server. No internet access is required on that server, and
no container runtime is involved.

This is a **beta**. It is complete and tested for the workflow described below, and the
limitations at the end are stated plainly rather than left to be discovered.

---

## What you get

A monitoring server you install with `dnf` and start with `systemctl`, providing:

| | |
|---|---|
| **Dashboards** | provisioned Grafana dashboards for PostgreSQL and host metrics |
| **Query Analytics** | per-query statistics from `pg_stat_statements` or `pg_stat_monitor` |
| **Advisors** | PostgreSQL configuration and health checks |
| **Metrics storage** | VictoriaMetrics, 30-day retention by default |
| **Agents** | `pfm-client` for the monitored hosts, installed from the same bundle |

The server runs as the unprivileged `pfm` account under full systemd hardening. Only
port 8443 (HTTPS) needs to be reachable.

## PostgreSQL only — by design

PFMM monitors PostgreSQL. Adding a MySQL or MongoDB service is refused:

```
$ pfm-admin add mysql ...
Service type "mysql" is not supported by this deployment.
```

This is deliberate, and it extends to what the interface offers. Upstream PMM ships
advisor checks for MySQL and MongoDB as well as PostgreSQL; the ones that target databases
this build refuses can never produce a result, so they are not listed. If you are comparing
against PMM and expect to see more checks, that is why — nothing is disabled or broken.

## Installing

See `INSTALL.md` inside the bundle. In outline:

1. Unpack the tarball and import the signing key
2. Point `dnf` at the bundled repository
3. `sudo dnf --enablerepo=pfmm install pfm-server`
4. `sudo systemctl start pfm.target`
5. Open `https://<host>:8443/` (default `admin` / `admin` — change it)

The bundle carries PFMM, PostgreSQL 18 and ClickHouse: everything an EL9 repository does
not provide. Your distribution's own packages — `nginx`, `perl`, `polkit`, `openssl` —
come from your normal repositories, as they would for anything else you install. If the
server has no repository of its own, `fetch-os-dependencies.sh` collects them on a
connected machine for you to copy across.

## Verifying what you received

Every package is signed by Postgres1st, including the third-party ones, so a single
`rpm --import` covers the whole set and `gpgcheck=1` applies throughout. Check the
tarball's `.sha256` as well: the checksum proves the download arrived intact, the
signature proves who built it.

## Upgrading

Upgrades are ordinary `dnf` upgrades against a newer bundle. `/srv` is left alone, so
metrics history, dashboards and the Grafana database survive. Dashboards refresh without
a restart; other components need one. `INSTALL.md` has the detail.

Versions use a tilde — `3.9.0~beta1` — which `rpm` treats as a pre-release. It sorts
below a plain `3.9.0`, so upgrading from this beta to the final release works normally.

---

## Known limitations

Please read these before deploying anything you depend on.

**SELinux — validation planned before general availability.** This beta has not been
validated on a host with SELinux in enforcing mode, and we do not yet ship an SELinux
policy module. If your policy is enforcing, expect to review denials, and treat
`setenforce 0` as a temporary measure for evaluation only. The systemd hardening and the
polkit rule the server depends on are in place and tested; SELinux confinement
specifically is what remains, and validating it is planned work for the
general-availability release.

**Architecture.** This bundle installs only on the architecture it was built for. A
bundle for another architecture is a separate download.

**Upgrades across large gaps.** Upgrading has been tested between closely spaced builds.
A jump across several releases is untested. PostgreSQL major versions do not upgrade in
place — the data directory carries the major version, and a bundle built against a newer
major will stop rather than start against an existing cluster.

**Storage sizing.** The guidance of roughly 1 GB per monitored instance per month at
30-day retention is reasoned from the retention settings, not measured across a fleet.
Watch `/srv` on your first instances.

**Monitoring account.** `pg_monitor` is documented as sufficient, against upstream's
advice to use a superuser. Not every collector has been exercised with only that role.

**Exporter ports.** Metric exporters bind all interfaces on 42000-42010 and are
unauthenticated — this is upstream behaviour, on every monitored host, not just the
server. They expose database and OS statistics. Firewall them to the PFMM server;
`INSTALL.md` shows how.

**Default credentials.** The server ships with `admin` / `admin`. Change it before the
host is reachable by anyone else.

**Signing key.** This beta is signed with a key whose custody is not yet finalised. The
key used for the general-availability release may differ; its release notes will say so,
and you would import the new key at that point.

---

## Licensing

PFMM is built on Percona Monitoring and Management and Grafana, and is distributed under
the **GNU Affero General Public License, version 3**. Everything Postgres1st builds is
AGPLv3, as is the Grafana it embeds. The remainder are Apache 2.0 (ClickHouse,
VictoriaMetrics) and the PostgreSQL License (PostgreSQL). Each package declares its own
licence — `rpm -qi <package>` reports it.

The AGPL entitles you to the corresponding source for the AGPL components. Contact
Postgres1st to obtain it.

## Support

Report problems to Postgres1st with the output of:

```bash
curl -sk -o /dev/null -w '%{http_code}\n' https://127.0.0.1:8443/v1/readyz
systemctl --failed
journalctl -u pfm-managed -n 200
```
