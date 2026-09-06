# Postgres1st WatchTower client — monitoring a PostgreSQL host (RHEL / Rocky Linux 9, @ARCH@)

Run this on **each PostgreSQL server you want to monitor** — not on the Postgres1st WatchTower monitoring
server itself. For installing the monitoring server, see `INSTALL.md`.

No internet access is required on this host.

**You need, before starting:**

| | |
|---|---|
| The bundle | `pfw-server-el9-@ARCH@.tar.gz` — the same file used to install the monitoring server. It is named `server` because that is the larger part of it, but it contains the `pfw-agent` agent too. There is no separate client download. |
| A running Postgres1st WatchTower server | its address and the `admin` password |
| Architecture | `@ARCH@` — this bundle will *not* install on another |
| Privileges | root (or sudo) on this host, and an account on the database that can create a role |
| Network | this host must reach the Postgres1st WatchTower server on 8443/tcp. Nothing needs to reach *this* host unless you register services with `--metrics-mode=pull` (see step 6) |

---

## 1. Prepare the database

Create an account for monitoring and enable the statistics extension. Run against the
instance you want to monitor:

```sql
CREATE USER pfw WITH PASSWORD 'StrongPassword';
GRANT pg_monitor TO pfw;                      -- PostgreSQL 10+
CREATE EXTENSION IF NOT EXISTS pg_stat_statements;
```

`pg_stat_statements` must also be loaded at startup, which needs a restart if it is not
already there:

```
shared_preload_libraries = 'pg_stat_statements'
```

`pg_monitor` is a built-in role and is sufficient — a superuser is not required. Query
Analytics works with `pg_stat_statements`; if you have `pg_stat_monitor` installed
instead, pass `--query-source=pgstatmonitor` in step 4.

## 2. Copy and unpack the bundle

Nothing on this host was set up by the monitoring server's installation. Copy the tarball
across and unpack it:

```bash
tar xzf pfw-server-el9-@ARCH@.tar.gz
```

This gives you a directory `pfw-repo/` containing the RPMs and their `repodata/`.

## 3. Import the signing key

Every package is signed by Postgres1st, so a single key covers the whole set:

```bash
sudo rpm --import /absolute/path/to/pfw-repo/RPM-GPG-KEY-postgres1st
```

Check the fingerprint against the one published with the release before importing:

```bash
gpg --show-keys --with-fingerprint pfw-repo/RPM-GPG-KEY-postgres1st
```

## 4. Point yum at the repository and install

```bash
sudo tee /etc/yum.repos.d/pfw.repo >/dev/null <<'EOF'
[pfw]
name=Postgres1st WatchTower
baseurl=file:///absolute/path/to/pfw-repo
enabled=1
gpgcheck=1
gpgkey=file:///absolute/path/to/pfw-repo/RPM-GPG-KEY-postgres1st
EOF

sudo dnf --enablerepo=pfw install pfw-agent
```

Use the **absolute** path to the unpacked `pfw-repo` directory, in both fields.

Only `pfw-agent` is installed here. The rest of the bundle is for the monitoring server.

The operating-system packages `pfw-agent` depends on come from your own repositories, as
they would for anything else you install. If this host has no repository of its own, the
bundle includes `fetch-os-dependencies.sh` — run it on an internet-connected EL9 host of
the same architecture and copy the result across, as described in `INSTALL.md`.

The package enables and starts `pfw-agent` for you; there is nothing to start by hand.

## 5. Register this host, then the database

```bash
sudo pfw-admin config --server-url=https://admin:<password>@<pfw-host>:8443 \
     --server-insecure-tls <this-host-address> generic <node-name>

sudo pfw-admin add postgresql --host=127.0.0.1 --port=5432 \
     --username=pfw --password=StrongPassword \
     --environment=production --cluster=<cluster-name> <service-name>
```

Both commands are needed and in this order: the first registers the machine as a node,
the second registers the database running on it. `pfw-admin add` fails if the node is not
registered first.

Both commands take passwords as arguments, which puts them in your shell history and makes
them briefly visible in `ps` to every local user. There is no flag that avoids this. On a
shared host, run them in a shell with `HISTCONTROL=ignorespace` set and a leading space,
and treat the monitoring role's password as one that local users may have seen.

`--server-insecure-tls` is needed because the server's first boot generates a self-signed
certificate, which the client would otherwise refuse. Drop it once the server has a
trusted certificate. `--environment` and `--cluster` are optional but worth setting — the
dashboards group by them.

## 6. Exporter ports — usually nothing to do

`pfw-agent` runs exporters on **42000-42010/tcp**. They bind **127.0.0.1 by default**
and require HTTP basic auth with a credential generated per agent at registration.

The command in step 5 passes no `--metrics-mode`, which means push mode: the agent sends
metrics *to* the server over the connection it already opened on 8443, nothing listens
off-host, and **no firewall rule is needed**.

You only need to open these ports if you deliberately register a service with
`--metrics-mode=pull`, so the server scrapes this host over the network. In that case
bind the exporters off-host and restrict them to the server:

```bash
# On the client: register the service for pull-mode scraping.
sudo pfw-admin add postgresql --metrics-mode=pull --expose-exporter ...

sudo firewall-cmd --permanent --add-rich-rule='rule family="ipv4" source address="<pfw-server-ip>" port port="42000-42010" protocol="tcp" accept'
sudo firewall-cmd --reload
```

Without `--expose-exporter` the exporter stays on loopback and the server cannot reach it.
To restore the previous fleet-wide behaviour while migrating, set
`PFW_EXPOSE_EXPORTERS=true` on the **server** and restart `pfw-managed`.

They expose database and operating-system statistics. Do not leave them open to the
network at large.

## 7. Verify

```bash
sudo pfw-admin list
```

The service should appear, and show in the Postgres1st WatchTower server's UI within a minute or two, once
the first scrape lands.

---

## Removing an instance

```bash
sudo pfw-admin remove postgresql <service-name>
```

To remove monitoring from this host entirely:

```bash
sudo pfw-admin remove postgresql <service-name>
sudo systemctl disable --now pfw-agent
sudo dnf remove pfw-agent
```

The `pfw` database role and the `pg_stat_statements` extension are left alone — drop them
yourself if you no longer want them.

## Upgrading the client

Unpack the newer bundle, repoint the repository at it, and upgrade:

```bash
sudo dnf clean all
sudo dnf --enablerepo=pfw upgrade pfw-agent
sudo systemctl restart pfw-agent
```

Keep the client at the same version as the server. A client older than the server may not
report everything the server expects.

## Troubleshooting

```bash
systemctl status pfw-agent
journalctl -u pfw-agent -n 200
sudo pfw-admin list
```

**The service does not appear on the server.** Check that `pfw-admin config` succeeded and
that this host can reach the server on 8443.

**The service appears but has no data.** In the default push mode the agent sends metrics
to the server, so check the agent is connected (`pfw-admin status`) rather than any
inbound firewall rule. Confirm the exporter is running with `ss -ltn | grep 420` — it
should be listening on `127.0.0.1`. If you registered the service with
`--metrics-mode=pull`, then the scrape does go the other way: the exporter must be bound
off-host with `--expose-exporter` and reachable from the server on 42000-42010 (step 6).

**Authentication failures in the log.** Confirm the `pfw` role and password from step 1,
and that the database allows connections from `127.0.0.1` in `pg_hba.conf`.
