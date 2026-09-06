# Dashboards overview

Dashboards are a compilation of visualizations, including charts and metrics, that enable you to view performance metrics from node to single query for multiple databases in a centralized location.

A dashboard is a group of one or more panels organized and arranged into rows. Panels refer to individual components or visual elements that display specific data or visualizations within the dashboard's layout. These panels are the building blocks that collectively form a dashboard, providing a means to present and visualize data in various formats. Dashboards are grouped into folders. You can customize these by renaming them or creating new ones.

Dashboards provide insightful and actionable data, enabling you to gain an overview of your system status quickly. These dashboards enable you to drill down into specific time frames, apply filters, and analyze data trends for troubleshooting and performance optimization. Customizable dashboards and real-time alerting facilitate seamless monitoring of database performance.


## Available dashboards

Performance Monitoring and Management (Postgres1st WatchTower) offers a range of dashboards you can access. Some of these dashboards are as follows:

=== "Insight"

    | Dashboard                                                                                | Description |
    |------------------------------------------------------------------------------------------|-------------|
    | [Advanced Data Exploration]                                                              | Explore and analyze metrics with custom queries
    | [Home Dashboard]                                                                         | Overview of monitored environments and quick access to key dashboards
    | [Prometheus Exporter Status]                                                             | Monitor exporter health and availability
    | [Prometheus Exporters Overview]                                                          | Resource usage (CPU, memory) across all exporters
    | [VictoriaMetrics]                                                                        | VictoriaMetrics performance and storage metrics
    | [VictoriaMetrics Agents Overview]                                                        | VictoriaMetrics agents status and data collection

=== "Postgres1st WatchTower"

    | Dashboard                                                                                | Description |
    |------------------------------------------------------------------------------------------|-------------|
    | [Postgres1st WatchTower Inventory]                                                                          | Manage monitored services, nodes, and agents
    | [Environment Overview]                                                                   | High-level view of all monitored environments
    | [Environment Summary]                                                                    | Aggregated metrics across environments

=== "OS"

    | Dashboard                                                                                | Description |
    |------------------------------------------------------------------------------------------|-------------|
    | [CPU Utilization Details]                                                                | CPU usage, load averages, and core utilization
    | [Disk Details]                                                                           | Disk I/O, latency, and space utilization
    | [Network Details]                                                                        | Network traffic, errors, and interface statistics
    | [Memory Details]                                                                         | Memory usage, swap, and caching
    | [Node Temperature Details]                                                               | Hardware temperature monitoring
    | [Nodes Compare]                                                                          | Side-by-side comparison of multiple nodes
    | [Nodes Overview]                                                                         | Summary view of all monitored nodes
    | [Node Summary]                                                                           | System information and resource usage for a single node
    | [NUMA Details]                                                                           | NUMA node memory allocation and performance
    | [Processes Details]                                                                      | Process-level CPU, memory, and I/O metrics

=== "PostgreSQL"

    | Dashboard                                                                                | Description |
    |------------------------------------------------------------------------------------------|-------------|
    | [PostgreSQL Instances Overview]                                                          | High-level overview of all PostgreSQL instances
    | [PostgreSQL Instance Summary]                                                            | PostgreSQL instance health and performance
    | [PostgreSQL Instances Compare]                                                           | Compare metrics across PostgreSQL instances

=== "Valkey/Redis"

    | Dashboard                                                                                | Description |
    |------------------------------------------------------------------------------------------|-------------|
    | [Valkey/Redis Overview]                                                                  | Deployment health and performance summary
    | [Valkey/Redis Clients]                                                                   | Client connections and blocked clients
    | [Valkey/Redis Cluster Details]                                                           | Cluster topology and replication offsets
    | [Valkey/Redis Command Details]                                                           | Command throughput and latency patterns
    | [Valkey/Redis Load]                                                                      | Workload distribution and I/O threading
    | [Valkey/Redis Memory]                                                                    | Memory usage and eviction patterns
    | [Valkey/Redis Network]                                                                   | Network bandwidth and traffic patterns
    | [Valkey/Redis Persistence]                                                               | RDB and AOF operations
    | [Valkey/Redis Replication]                                                               | Replication lag and synchronization status
    | [Valkey/Redis Slowlog]                                                                   | Slow command identification and bottleneck detection

