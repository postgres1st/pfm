# MongoDB replica set topology

## Description
This advisor warns if the Replica Set cluster has less than three members.

## Resolution
Add at least another data-bearing member. This kind of topology ensures High Availability and resilience in case of network partitioning (“split-brain” condition).
