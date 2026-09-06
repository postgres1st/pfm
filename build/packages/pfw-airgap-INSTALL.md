# Postgres1st WatchTower — air-gapped installation (RHEL / Rocky Linux 9, @ARCH@)

Postgres1st WatchTower, delivered as a yum repository.
**No internet access is required on any host.**

**This document covers the monitoring server.** To monitor a PostgreSQL host, install the
`pfw-agent` agent on that host following **`INSTALL-CLIENT.md`**, which ships alongside
this file.

One bundle serves both: `pfw-server-el9-@ARCH@.tar.gz` contains the server and the agent.
The filename says `server` because that is the larger part of it; there is no separate
client download.

The bundle carries Postgres1st WatchTower itself, PostgreSQL (PGDG) and ClickHouse — none of which your
distribution provides. It does **not** carry your distribution's own packages: `nginx`,
`perl`, `polkit` and the rest are the OS vendor's to supply, and a current EL9 host
already has them. If the server cannot reach your OS repositories, step 4 fetches them
for you from a connected machine.

## Requirements

| | |
|---|---|
| Architecture | `@ARCH@` — this bundle will *not* install on another |
| OS | RHEL 9 / Rocky Linux 9 / AlmaLinux 9, with systemd (any stock install) |
| CPU / RAM | 2 vCPU / 4 GB minimum; 4 vCPU / 8 GB for more than a handful of instances |
| Disk | at least 5 GB free for the install, plus metric storage under `/srv` |
| Privileges | root (or sudo) for the install; the services then run as the unprivileged `pfw` account |

**Storage sizing.** Metrics live under `/srv` and are the only component that grows
with the number of monitored instances. Budget roughly **1 GB per monitored
PostgreSQL instance per month** at the default 30-day retention, and put `/srv` on
its own filesystem if you can — a full `/srv` stops ingestion.

---

## 1. Copy and unpack

Copy the tarball to the target host, then:

```bash
tar xzf pfw-server-el9-@ARCH@.tar.gz
```

This gives you a directory `pfw-repo/` containing the RPMs and their `repodata/`.

## 2. Import the signing key

Every package in this bundle is signed by Postgres1st — including the third-party
ones, so a single key covers the whole set:

```bash
sudo rpm --import /absolute/path/to/pfw-repo/RPM-GPG-KEY-postgres1st
```

Check the fingerprint against the one published with the release before importing:

```bash
gpg --show-keys --with-fingerprint pfw-repo/RPM-GPG-KEY-postgres1st
```

## 3. Point yum at the bundled repository

```bash
sudo tee /etc/yum.repos.d/pfw.repo >/dev/null <<'EOF'
[pfw]
name=Postgres1st WatchTower Server (air-gapped)
baseurl=file:///absolute/path/to/pfw-repo
enabled=1
gpgcheck=1
gpgkey=file:///absolute/path/to/pfw-repo/RPM-GPG-KEY-postgres1st
EOF
```

Use the **absolute** path to the unpacked `pfw-repo` directory, in both fields.

> With `gpgcheck=1`, dnf refuses any package whose signature does not verify — so a
> tampered or substituted RPM fails the install rather than being silently accepted.
> Verify the tarball's `.sha256` as well; the two checks answer different questions
> (the checksum proves the download is intact, the signature proves who built it).

## 4. Make the operating-system packages available

This bundle ships Postgres1st WatchTower, PostgreSQL and ClickHouse. It does **not** ship your
distribution's own packages — `nginx`, `perl`, `polkit`, `openssl` and friends are the
OS vendor's to supply.

**In almost every case there is nothing to do here.** A host that runs an air-gapped
EL9 server already has a package source — Satellite, Katello, a mirrored `reposync` —
because without one it could not be patched or provisioned at all. Point `dnf` at it as
usual and go to step 5.

The rest of this step is for the exception: an evaluation box or a one-off server that
sits outside your managed fleet and has no repository of its own. For that case, run the
bundled script on an internet-connected EL9 host of the *same architecture* and copy the
result across:

```bash
./fetch-os-dependencies.sh              # on a connected @ARCH@ host
# copy the resulting pfw-os-deps/ directory to this server, then:
sudo tee /etc/yum.repos.d/pfw-os-deps.repo >/dev/null <<'EOF'
[pfw-os-deps]
name=Postgres1st WatchTower OS dependencies
baseurl=file:///absolute/path/to/pfw-os-deps
enabled=1
gpgcheck=0
EOF
```

