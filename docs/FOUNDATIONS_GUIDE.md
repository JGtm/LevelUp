# Foundations Guide

> Onboarding guide for any developer who needs to write a new page service or chart in this codebase.
>
> Read time: ~15 min. After reading, you should be able to add a new page that consumes Halo data, renders ECharts, and respects multi-title + i18n contracts — without re-inventing existing layers.

---

## 1. Why this guide exists

The Go + React migration of LevelUp is built on **four cross-cutting foundations** that every page service consumes :

| # | Foundation | Source of truth | Purpose |
|---|---|---|---|
| 1 | **Canonical types** | `apps/go-api/internal/games/canonical/` | Title-agnostic shapes for data flowing between layers |
| 2 | **Adapter pattern** | `apps/go-api/internal/games/adapter.go` | Two interfaces (`TitleDataAdapter` + `TitleSemanticAdapter`) — separation data vs labels |
| 3 | **i18n TOML manifests** | `apps/web/src/lib/i18n/manifests/*.toml` | Single source of truth for all UI strings (FR + EN) |
| 4 | **ECharts wrappers** | `apps/web/src/components/charts/*.tsx` | Reusable client-side chart components |

These foundations were locked in Phase 0 of `.ai/V7/PLAN_META_FOUNDATIONS_GO.md` and validated across 8 page migrations (Phases 1–3). They are stable. **Don't reinvent them.**

---

## 2. Layered architecture (Go side)

```
apps/go-api/internal/
├── api/handlers/   ← HTTP : decode request, call service via port, encode JSON
├── api/middleware/ ← cross-cutting : auth, CSRF, slog, TitleExtractor
├── port/           ← interfaces : *Service, Repository… (handler ↔ service decoupling)
├── service/        ← orchestration : combine repo + analysis → response
├── domain/         ← pure types : structs, enums (no DB, no HTTP)
├── analysis/       ← pure algorithms : stateless functions
├── games/          ← multi-title core : canonical types + adapters
├── platform/duckdb/← infrastructure : implements port.Repository
├── config/         ← config + feature flags
└── ops/            ← backup, restore, diagnose (off the HTTP path)
```

**Rule of thumb** :

- A function that does a calculation → `analysis/`.
- A type shared between layers → `domain/` (or `games/canonical/` if cross-title).
- A function that combines DB + algo → `service/`.
- A function that decodes HTTP → calls service → encodes JSON → `api/handlers/`.
- A SQL query → `platform/duckdb/`.

For the full set of rules, see the `arch-rules` skill in `.claude/skills/arch-rules/SKILL.md`.

---

## 3. The four foundations in detail

### 3.1 Canonical types

Defined in `apps/go-api/internal/games/canonical/`. Don't add title-specific fields here.

Core types (non-exhaustive) :

```go
canonical.MatchSummary        // list/history rows
canonical.MatchDetail         // single-match detail page
canonical.MatchParticipant    // scoreboard row
canonical.PlayerStats         // aggregated player stats
canonical.PlayerIdentity      // XUID + gamertag + emblem
canonical.CareerSnapshot      // rank + XP progression
canonical.AssetReference      // localized mode/map/playlist
canonical.Outcome             // enum: Win / Loss / Tie / DNF
canonical.MatchType           // enum: Ranked / Social / Custom / Firefight
```

**FieldKey** constants (`canonical/fields.go`) name the data fields that reference TOML labels in `config/titles/{slug}/mappings/fields.toml`. Example: `kills`, `deaths`, `accuracy`, `kda`, `kdr`, `team_mmr`.

For the full type catalog, see the `canonical-types` skill.

### 3.2 Adapter pattern (multi-title)

Two interfaces, separated by Single Responsibility :

