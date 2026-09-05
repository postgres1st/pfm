# Deferred: the container and deb packaging paths

Issues deliberately **not** fixed during the PGF WatchTower rename, because they live in
delivery paths beta2 does not ship. Not a bug list to triage — a list to work through when
the Docker builds are picked up, so none of it is rediscovered one failure at a time.

**Not release blockers for beta2.** beta2 ships a native RPM bundle only; nothing here is
built by `build-pfw-airgap` (verified: zero references to `build/docker` in it). Blockers
live in `docs/beta2-release-readiness.md`.

Measured 4 Sep 2026 at `8f79ef9c4`.

| Path | `pfw-*` refs | `pmm-*`/`pfm-*` refs |
|---|---|---|
| `build/docker` | 9 | 31 |
| `build/ansible` | 18 | 98 |

Both are majority pre-rename. The counts matter less than the specific divergences below,
where the container path and the native path now disagree about the same thing.

## 1. The distribution sentinel is written under the old name

`build/docker/server/entrypoint.sh:77` writes `/srv/pmm-distribution`. The native path
writes `/srv/pfw-distribution`, and `build/ansible/roles/initialization/tasks/main.yml:6`
still reads the old name.

**This is the one item with a live consequence.** `pfw-managed` reads the new path and
falls back to the old one, and `test-frozen-identifiers` asserts that fallback survives —
so the fallback is load-bearing for the container deployment *today*, not merely a
courtesy to history. Removing it would break the container path, not just old hosts.

When rebranding Docker: move the entrypoint and the ansible lookup together, then decide
whether the fallback still earns its place.

## 2. nginx config carries locations removed from the native path

`build/ansible/roles/nginx/files/conf.d/pmm.conf` still serves `/pmm-static` and
`/percona-blog/feed`. Both were removed natively: `/pmm-static` aliased a tree the RPM
never creates, and the feed proxied percona.com from a product documented as needing no
internet. The Grafana news panel that consumed the feed is gone from the dashboards, so
the container path serves two endpoints nothing asks for, one of them off-host.

## 3. supervisord program names and their commands disagree

`build/ansible/roles/supervisord/files/pmm.ini` declares `[program:pmm-init]`,
`[program:pmm-managed]`, `[program:pmm-agent]` while the commands beside them run
`/usr/sbin/pfw-agent` and friends.

**Do not "fix" this by renaming the programs.** They are keys into `systemdUnitName()`
(`managed/services/supervisord/systemd.go`), which builds `"pfw-" + TrimPrefix(name,
"pmm-") + ".service"`; renaming them produces `pfw-pfw-managed.service`. The mixed
appearance is correct and is explained in `docs/pfw-identifier-map.md`. It is listed here
only because it reads like an oversight and has already been queried twice.

## 4. deb packaging installs binaries that no longer exist

`build/packages/deb/{install,links,rules}` reference `pmm-agent` and `pmm-admin`. The
binaries are `pfw-agent` and `pfw-admin`, so the deb would install nothing usable and its
`/usr/bin` symlinks would dangle. Latent rather than broken: the airgap bundle is RPM-only
and nothing builds the deb today.

`build/packages/deb/copyright` also still says `Upstream-Name: pmm-client`.

## 5. Documentation for deployment modes we do not ship

`documentation/docs/install-pmm/` carries 31 pages for Docker, Podman, Helm, AWS and the
virtual appliance, against 28 for what we actually ship. `site_url` is empty, so the site
is not published and no customer reads them yet — but publishing as-is would tell people
to do things that cannot work.

Decide deletion before renaming: there is no point renaming pages that should not exist.
