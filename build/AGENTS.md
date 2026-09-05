# PGF WatchTower Build and Packaging Guidelines

> **Parent guide**: [AGENTS.md](../AGENTS.md) — product overview, architecture, domain model, global conventions

The `/build` directory contains everything needed to build, package and distribute the
server and client.

**What actually ships is the air-gapped RPM bundle**, produced by
`scripts/build-pfw-airgap` and verified by `scripts/test-pfw-airgap`,
`scripts/test-pfw-negative-control` and `scripts/test-pfw-upgrade`. Start there.

The Docker, DEB and AMI paths below are inherited from upstream and are **not** part of
the shipped product. They have not been through the rebrand — their artifact names and
image paths are still `pmm-*` — so treat a `pmm-` name in those trees as untouched
upstream rather than as a bug to fix.

## Architecture

### Build Artifacts

| Artifact | Format | Source |
|----------|--------|--------|
| Server Docker image (inherited, pre-rename) | Docker (EL9) | `docker/server/Dockerfile.el9` |
| Client Docker image (inherited, pre-rename) | Docker (EL9) | `docker/client/Dockerfile.el9` |
| Server RPMs — **shipped** | RPM (EL9) | `packages/rpm/server/SPECS/` |
| Client RPM — **shipped** | RPM (EL9) | `packages/rpm/client/pfw-client.spec` |
| Client DEB (inherited, unused) | DEB | `packages/deb/` |
| Server AMI (inherited, unused) | AWS AMI | `packer/pmm.json` |

### Build Pipeline

```
Source code (Go, TypeScript)
  → Binary compilation (make release)
    → RPM/DEB packaging (packages/)
      → Docker image build (docker/)
        → Machine image build (packer/)
```

## Ansible Roles

### Server Provisioning Roles

| Role | Purpose |
|------|---------|
| `clickhouse` | Install and configure ClickHouse for QAN |
| `grafana` | Install Grafana, provision datasources and dashboards |
| `nginx` | Configure Nginx as reverse proxy (SSL termination, routing) |
| `postgres` | Install and configure PostgreSQL for pmm-managed |
| `supervisord` | Configure Supervisord for process management |
| `dashboards` | Provision Grafana dashboards |
| `initialization` | Server first-run setup |
| `pmm-images` | Image metadata and version info |

### Cloud-Specific Roles

| Role | Purpose |
|------|---------|
| `cloud-node` | Cloud VM preparation |
| `lvm-init` | LVM storage setup for persistent data |
| `init-admin-password-ami` | Set admin password from EC2 instance ID |
| `ami` | AMI-specific customization |

## Packer Templates

### Server Images (`packer/pmm.json`)

Builds machine images:
- **amazon-ebs** — AWS AMI

### Make Targets

```bash
make pmm-ami              # Build AWS AMI
make rpmbuild-el9         # Build RPM build environment image
```

## Patterns and Conventions

### Do
- Use Ansible roles for server provisioning — they're the single source of truth for server setup
- Keep Dockerfiles minimal — delegate to Ansible for complex provisioning
- Use multi-stage Docker builds where appropriate
- Keep RPM/DEB specs in sync with actual binary and config file paths
- Test image builds in CI before merging

### Don't
- Don't hardcode versions in Dockerfiles — use build args
- Don't modify Ansible roles without testing the full image build
- Don't add secrets or credentials to build scripts or Dockerfiles
- Don't skip the RPM build step when modifying server components

## Key Files to Reference

- `build/docker/server/Dockerfile.el9` — server Docker image definition (inherited)
- `build/docker/server/entrypoint.sh` — Server container entrypoint
- `build/ansible/pmm-docker/main.yml` — Docker provisioning playbook
- `build/ansible/roles/` — All Ansible roles for server components
- `build/packages/rpm/server/SPECS/` — RPM spec files for server components
- `build/packer/pmm.json` — Machine image definitions
- `build/scripts/` — Build scripts for all artifact types

## Air-gapped PGF WatchTower bundle

The offline bundle a customer installs on a network-isolated RHEL/Rocky 9 host is built
and tested by a separate pipeline that does NOT go through the Jenkins/`pmm-submodules`
path described above — it builds from this checkout.

- `build/scripts/README.md` — **read this first**: the pipeline, the three test suites and
  why each exists, and the traps already paid for
