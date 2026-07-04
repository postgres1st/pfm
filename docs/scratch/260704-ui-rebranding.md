# UI Rebranding Assessment — Percona/PMM → Postgres1st

**Branch:** `ui-rebranding` · **Date:** 2026-07-04 · **Scope:** user-facing UI only

## Decisions (locked)

| # | Decision | Choice |
|---|----------|--------|
| D1 | Product full name | `Percona Monitoring and Management` → `Postgres1st Monitoring and Management` |
| D2 | Acronym / short name | `PMM` → **`PFMM`** |
| D3 | Company name (standalone) | `Percona` → `Postgres1st` |
| D4 | Grafana dashboard JSONs (74 files) | **In scope** |
| D5 | Percona-domain URLs (~96) | Replace with placeholders **and** log each to a review file |
| D6 | Internal identifiers (routes, storage keys, test-ids, package/plugin ids, dashboard UIDs, API flags) | **Out of scope** (load-bearing; backend/Grafana-coordinated) |

## Open items requiring input

- **O1 — Third-party product names.** `Percona Server`, `Percona XtraDB / Galera`, `Percona Custom Resources`, `Percona Clusters` name *other Percona products being monitored*, not our product. Default: **leave as-is**. Listed in §5 for veto.
- **O2 — Brand assets.** New Postgres1st artwork does not exist in-repo. Logos/favicon need real design assets (see §3). Interim options: (a) you supply SVGs/ICO, or (b) I generate text-based placeholder SVGs (`PFMM` wordmark) so the build is coherent until final art lands.
- **O3 — External design-system dependency.** The main app's theme comes from `@percona/percona-ui@1.0.22` (npm), imported ~15+ places incl. `pmmThemeOptions`. Any Percona logo/color baked into that package is **not** rebrandable from this repo — needs a forked/replacement package. This is the ceiling on a "complete" visual rebrand.

---

## Change scope by category

### 1. Visible product-name text (~70 strings + ~20 dashboard titles + ~50 tooltips)
No i18n — all hardcoded in `*.messages.ts` / `*.constants.ts` and JSX. Replace rules: apply D1/D2/D3.

**`ui/apps/pmm` — highest concentration:**
- `src/lib/constants.ts:4` — `PMM_TITLE = 'Percona Monitoring and Management'`
- `src/components/app-bar/AppBar.messages.ts:2` — brand text `'PMM'`
- `src/components/footer/Footer.messages.ts:2` — `PMM ${version}`
- `src/components/page/Page.messages.ts:4` — `'PMM Home'`
- `src/pages/help-center/welcome-card/WelcomeCard.messages.ts` — full name + tour copy
- `src/pages/help-center/HelpCenter.constants.ts` — "Percona support", "Percona forum", "Percona Support" (support-share copy)
- `src/pages/settings/Settings.messages.ts` — ~15 tooltips + "Percona Alerting", "send … to Percona"
- `src/pages/updates/**`, `src/pages/update-clients/UpdateClients.messages.ts` — many "PMM Server / PMM Client / PMM {version}"
- `src/pages/updates/update-info/UpdateInfo.messages.ts`, `update-card/`, `update-in-progress-card/`
- `src/contexts/tour/steps/product.messages.ts`, `alerting.messages.ts` — "Percona Dashboards", "Percona Templates", "Percona offers…", "Manage PMM's…"
- `src/pages/rta/overview/details-pane/QueryAndDetails.messages.ts` — "The PMM service name…"
- `src/contexts/navigation/navigation.constants.ts` — one branded nav label: `PMM HA` → `PFMM HA` (rest of nav is generic DB terms; no change)

**`dashboards/pmm-app` — dashboard JSON visible strings:**
- ~20 branded titles: "PMM Health", "PMM HA Health Overview", "PMM Query Analytics", "PMM Upgrade", "Connected PMM Agents", "Percona News", etc.
- ~50 panel `description` tooltips mentioning PMM/Percona
- Markdown text panels rendering the full name + embedded logo:
  - `dashboards/PMM Health/Environments_Overview.json:49,88,133`
  - `dashboards/Kubernetes (experimental)/Databases_on_Kubernetes.json`
  - Valkey dashboard footers (docs/blog/forum links)
- Plugin nav include names: `plugin.json:298` "PMM Health", `:303` "PMM HA Health Overview"

### 2. Title & favicon (~4 spots)
- `ui/apps/pmm/index.html:7` — `<title>Percona Monitoring and Management</title>`
- `ui/apps/pmm/index.html:5` — favicon `href="/favicon.ico"`
- `ui/apps/pmm/src/lib/constants.ts:4` + `src/utils/document.utils.ts` — runtime `document.title` (`PMM_TITLE`)
- `ui/apps/pmm/public/favicon.ico` — needs new art (O2)

