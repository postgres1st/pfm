# PFMM Server — air-gapped installation (RHEL / Rocky Linux 9, @ARCH@)

Postgres1st Monitoring and Management (PFMM) server, delivered as a self-contained
yum repository. **No internet access is required at any point.** Every transitive
dependency is bundled.

## Requirements

| | |
|---|---|
| Architecture | `@ARCH@` — this bundle will *not* install on another |
| OS | RHEL 9 / Rocky Linux 9 / AlmaLinux 9, with systemd (any stock install) |
| CPU / RAM | 2 vCPU / 4 GB minimum; 4 vCPU / 8 GB for more than a handful of instances |
| Disk | ~2.5 GB for the packages, plus metric storage under `/srv` |
| Privileges | root (or sudo) for the install; the services then run as the unprivileged `pfm` account |

**Storage sizing.** Metrics live under `/srv` and are the only component that grows
with the number of monitored instances. Budget roughly **1 GB per monitored
PostgreSQL instance per month** at the default 30-day retention, and put `/srv` on
its own filesystem if you can — a full `/srv` stops ingestion.

---

## 1. Copy and unpack

Copy the tarball to the target host, then:

```bash
tar xzf pfmm-server-el9-@ARCH@.tar.gz
```

This gives you a directory `pfmm-repo/` containing the RPMs and their `repodata/`.

## 2. Import the signing key

Every package in this bundle is signed by Postgres1st — including the third-party
ones, so a single key covers the whole set:

```bash
sudo rpm --import /absolute/path/to/pfmm-repo/RPM-GPG-KEY-postgres1st
```

Check the fingerprint against the one published with the release before importing:

```bash
gpg --show-keys --with-fingerprint pfmm-repo/RPM-GPG-KEY-postgres1st
```

## 3. Point yum at the bundled repository

```bash
sudo tee /etc/yum.repos.d/pfmm.repo >/dev/null <<'EOF'
[pfmm]
name=PFMM Server (air-gapped)
baseurl=file:///absolute/path/to/pfmm-repo
enabled=1
gpgcheck=1
gpgkey=file:///absolute/path/to/pfmm-repo/RPM-GPG-KEY-postgres1st
EOF
```

Use the **absolute** path to the unpacked `pfmm-repo` directory, in both fields.

> With `gpgcheck=1`, dnf refuses any package whose signature does not verify — so a
> tampered or substituted RPM fails the install rather than being silently accepted.
> Verify the tarball's `.sha256` as well; the two checks answer different questions
> (the checksum proves the download is intact, the signature proves who built it).

## 4. Install

```bash
sudo dnf --disablerepo='*' --enablerepo=pfmm install pfm-server
```

`--disablerepo='*'` guarantees nothing is pulled from the network — the install
will fail loudly rather than silently reaching out.

## 5. Start

```bash
sudo systemctl start pfm.target
```

First boot provisions PostgreSQL, ClickHouse, Grafana and the TLS certificate,
so allow a minute or two. To start automatically on boot:

```bash
sudo systemctl enable pfm.target
```

## 6. Verify

```bash
curl -sk -o /dev/null -w '%{http_code}\n' https://127.0.0.1:8443/v1/readyz   # expect 200
systemctl list-units 'pfm*' --all
```

Then open **`https://<host>:8443/`** (default credentials `admin` / `admin` —
change them on first login).

---

## What gets installed

| Component | Purpose |
|---|---|
| `pfm-server` | systemd units, `pfm.target`, first-boot provisioning |
| `pfm-managed` | control plane / API |
| `pfm-grafana` | dashboards UI |
| `pfm-client` | monitoring agent + exporters |
| `percona-dashboards` | dashboard definitions + panel plugins |
| `percona-victoriametrics` | metrics storage |
| `percona-qan-api2` | Query Analytics API |
| `vmproxy`, `pmm-dump` | metrics proxy, support-bundle export |
| PostgreSQL 18 (PGDG) | server's own metadata store |
| ClickHouse | Query Analytics storage |
| nginx, polkit, openssl | TLS reverse proxy, privilege mediation |

Services run as the unprivileged **`pfm`** account under full systemd hardening.
There is no container runtime involved.

## PostgreSQL-only

This build monitors **PostgreSQL only**. Attempting to add a MySQL or MongoDB
service is rejected by design:

