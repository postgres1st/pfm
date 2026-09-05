# MySQL SQL mode not fitting best practice

## Description

In order for the server to check data integrity, the sql_mode should have TRADITIONAL, STRICT_ALL_TABLES, and STRICT_TRANS_TABLES set in sql_mode.

The advisors raise an alert if one or more of them are missing.

## Resolution

Set sql_mode in a way that it contains TRADITIONAL, STRICT_ALL_TABLES and STRICT_TRANS_TABLES.
