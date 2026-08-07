# pfm-agent

The PFMM client agent. It runs on each monitored host, supervises the exporters and
relays their metrics to the PFMM server.

# Contributing notes

## Pre-requisites:
git, make, curl, go, gcc, docker, docker-compose, a running PFMM server

## Local setup
Install one or more exporters:
* node_exporter
* postgres_exporter
* rds_exporter
* azure_metrics_exporter

These are the exporters PFMM builds and ships. The MySQL, MongoDB and ProxySQL exporters
upstream carries are not built here — the service-type allowlist
(`managed/models/service_type_allowlist.go`) rejects those services, so nothing would
drive them.

#### To run pfm-agent
- Run a PFMM server, or [pfm-managed](../managed) directly.
- Run `make setup-dev` to configure pfm-agent
- Run `make run` to run pfm-agent


## Testing
Run `make env-up` to set-up environment.
Run `make test` to run tests.

## Code style
Before making PR, please run `make check-all` locally to run all checkers and linters.
