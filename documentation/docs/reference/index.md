# PGF WatchTower Architecture

PGF WatchTower is a client/server application built by Percona comprising its own and third-party components and tools.

<!-- The source of this image is maintained at https://miro.com/app/board/uXjVOPgKgrE=/ -->

![Client-Server Architecture](../images/arch/C_S_Architecture.jpg)

## PGF WatchTower Server

PGF WatchTower Server is the heart of PGF WatchTower. It receives data from clients, collects it, and stores it. Metrics are drawn as tables, charts and graphs within [_dashboards_](../use/dashboards-panels/index.md), each a part of the web-based [user interface](../reference/ui/ui_components.md).

## PGF WatchTower Agent

PGF WatchTower Agent is a collection of agents and exporters that run on the host being monitored.

PGF WatchTower Agent runs on every database host or node you want to monitor. The client collects server metrics, general system metrics, query analytics and sends it to the server. Except when monitoring AWS RDS instances, a PGF WatchTower Agent must be running on the host to be monitored.

## PGF WatchTower context

The PGF WatchTower Agent package provides:

- Exporters for each database and service type. When an exporter runs, it connects to the database or service instance, runs the metrics collection routines, and sends the results to PGF WatchTower Server.
- `pfw-agent`: Run as a daemon process, it starts and stops exporters when instructed.
- `vmagent`: A VictoriaMetrics daemon process that sends metrics data (_pushes_) to PGF WatchTower Server.

The PGF WatchTower Server package provides:

- `pmm-managed`
- Query Analytics
- Grafana
- VictoriaMetrics

### PGF WatchTower Server

<!-- The source of this image is maintained at https://miro.com/app/board/uXjVOPgKgrE=/ -->

![PGF WatchTower Server](../images/arch/PMM-Server-Component-Based-View.jpg)

PGF WatchTower Server includes the following tools:

- Query Analytics (QAN) enables you to analyze database query performance over periods of time. In addition to the client-side QAN agent, it includes the following:

    - QAN API is the back-end for storing and accessing query data collected by the QAN agent running on a PGF WatchTower Agent.
    - QAN App is a web application for visualizing collected Query Analytics data, which is part of the PGF WatchTower Server's UI.


- Metrics Monitoring provides a historical view of and analysis of metrics that are critical to PostgreSQL server instances. It includes the following:

  - [VictoriaMetrics](https://github.com/VictoriaMetrics/VictoriaMetrics) is a scalable time-series database. 
  - [ClickHouse](https://clickhouse.com) is a third-party column-oriented database that facilitates the Query Analytics functionality.
  - [Grafana](http://docs.grafana.org) is a third-party dashboard and graph engine for visualizing data aggregated in an intuitive web interface.
  - [PGF WatchTower Dashboards](https://github.com/percona/pmm/tree/main/dashboards) is a set of monitoring dashboards developed by Percona.

### PGF WatchTower Agent

![PGF WatchTower Agent](../images/arch/PMM-Client-Component-Based-View.jpg)

The PGF WatchTower Agent package consists of the following:

- `pfw-admin` is a command-line tool for managing PGF WatchTower Agent, for example, adding and removing database instances that you want to monitor. For more information, see [pfw-admin command overview](../use/commands/pfw-admin/pfw-admin.md).

- `pfw-agent` is a client-side component of a minimal command-line interface, which is a central entry point in charge of bringing the client functionality: it carries on client’s authentication, gets the client configuration stored on the PGF WatchTower Server, manages exporters and other agents.

- `node_exporter` is an exporter that collects general system metrics.

- `mysqld_exporter` is an exporter that collects MySQL server metrics.

- `mongodb_exporter` is an exporter that collects MongoDB server metrics.

- `postgres_exporter` is an exporter that collects PostgreSQL performance metrics.

- `valkey_exporter` is an exporter that collects Valkey and Redis performance metrics.

- `proxysql_exporter` is an exporter that collects ProxySQL performance metrics.

- `rds_exporter` is an exporter that collects Amazon RDS performance metrics.

- `azure_database_exporter` is an exporter that collects Azure database performance metrics.

To make data transfer from PGF WatchTower Agent to PGF WatchTower Server secure, all exporters are able to use SSL/TLS encrypted connections, and their communication with PGF WatchTower Server is protected by the HTTP basic authentication.

<!-- The source of this image is maintained at https://miro.com/app/board/uXjVOPgKgrE=/ -->

![Client Server Interactions](../images/arch/C_S_Interactions.jpg)