```
$ pfm-admin add mysql ...
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

The server ships with `admin` / `admin`. Change it before the host is reachable by
anyone else:

```bash
sudo pfm-admin --server-url=https://admin:admin@127.0.0.1:8443 --server-insecure-tls \
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
sudo install -o pfm -g pfm -m 0644 fullchain.pem /srv/nginx/certificate.crt
sudo install -o pfm -g pfm -m 0600 privkey.pem   /srv/nginx/certificate.key
sudo systemctl restart pfm-nginx
```

They must be readable by `pfm`; nginx runs as that account. `pfm-init` only generates
a certificate when one is absent, so yours is not overwritten on restart or upgrade.

### Restrict what is reachable

Only **8443/tcp** (HTTPS UI and API) needs to be open, and 8080/tcp if you want the
HTTP redirect. Everything else the server runs binds to loopback.

The exception is the metric exporters, which bind all interfaces on **42000-42010/tcp**
by design — that is how the server scrapes them, and it applies to every monitored host,
not just this one. They expose database and OS statistics without authentication, so
restrict them to the PFMM server:

```bash
sudo firewall-cmd --permanent --add-port=8443/tcp
sudo firewall-cmd --permanent --add-rich-rule='rule family=ipv4 \
     source address=<pfmm-server-ip> port port=42000-42010 protocol=tcp accept'
sudo firewall-cmd --reload
```

### Set data retention

Retention defaults to **30 days** for both metrics and Query Analytics. Change it in
the UI under *Configuration → Settings*, or via the API:

```bash
curl -sk -u admin:<password> -X PUT https://127.0.0.1:8443/v1/server/settings \
     -H 'Content-Type: application/json' -d '{"data_retention":"2592000s"}'