### 3. Brand assets (logos/icons) — needs new art (O2)
Visible logos (content + filename both carry brand):
- `ui/apps/pmm/src/icons/pmm-titled.svg` — sidebar wordmark (primary)
- `ui/apps/pmm/src/icons/pmm-rounded.svg` — app-bar mark
- `ui/apps/pmm/src/icons/pmm-titled-outlined.svg` — Updates / WelcomeCard
- `ui/apps/pmm/src/icons/percona.svg` — release notes
- `ui/apps/pmm-compat/src/img/pmm-logo.svg` — Grafana plugin catalog logo
- `dashboards/pmm-app/src/img/pmm-logo.svg` (+ duplicate `src/shared/assets/pmm-logo.svg`) — plugin + dashboard header
- `ui/apps/pmm/public/favicon.ico`

Filenames carry brand but content is a generic DB glyph (leave content, rename optional → treat as internal, §6):
- `percona-{my,mo,po,va,intelligence}.svg` (MySQL/Mongo/Postgres/Valkey/Advisors nav icons)

Referenced via: `ui/apps/pmm/src/icons/Icon.constants.ts` import map; `plugin.json` logo fields; CSS `.pmm-logo` (`dashboards/pmm-app/src/shared/styles.scss:67`); dashboard markdown `<img src="…/pmm-app/img/pmm-logo.svg">`.

### 4. Grafana plugin metadata
- `dashboards/pmm-app/src/plugin.json` — `name:"PMM"`, `description:"Percona Management and Monitoring dashboards"`, `author.name:"Percona"`, keywords, logo paths, project/license links
- `ui/apps/pmm-compat/src/plugin.json` — `name:"PMM Compatibility"`, description "…layer between Grafana and PMM UI", author, keywords, github links
- Plugin **ids** (`pmm-app`, `pmm-compat-app`, `pmm-qan-app-panel`) → **out of scope** (D6): backend/Grafana routing depends on them.

### 5. External Percona URLs (~96) — replace + log (D5)
Central constants to route through a single placeholder + tracking file:
- `ui/apps/pmm/src/lib/constants.ts:14,15,16,54,55` (`PMM_SUPPORT_URL`, docs/QAN/forums)
- `ui/apps/pmm/src/pages/settings/Settings.messages.ts` (~11 doc links)
- `ui/apps/pmm/src/pages/help-center/HelpCenter.constants.ts:33,48,63,123`
- `ui/apps/pmm/src/contexts/tour/steps/product.steps.tsx:57,75,100,124`
- `ui/apps/pmm/src/pages/updates/update-card/UpdateCard.constants.ts:2,5,7`
- `ui/apps/pmm/src/pages/settings/components/advanced/Advanced.constants.ts:29`
- `ui/apps/pmm/src/pages/rta/selection/RealtimeSelection.constants.ts:4,5`
- Dashboard JSON: 65 URLs (`www.percona.com`, `per.co.na`, `docs.percona.com`, `forums.percona.com`) + RSS `feedUrl:"/percona-blog/feed"`
- Plugin manifests: `github.com/percona/pmm` project/license links
- **Deliverable:** `docs/scratch/260704-ui-rebranding-urls.md` — every original URL, its file:line, and the placeholder used.

**O1 — third-party product names to LEAVE (verify):** `Percona Server`, `Percona XtraDB / Galera Cluster Size`, `Percona Custom Resources - Clusters/Backups`, `Percona Clusters`, `Percona Clusters not Ready`, "Query Response Time plugin in Percona Server".

### 6. Internal identifiers — OUT OF SCOPE (D6), listed for the record
~400+ occurrences. Not user-visible; several load-bearing:
- Route base `/pmm-ui` (visible in URL bar but backend/router-coupled), `PMM_*` path constants, `/graph/d/pmm-home`
- `pmm-ui.*` localStorage/session keys, `pmm-{light,dark}-theme` CSS classes
- `data-testid="pmm-*"` (~75+ in `ui/`, plus dashboards) — tests depend on them
- Package names (`ui`, `pmm-app`, `pmm-compat`, `@pmm/shared`), authors "Percona"/"Percona LLC"
- Plugin ids, panel types (`pmm-check-panel`, `pmm-update-panel`, `pmm-qan-app-panel`)
- API/domain flags (`isPMMAdmin`, `snoozedPmmVersion`, `pmmPublicAddress`)
- Dashboard data-layer: `pmm-server` node, `pmm_annotation`, `pmm."metrics"`, UIDs `pmm-home`/`pmm-qan` (most of the 74-file hit count)

---

## Proposed execution order

1. **Centralize + placeholder URLs** (§5) → produce the URL review file. Do this first so text edits don't fight URL edits.
2. **`ui/apps/pmm` visible text** (§1) — messages/constants files, then JSX.
3. **Title + favicon reference** (§2).
4. **Plugin metadata** (§4) — both `plugin.json`s.
5. **Dashboard JSON** visible titles/tooltips/markdown (§1, dashboards) — scripted where safe, hand-checked for the third-party names (O1).
6. **Assets** (§3) — pending O2 (real art vs. placeholder SVGs).

Baseline tests to run per surface before/after: `ui/` via turbo/yarn; `dashboards/pmm-app` via its yarn/jest. (Not yet run — establish green baseline before edits.)
