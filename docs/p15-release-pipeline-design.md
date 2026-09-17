# P15 — Release integrity and evidence: pipeline design

Roadmap outcome (as written): "Publish CI results, checksums, verifiable release
signatures, installation-test evidence and a customer verification procedure with
every release."

## What was already true before this

Checked against the actual beta2 release (`PFW_3.9.0_BETA2`, 2026-09-07):

| Requirement | Status |
|---|---|
| Checksums | Done — `.sha256` per tarball |
| Verifiable release signatures | Done — GPG-signed RPMs, key fingerprint published, `gpgcheck=1` |
| Customer verification procedure | Done — "Quick start" / "Verifying" sections in the release notes |
| Installation-test evidence | Partial — pass counts stated in prose ("46/0, 34/0, 5/0, 58/0"), nothing attached to check |
| CI results published | **Not done at all** |

Checked every workflow in `.github/workflows/`: nothing builds the RPM bundle,
runs the four test suites, signs anything, or publishes a release. The one
workflow that could (`release-doc.yml`) is dead code on this fork — gated
`github.repository == 'percona/pmm'`, and both it and `sbom.yml` only trigger on
`v[0-9]+.[0-9]+.[0-9]+*` tags, which neither `PFMM_3.9.0_BETA1` nor
`PFW_3.9.0_BETA2` matches. **Every release to date has been built, signed, tested,
and published entirely by hand.** This sharpens what `docs/ga-readiness.md` item 1
already said in prose, with the specific reason why.

## What this adds

Two workflows, split exactly where `build-pfw-airgap`'s own header comment
already says the split belongs — at the `bundle` stage, the only one that reads
the GPG signing key:

- **`release-build.yml`** — runs every stage except `bundle` (component builds,
  RPMs, dependency closure) and uploads the three paths a human needs to finish
  the job on the trusted, offline-key workstation: `work/seed`, `work/closure`,
  `os-dependencies.txt`.
- **`release-test.yml`** — downloads an already-signed bundle from a release,
  verifies its checksum, runs `test-pfw-airgap` / `test-pfw-negative-control` /
  `test-pfw-upgrade` / `test-pfw-qan`, and attaches the raw log as a release
  asset — so "46/0" becomes something openable, not just a claim.

The signing step itself stays manual, on purpose — same security posture beta2's
own release notes already describe ("the signing key is held on a workstation,
not on the build machines").

## Decisions made

- **Tag convention going forward**: `pfw-v<version>` (e.g. `pfw-v3.9.0-beta2`).
  Neither existing tag (`PFMM_3.9.0_BETA1`, `PFW_3.9.0_BETA2`) matches any CI
  trigger today; this is the convention future release tags should follow so
  `release-build.yml` can eventually trigger on `push: tags:` directly instead
  of only `workflow_dispatch`.

## Deliberately left open

- **`release-build.yml`'s trigger and runner.** `build-pfw-airgap` needs ≥16GB
  RAM / ≥30GB disk (mainly the Grafana webpack stage) — default GitHub-hosted
  runners don't meet that. Larger hosted runners bill per-minute; a self-hosted
  runner has no per-minute cost but needs someone to provision and maintain a
  real machine. That's a budget/infra call, not a development one, so the
  workflow currently only runs on `workflow_dispatch` — it triggers nothing,
  costs nothing, until whoever owns that decision wires it to real tags and
  picks a runner. The `push: tags:` trigger is commented out in the file,
  ready to uncomment.
- **Auto-publish vs. human-click-publish.** `release-test.yml` attaches evidence
  to an existing release; it does not decide whether that release goes from
  draft to published. That stays a deliberate human action after reading the
  attached test results — the actual point of no return for customers
  shouldn't be automatic.
- **Native SELinux-enforcing validation** cannot come from either workflow —
  SELinux transitions and file capabilities are both inert under Docker (the
  same limitation already documented for P0.5/P17 in
  `docs/p0-blockers-handoff.md`). That stays a manual native-VM step regardless
  of how complete this pipeline gets.

## Neither workflow has been run

Both are sketches reviewed for YAML validity only (`yaml.safe_load`), not
exercised against a real build — `release-build.yml` in particular is expected
to fail on `ubuntu-latest` runners exactly because of the unresolved runner-sizing
question above.
