# Contributing notes

## Pre-requirements
git, make, curl, go, docker

Dependencies are Go modules; there is nothing to install first.

## Local setup

#### To run qan-api2
Run `make env-up` to start ClickHouse and the supporting containers.
Run `make run` to start qan-api2.

`make env-down` tears the containers down again.

## Run as part of a server container

Run `PMM_CONTAINER=<container-name> make release deploy` to deploy a locally built
qan-api2 into a running server container. `PMM_CONTAINER` defaults to `pmm-server`; it
keeps that name because the Makefile does, and the binary is still deployed to
`/usr/sbin/percona-qan-api2` inside the container.

## Testing
Run `make test-env-up` to set up the test environment (ClickHouse plus test data).
Run `make test` to run tests.
Run `make test-env-down` when finished.

`make help` lists every target.