=== "HA"

    | Dashboard                                                                                | Description |
    |------------------------------------------------------------------------------------------|-------------|
    | [PXC/Galera Node Summary]                                                                | Individual node health in PXC/Galera clusters
    | [PXC/Galera Cluster Summary]                                                             | Cluster-wide health and replication flow
    | [PXC/Galera Nodes Compare]                                                               | Compare metrics across PXC/Galera nodes
    | [HAProxy Instance Summary]                                                               | HAProxy load balancer performance and backend health


[Advanced Data Exploration]: ../../reference/dashboards/dashboard-advanced-data-exploration.md
[Home Dashboard]: ../../reference/dashboards/dashboard-home.md
[Prometheus Exporter Status]: ../../reference/dashboards/dashboard-prometheus-exporter-status.md
[Prometheus Exporters Overview]: ../../reference/dashboards/dashboard-prometheus-exporters-overview.md
[VictoriaMetrics]: ../../reference/dashboards/dashboard-victoriametrics.md
[VictoriaMetrics Agents Overview]: ../../reference/dashboards/dashboard-victoriametrics-agents-overview.md
[Postgres1st WatchTower Inventory]: ../../use/dashboard-inventory.md
[Environment Overview]: ../../reference/dashboards/dashboard-env-overview.md
[Environment Summary]: ../../reference/dashboards/dashboard-env-overview.md
[CPU Utilization Details]: ../../reference/dashboards/dashboard-cpu-utilization-details.md
[Disk Details]: ../../reference/dashboards/dashboard-disk-details.md
[Network Details]: ../../reference/dashboards/dashboard-network-details.md
[Memory Details]: ../../reference/dashboards/dashboard-memory-details.md
[Node Temperature Details]: ../../reference/dashboards/dashboard-node-temperature-details.md
[Nodes Compare]: ../../reference/dashboards/dashboard-nodes-compare.md
[Nodes Overview]: ../../reference/dashboards/dashboard-nodes-overview.md
[Node Summary]: ../../reference/dashboards/dashboard-node-summary.md
[NUMA Details]: ../../reference/dashboards/dashboard-numa-details.md
[Processes Details]: ../../reference/dashboards/dashboard-processes-details.md
[Prometheus Exporter Status]: ../../reference/dashboards/dashboard-prometheus-exporter-status.md
[Prometheus Exporters Overview]: ../../reference/dashboards/dashboard-prometheus-exporters-overview.md
[PostgreSQL Instances Overview]: ../../reference/dashboards/dashboard-postgresql-instances-overview.md
[PostgreSQL Instance Summary]: ../../reference/dashboards/dashboard-postgresql-instance-summary.md
[PostgreSQL Instances Compare]: ../../reference/dashboards/dashboard-postgresql-instances-compare.md
[Valkey/Redis Overview]: ../../reference/dashboards/dashboard-valkey-redis-overview.md
[Valkey/Redis Clients]: ../../reference/dashboards/dashboard-valkey-redis-clients.md
[Valkey/Redis Cluster Details]: ../../reference/dashboards/dashboard-valkey-redis-cluster-details.md
[Valkey/Redis Command Details]: ../../reference/dashboards/dashboard-valkey-redis-command-detail.md
[Valkey/Redis Load]: ../../reference/dashboards/dashboard-valkey-redis-load.md
[Valkey/Redis Memory]: ../../reference/dashboards/dashboard-valkey-redis-memory.md
[Valkey/Redis Network]: ../../reference/dashboards/dashboard-valkey-redis-network.md
[Valkey/Redis Persistence]: ../../reference/dashboards/dashboard-valkey-redis-persistence-details.md
[Valkey/Redis Replication]: ../../reference/dashboards/dashboard-valkey-redis-replication.md
[Valkey/Redis Slowlog]: ../../reference/dashboards/dashboard-valkey-redis-slowlog.md
[PXC/Galera Node Summary]: ../../reference/dashboards/dashboard-pxc-galera-node-summary.md
[PXC/Galera Nodes Compare]: ../../reference/dashboards/dashboard-pxc-galera-nodes-compare.md
[HAProxy Instance Summary]: ../../reference/dashboards/