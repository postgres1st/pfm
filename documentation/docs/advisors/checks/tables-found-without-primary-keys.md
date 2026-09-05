# Found tables without primary keys

## Description

Checks if there are any InnoDB tables without a Primary Key. For more information about InnoDB Primary Keys, see the [Tuning InnoDB Primary Keys blogpost](https://www.percona.com/blog/tuning-innodb-primary-keys/). 

## Resolution

Consider adding a sequential Primary Key to your tables.
