# Plan méta — Fondations transverses pour les portages Go

> Plan parent qui consolide les abstractions partagées entre les 6 plans de portage en cours
> et les 3 pages live qui doivent être mises à jour pour les adopter.
>
> Branche cible : `feat/multi-title-adapters-and-mappings` (puis sous-branches par phase).
> Date : 2026-04-27.
> Plans enfants impactés : `PLAN_MATCH_VIEW_GO_PORTAGE.md`, `PLAN_TIMESERIES_GO_PORTAGE.md`,
> `PLAN_CITATIONS_GO_PORTAGE.md`, `PLAN_CAREER_GO_PORTAGE.md`, `PLAN_SYNTHESIS_GO_PORTAGE.md`,
> `docs/AUDIT_TEAMMATES_V7_COCKPIT.md`. Pages live concernées : Home, Media, Explorer
> + WIP (palmares, session).

---

## État d'avancement (2026-04-27)

> **Phase 0 complète.** Branche `feat/foundations-axes-1-3-4`, 13 commits,
> ~9300 lignes ajoutées, ~165 tests cumulés, suite intégration verte,
> 0 régression. Branche prête à push/merger ou à laisser en attente.
>
> **Prochaine étape** : Phase 1 — pilotes Squad + MatchView. Commencer par
> rédiger `.ai/PLAN_SQUAD_GO_PORTAGE.md` depuis l'audit Teammates, puis
> amender complètement `PLAN_MATCH_VIEW_GO_PORTAGE.md` (déjà annoté niveau 1).
>
> Voir `.ai/thought_log.md` pour le récap consolidé de fin de session
> (entrée `[2026-04-27] Fin de session — Phase 0 close, prête pour Phase 1`).

| Phase | État | Effort réel / estimé |
|---|---|---:|
| Phase 0 — Fondations Go + frontend | ✅ Complète | ~1 session vs ~6 j-h estimés |
| Phase 1 — Pilotes Squad + MatchView | À démarrer | ~12 j-h estimés |
| Phase 2 — Roll-out (Career, Synthesis, Citations, Timeseries, pages live, WIP) | Bloqué par Phase 1 | ~12 j-h |
| Phase 3 — Cleanup Plotly / Recharts | Bloqué par Phase 2 | ~3 j-h |
| Phase 4 — Documentation et skills | Démarrable en parallèle Phase 3 | ~4 j-h |

---

## Table des matières

Le plan est structuré en **3 blocs de lecture** :

**Bloc A — Intro et critères**
- § 0 — Synthèse exécutive
- § 1 — Périmètre
- § 2 — Critères de succès

