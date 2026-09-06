# pfw-admin

The command-line client for Postgres1st WatchTower. It registers nodes and PostgreSQL services with a Postgres1st WatchTower
server and reports agent status.

# Contributing notes

## Pre-requirements:
git, make, curl, go, gcc, a running Postgres1st WatchTower server, pfw-agent

## Local setup
### To run pfw-admin commands
- Run a Postgres1st WatchTower server, or [pfw-managed](../managed) directly.
- Run pfw-agent: `cd ../agent`.
- Run pfw-admin commands:
    ```shell script
    go run main.go status
    ```

You should see something like this:
 ```
Agent ID : fcbe3cb4-a95a-43f4-aef5-c3494caa5132
Node ID  : 77be6b4d-a1d9-4687-8fae-7acbaee7db47
Node name: postgres-server-test-1

PMM Server:
	URL    : https://127.0.0.1:443/
	Version: 3.9.0-HEAD-fcde194

PMM Client:
	Connected        : true
	Time drift       : 41.93µs
	Latency          : 211.026µs
	Connection uptime: 100
	pfw-admin version: 3.9.0
	pfw-agent version: 3.9.0
Agents:
	3329a405-8a5d-4414-9890-b6ae4209e0cc NODE_EXPORTER                  RUNNING        40001
```

The sample above matches what the binary prints today. The `pfw-admin`/`pfw-agent`
version labels have been rebranded; the `PMM Server:` / `PMM Client:` headings have
**not** — they come from the status template in `commands/status.go` and are part of
the CLI-text work still outstanding (see `docs/pfw-identifier-map.md`, A.5).

It means that everything works.

## Testing
pfw-admin doesn't require setting-up an environment.
Run `make test` to run tests.