```go
// TitleDataAdapter — loads canonical data from title-specific DuckDB
type TitleDataAdapter interface {
    LoadMatchSummaries(ctx, []string) ([]canonical.MatchSummary, error)
    LoadMatchDetail(ctx, string) (*canonical.MatchDetail, error)
    LoadPlayerStats(ctx, string, StatsScope) (*canonical.PlayerStats, error)
    // ...
}

// TitleSemanticAdapter — exposes labels + assets + outcomes (read-only TOML)
type TitleSemanticAdapter interface {
    Fields() *mappings.FieldMappingSet
    Assets() *mappings.AssetMappingSet
    Outcomes() *mappings.OutcomeMappingSet
}
```

Inject them in your service via the `Resolver` :

```go
type MyService struct {
    data     games.TitleDataAdapter
    semantic games.TitleSemanticAdapter
}

func (s *MyService) GetPage(ctx context.Context) (*domain.MyPage, error) {
    summaries, err := s.data.LoadMatchSummaries(ctx, ids)
    if errors.Is(err, games.ErrCapabilityNotSupported) {
        // graceful degradation — return partial response, never panic
        return &domain.MyPage{HasData: false}, nil
    }
    // ...
}
```

For TOML files (`fields.toml`, `assets.toml`, `outcomes.toml`), see ADR 0002 + `arch-rules` skill.

### 3.3 i18n TOML manifests

All UI strings in `apps/web/src/features/**` and `apps/web/src/components/**` come from manifests in `apps/web/src/lib/i18n/manifests/`.

**Workflow** :

1. Add the key in the relevant manifest (e.g. `home.toml`) :

   ```toml
   [home.foo.bar]
   fr = "Bonjour {name}"
   en = "Hello {name}"
   ```

2. Regenerate the TS module :

   ```bash
   node apps/web/scripts/build_i18n_manifests.mjs
   ```

   Produces `apps/web/src/lib/i18n/generated/home.ts` with `homeManifest` const + `HomeManifestKey` type.

3. Consume in your component :

   ```tsx
   import { formatMessage } from '@/lib/i18n/format'
   import { homeManifest } from '@/lib/i18n/generated/home'
   import { useAppShellStore } from '@/stores/appShellStore'

   const locale = useAppShellStore((s) => s.locale)
   const text = formatMessage(homeManifest, 'home.foo.bar', locale, { name: 'World' })
   ```

**ICU MessageFormat** is supported for plurals + interpolation : `{n, plural, one {# match} other {# matches}}`.

**Lint** : `npm run lint` errors out on hardcoded JSX strings inside `features/` or `components/`. `node tools/lint-no-hardcoded-fields.mjs` runs in pre-commit and detects label collisions with `fields.toml`.

For the rules + allow-list, see ADR 0003.

### 3.4 ECharts wrappers

11 reusable wrappers (9 in `apps/web/src/components/charts/`, 2 page-specific in `features/timeseries/`):

| Wrapper | Use case |
|---|---|
| `<TimeseriesLineChart>` | multi-series time / category / value lines |
| `<BarStackedChart>` | stacked bars (outcomes, components) |
| `<BarGroupedChart>` | side-by-side bars (filtered vs total) |
| `<HistogramChart>` | distribution buckets |
| `<ScatterChart>` | multi-series correlation scatter |
| `<DonutChart>` | pie/donut with semantic slice colors |
| `<Heatmap2DChart>` | 2D heatmap with sequential or divergent palette |
| `<RadarChart>` | N-series 6-axis radar (narrative engine) |
| `<OutcomeSequenceTape>` | RLE narrative band of recent outcomes |
| `<TimeseriesCombatYield>` | OC + DR with p80 reference markLines |
| `<TimeseriesKdaBars>` | bars K + bars D + line K/D ratio (dual yAxis) |

**Live sandbox** : `npm run dev` then navigate to `/lab/charts` for samples of all 11.

**Pattern** :

```tsx
import { HistogramChart } from '@/components/charts/HistogramChart'
import { distributionBucketsToSeries } from '@/features/timeseries/seriesAdapters'

<HistogramChart
  series={distributionBucketsToSeries(buckets, { key: 'demo', name: 'KD' })}
  colorToken="perf-tier-2"
  xAxisLabel={t('timeseries.distributions.kda_axis_x')}
  height={280}
/>
```

