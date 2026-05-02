# Axe 6 — DRY / réinvention de roue

Date : 2026-04-29
Branche : feat/multi-title-static-fs-rescope
Périmètre : tous les sous-packages Go + apps/web/src/

## Synthèse (3-5 lignes max)

Couche Go riche en helpers locaux non promus en types canoniques : 4 redéclarations de `OutcomeWin/Loss/Tie/DNF` (avec préfixes `home*`, `Outcome*`, `homeOutcome*`), 6 implémentations de `perfTier`/`perfColor` aux seuils incohérents (80/65/50/35 vs 80/60/40/20), `IsBot(xuid)` réinventé 9 fois en Go+SQL malgré sa documentation. Côté front, 9 helpers `formatDate`/`formatNumber`/`formatPercent`/`formatDuration` ad-hoc dispersés dans 9 features, 3 implémentations de `KPICard`, et surtout `normalizeModeLabel()` réimplémenté en TS (`match-card.tsx`) malgré l'existence de `analysis/mode_label.go` côté Go. Verdict : 1 BLOQUANT (seuils Perf divergents = bug visuel latent), 7 DETTE, 4 AMÉLIORATION.

## Top duplications par famille

| Famille | Implémentations | Fait foi ? | Sévérité |
|---|---|---|---|
| Seuils PerfTier (5 paliers) | `analysis/squad_score.go:114`, `service/squad_service_v2.go:520+642`, `service/match_view_service.go:75`, `service/match_view_service.go:1173`, `domain/chart/base.go:99`, `apps/web/.../scales/instances.ts:22` | TS (80/65/50/35) | **BLOQUANT** |
| Constantes Outcome int | `domain/outcomes.go:6-9`, `analysis/performance_score.go:47-48`, `analysis/comeback.go:40-41`, `analysis/home.go:31-34` | `domain/outcomes.go` (incomplet, manque DNF) | DETTE |
| `outcome int → key string` | `analysis/home.go:63-67`, `lib/outcome-color.ts:16`, `MediaMatchPicker.tsx:51`, `SessionDetailPage.tsx:97`, `SynthesisHighlightsSection.tsx:17` | aucun | DETTE |
| `IsBot(xuid)` | `analysis/scoreboard_extremes.go:88` (Go) + 8 SQL `xuid LIKE 'bid(%'` | aucun helper public | DETTE |
| `normalizeModeLabel(raw, map)` | `analysis/mode_label.go:40` (Go) + `components/ui/match-card.tsx:50-79` (TS, port direct) | Go (a docstring + tests) | DETTE |
| `formatDate`/`Number`/`Percent`/`Duration` (front) | 9 helpers ad-hoc (HomePage, SessionDetail, Lab, Palmares, MatchCard, MediaMatchPicker, MediaViewer, HistoryTable, charts/_utils) | aucun (`format.ts` ne couvre que ICU MessageFormat) | DETTE |
| Seuils SquadGrade (S/A/B/C/D/F) | `analysis/squad_score.go:110` + `service/squad_service_v2.go:638` (`squadGrade`) | dupliqué identique 2× | AMÉLIORATION |
| `KPICard` composant | `components/layout/KPIStrip.tsx:56`, `features/home/HomePage.tsx:47`, `features/squad/SquadLayout.tsx:129`, `features/lab/LabPage.tsx:111` (MetricCard), `features/palmares/SeasonPassPage.tsx:51` (StatCard) | `KPIStrip` (le plus complet) | DETTE |
| Pattern `useState + try/catch + localStorage` | `MatchHistoryPage.tsx:38-54`, `SquadLayout.tsx:236-254`, `SquadLayout.tsx:269-284` | aucun (pas de hook `useLocalStorageState`) | AMÉLIORATION |
| Fragments SQL `JOIN match_participants` | 24 occurrences dans 8 fichiers `platform/duckdb/` | aucun helper | AMÉLIORATION |

## Constats

### [BLOQUANT] Seuils PerfTier divergents entre les 6 implémentations Go

Six fonctions Go classent un score de performance 0..100 en paliers de couleur, mais avec **deux jeux de seuils incompatibles** :