```

> Do **not** edit `/run/pfm/*.env` or `/usr/lib/pfm/defaults/*.env` to change
> retention. `pfm-managed` renders those files from this setting and will overwrite
> them; on first boot it also restarts the affected services when it does.

## Upgrading an existing install

Upgrades are ordinary `dnf` upgrades against a newer bundle. Nothing is uninstalled
and `/srv` is left alone, so metrics history, dashboards and the Grafana database
survive.

```bash
# 1. unpack the new bundle over a NEW directory (do not overwrite the running one)
tar xzf pfmm-server-el9-@ARCH@.tar.gz -C /opt/pfmm-new

# 2. repoint the repo file at it, and re-import the key if the release notes say it changed
sudo sed -i 's|baseurl=.*|baseurl=file:///opt/pfmm-new/pfmm-repo|;
             s|gpgkey=.*|gpgkey=file:///opt/pfmm-new/pfmm-repo/RPM-GPG-KEY-postgres1st|' \
        /etc/yum.repos.d/pfmm.repo

# 3. upgrade
sudo dnf clean all
sudo dnf --disablerepo='*' --enablerepo=pfmm upgrade
```

Then restart what changed. `dnf` does not restart these services for you:

```bash
sudo systemctl daemon-reload           # if any unit file changed
sudo systemctl restart pfm.target      # or restart individual pfm-* services
```

**What needs a restart, and what does not:**

| Change | Action |
|---|---|
| Dashboards (`percona-dashboards`) | **Nothing.** Grafana rescans the provisioning path every 60s and picks them up. |
| Grafana config (`/etc/grafana/pfm.ini`) | `systemctl restart pfm-grafana` |
| Any unit file | `systemctl daemon-reload`, then restart that service |
| `pfm-managed`, exporters, agent | `systemctl restart pfm.target` is simplest |

**PostgreSQL major versions do not upgrade in place.** The data directory carries
the major (`/srv/postgres18`), so a bundle built against a newer major will not find
your cluster. `pfm-init` detects this and stops with the required action rather than
starting an empty database — migrate with `pg_upgrade` or a dump/restore first. The
release notes call out any bundle that moves the major.

**Verify afterwards** exactly as for a fresh install:

```bash
curl -sk -o /dev/null -w '%{http_code}\n' https://127.0.0.1:8443/v1/readyz   # expect 200
systemctl --failed
```

If `readyz` does not reach 200, start with `journalctl -u pfm-managed` and
`journalctl -u pfm-grafana`.

## Adding a monitored PostgreSQL instance

### 1. Prepare the database

Create an account for monitoring and enable the statistics extension. Run on the
instance you want to monitor:

```sql
CREATE USER pfm WITH PASSWORD 'StrongPassword';
GRANT pg_monitor TO pfm;                      -- PostgreSQL 10+
CREATE EXTENSION IF NOT EXISTS pg_stat_statements;
```

`pg_stat_statements` must also be loaded at startup, which needs a restart if it is
not already there:

```
shared_preload_libraries = 'pg_stat_statements'
```

`pg_monitor` is a built-in role and is sufficient — a superuser is not required.
Query Analytics works with `pg_stat_statements`; if you have `pg_stat_monitor`
installed instead, pass `--query-source=pgstatmonitor`.

### 2. Install the client on that host

Use the same bundle, so the database host needs no internet either:

```bash
sudo dnf --disablerepo='*' --enablerepo=pfmm install pfm-client
```

### 3. Register the host, then the instance

```bash
sudo pfm-admin config --server-url=https://admin:<password>@<pfmm-host>:8443 \
     --server-insecure-tls <this-host-address> generic <node-name>

sudo pfm-admin add postgresql --host=127.0.0.1 --port=5432 \
     --username=pfm --password=StrongPassword \
     --environment=production --cluster=<cluster-name> <service-name>
```

Drop `--server-insecure-tls` once you have installed a trusted certificate.
`--environment` and `--cluster` are optional but worth setting — the dashboards
group by them.

Confirm it arrived:

```bash
sudo pfm-admin list
```

The service appears in the UI within a minute or two, once the first scrape lands.

### Removing an instance

```bash
sudo pfm-admin remove postgresql <service-name>
```

## Ports

| Port | Bound on | Purpose |
|---|---|---|
| 8443 | all interfaces | HTTPS UI and API — the only port that must be reachable |
| 8080 | loopback | HTTP, redirects to HTTPS |
| 42000-42010 | all interfaces, **on every monitored host** | exporters, scraped by the server |

Everything else — PostgreSQL 5432, ClickHouse 9000/8123, VictoriaMetrics 9090,
Grafana 3000, qan-api2 9911/9922 — binds to **loopback only**.

> The exporter ports carry database and OS statistics and are **not
> authenticated**. They bind all interfaces because the server scrapes them over
> the network. Firewall them to the PFMM server (see *Restrict what is reachable*).

## Operations

### Health

```bash
curl -sk -o /dev/null -w '%{http_code}\n' https://127.0.0.1:8443/v1/readyz   # expect 200
systemctl --failed
systemctl list-units 'pfm*'
```

`pfm-clickhouse-perms.service` showing `inactive (dead)` is normal — it is a one-shot
that runs before ClickHouse and exits.

### Logs

Everything logs to the journal:

```bash
journalctl -u pfm-managed -f          # control plane, the usual starting point
journalctl -u pfm-grafana             # UI
journalctl -u pfm-init                # first-boot provisioning
journalctl -u pfm-clickhouse          # Query Analytics storage
```

### Stopping and starting

```bash
sudo systemctl stop pfm.target        # stops the whole stack
sudo systemctl start pfm.target
```

### Backing up the server

The server's own state is entirely under `/srv` (PostgreSQL, ClickHouse, Grafana,
TLS material). Stop the stack for a consistent copy:

```bash
sudo systemctl stop pfm.target
sudo tar czf pfmm-srv-$(date +%F).tar.gz -C / srv
sudo systemctl start pfm.target
```

To restore, stop the stack, replace `/srv` from the archive, make sure it is still
owned by `pfm`, and start again. Restore onto the **same PostgreSQL major version**
— see the note in *Upgrading*.

> `/srv` also holds `pmm-encryption.key`, which the server uses to encrypt stored
> credentials for monitored instances. Back it up with everything else and keep the
> archive somewhere restricted: without that file those credentials cannot be
> decrypted, and with it they can.

### Uninstalling

```bash
sudo systemctl disable --now pfm.target
sudo dnf remove pfm-server pfm-managed pfm-client pfm-grafana percona-dashboards
```

Package removal deliberately leaves `/srv` in place, so your metrics survive a
reinstall. Delete it explicitly if you want the data gone.

## Troubleshooting

```bash
systemctl --failed                  # what broke
journalctl -u pfm-init.service      # first-boot provisioning
journalctl -u pfm-managed.service   # control plane
journalctl -u pfm-grafana.service   # dashboards UI
```

Persistent state lives under `/srv` (`/srv/postgres18`, `/srv/clickhouse`,
`/srv/grafana`, `/srv/logs`). Removing `/srv` and restarting `pfm.target`
re-runs first-boot provisioning from scratch.
