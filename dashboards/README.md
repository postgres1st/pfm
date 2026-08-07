## Grafana dashboards for PostgreSQL monitoring

The dashboards PFMM provisions into Grafana. Sources live under `dashboards/`, grouped by
folder — the folder name is what appears in Grafana's dashboard list:

| Folder | Covers |
|---|---|
| `PostgreSQL` | instance summary, instance comparison and overview, replication, Patroni detail, checkpoints/buffers/WAL, top queries — and the HAProxy instance summary |
| `OS` | CPU, memory, disk, network, NUMA, processes, node summary and comparison |
| `Insight` | cross-cutting views — home dashboard, advanced data exploration, exporter status, VictoriaMetrics |
| `Query Analytics` | QAN |
| `PMM Health` | the monitoring stack's own health |
| `Experimental`, `Kubernetes (experimental)` | not provisioned by default |

The `PMM Health` folder keeps its upstream name; renaming it is part of the outstanding
rebrand rather than something to change in this file alone, since the name is also a
provisioning path.

The upstream MySQL, MongoDB, ProxySQL, PXC and Valkey/Redis dashboards are **not** in this
repository. They were removed rather than hidden: the service-type allowlist
(`managed/models/service_type_allowlist.go`) rejects those services, so they could never
populate. HAProxy is kept because it commonly fronts a Patroni cluster.

`pmm-app/dist/` is build output — edit the sources under `dashboards/`, not there.

These dashboards are part of Postgres1st Monitoring and Management (PFMM), derived from
[Percona Monitoring and Management](https://github.com/percona/pmm).

## Contributing

We welcome contributions to this repository! Detailed information in [CONTRIBUTING.md](CONTRIBUTING.md)