The script downloads the full dependency closure, not just the top-level names, so the
result installs on a host that has none of them. It refuses to run on a mismatched
architecture or EL version rather than producing a set that fails here.

## 5. Install

```bash
sudo dnf --enablerepo=pfw install pfw-server
```

`--enablerepo=pfw` names where Postgres1st WatchTower, PostgreSQL and ClickHouse come from — and works
whether or not you left `enabled=1` in the repository file, so it is safe if your site
adds repositories disabled by default. Your own repositories stay enabled, which is how
`nginx`, `perl` and the other operating-system packages get resolved, exactly as they
would for anything else you install on this host.

## 6. Start

```bash
sudo systemctl start pfw.target
```

First boot provisions PostgreSQL, ClickHouse, Grafana and the TLS certificate,
so allow a minute or two. To start automatically on boot:

```bash
sudo systemctl enable pfw.target
```

## 7. Verify

```bash
curl -sk -o /dev/null -w '%{http_code}\n' https://127.0.0.1:8443/v1/readyz   # expect 200
systemctl list-units 'pfw*' --all
```

Then open **`https://<host>:8443/`** and sign in as `admin`. The password is
generated during first boot; read it with (
change them on first login).

---

## What gets installed

| Component | Purpose |
|---|---|
| `pfw-server` | systemd units, `pfw.target`, first-boot provisioning |
| `pfw-managed` | control plane / API |
| `pfw-grafana` | dashboards UI |
| `pfw-agent` | monitoring agent + exporters |
| `pfw-dashboards` | dashboard definitions + panel plugins |
| `pfw-victoriametrics` | metrics storage (VictoriaMetrics and vmalert) |
| `pfw-qan-api2` | Query Analytics API |
| `pfw-vmproxy` | metrics proxy |
| `pfw-dump` | support-bundle export — **in the repo, not installed by default** |
| PostgreSQL 18 (PGDG) | server's own metadata store — bundled; not in any EL repo |
| ClickHouse | Query Analytics storage — bundled; not in any EL repo |

`pfw-dump` is the only package above that `dnf install pfw-server` does not pull in: it is
a support tool rather than part of the running server. Install it when you need it:

```bash
sudo dnf --enablerepo=pfw install pfw-dump
```

Pulled from your operating system, not from this bundle:

| Component | Purpose |
|---|---|
| nginx | TLS reverse proxy on 8443 |
| polkit | lets the unprivileged `pfw` account drive its own systemd units |
| openssl, perl, systemd, and their dependencies | supporting libraries and tooling |

Services run as the unprivileged **`pfw`** account under full systemd hardening.
There is no container runtime involved.

## PostgreSQL-only

This build monitors **PostgreSQL only**. Attempting to add a MySQL or MongoDB
service is rejected by design:

```
$ pfw-admin add mysql ...
Service type "mysql" is not supported by this deployment.
```

The same restriction applies to **Advisors**: the list shows only the PostgreSQL
checks. Upstream ships checks for MySQL and MongoDB too, but since services of those
types cannot be registered here, those checks could never produce a result -- listing
them would advertise coverage this build does not have.

If you are comparing against upstream PMM and expect to see more advisor checks, that
is why. Nothing is disabled or broken; the checks that cannot apply are simply not
offered.

## Configure for production

The defaults get you a running server. These four steps make it fit to expose.

### Change the admin password

The admin password is generated during first boot and stored in
`/srv/.pfw-secrets/grafana.env` (mode 0600). Read it with:

```bash
sudo sed -n 's/^GF_SECURITY_ADMIN_PASSWORD=//p' /srv/.pfw-secrets/grafana.env
```

Back up `/srv/.pfw-secrets`: it is hidden and 0700, so `cp -r /srv/*` and tar or rsync
without `.[!.]*` skip it silently, and nothing else holds these values. Change the
password before the host is reachable by
anyone else:

```bash
sudo pfw-admin --server-url=https://admin:admin@127.0.0.1:8443 --server-insecure-tls \
     config --help >/dev/null   # confirms the CLI reaches the server
```

then log in at `https://<host>:8443/` and change it in the profile menu, or via the
Grafana API.

### Replace the self-signed certificate

First boot generates a self-signed certificate, so browsers will warn. Drop your own
PEM files in and restart nginx — the paths are fixed:

```
/srv/nginx/certificate.crt    server certificate (or fullchain)
/srv/nginx/certificate.key    private key
/srv/nginx/ca-certs.pem       issuing chain, if your CA needs it
```

