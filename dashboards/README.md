## Grafana dashboards for PostgreSQL monitoring

The dashboards PGF WatchTower provisions into Grafana. Sources live under `dashboards/`, grouped by
folder — the folder name is what appears in Grafana's dashboard list:

| Folder | Covers |
|---|---|
| `PostgreSQL` | instance summary, instance comparison and overview, replication, Patroni detail, checkpoints/buffers/WAL, top queries — and the HAProxy instance summary |
| `OS` | CPU, memory, disk, network, NUMA, processes, node summary and comparison |
| `Insight` | cross-cutting views — home dashboard, advanced data exploration, exporter status, VictoriaMetrics |
| `Query Analytics` | QAN |
| `WatchTower Health` | the monitoring stack's own health |
| `Experimental`, `Kubernetes (experimental)` | not provisioned by default |

The `WatchTower Health` folder was renamed from `PMM Health` as part of the PGF WatchTower
rebrand. Because `foldersFromFilesStructure: true` is set in the Grafana provisioning config,
the directory name *is* the folder name shown in the UI, and the folder also appears in the
`includes` paths of `dashboards/pmm-app/src/plugin.json` — rename all three together.

The upstream MySQL, MongoDB, ProxySQL, PXC and Valkey/Redis dashboards are **not** in this
repository. They were removed rather than hidden: the service-type allowlist
(`managed/models/service_type_allowlist.go`) rejects those services, so they could never
populate. HAProxy is kept because it commonly fronts a Patroni cluster.

`pmm-app/dist/` is build output — edit the sources under `dashboards/`, not there.

## Contributing

We welcome contributions to this repository! Detailed information in [CONTRIBUTING.md](CONTRIBUTING.md)
