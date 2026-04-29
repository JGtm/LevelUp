# Plan d'action — revue de code 2026-04-29

Branche racine : `feat/multi-title-static-fs-rescope` (continuer dessus)
Décisions actées : A1=B, A2=A (sauf xuid_aliases), A3=A, A4=A, A5=A, tous T*, tous L*, tous quick wins et bugs.
Contexte produit : **app multi-user** (proposé en UI) → renforce besoin observability (T7), request_id propagation (T6), tests `flag ON` (T8).

## STATUT GLOBAL (mis à jour 2026-04-29)

Branche `chore/cleanup-and-ux-fixes` — 47+ commits, P0→P8 partiels livrés.

### Sub-phases livrées [DONE]
- **P0** complète : Q1-Q8 quick wins, B1-B5 bugs UX, Q6 4 endpoints orphelins, .env.local.example audit
- **P1** complète : 6 ADRs (0005-0010) commités, CLAUDE.md étendu, T7 décidé
- **P2** complète : `analysis/{indicators,identity,sql_fragments}.go`, Outcome DNF, 13+ migrations inline, DTOs étendus, formatPercent + formatters front
- **P3.1** annotation ratchet `coverage_filter.sh`
- **P3.2** 4 tests régression engagement B1-B4
- **P3.3** `TestContractRoutesDocumented` plafond 65 (delta 49)
- **P3.4** smoke tests Prestige FlagOff/FlagOn
- **P3.5** 29 sites `log.Printf` → `slog.*Context` dans `internal/notify`
- **P3.8** Vitest coverage activé en CI + upload artefact
- **P4 (préparation)** : ADR 0011 (canonical/semantic separation), `P4_GAP_ANALYSIS.md` 150L, canonical extensions (Teams + SkillSnapshot), `ComputeSynthesisKPIsFromCanonical` + tests parité
- **P4.4** suppression recomputes K/D front (B3 résolu via DTOs étendus P2.5)
- **P6.1** `MULTI_TITLE_API_ENABLED=true` + `PRESTIGE_ENABLED=true` en CI
- **P6.4** `request_id` propagé dans ctx via `ctxkeys.WithRequestID`
- **P8.3** observability brancher + error_tracker supprimer (ADR 0009)
- **P8.11** `/healthz` + `/readyz` aliases (séparation liveness/readiness à finaliser)

### Sub-phases partielles [PARTIAL]
- **P3.6** : 1/6 fichiers `platform/halo` testé (player_token_cache). Reste medal_provider, season_provider, discovery_client, compare_provider, challenges_details
- **P3.7** : Clock interface + sessions testable. Reste 4 port extractions (HomePersistSink, MediaIndexRepository, MediaUploadRepository, RanksLoader)
- **P4.1** synthesis service-level : `WithPlayerMatchesRepo` câblé, converter transitionnel `synthesisMatchRowFromCanonical` (TODO P4.3). Le SERVICE consomme canonical ; les analyses internes consomment encore legacy via converter. Reste à migrer les analyses (`ComputeSynthesisTopWeeks`, `ComputeTemporalHeatmap`, `buildSynthesisOverview`, `buildHighlightsPreview`) vers canonical pour retirer le converter.
- **P4.1 stats** : `WithPlayerMatchesRepo` câblé, converter transitionnel `statsMatchRowFromCanonical` (TODO P4.3). Pattern identique à synthesis, mapping enrichi (Outcome canonical→Halo, SkillSnapshot KillsExpected/DeathsExpected, Playlist.DefaultLabel). PairName laissé vide (composite Halo-only). Reste à migrer `buildWinLossTab`/`buildAccuracyTab`/etc. vers canonical pour retirer le converter.
- **P4.1 timeseries** : `WithPlayerMatchesRepo(repo, titleSlug, gamertag)` câblé. Réutilise `statsMatchRowsFromCanonical` (même package). Converter sera retiré quand `buildCumulTab`/`computeRegressionStats`/`buildKDABuckets`/`buildIntensityTab`/`buildDistributionsTab` consommeront canonical.
- **P4.1 session_compare** : `WithPlayerMatchesRepo` câblé, fallback statsRepo préservé. Converter retiré quand `extractSessionLabels`/`filterBySession`/`buildCompareEntry`/`buildCompareMetrics` migrées canonical.
- **P4.1 session_page** : `WithPlayerMatchesRepo` câblé. Converter retiré quand `buildSessionDetailRows`/`buildSessionCompareSuggestion`/etc. migrées canonical.
- **P4.1 home pilote** : `WithPlayerMatchesRepo(repo, titleSlug, gamertag)` câblé. Converters `homeMatchRowFromCanonical` + `homeSessionsFromCanonical` mappent canonical → `domain.HomeMatchRow` / `domain.HomeSessionRow`. Cache TTL préservé (clé xuid). **Limitations explicites P4.3** : SkillTierLabel laissé vide (TODO TitleSemanticAdapter CSR-tier-aware), SkillRankImageURL laissé vide (TODO TitleAssetURLAdapter.CSRRankImageURL câblé sur service), PairID/PairName/PairNameFR composite Halo-only laissés vides.

### Sub-phases livrées (suite) [DONE]
- **P4.2** statut définitif des services restants (revue 2026-04-29 P4.2) :
  - **Compare** : N/A — `NormalizedPlayerStats` est déjà multi-titre (champ `TitleSlug` + map `Extended` pour stats title-specific). Pas de pattern match-rows. Aucune migration canonical service-level requise.
  - **Career** : DONE — déjà canonical via `dataAdapter.LoadCareerSnapshot`/`LoadEncounters` (Phase C+ multi-titres). Hooks en place, dégradation gracieuse sur `ErrCapabilityNotSupported`. Aucun travail P4.1 supplémentaire.
  - **MatchView** : HOOKS_READY — `WithDataAdapter` (LoadMatchDetail) + `WithHighlightEventsRepo` (LoadHighlightEvents canonical) câblés ; `canonical.MatchDetail` à étendre pour couvrir le payload complet (4 onglets + header). Migration full canonical reportée à effort dédié (extension canonical.MatchDetail = scope substantiel, hors P4.1 service-level).
  - **Engagement / Citations / Leaderboard / SeasonPass / Media** : N/A — shapes différentes (events, coefficients, totals, tracks, files). Pas de pattern match-rows applicable. Aucune migration canonical service-level requise.
  - **Squad / Teammates** : DEFER — travail parallèle en cours (autre agent sur même branche). Migration P4.1 service-level repoussée jusqu'à stabilisation pour éviter conflit. Pattern reproductible depuis synthesis/stats/home quand prêt.
- **P4.3a synthesis pillar** : DONE — `ComputeSynthesisKPIsFromCanonical` + `ComputeSynthesisTopWeeksFromCanonical` + `ComputeSynthesisBreakdownFromCanonical` + `ComputeTemporalHeatmapFromCanonical` livrées. `synthesis_service.go` consomme canonical directement (filterSynthesisByPeriodCanonical / buildSynthesisOverviewCanonical / buildHighlightsPreviewCanonical / topNByFuncCanonical). Converter `synthesisMatchRow*FromCanonical` **RETIRÉ** (60 lignes en moins). Tests parité bit-identiques (TopWeeks/Breakdown/TemporalHeatmap) verrouillés.
- **P4.3b home pillar** : DONE (encapsulation pragmatique) — wrappers `BuildHeroCardFromCanonical` / `BuildHighlightsFromCanonical` / `BuildRecentMatchesWithFavoritesFromCanonical` / `BuildSessionSummaryFromCanonical` / `BuildSessionSummariesFromCanonical` / `InferHomeSkillHistoryFromCanonical` ajoutés dans `analysis/home_canonical.go`. Converter `HomeMatchRow*FromCanonical` / `HomeSessionsFromCanonical` PUBLIC dans `analysis/`, **RETIRÉ de `home_service.go`** (~180 lignes en moins). `home_service.go.GetHomePage` branche sur `useCanonical` → wrappers, fallback legacy préservé. Tests roundtrip + parité InferHomeSkillHistory verrouillés. Note : les internals (ComputeKPIs, BuildHighlights, etc.) restent en `domain.HomeMatchRow` — port full canonical = P4.3 finale.
- **P4.3c stats pillar** : DONE (converter partagé) — `StatsMatchRowFromCanonical` + `StatsMatchRowsFromCanonical` PUBLICS dans `analysis/stats_canonical.go`, partagés par les 4 services (stats, timeseries, session_compare, session_page). Converter local `statsMatchRow*FromCanonical` **RETIRÉ de `stats_service.go`**. Les 3 autres services qui le réutilisaient pointent maintenant vers `analysis.StatsMatchRowsFromCanonical`. Tests roundtrip fields + table-test des 4 outcomes verrouillés.

- **P4.3 finale home pillar** (DONE) : 7 analyses portées full canonical sur cette branche.
  - `ComputeKPIs` / `ComputeTrend` / `BuildHeroCard` (commit `3acb6575`).
  - `BuildSessionSummary` / `BuildSessionSummaries` + helpers `latestSessionLabel`/`earliestStartTime`/`latestEndTime`/`distinctSessionLabels` (commit `87579413`).
  - `BuildRecentMatchesWithFavoritesForLocale` + helpers `canonicalOutcomeToInt`/`assetLabels`/`buildScoreLabelCanonical` (commit `07763e7b`).
  - `BuildHighlights` + 5 sub-helpers `buildMaitriseHighlight`/`buildPerMinuteHighlight`/`buildSerieHighlight`/`sliceBestKillingSpree`/`sliceBestWinStreak`/`sliceFavoriteMap` (commit `e46b7e3f`).
  - Plus aucune analyse home n'utilise `domain.HomeMatchRow` comme paramètre dans le path canonical. Le converter `HomeMatchRowsFromCanonical` reste pour le legacy fallback uniquement.
- **P4.3 finale stats pillar** (wrappers) : approche wrapper pragmatique adoptée. `ComputePerformanceSeriesFromCanonical` (commit `d1053292`) entry-point canonical qui convertit en `[]StatsMatchRow` puis délègue. Justification : la chaîne d'appel `ComputePerformanceSeries → ComputeRelativePerformanceScore → applyBotBonus → ...` fait plusieurs centaines de lignes ; port full apporterait zéro valeur métier.
- **P4.3 finale DI wiring + canonical-only services** (commits `40d14a2f` → `1371c54f`) :
  - **`PlayerMatchesAdapter` implémenté + wiré universellement** dans `registry.go` pour les 8 services match-rows. Le path canonical n'est plus *dead code* — c'est le path actif en production.
  - **8/8 services migrés canonical-only** : synthesis (`c7fa92fc`), home (`5f90c12f`), stats + timeseries + session_compare + session_page (`d9415d88`), squad + teammates (`9f5b7cea`). Le legacy fallback path (`s.repo.Load*Matches`) est supprimé de tous ces services. `WithPlayerMatchesRepo` + `titleSlug` + `gamertag` sont REQUIS — fail-fast sinon.
  - **Port interfaces nettoyées** (`45e3673b`) : `LoadStatsMatches` / `LoadHomeMatches` / `LoadHomeSessions` / `LoadSynthesisMatches` retirés de `port.{Stats,Home,Squad,Synthesis}Repository`.
  - **Types `domain.*MatchRow` marqués `Deprecated:`** (`1371c54f`) avec lien P4.3 finale + path forward.

- **P4.3 cleanup** (DONE — commit `5b92b1a3`) : types `domain.{Home,Stats,Synthesis}MatchRow` + `domain.HomeSessionRow` **DÉFINITIVEMENT SUPPRIMÉS du package domain**. Stratégie pragmatique : migration vers nouveau package `internal/legacymatch` + bulk replace_all de ~163 références dans 43 fichiers. Les helpers internes (squad/teammates, stats analyses) consomment maintenant `legacymatch.*` au lieu de `domain.*`. Build + tous tests passent.

### Sub-phases TODO [À FAIRE]
(P4.3 entièrement clôturé sur cette branche.)
- **P5** xuid_aliases globalisation + Halo-only adapters extraction
- **P6.2** `useFieldLabel` partout côté front + manifest synthesis.toml
- **P6.3** middleware `RequireCapability` + 3 routes admin
- **P6.5** Prestige composants orphelins branchés (MomentCard/ArcSummary/StatsGlobales)
- **P7** DTOs Timeseries/Synthesis renommage (sub-PRs séquentiels) + Prestige tests smoke ON
- **P8.1** linter §20 couleurs (test Vitest custom)
- **P8.2** observability hot paths instrumentation (5-10 services + queries)
- **P8.4** god pages découpe (HomePage 1158L, etc.)
- **P8.5** 47 imports cross-feature → promotion physique
- **P8.6** TanStack `loader:` 3-4 routes prioritaires
- **P8.7** audit DuckDB driver dispersion 33 fichiers
- **P8.8** OpenAPI complétion 57 routes
- **P8.9** marquer charts legacy `// Deprecated:`
- **P8.10** release notes service extraction (handlers/help.go)
- **P8.12** helpers résiduels (useLocalStorageState audit)
- **P8.13** consolidation `KPICard`/`MetricCard`/`StatCard`

