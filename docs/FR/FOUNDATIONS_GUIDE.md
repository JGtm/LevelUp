# Guide des fondations

> Guide d'onboarding pour tout dev qui doit écrire un nouveau service de page ou un chart dans cette codebase.
>
> Temps de lecture : ~15 min. Après la lecture, tu dois être capable d'ajouter une nouvelle page qui consomme les données Halo, rend des charts ECharts, et respecte les contrats multi-titres + i18n — sans réinventer les couches existantes.

---

## 1. Pourquoi ce guide existe

La migration Go + React de LevelUp repose sur **quatre fondations transverses** consommées par chaque service de page :

| # | Fondation | Source de vérité | Rôle |
|---|---|---|---|
| 1 | **Types canoniques** | `apps/go-api/internal/games/canonical/` | Shapes title-agnostiques pour la donnée entre couches |
| 2 | **Pattern adapter** | `apps/go-api/internal/games/adapter.go` | 2 interfaces (`TitleDataAdapter` + `TitleSemanticAdapter`) — séparation données vs libellés |
| 3 | **Manifests TOML i18n** | `apps/web/src/lib/i18n/manifests/*.toml` | Source unique pour toutes les strings UI (FR + EN) |
| 4 | **Wrappers ECharts** | `apps/web/src/components/charts/*.tsx` | Composants charts client-side réutilisables |

Ces fondations ont été figées en Phase 0 du `.ai/V7/PLAN_META_FOUNDATIONS_GO.md` et validées sur 8 migrations de pages (Phases 1–3). Elles sont stables. **Ne les réinvente pas.**

---

## 2. Architecture en couches (côté Go)

```
apps/go-api/internal/
├── api/handlers/   ← HTTP : decode requête, appelle service via port, encode JSON
├── api/middleware/ ← cross-cutting : auth, CSRF, slog, TitleExtractor
├── port/           ← interfaces : *Service, Repository… (découplage handler ↔ service)
├── service/        ← orchestration : combine repo + analysis → réponse
├── domain/         ← types purs : structs, enums (zéro DB, zéro HTTP)
├── analysis/       ← algorithmes purs : fonctions stateless
├── games/          ← cœur multi-titres : types canoniques + adapters
├── platform/duckdb/← infrastructure : implémente port.Repository
├── config/         ← config + feature flags
└── ops/            ← backup, restore, diagnose (hors chemin HTTP)
```

**Règle générale** :

- Une fonction qui fait un calcul → `analysis/`.
- Un type partagé entre couches → `domain/` (ou `games/canonical/` si cross-titres).
- Une fonction qui combine DB + algo → `service/`.
- Une fonction qui décode HTTP → appelle service → encode JSON → `api/handlers/`.
- Une requête SQL → `platform/duckdb/`.

Pour l'ensemble des règles, voir le skill `arch-rules` dans `.claude/skills/arch-rules/SKILL.md`.

---

## 3. Les quatre fondations en détail

### 3.1 Types canoniques

Définis dans `apps/go-api/internal/games/canonical/`. N'ajoute pas de champs title-specific ici.

Types principaux (non exhaustif) :

```go
canonical.MatchSummary        // lignes liste/historique
canonical.MatchDetail         // page détail d'un match
canonical.MatchParticipant    // ligne scoreboard
canonical.PlayerStats         // stats agrégées
canonical.PlayerIdentity      // XUID + gamertag + emblème
canonical.CareerSnapshot      // rang + progression XP
canonical.AssetReference      // mode/map/playlist localisé
canonical.Outcome             // enum : Win / Loss / Tie / DNF
canonical.MatchType           // enum : Ranked / Social / Custom / Firefight
```

Les constantes **FieldKey** (`canonical/fields.go`) nomment les champs de données qui référencent les libellés TOML dans `config/titles/{slug}/mappings/fields.toml`. Exemple : `kills`, `deaths`, `accuracy`, `kda`, `kdr`, `team_mmr`.

Pour le catalogue complet, voir le skill `canonical-types`.

### 3.2 Pattern adapter (multi-titres)

Deux interfaces séparées par responsabilité unique :

```go
// TitleDataAdapter — charge les données canoniques depuis la DuckDB title-specific
type TitleDataAdapter interface {
    LoadMatchSummaries(ctx, []string) ([]canonical.MatchSummary, error)
    LoadMatchDetail(ctx, string) (*canonical.MatchDetail, error)
    LoadPlayerStats(ctx, string, StatsScope) (*canonical.PlayerStats, error)
    // ...
}

// TitleSemanticAdapter — expose libellés + assets + outcomes (TOML read-only)
type TitleSemanticAdapter interface {
    Fields() *mappings.FieldMappingSet
    Assets() *mappings.AssetMappingSet
    Outcomes() *mappings.OutcomeMappingSet
}
```