- `build/scripts/build-pfw-airgap` — the pipeline (resumable stages)
- `build/scripts/pfw-airgap-vars` — every input pinned to an exact commit; never branch names
- `build/packages/pfw-airgap-INSTALL.md` — the operator guide shipped with the bundle

A fresh-install test cannot observe an upgrade defect, and a green suite is not evidence
until its assertions have been shown to fail when what they guard is broken. Hence three
suites, not one. Re-run `build/scripts/test-pfw-negative-control` after changing any
assertion.

### Deferred: the container and deb paths

`docs/deferred-container-path.md` lists what the rename deliberately did not touch in
`build/docker`, `build/ansible` and `build/packages/deb`. beta2 ships a native RPM bundle,
so none of it is built or tested here. Read it before picking up the Docker builds — one
item, the distribution sentinel, already has a live consequence for the container path.

### Open: revisit the VictoriaMetrics pin

`PFW_VM_COMMIT` is `pmm-6401-v1.147.0`, which is **not an upstream release**. It is a
Percona branch tag (`pmm-6401-read-prometheus-data-files`) carrying PMM-specific work for
reading Prometheus data files, and it has diverged from upstream: checked 2026-09-03, 217
commits ahead of v1.151.0 and 467 behind. Latest upstream at that date was **v1.151.0**.

Deliberately left alone for now. Moving to a plain upstream tag would drop that feature,
so it is a functional decision rather than a version bump — and it needs someone to
establish whether PGF WatchTower actually depends on reading Prometheus data files.

Now that we build this component from source we are no longer bound to Percona's tag,
which makes the move *possible*; that is what changed, not the risk.

### Done: the last two components now build from source

Four server components used to be fetched prebuilt from Percona's S3 build cache. Two of
them, `pfw-qan-api2` and `vmproxy`, are now built here — their source is in this repo and
they take seconds. **`pfw-victoriametrics` and `pmm-dump` are still fetched, and that
is the only thing blocking a complete aarch64 bundle.**

Why it has to change rather than being re-pinned again:

- **There is no aarch64 in that cache at all.** Verified 2026-09-02 by a paginated listing
  of the whole bucket: 228 keys under `RELEASE/`, none containing `aarch64`, and the
  `RELEASE/el9-aarch64/` tree the old pins referenced is gone. `repo.percona.com` is not a
  fallback — it serves `pmm-client` for both arches but no server-side components, and
  there is no `pmm3-server` repo.
- **The cache is rotated, not archival.** Every one of the eight original pins had expired,
  x86_64 included, so the bundle was not reproducible on either architecture. Expect the
  current pins to expire too; list `RELEASE/el9/` and re-pin rather than assuming the build
  is broken.
- **Their RPMs carry Percona's branding**, and our rebranded specs for them are dormant, so
  the rebrand never reaches the artifact. `pfw-qan-api2` only started reporting "for PGF
  WatchTower" once we built it ourselves.

What the work involves. Both have from-source specs in `build/packages/rpm/server/SPECS/`
already. Unlike the first two, their source is in other repos, so a stage must fetch a
pinned tarball first — the shape `stage_exporters` already uses for the exporters. Then
they emit and build like any other spec. Budget for the same four snags the first two hit,
all of which failed before they worked:

1. add them to the build; the existing `cp -a bin` stages whatever lands in `bin/`
2. `emit()` their specs, checking each one's placeholder version — qan-api2's was
   `full_pmm_version 3.0.0`, not the `2.0.0` that `monorepo()` rewrites
3. stage the `LICENSE` and `README.md` each spec puts in `%license`/`%doc`, or rpmbuild
   fails on the glob
4. inject `%global debug_package %{nil}` — with `%build` a no-op there are no sources to
   extract and rpmbuild dies on an empty `debugsourcefiles.list`

Note `victoriametrics.spec` says 1.147.0 and pins the Percona tag `pmm-6401-v1.147.0`
while we fetch the 1.149.0 RPM. Bumping it needs that tag to exist upstream — not a
version-field edit, or `Source0` 404s.

Until this lands, `require_s3_components` fails the `s3`/`closure`/`bundle` stages on
aarch64 with an explanation. The binary and grafana stages still run, so aarch64 remains
usable for development.
