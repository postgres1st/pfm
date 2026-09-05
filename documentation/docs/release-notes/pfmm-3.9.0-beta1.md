# PFMM 3.9.0 beta1 — release notes

**Postgres1st Monitoring and Management (PFMM)** — PostgreSQL monitoring and query
analytics, delivered as a signed package repository you install on your own RHEL,
Rocky Linux or AlmaLinux 9 server. No internet access is required on that server, and
no container runtime is involved.

This is a **beta**. It is complete and tested for the workflow described below, and the
limitations at the end are stated plainly rather than left to be discovered.

!!! note "Names on this page are beta1's"

    The product was renamed to **PGF WatchTower** after beta1, and its packages,
    units and service account moved from `pfm-*`/`pfm` to `pfw-*`/`pfw`. This page
    keeps beta1's names on purpose: it describes the release that shipped, and the
    commands below are the ones that work on a beta1 host. There is no upgrade path
    from beta1 to beta2 — see the beta2 notes for a fresh install.

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
3. `sudo dnf --enablerepo=pfw install pfm-server`
4. `sudo systemctl start pfm.target`
5. Open `https://<host>:8443/` and sign in as `admin`. The password is generated
   during first boot; read it with the command under *Bootstrap credentials* below.

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

## Exporter endpoint security — one breaking change

**Exporters now use a real per-agent credential.** Previously, when no password was
stored, the exporter's HTTP basic auth fell back to the agent's own ID — a value the
inventory API returns and logs contain, so anyone able to list agents could derive
any exporter's credential. Each agent is now issued 128 bits of random hex at
registration, and existing agents are given one automatically when the server starts
after upgrading. No action is required.

**BREAKING — exporters now bind `127.0.0.1` by default.** This affects only services
registered with `--metrics-mode=pull`. The default registration uses push mode, where
the agent sends metrics over the connection it already holds to the server, and those
exporters already bound loopback — so most deployments are unaffected and need no
firewall change.

If you do scrape a client host over the network, the failure is silent: metrics simply
stop arriving. Three remedies, in order of preference:

1. Re-register the service with `--expose-exporter` (per service, recommended).
2. Switch it to push mode, which needs no inbound listener at all.
3. Set `PFM_EXPOSE_EXPORTERS=true` on the **server** and restart `pfm-managed` to
   restore the previous fleet-wide behaviour. Intended for recovering a fleet that has
   already gone dark, while you re-register services; remove it afterwards.

**Two things to be aware of:**

- **Debug logging now contains real exporter credentials.** `PMM_DEBUG=1` logs the
  agent state request, and the `HTTP_AUTH` value it carries used to be the public
  agent ID. Treat debug logs from this release as secret-bearing.
- **The RDS and Azure exporters are not covered.** They still bind all interfaces and
  carry no authentication at all. A single process serves many monitored instances, so
  there is no one agent whose credential or `--expose-exporter` setting would apply;
  closing that needs a separate change. Firewall those ports if you use either.

## Known limitations

Please read these before deploying anything you depend on.

**SELinux — a policy module ships; automated enforcing validation does not.** The
package installs an SELinux policy module (`pfm_nginx`) covering nginx's TLS material
and buffers under `/srv/nginx`, and loads it on install where `semodule` is available.
The rest of `/srv` -- the PostgreSQL, ClickHouse, Grafana and VictoriaMetrics data
directories -- is deliberately left unlabelled by it. The earlier statement that no module
was shipped is out of date.

Enforcing mode has been exercised by hand on RHEL 9.8 (x86_64), and that run shaped
what ships: the nginx unit relabels `/srv/nginx` before starting because a certificate
placed with `mv` or `cp -a` keeps the wrong label and TLS then fails silently, and the
ClickHouse unit's capability bounding set was set from measured behaviour on that host.
Those are findings a container cannot produce.

What is missing is *reproducible* validation. The automated checks are structural: the
suites confirm the compiled module is in the bundle and owned by the `pfm-server`
package, and that the nginx unit declares the relabel step. None of them runs with
SELinux enforcing -- the container images have neither `semodule` nor a policy store --
so nothing verifies that the policy actually permits what the stack needs. So the manual run is not repeated per
release, and **aarch64 has not been covered by it** — the hand testing was x86_64 only.
Enforcing-mode confinement is therefore not covered by any automated gate before
general availability.

If your policy is enforcing, review denials before relying on this build, and treat
`setenforce 0` as an evaluation measure only. When checking for denials, note that
`dontaudit` rules suppress them: run `setenforce 0`, then `semodule -DB`, then
reproduce.

The systemd hardening directives and the polkit rule the server depends on are shipped
and installed, but no test asserts either: the hardening is enforced by systemd rather
than by anything the suites inspect, and the polkit rule has no authorization probe.
Treat both as configured rather than as verified.

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

**Bootstrap credentials are now generated per install.** The Grafana admin
password and the internal PostgreSQL role passwords are generated at first boot
and stored in `/srv/.pfm-secrets` (mode 0600). Read the initial admin password
with:

    sudo sed -n 's/^GF_SECURITY_ADMIN_PASSWORD=//p' /srv/.pfm-secrets/grafana.env

That command applies to **fresh installs only**. An upgraded host keeps the admin
user it already has, so the file carries no `GF_SECURITY_ADMIN_PASSWORD` key and
the command prints nothing and exits 0 — keep using the password you already have.

**Back up `/srv/.pfm-secrets`.** On a fresh install it is the only copy of two
passwords the PostgreSQL roles already hold. The directory is hidden and mode 0700,
so `cp -r /srv/*` and `tar`/`rsync` invocations that do not include `.[!.]*` skip it
silently. A marker at `/srv/pfm-secrets-generated` records that the credentials were
generated; if that marker is present and the secret files are gone, the server
refuses to start rather than writing a known constant over a generated password.
Restore the directory from a backup, or reset both roles by hand as the PostgreSQL
superuser and write the new values back into the files.

**Upgraded hosts keep their existing database passwords.** An upgrade relocates
them out of the world-readable unit file into the same 0600 store, but does not
rotate them: a host first provisioned by an earlier build still authenticates
with the previous values. Rotate them by hand, or reinstall, if that matters for
your deployment. The Grafana admin password is likewise unchanged on upgrade.

On an upgraded host, `/etc/grafana/pfm.ini` retains its existing `password = grafana`
line. The file is `%config(noreplace)`, so packaging deliberately does not overwrite it.
`GF_DATABASE_PASSWORD` from `/srv/.pfm-secrets/grafana.env` is intended to take
precedence, via Grafana's standard `GF_<SECTION>_<KEY>` override — confirming that
`pfm-grafana`, a fork, still honours this is a pending native-VM item. The file is mode
0640 `root:pfm`, so it is not world-readable either way. A fresh install ships the line
empty; on an existing host, operators who want certainty rather than relying on the
override should blank the value by hand.

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