```bash
sudo install -o pfw -g pfw -m 0644 fullchain.pem /srv/nginx/certificate.crt
sudo install -o pfw -g pfw -m 0600 privkey.pem   /srv/nginx/certificate.key
sudo systemctl restart pfw-nginx
```

They must be readable by `pfw`; nginx runs as that account. `pfw-init` only generates
a certificate when one is absent, so yours is not overwritten on restart or upgrade.

`install` above is deliberate: it *creates* the destination file, which is what gives it
the right SELinux label. `mv` and `cp -a` preserve the label the file had elsewhere, and
nginx then cannot read it — on an enforcing host that shows up only as `cannot load
certificate ... Permission denied`, with nothing in the audit log to explain it. If you
place the files any other way, relabel them:

```bash
sudo restorecon -R /srv/nginx
```

`pfw-nginx` also runs this itself before every start, so a restart repairs it either way.

### Restrict what is reachable

Only **8443/tcp** (HTTPS UI and API) needs to be open, and 8080/tcp if you want the
HTTP redirect. Everything else the server runs binds to loopback.

The metric exporters listen on **42000-42010/tcp**, bound to **127.0.0.1 by default** and
protected by HTTP basic auth with a per-agent credential generated at registration. In the
default push mode nothing needs to reach them from off-host, so only 8443 has to be open:

```bash
sudo firewall-cmd --permanent --add-port=8443/tcp
sudo firewall-cmd --reload
```

If you register services with `--metrics-mode=pull --expose-exporter`, those exporters bind
all interfaces and the server scrapes them over the network. Restrict them to the server:

```bash
sudo firewall-cmd --permanent --add-rich-rule='rule family="ipv4" source address="<pfw-server-ip>" port port="42000-42010" protocol="tcp" accept'
sudo firewall-cmd --reload
```

### Set data retention

Retention defaults to **30 days** for both metrics and Query Analytics. Change it in
the UI under *Configuration → Settings*, or via the API:

```bash
curl -sk -u admin:<password> -X PUT https://127.0.0.1:8443/v1/server/settings \
     -H 'Content-Type: application/json' -d '{"data_retention":"2592000s"}'
```

> Do **not** edit `/run/pfw/*.env` or `/usr/lib/pfw/defaults/*.env` to change
> retention. `pfw-managed` renders those files from this setting and will overwrite
> them; on first boot it also restarts the affected services when it does.

## Upgrading an existing install

Upgrades are ordinary `dnf` upgrades against a newer bundle. Nothing is uninstalled
and `/srv` is left alone, so metrics history, dashboards and the Grafana database
survive.

```bash
# 1. unpack the new bundle over a NEW directory (do not overwrite the running one)
tar xzf pfw-server-el9-@ARCH@.tar.gz -C /opt/pfw-new

# 2. repoint the repo file at it, and re-import the key if the release notes say it changed
sudo sed -i 's|baseurl=.*|baseurl=file:///opt/pfw-new/pfw-repo|;
             s|gpgkey=.*|gpgkey=file:///opt/pfw-new/pfw-repo/RPM-GPG-KEY-postgres1st|' \
        /etc/yum.repos.d/pfw.repo

# 3. upgrade
sudo dnf clean all
sudo dnf --enablerepo=pfw upgrade 'pfw-*'
```

Then restart what changed. `dnf` does not restart these services for you:

```bash
sudo systemctl daemon-reload           # if any unit file changed
sudo systemctl restart pfw.target      # or restart individual pfw-* services
```

**What needs a restart, and what does not:**

| Change | Action |
|---|---|
| Dashboards (`pfw-dashboards`) | **Nothing.** Grafana rescans the provisioning path every 60s and picks them up. |
| Grafana config (`/etc/grafana/pfw.ini`) | `systemctl restart pfw-grafana` |
| Any unit file | `systemctl daemon-reload`, then restart that service |
| `pfw-managed`, exporters, agent | `systemctl restart pfw.target` is simplest |

**PostgreSQL major versions do not upgrade in place.** The data directory carries
the major (`/srv/postgres18`), so a bundle built against a newer major will not find
your cluster. `pfw-init` detects this and stops with the required action rather than
starting an empty database — migrate with `pg_upgrade` or a dump/restore first. The
release notes call out any bundle that moves the major.

**Verify afterwards** exactly as for a fresh install:

```bash
curl -sk -o /dev/null -w '%{http_code}\n' https://127.0.0.1:8443/v1/readyz   # expect 200
systemctl --failed
```

