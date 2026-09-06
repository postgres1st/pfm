# Postgres1st WatchTower 3.9.0 beta2 — release notes

**Postgres1st WatchTower** — PostgreSQL monitoring and query analytics, delivered as a signed
package repository you install on your own RHEL, Rocky Linux or AlmaLinux 9 server. No
internet access is required on that server, and no container runtime is involved.

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

**The product is Postgres1st WatchTower.** Packages are `pfw-*`, units are `pfw-*.service`, the
service account is `pfw`, and everything installs under `/opt/postgres1st/watchtower`.
The client agent is `pfw-agent`; its configuration lives at
`/opt/postgres1st/watchtower/config/pfw-agent.yaml`.

**Every server component is now built from source.** beta1 fetched four of them prebuilt
from Percona's public build cache. Those artifacts carried Percona's branding regardless
of what our specs said, the cache rotates its keys so the pinned versions expired, and it
publishes x86_64 only. Building them ourselves fixed all three at once, and is what makes
an **aarch64 bundle** possible — beta1 could not produce one.

**Security fixes carried forward from the beta1 line**, all present here: bootstrap
credentials generated per install rather than a shipped constant, the internal PostgreSQL
password no longer written to a world-readable file, and exporter credentials generated on
every agent-creation path.

**Zero egress by design.** The server makes no outbound requests: Grafana's news feed is
off, and the upstream blog-feed proxy and its dashboard panel are gone.

---

## What you get

| | |
|---|---|
| **Dashboards** | provisioned Grafana dashboards for PostgreSQL and host metrics |
| **Query Analytics** | per-query statistics from `pg_stat_statements` or `pg_stat_monitor` |
| **Advisors** | PostgreSQL configuration and health checks |
| **Metrics storage** | VictoriaMetrics, 30-day retention by default |
| **Agents** | `pfw-agent` on the monitored hosts, installed from the same bundle |

The server runs as the unprivileged `pfw` account under full systemd hardening.

---

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

## Install

1. Copy the bundle to the server and unpack it.
2. `sudo ./fetch-os-dependencies.sh` on an internet-connected host if the server lacks
   its distribution's own packages.
3. `sudo dnf --enablerepo=pfw install pfw-server`
4. `sudo systemctl start pfw.target`

Read the generated admin password once, from `/srv/.pfw-secrets/grafana.env`. **Back up
`/srv/.pfw-secrets`** — on a fresh install it is the only copy.

Upgrade a beta2-or-later host with `sudo dnf --enablerepo=pfw upgrade 'pfw-*' vmproxy`.

---

## Limitations

- **No upgrade from beta1.** See the warning above.
- **PostgreSQL only.** MySQL, MongoDB, ProxySQL and Valkey service types are rejected at
  registration. This is deliberate, and controlled by `PFW_DB_TYPES`.
- **Container and Helm deployment are not part of this release.** The bundle installs
  natively; the Docker path is not rebranded and is not tested here.
- **The x86_64 bundle is built from the same tree but is not certified in this release.**
