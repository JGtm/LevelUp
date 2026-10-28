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
| 3 | **i18n TOML manifests** | `apps/web/src/lib/i18n/manifests/*.toml` | Single source of truth for all UI strings (one entry per language of `config/locales.toml`) |
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

All UI strings in `apps/web/src/features/**` and `apps/web/src/components/**` come from manifests in `apps/web/src/lib/i18n/manifests/`. A leaf carries one entry per language listed in `config/locales.toml` (`fr` and `en` today; to add one, see 3.5).

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

**Lint** : `npm run lint` errors out on hardcoded JSX strings inside `features/` or `components/`. `node tools/lint-no-hardcoded-fields.mjs` runs in the lefthook **pre-push** hook (`lefthook.yml`) and detects label collisions with `fields.toml`.

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

### 3.5 Adding a language (ADR 0040)

A language costs one entry in a list, the translations and the data: no code changes. A language that
is listed and incomplete is refused by the generator, the compiler or a test; it is never served
through a silent fallback. Worked example: Spanish (`es`).

1. **List it** in `config/locales.toml`, the single source of the languages (the order of the entries is
   the display order; `fr` stays the `default`, `en` the `fallback` of game names):

   ```toml
   [[locale]]
   code = "es"                  # primary BCP-47 subtag, lower case: key of every per-language table
   intl = "es-ES"               # tag handed to Intl.* (numbers, dates); its primary subtag must be `code`
   halo = ["es-ES", "es-MX"]    # Halo tags of asset_translations, by preference
   required = true              # true: curated texts must cover it (see step 4); false: they fall back to `en`
   ```

   A language that ships an interface is `required = true`: with `false`, the curated labels (metric
   names, outcomes, Prestige levels) silently stay English. The primary subtag of `intl` must equal
   `code`: some callers hand an Intl tag where a language is expected (`normalizeLocale("es-ES")`), so
   a tag such as `en-GB` on a language other than `en` is read back as `en`.

   For `populate-assets` to fetch it (step 5), each `halo` tag must be one of `domain.TargetLanguages`
   (`apps/go-api/internal/domain/asset_langs.go`, the 14 languages of Halo Infinite).

2. **Generate** the two derived lists and the manifest modules:

   ```bash
   node apps/web/scripts/build_i18n_manifests.mjs
   ```

   It rewrites `apps/web/src/lib/i18n/generated/locales.ts` and `apps/go-api/internal/i18n/locales_gen.go`
   (both committed), then every manifest module, and stops at the first leaf that lacks the language with
   `<file> : cle "<path>" : langue manquante "<code>"`. `TestGeneratedTableMatchesLocalesTOML` (Go) and
   `generatedLocalesFresh.guard.test.ts` (web) fail on any difference between the generated files and
   `locales.toml`.

3. **Translate the interface (web).** Add `es = "..."` to every leaf of
   `apps/web/src/lib/i18n/manifests/*.toml` (ICU placeholders unchanged), then regenerate (step 2). Each
   feature dictionary under `features/<feature>/i18n/` gets an `es.ts` typed by its `contract.ts` and one
   line in its `index.ts` (`Record<Locale, T>`); `npx tsc -b` names every dictionary that lacks the
   language, and `lib/i18n/locale-keyed-objects.guard.test.ts` requires every other locale-keyed object to
   be typed by `Locale`. `tsc` also names the test fixtures typed by the generated contract (the replay
   labels carry one member per language: about twenty `*.test.ts(x)` files under `features/match-replay/`).
   Per-language data that is not text needs its own files: the announcer sound stems of
   `features/match-replay/sound/` need one `.wav` per language (`replaySoundAssets.guard.test.ts`). The
   language selector of the Settings page lists `fr` and `en` only (ADR 0040, out of scope of the plan): it
   belongs to the first real language.

4. **Translate what the server writes (Go).**
   - Message catalogue: add `es = "..."` to every message of `apps/go-api/internal/i18n/messages/*.toml`,
     with the same `{variables}`; `TestCatalogueComplet` (`go test ./internal/i18n/`) lists what is missing.
   - Curated per-language tables (`config/titles/{slug}/mappings/*.toml`, `internal/ops/citation_texts.toml`,
     `config/prestige/tuning.toml` (`levels.names`), ...): a `required` language is mandatory (the loader refuses a missing language and an unknown code,
     `i18n.Text.Validate`); a listed language that is not `required` is optional and falls back to `en`
     (counter `levelup.i18n.text_fallback_total`).
   - Optional documents: `docs/ES/RELEASE_NOTES.md` (absent: the English notes are served after a notice
     written in the requested language, counter `levelup.i18n.document_fallback_total`) and a demo
     fixture `internal/service/demo_fixtures/challenges.es.json`.

5. **Game names.** The names of maps, playlists, pairs and game variants come from the Halo API:

   ```bash
   levelup populate-assets --langs es-ES,es-MX --dry-run   # then without --dry-run
   ```

   It fills `asset_translations` (one row per asset and Halo tag; the default `--langs` is all 14); the
   readers choose the row of the request language through `i18n.Text.Pick` and fall back to `en`, with no
   code and no column. The names written by hand (medals, achievements, milestones, citations, weapons...)
   are rows of `medal_translations` / `asset_translations` or entries of the per-language TOML tables of
   step 4; the administration form offers one input per listed language.

6. **Regenerate the contract and check.** The replay labels are one property per language in the OpenAPI
   document: `make openapi-gen && make generate-types` (`TestOpenAPIYAMLIsUpToDate` refuses a stale
   `api/openapi.yaml`). Then `cd apps/go-api && go test ./internal/i18n/... ./internal/archlint/...`,
   `cd apps/web && npx tsc -b && npx vitest run` and `node tools/check-generated-types-fresh.mjs`.

| What is missing for a listed language | Who refuses |
|---|---|
| a manifest leaf | the generator (step 2) |
| a feature dictionary | `tsc` (`Record<Locale, T>`) |
| a catalogue message or a `{variable}` | `TestCatalogueComplet` |
| a `required` language in a curated table, or an unknown code | the loader at boot |
| a game name | nobody: it falls back to `en` and is counted |
| a generated file out of date | the two freshness tests (step 2) |

**Proof mode.** `node apps/web/scripts/build_i18n_manifests.mjs --pseudo xx [--out <dir>]` fabricates a
language `xx` from `en` (text between `⟦` and `⟧`, ICU placeholders intact) without touching
`config/locales.toml` or any committed file: the modules, `locales.ts` and `locales_gen.go` go to `--out`
(a temporary folder when omitted). Any visible text outside the brackets in the application is a string
that does not go through a manifest. It is a proof tool, never a shipped language. It writes the generated modules only: to run the
application in `xx`, the same files as for a real language (steps 1 to 6) are needed, derived from `en`
(throwaway-branch procedure and results: `.ai/V8/PLAN_I18N_MULTILANGUE.md`, step 10). The unknown-language
code used by the tests is `zz`, never `xx`.

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
| `docs/adr/0040-i18n-n-locales.md` | Why one list of languages (`config/locales.toml`) and how to add a language (§ 3.5) |
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