If `readyz` does not reach 200, start with `journalctl -u pfw-managed` and
`journalctl -u pfw-grafana`.

## Monitoring a PostgreSQL host

Everything above installs the monitoring server. To start monitoring a database, install
the `pfw-agent` agent on that database host — **see `INSTALL-CLIENT.md`**, which ships in
this bundle and is written to be followed on the database host itself.

It uses the same tarball you unpacked here; there is no separate client download.

## Ports

| Port | Bound on | Purpose |
|---|---|---|
| 8443 | all interfaces | HTTPS UI and API — the only port that must be reachable |
| 8080 | loopback | HTTP, redirects to HTTPS |
| 42000-42010 | loopback by default; all interfaces only with `--expose-exporter` | exporters, authenticated with a per-agent credential |

Everything else — PostgreSQL 5432, ClickHouse 9000/8123, VictoriaMetrics 9090,
Grafana 3000, qan-api2 9911/9922 — binds to **loopback only**.

> The exporter ports carry database and OS statistics. They require HTTP basic auth
> with a credential generated per agent at registration, and bind **loopback by
> default**. They only bind all interfaces for services registered
> `--metrics-mode=pull --expose-exporter`; firewall those to the Postgres1st WatchTower
> server (see *Restrict what is reachable*).

## Operations

### Health

```bash
curl -sk -o /dev/null -w '%{http_code}\n' https://127.0.0.1:8443/v1/readyz   # expect 200
systemctl --failed
systemctl list-units 'pfw*'
```

`pfw-clickhouse-perms.service` showing `inactive (dead)` is normal — it is a one-shot
that runs before ClickHouse and exits.

### Logs

Everything logs to the journal:

```bash
journalctl -u pfw-managed -f          # control plane, the usual starting point
journalctl -u pfw-grafana             # UI
journalctl -u pfw-init                # first-boot provisioning
journalctl -u pfw-clickhouse          # Query Analytics storage
```

### Stopping and starting

```bash
sudo systemctl stop pfw.target        # stops the whole stack
sudo systemctl start pfw.target
```

### Backing up the server

The server's own state is entirely under `/srv` (PostgreSQL, ClickHouse, Grafana,
TLS material). Stop the stack for a consistent copy:

```bash
sudo systemctl stop pfw.target
sudo tar czf pfw-srv-$(date +%F).tar.gz -C / srv
sudo systemctl start pfw.target
```

To restore, stop the stack, replace `/srv` from the archive, make sure it is still
owned by `pfw`, and start again. Restore onto the **same PostgreSQL major version**
— see the note in *Upgrading*.

> `/srv` also holds `pfw-encryption.key`, which the server uses to encrypt stored
> credentials for monitored instances. Back it up with everything else and keep the
> archive somewhere restricted: without that file those credentials cannot be
> decrypted, and with it they can.

### Uninstalling

```bash
sudo systemctl disable --now pfw.target
sudo dnf remove pfw-server pfw-managed pfw-agent pfw-grafana pfw-dashboards
```

Package removal deliberately leaves `/srv` in place, so your metrics survive a
reinstall. Delete it explicitly if you want the data gone.

## Troubleshooting

```bash
systemctl --failed                  # what broke
journalctl -u pfw-init.service      # first-boot provisioning
journalctl -u pfw-managed.service   # control plane
journalctl -u pfw-grafana.service   # dashboards UI
```

Persistent state lives under `/srv` (`/srv/postgres18`, `/srv/clickhouse`,
`/srv/grafana`, `/srv/logs`). Removing `/srv` and restarting `pfw.target`
re-runs first-boot provisioning from scratch.

## Query Analytics and `pg_stat_monitor`

`pfw-admin add postgresql` defaults to `--query-source=pgstatmonitor`, and
**`pg_stat_monitor` is not part of this bundle**. It belongs on the database you are
monitoring, not on the Postgres1st WatchTower server, so installing it is the operator's step —
the bundle would have no way to reach a monitored host anyway.

Either install `pg_stat_monitor` on the monitored PostgreSQL and keep the default, or
register the service with the extension PostgreSQL already ships:

```bash
pfw-admin add postgresql --query-source=pgstatements --host=… --port=5432 \
  --username=… --password=… <service-name>
```

`pgstatements` uses `pg_stat_statements`, which is in PostgreSQL's own contrib package
and needs only to be enabled. It reports less than `pg_stat_monitor` — no per-bucket
histograms and no query examples — but it needs nothing extra installed.
