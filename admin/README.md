# pfm-admin

The command-line client for PFMM. It registers nodes and PostgreSQL services with a PFMM
server and reports agent status.

# Contributing notes

## Pre-requirements:
git, make, curl, go, gcc, a running PFMM server, pfm-agent

## Local setup
### To run pfm-admin commands
- Run a PFMM server, or [pfm-managed](../managed) directly.
- Run pfm-agent: `cd ../agent`.
- Run pfm-admin commands:
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
	pmm-admin version: 3.9.0
	pmm-agent version: 3.9.0
Agents:
	3329a405-8a5d-4414-9890-b6ae4209e0cc NODE_EXPORTER                  RUNNING        40001
```

The `PMM Server:` / `PMM Client:` headings and the `pmm-admin`/`pmm-agent` version labels
come from the status template in `commands/status.go` and have not been rebranded yet —
the sample above matches what the binary prints today, not what it will print after the
rename.

It means that everything works.

## Testing
pfm-admin doesn't require setting-up an environment.
Run `make test` to run tests.