Each wrapper exposes a pure `buildXxxOption` builder (testable without React) — see ADR 0001.

For the color tokens (`perf-tier-2`, `outcome-win`, etc.), see the `color-tokens` skill.

---

## 4. End-to-end example

You want to add a "Recent enemies" page that lists the players you faced most often in the last 30 days, with a chart showing K/D against each.

### 4.1 Backend (Go)

```go
// 1. Service consumes canonical types via adapters
type EnemiesService struct {
    data games.TitleDataAdapter
}

func (s *EnemiesService) GetEnemies(ctx context.Context, slug string) (*domain.EnemiesPage, error) {
    encounters, err := s.data.LoadEncounters(ctx, slug, canonical.StatsScope{
        From: time.Now().AddDate(0, 0, -30),
        To:   time.Now(),
    })
    if errors.Is(err, games.ErrCapabilityNotSupported) {
        return &domain.EnemiesPage{HasData: false}, nil
    }
    if err != nil {
        slog.ErrorContext(ctx, "load encounters failed", "err", err, "player", slug)
        return nil, err
    }
    return &domain.EnemiesPage{
        HasData:    true,
        Encounters: encounters, // []canonical.EncounterRow
    }, nil
}

// 2. Handler binds HTTP, no business logic
func (h *Handler) GetEnemies(w http.ResponseWriter, r *http.Request) {
    page, err := h.svc.GetEnemies(r.Context(), chi.URLParam(r, "slug"))
    if err != nil {
        writeError(w, http.StatusInternalServerError, "load_failed", err.Error())
        return
    }
    writeJSON(w, http.StatusOK, page)
}
```

### 4.2 Frontend

1. **Add manifest entries** (`apps/web/src/lib/i18n/manifests/enemies.toml`) :

   ```toml
   [enemies.title]
   fr = "Adversaires fréquents"
   en = "Frequent enemies"

   [enemies.empty]
   fr = "Aucun adversaire récurrent sur les 30 derniers jours."
   en = "No recurring enemy in the last 30 days."
   ```

2. **Regenerate** :

   ```bash
   node apps/web/scripts/build_i18n_manifests.mjs
   ```

3. **Page component** :

   ```tsx
   import { formatMessage } from '@/lib/i18n/format'
   import { enemiesManifest } from '@/lib/i18n/generated/enemies'
   import { BarGroupedChart } from '@/components/charts/BarGroupedChart'
   import { useAppShellStore } from '@/stores/appShellStore'

   export function EnemiesPage() {
     const locale = useAppShellStore((s) => s.locale)
     const t = (k) => formatMessage(enemiesManifest, k, locale)
     const { data } = useEnemiesQuery()

     if (!data?.has_data) {
       return <EmptyStateCard title={t('enemies.title')} description={t('enemies.empty')} />
     }

     const series = [{
       key: 'enemies.kd',
       meta: { gamertag: 'enemies' },
       datapoints: data.encounters.map((e) => ({
         category: e.identity.gamertag,
         components: { Wins: e.wins ?? 0, Losses: e.losses ?? 0 },
       })),
     }]

     return (
       <Card>
         <CardHeader>
           <CardTitle>{t('enemies.title')}</CardTitle>
         </CardHeader>
         <CardContent>
           <BarGroupedChart
             series={series}
             componentColors={{ Wins: 'outcome-win', Losses: 'outcome-loss' }}
           />
         </CardContent>
       </Card>
     )
   }
   ```

That's it — multi-title-ready, i18n FR/EN, ECharts wrapper, no hex codes.

---

## 5. FAQ

**Q: I need a chart that doesn't fit any of the 11 wrappers. What do I do?**
A: Three options, in order of preference:

