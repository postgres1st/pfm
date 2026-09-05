# MySQL Replica node is not configured as READ-ONLY

## Description

To prevent accidental writes that may lead to data inconsistency, a replica node must have the READ-ONLY flag active.

The current node has a READ-ONLY value of 0 and is at high risk.

## Resolution

Set the value of READ-ONLY to 1, to prevent writes on this node.
**SET GLOBAL READ-ONLY=1;**
