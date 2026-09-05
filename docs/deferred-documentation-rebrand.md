# Deferred: the documentation site's rebrand

Issues found in `documentation/` during the beta2 rebrand pass and deliberately **not**
fixed. Not a bug list to triage — a list to work through before GA, so none of it is
rediscovered one broken link at a time.

**Treat this as a rewrite, not a rename.** That is the main conclusion of the survey below.
A sweep that replaced every `pmm-*` string with `pfw-*` would produce correctly-named
documentation for a product that does not exist: the site's install section documents five
delivery paths we do not ship and omits the one we do (§1). The names are the smaller half
of the problem and the easier half to fix.

**Not release blockers for beta2.** The RPM bundle takes exactly one file from this tree
(`build-pfw-airgap` copies the current release's notes to `RELEASE_NOTES.md`); nothing
else here is packaged. The bundled customer documents — `pfw-airgap-INSTALL.md`,
`pfw-airgap-INSTALL-CLIENT.md`, and the beta2 notes — are clean and are guarded by
`build/scripts/check-release-claims`. Blockers live in `docs/beta2-release-readiness.md`.

Measured 6 Sep 2026 at `2c378f1ca`, the commit that removed the unsupported-database pages.

## Already done, for context

Recording this so nobody re-does it, and so the numbers below are read as the remainder
rather than the whole:

- **84 files removed** — 48 advisor checks, 29 dashboard references, 4 connect-database
  guides and 2 QAN pages for MySQL, MongoDB and ProxySQL, plus an orphaned `kapa.js` that
  existed to build an `ask-percona-ai` button. The product refuses those services at
  registration and ships no dashboards for them, so the pages documented things a customer
  cannot reach. 356 → 273 pages.
- **The fallout was repaired**, not left dangling: 3 tab sections and 28 dead
  reference-link definitions in the dashboards index, 2 tabs in `QAN-stored-metrics.md`,
  and 6 pages that had become actively false (`add.md` claimed support for "MySQL,
  PostgreSQL, MongoDB, Valkey, ProxySQL, and HAProxy"). Verified: zero dead links remain.
- **Upstream references are kept on purpose.** 48 links to `docs.percona.com`,
  `percona-pmm.readme.io` and `percona.github.io` stay, framed as upstream. For components
  we ship unmodified, that is the most detailed reference available, and
  `documentation/docs/get-help.md` already states plainly that Percona is not our support
  channel. Deleting them would trade a true statement for a tidier grep.

## 1. The structure describes a different product

`documentation/docs/install-pmm/` is 55 pages, and its deployment options are:

```
install-pmm-server/deployment-options/{aws,docker,helm,podman,virtual}
```

beta2 ships **none** of those. It ships a signed, air-gapped RPM bundle, and there is no
deployment-options page for it. Across the whole 273-page site, exactly **one** page
mentions the air-gapped bundle at all, while the real install guide —
`build/packages/pfw-airgap-INSTALL.md`, 17.5 KB — lives outside the site entirely and is
shipped in the tarball instead.

So a reader arriving at the documentation is offered five ways to install the product,
all of which are wrong, and is not told the correct one. Renaming the binaries inside
those pages makes the wrong instructions look more authoritative, not less.

The same holds for `discover-pmm/`, `configure-pmm/` and `pmm-upgrade/`: they are upstream
PMM's information architecture, written for a multi-database product with several delivery
paths. This product is PostgreSQL-only with one delivery path. The nav, the page set and
the framing all need to be re-derived from what we actually ship — which is why the
unsupported-database page removal (above) was the right first step and not the last one.

Counts of docker/podman/helm/kubernetes/virtual-appliance content, for scale: 67, 24, 22,
28 and 13 pages mention them respectively. Much of that is incidental prose rather than
whole pages, but it is the same class of statement — describing capabilities and paths
that are not ours.

## 2. Links that route customers to Percona

The two categories below are not cosmetic: each one hands a customer an instruction that
does the wrong thing, or sends them somewhere that cannot help.

| Category | Links | What is wrong |
|---|---|---|
| Package sources | 21 | `yum install -y https://repo.percona.com/yum/percona-release-latest.noarch.rpm` and `wget https://downloads.percona.com/downloads/pmm3/…` — instructions that install **Percona's** repository and PMM, in the documentation for an air-gapped product that ships its own signed bundle |
| Issue tracker | 13 | `perconadev.atlassian.net` — including a "Create PMM documentation issue" link, which routes our users' bug reports into Percona's Jira |
| Support funnel | 40 | `www.percona.com`, `forums.percona.com`, `per.co.na`. Includes Percona's **Privacy Policy** and **Trademark Policy** linked as though they govern this product, and a services-lifecycle policy page presented as our support policy |

The privacy and trademark links are the ones to fix first. The others send someone to the
wrong place; those two misstate whose terms apply.

The clearest example of how this class of defect hides is
`documentation/api/welcome/monitoring.md:11`:

```markdown
- [Install PGF WatchTower](https://per.co.na/pmm/quickstart)
```

A previous rename pass rewrote the visible label and left the target. The page reads
correctly, and the link sends the reader to Percona's shortener for PMM's quickstart.
Grepping for the brand name in prose will never find these — the prose is already
rebranded. They have to be found by checking link *targets*, which is what the guard in
section 3 would do.

## 3. The old product and binary names

The site still writes the pre-rename names throughout. These are what a customer types or
navigates, so each one is a broken instruction or a wrong URL. Mechanical, and worth doing
only *after* §1 — renaming a page that should not exist is wasted work, and the removal in
§1 already cleared a quarter of the site.

| Name | Occurrences | Files |
|---|---|---|
| `pmm-server` | 418 | 65 |
| `pmm-admin` | 62 | 15 |
| `pmm-managed` | 23 | 6 |
| `pmm-agent` | 10 | 7 |

Directories, which also determine published URLs:

```
documentation/docs/install-pmm/          (plan-pmm-installation, install-pmm-server,
                                          install-pmm-client)
documentation/docs/configure-pmm/
documentation/docs/discover-pmm/
documentation/docs/uninstall-pmm/
documentation/docs/pmm-upgrade/
documentation/docs/use/commands/pmm-admin/
documentation/api/pmm-server-config/
```

Assets: `assets/pmm-fav.svg`, `assets/pmm-logo.png`, `assets/pmm-mark.svg`, `css/pmm.css`.

**Do not do this with a blanket `sed`.** Not all `pmm-*` strings are stale — some are
frozen stored and on-the-wire identifiers that must not move (`pmm-agent.log`,
`pmm-agent.yaml`, the `PMM_*` environment variables, the `pmm-check` / `pmm-update`
Grafana plugin directories that dashboards resolve panels by). `docs/pfw-identifier-map.md`
is the authority on which tier each name is in, and it has itself drifted before, so verify
rows rather than trusting them. A blind sweep during this workstream already produced a
spec reference to a file that does not exist.

The shape that works, used for the renames that are already done: derive the mapping from
the artifacts, sweep, then add a guard that fails if the old name reappears — and
negative-control the guard in both directions before believing it.

## 4. The guard does not reach this tree

`build/scripts/check-release-claims` asserts three rebranding negatives — no `systemctl`
instruction naming a unit the packaging does not ship, no command or unit under an
abandoned prefix (`pmm-`, `pfm-`), and no Percona/PMM URL — but only across the install guides and the
release notes — the instruction checks narrow further, to the guides plus the *current*
release's notes, since a superseded release note is a record of what that release shipped
and rewriting it would make a historical document false. The rest of `documentation/` is
not examined at all.

`build/scripts/test-docs-site` covers part of the gap, and only part. It builds the site
and fails on a dead link or on any tracker or marketing endpoint anywhere in the output,
which is what keeps the removed trackers from returning during the rewrite — the risk
window, since they arrived through copied upstream templates in the first place. It is
explicitly **not** a release gate.

What it deliberately does not assert is everything in sections 1 to 3: it would be
permanently red if it did, which is how a check ends up disabled rather than fixed. So
nothing yet stops the *prose* defects from growing back.

When sections 1 to 3 are closed, extend `check-release-claims` to `documentation/`. The
URL check cannot be applied wholesale: upstream references legitimately stay, so the
docs-tree version must forbid the funnel, tracker and package-source hosts specifically
while permitting `docs.percona.com`.
