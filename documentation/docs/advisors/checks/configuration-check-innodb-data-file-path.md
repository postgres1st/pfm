# MySQL InnoDB tablespace has a max cap and cannot auto-extend

## Description

Some InnoDB Tablespaces specify a maximum size limit, which means that the file size can not exceed that limit.

Reaching this limit could cause production problems.

For more information, see [The System Tablespace in the MySQL documentation](https://dev.mysql.com/doc/refman/8.4/en/innodb-system-tablespace.html).


## Resolution

In most cases, we do not recommend having any maximum size limit on InnoDB Tablespaces.