`apps/go-api/internal/service/match_view_service.go:73-85` (`perfColorToken`, 5 paliers) :
```go
case score >= 80: return "perf-tier-1"
case score >= 60: return "perf-tier-2"   // 60, pas 65
case score >= 40: return "perf-tier-3"   // 40, pas 50
case score >= 20: return "perf-tier-4"   // 20, pas 35
```

`apps/go-api/internal/analysis/squad_score.go:108-125` (`resolveSquadGrade`, 5 paliers, alignement TS) :
```go
case score >= 90: return "S"
case score >= 80: return "A"
case score >= 65: return "B"   // 65, pas 60
case score >= 50: return "C"   // 50, pas 40
case score >= 35: return "D"   // 35, pas 20
```

Source TS canonique `apps/web/src/lib/accessibility/scales/instances.ts:19-22` :
```ts
/** Score de performance global (0–100). Seuils : 80 / 65 / 50 / 35. */
export const perfScale = makeOrdinalScale({ thresholds: [80, 65, 50, 35] })
```

Conséquence : la couleur de tier d'un même score (ex. 62) varie selon la surface :
- Match View Go : `perf-tier-2` (vert), `#3b82f6` bleu (`match_view_service.go:1175`)
- Squad Go : grade `D` (35-50), label `poor`
- Front (`apps/web/.../perf-color.ts`) : `perf-tier-3` (jaune)

Le commentaire `apps/web/src/lib/perf-color.ts:8` annonce explicitement « Seuils canoniques : instances.ts (perfScale [80, 65, 50, 35]) » mais le backend Go ne respecte ce seuil que dans `squad_score.go` / `squad_service_v2.go`. Tout consommateur Match View reçoit du backend une coloration différente de celle calculée côté front.

Action : extraire `analysis/perf_tier.go` exposant `PerfTier(score float64) string` retournant un label sémantique aligné sur `perf-tier-{1..5}` avec les seuils canoniques 80/65/50/35, supprimer `perfColorToken`, `perfColor`, `domain/chart/base.go::PerfColor`, ne garder que les implémentations alignées.

---

### [DETTE] Triple/quadruple redéclaration des constantes Outcome

`apps/go-api/internal/domain/outcomes.go:5-10` (le seul candidat canonique, mais incomplet) :
```go
const ( OutcomeUnknown = 0; OutcomeDraw = 1; OutcomeWin = 2; OutcomeLoss = 3 )
// MANQUE : OutcomeDNF = 4
```

`apps/go-api/internal/analysis/performance_score.go:46-49` :
```go
const ( OutcomeWin = 2; OutcomeLoss = 3 )  // re-déclare dans analysis
```

`apps/go-api/internal/analysis/comeback.go:38-42` :
```go
// "OutcomeWin=2, OutcomeLoss=3 déjà déclarées dans performance_score.go"
const ( OutcomeTie = 1; OutcomeDNF = 4 )  // ajoute Tie+DNF dans analysis
```

`apps/go-api/internal/analysis/home.go:30-40` :
```go
const ( homeOutcomeWin = 2; homeOutcomeLoss = 3; homeOutcomeTie = 1; homeOutcomeDNF = 4 )
```

Et 5+ sites comparent encore à des littéraux : `analysis/citations_custom.go:67,83,98,112` (`if ctx.Outcome != 2`), `analysis/match_impact.go:74-76` (`if p.Outcome == 2`), `service/career_service.go:470` (`if r.Outcome == 2 // WIN`), `platform/duckdb/match_history_repo.go:77`, `platform/duckdb/queries_squad.go:12,14,127`, `platform/duckdb/compare_repo.go:35`, `api/post_sync_deltas.go:144`, `platform/duckdb/queries_match.go:363,365`.

Action : compléter `domain/outcomes.go` avec `OutcomeDNF = 4`, supprimer les redéclarations dans `analysis/performance_score.go`, `analysis/comeback.go`, `analysis/home.go`, remplacer `homeOutcomeWin` etc. par `domain.OutcomeWin`. Pour les SQL, créer `internal/platform/duckdb/sql_fragments.go` exposant `const SQLIsWin = "outcome = 2"` ou `const SQLWinExpr = "CASE WHEN outcome = 2 THEN 1 ELSE 0 END"` (équivalent Go du `WIN_RATE_EXPR` Python documenté en CLAUDE.md mais jamais porté).

