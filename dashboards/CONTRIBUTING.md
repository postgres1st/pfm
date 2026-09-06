# Contributing to Postgres1st WatchTower dashboards

This directory holds the Grafana dashboards Postgres1st WatchTower provisions, and the `pmm-app` Grafana
plugin that bundles them along with the Query Analytics panel.

See the [repository root CONTRIBUTING.md](../CONTRIBUTING.md) for the general contribution
guide, and [README.md](README.md) for what is in each dashboard folder.

## Reporting a problem

Open an issue on [postgres1st/pfm](https://github.com/postgres1st/pfm). Useful reports are:

- _Reproducible._ Include steps to reproduce the problem.
- _Specific._ Include as much detail as possible: which version, what environment, etc.
- _Unique._ Do not duplicate existing tickets.
- _Scoped to a single bug._ One bug per report.

## Setting up a local development environment

The devcontainer ships Node 22 + Yarn and wires the QAN plugin straight into the bundled
Grafana. From the repo root **on the host**:

```bash
make env-up      # first run only; reuses the container afterwards
make env         # shell into the container
```

Then **inside the container**:

```bash
make run-qan-ui
```

`run-qan-ui` runs `yarn install` under `dashboards/pmm-app`, symlinks
`dashboards/pmm-app/dist` into `/srv/grafana/plugins/pmm-app/dist`, injects the livereload
snippet into Grafana's `index.html` via `setup-livereload`, and starts the webpack watcher.

Open `https://localhost/graph/d/pmm-qan/` — the livereload server on port `35730` reloads
the page whenever webpack finishes a rebuild.

For a one-off build without the watcher, use `make build-qan-ui`.

## Editing dashboards

Dashboard JSON lives under [`dashboards/`](dashboards), grouped by folder. Edit those
sources — **not** `pmm-app/dist/`, which is build output and is overwritten.

Only PostgreSQL, HAProxy and the generic host and insight dashboards are present. The
MySQL, MongoDB, ProxySQL, PXC and Valkey/Redis sets were removed because the service-type
allowlist (`managed/models/service_type_allowlist.go`) rejects those services, so they
could never populate. Adding a dashboard for a database this build does not monitor is not
a useful contribution.

## Submitting a pull request

1. Fork the repository and clone your fork.
2. Create a branch named for what it does — for example `dashboards-postgres-vacuum-tooltips`.
3. Make your changes, keeping the branch scoped to one topic.
4. Write a commit message that explains *why*, not just what.
5. Push the branch and open a pull request against `main` on
   [postgres1st/pfm](https://github.com/postgres1st/pfm).

Before opening it, check that the dashboard still loads in Grafana and that any panel you
touched renders with real data — a JSON change that parses is not the same as one that
works.
