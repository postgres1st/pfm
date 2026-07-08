# PFMM UI rebrand — follow-up fixes (fresh session)

Picks up after the initial rebrand + theme work. **Read the memory first**: `ui-rebrand-postgres1st` (naming/scope/palette) and `pmm-ui-build-in-container` (how to build/test/lint/screenshot). This file is the actionable to-do; the memory is the reference.

## Current state (2026-07-07)
- Branch: `ui-rebranding` (worktree `.worktree/ui-rebranding`), remote `pfm` = `github.com/postgres1st/pfm`.
- **PR #18** open → `main`. HEAD at time of writing: `37b658de`.
- Done: full UI rebrand (text/URLs/plugins/74 dashboards/assets+renames), disable Tour+Updates behind flags, azure+amber accessible theme (soft chips/alerts, raised Papers, AA text), aggressive review + fixes, CI lint green.
- **Pre-push gate**: pushing requires an aggressive-review attestation for the exact HEAD in `.git/pgf-review-ui-rebranding.md` (any new commit invalidates it — re-run review, re-write note with new HEAD sha, then push).

## Run-app verification results (2026-07-08) — #1 executed
Stood up the real stack natively on arm64 and drove it with headless chromium (login admin/admin). How: override `PMM_SERVER_IMAGE=pmm-server-pg-arm64:3.8.1` (upstream `perconalab/pmm-server:3-dev-container` is amd64-only, no qemu here), run as the image's default UID 1000 (arbitrary-UID/NSS-wrapper image — running as root breaks `nginx: getpwnam("nginx")`), **drop** the `pmm-compat` bind mount (it forces `/srv/grafana/plugins` root-owned and breaks first-boot `/srv` init — `docker cp` it after boot instead), keep only the `pmm-dev.conf` proxy mount, skip mysql/watchtower. Vite runs in a `node:22-alpine --network host` container; pmm-server reaches it via `extra_hosts: host.docker.internal:host-gateway`. Standalone compose: `ui/dev-standalone.arm64.yml` (untracked).

**Verified working in the running app:** rebrand text/content (Welcome to PFMM, PFMM docs/dump, Postgres1st support/forum), azure primary on app pages (Settings, Help), and Tour/Updates removal *conclusively* — no Updates nav item, `/pmm-ui/updates` → Not found, and **zero `/v1/server/updates` XHR** from Help/Settings/RTA. The only update-check comes from Grafana's bundled **"PMM Upgrade" dashboard panel** (out-of-repo chrome; that stock dashboard is also un-rebranded because #4's JSON wasn't mounted).

**Bugs found & FIXED (in repo, verified live + lint/tsc/tests green):**
- **Sidebar logo was invisible/clipped** — `icons/pfmm-titled.svg` wrapped the elephant in a nested `<svg viewBox="46 42 180 174">`; through `svg?react`+MUI `SvgIcon` the nested viewBox got clobbered to the outer `0 0 141 48`, drawing the elephant almost entirely outside the box (only a sliver showed, then `overflow:hidden`). Fixed by flattening the nested `<svg>` to `<g transform="translate(47,1.2835) scale(0.261111) translate(-46,-42)">` (single coord space, nothing to corrupt). NB `pfmm-titled` has no wordmark ("wordmark cropped out") — it's an off-center elephant in a 141-wide canvas; positioning can be tightened (used only in `NavigationHeading`).
- **Help Center card top-borders were harsh yellow** — `HelpCenterCard.tsx` sourced `chartTokens['chart2']` from `@percona/percona-ui`. Now uses brand amber `theme.palette.secondary.main` (#F5B94D). Removed the now-unused `semanticTokens*` import.

**Login page = out of this repo (confirmed).** "Percona Monitoring and Management" / "Your single pane of glass" / Grafana-diamond logo / "proudly powered by" footer are **Grafana's own login**, hardcoded in the **percona/grafana fork** `public/app/core/components/Branding/Branding.tsx` (`@PERCONA`-marked `AppTitle`, `LoginTitle`, `GetLoginSubTitle`, `LoginLogo`→`img/icons/mono/pmm-logo.svg`, `MenuLogo`→`img/percona-logo.svg`). The fork is NOT checked out here (`utils/grafana` is an unrelated Go pkg). Real fix = edit those symbols + swap the 2 logo SVGs in the fork, rebuild the Grafana frontend, rebake the pmm-server image (separate workstream; also the login "Log in" button is Grafana-blue, not azure). Demonstrated live via a throwaway bundle-sed + logo-swap PoC in the running container (proves it's trivial at source; not committable).

