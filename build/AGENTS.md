# PFMM Build and Packaging Guidelines

> **Parent guide**: [AGENTS.md](../AGENTS.md) — product overview, architecture, domain model, global conventions

The `/build` directory contains everything needed to build, package and distribute the
server and client.

**What actually ships is the air-gapped RPM bundle**, produced by
`scripts/build-pfmm-airgap` and verified by `scripts/test-pfmm-airgap`,
`scripts/test-pfmm-negative-control` and `scripts/test-pfmm-upgrade`. Start there.

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
| Client RPM — **shipped** | RPM (EL9) | `packages/rpm/client/pfm-client.spec` |
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

## Air-gapped PFMM bundle

The offline bundle a customer installs on a network-isolated RHEL/Rocky 9 host is built
and tested by a separate pipeline that does NOT go through the Jenkins/`pmm-submodules`
path described above — it builds from this checkout.

- `build/scripts/README.md` — **read this first**: the pipeline, the three test suites and
  why each exists, and the traps already paid for
- `build/scripts/build-pfmm-airgap` — the pipeline (resumable stages)
- `build/scripts/pfmm-airgap-vars` — every input pinned to an exact commit; never branch names
- `build/packages/pfmm-airgap-INSTALL.md` — the operator guide shipped with the bundle

A fresh-install test cannot observe an upgrade defect, and a green suite is not evidence
until its assertions have been shown to fail when what they guard is broken. Hence three
suites, not one. Re-run `build/scripts/test-pfmm-negative-control` after changing any
assertion.