Injecte-les dans ton service via le `Resolver` :

```go
type MyService struct {
    data     games.TitleDataAdapter
    semantic games.TitleSemanticAdapter
}

func (s *MyService) GetPage(ctx context.Context) (*domain.MyPage, error) {
    summaries, err := s.data.LoadMatchSummaries(ctx, ids)
    if errors.Is(err, games.ErrCapabilityNotSupported) {
        // dégradation gracieuse — réponse partielle, jamais panic
        return &domain.MyPage{HasData: false}, nil
    }
    // ...
}
```

Pour les fichiers TOML (`fields.toml`, `assets.toml`, `outcomes.toml`), voir l'ADR 0002 + skill `arch-rules`.

### 3.3 Manifests TOML i18n

Toutes les strings UI dans `apps/web/src/features/**` et `apps/web/src/components/**` viennent des manifests dans `apps/web/src/lib/i18n/manifests/`.

**Workflow** :

1. Ajouter la clé dans le manifest concerné (par ex. `home.toml`) :

   ```toml
   [home.foo.bar]
   fr = "Bonjour {name}"
   en = "Hello {name}"
   ```

2. Régénérer le module TS :

   ```bash
   node apps/web/scripts/build_i18n_manifests.mjs
   ```

   Produit `apps/web/src/lib/i18n/generated/home.ts` avec la const `homeManifest` + le type `HomeManifestKey`.

3. Consommer dans ton composant :

   ```tsx
   import { formatMessage } from '@/lib/i18n/format'
   import { homeManifest } from '@/lib/i18n/generated/home'
   import { useAppShellStore } from '@/stores/appShellStore'

   const locale = useAppShellStore((s) => s.locale)
   const text = formatMessage(homeManifest, 'home.foo.bar', locale, { name: 'World' })
   ```

**ICU MessageFormat** est supporté pour les pluriels + interpolation : `{n, plural, one {# match} other {# matchs}}`.

**Lint** : `npm run lint` échoue sur les strings JSX hardcodées dans `features/` ou `components/`. `node tools/lint-no-hardcoded-fields.mjs` tourne en pre-commit et détecte les collisions de libellés avec `fields.toml`.

Pour les règles + allow-list, voir l'ADR 0003.

### 3.4 Wrappers ECharts

11 wrappers réutilisables (9 dans `apps/web/src/components/charts/`, 2 page-specific dans `features/timeseries/`) :

| Wrapper | Cas d'usage |
|---|---|
| `<TimeseriesLineChart>` | lignes multi-séries time / category / value |
| `<BarStackedChart>` | barres empilées (outcomes, components) |
| `<BarGroupedChart>` | barres côte-à-côte (filtré vs total) |
| `<HistogramChart>` | distribution buckets |
| `<ScatterChart>` | scatter multi-séries (corrélations) |
| `<DonutChart>` | pie/donut avec slice colors sémantiques |
| `<Heatmap2DChart>` | heatmap 2D avec palette sequential ou divergent |
| `<RadarChart>` | radar 6 axes N séries (narrative engine) |
| `<OutcomeSequenceTape>` | bande narrative RLE des outcomes récents |
| `<TimeseriesCombatYield>` | OC + DR avec markLines de référence p80 |
| `<TimeseriesKdaBars>` | bars K + bars D + line K/D ratio (dual yAxis) |

**Sandbox visuel** : `npm run dev` puis va sur `/lab/charts` pour des samples des 11.

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

Chaque wrapper expose un builder pur `buildXxxOption` (testable sans React) — voir ADR 0001.

Pour les tokens couleur (`perf-tier-2`, `outcome-win`, etc.), voir le skill `color-tokens`.

---

## 4. Exemple de bout en bout

Tu veux ajouter une page "Adversaires fréquents" qui liste les joueurs que tu as croisés le plus souvent dans les 30 derniers jours, avec un graphe montrant K/D contre chacun.

### 4.1 Backend (Go)

