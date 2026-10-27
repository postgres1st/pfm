# About PGF WatchTower commands

PGF WatchTower provides two command-line tools for managing your monitoring setup from the terminal. 

Use these tools to add databases, configure agents, check status, and troubleshoot issues without leaving the command line.

You can also perform most of these tasks through the [PGF WatchTower web interface](../../reference/ui/ui_components.md) or the [PGF WatchTower API](../../api/index.md).

## Command-line tools

`pfw-admin`: The primary CLI tool for administering PGF WatchTower. Use it to add and remove database services, check connection status, list monitored services, modify agent configurations, create diagnostic archives, and annotate dashboards. Communicates directly with PGF WatchTower Server.

    `pfw-admin` is installed automatically as part of the [PGF WatchTower Agent](../../install-pmm/install-pmm-client/index.md) package.

    See [`pfw-admin` reference](pfw-admin/pfw-admin.md) for syntax, common flags, and links to all subcommands.

`pfw-agent`: The daemon process that runs on each monitored host. It manages exporters and agents locally, coordinating data collection and communication between PGF WatchTower Agent and PGF WatchTower Server. 

    You typically don't interact with pfw-agent directly, `pfw-admin` communicates with PGF WatchTower Server, which then sends commands to `pfw-agent`. See [Coordinate monitoring agents with pfw-agent](pfw-agent.md) for configuration options and startup flags.

## Next steps
 
- [Get started with `pfw-admin`](../commands/pfw-admin/pfw-admin.md)
- [Add databases to monitoring](pfw-admin/add.md)
- [Manage inventory and modify agents](pfw-admin/inventory.md)
- [Configure, register, and remove services](pfw-admin/config.md)
- [Check status and troubleshoot](pfw-admin/status.md)