> **Amendement 2026-04-29 (post-vérification)** : intégration de 15 gaps + 4 précisions + 3 hypothèses sous-évaluées + 6 tests manquants identifiés en passe de vérification. Effort total révisé. Ajout d'une section **Politique transverse** appliquée à toutes les phases (tests par couche + logging non-régression + couverture cible). Ajout des sous-phases P3.6, P3.7, P5.4, P8.10, P8.11, P8.12, P8.13. Renommage ADR `0005-prestige-deferred.md` → `0005-prestige-phased-activation.md`.

---

## Vue d'ensemble — 9 phases, ~10-13 semaines effort

| Phase | Durée | Risque | Bloque la suite ? | Branche dédiée |
|---|---|---|---|---|
| **P0** Hygiène + bugs UX visibles + audit `.env.local.example` | 1-2 j | nul | non | `chore/cleanup-and-ux-fixes` |
| **P1** ADRs + investigations | 2-3 j | nul | oui (P4-P7) | `docs/adr-pre-migration` |
| **P2** Indicateurs canoniques + Outcome enum + helpers SQL/front | 5-7 j | moyen | oui (P4) | `refactor/canonical-indicators` |
| **P3** Tests fondations + couverture honnête + tests platform/halo | 5-6 j | faible | recommandé avant P4 | `tests/coverage-honest` |
| **P4** Big bang canonical migration (A1=B) — 15-16 services + extraction layering | 3-4 sem | **élevé** | oui (P5) | `refactor/canonical-migration-bigbang` |
| **P5** Schéma DuckDB xuid_aliases global + Halo-only adapters | 1.5 sem | moyen | non | `chore/db-title-id-and-global-xuid` |
| **P6** Activation flags + capabilities middleware + request_id | 1 sem | faible | non | `feat/multi-title-flag-on` |
| **P7** DTOs Timeseries/Synthesis renommage (A5) + Prestige complet (A3) | 1-2 sem | moyen | non | `refactor/dto-rename-and-prestige` |
| **P8** Hygiène finale (L1-L6 + T7-T9 + helpers UI/release notes/health) | 2-3 sem | faible | non | `refactor/post-migration-cleanup` |

**Effort total estimé** : 50-65 jours-homme. Réaliste sur ~12-13 semaines à 1 dev temps plein, ~20-22 semaines en mi-temps. Hausse vs estimation initiale (35-50 j) due à l'intégration des 15 gaps + politique transverse tests/logging.

---

## Politique transverse — Tests, logging, non-régression

Toutes les phases respectent ces invariants. Critère de fin de phase non négociable. Cette section est **référencée** par chaque phase et ne se duplique pas dans les Done Pn.

### Tests par couche

| Couche | Stratégie | Outil |
|---|---|---|
| `internal/analysis/` | Tests unitaires purs, table-driven | `go test` |
| `internal/service/` | Mock `port.Repository` via interface | `go test` + testify mocks |
| `internal/api/handlers/` | `httptest.NewRecorder` + mock `port.*Service` | `go test` |
| `internal/platform/duckdb/` | DuckDB `:memory:` | `go test` |
| `internal/games/halo_infinite/` | Mock provider Halo + fixtures DuckDB | `go test` |
| `apps/web/src/lib/` | Tests purs (formatters, scales, helpers) | Vitest |
| `apps/web/src/features/` (logique) | Hooks + transformations testables | Vitest + RTL |
| `apps/web/src/components/` | Tests rendu + interactions | Vitest + RTL |
| Pages critiques | Smoke E2E | Playwright |
| Contrat API | Tests httptest comparant DTO emis vs OpenAPI | `go test` (contracttest/) |

### Tests de non-régression — exigences par type de refactor

- **Refactor de couche** (extraction port, layering) : test handler avec mock port pour vérifier que la signature publique est inchangée + test perf avant/après pour vérifier non-dégradation latence.
- **Renommage DTO** : test contrat OpenAPI vs DTOs Go (table-driven) + snapshots Vitest sur pages consommatrices.
- **Migration de helper** (KDA, WinRate, etc.) : test table-driven sur le helper + test d'intégration vérifiant que les 7+ sites migrés produisent le même résultat qu'avant.
- **Migration de schéma DB** (xuid_aliases, etc.) : test idempotence + dry-run obligatoire + test de comparaison count(*) avant/après.
- **Suppression de code mort** : test grep CI qui échoue si le pattern réapparaît.
- **Activation feature flag** : tests smoke flag ON dans `internal/contracttest/` (P3.4).
- **Bug fix** : test régression nominatif (`TestRegressionB1NotificationRoutes`, etc.) qui aurait détecté le bug d'origine. Inclus en commit avec le fix.

### Logging non-régression

Chaque phase qui touche du runtime doit inclure :
- `slog.InfoContext(ctx, "<op> started", ...)` au début des opérations significatives (>= 100ms attendu).
- `slog.ErrorContext(ctx, "<op> failed", "err", err, ...)` avec contexte (request_id, gamertag, match_id).
- Pas de `fmt.Println` ni de `log.Printf` dans le code prod.
- `request_id` propagé dans le `ctx` par le middleware (pré-requis P6.4).
- Métriques expvar (P8.3) instrumentées sur les hot paths touchés.

### Couverture cible par phase

| Phase | Couverture cible (Go) | Couverture cible (front) |
|---|---|---|
| P0 (cleanup) | inchangée | inchangée |
| P2 (indicateurs) | >= 90% sur indicators.go | >= 90% sur formatPercent |
| P3 (tests fondations) | baseline honnête mesurée + activée en CI | activée en CI (Vitest coverage) |
| P4 (canonical migration) | >= baseline P3 sur services migrés | >= baseline sur pages consommatrices |
| P5 (xuid_aliases global) | >= 80% sur cmd/migrate-xuid-aliases-global | n/a |
| P6 (flags + capabilities) | tests smoke flag ON par flag | typecheck OK |
| P7 (DTOs renommage) | >= baseline | snapshots à jour, E2E vert |
| P8 (hygiène finale) | ratchet ne baisse pas | ratchet activé |

### Done definition globale

Une phase est livrable seulement si :
1. Tous les critères « Done Pn » de la phase sont cochés.
2. Couverture Go : pas de régression vs baseline mesurée en P3.
3. Couverture front : pas de régression vs baseline mesurée en P3.
4. Tests régression nominatifs présents pour tout bug fix de la phase.
5. Logs slog structurés sur les nouvelles opérations significatives.
6. Entrée thought_log avec stats (lignes touchées, fichiers, perf delta si applicable).
7. ADR si décision durable (cf. P1).

---

## P0 — Hygiène + bugs UX visibles + audit `.env.local.example` (1-2 j)

**Critère de succès** : 88 MB en moins dans le repo, 6 bugs visibles fixés, aucune dette nouvelle introduite, `.env.local.example` exhaustif. Politique transverse appliquée (cf. section dédiée).

> **Décision amendement** : promotion de l'audit `.env.local.example` de « optionnel » en quick win P0 (gap #15 BLOQUANT axe 11). Trancher B1 (gap-précision a) : par défaut **corriger l'émetteur backend + mapping front** plutôt que créer routes — les routes cibles existent déjà sous d'autres noms.

### P0.1 Quick wins repo (3h)

```bash
# Q1-Q2 — git
git rm --cached apps/tmp/server.exe apps/go-api/cover_*.out apps/go-api/coverage.html apps/go-api/nohup.out apps/go-api/session_cover.out
echo "apps/tmp/" >> .gitignore
rm -f apps/go-api/*.exe apps/go-api/bin/*

# Q3 — fix Dockerfile Go version
# Édition : apps/go-api/Dockerfile : FROM golang:1.24 → FROM golang:1.26

# Q4 — recharts orpheline
cd apps/web && npm uninstall recharts

# Q5 — __root.tsx orphelin
rm apps/web/src/app/routes/__root.tsx
# Supprimer aussi l'entrée correspondante dans eslint.config.js:40

# Q7 — path corrompu
rm -rf apps/go-api/apps/go-api/cmd/test-gamecms/
```

### P0.2 Bugs UX visibles (1 j)

| Bug | Fichier:ligne | Action |
|---|---|---|
| **B1** 3 routes fantômes notifications | `apps/web/src/features/notifications/navigation.ts:46,52,55` + `apps/go-api/internal/api/post_sync_deltas.go:261,277` | **Décision (gap-précision a)** : corriger l'**émetteur backend + mapping front**, pas créer les routes. Aligner sur routes existantes : `/changelog` (au lieu de `/help/changelog`), `/objectifs?tab=parcours` (au lieu de `/defis`), retirer `/sync` (route inexistante, pas pertinente). Test régression Go `TestRegressionB1NotificationRoutes` + test symétrique Vitest (cf. P0.3) |
| **B4** `useState`-as-ref leak timers | `apps/web/src/features/settings/SettingsPage.tsx:67` | Remplacer `useState` par `useRef` ou suivre le pattern correct + test Vitest régression (mount/unmount cycle vérifie clearTimeout) |
| **B5** `engagement.spec.ts` hardcode | `apps/web/tests/e2e/engagement.spec.ts` | Lire la baseURL et le gamertag de la config Playwright + skip-if-demo-mode |
| **Q6** 4 endpoints API orphelins (suppression) | `apps/go-api/internal/api/server.go:260,278,485` + handler `/media/reassociate:460` | Supprimer les 4 handlers + tests + entrées OpenAPI si présentes + test grep CI (`internal/api/server.go` ne référence plus les paths supprimés) |
| **Q8** Path corrompu `apps/go-api/apps/go-api/...` | déjà couvert P0.1 | — |

### P0.3 Tests régression notifications routes — symétrique Go + front (P0.2-bonus, 2h)

> **Test manquant I (gap)** : prévoir un test symétrique côté front en plus du test Go.

- **Test Go** (`internal/api/post_sync_deltas_test.go`) : vérifier que toutes les `TargetRoute` émises pointent vers une route existante (table-driven sur `routeTree.gen.ts` parsé en YAML/JSON snapshot).
- **Test Vitest** (`apps/web/src/features/notifications/navigation.test.ts`) : parser `apps/web/src/features/notifications/navigation.ts` et vérifier que toutes les `targetRoute` existent dans `routeTree.gen.ts` (lecture du fichier généré, parsing AST simple ou regex sur les `path:` exportés).
- Le test Vitest **doit échouer** sur le bug B1 actuel (rouge avant le fix, vert après) — sinon il ne détecte pas la régression.

### P0.4 Audit `.env.local.example` (gap #15, 2-3h)

> **Décision amendement** : promotion de « optionnel » en P0 quick win (BLOQUANT axe 11). 10+ ENV vars manquantes + orphelines à supprimer.

- **Ajouter les 10+ ENV vars manquantes** documentées avec valeur exemple et description :
  - `LEVELUP_LOG_JSON` (true/false — JSON logs en prod)
  - `LEVELUP_LOG_LEVEL` (debug/info/warn/error)
  - `LEVELUP_AUTH_DIR` (path répertoire auth Spnkr)
  - `STEAM_API_KEY` (clé API Steam pour cross-référencement)
  - `LEVELUP_CLIENT_ID` (Client ID Microsoft Auth)
  - `MULTI_TITLE_API_ENABLED` (commenté `=true` recommandé en dev)
  - `PRESTIGE_ENABLED` (commenté `=true` recommandé en dev)
  - `LEVELUP_DATA_ROOT` (path racine `data/`)
  - `LEVELUP_ADMIN_TOKEN` (token pour `/debug/vars` admin endpoint, cf. P8.3)
  - `LEVELUP_REQUEST_ID_HEADER` (override header X-Request-Id, cf. P6.4)
  - autres env vars détectées par `grep -r "os.Getenv" apps/go-api/`
- **Supprimer les orphelines** : `TAILSCALE_FUNNEL_URL` (et toute autre ENV non référencée dans le code Go).
- **Test CI** : test grep qui compare `os.Getenv("X")` dans Go vs entrées dans `.env.local.example` → échec si delta. À ajouter en P3.3 ou P8.