1. Compose existing wrappers in a parent component (e.g. `TimeseriesCombatYield` composes `<ChartCard>` + custom buildOption).
2. Add a `composeXxx` helper in `_utils.ts` if the missing piece is small (axis style, tooltip).
3. Create a new wrapper in `components/charts/` only if the visualization is fundamentally new and reusable. Add tests for the pure builder.

**Q: My page has a label that doesn't have a canonical FieldKey. Where does it live?**
A: In your page-specific manifest (`apps/web/src/lib/i18n/manifests/<page>.toml`). Don't add it to `fields.toml` unless the same label appears across multiple pages and is title-specific.

**Q: How do I degrade gracefully when a title doesn't support a feature?**
A: Adapters return `games.ErrCapabilityNotSupported`. Catch it in your service and return a partial response with `HasData: false` (or equivalent flag). The frontend renders an `<EmptyStateCard>` or `<CapabilityGap>` component with explanatory text.

**Q: Can I write a service that calls another service?**
A: No (anti-pattern, cf. `arch-rules` skill). If two services need the same logic, extract it into `analysis/` (pure) or `service/shared.go` (orchestration helper without state).

**Q: Where do I put fixtures for tests?**
A: 
- Pure algorithm tests : inline in the test file (`*_test.go`).
- Service tests : mock `port.Repository` via interface.
- Integration tests with DuckDB : `:memory:` DB + tag `//go:build integration`.
- Frontend : `apps/web/src/test/handlers.ts` for MSW mocks.

**Q: I added a new chart wrapper. Where should I list it?**
A: 
- Add it to `apps/web/src/features/lab/ChartsShowcasePage.tsx` (visual sandbox).
- Update `apps/web/src/components/charts/README.md` (catalog).
- If the chart is page-specific (like `TimeseriesCombatYield`), keep it in `features/<page>/` instead of `components/charts/`.

**Q: I'm building a panel/section for the admin monitoring dashboard. Which primitives do I use?**
A: The admin dashboard (2026-07 overhaul) has its own canonical primitives under
`apps/web/src/features/admin/components/` — use them instead of re-declaring locals
(guard-rail: `admin-ui.guard.test.ts`):

- `AdminKpi` — the ONLY KPI card (wraps foundations `KpiCard`; props: label, value,
  accent, delta, sub, size, to). Never declare a local `*Kpi` component.
- `SectionHeader` — section title (caps muted) + optional description/actions slot.
- `AdminTable` / `AdminTh` / `AdminTr` / `AdminTd` — static native tables.
  Interactive tables (sort/filter/actions) use TanStack Table (`DetectionsPanel`).
- `useCounterSnapshot(storageKey, generatedAt, build)` — the rolling-baseline
  localStorage delta pattern ("vs previous visit"). Never call
  `readCountersSnapshot`/`writeCountersSnapshot` directly outside `countersTrend.ts`.

---

## 6. References

| Doc | Purpose |
|---|---|
| `docs/adr/0001-charts-stack-echarts.md` | Why ECharts (decision context) |
| `docs/adr/0002-canonical-player-match-row.md` | Why canonical types |
| `docs/adr/0003-i18n-manifest-and-linter.md` | Why TOML manifests + lint |
| `docs/adr/0004-narrative-engine.md` | Why 8 roles + 6-axis radar |
| `.ai/V7/PLAN_META_FOUNDATIONS_GO.md` | Master plan (Phases 0–4) |
| `.claude/skills/arch-rules/SKILL.md` | Layer rules + multi-title contract |
| `.claude/skills/canonical-types/SKILL.md` | Type catalog |
| `.claude/skills/color-tokens/SKILL.md` | Color token system |
| `.claude/skills/foundations-usage/SKILL.md` | Quick checklist when writing new code |
| `apps/web/src/components/charts/README.md` | Chart wrapper catalog |
| `apps/go-api/internal/analysis/temporal/README.md` | Temporal helpers |
| `apps/go-api/internal/analysis/breakdown/README.md` | Breakdown by map/mode |
| `apps/go-api/internal/analysis/narrative/README.md` | Narrative engine |