**Not verified:** light mode in-app (harness couldn't force it off Grafana prefs — app stayed dark); dashboards (#4) still render Grafana's stock panels.

## How to verify anything (no host Node — use the Node 22 container)
```bash
cd .worktree/ui-rebranding
# lint + test + build (whole ui workspace)
docker run --rm -v "$PWD:/src" -w /src/ui node:22-alpine sh -c \
  'yarn install --frozen-lockfile && yarn lint && yarn test && yarn build'
```
Host runs Node 24 which breaks install (@grafana/plugin-e2e caps at ≤22). Always use the container. `yarn lint` matters — CI enforces `--max-warnings 0` and `no-explicit-any` is an error.

---

## Follow-ups (priority order)

### 1. [HIGH] Real running-app verification — the one that matters most — ✅ EXECUTED (2026-07-08, see "Run-app verification results" above)
Everything so far is verified by build + unit tests + a *component preview* + code review + lint. The app was **never run end-to-end**. The K8s CR-kind bug (caught only by manual review) proves subtle issues survive all automated checks.
- **Do:** stand up the real stack. `ui/docker-compose.yml` runs `perconalab/pmm-server:3-dev-container` + serves the UI (nginx `pmm-dev.conf` proxies to a vite dev server; mounts `apps/pmm-compat/dist`). Needs the plugins built and PMM admin login. Alternatively check with the team how they run it locally.
- **Verify in the running app:** the elephant logo in the real sidebar/app-bar; the azure theme on real pages; Tour/Updates actually gone (nav item, `/updates` route, no update-check network calls in devtools); soft chips/alerts as the app actually renders them (see #2); rebranded dashboards in Grafana (see #4).
- This closes #2 and #4 too.

### 2. [HIGH] Theme fixes are PARTIAL — custom status components not covered
> **Progress (2026-07-08):** the Help Center card yellow accent (a base-palette `chart2` leak, not a MUI status component) is fixed → brand amber. The five custom status components below are still uncovered.

The `MuiChip`/`MuiAlert` overrides in `ui/apps/pmm/src/lib/theme.ts` only match MUI components with `color/severity` ∈ success/warning/info/error. The app also has **custom status components** that may use `color="default"`, dynamic colors, or their own styling and therefore **won't get the soft-badge/contrast treatment**:
  - `src/pages/update-clients/severity-chip/SeverityChip.tsx`
  - `src/components/sidebar/nav-item/nav-item-badge/NavItemBadge.tsx`
  - `src/pages/rta/components/state-cell/StateCell.tsx`
  - `src/components/ha-badge/HighAvailabilityBadge.tsx`
  - `src/pages/rta/components/services-autocomplete-input/components/ServiceTags.tsx`
- **Do:** read each, determine how it colors itself, and either (a) route it through the severity colors so the theme variant applies, or (b) give it the same soft-tint + readable-text treatment locally. Re-run the WCAG contrast check on each (light + dark) — target AA (the earlier audit method: sample rendered pixels or compute from resolved palette).
- Note the earlier finding: base design-system semantic colors fail AA *as text on light bg*; their `contrastText` is a dark same-hue shade meant for tints/alerts, not white/black.

### 3. [MED] Placeholder URLs will 404
All doc/support/forum/blog links now point to non-existent `postgresfirst.com` paths (domain swapped, paths preserved). Full list + file:line in `docs/scratch/260704-ui-rebranding-urls.md`.
- **Do:** replace with real Postgres1st destinations once they exist, OR route them through a single constant so they're changed in one place, OR hide the links until infra is live. Decide with product.

### 4. [MED] Dashboards never rendered in Grafana
74 JSONs validate + data tokens preserved, but not visually checked. The K8s CR-kind mapping (now reverted) shows data-layer subtleties hide behind valid JSON.
- **Do:** load the pmm-app plugin in a running Grafana and eyeball the rebranded dashboards — titles, markdown panels, legends, value mappings, links. Watch for strings that were semantically load-bearing, not just display.

### 5. [MED] Footer version display disappears — product decision
`src/components/footer/Footer.tsx` returns null when `versionInfo` is absent; that came from the (now-inert) updates provider. So the footer + server version vanish on native pages (Page.tsx renders `<Footer/>`).
- **Do:** decide — accept (footer gone with Updates), or wire the version from a source independent of the update-check so it stays visible.

### 6. [LOW] No test asserts the DISABLED state
`WelcomeCard.test`/`HelpCenter.test` mock `TOUR_ENABLED=true` to keep testing the enabled path; nothing asserts the shipped (disabled) behavior.
- **Do:** add tests for flags=false — nav has no "Updates" item, `/updates` route 404s, UpdateModal not mounted, tour buttons/tips-card absent, no update-check query fires. Guards against silent regression.

### 7. [LOW] Latent shared-const mutation (dormant, pre-existing) — ✅ FIXED (2026-07-08)
`src/contexts/navigation/navigation.utils.tsx` `addConfiguration` mutated shared `NAV_CONFIGURATION` children (`updates.secondaryText`/`.badge`). Fixed: clone the children array + the updates item before writing, return `{ ...NAV_CONFIGURATION, children }`. Added `navigation.utils.test.tsx` (mocks `UPDATES_ENABLED=true`) asserting the shared constant stays pristine — red→green verified.

### Scope ceilings (awareness, not necessarily "fixable")
- Base look/fonts/many colors still come from `@percona/percona-ui` (npm dep, name unchanged) — brand override is accents only, no fork.
- Grafana's own chrome (login, dashboard rendering) is a separate bundled product, out of this repo.
- Palette values (azure `#0E7ABE`/`#5EAEE0`, amber `#F5B94D`, tints) are logo-derived approximations, not an official brand sheet — re-check if one appears.

## Re-creating the visual preview harness (was removed before merge)
The throwaway `apps/pmm/preview.html` + `src/preview.tsx` (renders real MUI components under `postgres1stThemeOptions`, screenshotted via headless chromium) were deleted so `tsc` stays clean. Recreate them the same way if you need isolated visual checks; **delete again before committing** (they trip `no-implicit-any`/tsc and aren't shippable). Serve via vite in-container on `/pmm-ui/preview.html`, screenshot with alpine `chromium-browser --headless=new`.