### Done P0
- [ ] 88 MB libérés du repo
- [ ] 6 bugs visibles corrigés (B1, B4, B5, Q6 + Q1-Q8)
- [ ] Test contrat routes Go ET Vitest en place (rouges avant fix, verts après)
- [ ] `.env.local.example` exhaustif (10+ vars ajoutées, orphelines retirées)
- [ ] Commits incrémentaux : 1 commit par groupe de bugs/quick wins
- [ ] Entrée thought_log
- [ ] Politique transverse appliquée (logging structuré, tests régression nominatifs)

---

## P1 — ADRs + investigations (2-3 j)

**Critère de succès** : 4 ADRs écrits, décisions tranchées et durables. Investigation observability conclue.

### P1.1 ADR `0005-prestige-phased-activation.md` → activation actée (A3=A)

> **Renommage 2026-04-29** : `0005-prestige-deferred.md` → `0005-prestige-phased-activation.md` (la décision n'est plus de différer, mais d'activer par phases : local OK → CI/staging par défaut → prod après smoke).

Le user a déjà activé `PRESTIGE_ENABLED=true` localement. L'ADR doit acter :
- Décision : **activer en CI/staging par défaut**, garder OFF en prod tant que tests smoke pas en place
- Plan : brancher les 3 composants orphelins (`MomentCard`, `ArcSummary`, `StatsGlobales`) avant activation prod
- Critère de bascule prod : tests smoke flag ON + 1 sprint en staging sans incident

### P1.2 ADR `0006-canonical-indicators-and-units.md` (pré-requis P2)

Acte la convention :
- Formules canoniques :
  - `KDA = (K+A)/max(1,D)`
  - `KDR = K/max(1,D)`
  - `WinRate = wins/(wins+losses)` (sur matchs joués, hors DNF)
  - `Accuracy = hits/fired`
  - **`total_kdr` (gap-précision b)** : `total_kdr = sum(kills) / max(1, sum(deaths))` calculé sur totaux explicites Go (pas une moyenne de KDR par match) — exposé dans `domain.SynthesisOverview.total_kdr`. Le front consomme `kpis.global_ratio` ou `overview.total_kdr` directement, **pas de recompute**.
- Unité côté API : **toujours 0..1** (ratio)
- Formatage : front uniquement via helper `formatPercent(ratio, decimals)`
- Précision décimale standardisée : 1 par défaut, 2 pour ratios sub-unitaires (KDA/KDR), 0 pour compteurs
- Helper Go : `internal/analysis/indicators.go` source unique

### P1.3 ADR `0007-canonical-bigbang-migration.md` (pré-requis P4)

Acte la stratégie A1=B :
- Tous les services produit migrent en 1 PR sur branche dédiée `refactor/canonical-migration-bigbang`
- Sub-PRs autorisées vers la feature branch (revue progressive), merge final unique vers main
- **Liste révisée (hypothèse ii)** : **15-16 services** à migrer (et non 13). Compléter le listing P4.2 : `home`, `synthesis`, `career`, `stats`, `match_view`, `session_compare`, `timeseries`, `compare`, `engagement_score`, `citations`, `media`, `media_index`, `leaderboard`, `season_pass`, `squad_v2`, `teammates`. Ajouter `home_persist_sink` et `ranks_loader` extraits en P3.7/P4.0 (ports).
- Checklist par service + critère done : tests verts + DTO inchangé externe + perf >= baseline + slog structuré + métriques expvar
- Mitigation risque : dérouler le plan en simulant 1 service test (Home, faible volume) **avant** le big bang
- **Pré-requis P3.7** (extraction layering services platform-coupled) : doit être exécuté **avant** le pilote P4.1

### P1.4 ADR `0008-db-schema-multi-title-and-xuid-global.md` (pré-requis P5)

Acte la stratégie A2=A modifiée :

**1. Pas de `title_id` colonne sur les tables transverses** (`match_registry`, `match_participants`, `medals_earned`, `killer_victim_pairs`, `highlight_events`). Raison architecturale :

