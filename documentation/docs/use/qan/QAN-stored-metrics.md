# QAN Stored metrics

Stored metrics captures queries after they complete, so you can review historical performance, find slow queries, and track optimization progress over time.

## Supported databases

Stored metrics supports PostgreSQL with the following requirements:

=== "PostgreSQL"
    - PostgreSQL 11 or later
    - `pg_stat_monitor` extension (recommended) or `pg_stat_statements` extension
    - Appropriate `shared_preload_libraries` configuration
    - Superuser privileges for Postgres1st WatchTower monitoring account

## Dashboard layout

The Stored metrics view contains three panels:

- [Filters panel](panels/filters.md): narrow results by database, service, or query type
- [Overview panel](panels/overview.md): see query metrics and trends
- [Details panel](panels/details.md): examine individual query performance

## Data collection

Stored metrics collects data once per minute. When collection delays occur, gaps may appear in the sparkline.

## Monitor PMM Server's internal PostgreSQL

By default, Query Analytics hides queries from PMM Server's internal PostgreSQL database. This keeps the focus on your monitored databases.

Enable this when you need to troubleshoot PMM Server performance, check resource usage, or ensure applications are not accidentally using the default `postgres` database. This is particularly useful in [High Availability (HA) deployments](../../install-pmm/HA.md).

To enable:
{.power-number}

1. Go to **Configuration > Settings > Advanced settings**.
2. Switch on the **QAN for PMM Server** option.
3. Open **Query Analytics** and filter by `watchtower-postgresql` to view queries.

When enabled, you'll see queries related to Postgres1st WatchTower's internal operations—inventory, settings, advisor checks, alerts, backups, and authentication. These are usually lightweight, but unusual spikes may indicate performance issues.

!!! warning
    Do not use PMM Server's PostgreSQL database for application workloads. Use dedicated databases for your applications.

## See also

- [Filters panel](panels/filters.md)
- [Overview panel](panels/overview.md)
- [Details panel](panels/details.md)