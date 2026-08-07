# PFMM client — monitoring a PostgreSQL host (RHEL / Rocky Linux 9, @ARCH@)

Run this on **each PostgreSQL server you want to monitor** — not on the PFMM monitoring
server itself. For installing the monitoring server, see `INSTALL.md`.

No internet access is required on this host.

**You need, before starting:**

| | |
|---|---|
| The bundle | `pfmm-server-el9-@ARCH@.tar.gz` — the same file used to install the monitoring server. It is named `server` because that is the larger part of it, but it contains the `pfm-client` agent too. There is no separate client download. |
| A running PFMM server | its address and the `admin` password |
| Architecture | `@ARCH@` — this bundle will *not* install on another |
| Privileges | root (or sudo) on this host, and an account on the database that can create a role |
| Network | this host must be reachable **from** the PFMM server on ports 42000-42010 |

---

## 1. Prepare the database

Create an account for monitoring and enable the statistics extension. Run against the
instance you want to monitor:

```sql
CREATE USER pfm WITH PASSWORD 'StrongPassword';
GRANT pg_monitor TO pfm;                      -- PostgreSQL 10+
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
tar xzf pfmm-server-el9-@ARCH@.tar.gz
```

This gives you a directory `pfmm-repo/` containing the RPMs and their `repodata/`.

## 3. Import the signing key

Every package is signed by Postgres1st, so a single key covers the whole set:

```bash
sudo rpm --import /absolute/path/to/pfmm-repo/RPM-GPG-KEY-postgres1st
```

Check the fingerprint against the one published with the release before importing:

```bash
gpg --show-keys --with-fingerprint pfmm-repo/RPM-GPG-KEY-postgres1st
```

## 4. Point yum at the repository and install

```bash
sudo tee /etc/yum.repos.d/pfmm.repo >/dev/null <<'EOF'
[pfmm]
name=PFMM
baseurl=file:///absolute/path/to/pfmm-repo
enabled=1
gpgcheck=1
gpgkey=file:///absolute/path/to/pfmm-repo/RPM-GPG-KEY-postgres1st
EOF

sudo dnf --enablerepo=pfmm install pfm-client
```

Use the **absolute** path to the unpacked `pfmm-repo` directory, in both fields.

Only `pfm-client` is installed here. The rest of the bundle is for the monitoring server.

The operating-system packages `pfm-client` depends on come from your own repositories, as
they would for anything else you install. If this host has no repository of its own, the
bundle includes `fetch-os-dependencies.sh` — run it on an internet-connected EL9 host of
the same architecture and copy the result across, as described in `INSTALL.md`.

The package enables and starts `pfm-agent` for you; there is nothing to start by hand.

## 5. Register this host, then the database

```bash
sudo pfm-admin config --server-url=https://admin:<password>@<pfmm-host>:8443 \
     --server-insecure-tls <this-host-address> generic <node-name>

sudo pfm-admin add postgresql --host=127.0.0.1 --port=5432 \
     --username=pfm --password=StrongPassword \
     --environment=production --cluster=<cluster-name> <service-name>
```

Both commands are needed and in this order: the first registers the machine as a node,
the second registers the database running on it. `pfm-admin add` fails if the node is not
registered first.

Both commands take passwords as arguments, which puts them in your shell history and makes
them briefly visible in `ps` to every local user. There is no flag that avoids this. On a
shared host, run them in a shell with `HISTCONTROL=ignorespace` set and a leading space,
and treat the monitoring role's password as one that local users may have seen.

`--server-insecure-tls` is needed because the server's first boot generates a self-signed
certificate, which the client would otherwise refuse. Drop it once the server has a
trusted certificate. `--environment` and `--cluster` are optional but worth setting — the
dashboards group by them.

## 6. Open the exporter ports to the PFMM server

`pfm-client` runs exporters that bind **42000-42010 on all interfaces** and are **not
authenticated**. This is upstream behaviour and applies to every monitored host. The PFMM
server scrapes them over the network, so they must be reachable from it — and from
nothing else:

```bash
sudo firewall-cmd --permanent --add-rich-rule='rule family="ipv4" source address="<pfmm-server-ip>" port port="42000-42010" protocol="tcp" accept'
sudo firewall-cmd --reload
```

They expose database and operating-system statistics. Do not leave them open to the
network at large.

## 7. Verify

```bash
sudo pfm-admin list
```

The service should appear, and show in the PFMM server's UI within a minute or two, once
the first scrape lands.

---

## Removing an instance

```bash
sudo pfm-admin remove postgresql <service-name>
```

To remove monitoring from this host entirely:

```bash
sudo pfm-admin remove postgresql <service-name>
sudo systemctl disable --now pfm-agent
sudo dnf remove pfm-client
```

The `pfm` database role and the `pg_stat_statements` extension are left alone — drop them
yourself if you no longer want them.

## Upgrading the client

Unpack the newer bundle, repoint the repository at it, and upgrade:

```bash
sudo dnf clean all
sudo dnf --enablerepo=pfmm upgrade pfm-client
sudo systemctl restart pfm-agent
```

Keep the client at the same version as the server. A client older than the server may not
report everything the server expects.

## Troubleshooting

```bash
systemctl status pfm-agent
journalctl -u pfm-agent -n 200
sudo pfm-admin list
```

**The service does not appear on the server.** Check that `pfm-admin config` succeeded and
that this host can reach the server on 8443.

**The service appears but has no data.** The scrape goes the other way — check the server
can reach *this* host on 42000-42010 (step 6), and that the exporters are listening:
`ss -ltn | grep 420`.

**Authentication failures in the log.** Confirm the `pfm` role and password from step 1,
and that the database allows connections from `127.0.0.1` in `pg_hba.conf`.
