# Remove services from monitoring

To stop monitoring a service, use the `pfw-admin remove` command with the appropriate service type and name:

```sh
pfw-admin remove <service-type> <service-name>
```
## Command reference
- `service-type`: The type of service to remove: mysql, mongodb, postgresql, proxysql, haproxy, or external
- `service-name`: The name of the service as displayed in Postgres1st WatchTower inventory

## Example
To remove a MySQL service:
```sh
pfw-admin remove mysql mysql-prod-db1
```

## Verify service removal
After removing a service, you can verify it's no longer being monitored by listing all monitored services:

```sh
pfw-admin list
```

## Related topics
- [Percona release](https://www.percona.com/doc/percona-repo-config/percona-release.html)
- [PMM Client architecture](../reference/index.md#pmm-client)