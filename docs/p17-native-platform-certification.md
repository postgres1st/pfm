# P17 — Supported-platform declaration: native certification matrix

Roadmap outcome (as written): "State exactly which operating systems and
architectures are certified for first GA. Do not advertise unvalidated
combinations as supported."

This is the record of actually running install + first-login validation
against each OS/architecture combination this product claims to support
(`RHEL 9 / Rocky Linux 9 / AlmaLinux 9`, per `README.md` and
`build/packages/pfw-airgap-INSTALL.md`). The certification bar below is the
six-outcome checklist from [`docs/p7-first-login-verification.md`](p7-first-login-verification.md)
(first-login instructions, install validation, initial authentication,
service startup, credential storage, agent registration) — that is what the
roadmap outcome actually asks for. Leave a row `Not yet run` rather than
guessing, since an un-run row and a passing row must stay visually distinct.

Now that every claimed OS/arch combination below is either CERTIFIED or an
explicitly accepted known limitation,
`documentation/docs/install-pmm/install-pmm-server/prerequisites.md` and
`.../install-pmm-client/prerequisites.md` can be rewritten to state this
supported list — that rewrite is still **not** part of this document. Those
pages currently claim Debian/Ubuntu/Oracle Linux/Amazon Linux/Docker/Podman/
Kubernetes/ARM64-via-emulation, none of which are true for this product.

---

## Certification matrix

Bundle tested: `PFW_3.9.0_BETA2` (or the current release at time of test — record which).

| OS | Arch | Status | Evidence |
|---|---|---|---|
| RHEL 9 | x86_64 | ✅ CERTIFIED | [postgres1st/pfm#78](https://github.com/postgres1st/pfm/pull/78) |
| RHEL 9 | aarch64 | ✅ CERTIFIED | [postgres1st/pfm#78](https://github.com/postgres1st/pfm/pull/78) |
| Rocky Linux 9 | x86_64 | ✅ CERTIFIED | [postgres1st/pfm#78](https://github.com/postgres1st/pfm/pull/78) |
| Rocky Linux 9 | aarch64 | ✅ CERTIFIED | [postgres1st/pfm#78](https://github.com/postgres1st/pfm/pull/78) |
| AlmaLinux 9 | x86_64 | ✅ CERTIFIED | [postgres1st/pfm#78](https://github.com/postgres1st/pfm/pull/78) |
| AlmaLinux 9 | aarch64 | ✅ CERTIFIED | [postgres1st/pfm#78](https://github.com/postgres1st/pfm/pull/78) |
| Ubuntu 24.04 | x86_64 / aarch64 | ❌ Not supported | [postgres1st/pfm#78](https://github.com/postgres1st/pfm/pull/78) |
| Debian | x86_64 / aarch64 | ❌ Not supported | [postgres1st/pfm#78](https://github.com/postgres1st/pfm/pull/78) |
| SLES 15 SP7 | x86_64 / aarch64 | ❌ Not supported — install fails | [postgres1st/pfm#78](https://github.com/postgres1st/pfm/pull/78) |
| Oracle Linux 9 | x86_64 / aarch64 | Not yet run | [postgres1st/pfm#78](https://github.com/postgres1st/pfm/pull/78) |
| Windows TDE | — | Not yet run | [postgres1st/pfm#78](https://github.com/postgres1st/pfm/pull/78) |

**GA supported-platform declaration**: RHEL 9, Rocky Linux 9, AlmaLinux 9 —
both x86_64 and aarch64. All other combinations above are either explicitly
unsupported or unvalidated and must not be advertised as supported.

---

## Open items

- Oracle Linux 9 and Windows TDE remain unvalidated — no AMI / not run.
  They must stay off the supported list until run, per the roadmap outcome.
- SLES 15 SP7 install fails — [postgres1st/pfm#80](https://github.com/postgres1st/pfm/issues/80).
