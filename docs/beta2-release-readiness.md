# Postgres1st WatchTower 3.9.0 beta2 — release readiness

Written 4 Sep 2026, re-verified at `60378b057`. Records what is verified, what is not, and the
decisions that will otherwise be re-litigated. Companion to `docs/p0-blockers-handoff.md`
(security work) and `docs/pfw-identifier-map.md` (naming).

## Verified

Built from an empty tree on aarch64, then tested against the artifact it produced —
not against the source.

| | |
|---|---|
| bundle | `pfw-server-el9-aarch64.tar.gz`, 486 MB, 15 packages |
| sha256 | `8eed7093090b75e4ef348aaa600b259b5d7e86d91cd11c5c10d287e0eea01239` |
| signatures | all nine of our RPMs verify `digests signatures OK` |
| versions | every package `3.9.0~beta2`; `pfw-victoriametrics` `1.147.0` |

| suite | result |
|---|---|
| `test-pfw-airgap` | 46 passed, 0 failed |
| `test-pfw-upgrade` | 34 passed, 0 failed |
| `test-pfw-negative-control` | 58 passed, 0 failed — 56 controlled, **0 not discriminating** |
| `test-pfw-qan` | 5 passed, 0 failed, itself negative-controlled |

All four re-run against this bundle after the encryption-key and distribution-sentinel
renames and the Grafana asset rebrand — the earlier results predated changes to first-boot
provisioning, which is the worst place to carry a stale pass.

**This is the first aarch64 bundle.** It was impossible before: four components were
fetched prebuilt from Percona's S3 cache, which publishes x86_64 only. Everything is now
built from source and `stage_s3` is gone.

## Not verified — do not claim these

- **x86_64.** The pipeline supports it and the pins resolve, but cross-building is not
  supported, so it needs an x86_64 host. Nothing here has been built or tested there.
- **A real RHEL host.** SELinux transitions and file capabilities are both inert under
  `docker run --privileged`, so the container suites cannot substitute. `pfw_nginx.pp`
  in particular has never been loaded on an enforcing host.
- **Reboot persistence, low-disk, low-memory, interrupted transactions.**
- **Scale.** No measured fleet-scale number exists.

## Blocking, in order

~~1. The bundle predates the `pg_stat_monitor` documentation.~~ **Done** — `8eed7093`
   carries it.

~~2. The Grafana fork is still `pfm-*`.~~ **Done** — 37 assets renamed at fork commit
   `aa2b9b2`, both pin sites bumped, and verified in the shipped package: zero `pfm-`
   paths. The plugin *directories* (`pmm-check`, `pmm-update`, `pmm-pt-summary-*`) were
   deliberately left alone — dashboards resolve panels by those ids.

**Nothing is blocking on this machine.** What remains needs different hardware:

1. **x86_64.** Cross-building is not supported — the Go and webpack builds and `rpmbuild`
   all emit host-native output — so an x86_64 bundle must be built on an x86_64 host.
   This is no longer gated on anything else: the Grafana fork is pushed to
   `postgres1st/grafana` (public) and the build now clones `PFW_GRAFANA_REPO` at the
   pinned commit when no local clone is supplied, so a fresh host needs no seeding.
   Point `PFW_GRAFANA_FORK` at an existing clone only to skip the fetch.
2. **A real enforcing RHEL host.** `pfw_nginx.pp` has never been loaded; container suites
   cannot prove it, because SELinux transitions are inert in Docker.
3. **Reboot persistence and scale.** Neither has been exercised.

Work deferred past beta2 is tracked in `docs/ga-readiness.md`: the full release process,
the internal `pmm-managed` database/role rename, negative-controlling `test-pfw-client`,
and the documentation rewrite.

Not blockers, and deliberately out of scope: everything in
`docs/deferred-container-path.md` and `docs/deferred-documentation-rebrand.md` (the
documentation site still carries Percona links and pre-rename names; the bundled customer
documents are clean and guarded). beta2 ships a native RPM bundle, so the Docker,
ansible and deb paths are not built or tested here. That list exists so the divergences
are worked through when the container builds are picked up, rather than rediscovered one
failure at a time.

## Stored-state paths still carrying the old name

`/srv/pfw-encryption.key` was renamed from `pmm-encryption.key` **before** beta2 ships,
which is the only safe window: `encryption.New()` generates a fresh key when it finds
none, so renaming the default on a host that already has one would silently re-key it and
make everything already encrypted undecryptable. Nothing fails at the moment of loss. A
fallback adopts the legacy file if present, and warns.

`/srv/pfw-distribution` was renamed the same way, and it is the higher-stakes of the two:
it is the sentinel that stops `provision_srv()` re-running initdb over a real cluster, so
"absent" means "never provisioned". `srv_provisioned()` accepts either name, `%post`
accepts either, and pfw-managed falls back when reading. The Docker entrypoint still
writes the old name and is deliberately untouched -- that path is parked pre-rename -- so
the fallback is load-bearing there, not only for history.

Both fallbacks are asserted by `test-frozen-identifiers`, because removing one is
invisible until a host is destroyed by it.

## Decisions that should not be re-litigated

- **No upgrade path from beta1.** Package names, units, the service account and the
  install root all moved. `3.9.0~beta1 < 3.9.0~beta2 < 3.9.0`, verified with
  `rpm.labelCompare`, so beta2 → rc → GA upgrades in place.
- **`pg_stat_monitor` is not shipped.** It belongs on the monitored database, which is
  the operator's to install. `pfw-admin` defaults to `--query-source=pgstatmonitor`;
  `pgstatements` is the alternative and needs nothing extra. `test-pfw-qan` is what
  notices if the default and the shipped reality diverge again.
- **The VictoriaMetrics pin is a Percona branch tag**, not an upstream release —
  `pmm-6401-v1.147.0`, carrying PMM-specific "read Prometheus data files" work, diverged
  217 ahead of v1.151.0 and 467 behind. Moving to upstream drops a feature; see
  `build/AGENTS.md`.
- **supervisord program names stay `pmm-*`.** They are keys into `systemdUnitName()`,
  which builds `"pfw-" + TrimPrefix(name, "pmm-")`. Renaming yields
  `pfw-pfw-managed.service`.
- **Tier C identifiers do not move**: the Go module path, `PMM_*` variables, Grafana
  plugin ids, the ClickHouse database, the `pmm-managed` database and role.

## The one lesson worth carrying

Every defect this cycle was a **list that had to agree with another list, in a different
file, with nothing checking**: emit vs build loop, closure seed vs `Requires:`, spec
version vs `PFW_VERSION`, nginx glob vs conf.d filenames, control fixtures vs package
names, `SRV` vs the host being addressed.

`build/scripts/test-frozen-identifiers` now holds 38 assertions, most of them comparing
two such lists rather than pinning a string, and every one negative-controlled. When
adding a pair that must agree, add the assertion in the same commit — none of these were
found by reading.
