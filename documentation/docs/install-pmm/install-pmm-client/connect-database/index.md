# Connect databases to PFMM
Postgres1st (PFMM) supports monitoring for PostgreSQL and various cloud database services. 

## Supported database technologies

- [PostgreSQL](postgresql.md)
- [Amazon RDS](aws.md)
- [Microsoft Azure](azure.md)
- [Google Cloud Platform](google.md) (PostgreSQL)
- [Linux](linux.md)
- [External services](external.md)
- [HAProxy](haproxy.md)
- [Remote instances](remote.md)

| Database type                                | Local monitoring | Remote monitoring | Query Analytics (QAN) | Performance schema | Backup integration |
|----------------------------------------------|------------------|-------------------|------------------|---------------------|---------------------|
| [PostgreSQL](../connect-database/postgresql.md)          | <span style="color:green">✔</span> | <span style="color:green">✔</span> | <span style="color:green">✔</span> | <span style="color:green">✔</span> | <span style="color:red">✘</span> |
| [AWS RDS/Aurora](../connect-database/aws.md)             | <span style="color:red">✘</span>  | <span style="color:green">✔</span> | <span style="color:green">✔</span> | <span style="color:green">✔</span> | <span style="color:red">✘</span> |
| [Azure Database](../connect-database/azure.md)           | <span style="color:red">✘</span>  | <span style="color:green">✔</span> | <span style="color:green">✔</span> | <span style="color:green">✔</span> | <span style="color:red">✘</span> |
| [Google Cloud SQL](../connect-database/google.md)        | <span style="color:red">✘</span>  | <span style="color:green">✔</span> | <span style="color:green">✔</span> | <span style="color:green">✔</span> | <span style="color:red">✘</span> |
| [HAProxy](../connect-database/haproxy.md)                | <span style="color:green">✔</span> | <span style="color:green">✔</span> | <span style="color:red">✘</span>  | <span style="color:red">✘</span>  | <span style="color:red">✘</span> |

## Modify existing services

To change the parameters of a previously-added service, remove the service and re-add it with the new parameters.

## New to PFMM?
If you're setting up monitoring for the first time, follow the installation and setup instructions in the [PFMM installation overview](../../index.md).
