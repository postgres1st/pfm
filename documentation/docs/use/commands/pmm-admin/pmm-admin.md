# pfw-admin command overview

`pfw-admin` is the command-line tool for managing your PGF WatchTower monitoring setup. Use it to add databases, check connection status, update agent configurations, and troubleshoot issues from your terminal.

`pfw-admin` is installed automatically with PMM Client.

To add services through the UI instead, see [Connect databases via the web interface](../../../install-pmm/install-pmm-client/connect-database/index.md). For programmatic access, see the [PGF WatchTower API](../../../api/index.md).

Use `pfw-admin` to:

- add PostgreSQL services to monitoring
- check connection status between PMM Client and PMM Server
- list monitored services and their agents
- modify agent configurations without removing services
- create diagnostic archives for troubleshooting

## Syntax

Run `pfw-admin` commands in this format:

```bash
pfw-admin COMMAND [COMMAND-FLAGS] [ARGUMENTS] [FLAGS]
```

## Quick start

Try these common commands to verify your setup and start monitoring:

### Check PMM Client status

```bash
pfw-admin status
```

### Add a MySQL database

```bash
pfw-admin add mysql mysql-prod 192.168.1.10:3306 --username=pmm --password=pass
```

### Add a MongoDB database

```bash
pfw-admin add mongodb mongodb-prod 192.168.1.20:27017 --username=pmm --password=pass
```

### List monitored services

```bash
pfw-admin list
```

### Create diagnostic archive

```bash
pfw-admin summary
```

For complete options and flags, see [Add database services](../pmm-admin/add.md), [Manage inventory](../pmm-admin/inventory.md), [Configuration commands](../pmm-admin/config.md), and [Status and diagnostics](../pmm-admin/status.md).

## Command reference

Find all available commands for managing monitored services:

| Command | Description | Documentation |
|---------|-------------|---------------|
| `pfw-admin add` | Add database services to monitoring | [Add database services](../pmm-admin/add.md) |
| `pfw-admin inventory` | List and modify agents and services | [Manage inventory](../pmm-admin/inventory.md) |
| `pfw-admin config` | Configure local pfw-agent | [Configuration commands](../pmm-admin/config.md) |
| `pfw-admin register` | Register node with PMM Server | [Configuration commands](../pmm-admin/config.md) |
| `pfw-admin remove` | Remove service from monitoring | [Configuration commands](../pmm-admin/config.md) |
| `pfw-admin annotate` | Add event annotations | [Configuration commands](../pmm-admin/config.md) |
| `pfw-admin status` | Show PMM Client status | [Status and diagnostics](../pmm-admin/status.md) |
| `pfw-admin list` | List monitored services | [Status and diagnostics](../pmm-admin/status.md) |
| `pfw-admin summary` | Create diagnostic archive | [Status and diagnostics](../pmm-admin/status.md) |

### Add and remove services

- [`pfw-admin add`](../pmm-admin/add.md): Add database services to monitoring
- [`pfw-admin remove`](../pmm-admin/config.md): Remove service from monitoring

### Manage inventory

- [`pfw-admin inventory`](../pmm-admin/inventory.md): List and modify agents and services

### Configure and register

- [`pfw-admin config`](../pmm-admin/config.md): Configure local pfw-agent
- [`pfw-admin register`](../pmm-admin/config.md): Register node with PMM Server
- [`pfw-admin annotate`](../pmm-admin/config.md): Add event annotations

### Status and troubleshooting

- [`pfw-admin status`](../pmm-admin/status.md): Show PMM Client status
- [`pfw-admin list`](../pmm-admin/status.md): List monitored services
- [`pfw-admin summary`](../pmm-admin/status.md): Create diagnostic archive

## Get help

Run `--help` with any command to see available flags and usage:

```bash
pfw-admin COMMAND --help
```

For example:

```bash
pfw-admin add mysql --help
pfw-admin inventory change agent --help
```

## See also

- [PMM Client agent](../pmm-agent.md)
- [Connect databases to PGF WatchTower](../../../install-pmm/install-pmm-client/connect-database/index.md)
- [Remove databases from monitoring](../../remove-services.md)