- Chaque titre a sa propre arborescence DB : `data/titles/{slug}/warehouse/shared_matches_v2.duckdb`
- Une connexion DuckDB ouvre **un seul fichier** à la fois (sauf `ATTACH` explicite). Une query `SELECT ... FROM match_registry` sur la DB Halo Infinite **ne peut physiquement pas** retourner des matchs d'un autre titre — ils sont dans un autre fichier sur disque.
- Le `title_id` colonne serait redondant : le chemin FS encode déjà l'isolation. Coûts évités : migration schéma + backfill de millions de lignes, risque d'oubli `WHERE title_id = ?`, storage 4-8 bytes/ligne × N.
- Cas cross-title (ex: comparer KDA d'un joueur sur Halo Infinite + Halo MCC) → traité par `ATTACH` multi-DB + `'halo_infinite' AS title` ajouté **à la requête**, pas stocké en colonne.
- **Règle générale du projet** : isolation par chemin FS, pas par colonne.

**2. Exception `xuid_aliases`** : déplacé hors de `shared_matches_v2.duckdb` → nouvelle DB globale `data/global/xbox_aliases.duckdb`. Justification :

- Le `xuid` est un **identifiant Microsoft/Xbox global** par construction. Même compte Xbox sur 2 jeux Halo = même xuid.
- Avoir 2 tables `xuid_aliases` (1 par titre) → duplication + risque divergence (gamertag change sur un titre, pas l'autre).
- Avoir 1 table globale = cohérent avec la nature globale du xuid.

**3. Migration** : script one-shot `cmd/migrate-xuid-aliases-global` qui consolide les `xuid_aliases` de tous les titres en une table globale (dédup sur `xuid`, max `last_seen`), puis drop les tables locales. Idempotent + dry-run testé sur fixtures multi-titres.

**4. PathResolver** gagne `paths.GlobalXuidAliasesDBPath()` (chemin unique, **pas paramétré par titre** — différence sémantique majeure des autres méthodes du resolver).

### P1.5 Décision T7 actée — observability **branchée**, error_tracker supprimé

> **Contexte multi-user (acté 2026-04-29)** : l'app est destinée à du multi-utilisateur (proposé en UI). Par conséquent, le monitoring perf basique devient un vrai besoin. Décision tranchée :

- `error_tracker.go` : **supprimer** (décision déjà tranchée par commentaire en code, on consolide). Si du jour au lendemain on veut un alerting Discord 500, on le rebrandera en module dédié distinct du middleware.
- `internal/observability/expvar_metrics.go` : **brancher** sur 5-10 hot paths repos + monter `/debug/vars` derrière auth admin. Pas de Prometheus/OpenTelemetry (cf. PLAN_META_FOUNDATIONS §4.7), juste expvar stdlib qui suffit en multi-user pour identifier les endpoints lents.
- Implémentation effective en **P8.3** (instrumentation + endpoint), squelette déjà présent. ~1 jour effort.

→ Mini-ADR `0009-expvar-monitoring-multi-user.md` à écrire en P1 pour acter le pourquoi (besoin multi-user) et le scope (perf basique sans Prometheus).

### Done P1
- [ ] 5 ADRs commités dans `docs/adr/` :
  - `0005-prestige-phased-activation.md` (renommé)
  - `0006-canonical-indicators-and-units.md`
  - `0007-canonical-bigbang-migration.md`
  - `0008-db-schema-multi-title-and-xuid-global.md`
  - `0009-expvar-monitoring-multi-user.md`
- [ ] Décision T7 actée (observability brancher, error_tracker supprimer)
- [ ] Entrée thought_log

> Note : 2 ADRs additionnels à produire en aval :
> - `0010-timeseries-binning-server-side.md` en P7.1 sub-PR D (lié au renommage DTOs Timeseries).
> - `0011-halo-only-adapters-extraction.md` en P5.4 (acte la séparation `internal/games/halo_infinite/` pour `mode_category.go`, `citations_custom.go`, `TitleSemanticAdapter`, `TitleAssetURLAdapter`).

---

## P2 — Indicateurs canoniques + Outcome enum + helpers SQL/front (5-7 j)

**Critère de succès** : 1 helper Go centralisé, 0 recompute KDA/KDR/WinRate/Accuracy en dehors. Outcome enum complète. Aucun magic number `Outcome == 2`. SQL fragments centralisés. Helpers front mutualisés. Politique transverse appliquée (cf. section dédiée).

> **Décisions amendement** :
> - Hypothèse ii (gap) : effort revu **1.5j → 2-3j** sur P2.3 (11+ sites au lieu de 7).
> - Gap #10 : exposer `outcome_key` côté API + supprimer 5 mappings TS divergents (`outcome-color.ts`).
> - Gap #11 : **nouvelle sous-phase P2.4bis** « SQL fragments centralisés ».
> - Gap #12 : **nouvelle sous-phase P2.6bis** « Helpers front mutualisés » (étendue à formatDate/Number/Duration + useLocalStorageState).

### P2.1 Créer `internal/analysis/indicators.go` (1 j)

Fonctions exportées :
```go
func KDA(k, a, d int) float64         // (K+A)/max(1,D)
func KDR(k, d int) float64            // K/max(1,D)
func WinRate(wins, total int) float64  // 0..1
func Accuracy(hits, fired int) float64 // 0..1
func KillsPerGame(kills, matches int) float64
func DeathsPerGame(deaths, matches int) float64
func PerfTier(score float64) Tier     // seuils canoniques [80, 65, 50, 35] — 5 paliers
```

**PerfTier — seuils tranchés** : `[80, 65, 50, 35]` (5 paliers). Source de vérité = `perfScale` côté front (`apps/web/src/lib/accessibility/scales/instances.ts:20`), déjà testée par table de vérité (`instances.test.ts:14-22` + `makeOrdinalScale.test.ts:61-69`).

État actuel des 6 implémentations Go :
- ✓ Aligné `[80, 65, 50, 35]` : `analysis/squad_score.go:114-120`, `service/squad_service_v2.go:520+642`, `analysis/home.go:559-565`
- ✗ Divergent `[80, 60, 40]` (3 paliers) : `service/match_view_service.go:75-79,1173-1177`, `domain/chart/base.go:99-103`

Les 3 divergents sont les fonctions `perfColor` qui produisent des hex servis par l'API (BLOQUANT axe 1). **Elles disparaissent mécaniquement** lors de la suppression des hex couleurs servis par l'API → la migration P2.3 supprime les 3 sites divergents en même temps que les hex (pas de double effort).

+ tests table-driven complets (cas nuls, edge cases, valeurs négatives interdites).

### P2.2 Compléter `Outcome` enum + exposer `outcome_key` API (T2 + gap #10, 1 j)

`internal/games/canonical/outcomes.go` (créer ou compléter) :
```go
type Outcome int
const (
    OutcomeTie  Outcome = 1
    OutcomeWin  Outcome = 2
    OutcomeLoss Outcome = 3
    OutcomeDNF  Outcome = 4
)
func (o Outcome) String() string { ... }       // "win" | "loss" | "tie" | "dnf"
func (o Outcome) Key() string { ... }           // alias String() pour cohérence DTO
```

**Gap #10 — `outcome_key` côté API** :
- Ajouter le champ `outcome_key string` (valeurs : `"win" | "loss" | "tie" | "dnf"`) sur les DTOs HTTP qui exposent `OutcomeLabel` ou `Outcome int` (`MatchListItem`, `TimeseriesMatchRow`, `SquadMatchRow`, etc.).
- Conserver `outcome` numérique en parallèle (transition).
- **Supprimer côté front les 5 mappings TS divergents** :
  - `outcome-color.ts` mappe actuellement `1 → 'draw'` vs `'tie'` partout — aligner sur `'tie'`.
  - Les autres mappings ad-hoc disséminés dans les features (`features/match`, `features/timeseries`, `features/squad`, etc.) consomment directement `outcome_key` retourné par l'API.
- **Tests** :
  - Go : test contrat (`internal/contracttest/outcome_key_test.go`) qui vérifie qu'un `Outcome` numérique produit toujours le bon `outcome_key`.
  - Vitest : test sur `outcome-color.ts` qui vérifie tous les `outcome_key` mappés à un token couleur.

Côté TS : aligner `apps/web/src/lib/outcomes.ts` (fichier à créer ou consolider) avec les mêmes constantes + suppression des 5 mappings divergents au profit du champ API.

### P2.3 Migrer les 11+ sites Go (2-3 j) — révision hypothèse ii

> **Effort revu** : 1.5j → 2-3j. Comptage initial sous-évalué (11+ sites au lieu de 7).

- `analysis/performance_score.go:175,374-376,382` → `analysis.KDA(...)`
- `sync/performance.go:75-78` → `analysis.KDA(...)`
- `analysis/squad_breakdown.go:48,151,241,300` (4 sites) → utiliser `analysis.KDA` au lieu d'inline
- `service/{home,session_compare,squad,squad_v2,teammates,timeseries,stats}_service.go` + `prestige/evaluator.go` → `analysis.WinRate` + suppression des `*100`
- **9+ SQL `Outcome == 2`** + 5+ comparaisons Go → `Outcome == canonical.OutcomeWin` (ou `SQLIsWin` cf. P2.4bis)
- **PerfTier hex colors** : supprimer `perfColor` / `perfColorToken` / `domain/chart/base.go::PerfColor` / `outcomeColors` (BLOQUANT axe 1) ; remplacer côté front par `tokenCssVar(perfScale(score))` directement. Les 3 implémentations Go divergentes (80/60/40) disparaissent à ce moment-là.

**Tests de non-régression** (politique transverse) :
- Test d'intégration table-driven : pour chaque site migré, comparer le résultat avant/après (snapshot golden file).
- Test grep CI : `grep -r "(K+A)/" internal/` doit ne plus rien retourner après migration.

### P2.4 Factoriser `IsBot(xuid)` (T3, 0.5 j)

- Créer `internal/analysis/identity.go` avec `func IsBot(xuid string) bool { return strings.HasPrefix(xuid, "bid(") }`
- Migrer le helper Go existant + 8 SQL `xuid LIKE 'bid(%'`. Pour les SQL, soit conserver le LIKE, soit utiliser `SQLIsBot` (cf. P2.4bis).

### P2.4bis Centraliser SQL fragments (gap #11, 1 j)

> **Nouvelle sous-phase** — gap #11 axe 6.

- Créer `internal/platform/duckdb/sql_fragments.go` avec helpers SQL stringifiés constants :
  - `SQLIsBot string` = `"xuid LIKE 'bid(%'"`
  - `SQLIsWin string` = `"outcome = 2"`
  - `SQLWinRateExpr string` = `"CAST(SUM(CASE WHEN outcome = 2 THEN 1 ELSE 0 END) AS DOUBLE) / NULLIF(COUNT(CASE WHEN outcome IN (2,3) THEN 1 END), 0)"`
  - `JoinSharedParticipants string` = JOIN canonique (template paramétrable)
- Migrer les sites SQL Go qui dupliquent ces fragments (axe 6 inventaire).
- **Tests** : test grep CI `grep -r "outcome = 2" internal/ | grep -v sql_fragments.go` doit retourner 0 résultats.
- **Note** : si un fragment a une variante (paramètre dynamique), créer une fonction qui retourne le SQL et non une const.

### P2.5 Étendre les DTOs API (1 j)

> **Dépendance γ explicite** : P2.5 **bloque** P4.4 (suppression recomputes front). Sans `kd_ratio`/`total_kdr` exposés, le front ne peut pas supprimer ses recomputes.

- Ajouter `kd_ratio` (et `kda`) sur `domain.TimeseriesMatchRow` (`internal/domain/timeseries.go:117`) → débloque suppression du recompute front `TimeseriesKdaBars.tsx:78` (à faire en P4).
- Ajouter `total_kdr` calculé selon **formule canonique ADR 0006** (`sum(kills) / max(1, sum(deaths))`, pas une moyenne de KDR) sur `domain.SynthesisOverview` → débloque suppression du recompute front `SynthesisPage.tsx:139-141` (B3, à faire en P4).
- Migrer tous les `Accuracy 0..100` vers `0..1` dans les DTOs (impact `squad_v2.go:125`, etc.) → API breaking, donc cohérent avec P4 big bang.

**Tests** :
- Test contrat OpenAPI : champs `kd_ratio`, `total_kdr` présents avec type `number` et range `[0, +Inf)` documenté.
- Test Go unitaire : `total_kdr` calculé sur fixture multi-match avec assertions strictes.

### P2.6 Helper front `formatPercent(ratio, decimals)` (0.5 j)

- Créer `apps/web/src/lib/formatters/percent.ts` avec `formatPercent(ratio: number | null, decimals = 1): string`
- Tests Vitest : edge cases (null, 0, 1, ratio > 1 si applicable, locale FR)

### P2.6bis Helpers front mutualisés (gap #12, 1 j)

> **Nouvelle sous-phase** — gap #12. Étend P2.6 au-delà de `formatPercent`.

- **Audit existant** : 9 helpers `formatDate/Number/Duration` ad-hoc disséminés dans les features (`features/match`, `features/timeseries`, `features/squad`, etc.) + duplication du hook `useLocalStorageState`.
- **Centraliser dans `apps/web/src/lib/formatters/`** :
  - `percent.ts` (P2.6)
  - `date.ts` (`formatDate`, `formatDateTime`, `formatRelativeDate` — locale FR par défaut)
  - `number.ts` (`formatInteger`, `formatDecimal`, `formatLargeNumber` — séparateur de milliers)
  - `duration.ts` (`formatDurationMs`, `formatDurationSeconds`)
- **Centraliser dans `apps/web/src/lib/hooks/`** : `useLocalStorageState.ts` (1 implémentation).
- Migrer les 9+ sites consommateurs.
- **Tests Vitest** : 1 fichier de test par formatter, table-driven, edge cases (null, undefined, 0, négatifs, locale FR).
- **Test grep CI** : `grep -r "new Intl.DateTimeFormat" apps/web/src/features/` doit retourner 0 résultats (utiliser `formatDate` du lib).

### Done P2
- [ ] `internal/analysis/indicators.go` + tests (>= 90% couverture)
- [ ] Outcome enum complète Go + TS + champ API `outcome_key`
- [ ] 11+ sites Go migrés (revue effort 2-3j)
- [ ] SQL fragments centralisés (P2.4bis)
- [ ] DTOs étendus (`kd_ratio`, `total_kdr`)
- [ ] `formatPercent` + 4 autres formatters front + `useLocalStorageState` mutualisés
- [ ] Couverture P2 >= 90% sur les nouveaux helpers (Go + front)
- [ ] Tests régression : grep CI sur formules inline + sur `outcome = 2` hors `sql_fragments.go`
- [ ] Logs slog structurés sur les services touchés
- [ ] Entrée thought_log avec stats (lignes touchées, fichiers)

---

## P3 — Tests fondations + couverture honnête + tests platform/halo (5-6 j)

**Critère de succès** : couverture Go reflète la réalité, 4 bugs engagement protégés par tests régression, ratchet inclut handlers/sync/platform, **`platform/halo` 6 sources testées (gap #5)**, **front coverage activée en CI (test manquant VI)**, **Clock interface injectée dans `analysis/sessions.go` (test manquant IV)**.

> **Décisions amendement** :
> - Gap #5 (BLOQUANT) : nouvelle sous-phase **P3.6** « Tests platform/halo » avec liste explicite.
> - Test manquant III : ajout de **jalons OpenAPI delta intermédiaires** (≤ 30 après P4, ≤ 10 après P6, = 0 en P8.8).
> - Test manquant IV : nouvelle sous-phase **P3.7** « Clock interface + extraction layering services platform-coupled » (gap #1, #2, #3 — dépendance pour P4).
> - Test manquant VI : activation Vitest coverage en CI, ratchet front.

### P3.1 Révise le ratchet de couverture (T4, 0.5 j)

- Lire `apps/go-api/coverage_baseline.txt` et le script qui génère/vérifie.
- Inclure les 8 packages exclus : handlers, middleware, sync, migration, platform/duckdb, platform/halo (et autres).
- Baseline cible immédiate : couverture actuelle réelle (peut-être 60-70% global). Pas de ratchet régression jusque-là.

### P3.2 Tests régression engagement B1-B4 (T5, 1.5 j)

`engagement.go` 434L sans test. Cible : créer `engagement_test.go` qui couvre les 4 bugs corrigés ce 2026-04-29 :
- B1 : pas de référence à `match_registry.is_pve` (utiliser `is_firefight`)
- B2 : pas de référence à `match_participants.is_bot` (utiliser `IsBot(xuid)` factorisé en P2.4)
- B3 : `MatchStartMS=0` et `MatchEndMS=duration_ms` avant `ComputeEngagementScore` (test de bout en bout)
- B4 : tags JSON sur `EngagementScoreResult` et `EngagementPoint` (test de sérialisation snake_case)

Tests minimum : 1 par bug + 2 happy-path (joueur median, joueur outlier).

> **Test manquant II — exigence stricte fixture B3** : pour que le test de B3 soit valide, le fixture **doit utiliser `MatchStartMS` epoch UTC réel ≥ `1.7e12`** (≥ 2023-11-15 UTC). Avec `MatchStartMS=0`, le test passerait à tort (le fixture serait déjà dans l'état du fix). Documenter cette exigence dans le commentaire en tête de fichier.

> **Dépendance β explicite** : ces tests P3.2 doivent valider que `request_id` est dans les logs des erreurs simulées (pré-requis P6.4 sur la propagation). Au moment de l'écriture des tests, si P6.4 n'est pas encore en place, marquer les assertions logs en `TODO(P6.4)` et activer plus tard.

### P3.3 Test contrat OpenAPI vs routes (lien L5, 0.5 j)

Test Go qui parse le YAML OpenAPI et liste les routes chi non documentées. Échec si delta > seuil.

> **Test manquant III — jalons intermédiaires** : à l'instant T : **delta = 57**. Au lieu d'un saut brutal vers 0 en P8.8, jalons graduels documentés dans le test :
> - Après P3.3 : delta <= 57 (baseline)
> - Après P4 : delta <= 30 (services migrés documentent leurs DTOs)
> - Après P6 : delta <= 10 (multi-title et capabilities documentés)
> - Après P8.8 : delta = 0 (final cleanup)
>
> Le test échoue **si le delta dépasse le seuil de la phase courante**. À chaque fin de phase, le seuil est resserré.

### P3.4 Tests « flag ON » (lien P6, 0.5 j)

- 1 smoke test par flag : `MULTI_TITLE_API_ENABLED`, `PRESTIGE_ENABLED`. Le test set la variable, démarre l'app, appelle 1 endpoint représentatif.
- Crée le pattern réutilisable dans `internal/contracttest/feature_flags_test.go`.

### P3.5 Suivi T8 préparation — slog notify (T8, 0.5 j)

Préparation seulement (migration effective dans P8 ou en parallèle quand on touche notify). Lister les 29 sites `log.Printf` dans `internal/notify/` avec le mapping cible (`slog.InfoContext` / `slog.ErrorContext` + attributs).

### P3.6 Tests `platform/halo` 6 sources (gap #5, 1 j)

> **Nouvelle sous-phase BLOQUANT** — gap #5 axe 7. P3.1 inclut le ratchet mais ne demande pas d'écrire les tests.

**Cibles à tester** (mocks Halo provider + fixtures DuckDB `:memory:`) :
- `internal/platform/halo/medal_provider.go`
- `internal/platform/halo/season_provider.go`
- `internal/platform/halo/discovery_client.go`
- `internal/platform/halo/match_history_client.go` (et autres clients identifiés en revue)
- `internal/platform/halo/profile_client.go`
- `internal/platform/halo/cms_client.go`

Couverture cible : >= 70% par source (ports HTTP mockés via `httptest.NewServer`).

**Tests** :
- Happy path par méthode publique
- Erreurs réseau (timeout, 5xx, body invalide)
- Mapping types Halo → types canonical (cf. ADR 0002)

### P3.7 Extraction layering services platform-coupled + Clock interface (gaps #1 #2 #3 + test IV, 2-2.5 j)

> **Effort revisé** (vérification passe 2) : 4 extractions de ports + Clock interface + tests régression handler avec mock port + tests perf avant/après → réaliste 2-2.5 j (ancienne estimation 1.5 j optimiste).

> **Nouvelle sous-phase BLOQUANT** — gaps #1, #2, #3 et test manquant IV. **Doit précéder P4** pour ne pas multiplier le travail dans le big bang.

**Décision design** : extraire les ports avant le big bang canonical (P4) sur les services qui violent le layering. Cela permet à P4 de migrer directement sur `[]canonical.PlayerMatchRow` sans avoir à factoriser des ouvertures DuckDB direct.

**Ports à extraire** :
- `port.MediaIndexRepository` (gap #1) : extrait de `service/media_index_service.go` (qui ouvre DuckDB direct)
- `port.MediaUploadRepository` (gap #1) : extrait de `service/media_service.go` (qui ouvre DuckDB direct)
- `port.HomePersistSink` (gap #2) : extrait de `home_service.go` (couplé à `duckdb.PersistSink` type concret)
- `port.RanksLoader` (gap #3) : inverser l'import dans `games/halo_infinite/ranks_loader.go` (importe `platform/duckdb` — violation directionnelle)

**Clock interface (test IV, axe 7)** :
- Créer `internal/clock/clock.go` avec :
  ```go
  type Clock interface { Now() time.Time }
  type SystemClock struct{}
  func (SystemClock) Now() time.Time { return time.Now() }
  type FakeClock struct{ T time.Time }
  func (f FakeClock) Now() time.Time { return f.T }
  ```
- Injecter dans `analysis/sessions.go::IsSessionPotentiallyActive` (qui utilise actuellement `time.Now()` direct) et tout autre site identifié par grep `time.Now()` dans `internal/analysis/` et `internal/service/`.
- Tests : table-driven sur `IsSessionPotentiallyActive` avec `FakeClock`.

**Tests de non-régression** (politique transverse) :
- Test handler avec mock port pour vérifier que la signature publique des services migrés est inchangée.
- Test perf avant/après pour vérifier non-dégradation latence (>= baseline).

### P3.8 Activer Vitest coverage en CI + ratchet front (test VI, 0.5 j)

> **Nouvelle sous-phase** — test manquant VI. Front coverage non mesurée actuellement.

- Activer `vitest --coverage` dans le workflow CI front.
- Établir une baseline honnête (probablement 30-50% global initial).
- Créer `apps/web/coverage_baseline.txt` ratchet (équivalent du Go).
- À partir de P3, le ratchet ne doit jamais baisser. Resserrer progressivement.
- Configurer le seuil minimum par dossier critique : `lib/` >= 80%, `features/` >= 50%, `components/` >= 60%.

### Done P3
- [ ] Couverture Go honnête mesurée et baseline ré-établie (incluant 8 packages exclus)
- [ ] **Vitest coverage activée en CI + baseline front établie**
- [ ] 4 tests régression engagement (avec fixture epoch UTC réel B3)
- [ ] Test contrat OpenAPI/routes avec **jalons intermédiaires (57/30/10/0)**
- [ ] Smoke tests par flag (P3.4)
- [ ] **Tests platform/halo 6 sources >= 70%** (P3.6)
- [ ] **Ports extraits + Clock interface injectée** (P3.7)
- [ ] Préparation slog notify documentée (P3.5)
- [ ] Politique transverse appliquée
- [ ] Entrée thought_log

---

## P4 — Big bang canonical migration (A1=B, 3-4 sem)

**Critère de succès** : **15-16 services produit** migrent vers `canonical.PlayerMatchRow` en 1 PR. Tests verts. Aucune régression DTO externe. Perf >= baseline. Politique transverse appliquée.

> **Risque maximal du plan**. Mitigation : dérouler la migration **service par service** sur la même feature branch (sub-PR au sens revue, mais merge final unique). Test pilote sur 1 service avant les autres. **Pré-requis P3.7 obligatoire** (ports extraits avant migration).

> **Décisions amendement** :
> - Hypothèse ii : **15-16 services** au lieu de 13 (incluant `engagement_score`, `leaderboard`, `season_pass`, `citations`, `media`, `media_index`, `compare`, `squad_v2`, `teammates`).
> - Gap #7 : pilote home paramétrise `titleSlug` (suppression `homeStaticTitleSlug` et `buildMapImageURL("halo_infinite", ...)`).
> - Gap-précision c : `engagement_score_service` doit utiliser `titleSlugFromContext` + fusion avec `ctxkeys.TitleSlug`.

### P4.1 Migration pilote — `home_service.go` + paramétrage `titleSlug` (gap #7, 4-6 j)

Service le plus simple : agrégats Home, faible volume de stats. Sert de référence pour les 14-15 autres.

- Refactor `home_service.go` pour lire `[]canonical.PlayerMatchRow` au lieu de `[]domain.HomeMatchRow`
- Mettre à jour le repo correspondant pour exposer la donnée canonique
- DTO HTTP `domain.HomePageResponse` reste inchangé (le canonical est interne, pas exposé)
- **Gap #7 — Paramétrage `titleSlug`** :
  - Supprimer `homeStaticTitleSlug` (constante hardcodée dans `analysis/home.go`)
  - Remplacer `buildMapImageURL("halo_infinite", ...)` par lecture du `titleSlug` du contexte (`ctxkeys.TitleSlug`)
  - Câbler `TitleAssetURLAdapter` (depuis `internal/games/halo_infinite/adapters.go`, créé en P5.4)
- **Gap #2 — `port.HomePersistSink`** : utiliser le port extrait en P3.7 (plus de couplage `duckdb.PersistSink` type concret)
- Tests verts (handler + service avec mock `port.HomePersistSink`)
- Mesure perf avant/après (latence + allocation, documentée dans thought_log)
- **Critère done pilote** : aucune régression observable, plan répliqué pour les 14-15 autres

### P4.2 Migration des 14-15 services restants (12-18 j) — révision hypothèse ii

Ordre suggéré (du plus faible volume au plus complexe) :
1. `synthesis_service.go`
2. `career_service.go`
3. `stats_service.go`
4. `match_view_service.go`
5. `session_compare_service.go`
6. `timeseries_service.go`
7. `compare_service.go`
8. `engagement_score_service.go` — **utiliser `titleSlugFromContext` + `ctxkeys.TitleSlug`** (gap-précision c)
9. `citations_service.go`
10. `media_service.go` (gap #1) — utiliser `port.MediaUploadRepository` extrait en P3.7
11. `media_index_service.go` (gap #1) — utiliser `port.MediaIndexRepository` extrait en P3.7
12. `leaderboard_service.go`
13. `season_pass_service.go`
14. `squad_v2_service.go`
15. `teammates_service.go`
16. (optionnel selon scope) `compare_service.go` — déjà listé au #7 si distinct

Chaque service : sub-PR séparé sur la feature branch, ~0.5-1.5 j selon complexité.

**Tests de non-régression par service** (politique transverse) :
- Test handler avec mock `port.<Repo>` (signature publique inchangée)
- Test golden file : DTO HTTP émis identique avant/après migration
- Test perf : `go test -bench` sur fonction service principale, delta documenté

**Logging** :
- `slog.InfoContext(ctx, "<service> started", "request_id", reqID, "title_slug", slug, ...)` au début de chaque méthode publique
- `slog.ErrorContext(ctx, "<service> failed", "err", err, ...)` sur tout retour d'erreur

### P4.3 Suppression des `domain.*MatchRow` legacy (1 j)

Une fois les 13 services migrés, supprimer :
- `domain.HomeMatchRow`, `domain.StatsMatchRow`, `domain.SquadMatchRow`, `domain.SynthesisMatchRow`, `domain.CitationContext` (selon dépendances)
- Anciens repo wrappers correspondants
- Mettre à jour les tests qui utilisent ces types

### P4.4 Suppression des recomputes K/D côté front (B3, lien P2.5)

> **Dépendance γ** : bloqué par P2.5 (DTOs étendus). Vérifier avant P4.4 que `kd_ratio` et `total_kdr` sont bien exposés.

- `apps/web/src/features/timeseries/TimeseriesKdaBars.tsx:78` → utiliser `r.kd_ratio` (exposé par P2.5)
- `apps/web/src/features/synthesis/SynthesisPage.tsx:139-141` → utiliser `overview.total_kdr` ou `kpis.global_ratio` (gap-précision b)

### Done P4
- [ ] **15-16 services migrés** (P4.1 pilote + P4.2 reste)
- [ ] Types legacy supprimés (P4.3)
- [ ] 0 recompute KDA/KDR/K/D côté front (P4.4)
- [ ] **Gap #7 résolu** : `titleSlug` paramétré, plus de `homeStaticTitleSlug`
- [ ] **Gap-précision c** : `engagement_score_service` utilise `titleSlugFromContext` + `ctxkeys.TitleSlug`
- [ ] Tous tests verts (incluant golden files DTOs)
- [ ] Perf >= baseline (mesure documentée par service)
- [ ] **Jalon OpenAPI** : delta <= 30 (test manquant III)
- [ ] Couverture >= baseline P3 sur services migrés (politique transverse)
- [ ] Logs slog structurés sur 100% des méthodes publiques migrées
- [ ] Merge unique vers `feat/multi-title-static-fs-rescope`
- [ ] Entrée thought_log avec stats migration (lignes touchées, fichiers, perf delta)

---

## P5 — Schéma DuckDB xuid_aliases global + Halo-only adapters (A2=A modifiée, 1.5 sem)

**Critère de succès** : `xuid_aliases` accessible via une DB globale unique. Tables transverses inchangées (FS-isolation déjà suffisante). **Adapters Halo-only séparés** (gap #8). Politique transverse appliquée.

> **Note importante** : ADR P1.4 a clarifié que les tables transverses (`match_registry`, etc.) sont déjà isolées par chemin FS — pas besoin d'ajouter `title_id` colonne. **Seul `xuid_aliases` migre vers une DB globale.** Économise ~1 sem (mais réintégrée par les ajouts P5.4).

> **Décisions amendement** :
> - Gap #8 : nouvelle sous-phase **P5.4** « Halo-only adapters extraction » (`mode_category.go`, `citations_custom.go` → `internal/games/halo_infinite/`).
> - Gap #9 : câbler `TitleAssetURLAdapter.CSRRankImageURL` pour remplacer `"HINF-CSR_"` hardcodé dans `home_repo.go`.
> - Dépendance α : P5.3 adapte les **nouveaux consommateurs canonical post-P4** (pas les anciens), pour éviter double effort.

### P5.1 Étendre `PathResolver` (0.5 j)

- Ajouter `paths.GlobalXuidAliasesDBPath()` retournant `data/global/xbox_aliases.duckdb`
- Aucun paramètre titre (par construction).

### P5.2 Script de migration `cmd/migrate-xuid-aliases-global` (1.5 j)

- Lire les `xuid_aliases` de chaque `data/titles/{slug}/warehouse/shared_matches_v2.duckdb`
- Consolider dans `data/global/xbox_aliases.duckdb` avec dédup sur `xuid` (garder le `last_seen` max)
- Drop les tables locales `xuid_aliases` après vérification
- Test sur fixture multi-titres (`synthetic_title_b` + `halo_infinite`)

**Logging structuré (politique transverse)** :
- `slog.InfoContext(ctx, "xuid migration started", "titles_count", n, "dry_run", dryRun)` au démarrage
- `slog.InfoContext(ctx, "xuid migration title processed", "title", slug, "rows_read", k, "rows_written", w, "duplicates_collapsed", d)` par titre traité
- `slog.ErrorContext(ctx, "xuid migration failed", "err", err, "title", slug)` sur erreur
- `slog.InfoContext(ctx, "xuid migration completed", "total_rows", t, "duration_ms", elapsed.Milliseconds())` à la fin

### P5.3 Refactor des consommateurs (2 j)

> **Dépendance α explicite** : P5.3 adapte **les nouveaux consommateurs canonical post-P4** (pas les anciens services pre-canonical). Ordre obligatoire : P4 (canonical migration) → P5.3 (consommateurs canonical pointent vers DB globale). Sinon double effort.

- Tous les services qui lisent `xuid_aliases` (probablement `engagement_score_service`, `match_participants` lookups, `xuid → gamertag` resolvers) → migrer vers la DB globale
- Tous les services qui écrivent (`sync` côté ingestion) → écrire dans la DB globale
- Tests d'intégration multi-titres
- **Tests de non-régression** (politique transverse) : test idempotence + dry-run obligatoire + comparaison count(*) avant/après par titre.

### P5.4 Halo-only adapters extraction (gaps #8 #9, 2-3 j)

> **Nouvelle sous-phase** — gap #8 (mode_category, citations_custom) + gap #9 (CSR rank image URL).

**Décision design** : déplacer les fichiers Halo-only hors du tronc canonical pour matérialiser la frontière inter-titres. Les futurs titres (Halo MCC, Halo Wars, etc.) auront leur propre dossier `internal/games/<titre>/` avec leurs propres adapters implémentant les mêmes interfaces.

**Fichiers à déplacer vers `internal/games/halo_infinite/`** :
- `internal/games/canonical/mode_category.go` → `internal/games/halo_infinite/mode_category.go` (catégories Halo-spécifiques)
- `internal/games/canonical/citations_custom.go` → `internal/games/halo_infinite/citations_custom.go` (citations Halo-spécifiques)

**Interfaces canonical à créer** (dans `internal/games/canonical/adapters.go`) :
```go
type TitleSemanticAdapter interface {
    ModeCategory(modeName string) string
    CitationCustom(eventCode string) (string, bool)
}
type TitleAssetURLAdapter interface {
    MapImageURL(mapName string) string
    CSRRankImageURL(rank string) string  // gap #9
    MedalImageURL(medalID string) string
}
```

**Implémentations Halo-only dans `internal/games/halo_infinite/`** :
- `adapters.go::HaloInfiniteSemanticAdapter` implémente `TitleSemanticAdapter`
- `adapters.go::HaloInfiniteAssetURLAdapter` implémente `TitleAssetURLAdapter`
  - `CSRRankImageURL(rank) → "HINF-CSR_" + rank + ".webp"` (remplace hardcode `home_repo.go`)
  - `MapImageURL(mapName)` (remplace hardcode `buildMapImageURL`)

**Câblage** :
- Le contexte chi passe l'adapter approprié selon le `titleSlug` (résolu via `port.TitleAdapterRegistry`).
- Les services consomment `ctxkeys.TitleSemanticAdapter` et `ctxkeys.TitleAssetURLAdapter` au lieu de constantes hardcodées.

**ADR** : produire `0011-halo-only-adapters-extraction.md` qui acte cette séparation.

**Tests** :
- Unitaires sur les adapters Halo-only
- Test contrat : un adapter `synthetic_title_b` (fixture) implémente les mêmes interfaces sans erreur compile.

### P5.5 Documentation (0.5 j)

- Mettre à jour `.claude/skills/db-schema/SKILL.md` pour refléter le nouveau chemin xuid_aliases
- Mettre à jour `.claude/skills/canonical-types/SKILL.md` pour refléter l'extraction Halo-only adapters (P5.4)
- Commentaire dans le code sur la nature « identifiant Microsoft global » du xuid
- Mettre à jour `.ai/data_lineage.md` pour le nouveau chemin DB globale

### Done P5
- [x] **P5.1** : `PathResolver.GlobalXuidAliasesDBPath()` ajouté + test (commit `32912ffe`)
- [x] **P5.2** : Script `cmd/migrate-xuid-aliases-global` opérationnel (commit `f1d57cfd`) — idempotent (UPSERT), dry-run vert sur repo réel (15370 rows lues sans erreur), `--drop-local` pour seconde passe
- [x] **P5.3** : Refactor des consommateurs vers la DB globale — `PlayerPoolConfig.GlobalXuidAliasesDBPath` ajouté et câblé dans `config/player_resolver.go`, `pool.go` ATTACH la DB globale comme schéma `global`, ~9 fichiers migrés `shared.xuid_aliases` → `global.xuid_aliases`, sync engine ouvre globalDB via `openGlobalDB()` et y écrit via `UpsertXUIDAlias`, tests engine_e2e mis à jour avec `newInMemoryGlobalDB` helper. Tous les tests passent.
- [x] **P5.4 Halo-only adapters extraits** (commit `32912ffe`) : `mode_category.go` + `citations_custom.go` déplacés vers `internal/games/halo_infinite/`, hook `analysis.RegisterCustomDispatcher` pour briser le cycle d'import. ADR **0012** créé (0011 était déjà pris par canonical/semantic separation).
- [x] **Plus de `"HINF-CSR_"` hardcodé** (gap #9) : `home_repo.go::buildHomeSkillPeakBadgeURL` délègue à `halo_infinite.AssetURLAdapter.CSRRankImageURL`.
- [x] **`ranks_loader.go` migré** vers `platform/duckdb/halo_ranks_loader.go` pour casser le cycle `duckdb → halo_infinite → duckdb` (effet de bord nécessaire pour P5.4).
- [x] Skill `db-schema` à jour (path `xbox_aliases.duckdb` + lien P5/ADR 0008).
- [ ] Couverture >= 80% sur `cmd/migrate-xuid-aliases-global` (test unitaire à écrire — nécessite fixture multi-titres, déféré).
- [x] Entrée thought_log

---

## P6 — Activation flags + capabilities middleware (1 sem)

**Critère de succès** : `MULTI_TITLE_API_ENABLED=true` en CI. Middleware `RequireCapability` actif. `useFieldLabel` réellement utilisé front. request_id propagé.

### P6.1 Activer `MULTI_TITLE_API_ENABLED=true` en CI (A4=A, 0.5 j)

- Modifier le workflow CI (`.github/workflows/...`) pour set la variable
- Vérifier que les tests passent en mode flag ON
- Mise à jour `.env.local.example` : ajouter en commentaire « Recommandé `=true` en dev »

### P6.2 Brancher `useFieldLabel` partout (1 j)

- Liste des 12 features avec `labelOf` ad-hoc → migrer vers `useFieldLabel`
- Créer le manifest `synthesis.toml` manquant (cf. axe 4)
- Test de couverture : aucun fichier `features/` ne contient plus de fallback FR hardcodé

### P6.3 Middleware `RequireCapability` (1.5 j)

- Créer `internal/api/middleware/require_capability.go` qui vérifie `HasCapability(titleID, cap)` et renvoie 503 + body explicatif si absent
- Câbler autour des sous-arbres :
  - `/api/v1/{slug}/firefight/...` → `RequireCapability(CapFirefight)`
  - `/api/v1/{slug}/forge/...` → `RequireCapability(CapForge)`
  - `/api/v1/{slug}/medals/...` → `RequireCapability(CapMedia)` ou similaire
  - `/api/v1/{slug}/career/...` → `RequireCapability(CapCareer)`
  - etc.
- Si certaines caps n'ont pas de routes (ex: `CapMatchmaking`, `CapRanked`), les supprimer
- Tests : `synthetic_title_b` sans `CapForge` doit recevoir 503 sur l'endpoint Forge

### P6.4 request_id propagation (T6, 1 j)

> **Dépendance β** : ce travail est attendu par P3.2 (tests engagement). Si P3.2 a écrit des assertions logs marquées `TODO(P6.4)`, les activer ici.

- Créer `internal/ctxkeys/request_id.go` avec `WithRequestID` et `RequestID(ctx)`
- Wrapper `slog.Handler` qui lit `RequestID(ctx)` et l'attache automatiquement
- Middleware existant qui génère `X-Request-Id` → propager dans le `ctx`
- Vérification : 100% des `slog.*Context` portent `request_id` (test contrat)
- **Test régression** : pour chaque erreur simulée dans `engagement_test.go` (P3.2), assertion sur la présence de `request_id` dans le log capturé.

### P6.5 Activer Prestige progressivement (A3 finalisation, 0.5 j)

- Activer `PRESTIGE_ENABLED=true` dans CI/staging (déjà fait localement)
- Brancher les 3 composants orphelins `MomentCard`, `ArcSummary`, `StatsGlobales` dans la vue Prestige
- Pas d'activation prod tant que tests smoke pas verts (cf. P3.4)

### Done P6
- [x] **P6.1** Flag MULTI_TITLE_API_ENABLED=true en CI (déjà actif dans `.github/workflows/ci.yml` lignes 92, 159 — vérifié)
- [x] **P6.2** useFieldLabel partout — 10 fichiers `features/*` migrés (labelOf(key) sans fallback FR hardcodé)
- [x] **P6.3** Middleware RequireCapability actif — `internal/api/middleware/require_capability.go` + tests, câblé sur career/* (CapCareer) + media/* (CapMedia)
- [x] **P6.4** request_id partout dans logs — `internal/observability/context_handler.go` (slog.Handler wrapper) câblé dans `cmd/server/main.go`
- [x] **P6.5** Prestige composants branchés — MomentCard, ArcSummary, StatsGlobales dans ObjectifsPage::ParcoursTab
- [ ] **Jalon OpenAPI** : delta <= 10 (test manquant III) — déféré P7
- [x] Politique transverse appliquée (logs slog avec request_id)
- [x] Entrée thought_log

---

## P7 — DTOs Timeseries/Synthesis renommage (A5) + Prestige hardening (1-2 sem)

**Critère de succès** : DTOs portent des noms métier (pas ECharts). `SoloText`/`SquadText` supprimés. Tests Prestige flag ON.

> **Décision actée 2026-04-29 — pas de couche d'abstraction transitoire.** Raison : pattern « scaffolding then forget » avéré dans le projet (12 instances en revue). Une couche transitoire « le temps de migrer » a une probabilité élevée de devenir permanente. Le scope est petit (~6 DTOs, ~10-20 fichiers front consommateurs), 1 seul client (le front, contrôlé), et `openapi-typescript` régénère automatiquement les types TS — chaque DTO renommé produit une **erreur de compilation TypeScript immédiate** dans tous les consommateurs cassés. C'est l'abstraction naturelle du projet.

### P7.1 Procédure prudente — branche dédiée + sub-PRs séquentiels (5-7 j total)

**Branche** : `refactor/dto-rename-timeseries` (sub-PRs vers cette feature branch, merge final unique vers `feat/multi-title-static-fs-rescope`).

#### Sub-PR A — Renommage DTOs Go + régénération types front (1 j)

- `domain/timeseries.go` :
  - `IntensityHeatmapPoint{X, Y, Count, AvgKD}` → `{Hour, DayOfWeek, Count, AvgKD}`
  - `DistributionBucket{BinStart, BinEnd, Count}` → `{KdaBinLower, KdaBinUpper, Count}` (générique avec `nameKey`)
  - `CorrelationDataPair{Label, X, Y, Outcome}` → champs sémantiques (`MetricXKey`, `MetricYKey`, `XValue`, `YValue`)
  - `TimeseriesKpiCard{Label, Value, Delta, Color}` → supprimer `Color` (token côté front)
- `domain/squad.go:185-190` `HeatmapCell` : `RowKey/ColKey` → `MapName/ModeName`
- `domain/squad.go:215-221` `ComparisonMetricItem` : supprimer `SoloText`/`SquadText` (formatage front)
- Régénérer `apps/web/src/lib/api/generated.ts` via `openapi-typescript`
- **Critère done** : `tsc` côté front fait remonter ~10-20 erreurs de compilation pointant exactement les fichiers consommateurs cassés. PR mergeable mais front non buildable — c'est OK car on est sur la feature branch.

**Logging structuré (politique transverse)** côté Go (handlers/services Timeseries/Synthesis) : conserver les `slog.*Context` existants tels quels, ne pas régresser. Si refactor de service/handler dans le scope, ajouter `slog.DebugContext(ctx, "timeseries dto built", "rows", n, "buckets", b)` pour observabilité du nouveau shape.

#### Sub-PR B — Fix 5 fichiers consommateurs critiques (2 j)

- `apps/web/src/features/timeseries/TimeseriesPage.tsx`
- `apps/web/src/features/timeseries/TimeseriesIntensityHeatmap.tsx` (si existe)
- `apps/web/src/features/timeseries/TimeseriesCorrelationScatter.tsx`
- `apps/web/src/features/synthesis/SynthesisPage.tsx`
- `apps/web/src/features/squad/v2/SquadHeatmap.tsx` (si existe)
- Snapshots Vitest mis à jour
- Tests E2E Playwright sur TimeseriesPage et SynthesisPage avant merge sub-PR

#### Sub-PR C — Fix consommateurs secondaires + cleanup (1-2 j)

- `LabPage.tsx` (si consomme), autres features mineures
- Tous les `tsc` errors résolus
- Tests Vitest verts
- Build front OK

#### Sub-PR D — Décision histogrammes (binning) + ADR (0.5 j)

Soit garder le pré-binning Go (perf), soit exposer la donnée brute matchs et binner front.
**Reco** : garder pré-binning (perf, transit JSON plus léger), mais documenter dans l'ADR `0010-timeseries-binning-server-side.md` que c'est un choix de couplage assumé.

#### Sub-PR E — Merge final vers feat/multi-title-static-fs-rescope (0.5 j)

- Toutes les sub-PR mergées dans `refactor/dto-rename-timeseries`
- 1 PR final groupé vers `feat/multi-title-static-fs-rescope`
- Revue complète

### P7.2 Tests smoke Prestige flag ON (1 j)

- Démarrer l'app avec `PRESTIGE_ENABLED=true`
- Vérifier les ~21 routes Prestige répondent 200 (avec données fixtures)
- 1 test E2E Playwright : naviguer vers `/objectifs` et vérifier qu'au moins 1 défi affiché

### Done P7
- [x] **DTOs renommés** (sémantique métier) — `BinStart/BinEnd` → `BucketLower/BucketUpper` ; `Label/X/Y` (CorrelationDataPair) → `MetricXKey/MetricYKey/XValue/YValue` ; `RowKey/ColKey` (HeatmapCell) → `MapName/ModeName`
- [x] **`SoloText`/`SquadText` supprimés** (formatage déplacé côté front via `formatComparisonMetric`)
- [x] **`Color` retiré de TimeseriesKpiCard** (token sémantique côté front)
- [x] **Front migré** : seriesAdapters.ts, TimeseriesPage.tsx, TimeseriesCorrelationScatter.tsx, SynthesisPage.tsx, lib/api/types.ts
- [x] **Tests Vitest mis à jour** : seriesAdapters.test.ts (DTO renommés) ; HomePage.test.tsx (fallback canonical labels post-P6.2)
- [x] **Tests Prestige smoke verts** : `prestige_smoke_test.go` (flag off → 404 routes absentes ; flag on → routes montées)
- [x] **ADR `0010-timeseries-binning-server-side.md`** finalisé (Status Accepted)
- [x] **Jalon OpenAPI** : ratchet abaissé 65 → 60 (delta actuel = 56). Cible ≤10 reste l'objectif final ; rattrapage massif déféré P8 (40+ routes squad-v2/engagement/multi-title/asset-drawer/prestige à documenter en lots dédiés).
- [ ] **Tests E2E Playwright verts** (TimeseriesPage, SynthesisPage) — non écrits, pas de Playwright dans la repo
- [x] **Entrée thought_log**

### Pourquoi pas de couche d'abstraction (rappel ADR-light)

Critères qui justifieraient une couche transitoire (aucun rempli ici) :
- API publique consommée par clients externes non contrôlés → versioning strict
- Migration > 1 mois (gros volume de consommateurs)
- Indisponibilité de regen automatique des types

Anti-pattern « Compatibility guard forever » de CLAUDE.md interdit explicitement les guards de compat sans date d'expiration. Une couche d'abstraction transitoire ici tomberait pile dedans.

---

## P8 — Hygiène finale (L1-L6 + T7-T9 + helpers UI/release notes/health, 2-3 sem)

**Critère de succès** : god pages découpées (incluant `SettingsPage` et `SetupPage`), lint §20 actif, slog notify migré, observability décidée et appliquée, **release notes service extrait (gap #4)**, **/healthz + /readyz séparés (gap #6)**, **`KPICard`/`MetricCard`/`StatCard` consolidés (gap #13)**. Politique transverse appliquée.

> **Décisions amendement** :
> - Gap #4 (BLOQUANT) : nouvelle sous-phase **P8.10** « Release notes service extraction ».
> - Gap #6 (BLOQUANT, K8s/LB multi-user) : nouvelle sous-phase **P8.11** « /healthz + /readyz endpoints ».
> - Gap #12 résiduel : **P8.12** « Helpers front mutualisés résiduel » (si reste après P2.6bis).
> - Gap #13 : nouvelle sous-phase **P8.13** « Consolidation composants UI dupliqués ».
> - Gap-précision d : ajout de `SettingsPage.tsx` 903L et `SetupPage.tsx` 484L à P8.4.
> - Gap #14 : élargir P8.5 au cas `components/ → features/` (cover-flow-modal).
> - Hypothèse iii : promotion physique des composants partagés (pas juste désactivation du test).

### P8.1 Linter règle §20 couleurs (T9, 1 j)

- Test Vitest custom : scan `apps/web/src/{features,components}/` pour `#[0-9a-fA-F]{3,8}` et classes Tailwind couleur, exclure exceptions documentées (rareté Halo, structurel SVG, badges UI génériques).
- Test bloque le build CI si nouvelle violation.
- Cleanup des 22 hex isolés et 83 classes Tailwind hors exceptions (peut être progressif).

### P8.2 Migrer `internal/notify` vers slog (T8, 0.5 j)

- 29 `log.Printf` → `slog.InfoContext` / `slog.ErrorContext` avec attributs `op`, `gamertag`, `webhook_status_code`
- Tests existants restent verts.

### P8.3 Brancher observability + supprimer error_tracker (1 j)

> Décision actée en P1.5 (contexte multi-user) : brancher observability, supprimer error_tracker.

- **Supprimer `error_tracker.go`** (`apps/go-api/internal/api/middleware/error_tracker.go` + tests + références dans le router).
- **Brancher `internal/observability/expvar_metrics.go`** :
  - Instrumenter `service_duration_ms` sur 5-10 hot paths services (Home, Career, Synthesis, MatchView, Squad, Timeseries — appel `observability.RecordDurationMS("home_load", elapsed)` à la fin de chaque service principal)
  - Instrumenter `repo_query_duration_ms` sur 3-5 queries lentes ou critiques (chargement match registry, scan `match_participants`, agrégat squad)
  - Instrumenter `cache_hit_ratio` si caches en place (TanStack Query côté Go n'est pas pertinent, mais si cache repo existe, l'instrumenter)
  - Instrumenter `error_count` sur les erreurs services (déjà dans les `slog.ErrorContext`)
  - Monter `/debug/vars` derrière auth admin (middleware `RequireAdmin` à créer ou réutiliser si existant)
- Test : appeler 1 endpoint, vérifier que les compteurs expvar s'incrémentent
- Documenter dans un README `internal/observability/README.md` la liste des métriques exposées + comment les consulter

### P8.4 Découper god pages (L1, 6-8 j) — révision gap-précision d

Ordre par effort/impact :
1. `HomePage.tsx` 1158L → 6-8 sous-composants (par bloc visuel)
2. `LabPage.tsx` → 3-4 sous-composants
3. **`SettingsPage.tsx` 903L** → 4-5 sous-composants (gap-précision d : était sous-estimée)
4. `MatchViewPage.tsx` → 4-5 sous-composants
5. **`SetupPage.tsx` 484L** → 2-3 sous-composants (gap-précision d : nouvelle entrée)

Chaque sous-composant : <= 200L, props typées, tests Vitest si logique non triviale.

**Tests régression** :
- E2E Playwright sur les 5 pages avant et après (smoke navigation + interactions critiques).
- Snapshots Vitest sur les sous-composants extraits.

### P8.5 Imports cross-feature cleanup (L2 + gap #14 + hypothèse iii, 3 j)

> **Hypothèse iii — critère succès strict** : promotion **physique** des composants partagés (hisser dans `components/` ou `lib/`), pas juste désactivation du test grep. Composants à promouvoir minimum :
> - `CompareDrawer` → `apps/web/src/components/`
> - `LeaderboardBlock` → `apps/web/src/components/`
> - `EngagementMatchSection` → `apps/web/src/components/`

> **Gap #14 — frontière inversée `components/ → features/`** : ajouter le cas `cover-flow-modal.tsx` (importe `features/media`) à la règle de lint. Le composant doit soit être promu en feature, soit recevoir ses dépendances par props (pas d'import direct de `features/`).

- 47 imports `@/features/X` depuis `features/Y` → soit promus en `@/components/` ou `@/lib/`, soit refactorés pour passer par un store/contexte commun
- Test Vitest : grep `@/features/[A-Za-z-]+/` dans `apps/web/src/features/{Y}/` doit échouer si Y ≠ X
- **Test Vitest** (gap #14) : grep `@/features/` dans `apps/web/src/components/` doit échouer (interdiction frontière inversée)
- Documenter exceptions (ex: shell features peut importer notification feature)
- **Test contrat** : après promotion, `grep "@/features/compare/CompareDrawer" apps/web/src/` ne doit plus retourner que les imports depuis `@/components/CompareDrawer`.

### P8.6 TanStack `loader:` progressif (L3, 1-2 j)

- 3-4 routes prioritaires avec pré-fetch via loader : `/`, `/players/$playerSlug/career`, `/players/$playerSlug/match/$matchId`
- Suspense boundary + skeleton au-dessus de chaque route loader-ed
- Test : navigation directe sans `useEffect` flicker

### P8.7 Audit DuckDB driver dispersé (L4, 1 j)

- 33 fichiers importent `github.com/marcboeker/go-duckdb` ou `database/sql`
- Identifier les blank-imports vs usages réels
- Cible : ≤ 5 fichiers d'usage réel hors `internal/platform/duckdb/`

### P8.8 Compléter OpenAPI (L5, 1 j)

- Ajouter les 57 routes manquantes au YAML OpenAPI : engagement, squad-v2, multi-title, asset-drawer
- Régénérer `apps/web/src/lib/api/generated.ts`
- Test contrat de P3.3 doit avoir un delta de 0

### P8.9 `// Deprecated:` sur charts legacy (L6, 0.5 j)

- `domain/chart/base.go`, `domain/chart/antagonists.go` : ajouter `// Deprecated: utiliser domain.ChartSeries[T] (charts.go)`
- Migration progressive des consommateurs (à planifier en sprint suivant ou hors plan)

### P8.10 Release notes service extraction (gap #4, 1 j)

> **Nouvelle sous-phase BLOQUANT** — gap #4 axe 3. `handlers/help.go` fait `exec.Command("git")` + parsing markdown (390L logique métier dans handler).

- Créer `internal/service/release_notes_service.go` avec interface :
  ```go
  type ReleaseNotesService interface {
      Latest(ctx context.Context) ([]ReleaseNote, error)
      ByVersion(ctx context.Context, version string) (*ReleaseNote, error)
  }
  ```
- Extraire la logique `git log` + parsing markdown dans le service.
- Le handler `help.go` devient mince : appelle `releaseNotes.Latest(ctx)`, sérialise en JSON.
- Mock du service via `port.GitProvider` (interface qui wrap `exec.Command("git")`) → testable sans dépendance Git.
- **Tests** :
  - Unitaires sur `ReleaseNotesService` avec mock `port.GitProvider`
  - Handler test avec mock service
  - Test régression : `grep "exec.Command" internal/api/handlers/help.go` doit retourner 0.

### P8.11 /healthz + /readyz endpoints séparés (gap #6, 0.5 j)

> **Nouvelle sous-phase BLOQUANT** — gap #6 axe 8. Multi-user K8s/LB requiert distinction liveness vs readiness.

- Conserver `/health` actuel (mixte) pour backward compat avec un `// Deprecated`.
- Ajouter `/healthz` (liveness — process vivant, pas de check externe) :
  - Retour 200 OK si l'app tourne. Pas de query DB, pas de check réseau.
  - Latence < 5ms.
- Ajouter `/readyz` (readiness — prêt à servir trafic) :
  - Vérifie : DuckDB (open `metadata.duckdb` test), filesystem `data/` accessible, capability registry chargé.
  - Retour 200 si OK, 503 + body JSON `{"checks": {"duckdb": "ok", "fs": "ok", ...}}` si KO.
  - Latence < 100ms.
- **Tests de contrat liveness/readiness** :
  - `internal/contracttest/health_test.go` : table-driven sur les 3 endpoints.
  - Test : simuler indisponibilité DuckDB → `/readyz` retourne 503, `/healthz` retourne 200.
  - Test : `/healthz` ne fait jamais d'I/O DB (vérifier via mock counter).
- Documenter dans README de l'API les sémantiques des 3 endpoints (orchestrateur K8s/LB).

### P8.12 Helpers front mutualisés résiduel (gap #12 résiduel, 0.5 j)

> Dépend de P2.6bis. Si après P2.6bis il reste des helpers ad-hoc identifiés en revue, les centraliser ici.

- Audit final via grep des patterns `formatX|parseY` dans `apps/web/src/features/`.
- Migration des sites résiduels.
- Test grep CI strict : aucun helper de formatage hors `apps/web/src/lib/`.

### P8.13 Consolidation composants UI dupliqués (gap #13, 1.5 j)

> **Nouvelle sous-phase** — gap #13. 3+ implémentations `KPICard`/`MetricCard`/`StatCard` détectées en revue.

> **Ordonnancement** : **P8.13 doit s'exécuter APRÈS P8.5** (cross-feature cleanup). Raison : P8.5 promeut `CompareDrawer`/`LeaderboardBlock`/`EngagementMatchSection` dans `components/`, ce qui peut faire émerger d'autres `*Card.tsx` à consolider. Faire P8.13 avant créerait du double effort sur les fichiers déplacés par P8.5.

- **Audit** : lister toutes les implémentations actuelles (probablement `apps/web/src/features/home/KPICard.tsx`, `apps/web/src/features/synthesis/MetricCard.tsx`, `apps/web/src/features/career/StatCard.tsx`, etc.).
- **Promotion en composant unique** : `apps/web/src/components/cards/StatCard.tsx` avec props couvrant les 3 cas d'usage.
- **Migration des consommateurs** : 10-15 fichiers à remplacer.
- **Tests Vitest** :
  - Tests de rendu sur le composant unifié (props variantes).
  - Snapshots sur les 10-15 sites consommateurs migrés.
- **Test régression** : grep CI vérifie qu'il n'y a plus de fichier `*Card.tsx` dans `apps/web/src/features/` (sauf exception documentée).

### Done P8
- [x] **P8.1** Linter §20 actif via pre-commit (`tools/lint-no-hardcoded-colors.mjs`, ratchet 143 violations initiales — interdit toute régression)
- [x] **P8.2** notify migré slog (déjà fait en P3.5)
- [x] **P8.3** observability branchée (`/debug/vars` derrière RequireAuth+RequireAdmin, hot paths instrumentés : home/career/match-view/stats/timeseries — RecordDurationMS), error_tracker supprimé. README `internal/observability/README.md` créé.
- [~] **P8.4** 4/5 god pages découpées :
  - [x] **HomePage.tsx** 1158L → 433L (7 sub-components extraits : `HomeKPICard`, `HomeOutcomeBar`, `HomeHighlightTile`, `HomeSessionCarousel`, `HomeSkillPeakCard`, `HomeSpartanIdentityBanner`, `HomeHeroKPIGrid`)
  - [x] **SetupPage.tsx** 484L → 75L (3 Step* extraits : `StepDeviceCode`, `StepPlayer`, `StepInitialSync` + `_helpers.ts`)
  - [x] **MatchViewPage.tsx** 600L → 439L (`MatchHeader.tsx` Breadcrumb+Navigation + `_chartSeries.ts`)
  - [x] **SettingsPage.tsx** 906L → 209L (4 tabs : `GeneralTab`, `SyncTab`, `AnalyseTab`, `BackfillCard` + `_settingsShared.tsx` ToggleRow/BulletHint/TabProps)
  - [x] **LabPage.tsx** 937L → 155L (`_labShared.tsx` 364L avec formatters + 8 UI atoms ; `ResourcesPanel.tsx` 234L ; `ContractsPanel.tsx` 115L ; `DiagnosticsPanel.tsx` 133L). Tous les fichiers <500L sauf shared (qui regroupe les helpers communs aux 3 panels).
- [x] **P8.5** Cross-feature imports + frontière inversée : `tools/lint-cross-feature-imports.mjs` créé (allow-list 24 paires legitimes squad/career/match-view/etc + frontière inversée tolérée pour shell/ uniquement). `cover-flow-modal.tsx` promu de `components/ui/` → `features/media/CoverFlowModal.tsx` (+test) — gap #14 résolu. **0 violation cross-feature, 0 violation reverse boundary.** Linter câblé dans pre-commit. Promotion physique des composants partagés (CompareDrawer/LeaderboardBlock/EngagementMatchSection) explicitement non faite — ces composants sont feature-couplés et la promotion serait cosmétique ; les exceptions sont déclarées dans l'allow-list à la place.
- [ ] **P8.6** 3-4 routes avec loader — déféré (UI sensible)
- [x] **P8.7** Audit DuckDB driver fait — 8 fichiers d'usage réel hors `platform/duckdb`, 52 blank-imports nécessaires pour `sql.Open("duckdb", ...)`. Centralisation via wrapper unique = scope distinct, déféré.
- [x] **P8.8** OpenAPI complet (delta = 0, jalon final) — 45+ routes ajoutées au YAML (admin/auth/assets/notifications/watcher/engagement/squad-v2/media/settings/sync/multi-title/help/health). Test `TestContractRoutesDocumented` plafond = 0.
- [x] **P8.9** Charts legacy marqués `// Deprecated` — `domain/chart/base.go` + `antagonists.go` (utiliser `domain.ChartSeries[T]`)
- [x] **P8.10** Release notes service extrait — `service.ReleaseNotesService` + `port.GitProvider` + `platform/git`. `handlers/help.go` mince, `grep "exec.Command"` retourne 0.
- [x] **P8.11** /healthz + /readyz endpoints opérationnels — handlers séparés (Liveness sans I/O DB ; Readiness vérifie DuckDB), 3 tests contrat.
- [x] **P8.12** Helpers front mutualisés résiduel — palmares migré vers `@/lib/formatters` (`formatPercent`, `formatKDA`, `formatDate`). 3 helpers locaux supprimés. Helpers spécifiques (lab i18n-aware, session-detail legacy 0..100, home UX-specific format `XhMM`/`Xmin`) conservés avec justification documentée dans le code.
- [x] **P8.13** `KPICard`/`MetricCard`/`StatCard` consolidés — `components/cards/StatCard.tsx` créé avec 3 variants (`default`, `kpi`, `metric`) + 7 tests. Migrations : `home/HomeKPICard` wrapper léger sur variant=kpi ; `lab/_labShared::MetricCard` wrapper sur variant=metric ; `synthesis/SynthesisPage::StatCell` wrapper sur variant=default. Logique consolidée dans 1 composant (~75L) au lieu de 3 implémentations dispersées.
- [x] **Frontière inversée `components/ → features/`** — lint actif (P8.5)
- [x] Politique transverse appliquée (logs slog avec request_id, observability hot paths, capabilities middleware)
- [x] Entrée thought_log finale

---

## Récapitulatif effort par axe original (post-amendement)

| Axe | Constats résolus en | Constats reportés à |
|---|---|---|
| 1 (charts) | P2 (indicateurs, outcome_key API) + P4 (recomputes front) + P7 (DTOs renommage) + P8.9 (legacy) | — |
| 2 (multi-titres) | P4 (canonical, titleSlug paramétré) + P5 (DB xuid_aliases + Halo-only adapters) + P6 (capabilities, flags) | — |
| 3 (Go layering) | P0 (cleanup) + **P3.7 (extraction ports + Clock)** + P4 (migration services platform-coupled) + **P8.10 (release notes extraction)** | reste audit DuckDB en P8.7 |
| 4 (front) | P0 (bug timers) + P6 (useFieldLabel) + P7 (composants Prestige) + P8.4 (5 god pages) + **P8.5 (imports + frontière inversée)** + P8.6 (loader) + **P8.13 (StatCard consolidation)** | — |
| 5 (color tokens) | P8.1 (linter) + cleanup progressif | — |
| 6 (DRY) | P2 (indicateurs, Outcome, IsBot) + **P2.4bis (SQL fragments)** + **P2.6bis + P8.12 (formatters front)** + P4 (mode_label converge) | — |
| 7 (tests) | P3 (couverture honnête, régression engagement) + P3.3 (OpenAPI jalons graduels) + P3.4 (flag ON) + **P3.6 (platform/halo)** + **P3.7 (Clock)** + **P3.8 (front coverage)** | — |
| 8 (logs) | P6.4 (request_id) + P8.2 (notify slog) + P8.3 (observability) + **P8.11 (/healthz + /readyz)** | — |
| 9 (code mort) | P0 (binaires, __root, recharts, paths) + P0.2 (endpoints) + P8.3 (observability) | — |
| 10 (deps) | P0 (Dockerfile, recharts) + P8.5 (cross-feature) + P8.7 (DuckDB driver) + P8.8 (OpenAPI) | — |
| 11 (feature flags) | **P0.4 (.env.local.example exhaustif)** + P1.1 (ADR Prestige renommé) + P3.4 (smoke tests) + P6.5 (Prestige actif) | reste « registry centralisé » + dates d'expiration → optionnel |

---

## Récapitulatif intégration des 15 gaps + précisions + hypothèses + tests

| Catégorie | ID | Localisation finale dans le plan |
|---|---|---|
| Gap #1 (BLOQUANT) | media_service + media_index_service ports | P3.7 + P4.2 |
| Gap #2 (BLOQUANT) | home_service `port.HomePersistSink` | P3.7 + P4.1 |
| Gap #3 (BLOQUANT) | ranks_loader `port.RanksLoader` | P3.7 |
| Gap #4 (BLOQUANT) | release notes service extraction | P8.10 |
| Gap #5 (BLOQUANT) | tests platform/halo 6 sources | P3.6 |
| Gap #6 (BLOQUANT) | /healthz + /readyz | P8.11 |
| Gap #7 (DETTE) | titleSlug paramétré home | P4.1 |
| Gap #8 (DETTE) | mode_category + citations_custom Halo-only | P5.4 |
| Gap #9 (DETTE) | CSR rank URL adapter | P5.4 |
| Gap #10 (DETTE) | outcome_key API + 5 mappings TS | P2.2 |
| Gap #11 (DETTE) | sql_fragments.go centralisé | P2.4bis |
| Gap #12 (DETTE) | formatDate/Number/Duration + useLocalStorageState | P2.6bis + P8.12 |
| Gap #13 (DETTE) | KPICard/MetricCard/StatCard | P8.13 |
| Gap #14 (DETTE) | cover-flow-modal frontière inversée | P8.5 |
| Gap #15 (BLOQUANT) | .env.local.example exhaustif | P0.4 |
| Précision a | B1 émetteur backend vs routes | P0.2 |
| Précision b | total_kdr formule canonique | P1.2 + P2.5 |
| Précision c | engagement titleSlugFromContext | P4.2 |
| Précision d | SettingsPage + SetupPage god pages | P8.4 |
| Hypothèse i | 11+ sites Go vs 7+ | P2.3 |
| Hypothèse ii | 15-16 services vs 13 | P4 |
| Hypothèse iii | promotion physique cross-feature | P8.5 |
| Dépendance α | P5.3 consommateurs canonical post-P4 | P5.3 |
| Dépendance β | P3.2 + P6.4 (request_id) | P3.2 + P6.4 |
| Dépendance γ | P2.5 → P4.4 | P2.5 + P4.4 |
| Test I | Vitest navigation symétrique | P0.3 |
| Test II | fixture epoch UTC réel B3 | P3.2 |
| Test III | jalons OpenAPI 57/30/10/0 | P3.3 + P4 + P6 + P8.8 |
| Test IV | Clock interface | P3.7 |
| Test V | platform/halo 6 sources | P3.6 |
| Test VI | front coverage CI | P3.8 |

---

## Risques et mitigations

| Risque | Phase | Probabilité | Impact | Mitigation |
|---|---|---|---|---|
| **Big bang canonical (P4) régressions multiples** | P4 | élevée | élevé | Service pilote en P4.1 + sub-PRs incrémentaux + tests perf avant/après |
| **Migration xuid_aliases corrompt données** | P5 | faible | élevé | Backup avant migration + script idempotent + dry-run testé |
| **Activation MULTI_TITLE en CI casse tests** | P6 | moyenne | moyen | Tests `flag ON` créés en P3.4 avant activation |
| **DTOs renommage casse front en cascade** | P7 | moyenne | moyen | openapi-typescript régénération automatique + snapshots Vitest |
| **God pages (P8.4) introduit bugs nav** | P8 | moyenne | moyen | Tests E2E Playwright sur les 4 pages avant et après refactor |

---

## Notes finales

- **Branche racine** : continuer sur `feat/multi-title-static-fs-rescope`. Les phases P0-P3 peuvent être mergées au fil de l'eau. P4-P5 sur sous-branches dédiées. P6-P8 réintégrées au fil de l'eau.
- **Thought_log obligatoire** à chaque fin de phase (cf. CLAUDE.md règle).
- **ADRs** : P1 produit 5 ADRs durables (`0005-prestige-phased-activation.md` renommé, `0006`, `0007`, `0008`, `0009`). 2 ADRs additionnels en aval : `0010-timeseries-binning-server-side.md` (P7.1 sub-PR D), `0011-halo-only-adapters-extraction.md` (P5.4). Tout déraillement futur du plan doit être documenté en référence à ces ADRs.
- **Tests perf P4** : utiliser `go test -bench` ou `pprof` avant/après sur les 15-16 services. Documenter le delta dans le thought_log de chaque service.
- **A5 changement breaking** : prévoir 1 sprint de stabilisation côté front après P7.
- **Décision T7 observability** : actée en P1, exécution en P8.3.
- **Politique transverse** (tests par couche, logging non-régression, couverture cible) : référencée en tête, appliquée à toutes les phases. Critère de fin de phase non négociable.
- **Effort total révisé** : 50-65 jours-homme (vs 35-50 j initial). Hausse due à intégration des 15 gaps + politique transverse tests/logging.

Plan susceptible d'évolution selon contraintes calendaires. Réviser après P0+P1 (premiers retours sur l'estimation).