```go
// 1. Service consomme les types canoniques via les adapters
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

// 2. Handler bind HTTP, zéro logique métier
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

1. **Ajouter les entrées du manifest** (`apps/web/src/lib/i18n/manifests/enemies.toml`) :

   ```toml
   [enemies.title]
   fr = "Adversaires fréquents"
   en = "Frequent enemies"

   [enemies.empty]
   fr = "Aucun adversaire récurrent sur les 30 derniers jours."
   en = "No recurring enemy in the last 30 days."
   ```

2. **Régénérer** :

   ```bash
   node apps/web/scripts/build_i18n_manifests.mjs
   ```

3. **Composant page** :

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

C'est tout — multi-titres ready, i18n FR/EN, wrapper ECharts, zéro hex code.

---

## 5. FAQ

**Q : J'ai besoin d'un chart qui ne rentre dans aucun des 11 wrappers. Comment je fais ?**
R : Trois options, par ordre de préférence :

1. Composer des wrappers existants dans un composant parent (ex : `TimeseriesCombatYield` compose `<ChartCard>` + buildOption custom).
2. Ajouter un helper `composeXxx` dans `_utils.ts` si la pièce manquante est petite (axe, tooltip).
3. Créer un nouveau wrapper dans `components/charts/` uniquement si la viz est fondamentalement nouvelle et réutilisable. Ajouter des tests pour le builder pur.

**Q : Ma page a un libellé qui n'a pas de FieldKey canonique. Où il vit ?**
R : Dans le manifest spécifique à la page (`apps/web/src/lib/i18n/manifests/<page>.toml`). N'ajoute pas dans `fields.toml` sauf si le même libellé apparaît sur plusieurs pages et est title-specific.

**Q : Comment je dégrade gracieusement quand un titre ne supporte pas une feature ?**
R : Les adapters retournent `games.ErrCapabilityNotSupported`. Catch-le dans ton service et retourne une réponse partielle avec `HasData: false` (ou flag équivalent). Le frontend rend un `<EmptyStateCard>` ou `<CapabilityGap>` avec texte explicatif.

**Q : Je peux écrire un service qui appelle un autre service ?**
R : Non (anti-pattern, cf. skill `arch-rules`). Si deux services ont besoin de la même logique, extrais-la dans `analysis/` (pur) ou `service/shared.go` (helper d'orchestration sans state).

**Q : Où je mets les fixtures pour les tests ?**
R :
- Tests d'algorithmes purs : inline dans le fichier de test (`*_test.go`).
- Tests de service : mock `port.Repository` via interface.
- Tests d'intégration avec DuckDB : DB `:memory:` + tag `//go:build integration`.
- Frontend : `apps/web/src/test/handlers.ts` pour les mocks MSW.

**Q : J'ai ajouté un nouveau chart wrapper. Où je le liste ?**
R :
- Ajoute-le à `apps/web/src/features/lab/ChartsShowcasePage.tsx` (sandbox visuel).
- Update `apps/web/src/components/charts/README.md` (catalogue).
- Si le chart est page-specific (comme `TimeseriesCombatYield`), garde-le dans `features/<page>/` plutôt que `components/charts/`.

**Q : Je construis un panneau/une section pour le dashboard monitoring admin. Quelles primitives utiliser ?**
R : Le dashboard admin (refonte 2026-07) a ses propres primitives canoniques sous
`apps/web/src/features/admin/components/` — les utiliser au lieu de re-déclarer des
composants locaux (garde-rail : `admin-ui.guard.test.ts`) :

- `AdminKpi` — LA carte KPI unique (enveloppe la primitive foundations `KpiCard` ;
  props : label, value, accent, delta, sub, size, to). Ne jamais déclarer un `*Kpi` local.
- `SectionHeader` — titre de section (caps muted) + description/actions optionnelles.
- `AdminTable` / `AdminTh` / `AdminTr` / `AdminTd` — tables natives statiques.
  Les tables interactives (tri/filtre/actions) restent TanStack Table (`DetectionsPanel`).
- `useCounterSnapshot(storageKey, generatedAt, build)` — LE hook du pattern « baseline
  roulante » localStorage (delta vs visite précédente). Ne jamais appeler
  `readCountersSnapshot`/`writeCountersSnapshot` directement hors de `countersTrend.ts`.

---

## 6. Références

| Doc | Rôle |
|---|---|
| `docs/adr/0001-charts-stack-echarts.md` | Pourquoi ECharts (contexte de la décision) |
| `docs/adr/0002-canonical-player-match-row.md` | Pourquoi les types canoniques |
| `docs/adr/0003-i18n-manifest-and-linter.md` | Pourquoi les manifests TOML + lint |
| `docs/adr/0004-narrative-engine.md` | Pourquoi 8 rôles + radar 6 axes |
| `.ai/V7/PLAN_META_FOUNDATIONS_GO.md` | Plan maître (Phases 0–4) |
| `.claude/skills/arch-rules/SKILL.md` | Règles de couches + contrat multi-titres |
| `.claude/skills/canonical-types/SKILL.md` | Catalogue des types |
| `.claude/skills/color-tokens/SKILL.md` | Système de tokens couleur |
| `.claude/skills/foundations-usage/SKILL.md` | Checklist rapide quand tu écris du code |
| `apps/web/src/components/charts/README.md` | Catalogue des wrappers |
| `apps/go-api/internal/analysis/temporal/README.md` | Helpers temporels |
| `apps/go-api/internal/analysis/breakdown/README.md` | Décomposition par carte/mode |
| `apps/go-api/internal/analysis/narrative/README.md` | Narrative engine |