---

### [DETTE] Mapping outcome int → key dupliqué front + back, divergeant

5 implémentations distinctes du mapping `2/3/1/4 → "win"/"loss"/"tie"/"dnf"` :

`apps/go-api/internal/analysis/home.go:63-67` :
```go
var homeOutcomeTones = map[int]string{
    homeOutcomeWin: "win", homeOutcomeLoss: "loss",
    homeOutcomeTie: "tie", homeOutcomeDNF: "dnf",
}
```

`apps/web/src/lib/outcome-color.ts:16-20` (incomplet, manque DNF) :
```ts
const OUTCOME_KEY: Record<number, 'win' | 'loss' | 'draw' | 'dnf'> = {
  2: 'win', 1: 'draw', 3: 'loss',  // pas de 4 !
}
```

`apps/web/src/features/media/MediaMatchPicker.tsx:49-64` (`outcomeKeyOf`, complet).

`apps/web/src/features/session-detail/SessionDetailPage.tsx:97` (ternaire inline) :
```ts
outcome === 2 ? 'win' : outcome === 3 ? 'loss' : outcome === 1 ? 'tie' : outcome === 4 ? 'dnf' : null
```

`apps/web/src/features/synthesis/SynthesisHighlightsSection.tsx:17` (boolean) :
```ts
const isWin = item.outcome === 2
```

Divergence sémantique : `outcome-color.ts:18` mappe 1 → `'draw'` ; toutes les autres surfaces (Go + TS) utilisent `'tie'`. Le tooltip d'un match en égalité aura donc un label différent selon le composant (charts ECharts → `tie`, badges via `getOutcomeColor` → `draw`).

