# P17 — Supported-platform declaration: native certification matrix

Roadmap outcome (as written): "State exactly which operating systems and
architectures are certified for first GA. Do not advertise unvalidated
combinations as supported."

This is the record of actually running the certification checklist against
each OS/architecture combination this product claims to support
(`RHEL 9 / Rocky Linux 9 / AlmaLinux 9`, per `README.md` and
`build/packages/pfw-airgap-INSTALL.md`). Filled in as each combination is
actually run — leave a row `Not yet run` rather than guessing, since an
un-run row and a passing row must stay visually distinct.

Only once every row below is either PASS or explicitly marked as an accepted
known limitation should `documentation/docs/install-pmm/install-pmm-server/prerequisites.md`
and `.../install-pmm-client/prerequisites.md` be rewritten to state a supported list —
that rewrite is deliberately **not** part of this document. See the earlier
scoping note: those pages currently claim Debian/Ubuntu/Oracle Linux/Amazon
Linux/Docker/Podman/Kubernetes/ARM64-via-emulation, none of which are true for
this product, but fixing them before this matrix is filled in would just be
asserting something new and equally unverified.

---

## Certification matrix

Bundle tested: `PFW_3.9.0_BETA2` (or the current release at time of test — record which).

| OS | Arch | Status | Date | Tested by | Notes |
|---|---|---|---|---|---|
| RHEL 9 | x86_64 | ✅ PASS | 2026-09-07 | (prior session, see `docs/p0-blockers-handoff.md` P0.5) | RHEL 9.8, native, SELinux enforcing, zero denials, `dontaudit` disabled. Cold reboot exercised. Checks 3-6 below not confirmed run for this combination — see open items. |
| RHEL 9 | aarch64 | Not yet run | | | |
| Rocky Linux 9 | x86_64 | Not yet run | | | |
| Rocky Linux 9 | aarch64 | Not yet run | | | |
| AlmaLinux 9 | x86_64 | Not yet run | | | |
| AlmaLinux 9 | aarch64 | Not yet run | | | |

---

## Per-combination checklist

Each PASS above must mean all six checks passed, not just "it installed."
Record failures too — a red result here is exactly the evidence this document
exists to produce.

| # | Check | RHEL9/x86_64 | RHEL9/aarch64 | Rocky9/x86_64 | Rocky9/aarch64 | Alma9/x86_64 | Alma9/aarch64 |
|---|---|---|---|---|---|---|---|
| 1 | Normal install + SELinux enforcing, zero denials | ✅ | | | | | |
| 2 | Reboot / auto-start persistence | ✅ | | | | | |
| 3 | Wrong-architecture rejection | | | | | | |
| 4 | Low-disk scenario (fails loudly, not silently) | | | | | | |
| 5 | Low-memory scenario (2GB, below the 4GB minimum) | | | | | | |
| 6 | Interrupted package transaction (recovers cleanly on retry) | | | | | | |

---

## How each check is run

See the AWS setup steps and per-check commands worked out in-session
(launch instance → download signed release bundle → verify checksum + GPG
fingerprint → install → the six checks above). Summarized here for whoever
picks this up next:

1. **Setup** — Rocky/AlmaLinux 9 or RHEL 9 AMI (matching arch), `t3.large`/`t4g.large`
   or larger, download `pfw-server-el9-<arch>.tar.gz` + `.sha256` from the
   GitHub release, verify checksum, verify the GPG fingerprint against
   `5F62 9B10 E9EA 318F 0D10 42E9 20E4 C7CB 9663 03F6` before importing,
   `dnf --enablerepo=pfw install pfw-server`, `systemctl enable --now pfw.target`.
2. **Check 1** — `getenforce` (expect `Enforcing`), `/v1/readyz` returns 200,
   `systemctl list-units 'pfw*' --all` all active, `ausearch -m avc -ts recent`
   finds nothing.
3. **Check 2** — reboot the instance, reconnect, confirm every `pfw*` unit is
   active again unprompted.
4. **Check 3** — download the *other* arch's tarball on the same host, attempt
   to install a package from it, confirm `dnf` refuses it as the wrong
   architecture rather than silently accepting it.
5. **Check 4** — fill `/srv` to near-capacity, confirm the product fails
   loudly (visible errors in `journalctl -u pfw-managed`), not silently.
6. **Check 5** — a second, memory-constrained instance (`t3.small`, 2GB,
   below the documented 4GB minimum); confirm graceful failure or predictable
   degradation, not a silent OOM crash-loop.
7. **Check 6** — kill `dnf install` mid-transaction, confirm `rpm -Va`
   doesn't show corruption and a retried install completes cleanly.

---

## Open items

- RHEL9/x86_64's existing PASS (from `docs/p0-blockers-handoff.md`) only
  confirmed checks 1-2 — checks 3-6 have never been run for *any* combination,
  including the one already marked certified. Worth re-running the full six
  against RHEL9/x86_64 too, not just treating it as done.
- Low-disk and low-memory thresholds above are a starting guess (near-full
  `/srv`, 2GB RAM) — not validated against any documented minimum beyond
  "the product should not silently corrupt data or crash-loop forever."
