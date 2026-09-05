# About PGF WatchTower commands

PGF WatchTower provides two command-line tools for managing your monitoring setup from the terminal. 

Use these tools to add databases, configure agents, check status, and troubleshoot issues without leaving the command line.

You can also perform most of these tasks through the [PGF WatchTower web interface](../../reference/ui/ui_components.md) or the [PGF WatchTower API](../../api/index.md).

## Command-line tools

`pfw-admin`: The primary CLI tool for administering PGF WatchTower. Use it to add and remove database services, check connection status, list monitored services, modify agent configurations, create diagnostic archives, and annotate dashboards. Communicates directly with PMM Server.

    `pfw-admin` is installed automatically as part of the [PMM Client](../../../install-pmm/install-pmm-client/index.md) package.

    See [`pfw-admin` reference](pmm-admin/pmm-admin.md) for syntax, common flags, and links to all subcommands.

`pfw-agent`: The daemon process that runs on each monitored host. It manages exporters and agents locally, coordinating data collection and communication between PMM Client and PMM Server. 

    You typically don't interact with pfw-agent directly, `pfw-admin` communicates with PMM Server, which then sends commands to `pfw-agent`. See [Coordinate monitoring agents with pfw-agent](pmm-agent.md) for configuration options and startup flags.

## Next steps
 
- [Get started with `pfw-admin`](../commands/pmm-admin/pmm-admin.md)
- [Add databases to monitoring](pmm-admin/add.md)
- [Manage inventory and modify agents](pmm-admin/inventory.md)
- [Configure, register, and remove services](pmm-admin/config.md)
- [Check status and troubleshoot](pmm-admin/status.md)
