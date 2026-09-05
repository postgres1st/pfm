# MySQL binlog checksum not set

## Description

When archiving or backing up your binary logs, binary log checksums adds resilience to integrity checking and validity.

For more information, see [binlog_checksum in the MySQL documentation](https://dev.mysql.com/doc/refman/8.4/en/replication-options-binary-log.html#sysvar_binlog_checksum).  

## Resolution

In the server, set **binlog_checksum=CRC32** to improve consistency and reliability. The CRC32 checksum is the only checksum supported and is the default.

`SET GLOBAL binlog_checksum=CRC32;`

Resetting the variable, even to the existing value, forces a binary log rotation.
