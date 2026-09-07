# Postgres1st WatchTower 3.9.0 beta2 — release readiness

Written 4 Sep 2026, re-verified at `60378b057`, extended 7 Sep 2026 at `3841d8a50`.
Records what is verified, what is not, and the decisions that will otherwise be
re-litigated. Companion to `docs/p0-blockers-handoff.md` (security work) and
`docs/pfw-identifier-map.md` (naming).

## Verified — against the aarch64 bundle at `60378b057`, NOT against HEAD

> **Read this table with its date attached.** It describes one specific artifact, built
> before the label rename, the node-identity rename, the SELinux credential-store fix and
> its revert. That bundle no longer exists on the build machine. The four green suites
> below therefore attest to an artifact that is not what beta2 will ship, and none of them
> has been re-run against a bundle carrying the current tree.
>
> This section is deliberately left pointing at the old bundle rather than updated to a
> newer throwaway one: the sources are not tagged yet, so every bundle built between then
> and the tag is disposable. It gets rewritten once, against the tagged build.

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

- ~~**x86_64.**~~ **Done (7 Sep).** Built on an x86_64 host: 15 signed RPMs, and the four
  container suites passed there — but see the caveat below, because that was the *pre-fix*
  bundle.
- ~~**A real RHEL host.**~~ **Done (7 Sep), and it found a release blocker.** See item 2
  under "Blocking" for what it found and how it was fixed. `pfw_nginx.pp` now loads on an
  enforcing host and `/srv/nginx` carries `pfw_cert_t`.
- **The container suites have never run against a bundle carrying the SELinux fix.** The
  `46/0`, `34/0` and `58/0` figures are from the bundle built *before* `e52e98854`. The
  rebuilt bundle has had only a native install and a reboot. This matters more than the
  usual staleness: the fix adds an `ExecStartPost` with **no** `-` prefix, which is a new
  way for `pfw-init.service` — and therefore `pfw.target` — to fail hard, and nothing has
  exercised that path in a suite.
- ~~**Reboot persistence**~~ **done (7 Sep)**; **low-disk, low-memory and interrupted
  transactions** remain unexercised.
- **Scale.** No measured fleet-scale number exists.

## Blocking, in order

~~1. The bundle predates the `pg_stat_monitor` documentation.~~ **Done** — `8eed7093`
   carries it.

~~2. The Grafana fork is still `pfm-*`.~~ **Done** — 37 assets renamed at fork commit
   `aa2b9b2`, both pin sites bumped, and verified in the shipped package: zero `pfm-`
   paths. The plugin *directories* (`pmm-check`, `pmm-update`, `pmm-pt-summary-*`) were
   deliberately left alone — dashboards resolve panels by those ids.

**Nothing is blocking on this machine.** All three items below needed different hardware
and all three are now done — on EC2 x86_64 hosts, 7 Sep. What is left after them is not a
hardware problem but unfinished testing: the suites re-run against a bundle built from the
tagged tree, plus scale, low-disk, low-memory and interrupted transactions.

1. ~~**x86_64.**~~ **Done (7 Sep).** Built on a fresh `m7i.4xlarge` RHEL 9.8 host with
   nothing but `git`, `tmux` and `docker-ce` installed; the build clones
   `PFW_GRAFANA_REPO` at the pinned commit itself, so no seeding was needed. 15 RPMs,
   all signed. Confirms cross-building is still not supported and still not required.
2. ~~**A real enforcing RHEL host.**~~ **Done, and it found a release blocker.** The
   bundle was installed on a pristine RHEL 9.8 x86_64 host with SELinux enforcing, by
   the documented path with `gpgcheck=1`. `pfw_nginx.pp` loads and `/srv/nginx` carries
   `pfw_cert_t`.

   It also did not boot: `pfw-managed`, `pfw-grafana` and `pfw-qan-api2` restarted
   forever with `Failed to load environment files: Permission denied` and `readyz` stuck
   at 500. Cause: `/srv/.pfw-secrets` was generic `var_t`, and `EnvironmentFile=` is
   opened by **PID 1 as `init_t`** — a confined domain — before the unit's `User=` and
   `CapabilityBoundingSet=` apply to the child. That the three services are themselves
   unconfined is irrelevant; the reader is systemd. Fixed by giving the store its own
   `pfw_secret_t` type readable by `init_t` alone, plus a root `ExecStartPost` relabel
   that fails loudly instead of leaving a silent no-boot.

   Two lessons worth keeping. The container suites passed 46/0, 34/0 and 58/0 against
   this same bundle, so **a green container run says nothing about an enforcing host** —
   a static guard in `test-frozen-identifiers` is the standing substitute, and a native
   install is the only real check. And the first diagnosis was wrong: an earlier fix
   (`f63f05611`, now reverted) blamed `CapabilityBoundingSet=` and handed the store to
   root. It was proven unnecessary on the host — the original `0700`/`0600 pfw:pfw`
   store, unreadable by root-without-caps, boots fine once the label is right.
   **Verified from the built RPM, not by hand.** A rebuilt, signed bundle was installed
   on a second pristine host by the documented path with `gpgcheck=1`: `%post` loaded
   `pfw_nginx`, the packaged `ExecStartPost` labelled the store `pfw_secret_t`, and
   `readyz` reached 200 in about ten seconds with **no manual intervention at all**.
   `NRestarts=0` on all three units — they never failed once, against 231 and 266 before.
   The store kept its original `0700`/`0600 pfw:pfw`, confirming again that the
   permission change was never what fixed anything.
3. **Reboot persistence** — done, on that same host. `pfw.target` enabled, cold reboot,
   `readyz` back to 200 within seconds, every unit active, `NRestarts=0` across the boot.
   `/srv/.pfw-secrets` kept `pfw_secret_t` and `/run/pfw` was correctly recreated from
   tmpfs as `var_run_t`. Zero pfw-related SELinux denials in the whole audit log, checked
   with `dontaudit` disabled as well as enabled — the one denial present is a benign
   `siginh` on an `init_t`→`initrc_t` transition, unrelated to this stack.

   **Scale** has still not been exercised.

Work deferred past beta2 is tracked in `docs/ga-readiness.md`: the full release process,
the internal `pmm-managed` database/role rename, negative-controlling `test-pfw-client`,
moving the credential store out of `/srv` so its SELinux label stops being something the
product has to maintain, guarding every spec's version against `PFW_VERSION` rather than
only `pfw-server`'s, and the documentation rewrite.

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

`build/scripts/test-frozen-identifiers` now reports **71 passing assertions**, most of them
comparing two such lists rather than pinning a string, and every one negative-controlled.
When adding a pair that must agree, add the assertion in the same commit — none of these
were found by reading.

The SELinux credential-store defect (7 Sep) is the same shape one level down: the policy's
`.fc` and the paths the units actually read are two lists that must agree, and nothing
compared them. `secret_store_is_labelled_for_init` now does.
