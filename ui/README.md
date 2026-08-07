# Postgres1st Monitoring and Management UI

The web interface for PFMM — a PostgreSQL-only fork of [Percona Monitoring and
Management](https://github.com/percona/pmm). It is a React application served through
Grafana by a companion plugin.

See the [repository root README](../README.md) for what PFMM is and how it is installed.

## Stack

This repo uses the following stack across its packages:

- Yarn (https://yarnpkg.com/)
- Turborepo (https://turborepo.com/)
- Typescript (https://www.typescriptlang.org/)
- React (https://react.dev/)
- Rollup to bundle the different common packages (https://rollupjs.org/)
- Vite for development (https://vitejs.dev/)
- Vitest for unit tests (https://vitest.dev/)

## Apps

- **pmm** — the main UI application
- **pmm-compat** — Grafana plugin that handles communication between Grafana and the UI

Both directories keep their `pmm-` names: they are referenced by the build, by Grafana's
plugin directory layout and by nginx paths, so renaming them is a coordinated change
rather than a rename in this file.

## Packages

- **shared** — common code between applications

## Run in the devcontainer (recommended)

The devcontainer (see the root `CONTRIBUTING.md`) ships Node 22 + Yarn and a Vite dev server that runs end-to-end with the rest of the server. From the repo root **on the host**:

```bash
make env-up      # first run only; reuses the container afterwards
make env         # shell into the container
```

Then **inside the container**:

```bash
make run-ui
```

`run-ui` installs UI dependencies, symlinks the `pmm-compat` plugin into Grafana's plugin directory, injects livereload into Grafana's `index.html` (`setup-livereload`), and starts Vite on port `5173`.

Open `https://localhost/` — Grafana loads the `pmm-compat` plugin, which fetches the main UI from the Vite dev server. Edits under `ui/apps/pmm/src/` hot-reload in the browser without a full page refresh.

Notes:

- The Vite port is configurable via `PMM_PORT_VITE` in your `.env` (see `.env.dev.example`); it defaults to `5173`.
- `run-ui` installs an EXIT trap that restores the original `pmm-compat-app` plugin and restarts Grafana when you Ctrl-C. Don't kill the container mid-run, or the restore is skipped.
- For a one-shot build deployed into the container's system paths, use `make build-ui` instead.

### Update Grafana in the devcontainer

The devcontainer ships a prebuilt Grafana baked into the dev image named by `PMM_SERVER_IMAGE` in your `.env` (upstream's `perconalab/pmm-server:3-dev-container` by default). To develop against a local Grafana fork instead — PFMM's is [postgres1st/grafana](https://github.com/postgres1st/grafana) — mount your checkout into the container and rebuild it:

1. Clone the Grafana fork **next to** the `pmm` repo on the host, so it resolves to `../grafana` from the repo root:

   ```bash
   git clone https://github.com/postgres1st/grafana ../grafana
   ```

2. Uncomment the `grafana` volume mappings in `docker-compose.dev.yml`:

   ```yaml
   # grafana
   - ../grafana:/root/go/src/github.com/percona/grafana
   - ../grafana/public:/usr/share/grafana/public
   ```

   The first mount provides the Grafana source for the backend build; the second serves the fork's built frontend (`public/`). The in-container path keeps `github.com/percona/grafana` because that is still the fork's Go module path.

3. Recreate the container so the new mounts take effect — volume mappings are read at container create time (`make env-down` then `make env-up`, or recreate via your container tooling).

4. Rebuild the Grafana backend **inside the container**:

   ```bash
   make grafana-be-build
   ```

   This runs `make build-go` in `/root/go/src/github.com/percona/grafana`, copies the resulting `bin/linux/amd64/grafana` binary to `/usr/sbin/grafana`, and restarts Grafana via supervisor.

For frontend changes in the fork, rebuild its `public/` assets (`make build-js` inside the grafana checkout); they are served through the `../grafana/public` mount.

## Run locally on the host

Use this when you want to drive Vite from your IDE without `make env`. You still need a reachable server — the simplest way is to leave the devcontainer running (`make env-up`) so its ports are exposed; any other PFMM server reachable at `https://localhost:8443` works too.

Prerequisites:

- [Node 22](https://nodejs.org/en) (e.g. via [nvm](https://github.com/nvm-sh/nvm))
- [Yarn](https://yarnpkg.com/) 1.x

```bash
make setup       # yarn install across the workspace
make dev         # turbo dev → Vite on https://localhost:5174 (or 5173 if nginx certs are present)
```

Vite proxies `/v1`, `/graph`, and `/logs.zip` to the server (inside the devcontainer: `https://localhost:8443`; on the host when using the devcontainer-exposed ports: `https://localhost`) — see `apps/pmm/vite.config.ts`.

## Build for production

```bash
make build
```

## Other targets

```bash
make test        # vitest across the workspace
make lint
make format
```