**Bloc B — Références transverses (s'appliquent à toutes les phases)**
- § 3 — Stratégie de tests
- § 4 — Stratégie de logging
- § 5 — Architecture cible (axes 1 helpers, 2 charts, 3 canonical, 4 i18n)

**Bloc C — Plan d'exécution**
- § 6 — Phasing — **Phase 0** Fondations · **Phase 1** Pilotes Squad+MatchView · **Phase 2** Roll-out · **Phase 3** Cleanup Plotly · **Phase 4** Documentation et skills

**Bloc D — Références d'exécution (consultées pendant les phases)**
- § 7 — Impact détaillé sur les plans enfants (+ stratégie d'amendement + stratégie de release)
- § 8 — Risques et mitigations
- § 9 — Estimation totale et planning
- § 10 — Done definition globale
- § 11 — Annexes (helpers à factoriser, tokens couleur, fichiers nouveaux, ADR, coverage cibles)

> **Important** : il n'y a **que 5 phases** (Phase 0 à Phase 4 dans la § 6). Les
> sections § 7 à § 11 ne sont **pas** des phases supplémentaires — ce sont des
> références transverses consultées pendant l'exécution.

---

## 0. Synthèse exécutive

Les 6 plans de portage convergent vers un même socle (canonical row, capability-aware,
séries brutes JSON, tokens couleur) mais le réinventent partiellement chacun de leur côté :
4 plans déclarent un `ComputeMapBreakdown`, 4 redéfinissent le filtre `all/2y/1y/1m/1w`,
4 reposent la question « payload Plotly server-side vs client », 300+ clés i18n émergent
sans manifest commun, et 16 stubs `Charts.X interface{}=nil` polluent encore le domain.

**Décision** : livrer 4 axes de fondations communs avant achèvement des plans enfants,
avec **Squad + MatchView comme pages pilotes** (les 2 plus complexes — leur couverture
valide l'API des fondations sur les cas extrêmes).

**Stack technique tranchée** :
- Charts : **migration ECharts** (via `echarts-for-react`), Plotly et Recharts retirés en Phase 3.
- Données : type partagé **`canonical.PlayerMatchRow`** consommé par tous les services.
- i18n : **manifests TOML par domaine** + linter ESLint anti-strings hardcodées.
- Helpers : **3 sous-packages** `analysis/breakdown`, `analysis/temporal`, `analysis/narrative`.

**Effort total estimé** : ~44 j-h sur 5 phases (fondations, pilotes, roll-out,
cleanup, documentation et skills). Inclut les ajouts opérationnels :
cache `LoadPlayerMatches`, whitelist regex, observabilité minimale,
ICU MessageFormat, `<CapabilityGap>` 3 modes, politique d'évolution
canonical, maintenance golden snapshots, **loader unifié `HighlightEvents`
qui débloque first_events_rolling, intensity heatmap et cadence sur
Squad/MatchView/Timeseries**.

**Stratégie de release** : pas de cohabitation v1/v2. Synchronisation backend ↔
frontend par PR. Voir § 7.3.

---

## 1. Périmètre

### 1.1 Pages cible

| Page | État actuel | Action attendue |
|---|---|---|
| Squad / Teammates | Audit fait, plan détaillé en attente | **Pilote Phase 1** — adopte les 4 fondations dès l'écriture |
| Match View | Plan rédigé (1024 L), implémentation en cours | **Pilote Phase 1** — refacto en parallèle de la rédaction des fondations |
| Career | Plan rédigé | Phase 2 — adapter le plan pour pointer vers les fondations |
| Synthesis | Plan en cours de refonte | Phase 2 — le nouveau plan adopte les fondations directement (pas de Phase 0 bloquante) |
| Citations | En cours (fichiers non commit : `analysis/citations_engine.go`, `sync/citations.go`) | Phase 2 — adapter le plan pour pointer vers les fondations |
| Timeseries | Plan rédigé (798 L) | Phase 2 |
| Home | Live | Phase 2 — adoption manifest i18n + `<KPIStrip>` partagé + `<CapabilityGap>` |
| Media | Live | Phase 2 — adoption manifest i18n |
| Explorer | Live | Phase 2 — adoption manifest i18n + filtre période partagé |
| Palmares | WIP | Phase 2 — branchement sur fondations |
| Session | WIP | Phase 2 — branchement sur fondations |

### 1.2 Hors périmètre

- Le sync des données (`internal/sync/`) sauf nécessité documentée.
- Les pages backoffice (admin, debug).
- La migration ECharts ne touche pas le serveur Go (les charts restent rendus côté client).

---

## 2. Critères de succès

- [ ] Aucun helper `ComputeMapBreakdown`, `FilterByPeriod`, `BucketTime` dupliqué : 1 implémentation, N consommateurs.
- [ ] Aucun stub `interface{}=nil` dans `internal/domain/`.
- [ ] Aucun import `plotly.js` ni `react-plotly.js` ni `recharts` dans `apps/web/src/`.
- [ ] Toute string UI utilisateur passe par un manifest i18n typé (linter en CI).
- [ ] `canonical.PlayerMatchRow` est consommé par MatchView V2, Squad V2, Synthesis V2, Career V2, Timeseries V2.
- [ ] Squad et MatchView fonctionnent end-to-end avec les 4 fondations (pages pilotes).
- [ ] `go test ./...` et `npm run typecheck && npm run test` passent à chaque phase.
- [ ] `thought_log.md` à jour avec une entrée par phase.
- [ ] Couverture tests Go ≥ 90 % sur `analysis/`, ≥ 80 % sur `service/`, ≥ 70 % sur `platform/duckdb/`.
- [ ] Couverture frontend ≥ 80 % sur `components/charts/` et sur `features/{squad,match-view}/`.
- [ ] Documentation humaine + agents IA livrée et skills `.claude/skills/` mis à jour.

---

## 3. Stratégie de tests

L'objectif est qu'aucune régression silencieuse ne puisse apparaître entre une phase
et la suivante. Chaque phase doit livrer la batterie de tests correspondante avant
sa done definition.

### 3.1 Niveaux de tests

#### 3.1.1 Unitaires purs — `internal/analysis/`

Tests stateless, table-driven, 0 dépendance externe.

- `temporal_test.go` : 5 périodes × scénarios de bordure (rows vide, row à la borne,
  row au futur, granularité adaptive correcte).
- `breakdown_test.go` : matrices `outcome IN {WIN, LOSS, TIE, DNF}` × 3 maps × 3 modes,
  vérification des taux et des moyennes pondérées. Cas limite : 0 match, 1 match,
  100 % WIN, 100 % LOSS, ties exclusivement.
- `narrative_test.go` :
  - Dominance : 5 flags × labels + tokens.
  - Encounter : ordinaux 1, 2, 5, 100 ; ally_plus seuil 70 % ; tough sur K/D 1.5 borne.
  - Participation : 3 familles de mode (slayer, ctf, strongholds) × awards canoniques
    avec valeurs aux bornes des thresholds.
  - Impact roles : 8 rôles déclenchés isolément sur match synthétique (séquence
    d'events filmés contrôlée) ; cas combinés (silent_hero + top_killer simultanés).
- Coverage cible : **≥ 90 %**, mesurée via `go test -cover ./internal/analysis/...`.

#### 3.1.2 Repository — `internal/platform/duckdb/`

Tests sur DuckDB `:memory:` avec fixtures SQL injectées au setup.

- `player_matches_repo_test.go` : pour chaque champ de `PlayerMatchFilters`
  (Period, OutcomeIn, HadBotTeammate, IsFirefight, MinTimePlayedSeconds,
  ExcludeFriendsXUIDs, BTBExcluded, PlaylistRegex, MapIDs, Limit, OrderBy) un test
  isolé qui vérifie l'inclusion / exclusion attendue.
- Test de combinaison de filtres (AND multiples).
- Test d'exclusion bots (`xuid LIKE 'bid(%'`).
- Test du préfixe `shared.` (échec attendu si table locale au lieu de shared).
- Test capability gating : titre sans `match.history` retourne `ErrCapabilityNotSupported`.
- Test perf benchmark : `BenchmarkLoadPlayerMatches` sur 10 000 rows fixtures
  (gate informatif, pas bloquant).

#### 3.1.3 Service — mock `port.Repository`

Tests avec `port.PlayerMatchesRepository` mockée (manuelle ou via lib légère).

- `squad_service_test.go` : intersection N coéquipiers, dégradation si un coéquipier
  retourne `ErrCapabilityNotSupported`, ordering stable.
- `match_view_service_test.go` : composition errgroup, propagation des erreurs,
  fallback si une sous-requête échoue.
- `career_service_test.go`, `synthesis_service_test.go`, `citations_service_test.go`,
  `timeseries_service_test.go` : chaque branche logique du service couverte avec
  fixtures `[]canonical.PlayerMatchRow`.
- Coverage cible : **≥ 80 %**.

#### 3.1.4 Handler HTTP — `httptest`

- `match_view_handler_test.go`, `squad_handler_test.go`, etc. : pour chaque endpoint,
  vérifier statuts 200 (OK), 400 (paramètre invalide), 404 (joueur inexistant),
  503 (capability absente, dégradation gracieuse).
- Vérifier le shape JSON (présence des champs, types) via `assert.JSONEq` sur fixture.
- Vérifier les headers : `Content-Type: application/json`, `X-LevelUp-Title` echoed.

#### 3.1.5 Frontend — Vitest

- Pour chaque wrapper chart : test de rendu loading / error / empty / data.
- Test de l'option ECharts générée (snapshot de l'`option` object — détecte les
  régressions sans screenshots).
- Tests des hooks `useT`, `useFieldLabel`, `useOutcomeLabel`.
- Tests de `<CapabilityGap>` : affichage du label localisé selon la `reason`.
- Tests de `<NarrativeBadge>` : token couleur résolu correctement, label localisé.
- Tests de la règle ESLint `@levelup/no-hardcoded-strings` (positifs et négatifs).
- Coverage cible : **≥ 80 %** sur `components/charts/` et `features/{squad,match-view}/`.

#### 3.1.6 E2E — Playwright

Scénarios critiques couvrant les flux les plus complexes.

- `e2e/squad.spec.ts` : sélection de 3 coéquipiers, vérification de l'apparition
  des 4 sections clés (heatmap impact, radar 6 axes, lollipop W/L par carte,
  galerie médailles), changement de période, capability gap visible si applicable.
- `e2e/match-view.spec.ts` : ouverture d'un match avec dominance flag, vérification
  du badge dominance, scoreboard détaillé, kill feed, encounters cliquables.
- `e2e/career.spec.ts`, `e2e/timeseries.spec.ts`, `e2e/synthesis.spec.ts`,
  `e2e/citations.spec.ts` : flux principaux, toggle de filtres, switch de période.
- `e2e/i18n.spec.ts` : switch FR↔EN sur 3 pages, vérification qu'aucune clé brute
  ne fuit (pas de `tm_xxx` visible).
- `e2e/capability_gap.spec.ts` : page chargée pour titre synthétique sans capability
  X → dégradation visible et fonctionnelle.

#### 3.1.7 Golden parity — Go-only

Test de non-régression macroscopique. Pas de Python (projet Go-only).

- `cmd/foundations_golden_parity/main.go` : prend une DB joueur seed gelée
  (`testdata/golden/halo_infinite/{shared,player}.duckdb`), exécute les services
  cibles (`GetCareerPage`, `GetMatchView`, `GetSquadPage`, etc.) et produit un
  JSON canonique (clés triées, floats arrondis à 1e-3).
- Comparaison avec `testdata/golden/snapshots/{page}.json` figés.
- Diff humainement lisible si échec ; commande `--update` pour régénérer après
  changement intentionnel.
- Test exécuté en CI sur chaque PR touchant les services.

**Maintenance des snapshots** :

- Les snapshots vivent en **`testdata/golden/snapshots/`** versionnés Git (pas
  ignorés). Pas de génération à la volée en prod.
- Régénération via `go run ./cmd/foundations_golden_parity --update` — la commande
  produit un diff git et écrit dans le commit en cours.
- **Revue manuelle obligatoire en PR** : tout changement de snapshot doit être
  justifié dans le commit message. Un label GitHub `golden-snapshot-changed`
  appose automatiquement.
- Si la CI détecte un snapshot divergent **sans** flag `--update` explicite, elle
  échoue avec un message clair pointant vers `testdata/golden/README.md`.
- `testdata/golden/README.md` (livré en Phase 0) explique la procédure de
  régénération + les cas légitimes (changement de calcul, ajout de champ,
  refacto d'arrondi).
- DB de référence : `testdata/golden/halo_infinite/{shared,player}.duckdb` figés ;
  le script de régénération **n'altère jamais** la DB, seulement le snapshot.

#### 3.1.8 Régression visuelle ECharts

- Snapshot Vitest de l'`option` object généré par chaque wrapper avec un dataset
  fixe.
- Test de la sortie SVG (ECharts `getDataURL({type: 'svg'})`) sur fixtures
  reproductibles → diff de strings.
- Pas de Playwright screenshot (lourd, fragile).

#### 3.1.9 Performance — benchmarks Go

- `BenchmarkBreakdownByMap` : 10 000 rows.
- `BenchmarkFilterByPeriod` : 100 000 rows.
- `BenchmarkComputeParticipationProfile` : awards × 5 familles de mode.
- `BenchmarkIdentifyImpactRoles` : 50 events filmés × 4 joueurs.
- Sortie en CI via `go test -bench=. -benchmem` ; pas de gate dur, mais
  régression > 50 % surveillée.

#### 3.1.10 Capability gating — dégradation gracieuse

- Pour chaque service consommant un adapter : 1 test avec adapter qui retourne
  `games.ErrCapabilityNotSupported` → vérifier que le service ne panique pas et
  retourne un DTO partiel cohérent + champ `capability_gap` documenté.
- Test handler : statut 200 avec section flagguée plutôt que 5xx.

#### 3.1.11 i18n manifest — clés référencées vs définies

- Test Vitest `i18n_manifest.test.ts` :
  - Charge tous les manifests TOML compilés.
  - Parse tous les fichiers `apps/web/src/**/*.{ts,tsx}` à la recherche de `t('...')`.
  - Vérifie que **toutes les clés référencées sont définies** dans un manifest
    (FR ET EN).
  - Vérifie qu'**aucune clé définie n'est orpheline** (zéro référence).
  - Échec en CI si l'un des deux échoue.

#### 3.1.12 TOML mappings — complétude par titre

- Test Go `mappings_test.go` :
  - Pour chaque titre enregistré, charge `fields.toml`, `assets.toml`, `outcomes.toml`.
  - Vérifie que toutes les `FieldKey` canoniques utilisées par les services consommant
    ce titre sont mappées (sinon : capability absente → service doit dégrader).
  - Vérifie cohérence des clés FR ET EN.

#### 3.1.13 Linter custom — `@levelup/no-hardcoded-strings`

- Tests unitaires de la règle dans `apps/web/eslint-rules/no-hardcoded-strings.test.ts` :
  positif (chaîne hardcodée → erreur) et négatif (chaîne dans `t(...)` → OK,
  chaîne dans whitelist → OK).

### 3.2 Fixtures partagées

Pour éviter la duplication de fixtures entre tests :

```
internal/testutil/fixtures/
  match_rows.go          // PlayerMatchRow synthétiques (10 scénarios canoniques)
  events.go              // HighlightEvent fixtures (8 rôles isolés + combinés)
  awards.go              // PersonalScoreAwardRow fixtures (3 familles de mode)
  duckdb_seed.go         // Seed DB :memory: (shared + player) avec scénarios canoniques

apps/web/src/test/fixtures/
  chart_series.ts        // ChartSeries<T> fixtures pour wrappers
  player_match_rows.ts   // mirror des fixtures Go (export JSON figé)
```

### 3.3 CI gates

Avant merge sur `main`, la CI doit valider :

- `go test ./... -race -coverprofile=coverage.out` → couverture ≥ seuil par package.
- `go vet ./...` sans warning.
- `npm run typecheck && npm run lint && npm run test`.
- E2E Playwright sur 3 navigateurs (Chromium, Firefox, WebKit).
- Golden parity test passe.
- Bundle analyzer : taille du bundle JS < seuil (varie selon phase, voir Phase 3).

### 3.4 Garde-fous "tester ce qu'on déclare livrer"

À chaque phase, la done definition mentionne **explicitement** les tests à passer.
Pas de tâche cochable sans son test associé. La revue de phase vérifie cette règle.

---

## 4. Stratégie de logging

### 4.1 Conventions par couche

| Couche | Niveau par défaut | Cas d'usage |
|---|---|---|
| `analysis/` | aucun (fonctions pures) | Pas de log dans le code pur ; les erreurs sont propagées via `error` retourné. |
| `service/` | `slog.DebugContext` au début + fin | Trace de l'orchestration (sous-requêtes parallèles), durée totale. |
| `platform/duckdb/` | `slog.DebugContext` sur query name + `slog.ErrorContext` sur erreur | Trace de la requête + paramètres anonymisés. |
| `port/` | aucun | Interfaces uniquement. |
| `api/handlers/` | `slog.InfoContext` sur entrée + `slog.ErrorContext` sur erreur 5xx | Trace de la requête, statut, durée. |
| `api/middleware/` | déjà câblé (logging HTTP) | Conserver. |

### 4.2 Clés structurées standards

Les clés suivantes sont **réservées** et utilisées partout pour faciliter le grep :

| Clé | Type | Exemple |
|---|---|---|
| `err` | `error` | `slog.ErrorContext(ctx, "failed to load matches", "err", err)` |
| `match_id` | `string` | `slog.DebugContext(ctx, "loading match", "match_id", id)` |
| `player` | `string` (gamertag) | `slog.InfoContext(ctx, "career page", "player", gt)` |
| `xuid` | `string` | |
| `title_slug` | `string` | `slog.DebugContext(ctx, "resolveCurrentSeason", "title_slug", slug)` |
| `duration_ms` | `int64` | `slog.InfoContext(ctx, "service done", "duration_ms", d.Milliseconds())` |
| `query_name` | `string` | `slog.DebugContext(ctx, "running query", "query_name", "Q34")` |
| `period` | `string` | `slog.DebugContext(ctx, "filter", "period", "1y")` |
| `capability` | `string` | `slog.WarnContext(ctx, "capability missing", "capability", "match.history")` |

**Interdit** : `fmt.Println`, `log.Printf`, `log.Println`. Lint-checked via grep en CI
(commande dans `delivery-checklist`).

### 4.3 Niveaux

- `Debug` : détail du flux interne (sous-requêtes, branches prises).
- `Info` : événement métier significatif (page chargée, calcul terminé).
- `Warn` : situation non-bloquante mais notable (capability absente → dégradation).
- `Error` : erreur retournée à l'appelant ou erreur infra.

### 4.4 PII et sécurité

- Pas de log de données utilisateur sensibles (token, IP non hashée, payload complet).
- Les paramètres de requête SQL ne sont **pas** loggés sauf en mode `Debug` explicite
  (`slog.DebugContext(ctx, "query params", "filters", anonymize(filters))`).
- Tests : aucun champ contenant `password`, `token`, `secret` dans les logs.

### 4.5 Tests d'observabilité

- `internal/testutil/sloghandler.go` : capture les logs en mémoire pour assertion.
- Pour chaque service, au moins un test qui vérifie qu'**une erreur est loggée
  avec la clé `err`** quand le repo échoue.
- Pour chaque handler, au moins un test qui vérifie qu'**un log Info est émis**
  à l'entrée avec les clés attendues (`player`, `title_slug`).

### 4.6 Configuration

- Production : `slog.NewJSONHandler` avec niveau `Info`.
- Développement : `slog.NewTextHandler` avec niveau `Debug`.
- Configuration via env var `LEVELUP_LOG_LEVEL` (déjà présente, à vérifier).

### 4.7 Observabilité au-delà des logs

**Décision** : pas de Prometheus / OpenTelemetry full pour cette itération. Trop de
coût d'infra pour un projet en cours de fondation. Mais on installe le **squelette
minimal** pour que ces 4 métriques de base soient interrogeables :

| Métrique | Source | Endpoint |
|---|---|---|
| `service_duration_ms{service="X",status="ok|error"}` | middleware service | `/debug/vars` (expvar) |
| `repo_query_duration_ms{query="X"}` | wrapper repo | `/debug/vars` |
| `cache_hit_ratio{cache="player_matches"}` | cache `LoadPlayerMatches` | `/debug/vars` |
| `error_count{service="X"}` | middleware service | `/debug/vars` |

Implémentation :

- Helper `internal/observability/expvar_metrics.go` (~80L) qui expose les compteurs
  et durées via `expvar.Publish`.
- Middleware `instrumentService(svc)` qui mesure entrée/sortie de chaque méthode
  service principale.
- Endpoint `/debug/vars` exposé uniquement en dev et derrière auth admin en prod.
- Dashboard maison ou simple JSON consommable par `curl localhost:8080/debug/vars`.

**Tracing OpenTelemetry** : non couvert par cette itération. Ticket ouvert pour
plus tard (`docs/adr/future-otel.md`).

**Tests** : `internal/observability/expvar_metrics_test.go` valide les compteurs
(incrément, reset, format JSON).

---

## 5. Architecture cible

### 3.1 Axe 1 — Helpers transverses `internal/analysis/`

Trois sous-packages purs, 0 accès DB, 0 HTTP, testables en isolation.

#### 3.1.1 `analysis/temporal/`

Filtre période + bucketing temporel. Réutilisé par : Synthesis, Timeseries, Career
(Encounters), MatchView (Header dominance période), Squad (cadence, intensité).

```go
package temporal

type Period string

const (
    PeriodAll Period = "all"
    Period2Y  Period = "2y"
    Period1Y  Period = "1y"
    Period1M  Period = "1m"
    Period1W  Period = "1w"
)

// Since retourne le timestamp de référence, ou nil si Period == PeriodAll.
func (p Period) Since(now time.Time) *time.Time

// FilterByPeriod garde uniquement les lignes dont StartTime >= since.
func FilterByPeriod[T HasStartTime](rows []T, period Period, now time.Time) []T

type Granularity string

const (
    GranDay     Granularity = "1d"
    GranWeek    Granularity = "1w"
    GranMonth   Granularity = "1m"
    GranAdaptive Granularity = "adaptive" // dérive granularité depuis la période
)

type Bucket[T any] struct {
    Start, End time.Time
    Items      []T
    Label      string // ex. "2026-W17", "2026-04-27"
}

func BucketByGranularity[T HasStartTime](rows []T, gran Granularity, period Period) []Bucket[T]

// ResolveAdaptive retourne 1d pour 1w/1m, 1w pour 1y, 1m pour 2y, 1m pour all.
func ResolveAdaptive(period Period) Granularity
```

**Tests** : `analysis/temporal/temporal_test.go` — table-driven, golden buckets fixtures.

#### 3.1.2 `analysis/breakdown/`

Agrégations W/L/T/DNF par dimension. Réutilisé par : Synthesis (#3.1, #3.2 outcomes
by map/mode), Timeseries (Map breakdown), Career (Map history), Squad (lollipop W/L
par carte, perf vs historique), MatchView (mode_category dominance).

```go
package breakdown

type Counts struct {
    Played, Wins, Losses, Ties, DNF int
    WinRate, LossRate, TieRate, DNFRate float64
}

type MapRow struct {
    MapID    string
    MapLabel string // résolu via TitleSemanticAdapter avant retour
    Counts
    AvgPerformanceScore *float64
}

func ByMap(rows []canonical.PlayerMatchRow) []MapRow
func ByMode(rows []canonical.PlayerMatchRow) []ModeRow
func ByPlaylist(rows []canonical.PlayerMatchRow) []PlaylistRow
func ByModeCategory(rows []canonical.PlayerMatchRow) []ModeCategoryRow

// CompareToHistorical retourne le delta par dimension (session vs historique).
func CompareToHistorical(session, historical []MapRow) []MapDelta
```

**Tests** : `analysis/breakdown/breakdown_test.go` — fixtures basées sur
`canonical.PlayerMatchRow` avec outcomes connus.

#### 3.1.3 `analysis/narrative/`

Calculs de badges narratifs. Réutilisé par : MatchView (dominance flag, encounter
badges, citations tier), Squad (8 rôles d'impact, dominant pair name), Career
(top matches badges, encounter badges Némésis/Souffre-douleurs), Citations (tier
labels), Synthesis (top by period badges).

```go
package narrative

// DominanceFlag déjà défini dans canonical/match.go (1..5).
// Ce package fournit la résolution badge → label/token.

type DominanceBadge struct {
    Flag       canonical.DominanceFlag
    LabelKey   string // clé i18n manifest, ex. "narrative.dominance.domination"
    ColorToken string // token sémantique, ex. "narrative.dominance.win.strong"
}

func ResolveDominanceBadge(flag canonical.DominanceFlag) *DominanceBadge

type EncounterKind string

const (
    EncounterAllyPlus   EncounterKind = "ally_plus"   // > 70 % winrate as ally
    EncounterToughAlly  EncounterKind = "tough_ally"  // ally avec K/D > 1.5 contre soi
    EncounterToughEnemy EncounterKind = "tough_enemy" // ennemi avec K/D > 1.5 contre soi
    EncounterOrdinal    EncounterKind = "ordinal"     // n-ième fois rencontré
)

type EncounterBadge struct {
    Kind       EncounterKind
    LabelKey   string
    ColorToken string
    Detail     map[string]any // ex. {"ordinal": 5}
}

// Stats agrégées requises pour calculer les badges.
type EncounterStats struct {
    XUID            string
    Gamertag        string
    TotalEncounters int
    AllyCount, EnemyCount int
    WinrateAsAlly   *float64
    WinrateVsEnemy  *float64
    KillsDealt      int
    DeathsSuffered  int
    LastSeen        *time.Time
}

func ComputeEncounterBadges(stats EncounterStats, ordinal int) []EncounterBadge

// Radar 6 axes : Combat / Survie / Soutien / Score / Objectif / Impact.
type ParticipationAxis string

const (
    AxisCombat    ParticipationAxis = "combat"
    AxisSurvival  ParticipationAxis = "survival"
    AxisSupport   ParticipationAxis = "support"
    AxisScore     ParticipationAxis = "score"
    AxisObjective ParticipationAxis = "objective"
    AxisImpact    ParticipationAxis = "impact"
)

type ParticipationScore struct {
    Axis  ParticipationAxis
    Value float64 // 0..100 normalisé selon famille de mode
    Raw   float64 // valeur brute pour debug/tooltip
}

// ModeFamily : "slayer", "ctf", "strongholds", "oddball", "custom".
func ComputeParticipationProfile(
    awards []PersonalScoreAwardRow,
    modeFamily string,
) []ParticipationScore

// 8 rôles Squad (silent_hero, false_brother, top_killer, last_casualty,
// last_group_kill, first_group_death, first_blood, clutch_finisher).
type ImpactRole string

const (
    RoleFirstBlood       ImpactRole = "first_blood"
    RoleClutchFinisher   ImpactRole = "clutch_finisher"
    RoleLastCasualty     ImpactRole = "last_casualty"
    RoleLastGroupKill    ImpactRole = "last_group_kill"
    RoleFirstGroupDeath  ImpactRole = "first_group_death"
    RoleSilentHero       ImpactRole = "silent_hero"
    RoleFalseBrother     ImpactRole = "false_brother"
    RoleTopKiller        ImpactRole = "top_killer"
)

type RoleAssignment struct {
    XUID      string
    Role      ImpactRole
    MatchID   string
    ColorToken string
    Inverted  bool // true pour les rôles "négatifs" (couleur inversée)
}

// IdentifyImpactRoles prend les events filmés + outcomes de l'équipe et retourne
// les rôles attribués pour chaque match × joueur.
func IdentifyImpactRoles(
    events []canonical.HighlightEvent,
    teamOutcomes map[string]canonical.Outcome,
    squad []string, // xuids des coéquipiers + soi
) []RoleAssignment
```

**Notes Squad pilote** : `IdentifyImpactRoles` doit étendre l'algo
`internal/analysis/match_impact.go` actuel (4 rôles, bilatéral 1v1) à 8 rôles N-joueurs.
Le clutch est calculé sur fenêtre temporelle réelle (30 dernières secondes), pas via
approximation par tiers — voir `docs/AUDIT_TEAMMATES_V7_COCKPIT.md` § 11.5 (bloqueur 2).

**Tests** : `analysis/narrative/*_test.go` — fixtures par scénario : domination 50-15,
remontada (perte avancée puis victoire), 8 rôles isolés sur match synthétique, radar
6 axes avec familles de mode différentes.

### 3.2 Axe 2 — Stack chart unifié (ECharts)

#### 3.2.1 Décision payload côté API

**Toutes les données chart transitent en séries brutes JSON** (suppression des stubs
`PlotlyFigurePayload` et autres `interface{}` nil). Type partagé dans `internal/domain/charts.go` :

```go
package domain

type ChartSeries[T any] struct {
    Key        string         // identifiant stable, ex. "self.kills_per_match"
    LabelKey   string         // clé i18n manifest
    ColorToken *string        // token sémantique, jamais hex
    Datapoints []T
    Meta       map[string]any // axis hints, threshold lines, hover format
}

type ChartPoint2D struct {
    X     any      // time.Time, float64 ou string
    Y     float64
    Label *string  // override hover, optionnel
}

type ChartPointHeatmap struct {
    X, Y   string             // axis labels (résolus côté Go via TitleSemanticAdapter)
    Value  float64
    Detail map[string]any     // contenu du tooltip (KDA, outcome, etc.)
}

type ChartPointStacked struct {
    Category   string             // axis principal
    Components map[string]float64 // ex. {"win": 12, "loss": 5, "tie": 2, "dnf": 1}
}
```

**Conséquence** : tous les services qui exposaient un payload Plotly arrêtent. Les
plans Citations (`distribution_chart`) et Synthesis (cellules brutes pour Plotly client)
adoptent ce format. La distribution médailles Citations passe en `[]ChartSeries[ChartPoint2D]`
et est rendue côté client comme les autres.

#### 3.2.2 Wrappers ECharts côté web

Tous les wrappers vivent dans `apps/web/src/components/charts/` et héritent d'un
`<ChartCard>` racine qui gère :
- États loading / error / empty
- Injection du theme depuis `tokenCssVar()` (palette + outcome + narrative)
- Resize observer + cleanup ECharts instance
- Lazy-load `echarts` via dynamic import (réduit le bundle initial)

```
apps/web/src/components/charts/
  ChartCard.tsx              base : wrapper état + theme + lifecycle
  TimeseriesLine.tsx         timeline mono/multi-séries
  TimeseriesArea.tsx         stacked area
  BarStacked.tsx             W/L/T/DNF empilés (Synthesis #1, MatchView dominance)
  BarGrouped.tsx             3 barres par catégorie (Squad per-min, Real/Expected/Hist)
  Heatmap2D.tsx              joueur×carte, match×phase, perf×carte (Squad, MatchView)
  Radar.tsx                  6 axes participation (Squad, MatchView)
  Bullet.tsx                 winrate session vs historique (Squad, Career)
  Lollipop.tsx               W/L vertical par carte (Squad, Career)
  Histogram.tsx              distribution médailles, KDA (Citations, Timeseries)
  Scatter.tsx                Timeseries scatter
  Donut.tsx                  weapons top 8 (Squad, MatchView)
  Gauge.tsx                  rank progression, hero progression (Career)
  Cadence.tsx                kills/phase 60s (Squad, MatchView)
```

`<ChartCard>` API uniforme :

```tsx
type ChartCardProps<T> = {
  title?: string;            // i18n key résolu en amont
  series: ChartSeries<T>[];  // données du backend
  loading?: boolean;
  error?: Error | null;
  emptyMessage?: string;     // i18n key
  height?: number;
  // option ECharts override (utilisé par les wrappers spécialisés)
  buildOption: (series: ChartSeries<T>[]) => echarts.EChartsCoreOption;
};
```

#### 3.2.3 Migration progressive

- Phase 0 : ajouter `echarts` + `echarts-for-react` au bundle, écrire `<ChartCard>`.
- Phase 1 : Squad et MatchView consomment uniquement les wrappers ECharts (pas de
  cohabitation côté pilotes).
- Phase 2 : Career, Synthesis, Citations, Timeseries migrent un wrapper à la fois.
  Plotly et ECharts cohabitent dans le bundle pendant ~2-3 semaines.
- Phase 3 : suppression de `react-plotly.js`, `plotly.js-basic-dist`, `recharts` du
  `package.json`. Suppression des wrappers dans `apps/web/src/components/ui/timeseries-*.tsx`,
  `combat-yield-timeseries.tsx`, `plotly-chart.tsx`, `features/career/CareerChartsSection.tsx`.

**Garde-fous** :
- `eslint-plugin-import` règle `no-restricted-imports` interdisant `plotly.js*` et
  `recharts` après Phase 3.
- Bundle analyzer en CI pour vérifier que la taille du bundle baisse de ~2.5 MB après cleanup.

#### 3.2.4 Spécifications `buildOption` ECharts (référence)

Les wrappers `<BarStacked>` et `<Heatmap2D>` étant les plus consommés (Synthesis, MatchView, Squad), leur `buildOption` est spécifié ici à la valeur près. Les autres wrappers suivent le même contrat — la spec détaillée est documentée dans leur `*.tsx` ou dans le plan de la première page consommatrice.

**Conventions communes** :
- L'`option` ne contient **aucun hex** ; les couleurs proviennent de `resolveToken(token)` appelé depuis le composant React et passé en argument à `buildOption` via une fonction utilitaire `tokens: Record<SemanticToken, string>` (résolu une fois par render, pas dans le buildOption — sinon Vitest ne peut pas snapshot l'option).
- L'`option` ne contient **aucune string i18n hardcodée** ; les libellés sont fournis par le caller via `series[].label` ou un objet `labels: { x?: string, y?: string, ... }` passé en deuxième argument.
- Tous les charts ont `animation: false` par défaut (cf. CLAUDE.md règle UI : pas d'animation gratuite). Le caller peut activer si justifié.

**Type partagé `ChartSeries<T>`** (rappel) :
```ts
interface ChartSeries<T> {
  id: string                  // clé stable (ex: 'win', 'loss', 'tie', 'dnf')
  label: string               // libellé i18n résolu
  color?: SemanticToken       // token sémantique (résolu côté wrapper)
  data: T[]                   // forme dépend du chart
}
```

##### `<BarStacked>` — Stacked outcomes par catégorie

**Données attendues** : `series: ChartSeries<{category: string; value: number}>[]` — N séries (typiquement 4 pour W/L/D/DNF) qui partagent toutes le même axe catégoriel.

**Préparation amont (côté caller)** :
1. Calculer `categories = uniq(series.flatMap(s => s.data.map(d => d.category)))`.
2. Trier `categories` par total desc et tronquer à `maxCategories` (12 pour map, 10 pour mode côté Synthesis).
3. Aligner toutes les séries sur l'ordre `categories` (remplir `0` pour les trous — ECharts ne tolère pas les `data` désalignés en mode `stack`).

**`buildOption(series, { categories, tokens, labels })`** :
```ts
function buildBarStackedOption(
  series: ChartSeries<{ category: string; value: number }>[],
  ctx: {
    categories: string[]
    tokens: Record<SemanticToken, string>
    labels?: { yAxis?: string; tooltip?: string }
    rotateXAxisLabels?: number  // défaut -45
  },
): echarts.EChartsCoreOption {
  return {
    animation: false,
    grid: { left: 48, right: 16, top: 56, bottom: 80, containLabel: true },
    legend: {
      type: 'plain',
      orient: 'horizontal',
      top: 0,
      left: 'center',
      itemGap: 16,
    },
    tooltip: {
      trigger: 'axis',
      axisPointer: { type: 'shadow' },
      // formatter custom : empile les valeurs et affiche le total
      formatter: (params) => formatStackedTooltip(params),
    },
    xAxis: {
      type: 'category',
      data: ctx.categories,
      axisLabel: {
        rotate: ctx.rotateXAxisLabels ?? -45,
        interval: 0,         // forcer l'affichage de toutes les catégories
        hideOverlap: false,  // pas d'auto-hide — on veut tout voir
      },
      axisTick: { alignWithLabel: true },
    },
    yAxis: {
      type: 'value',
      name: ctx.labels?.yAxis,
      nameLocation: 'middle',
      nameGap: 36,
      minInterval: 1,        // pas de fractions sur un comptage de matchs
    },
    series: series.map((s) => ({
      type: 'bar',
      stack: 'outcomes',     // clé partagée → empilage
      name: s.label,
      data: s.data.map((d) => d.value),
      itemStyle: {
        color: s.color ? ctx.tokens[s.color] : undefined,
      },
      emphasis: { focus: 'series' },
      barMaxWidth: 40,
    })),
  }
}
```

**Tokens conventionnels Synthesis** (à passer en `series[].color`) : `outcome-win`, `outcome-loss`, `outcome-draw`, `outcome-dnf`. La nomenclature canonique côté Go est `Outcome.Win/Loss/Tie/DNF` ; côté UI le « Tie » utilise le token `outcome-draw` — alignement déjà documenté dans `apps/web/src/lib/accessibility/semantic-tokens.ts`.

**Tests Vitest attendus** : snapshot de l'option pour 4 séries × 5 catégories ; cas 1 série vide (toutes valeurs 0) ; cas categories=[] (chart empty).

##### `<Heatmap2D>` — Heatmap divergente avec masquage des cellules vides

**Données attendues** : `series[0].data: { x: number; y: number; value: number | null; meta?: Record<string, unknown> }[]`. La grille doit être **complète** (toutes les paires x×y présentes, valeur `null` si cellule vide). C'est le caller qui matérialise la grille — ECharts ne fait pas de remplissage automatique.

**Convention pour la heatmap WR Synthesis** :
- `x` = heure 0..23
- `y` = jour Lun..Dim (index 0..6, Lun=0)
- `value` = win rate (0..1) ou `null` si `total < minMatches`
- `meta.count` = nombre total de matchs dans la cellule (utilisé pour l'overlay texte et le tooltip)

**`buildOption(series, ctx)`** :
```ts
function buildHeatmap2DOption(
  series: ChartSeries<{ x: number; y: number; value: number | null; meta?: { count?: number } }>[],
  ctx: {
    xAxisLabels: string[]    // ex: ['00h', '01h', …, '23h']
    yAxisLabels: string[]    // ex: ['Lun', 'Mar', …, 'Dim']
    visualMap: {
      kind: 'divergent' | 'sequential'
      min: number
      max: number
      tokenLow: SemanticToken
      tokenMid?: SemanticToken    // requis si divergent
      tokenHigh: SemanticToken
      formatter?: (v: number) => string  // ex: (v) => `${(v*100).toFixed(0)}%`
    }
    tokens: Record<SemanticToken, string>
    showCellText?: boolean   // défaut true (overlay count)
  },
): echarts.EChartsCoreOption {
  const data = series[0].data.map((d) => [d.x, d.y, d.value, d.meta?.count ?? 0])
  const colorStops = ctx.visualMap.kind === 'divergent'
    ? [
        ctx.tokens[ctx.visualMap.tokenLow],
        ctx.tokens[ctx.visualMap.tokenMid!],
        ctx.tokens[ctx.visualMap.tokenHigh],
      ]
    : [ctx.tokens[ctx.visualMap.tokenLow], ctx.tokens[ctx.visualMap.tokenHigh]]

  return {
    animation: false,
    grid: { left: 56, right: 24, top: 32, bottom: 56, containLabel: true },
    tooltip: {
      trigger: 'item',
      formatter: (p: any) => {
        const [x, y, v, count] = p.value as [number, number, number | null, number]
        if (v === null || count === 0) return `${ctx.yAxisLabels[y]} ${ctx.xAxisLabels[x]} : —`
        const pct = ctx.visualMap.formatter ? ctx.visualMap.formatter(v) : v.toFixed(2)
        return `${ctx.yAxisLabels[y]} ${ctx.xAxisLabels[x]}<br/>WR : ${pct}<br/>Matchs : ${count}`
      },
    },
    xAxis: { type: 'category', data: ctx.xAxisLabels, splitArea: { show: false } },
    yAxis: {
      type: 'category',
      data: ctx.yAxisLabels,
      inverse: true,           // Lundi en haut
      splitArea: { show: false },
    },
    visualMap: {
      type: 'continuous',
      min: ctx.visualMap.min,
      max: ctx.visualMap.max,
      calculable: false,
      orient: 'vertical',
      right: 0,
      top: 'middle',
      formatter: ctx.visualMap.formatter,
      inRange: { color: colorStops },
      // CRITIQUE : les valeurs `null` sont rendues transparentes
      // sans masquer le tooltip ni le label texte de la cellule.
      outOfRange: { color: 'rgba(0,0,0,0)' },
    },
    series: [{
      type: 'heatmap',
      data,
      label: ctx.showCellText !== false
        ? {
            show: true,
            formatter: (p: any) => {
              const count = (p.value as any[])[3] as number
              return count > 0 ? String(count) : ''
            },
            fontSize: 10,
            color: 'inherit',
          }
        : { show: false },
      emphasis: {
        itemStyle: { borderColor: ctx.tokens['neutral' as SemanticToken] ?? '#888', borderWidth: 1 },
      },
    }],
  }
}
```

**Gestion `null` (point critique)** : ECharts ne supporte pas nativement les valeurs `null` dans `visualMap.continuous` — la cellule serait colorée à la valeur min. Le pattern retenu :
- Le caller passe `value: null` pour les cellules masquées et le `outOfRange` du `visualMap` les rend transparentes.
- Alternative équivalente : ne pas pousser la cellule dans `data` du tout — mais alors ECharts laisse un trou visuel, pas une cellule vide alignée. Préférer la première approche pour conserver la grille 7×24 visible.

**Tokens divergents Synthesis** : `heatmap-divergent-low` (rouge → 0%) + `heatmap-divergent-mid` (à créer dans `semantic-tokens.ts` — ambre/jaune → 50%) + `heatmap-divergent-high` (vert → 100%). **Action requise** : ajouter `heatmap-divergent-mid` à l'enum `SemanticToken` (`apps/web/src/lib/accessibility/semantic-tokens.ts:69`) et aux palettes default (`#F59E0B` amber-500) et okabe-ito (`#F0E442` Yellow). Tâche atomique, à faire en Phase 0 du méta-plan.

**Tests Vitest attendus** : snapshot grille 7×24 complète ; cas `value: null` (cellule transparente, label vide) ; cas `min_matches=0` (tout visible).

##### Wrappers Synthesis-specific hors catalogue

Deux charts Synthesis ne sont **pas** promus au catalogue méta-plan (`components/charts/`), restent dans `features/synthesis/charts/` :

1. **Combo top-by-week** (3 traces : 2 bars empilés + 1 line sur Y₂). Composition côté page via `<ChartCard>` racine + `buildOption` custom dans `features/synthesis/charts/buildTopByPeriodOption.ts`. Spec détaillée dans [`PLAN_SYNTHESIS_GO_PORTAGE.md`](./PLAN_SYNTHESIS_GO_PORTAGE.md) §8.1.
2. **Bipolaire Solo/Escouade** (2 bars horizontales miroir avec normalisation). Idem composition via `<ChartCard>`, buildOption custom dans `features/synthesis/charts/buildBipolarOption.ts`. Spec dans `PLAN_SYNTHESIS_GO_PORTAGE.md` §8.2.

Critère de promotion au catalogue : **3+ pages consommatrices** ou demande explicite. Sinon, la complexité d'un wrapper générique surpasserait l'inline `buildOption` page-level.

### 3.3 Axe 3 — Type canonique `canonical.PlayerMatchRow`

Synthesis V2 le formalise pour son besoin. Le méta-plan le **promeut comme contrat
inter-services** : tous les services qui agrègent des matchs joueur partent de cette
structure plutôt que de requêter la DB indépendamment.

#### 3.3.1 Composition

Dans `internal/games/canonical/match.go` (étend l'existant) :

```go
package canonical

type PlayerMatchRow struct {
    Summary    MatchSummary           // existant (match_id, start_time, map, mode, playlist, duration)
    Self       MatchParticipant       // existant, enrichi (KDA, KillsPerMin, AvgLifeSeconds calculés)
    Enrichment PlayerMatchEnrichment  // nouveau, LevelUp-specific
}

type PlayerMatchEnrichment struct {
    SessionID         *string
    SessionLabel      *string
    PerformanceScore  *float64
    DominanceFlag     DominanceFlag    // 0 si non calculé
    HadBotTeammate    bool
    IsWithFriends     bool
    FriendsXUIDs      []string         // sous-ensemble présent ce match
    TeamMMR           *float64
    EnemyMMR          *float64         // si dispo (head-to-head)
}

type DominanceFlag int

const (
    DominanceNone        DominanceFlag = 0
    DominanceDomination  DominanceFlag = 1
    DominanceHumiliation DominanceFlag = 2
    DominanceRemontada   DominanceFlag = 3
    DominanceDebandade   DominanceFlag = 4
    DominanceContreRem   DominanceFlag = 5
)
```

Étend `MatchParticipant` avec champs dérivables (KDA, KillsPerMin) calculés à la
construction de la row, jamais re-calculés dans les services.

**Politique d'évolution** : `canonical.PlayerMatchRow` est un contrat partagé par
6 services. Les changements obéissent à la règle :

- **Additif uniquement** : on peut ajouter un champ optionnel (`*string`, `*float64`),
  jamais supprimer ni renommer.
- **Dépréciation par tag** : un champ devenu obsolète est marqué
  `// Deprecated: use Y instead. Will be removed in vN+2.` mais reste dans le struct
  jusqu'à la version N+2.
- **Suppression** : seulement après migration explicite des 6 consommateurs et
  passage en `Deprecated` pendant au moins 1 itération.
- **Versioning du DTO** : pas de version explicite (`v1`, `v2`) côté Go ; on
  applique strictement la règle additive. Un test `canonical_test.go::TestNoBreakingChanges`
  charge un snapshot du struct figé et fait échouer la CI si un champ est supprimé
  ou renommé sans annotation `Deprecated`.

Voir ADR `docs/adr/0005-canonical-player-match-row-evolution.md` (livré en Phase 4).

#### 3.3.2 Repository unique

Dans `internal/port/player_matches.go` :

```go
package port

type PlayerMatchFilters struct {
    Period               *temporal.Period
    OutcomeIn            []canonical.Outcome
    HadBotTeammate       *bool   // pointer pour distinguer "exclure bots" vs "ne pas filtrer"
    IsFirefight          *bool
    IsRanked             *bool
    MinTimePlayedSeconds *int
    ExcludeFriendsXUIDs  []string
    BTBExcluded          bool
    PlaylistRegex        *string
    MapIDs               []string
    Limit                int    // 0 = pas de limite
    OrderBy              string // ex. "start_time DESC", "performance_score DESC"
}

type PlayerMatchesRepository interface {
    LoadPlayerMatches(
        ctx context.Context,
        slug string,
        gamertag string,
        filters PlayerMatchFilters,
    ) ([]canonical.PlayerMatchRow, error)
}
```

Implémentation dans `internal/platform/duckdb/player_matches_repo.go` (~300L) :
- Une seule requête SQL paramétrée qui jointure `shared.match_participants` +
  `shared.match_registry` + `player_match_enrichment` (player DB joueur principal).
- Préfixe `shared.` partout (corrige les bugs identifiés dans Career Q9, Career Q10).
- Exclusion bots `xuid LIKE 'bid(%'` appliquée systématiquement.
- Capability gating : retourne `games.ErrCapabilityNotSupported` si le titre n'a pas
  `match.history`.

#### 5.3.4 Caching et coalescence des appels concurrents

**Problème** : un joueur avec 10 000 matchs interrogé en parallèle par 6 services
(Squad, MatchView, Career, Synthesis, Citations, Timeseries) → 6 requêtes DuckDB
quasi identiques. Squad fait pire : intersection N coéquipiers = N appels concurrents
sur le même joueur.

**Solution** : couche cache au-dessus du repo, dans
`internal/platform/duckdb/player_matches_cache.go` (~120L).

```go
type cachedPlayerMatchesRepo struct {
    inner port.PlayerMatchesRepository
    sf    singleflight.Group         // coalescence des appels concurrents
    lru   *expirable.LRU[string, []canonical.PlayerMatchRow] // 5 min TTL, 200 entrées
}

func (c *cachedPlayerMatchesRepo) LoadPlayerMatches(
    ctx context.Context, slug, gt string, filters port.PlayerMatchFilters,
) ([]canonical.PlayerMatchRow, error) {
    key := cacheKey(slug, gt, filters) // hash stable des filtres
    if rows, ok := c.lru.Get(key); ok {
        observability.IncCounter("cache_hit_player_matches")
        return rows, nil
    }
    observability.IncCounter("cache_miss_player_matches")
    rows, err, _ := c.sf.Do(key, func() (any, error) {
        return c.inner.LoadPlayerMatches(ctx, slug, gt, filters)
    })
    if err != nil { return nil, err }
    c.lru.Add(key, rows.([]canonical.PlayerMatchRow))
    return rows.([]canonical.PlayerMatchRow), nil
}
```

- **TTL 5 minutes** : les matchs joueur ne changent qu'au sync — un sync
  invalide tout le cache du joueur (`InvalidatePlayer(slug, gt)` exposé).
- **`singleflight.Group`** : coalescence des appels concurrents sur la même clé
  → 1 seule requête DuckDB même si 6 services demandent en même temps.
- **LRU** : 200 entrées max, expulsion auto.
- Branchement : injection au boot via `app.NewPlayerMatchesRepo(inner)`.

**Tests** : `player_matches_cache_test.go` couvre :
- Hit / miss / TTL expiration.
- Coalescence : 100 goroutines concurrentes → 1 appel inner.
- Invalidation par `InvalidatePlayer`.
- Métriques `cache_hit_player_matches` / `cache_miss_player_matches` incrémentées.

#### 5.3.5 Sécurité — `PlayerMatchFilters.PlaylistRegex`

Le champ `PlaylistRegex` est un risque : interpolé dans une requête DuckDB
`regexp_matches(playlist_name, ?)`, il peut causer ReDoS (`(a+)+`) ou contourner
des filtres si l'attaquant contrôle l'input.

**Décision** : **whitelist côté handler**, pas de regex libre.

```go
// internal/api/handlers/playlist_regex_whitelist.go
var allowedPlaylistRegex = map[string]string{
    "ranked":  `(?i)ranked|classé`,
    "social":  `(?i)social|partie rapide|quick`,
    "btb":     `(?i)btb|big team battle|grande équipe`,
    "firefight": `(?i)firefight|baptême du feu`,
    // ... liste fermée
}

// Le handler reçoit un alias court (ex. ?playlist_kind=ranked) et résout
// la regex en interne. L'utilisateur ne peut jamais injecter de regex brute.
```

- Pas de regex libre dans `PlayerMatchFilters` → renommer le champ
  `PlaylistRegex` en `PlaylistKind` (alias court) et résoudre la regex au
  niveau du handler.
- Tests : `playlist_regex_whitelist_test.go` couvre injection ReDoS, alias
  inconnu (rejeté), case-insensitivity FR/EN.

#### 5.3.6 Loader unifié `HighlightEvents`

**Problème** : `shared.highlight_events` est consommé par au moins 5 pages (Squad
8 rôles d'impact, MatchView kill feed + cadence + dominance, Timeseries first
kill/death rolling + intensity heatmap, Career first_blood badges potentiels,
Synthesis). Aujourd'hui le code est éclaté : `Q32 LoadImpactEvents` dans
`squad_repo.go` couvre Squad, mais Timeseries ne charge **rien** côté
`highlight_events` et c'est ce qui bloque `first_events_rolling`. Cadence et
intensité sont signalées « MANQUANT » dans l'audit Squad et Timeseries pour la
même raison.

**Solution** : un loader unifié `port.HighlightEventsRepository` aligné sur le
pattern `LoadPlayerMatches`.

```go
// internal/games/canonical/events.go
package canonical

type HighlightEventType string

const (
    EventKill          HighlightEventType = "kill"
    EventDeath         HighlightEventType = "death"
    EventAssist        HighlightEventType = "assist"
    EventMedal         HighlightEventType = "medal"
    EventFinisher      HighlightEventType = "finisher"
    EventClutch        HighlightEventType = "clutch"
    EventFirstKill     HighlightEventType = "first_kill"
    EventFirstDeath    HighlightEventType = "first_death"
)

type HighlightEvent struct {
    MatchID     string
    EventType   HighlightEventType
    TimeMS      int64           // offset depuis début match
    KillerXUID  *string         // nil si pas applicable
    VictimXUID  *string
    PlayerXUID  *string         // pour medal / finisher / clutch
    WeaponID    *string
    Detail      map[string]any  // payload typé selon EventType
}
```

```go
// internal/port/highlight_events.go
package port

type HighlightEventFilters struct {
    MatchIDs   []string             // au moins un nécessaire (sinon trop coûteux)
    PlayerXUID *string              // filtre côté killer ou victim selon usage
    EventTypes []canonical.HighlightEventType
    Since      *time.Time
    Limit      int                  // 0 = pas de limite
    OrderBy    string               // ex. "match_id, time_ms ASC"
}

type HighlightEventsRepository interface {
    LoadHighlightEvents(
        ctx context.Context,
        slug string,
        filters HighlightEventFilters,
    ) ([]canonical.HighlightEvent, error)
}
```

Implémentation dans `internal/platform/duckdb/highlight_events_repo.go` (~200L) :
- Une seule requête SQL paramétrée sur `shared.highlight_events`.
- Préfixe `shared.` partout.
- Capability gating : `match.detail.events` ; retourne `games.ErrCapabilityNotSupported`
  si le titre n'a pas d'events filmés (PvE pur, par ex.).
- Couche cache `highlight_events_cache.go` identique à `LoadPlayerMatches`
  (`singleflight.Group` par `(slug, hash(filters))` + LRU 5 min, 200 entrées).
  Invalidation `InvalidateMatch(matchID)` exposée.

**Refacto consécutive** : `LoadImpactEvents` (Q32) dans `squad_repo.go` est
**supprimé** ; Squad consomme `LoadHighlightEvents(filters: {MatchIDs, EventTypes:[kill,death,...]})`.
Cleanup en Phase 1 lors du portage Squad.

**Helpers analysis associés** :

```go
// internal/analysis/temporal/rolling.go
// Rolling mean adaptatif réutilisable (Timeseries first_events_rolling,
// K/D rolling adaptatif, Form Score lissé, etc.)
func RollingMean[T Numeric](
    points []T,
    window int,    // ex. 10
    minPoints int, // ex. 3
) []float64

// Variante adaptative : window = max(minWindow, len(points) * pct / 100)
func RollingMeanAdaptive[T Numeric](
    points []T,
    minWindow int,    // ex. 3
    pct int,          // ex. 10 → window = 10 % de la série
) []float64
```

```go
// internal/analysis/narrative/first_events.go
// Pour chaque match, retourne (firstKillMS, firstDeathMS) du joueur ciblé.
type FirstEventsRow struct {
    MatchID       string
    StartTime     time.Time
    FirstKillMS   *int64
    FirstDeathMS  *int64
}

func ComputeFirstEventsPerMatch(
    events []canonical.HighlightEvent,
    playerXUID string,
) []FirstEventsRow
```

**Tests** : `highlight_events_repo_test.go` (chaque filtre + capability gating),
`highlight_events_cache_test.go` (hit/miss/coalescence), `rolling_test.go`
(window 10, min 3, séries vides/courtes/normales, adaptatif), `first_events_test.go`
(scénarios canoniques : aucun kill, kill seul, mort seule, les deux).

#### 5.3.7 Consommateurs

| Service | Avant | Après |
|---|---|---|
| `MatchViewService` | requête custom Q34 + Q9 + N sub-queries | `LoadPlayerMatches(filters)` + agrégations via `analysis/breakdown` |
| `SquadService` | requêtes Q30 + Q31 séparées | `LoadPlayerMatches` × N coéquipiers, intersection in-memory |
| `SynthesisService` | refonte en cours, nouveau plan | adopte directement |
| `CareerService` | Q9 incomplet (pas de filtres durs) | `LoadPlayerMatches(filters: BTB, dur≥180, outcomeIn)` + tri narrative |
| `TimeseriesService` | Q33b enrichi prévu | `LoadPlayerMatches(orderBy: start_time ASC, period)` |
| `CitationsService` | scope `compute_wins_*` | `LoadPlayerMatches(playlistKind)` |

#### 5.3.8 Consommateurs `LoadHighlightEvents`

| Service | Avant | Après |
|---|---|---|
| `SquadService` (impact 8 rôles) | `Q32 LoadImpactEvents` ad hoc dans `squad_repo.go` | `LoadHighlightEvents(filters: {MatchIDs, EventTypes:[kill,death,assist,medal]})` + `analysis/narrative.IdentifyImpactRoles` |
| `MatchViewService` (kill feed) | rien | `LoadHighlightEvents(filters: {MatchIDs:[id], EventTypes:[kill,death,assist]})` |
| `MatchViewService` (cadence intra-match) | rien | `LoadHighlightEvents(filters: {MatchIDs:[id], EventTypes:[kill]})` + bucket 60s |
| `MatchViewService` (dominance flag intermédiaire) | utilise déjà la sortie sync | inchangé (le flag est calculé au sync, pas en service) |
| `TimeseriesService` (first kill / first death rolling) | **bloqué — pas de loader** | `LoadHighlightEvents(filters: {MatchIDs:players_match_ids, EventTypes:[first_kill,first_death], PlayerXUID:gt})` + `analysis/narrative.ComputeFirstEventsPerMatch` + `analysis/temporal.RollingMeanAdaptive` |
| `TimeseriesService` (intensity heatmap match × phases) | **bloqué — pas de loader** | `LoadHighlightEvents` + bucketing 10 phases |
| `TimeseriesService` (cadence) | **bloqué — pas de loader** | `LoadHighlightEvents` + bucket 60s |
| `CareerService` (first_blood badges) | optionnel, non couvert | `LoadHighlightEvents(filters: {EventTypes:[first_kill,first_death]})` |
| `SynthesisService` | aucun usage planifié | — |

L'introduction de ce loader **débloque** 3 charts Timeseries listés MANQUANT dans
le plan enfant Timeseries (first events rolling, intensity heatmap, cadence
intra-match), 2 charts Squad (cadence trio, intensity match × phases listés
MANQUANT dans `docs/AUDIT_TEAMMATES_V7_COCKPIT.md` § 3.9 et § 4.5), et 2 sections
MatchView (kill feed, cadence).

### 3.4 Axe 4 — Manifest i18n centralisé

#### 3.4.1 Structure

Manifests TOML par domaine dans `apps/web/src/lib/i18n/manifests/` :

```
manifests/
  common.toml            terms transverses (kpi, periods, outcomes, modes communs)
  squad.toml             clés tm_*, squad_*
  match_view.toml        clés mv_*
  career.toml            clés career_*
  citations.toml         clés citations_*, medal_*, commendation_*
  timeseries.toml        clés ts_*
  synthesis.toml         clés synth_*
  home.toml, media.toml, explorer.toml, palmares.toml, session.toml
  narrative.toml         labels badges (dominance, encounter, role)
```

**Format ICU MessageFormat** (sous-set : pluralisation, interpolation typée).
Le format `{fr, en}` simple ne suffit pas — il faut gérer pluriels, genre, et
interpolation typée pour éviter les bugs de localisation.

```toml
[squad.section.synergies]
fr = "Synergies"
en = "Synergies"

[squad.kpi.team_score]
fr = "Score d'équipe"
en = "Team score"

# Pluralisation ICU MessageFormat
[squad.note.cadence]
fr = "Profil de cadence sur {n, plural, one {le dernier match} other {les # derniers matchs}}"
en = "Cadence profile over {n, plural, one {the last match} other {the last # matches}}"

# Interpolation typée (number, date)
[match_view.timestamp]
fr = "Joué le {date, date, long}"
en = "Played on {date, date, long}"

# Sélection (genre, état)
[squad.role_label]
fr = "{role, select, silent_hero {Héros silencieux} false_brother {Faux-frère} top_killer {Tueur en chef} other {Rôle}}"
en = "{role, select, silent_hero {Silent hero} false_brother {False brother} top_killer {Top killer} other {Role}}"
```

Loader runtime : lib `intl-messageformat` (~12KB gzipped, standard ICU). Wrapper
typé dans `apps/web/src/lib/i18n/format.ts` qui résout `t(key, vars)` avec
typage des `vars` dérivé du build step (les types `vars` sont générés à partir
de l'analyse des placeholders ICU dans chaque clé).

#### 3.4.2 Loader + types

Build step (script `apps/web/scripts/build_i18n_manifests.ts`) qui :
1. Parse tous les TOML au build.
2. Génère un module typé `apps/web/src/lib/i18n/generated/{domain}.ts` :
   ```ts
   export const tSquad = {
     "section.synergies": { fr: "Synergies", en: "Synergies" },
     "kpi.team_score": { fr: "Score d'équipe", en: "Team score" },
     // ...
   } as const;
   export type SquadKey = keyof typeof tSquad;
   ```
3. Génère un fichier `manifests.d.ts` qui union-type toutes les clés (pour autocomplétion globale).

API runtime :

```ts
import { useT } from '@/lib/i18n';
const { t } = useT();
const label = t('squad.section.synergies');                       // typed
const note = t('squad.note.cadence', { n: matches.length });       // interpolation
```

#### 3.4.3 Linter ESLint

Règle custom `@levelup/no-hardcoded-strings` dans `apps/web/eslint-rules/` :
- Détecte les chaînes JSX text > 2 mots qui ne sont pas dans `t(...)` ou un attribut technique.
- Détecte les `<X title="..." label="..." />` où la valeur est une chaîne longue.
- Whitelist : `data-testid`, `className`, `role`, `key`, `id`, dépendances `useEffect`, etc.
- Règle exécutée en CI ; gate de merge.

#### 3.4.4 i18n côté Go (résolution backend)

Les labels résolus par le serveur (map names, mode names, weapon names, outcome labels)
continuent de passer par `TitleSemanticAdapter` (TOML versionnés Git dans
`config/titles/{slug}/mappings/{fields,assets,outcomes}.toml`). Le manifest frontend
ne **remplace pas** ces TOML — il complète pour les strings UI pures.

#### 3.4.5 UX du composant `<CapabilityGap>`

Standardisation du rendu quand une fonctionnalité n'est pas disponible (capability
absente, scope insuffisant). Trois modes pour éviter le placeholder anémique
"Cette section n'est pas disponible".

```tsx
type CapabilityGapProps = {
  mode: 'hide' | 'placeholder' | 'cta';
  reason: string;            // clé i18n ICU dans manifest narrative
  cta?: {
    href: string;            // lien interne ou externe
    labelKey: string;        // clé i18n du libellé du CTA
  };
  // facultatif : icône via slot
  icon?: React.ReactNode;
};
```

**Modes** :

- `hide` : retourne `null`. Utilisé quand la section n'a aucun sens dans le contexte
  (ex. battlepass pour un titre sans matchmaking). Le squelette de page n'est même
  pas rendu.
- `placeholder` : affiche une carte avec l'icône, le label localisé, et une note
  discrète. Utilisé quand la donnée est attendue mais absente (ex. radar 6 axes
  si `personal_score_awards` non synchronisé).
- `cta` : `placeholder` + bouton d'action (lien vers doc d'activation, page de sync
  manuel, ou autre route). Utilisé quand l'utilisateur peut résoudre le manque.

**Décisions par section** : chaque page documente dans son plan enfant le mode
choisi par section (ex. battlepass = `placeholder`, radar = `cta` → "lancer un sync
des awards"). Une table de mapping `apps/web/src/lib/capability/gap_modes.ts`
centralise ces décisions.

**Tests** : `<CapabilityGap>.test.tsx` couvre les 3 modes × 2 langues + interaction
sur le bouton CTA.

---

## 6. Phasing

### Phase 0 — Fondations (effort ~6 j-h)

Branche : `feat/foundations-axes-1-3-4`.

#### 6.0.1 Tâches Go — code

- [ ] Créer `internal/analysis/temporal/` avec `period.go`, `bucket.go`.
- [ ] Créer `internal/analysis/breakdown/` avec `by_map.go`, `by_mode.go`, `by_playlist.go`, `compare.go`.
- [ ] Créer `internal/analysis/narrative/` avec `dominance.go`, `encounter.go`, `participation.go`, `impact_roles.go`.
- [ ] Étendre `internal/games/canonical/match.go` avec `PlayerMatchRow`, `PlayerMatchEnrichment`, `DominanceFlag`.
- [ ] Définir `internal/port/player_matches.go` avec `PlayerMatchFilters` et `PlayerMatchesRepository`.
- [ ] Implémenter `internal/platform/duckdb/player_matches_repo.go` avec capability gating (`games.ErrCapabilityNotSupported`), exclusion bots, préfixe `shared.`.
- [ ] Ajouter `internal/domain/charts.go` avec `ChartSeries[T]`, `ChartPoint2D`, `ChartPointHeatmap`, `ChartPointStacked`.
- [ ] Supprimer les champs morts `Charts.X interface{}` de `domain/career.go`, `domain/timeseries.go`, etc.
- [ ] `slog.DebugContext` sur les nouveaux loaders, `slog.ErrorContext` sur erreurs DB avec clés standards.
- [ ] **Audit + implémentation `dominance_flag` côté sync Go** :
  - [ ] Lire le code sync Go (`internal/sync/`) pour vérifier la présence de
        `_medal_verdicts.go` ou équivalent (porté de `_medal_verdicts.py`).
  - [ ] Si absent : implémenter le calcul de `dominance_flag` (1..5 :
        DOMINATION, HUMILIATION, REMONTADA, DEBANDADE, CONTRE_REMONTADA) à partir
        des scores live, du score final et des écarts intermédiaires. Tests
        unitaires Go avec 5 fixtures par flag + None.
  - [ ] Backfill des matchs existants : ajouter une option
        `levelup backfill dominance --player <gt>` (CLI Go) ou intégrer dans le
        backfill global. Tests sur DB `:memory:` avec matchs synthétiques.
  - [ ] Vérifier que `player_match_enrichment.dominance_flag` est lu correctement
        par le repo `LoadPlayerMatches`.
  - [ ] Cocher cette tâche **avant Phase 1** : sinon Squad/MatchView/Career
        retourneront `dominance_flag = None` partout et la moitié du plan est
        bloquée.

- [ ] Couche cache `internal/platform/duckdb/player_matches_cache.go` avec
      `singleflight.Group` + `expirable.LRU` (cf. § 5.3.4).
- [ ] **Loader unifié `HighlightEvents`** (cf. § 5.3.6) :
  - [ ] Étendre `internal/games/canonical/events.go` avec `HighlightEvent` + `HighlightEventType`.
  - [ ] Définir `internal/port/highlight_events.go` (`HighlightEventFilters`, `HighlightEventsRepository`).
  - [ ] Implémenter `internal/platform/duckdb/highlight_events_repo.go` (~200L) avec capability gating (`match.detail.events`).
  - [ ] Couche cache `highlight_events_cache.go` (singleflight + LRU 5 min).
  - [ ] Helper `internal/analysis/temporal/rolling.go` avec `RollingMean` et `RollingMeanAdaptive` (générique `Numeric`).
  - [ ] Helper `internal/analysis/narrative/first_events.go` avec `ComputeFirstEventsPerMatch`.
  - [ ] Tests `:memory:` du repo (chaque filtre : `MatchIDs`, `PlayerXUID`, `EventTypes`, `Since`, `Limit`, `OrderBy`, capability gating).
  - [ ] Tests `highlight_events_cache_test.go` (hit/miss/coalescence 100 goroutines).
  - [ ] Tests `rolling_test.go` (séries vide, courte, normale, paramètres adaptatifs).
  - [ ] Tests `first_events_test.go` (4 scénarios : aucun, kill seul, mort seule, les deux).
- [ ] Renommer `PlayerMatchFilters.PlaylistRegex` en `PlaylistKind` (alias court),
      ajouter whitelist côté handler (`internal/api/handlers/playlist_regex_whitelist.go`,
      cf. § 5.3.5).
- [ ] Squelette observabilité : `internal/observability/expvar_metrics.go` +
      middleware `instrumentService` + endpoint `/debug/vars` (cf. § 4.7).

#### 6.0.2 Tâches Go — tests (cf. § 3.1)

- [ ] `internal/analysis/temporal/temporal_test.go` : 5 périodes × bornes × granularités adaptive.
- [ ] `internal/analysis/breakdown/breakdown_test.go` : matrices outcomes × maps × modes, cas limites.
- [ ] `internal/analysis/narrative/dominance_test.go` : 5 flags + None.
- [ ] `internal/analysis/narrative/encounter_test.go` : ordinaux, ally_plus seuil 70 %, tough K/D 1.5.
- [ ] `internal/analysis/narrative/participation_test.go` : 3 familles de mode × bornes thresholds.
- [ ] `internal/analysis/narrative/impact_roles_test.go` : 8 rôles isolés + combinaisons.
- [ ] `internal/platform/duckdb/player_matches_repo_test.go` : 1 test par filtre (11 filtres) + combinaisons + capability gating + benchmark `:memory:`.
- [ ] `internal/platform/duckdb/player_matches_cache_test.go` : hit/miss/TTL,
      coalescence (100 goroutines → 1 appel inner), invalidation, métriques.
- [ ] `internal/api/handlers/playlist_regex_whitelist_test.go` : injection ReDoS,
      alias inconnu rejeté, case-insensitivity FR/EN.
- [ ] `internal/observability/expvar_metrics_test.go` : compteurs, durées, format JSON.
- [ ] `internal/games/canonical/canonical_test.go::TestNoBreakingChanges` : snapshot
      du struct `PlayerMatchRow`, échec si champ supprimé/renommé sans `Deprecated`.
- [ ] `internal/testutil/fixtures/match_rows.go`, `events.go`, `awards.go`, `duckdb_seed.go`.
- [ ] `internal/testutil/sloghandler.go` (capture logs pour assertion).
- [ ] `testdata/golden/halo_infinite/{shared,player}.duckdb` figés + snapshots
      initiaux + `testdata/golden/README.md` (procédure de régénération).
- [ ] Couverture Go ≥ 90 % `analysis/`, ≥ 70 % `platform/duckdb/` mesurée via `go test -cover`.
- [ ] `go test ./... -race` passe sans warning.
- [ ] `go vet ./...` passe.

#### 6.0.3 Tâches frontend — code

- [ ] Ajouter `echarts` et `echarts-for-react` au `package.json` (imports ciblés `echarts/core` + tree-shaking).
- [ ] Créer `apps/web/src/components/charts/ChartCard.tsx` (base : loading/error/empty/data, theme injection, resize observer, lazy import echarts).
- [ ] Créer `apps/web/src/components/feedback/CapabilityGap.tsx`.
- [ ] Créer `apps/web/src/lib/i18n/manifests/` avec `common.toml` + `narrative.toml`.
- [ ] Ajouter `intl-messageformat` au `package.json` (~12KB gzipped).
- [ ] Écrire le build step `apps/web/scripts/build_i18n_manifests.ts` :
  - Parse les TOML.
  - Détecte les placeholders ICU (`{n, plural}`, `{x, select}`, `{date, date}`, `{n, number}`).
  - Génère les types TypeScript des `vars` attendus pour chaque clé (typage automatique des `t()`).
  - Génère `lib/i18n/generated/*.ts` typés.
- [ ] Intégration au pipeline Vite (plugin custom).
- [ ] Wrapper runtime `apps/web/src/lib/i18n/format.ts` (utilise `intl-messageformat`).
- [ ] Écrire la règle ESLint `@levelup/no-hardcoded-strings` dans `apps/web/eslint-rules/no-hardcoded-strings.ts`.
- [ ] Activer la règle en `warn` global (Phase 2 passera en `error`).
- [ ] `<CapabilityGap>` 3 modes (`hide` / `placeholder` / `cta`) avec table de mapping
      `apps/web/src/lib/capability/gap_modes.ts` (cf. § 3.4.5).

#### 6.0.4 Tâches frontend — tests (cf. § 3.1.5, § 3.1.11, § 3.1.13)

- [ ] `ChartCard.test.tsx` : 4 états (loading, error, empty, data) + theme injection.
- [ ] `CapabilityGap.test.tsx` : labels localisés selon `reason`.
- [ ] `i18n_manifest.test.ts` : clés référencées vs définies, pas de clé orpheline (run sur le scope manifests Phase 0).
- [ ] `eslint-rules/no-hardcoded-strings.test.ts` : 10+ cas positifs / négatifs.
- [ ] `npm run typecheck && npm run lint && npm run test` passe.
- [ ] Couverture frontend ≥ 80 % sur les nouveaux fichiers.

#### 6.0.5 Done definition Phase 0

- [ ] Toutes les tâches code + tests cochées.
- [ ] Couvertures atteintes (Go et frontend).
- [ ] Aucun import Plotly/Recharts ajouté.
- [ ] Aucune string en dur dans les nouveaux composants.
- [ ] Sync `dominance_flag` audité, statut documenté dans `thought_log.md`.
- [ ] Entrée `thought_log.md` `[YYYY-MM-DD] Phase 0 — Fondations` complétée.

### Phase 1 — Pilotes Squad et MatchView (effort ~12 j-h)

Branche : `feat/foundations-pilots-squad-matchview` (depuis `feat/foundations-axes-1-3-4`).

Squad et MatchView sont les pages les plus complexes : leur succès valide les API des
fondations sur les cas extrêmes (heatmap 8 rôles, radar 6 axes, kill feed, scoreboard
détaillé). Tout ajustement d'API pour absorber un cas-limite se fait ici, pas dans
une phase abstraite.

#### 6.1.1 Squad — adoption des fondations

- [ ] Plan enfant à rédiger : `.ai/PLAN_SQUAD_GO_PORTAGE.md` (à partir de `docs/AUDIT_TEAMMATES_V7_COCKPIT.md`).
- [ ] `service/squad_service.go` : adopter `LoadPlayerMatches` × N coéquipiers, intersection in-memory.
- [ ] `service/squad_service.go` : agrégations via `analysis/breakdown` (lollipop W/L par carte, perf vs historique).
- [ ] `service/squad_service.go` : badges via `analysis/narrative.IdentifyImpactRoles` (8 rôles).
- [ ] `service/squad_service.go` : radar via `analysis/narrative.ComputeParticipationProfile`.
- [ ] Étendre l'algo `internal/analysis/match_impact.go` actuel (4 rôles bilatéral) à 8 rôles N-joueurs avec fenêtre temporelle réelle (clutch 30 dernières secondes).
- [ ] Migrer `IdentifyImpactRoles` vers le loader unifié : remplacer l'appel à `Q32 LoadImpactEvents` (squad_repo) par `LoadHighlightEvents(filters: {MatchIDs, EventTypes:[kill,death,assist,medal,clutch,finisher]})` ; supprimer `Q32` après migration.
- [ ] Câbler `<Cadence>` Squad sur `LoadHighlightEvents(EventTypes:[kill])` + bucketing 60s.
- [ ] Câbler la heatmap intensité Squad (match × 10 phases) sur `LoadHighlightEvents` + algo `ComputeMatchIntensityProfiles` (à porter dans `analysis/narrative` ou `analysis/temporal`).
- [ ] DTOs squad alignés sur `ChartSeries[T]` (suppression de tout payload Plotly server-side).
- [ ] `apps/web/src/features/squad/` : refonte de `SquadSynergiesPage.tsx` et `SquadContributionsPage.tsx`.
- [ ] Wrappers ECharts spécialisés : `<Heatmap2D>`, `<Radar>`, `<BarStacked>`, `<BarGrouped>`, `<Lollipop>`, `<Bullet>`, `<Cadence>`, `<Donut>`.
- [ ] `apps/web/src/lib/i18n/manifests/squad.toml` rempli (~80 clés FR + EN).
- [ ] `slog.InfoContext` sur entrée handler `/squad/*` avec clés `player`, `title_slug`, durée totale.

#### 6.1.2 Squad — tests

- [ ] `service/squad_service_test.go` : intersection N coéquipiers (1, 2, 3 friends), ordering stable, dégradation si capability absente.
- [ ] `analysis/narrative/impact_roles_squad_test.go` : 8 rôles attribués correctement sur 5 scénarios de match canoniques (golden test sur events).
- [ ] `api/handlers/squad_handler_test.go` : statuts 200/400/404/503, shape JSON, headers.
- [ ] Test capability gating : titre synthétique sans `match.history` → page partielle, pas 5xx.
- [ ] Tests Vitest pour chaque wrapper utilisé : option ECharts snapshot + 4 états.
- [ ] `e2e/squad.spec.ts` : sélection 3 coéquipiers, vérification 4 sections clés, switch période, capability gap visible si applicable.
- [ ] Test régression visuelle : snapshots SVG des wrappers sur fixture Squad.
- [ ] Test golden parity : `cmd/foundations_golden_parity` couvre `/squad/*`.

#### 6.1.3 MatchView — adoption des fondations

- [ ] Adapter `.ai/PLAN_MATCH_VIEW_GO_PORTAGE.md` aux fondations (passer en revue Phase D radar + Phase F encounters + Phase H kill feed/cadence).
- [ ] `service/match_view_service.go` : sous-requêtes errgroup utilisent `LoadPlayerMatches` quand applicable (history avg, dominance lookup).
- [ ] `service/match_view_service.go` : badges via `analysis/narrative.ResolveDominanceBadge` et `ComputeEncounterBadges`.
- [ ] `service/match_view_service.go` : radar participation via `ComputeParticipationProfile`.
- [ ] DTOs `MatchView*` alignés sur `ChartSeries[T]` (suppression des stubs Plotly).
- [ ] Boucler MatchView Phase J (lecture `personal_score_awards` côté Go) si pas déjà fait — sinon le radar reste vide.
- [ ] Câbler MatchView kill feed sur `LoadHighlightEvents(filters: {MatchIDs:[id], EventTypes:[kill,death,assist]})`.
- [ ] Câbler MatchView cadence intra-match sur `LoadHighlightEvents` + bucket 60s.
- [ ] Câbler MatchView impact timeline sur `LoadHighlightEvents` + `narrative.IdentifyImpactRoles` (réutilise l'algo Squad, pas de duplication).
- [ ] `apps/web/src/features/match-view/MatchViewPage.tsx` : refonte de la composition charts.
- [ ] Migration des charts Recharts → ECharts (`<BarStacked>` dominance, `<Cadence>`, `<Heatmap2D>` impact, `<Radar>` participation, `<Donut>` weapons, `<Lollipop>` antagonists).
- [ ] `apps/web/src/lib/i18n/manifests/match_view.toml` rempli (~150 clés FR + EN).
- [ ] `slog.InfoContext` sur entrée handler `/match-view/*` + clés `match_id`, `player`, durée.

#### 6.1.4 MatchView — tests

- [ ] `service/match_view_service_test.go` : composition errgroup, propagation des erreurs, fallback si une sous-requête échoue.
- [ ] `analysis/narrative/dominance_resolver_test.go` : table-driven sur les 5 flags + None.
- [ ] `analysis/narrative/encounter_badges_test.go` : ordinaux 1/2/5/100, ally_plus, tough nut.
- [ ] `analysis/narrative/participation_match_view_test.go` : awards × 3 familles de mode → 6 axes attendus.
- [ ] `api/handlers/match_view_handler_test.go` : statuts 200/400/404/503, shape JSON, headers.
- [ ] Test capability gating : titre sans `match.detail.core` → dégradation.
- [ ] Tests Vitest pour `<Radar>`, `<Cadence>`, `<Heatmap2D>` impact + `<Donut>` weapons.
- [ ] `e2e/match-view.spec.ts` : ouverture match avec dominance flag, badge visible, scoreboard détaillé, kill feed, encounters cliquables.
- [ ] Snapshots ECharts options sur fixture MatchView (1 match canonique).
- [ ] Test golden parity sur `/match-view/{id}`.

#### 6.1.5 Validation API des fondations

- [ ] Si une API d'`analysis/temporal`, `breakdown`, `narrative` doit changer pour absorber un cas Squad ou MatchView, le faire **maintenant** (l'objectif de la phase pilote est d'éviter de figer une API approximative).
- [ ] Documenter les changements d'API dans `thought_log.md` avec justification.

#### 6.1.6 Done definition Phase 1

- [ ] Squad page Synergies + Contributions opérationnelle end-to-end.
- [ ] MatchView page opérationnelle end-to-end.
- [ ] Les 2 pages utilisent uniquement les wrappers ECharts (0 Plotly, 0 Recharts).
- [ ] Aucune string en dur dans les composants Squad et MatchView (ESLint passe en `error` localement sur ces dossiers).
- [ ] Couverture Go ≥ 80 % sur `service/squad_service.go` et `service/match_view_service.go`.
- [ ] Couverture frontend ≥ 80 % sur `features/squad/` et `features/match-view/`.
- [ ] Tests E2E Squad + MatchView passent sur Chromium / Firefox / WebKit.
- [ ] Tests golden parity Squad + MatchView passent.
- [ ] `go test ./... -race`, `go vet ./...`, `npm run typecheck && npm run test && npm run lint` passent.
- [ ] Logs Info émis à l'entrée handler vérifiés via `internal/testutil/sloghandler.go`.
- [ ] Entrée `thought_log.md` `[YYYY-MM-DD] Phase 1 — Pilotes Squad + MatchView` complétée.

### Phase 2 — Roll-out (effort ~12 j-h)

Branche : `feat/foundations-rollout`.

#### 6.2.1 Pages portage restantes

- [ ] **Career** : adapter `PLAN_CAREER_GO_PORTAGE.md` sections 4.A à 4.G pour pointer
  vers `LoadPlayerMatches(filters)` (Q9 supprimé), `analysis/breakdown` (map history),
  `analysis/narrative` (top matches badges, encounter badges Némésis/Souffre-douleurs).
  Migration ECharts (`<Gauge>` rank/hero, `<TimeseriesLine>` XP history, `<Lollipop>` map W/L).
- [ ] **Synthesis** : le nouveau plan en cours adopte les fondations directement,
  pas de rebranchement nécessaire.
- [ ] **Citations** : adapter le plan pour migrer `MedalsDistributionChart` server-side
  → client (`<Histogram>` ECharts). Composite scope via `LoadPlayerMatches(playlistRegex)`.
- [ ] **Timeseries** : adapter `PLAN_TIMESERIES_GO_PORTAGE.md` Phases 1-9 pour utiliser
  `analysis/temporal` (period+granularité), `analysis/breakdown` (map breakdown),
  wrappers ECharts (`<TimeseriesLine>`, `<Heatmap2D>` WL, `<Histogram>`, `<Scatter>`,
  `<BarStacked>` outcomes over time).
- [ ] **Timeseries — débloquer `first_events_rolling`** : câbler le service sur
  `LoadHighlightEvents(filters: {MatchIDs:player_match_ids, EventTypes:[first_kill,first_death], PlayerXUID:gt})`
  + `narrative.ComputeFirstEventsPerMatch` + `temporal.RollingMeanAdaptive(window=10, minPoints=3)`.
- [ ] **Timeseries — intensity heatmap match × phases** : câbler sur
  `LoadHighlightEvents` + algo `ComputeMatchIntensityProfiles` (réutilise l'algo
  Squad porté en Phase 1).
- [ ] **Timeseries — cadence intra-match** : câbler sur `LoadHighlightEvents`
  + bucket 60s (réutilise wrapper `<Cadence>` Squad).

#### 6.2.2 Pages live

- [ ] **Home** : adoption `<KPIStrip>` partagé si introduit ; manifest `home.toml` ;
  composant `<CapabilityGap>` standardisé sur les sections désactivées (ex. battlepass
  si titre n'a pas `match.matchmaking`).
- [ ] **Media** : manifest `media.toml`. Pas de chart à migrer.
- [ ] **Explorer** : manifest `explorer.toml` ; adoption du `<PeriodFilter>` partagé
  (composant React qui consomme `temporal.Period` côté serveur).

#### 6.2.3 Pages WIP

- [ ] **Palmares** : branchement sur `analysis/narrative` (encounter badges, ordinal),
  `LoadPlayerMatches`. Migration vers wrappers ECharts si charts présents.
- [ ] **Session** : idem ; manifest `session.toml`.

#### 6.2.4 Composants UI partagés à extraire

- [ ] `<PeriodFilter>` segmented control (`apps/web/src/components/filters/PeriodFilter.tsx`).
- [ ] `<KPIStrip>` (`apps/web/src/components/layout/KPIStrip.tsx`) avec format `current` + `reference` + `trend`.
- [ ] `<NarrativeBadge>` rendant `DominanceBadge` ou `EncounterBadge` ou `ImpactRole` avec couleur via `tokenCssVar(badge.colorToken)`.

#### 6.2.5 Tâches tests Phase 2

- [ ] Tests service par page : Career, Synthesis, Citations, Timeseries, Palmares, Session avec mock `PlayerMatchesRepository`.
- [ ] Tests httptest pour chaque endpoint refactoré (statuts 200/400/404/503).
- [ ] Tests Vitest pour les nouveaux composants partagés (`<PeriodFilter>`, `<KPIStrip>`, `<NarrativeBadge>`).
- [ ] Tests E2E Playwright : `e2e/career.spec.ts`, `e2e/timeseries.spec.ts`, `e2e/synthesis.spec.ts`, `e2e/citations.spec.ts`, `e2e/i18n.spec.ts` (switch FR/EN sur 3 pages), `e2e/capability_gap.spec.ts`.
- [ ] Test régression i18n manifest étendu à tous les domaines : aucune clé orpheline, aucune clé manquante (FR + EN).
- [ ] Test linter `@levelup/no-hardcoded-strings` passé en `error` global et CI bloquante.
- [ ] Tests golden parity étendus à toutes les pages migrées.
- [ ] Couverture frontend ≥ 80 % sur features migrées + `components/charts/` + `components/{filters,layout,feedback}/`.
- [ ] Couverture Go ≥ 80 % sur services migrés.

#### 6.2.6 Done definition Phase 2

- [ ] 9 pages (6 portage + 3 live + WIP) utilisent les fondations.
- [ ] Tous les manifests i18n remplis ; ESLint `@levelup/no-hardcoded-strings` passe en
  `error` global, gate CI.
- [ ] Aucune string en dur dans `apps/web/src/features/`.
- [ ] Tests golden parity Career + Synthesis + Citations + Timeseries passent.
- [ ] `go test ./... -race`, `go vet ./...`, `npm run typecheck && npm run test && npm run lint` passent.
- [ ] E2E Playwright passe sur 3 navigateurs.
- [ ] Bundle web inclut ECharts ; Plotly et Recharts encore présents (cohabitation transitoire mesurée).
- [ ] Entrée `thought_log.md` complétée.

### Phase 3 — Cleanup Plotly / Recharts (effort ~3 j-h)

Branche : `feat/foundations-cleanup-plotly`.

#### 6.3.1 Tâches

- [ ] Vérifier qu'aucune feature ne consomme `react-plotly.js`, `plotly.js*`, `recharts` (`grep -r` ciblé).
- [ ] Supprimer du `package.json` : `react-plotly.js`, `plotly.js`, `plotly.js-basic-dist`, `recharts`, `@types/react-plotly.js`.
- [ ] Supprimer les wrappers obsolètes :
  - [ ] `apps/web/src/components/ui/plotly-chart.tsx`
  - [ ] `apps/web/src/components/ui/timeseries-line-chart.tsx`
  - [ ] `apps/web/src/components/ui/timeseries-heatmap.tsx`
  - [ ] `apps/web/src/components/ui/timeseries-histogram.tsx`
  - [ ] `apps/web/src/components/ui/timeseries-scatter.tsx`
  - [ ] `apps/web/src/components/ui/timeseries-kda-bars.tsx`
  - [ ] `apps/web/src/components/ui/combat-yield-timeseries.tsx`
  - [ ] `apps/web/src/features/career/CareerChartsSection.tsx` (déjà obsolète, voir audit)
- [ ] Activer la règle ESLint `no-restricted-imports` interdisant `plotly.js*` et `recharts`.
- [ ] Vérifier la baisse de bundle (`npm run build` + bundle analyzer) : attendu ~-2.5 MB.

#### 6.3.2 Tests Phase 3

- [ ] CI gate : aucun import `plotly.js*` ni `recharts` détecté (grep + ESLint).
- [ ] Bundle analyzer : taille du bundle JS principal réduite d'au moins 2 MB par rapport à la baseline début Phase 3.
- [ ] Toutes les pages chargent toujours sans erreur console (E2E smoke).
- [ ] `go test ./...`, `go vet ./...`, `npm run typecheck && npm run test && npm run lint` passent.

#### 6.3.3 Done definition Phase 3

- [ ] Aucun import `plotly.js*` ni `recharts` dans `apps/web/src/`.
- [ ] Bundle web allégé (vérifié via analyzer).
- [ ] CI gate ESLint `no-restricted-imports` actif.
- [ ] Entrée `thought_log.md` `[YYYY-MM-DD] Phase 3 — Cleanup` complétée.

### Phase 4 — Documentation et skills (effort ~3 j-h)

Branche : `feat/foundations-docs-skills`. Peut être démarrée en parallèle de Phase 3.

#### 6.4.1 Documentation humaine

- [ ] `docs/FOUNDATIONS_GUIDE.md` (anglais) : guide consolidé des 4 fondations avec
  exemples de consommation (`LoadPlayerMatches` + `breakdown.ByMap` + `<Heatmap2D>`).
  Sections : motivation, layered architecture, examples, FAQ.
- [ ] `docs/FR/FOUNDATIONS_GUIDE.md` : version française synchronisée (cf. CLAUDE.md règle 18).
- [ ] `docs/adr/0001-charts-stack-echarts.md` : ADR sur le choix ECharts (contexte,
  décision, alternatives évaluées Plotly/Recharts/Visx, conséquences).
- [ ] `docs/adr/0002-canonical-player-match-row.md` : ADR sur `canonical.PlayerMatchRow`
  comme contrat partagé.
- [ ] `docs/adr/0003-i18n-manifest-and-linter.md` : ADR manifest TOML + ESLint custom.
- [ ] `docs/adr/0004-narrative-engine.md` : ADR sur `analysis/narrative` (8 rôles + radar 6 axes).
- [ ] `internal/analysis/temporal/README.md` : usage rapide + 3 exemples.
- [ ] `internal/analysis/breakdown/README.md` : idem.
- [ ] `internal/analysis/narrative/README.md` : idem.
- [ ] `apps/web/src/components/charts/README.md` : catalogue des 14 wrappers avec
  signature props + cas d'usage par page.
- [ ] Mise à jour `docs/ARCHITECTURE_V6.md` (ou successeur Go) avec les fondations
  promues couches transverses.
- [ ] Mise à jour `docs/MIGRATION_GAP_PYTHON_TO_GO.md` § stack technique.

#### 6.4.2 Documentation agents IA

- [ ] Mise à jour `.ai/project_map.md` : nouveau découpage `analysis/temporal`,
  `breakdown`, `narrative`, `domain/charts`, `port/player_matches`, `components/charts`,
  `lib/i18n/manifests`.
- [ ] Mise à jour `.ai/data_lineage.md` : flux data
  `shared.match_participants → LoadPlayerMatches → breakdown/narrative → ChartSeries → ECharts`.
- [ ] Mise à jour `CLAUDE.md` : nouvelles règles transverses
  - règle "tout service consommant des matchs joueur passe par `LoadPlayerMatches`"
  - règle "aucun import `plotly.js*` ni `recharts`"
  - règle "toute string UI passe par un manifest i18n typé"
  - règle "les badges narratifs sont résolus via `analysis/narrative`, jamais
    inline dans un service ou un composant"
- [ ] Entrée finale `.ai/thought_log.md` consolidée résumant les 4 phases.

#### 6.4.3 Skills `.claude/skills/` à mettre à jour

Les skills Claude Code activés par les hooks doivent refléter les fondations.

- [ ] **`arch-rules`** (HIGH) : ajouter une section "Fondations transverses" listant
  les 3 sous-packages `analysis/{temporal,breakdown,narrative}`, le contrat
  `canonical.PlayerMatchRow`, l'interface `port.PlayerMatchesRepository`. Ajouter
  les conventions de logging (clés standards, niveaux par couche).
- [ ] **`plan-review`** (CRITICAL) : ajouter à la grille les critères :
  - "Le plan utilise-t-il `LoadPlayerMatches` plutôt que des SQL ad hoc ?"
  - "Les charts passent-ils par les wrappers `components/charts/` ECharts ?"
  - "Les strings UI sont-elles dans un manifest i18n ?"
  - "Les badges narratifs sont-ils résolus via `analysis/narrative` ?"
  - Critères tests étendus : couverture par couche, golden parity, capability gating, i18n manifest.
- [ ] **`delivery-checklist`** (CRITICAL) : ajouter aux checks pré-merge :
  - "Aucun import `plotly.js*` ni `recharts`."
  - "Manifest i18n complet (FR + EN), aucune clé orpheline, aucune clé manquante."
  - "Bundle analyzer mesuré, pas de régression > 200 KB sans justification."
  - "Test golden parity passe pour les pages touchées."
  - "Couvertures atteintes : ≥ 90 % `analysis/`, ≥ 80 % `service/`, ≥ 70 % `platform/`."
- [ ] **`frontend-patterns`** (skill existant) : ré-écrire la section charts pour
  pointer vers les wrappers ECharts. Ajouter section i18n manifest + `useT`. Ajouter
  section `<CapabilityGap>` et `<NarrativeBadge>`.
- [ ] **`canonical-types`** (skill existant) : documenter `PlayerMatchRow`,
  `PlayerMatchEnrichment`, `DominanceFlag`, exemples de consommation.
- [ ] Créer un nouveau skill **`foundations-usage`** : guide condensé pour qu'un agent
  IA (ou un dev humain) sache comment consommer les fondations dans un nouveau service
  ou un nouveau composant. Contenu : 3 exemples de bout-en-bout (Go service, React page,
  ajout d'un nouveau wrapper chart). Registration dans `.claude/skills/` avec frontmatter.

#### 6.4.4 Tests Phase 4

- [ ] Test de cohérence des ADR : chaque ADR référence un fichier ou commit existant.
- [ ] Vérification de la sync `docs/` ↔ `docs/FR/` (au moins ouverture des deux fichiers).
- [ ] Linter markdown sur les nouveaux docs (pas de lien mort).
- [ ] Lecture humaine relue : un dev qui n'a pas suivi les phases doit pouvoir partir
  de `docs/FOUNDATIONS_GUIDE.md` et écrire un nouveau service en consommant les
  fondations.

#### 6.4.5 Done definition Phase 4

- [ ] Tous les documents listés (humains + ADR + READMEs) créés et synchronisés FR/EN
  où applicable.
- [ ] Les 5 skills existants mis à jour, le skill `foundations-usage` créé.
- [ ] `CLAUDE.md` reflète les 4 nouvelles règles transverses.
- [ ] `.ai/project_map.md` et `.ai/data_lineage.md` à jour.
- [ ] Lecture finale validée par un humain (test "j'embarque un nouveau dev avec ce
  guide, est-ce qu'il peut produire ?").
- [ ] Entrée `thought_log.md` `[YYYY-MM-DD] Phase 4 — Documentation et skills` complétée.

---

> **FIN DU PHASING (Bloc C).** Les sections suivantes (§ 7 à § 11) sont des
> **références transverses** : elles ne décrivent pas une Phase 5 ou suivante,
> elles documentent l'impact, les risques, l'estimation, la done definition et
> les annexes pour l'ensemble des 5 phases ci-dessus. Elles sont consultées
> ponctuellement pendant l'exécution, pas lues séquentiellement.

---

## 7. Référence — Impact détaillé sur les plans enfants

### 7.1 Tableau d'impact

| Plan enfant | Section impactée | Action |
|---|---|---|
| `PLAN_MATCH_VIEW_GO_PORTAGE.md` | Phase D (radar), F (encounters), H (kill feed/cadence), I (charts) | Réécrire pour pointer vers `analysis/narrative` + `LoadPlayerMatches` + wrappers ECharts. |
| `PLAN_TIMESERIES_GO_PORTAGE.md` | Phases 1-9 | Remplacer Q33b enrichi custom par `LoadPlayerMatches`. Stubs `*_chart: PlotlyFigurePayload` supprimés. ECharts wrappers. |
| `PLAN_CITATIONS_GO_PORTAGE.md` | Phase 2.x (distribution chart), R03 (compute_wins regex) | `MedalsDistributionChart` migrée client. Scope via `LoadPlayerMatches(playlistRegex)`. |
| `PLAN_CAREER_GO_PORTAGE.md` | Phase A (DTO), B (Q9, Q10, Q11), E (frontend), F (tests) | DTOs alignés `ChartSeries[T]`. Q9/Q10/Q11 réécrits avec `LoadPlayerMatches` + `breakdown`. Charts ECharts. |
| `PLAN_SYNTHESIS_GO_PORTAGE.md` | Phase 0 (canonical) | Le canonical row formalisé ici devient l'API publique. La phase 0 du plan Synthesis disparaît. |
| `docs/AUDIT_TEAMMATES_V7_COCKPIT.md` § 11 | Briques transverses § 11.3 | Liste des helpers déjà couverte par les fondations (breakdown, temporal, narrative, ParticipationProfile, IdentifyImpactRoles). Le plan Squad à rédiger pointera vers les fondations. |

### 7.2 Stratégie d'amendement des plans enfants

Pour éviter qu'un dev ou un agent IA implémente du code mort en partant d'un plan
enfant périmé, l'amendement se fait en **3 niveaux échelonnés** :

#### Niveau 1 — Annotation en tête (Phase 0, immédiat)

Chaque plan enfant reçoit en tête un encart **"Note d'amendement"** qui :
- Renvoie vers `PLAN_META_FOUNDATIONS_GO.md`.
- Liste les sections obsolètes / à refactorer / à conserver.
- Indique la phase du méta-plan où le plan sera réécrit.

But : la lecture du plan enfant signale **immédiatement** ce qui est périmé.

L'annotation est rédigée sans toucher au corps du plan ; la richesse historique est
conservée.

#### Niveau 2 — Réécriture lors de la phase concernée

Quand le méta-plan attaque une page, le plan enfant correspondant est **réécrit**
avec API stabilisée :
- **Phase 1** : `PLAN_MATCH_VIEW_GO_PORTAGE.md` réécrit ; nouveau `PLAN_SQUAD_GO_PORTAGE.md` rédigé.
- **Phase 2** : `PLAN_CAREER_GO_PORTAGE.md`, `PLAN_CITATIONS_GO_PORTAGE.md`,
  `PLAN_TIMESERIES_GO_PORTAGE.md` réécrits.
- **Phase 2 / Synthesis** : le nouveau plan Synthesis (en cours côté équipe) est
  rédigé directement sur les fondations, sans amendement intermédiaire.

#### Niveau 3 — Archivage (Phase 4)

Les plans dont toute la substance est passée dans le code et dans le méta-plan
reçoivent un encart final `> Implémenté via PLAN_META_FOUNDATIONS_GO et commits cités.
Conservé pour traçabilité historique.` et déménagent dans `.ai/archive/`.

`docs/AUDIT_TEAMMATES_V7_COCKPIT.md` n'est **pas amendé** — c'est un audit Python,
pas un plan d'implémentation.

#### Tâches Phase 0 ajoutées par § 7.2

- [ ] Annotation niveau 1 ajoutée en tête de `PLAN_MATCH_VIEW_GO_PORTAGE.md`.
- [ ] Annotation niveau 1 ajoutée en tête de `PLAN_TIMESERIES_GO_PORTAGE.md`.
- [ ] Annotation niveau 1 ajoutée en tête de `PLAN_CITATIONS_GO_PORTAGE.md`.
- [ ] Annotation niveau 1 ajoutée en tête de `PLAN_CAREER_GO_PORTAGE.md`.
- [ ] Annotation niveau 1 ajoutée en tête de `PLAN_SYNTHESIS_GO_PORTAGE.md`.

### 7.3 Stratégie de mise en production — pas de cohabitation v1/v2

**Décision** : pas de fallback gracieux entre l'ancienne forme et la nouvelle. Pas
de `?v=2`, pas de header `Accept-Version`, pas de double-écriture en parallèle.

**Justification** : à l'exception de **Home** et **Media** (les seules pages
considérées comme stables et validées), aucune page de portage n'est en service
prod stable. La cohabitation v1/v2 introduirait un coût de maintenance supérieur
au coût d'une rupture franche, pour un bénéfice utilisateur nul (pas d'utilisateur
final qui dépende d'une v1 figée).

**Règle de release** :

- **Synchronisation backend ↔ frontend par PR** : tout PR qui modifie un DTO doit
  contenir le code frontend qui consomme la nouvelle forme. Aucun déploiement
  partiel.
- **Une page = une PR de bascule** : Career frontend + backend dans le même PR,
  Squad frontend + backend dans le même PR, etc. Pas de PR backend "préparatoire"
  qui livre un DTO non consommé.
- **Pages stables intouchées** : Home et Media n'ont pas de DTOs consommant
  `PlayerMatchRow`. Leurs DTOs ne changent pas en Phase 0/1/2. Si un changement
  est forcé (ex. adoption manifest i18n), il est fait en backward-compatible :
  on ajoute des clés sans casser l'existant.
- **Rollback** : `git revert` du PR de bascule. Le sync DB et les snapshots
  golden sont compatibles entre versions, donc revert toujours sûr.

**Conséquences pour les pages instables** (Career, Synthesis, Citations,
Timeseries, Squad, MatchView, Palmares, Session, Explorer) :

- Pendant la branche de portage, la page peut être cassée. C'est OK : c'est une
  branche de feature, pas la prod.
- Le merge sur `main` se fait quand la page est complète et passe les tests.
- L'utilisateur qui ouvre une de ces pages avant son merge voit potentiellement
  un état dégradé (404, message "page en cours de migration", ou ancien rendu).
  **Acceptable** vu le périmètre.

**Aucune compatibilité ascendante exigée** sur les DTOs, les query keys, les
shapes Plotly server-side. Le bénéfice de la rupture franche est de pouvoir
écrire la cible directement, sans gymnastique de migration.

---

## 8. Référence — Risques et mitigations

| Risque | Probabilité | Impact | Mitigation |
|---|---|---|---|
| API `analysis/narrative` mal calibrée → refacto en cascade | Moyenne | Haut | Phase 1 (pilotes) sert précisément à valider l'API sur Squad+MatchView. Tout changement avant Phase 2. |
| ECharts a un comportement de re-render différent de Plotly → bug visuel subtil | Moyenne | Moyen | Tests Vitest snapshot des wrappers + revue manuelle des pages pilotes en navigateur (CLAUDE.md règle UI). |
| `dominance_flag` non peuplé par le sync Go → top matches Career/MatchView dégradés | Haute | Haut | Tâche Phase 0 explicite (cf. § 6.0.1) : audit + implémentation `_medal_verdicts` Go + backfill via CLI `levelup backfill dominance`. À cocher avant Phase 1, sinon Squad/MatchView/Career bloqués. |
| Cache `LoadPlayerMatches` invalidé incorrectement → données stales servies après sync | Moyenne | Moyen | TTL 5 min court + invalidation explicite `InvalidatePlayer(slug, gt)` appelée après chaque sync. Tests dédiés (cf. § 6.0.2). |
| ICU MessageFormat lib alourdit le bundle | Faible | Faible | `intl-messageformat` ~12KB gzipped, négligeable face à ECharts. Aucune mitigation nécessaire. |
| Whitelist `playlist_kind` incomplète → joueurs Steam EN n'ont pas de match | Moyenne | Moyen | Tests E2E sur joueurs FR + EN ; alias multilingues dans la whitelist (`(?i)ranked|classé`). |
| Évolution non-rétrocompatible de `canonical.PlayerMatchRow` casse 6 services | Faible | Haut | Test `TestNoBreakingChanges` en CI + politique additive (cf. § 5.3.1). |
| Snapshots golden divergent en CI sans `--update` → CI bloquée | Moyenne | Faible | Doc claire `testdata/golden/README.md` + label GitHub auto. |
| `LoadHighlightEvents` sans filtre `MatchIDs` → scan complet de `shared.highlight_events` | Moyenne | Haut | Validation côté repo : `MatchIDs` obligatoire (au moins 1) sauf si `PlayerXUID + Since` ; sinon retour `errors.New("highlight_events: filter too broad")`. Tests d'erreur en place. |
| `Q32 LoadImpactEvents` (squad) supprimé sans avoir migré tous ses appelants | Faible | Moyen | Cleanup Q32 cochable seulement après migration Squad pilote validée (Phase 1 done definition). Grep sur `LoadImpactEvents` doit retourner 0 occurrence avant de supprimer. |
| `personal_score_awards` jamais lu côté Go → radar 6 axes vide | Haute | Haut | Plan MatchView Phase J (1 semaine) doit être bouclé avant que la phase 1 MatchView ne câble le radar. À sécuriser en Phase 0. |
| Manifest i18n explose (300+ clés) → maintenance lourde | Moyenne | Moyen | Découpe par domaine (un TOML par page), build step typé, revue ESLint. |
| Bundle ECharts plus lourd que prévu si tree-shaking mal configuré | Faible | Moyen | Imports ciblés (`echarts/core`, `echarts/charts/*`), bundle analyzer en CI. |
| Cohabitation Plotly + ECharts en Phase 2 alourdit temporairement le bundle | Certaine | Faible | Phase 3 nettoie ; +2.5 MB toléré 2-3 semaines. |
| Migration progressive Phase 2 désynchronise les pages → comportement incohérent | Moyenne | Moyen | Migrer une page à la fois, tests par page, pas de big-bang. |

---

## 9. Référence — Estimation totale et planning

| Phase | Effort | Branche | Bloque |
|---|---:|---|---|
| 0 — Fondations (incl. sync dominance, cache, ICU, observabilité, whitelist, HighlightEvents loader) | ~11 j-h | `feat/foundations-axes-1-3-4` | Phase 1 |
| 1 — Pilotes Squad + MatchView (incl. cadence, kill feed, intensity, refacto Q32) | ~13 j-h | `feat/foundations-pilots-squad-matchview` | Phase 2 |
| 2 — Roll-out (incl. Timeseries first_events_rolling, intensity, cadence) | ~13 j-h | `feat/foundations-rollout` | Phase 3 |
| 3 — Cleanup Plotly / Recharts | ~3 j-h | `feat/foundations-cleanup-plotly` | — |
| 4 — Documentation et skills (10 ADR, guides, skills) | ~4 j-h | `feat/foundations-docs-skills` | — |
| **Total** | **~44 j-h** | | |

Calendrier indicatif (1 dev) : ~7-8 semaines de travail effectif. Phases 3 et 4
parallélisables (un dev peut nettoyer Plotly pendant qu'un autre rédige la doc et
met à jour les skills).

---

## 10. Référence — Done definition globale

- [ ] Les 4 fondations livrées et utilisées (axes 1 helpers, 2 charts, 3 canonical, 4 i18n).
- [ ] Les 6 pages de portage utilisent les fondations.
- [ ] Les 3 pages live + 2 WIP utilisent les manifests i18n et le `<CapabilityGap>`.
- [ ] Plotly et Recharts retirés du frontend.
- [ ] ESLint `no-hardcoded-strings` actif en `error` sur `apps/web/src/features/` et `apps/web/src/components/`.
- [ ] Bundle web allégé (vérifié via analyzer, ≥ 2 MB économisés).
- [ ] Couverture Go ≥ 90 % `analysis/`, ≥ 80 % `service/`, ≥ 70 % `platform/duckdb/`.
- [ ] Couverture frontend ≥ 80 % sur `components/charts/` et `features/{squad,match-view,career,timeseries,synthesis,citations}/`.
- [ ] Tests E2E Playwright passent sur 3 navigateurs.
- [ ] Tests golden parity passent sur 6 pages.
- [ ] `go test ./... -race`, `go vet ./...`, `npm run typecheck && npm run test && npm run lint` passent.
- [ ] `dominance_flag` peuplé sur tous les matchs synchronisés (sync Go + backfill).
- [ ] Cache `LoadPlayerMatches` actif, taux de hit > 50 % en charge nominale.
- [ ] Whitelist `playlist_kind` couvre FR + EN, tests d'injection ReDoS passent.
- [ ] Observabilité `expvar` exposée sur `/debug/vars` avec les 4 métriques de base.
- [ ] ICU MessageFormat actif sur tous les manifests, types `vars` générés au build.
- [ ] `<CapabilityGap>` 3 modes implémenté, table `gap_modes.ts` peuplée.
- [ ] Snapshots golden versionnés Git, procédure `--update` documentée et testée.
- [ ] Test `TestNoBreakingChanges` actif sur `canonical.PlayerMatchRow`.
- [ ] Documentation humaine livrée (`docs/FOUNDATIONS_GUIDE.md` FR + EN, 9 ADR, READMEs sous-packages).
- [ ] Documentation agents IA livrée (`.ai/project_map.md`, `.ai/data_lineage.md`, `CLAUDE.md` règles).
- [ ] 5 skills existants mis à jour + skill `foundations-usage` créé.
- [ ] 5 entrées `thought_log.md` (1 par phase).
- [ ] Tous les plans enfants mis à jour pour pointer vers les fondations.

---

## 11. Référence — Annexes

### 11.1 Liste exhaustive des helpers à factoriser (post-audit des 6 plans)

| Helper | Source(s) | Fondation cible |
|---|---|---|
| `FilterByPeriod` | Synthesis, Timeseries, Career, MatchView | `analysis/temporal` |
| `BucketTimeAdaptive` | Synthesis, Timeseries, MatchView | `analysis/temporal` |
| `ResolveAdaptiveGranularity` | Synthesis, Timeseries | `analysis/temporal` |
| `ComputeMapBreakdown` | Synthesis, Timeseries, Career, Squad | `analysis/breakdown.ByMap` |
| `ComputeModeBreakdown` | Synthesis, Timeseries | `analysis/breakdown.ByMode` |
| `CompareToHistorical` | Squad (perf vs historique), Career (history) | `analysis/breakdown.CompareToHistorical` |
| `ResolveDominanceBadge` | MatchView, Career, Synthesis | `analysis/narrative.ResolveDominanceBadge` |
| `ComputeEncounterBadges` | MatchView, Career, Squad | `analysis/narrative.ComputeEncounterBadges` |
| `ComputeParticipationProfile` | Squad, MatchView | `analysis/narrative.ComputeParticipationProfile` |
| `IdentifyImpactRoles` (8 rôles) | Squad | `analysis/narrative.IdentifyImpactRoles` |
| `RelativeDate` (FR/EN) | MatchView, Career, Squad | `apps/web/src/lib/i18n/format.ts` |
| `FormatMMSS`, `FormatDuration` | tous | `apps/web/src/lib/format/duration.ts` |
| Friends XUIDs exclusion | Career, MatchView, Squad | `port.PlayerMatchFilters.ExcludeFriendsXUIDs` |
| `LoadPlayerMatches(filters)` | tous (au lieu de N requêtes custom) | `port.PlayerMatchesRepository` |
| `ChartSeries[T]` | tous | `domain/charts.go` |

### 11.2 Tokens couleur sémantiques attendus

À ajouter dans `apps/web/src/lib/accessibility/palettes/` si absents :

```
narrative.dominance.win.strong       # DOMINATION (vert profond)
narrative.dominance.win.comeback     # REMONTADA (bleu)
narrative.dominance.win.counter      # CONTRE_REMONTADA (cyan)
narrative.dominance.loss.strong      # HUMILIATION (violet)
narrative.dominance.loss.collapse    # DEBANDADE (orange foncé)
narrative.encounter.ally_plus        # vert clair
narrative.encounter.tough_ally       # gris-bleu
narrative.encounter.tough_enemy      # rouge-orangé
narrative.role.silent_hero           # vert
narrative.role.false_brother         # rouge
narrative.role.top_killer            # jaune
narrative.role.last_casualty         # gris
narrative.role.last_group_kill       # gris foncé
narrative.role.first_group_death     # rouge foncé
narrative.role.first_blood           # vert clair
narrative.role.clutch_finisher       # bleu clair
```

### 11.3 Fichiers Go nouveaux (récap)

```
internal/analysis/temporal/period.go
internal/analysis/temporal/bucket.go
internal/analysis/temporal/temporal_test.go
internal/analysis/breakdown/by_map.go
internal/analysis/breakdown/by_mode.go
internal/analysis/breakdown/by_playlist.go
internal/analysis/breakdown/compare.go
internal/analysis/breakdown/breakdown_test.go
internal/analysis/narrative/dominance.go
internal/analysis/narrative/encounter.go
internal/analysis/narrative/participation.go
internal/analysis/narrative/impact_roles.go
internal/analysis/narrative/narrative_test.go
internal/games/canonical/match.go            (étendu)
internal/port/player_matches.go              (nouveau)
internal/platform/duckdb/player_matches_repo.go (nouveau, ~300L)
internal/platform/duckdb/player_matches_repo_test.go
internal/platform/duckdb/player_matches_cache.go (nouveau, ~120L)
internal/platform/duckdb/player_matches_cache_test.go
internal/games/canonical/events.go            (HighlightEvent, HighlightEventType)
internal/port/highlight_events.go             (HighlightEventFilters, repo interface)
internal/platform/duckdb/highlight_events_repo.go      (~200L)
internal/platform/duckdb/highlight_events_repo_test.go
internal/platform/duckdb/highlight_events_cache.go     (singleflight + LRU)
internal/platform/duckdb/highlight_events_cache_test.go
internal/analysis/temporal/rolling.go          (RollingMean, RollingMeanAdaptive)
internal/analysis/temporal/rolling_test.go
internal/analysis/narrative/first_events.go    (ComputeFirstEventsPerMatch)
internal/analysis/narrative/first_events_test.go
internal/analysis/narrative/intensity_profile.go  (ComputeMatchIntensityProfiles, 10 buckets)
internal/analysis/narrative/intensity_profile_test.go
internal/api/handlers/playlist_regex_whitelist.go
internal/api/handlers/playlist_regex_whitelist_test.go
internal/observability/expvar_metrics.go     (nouveau, ~80L)
internal/observability/expvar_metrics_test.go
internal/games/canonical/canonical_test.go   TestNoBreakingChanges
internal/domain/charts.go                    (nouveau)
testdata/golden/halo_infinite/shared.duckdb  (figé)
testdata/golden/halo_infinite/player.duckdb  (figé)
testdata/golden/snapshots/{page}.json        (figés, regen via cmd/foundations_golden_parity --update)
testdata/golden/README.md                    (procédure regen + cas légitimes)
cmd/foundations_golden_parity/main.go
```

### 11.4 Fichiers frontend nouveaux (récap)

```
apps/web/src/components/charts/ChartCard.tsx
apps/web/src/components/charts/TimeseriesLine.tsx
apps/web/src/components/charts/TimeseriesArea.tsx
apps/web/src/components/charts/BarStacked.tsx
apps/web/src/components/charts/BarGrouped.tsx
apps/web/src/components/charts/Heatmap2D.tsx
apps/web/src/components/charts/Radar.tsx
apps/web/src/components/charts/Bullet.tsx
apps/web/src/components/charts/Lollipop.tsx
apps/web/src/components/charts/Histogram.tsx
apps/web/src/components/charts/Scatter.tsx
apps/web/src/components/charts/Donut.tsx
apps/web/src/components/charts/Gauge.tsx
apps/web/src/components/charts/Cadence.tsx
apps/web/src/components/charts/__tests__/*.test.tsx
apps/web/src/components/filters/PeriodFilter.tsx
apps/web/src/components/layout/KPIStrip.tsx
apps/web/src/components/feedback/CapabilityGap.tsx (3 modes : hide/placeholder/cta)
apps/web/src/components/feedback/NarrativeBadge.tsx
apps/web/src/lib/capability/gap_modes.ts                   (table de mapping section → mode)
apps/web/src/lib/i18n/format.ts                            (wrapper intl-messageformat)
apps/web/src/lib/i18n/manifests/{common,squad,match_view,career,citations,timeseries,synthesis,home,media,explorer,palmares,session,narrative}.toml
apps/web/src/lib/i18n/generated/*.ts          (build artifact, gitignored)
apps/web/scripts/build_i18n_manifests.ts
apps/web/eslint-rules/no-hardcoded-strings.ts
apps/web/eslint-rules/no-hardcoded-strings.test.ts
```

### 11.5 Documents et skills à livrer (Phase 4)

```
docs/FOUNDATIONS_GUIDE.md                   guide humain EN (fondations + exemples)
docs/FR/FOUNDATIONS_GUIDE.md                guide humain FR synchronisé
docs/adr/0001-charts-stack-echarts.md
docs/adr/0002-canonical-player-match-row.md
docs/adr/0003-i18n-manifest-and-linter.md
docs/adr/0004-narrative-engine.md
docs/adr/0005-canonical-player-match-row-evolution.md   politique additive + test TestNoBreakingChanges
docs/adr/0006-no-v1-v2-cohabitation.md                  rupture franche, sync backend/frontend par PR
docs/adr/0007-player-matches-cache-strategy.md          singleflight + LRU 5 min
docs/adr/0008-playlist-regex-whitelist.md               whitelist côté handler, pas de regex libre
docs/adr/0009-observability-expvar-baseline.md          expvar minimal, pas de Prometheus cette itération
docs/adr/0010-highlight-events-unified-loader.md        loader unifié + cache, refacto Q32 squad_repo
internal/analysis/temporal/README.md
internal/analysis/breakdown/README.md
internal/analysis/narrative/README.md
apps/web/src/components/charts/README.md

.claude/skills/foundations-usage/SKILL.md   nouveau skill (frontmatter + corps)
.claude/skills/arch-rules/SKILL.md          mis à jour (section fondations)
.claude/skills/plan-review/SKILL.md         mis à jour (critères tests étendus + fondations)
.claude/skills/delivery-checklist/SKILL.md  mis à jour (ECharts + manifest + couvertures)
.claude/skills/frontend-patterns/SKILL.md   mis à jour (wrappers ECharts + i18n manifest)
.claude/skills/canonical-types/SKILL.md     mis à jour (PlayerMatchRow + DominanceFlag)

CLAUDE.md                                   règles transverses ajoutées
.ai/project_map.md                          nouveau découpage
.ai/data_lineage.md                         flux data fondations
```

### 11.6 Coverage cibles consolidées

| Couche / dossier | Cible |
|---|---|
| `internal/analysis/` | ≥ 90 % |
| `internal/service/` | ≥ 80 % |
| `internal/platform/duckdb/` | ≥ 70 % |
| `internal/api/handlers/` | ≥ 80 % |
| `apps/web/src/components/charts/` | ≥ 80 % |
| `apps/web/src/features/squad/` | ≥ 80 % |
| `apps/web/src/features/match-view/` | ≥ 80 % |
| `apps/web/src/features/{career,timeseries,synthesis,citations}/` | ≥ 70 % |
| `apps/web/src/lib/i18n/` | ≥ 90 % (manifest loader + tests cross-référence) |
| `apps/web/eslint-rules/` | ≥ 95 % (règle + cas) |