Action : exposer côté API la clé canonique `outcome_key: "win" | "loss" | "tie" | "dnf"` à côté de `outcome: number` dans tous les DTOs match (canonique déjà défini dans `internal/games/canonical/enums.go:36-37` mais pas propagé). Côté front, supprimer `outcomeKeyOf` et le ternaire inline ; consommer directement `row.outcome_key`. Trancher `tie` vs `draw` (le backend utilise `tie`, le code TS d'accessibilité dit `draw`).

---

### [DETTE] `IsBot(xuid)` documenté mais jamais factorisé

CLAUDE.md, MIGRATION_GAP, plans de sprint mentionnent tous « la convention bot est `xuid LIKE 'bid(%'` ». Pourtant aucun helper `domain.IsBot(xuid string) bool` n'existe :

`apps/go-api/internal/analysis/scoreboard_extremes.go:88` (seule fonction Go) :
```go
if !strings.HasPrefix(r.XUID, "bid(") {
    out = append(out, r)
}
```

8 répétitions SQL : `internal/sync/engagement.go:247-248,288`, `platform/duckdb/engagement_score_repo_queries.go:68-69,130`, `migration/steps_shared.go:345`, plus la regex `xuid NOT LIKE 'bid(%'` partout. Le commit récent du thought_log [2026-04-29 B2] a corrigé 4 bugs où une variante (`is_bot` en colonne, qui n'existe pas) était utilisée à la place.

Action : créer `apps/go-api/internal/domain/bots.go` avec `func IsBot(xuid string) bool { return strings.HasPrefix(xuid, "bid(") }` et `const SQLIsBot = "xuid LIKE 'bid(%'"` + `SQLNotBot = "xuid NOT LIKE 'bid(%'"`. Réécrire les 9 sites de référence pour empêcher les divergences silencieuses du type B2.

---

### [DETTE] `normalizeModeLabel` réimplémenté en TS alors qu'il est en Go avec tests

`apps/go-api/internal/analysis/mode_label.go:27-77` est documenté, testé (`mode_label_test.go`), exposé : `NormalizeModeLabel(raw string, mapLabels ...string) string`. Logique non-triviale : strip ` sur <map>` / ` on <map>`, extraction préfixe `Arena:Slayer` → `Slayer`, strip suffixes `- Forge` / `- Ranked`. Le pipeline canonique est censé livrer `mode_ui` déjà normalisé.

Or `apps/web/src/components/ui/match-card.tsx:26-79` reconstruit la même logique en TS :
```ts
function escapeRegExp(value: string): string { /* ... */ }
function normalizeModeLabel(modeLabel, mapLabel): string | null {
  // strip ' : ' / ':', regex `\\s+(?:on|sur)\\s+${escapedMap}$`,
  // strip /\s*-\s*(?:Forge|Ranked)\b/gi
}
```

Si `mode_ui` était toujours normalisé côté backend (comme prévu dans le PLAN_MULTI_TITLE), ce code TS serait du dead code défensif. S'il ne l'est pas systématiquement (cas connu : home avec `Arena: Quick Play` à nettoyer côté front), c'est un bug d'API : la donnée canonique n'est pas canonique.

Action : auditer les surfaces backend qui exposent `mode_ui` non-normalisé (project_map.md mentionne déjà des fixes home `mode_ui` 2026-04-21), garantir que tous les endpoints émettent du déjà-normalisé, supprimer `normalizeModeLabel` côté TS. Ne pas la garder « par défense » : elle se désynchronisera.

---

### [DETTE] 9 helpers `formatDate/Number/Percent/Duration` ad-hoc côté front

Aucun module central de formatters. Inventaire :
- `apps/web/src/components/ui/match-card.tsx:30` `formatMatchDuration` (`Xm SSs`)
- `apps/web/src/components/ui/match-card.tsx:36` `formatMatchDateTime` (Intl, locale-aware, timezone)
- `apps/web/src/features/squad/v2/components/HistoryTable.tsx:28` `formatDate` (DD/MM/YY)
- `apps/web/src/features/squad/v2/components/HistoryTable.tsx:33` `formatDuration` (`m:SS`)
- `apps/web/src/features/session-detail/SessionDetailPage.tsx:395-407` `formatNumber`, `formatPercent`, `formatShortDateTime`
- `apps/web/src/features/palmares/PalmaresRelationsPage.tsx:27-39` `formatPercent`, `formatKDA`, `formatDate`
- `apps/web/src/features/lab/LabPage.tsx:73-104` `formatDate`, `formatNumber`, `formatDecimal`, `formatBytes`
- `apps/web/src/features/home/HomePage.tsx:166,176` `formatSessionDate`, `formatSessionDuration`
- `apps/web/src/components/charts/_utils.ts:116,124` `formatDateShort`, `formatNumber`

Divergences sémantiques : `formatPercent` SessionDetail (`%.1f%`) ≠ `formatPercent` Palmares (`% LOC `, espace insécable) ≠ `formatPercent` (front non-existant côté axe 1 BLOQUANT). `formatDuration` HistoryTable (`m:SS`) ≠ `formatMatchDuration` match-card (`Xm SSs`).

`apps/web/src/lib/i18n/format.ts` n'expose que `formatMessage()` (ICU). Aucun module `lib/format/numbers.ts` ou `lib/format/dates.ts`.

Action : créer `apps/web/src/lib/format/{numbers,dates,durations}.ts` avec `formatPercent(ratio, locale, opts?)`, `formatRatio(value, locale, decimals=2)`, `formatDuration(seconds, format: 'hms'|'ms'|'compact')`, `formatDateShort(d, locale)`, `formatDateTime(d, locale, tz?)`. Supprimer les 9 helpers locaux. Cohérent avec le BLOQUANT axe 1 (helper `formatPercent(ratio, decimals)`).

---

### [DETTE] 3+ implémentations distinctes de `KPICard`

`apps/web/src/components/layout/KPIStrip.tsx:56-87` : `KPICard` complet (label, primary, secondary, trend arrow, custom slot, wide).

`apps/web/src/features/home/HomePage.tsx:47-54` : `KPICard` minimal (label + value + compact).

`apps/web/src/features/squad/SquadLayout.tsx:129-136` : `KPICard` minimal (label + value).

`apps/web/src/features/lab/LabPage.tsx:111-119` : `MetricCard` (label + value + hint).

`apps/web/src/features/palmares/SeasonPassPage.tsx:51-60` : `StatCard` (label + value, classes différentes).

Tous rendent essentiellement « label + valeur » avec des marges, padding et radius légèrement différents. `KPIStrip` (le plus capable) est ignoré par 4 features qui réinventent leur version.

Action : promouvoir `components/ui/kpi-card.tsx` exposant `<KPICard label value secondary? trend? hint? variant?>`. Réécrire les 4 sites pour consommer ce composant. Ne pas garder `MetricCard` et `StatCard` si pas de besoin sémantique distinct.

---

### [DETTE] Pattern `useState + localStorage + try/catch JSON.parse` dupliqué

`apps/web/src/features/match-history/MatchHistoryPage.tsx:38-54`, `apps/web/src/features/squad/SquadLayout.tsx:236-254`, `apps/web/src/features/squad/SquadLayout.tsx:269-284` : trois copies quasi identiques du même pattern (init lazy depuis localStorage avec try/catch, setter qui sérialise avec try/catch). Pas d'effet (sync au render), pas de listener d'événement `storage` cross-tab.

Aucun hook `useLocalStorageState<T>(key, default)` dans `apps/web/src/lib/`. Pourtant Zustand persist est dispo (`@types` ok), et `globalFilterStore.ts` l'utilise (`persist`).

Action : extraire `apps/web/src/lib/hooks/useLocalStorageState.ts<T>(key, defaultValue)` qui encapsule lazy-init + setter sérialisant + (optionnel) sync `storage` event. Réécrire les 3 sites. Alternative : convertir ces états locaux en stores Zustand `persist` si la valeur doit traverser plusieurs composants (cas SquadLayout + MatchHistoryPage où la session label est un filtre métier).

---

### [DETTE] `winRate` / `killsPerGame` / `deathsPerGame` / `avgKD` répliqués (cf. axe 3, encore non corrigé)

L'axe 3 a déjà signalé que `service/session_compare_service.go:347-345` redéfinit `killsPerGame`, `deathsPerGame`, `avgKD` localement, et que `service/career_service.go`, `service/match_history_service.go`, `service/squad_service_v2.go` ont chacun leur version. Sans helper `internal/analysis/indicators.go::WinRate(matches)`, `KillsPerGame(matches)`, `AvgKD(matches)`. Ce constat est documenté en suivi axe 3 et ne sera pas re-traité ici, mais bloque le BLOQUANT axe 1 (cohérence des indicateurs canoniques).

Action : tracker dans le suivi axe 3 ; pas un constat nouveau axe 6.

---

### [AMÉLIORATION] `resolveSquadGrade` / `squadGrade` strictement identiques

`apps/go-api/internal/analysis/squad_score.go:110-125` `resolveSquadGrade(score float64) string` retourne S/A/B/C/D/F sur seuils 90/80/65/50/35. `apps/go-api/internal/service/squad_service_v2.go:638-652` `squadGrade(score float64) string` réimplémente exactement le même switch. Le commentaire dit « Aligne avec analysis.resolveSquadGrade ». Pourquoi ne pas appeler la fonction directement ?

Action : `squad_service_v2.go` doit consommer `analysis.ResolveSquadGrade(score)`. Supprimer `squadGrade`. Idem pour `scoreLabel(score)` (`squad_service_v2.go:518-530`) qui partage les seuils 80/65/50/35 → factoriser sous `analysis.ScoreLabel(score)`.

---

### [AMÉLIORATION] Fragments SQL `JOIN match_participants` répétés dans 8 fichiers

24 occurrences `JOIN [shared.]match_participants` dans `apps/go-api/internal/platform/duckdb/` réparties sur `match_history_repo.go`, `engagement_score_repo_queries.go`, `filters_repo.go`, `media_repo.go`, `queries.go`, `queries_career.go`, `queries_match.go` (12 occurrences à elle seule), `queries_squad.go`. Pas de helper `joinParticipants(alias)` ni de constantes SQL réutilisables.

CLAUDE.md mentionne explicitement le module Python `src/data/query/_sql_fragments.py` (`WIN_RATE_EXPR`, `IS_WIN`, `IS_LOSS`) comme pattern à porter. Aucun équivalent Go.

Action : créer `apps/go-api/internal/platform/duckdb/sql_fragments.go` avec les helpers : `JoinSharedParticipants(alias, matchAlias)` retournant `LEFT JOIN shared.match_participants AS <alias> ON <alias>.match_id = <matchAlias>.match_id` ; constantes `SQLOutcomeWin = "outcome = 2"`, `SQLOutcomeLoss = "outcome = 3"`, `SQLWinRateExpr = "AVG(CASE WHEN outcome = 2 THEN 1.0 ELSE 0.0 END)"`. Adopter progressivement.

---

### [AMÉLIORATION] Magic numbers `LIMIT N` dispersés

50 (`media_repo.go:263`, `queries_match.go:22`, `queries_squad.go:31`), 100 (`queries_match.go:210`), 150 (`queries_home_citations.go:93`, commenté), 200 (`engagement.go:351`), 24 (`media_service.go:22`, `defaultMediaPageSize`), 25 (`engine.go:34`, `historyPageSize`), 200 (`match_history.go:105`, `maxPageSize`). Certains ont des constantes nommées (`defaultMediaPageSize`, `maxPageSize`, `historyPageSize`) ; la plupart non.

Pas de centre de gravité : chaque fichier décide. Pas critique mais friction quand on doit tuner les SLAs API.

Action : pour les LIMIT non-cosmétiques, factoriser dans des constantes documentées au plus près du repository, ou dans `internal/config/limits.go` si la valeur a un sens global. Au minimum nommer chaque LIMIT.

---

### [AMÉLIORATION] 14 fichiers de tests Go construisent `domain.StatsMatchRow{}` à la main (96 occurrences)

`grep` montre 96 littéraux `domain.StatsMatchRow{...}` dans 14 fichiers de tests. Aucune factory `func makeMatchRow(opts...)` partagée — `analysis/performance_score_test.go:183` `makeHistoryRows(n)` est la seule, locale au fichier.

Conséquence : chaque ajout de champ obligatoire à `StatsMatchRow` (ex. : récent ajout `MMR`, `EngagementScore`) casse 96 fixtures sans helper de migration. Le commit B2 du 2026-04-29 a justement souffert de ce pattern (4 bugs latents).

Action : extraire `internal/domain/testfixtures/match_rows.go` (fichier `_test.go` partagé via build tag ou fichier `testing.go` exporté) avec `NewMatchRow(modifiers ...func(*StatsMatchRow)) StatsMatchRow` (functional options). Migration progressive.

## Constats hors-axe

- **Axe 1 lié** : `formatPercent` côté front existe déjà 4× avec arrondi/locale différents. Le fix axe 1 sur `accuracy 0..1 vs 0..100` doit s'accompagner d'un helper `formatPercent(ratio, decimals)` partagé pour ne pas re-créer de divergence. Documenté dans le suivi DRY ci-dessus, mais le BLOQUANT vit dans axe 1.
- **Axe 5 lié** : `match_view_service.go:1171-1182 perfColor` retourne du hex `#22c55e` / `#3b82f6` / `#f59e0b` / `#ef4444` côté Go pour des séries Plotly/ECharts. Couvert par axe 5 (couleurs hex servies par l'API).
- **Axe 4 lié** : les `formatDate` etc. front utilisent partout `Intl.DateTimeFormat` avec la locale codée en dur ou passée en string brut, jamais via le store de locale `useAppShellStore`. Cohérent avec le constat axe 4 sur la locale fragmentée.

## Suivi recommandé

1. **Action immédiate (BLOQUANT)** : créer `apps/go-api/internal/analysis/perf_tier.go` avec seuils canoniques 80/65/50/35, supprimer `perfColorToken` et les deux `perfColor` Go, faire converger les 6 implémentations. Sortir aussi tous les hex Go (`#22c55e` etc.) pour ne servir que des tokens sémantiques.
2. **Lot DRY backend Go** : `domain.OutcomeDNF`, `domain.IsBot()` + constantes SQL `SQLIsWin`/`SQLNotBot`/`SQLWinRateExpr`, supprimer `homeOutcome*` et les `Outcome*` redéclarés dans `analysis/`. Adopter `analysis.ResolveSquadGrade` partout. ~150 LOC de churn pour fermer 5 dettes.
3. **Lot DRY frontend** : `lib/format/{numbers,dates,durations}.ts` + `lib/hooks/useLocalStorageState.ts` + promotion de `KPIStrip.KPICard` en composant unique. Suppression des `normalizeModeLabel` TS et des `outcomeKeyOf` locaux après ajout `outcome_key` côté API.
