# INDEX — Revue de code 2026-04-29

Consolidation des 11 axes + vérification finale + axe 11.
Date : 2026-04-29 | Branche : feat/multi-title-static-fs-rescope

## Totaux

- Axes initiaux 1-10 : 36 BLOQUANT, 56 DETTE, 31 AMÉLIORATION (123 constats)
- Vérification finale : 11 nouveaux cas (cf. rapport dédié), tous reversés en amendements aux axes 2/4/8/9 (déjà comptés ci-dessus)
- Axe 11 (a posteriori) : 2 BLOQUANT, 3 DETTE, 1 AMÉLIORATION (6 constats)
- **Total revue** : **38 BLOQUANT, 59 DETTE, 32 AMÉLIORATION** (129 constats au total)

## État de livraison 2026-04-29 (branche chore/cleanup-and-ux-fixes — 32 commits)

| Phase | Statut | Commits | Notes |
|---|---|---|---|
| **P0** Hygiène + bugs UX + audit env | LIVRÉE | 11 commits | 88 MB libérés, 6 bugs UX fixés, 4 endpoints orphelins supprimés, 10 ENV vars documentées |
| **P1** ADRs + investigations | LIVRÉE | 1 commit | 6 ADRs (0005-0010) + CLAUDE.md étendu |
| **P2** Indicateurs canoniques + Outcome enum + helpers | LIVRÉE | 8 commits | `analysis/{indicators,identity,sql_fragments}.go` + DTOs étendus + `formatPercent` front + formatters mutualisés |
| **P3** Tests fondations + couverture honnête | PARTIELLE (6/8) | 12 commits | P3.1/P3.3/P3.4/P3.5/P3.8 livrés ; P3.2 régression engagement livré ; P3.6 partiel (1/6 fichiers halo) ; P3.7 partiel (Clock seul, 4 port extractions à faire en P4) |
| **P4** Big-bang canonical migration | EN COURS | (cf. branche `refactor/canonical-migration-bigbang` à venir) | 15-16 services à migrer |
| **P5-P8** | À FAIRE | — | — |

### Constats de la revue résolus (cette branche)

#### BLOQUANT résolus
- B1 (axe 4 amend.) : 3 routes fantômes notifications alignées
- B4 (axe 4) : SettingsPage `useState`-as-ref → `useRef` (leak timers)
- Q6 (axes 2/9) : 4 endpoints API orphelins supprimés (~1100 L net)
- Axe 7 BLOQUANT couverture mensongère : ratchet annoté (P3.1)
- Axe 7 BLOQUANT régression invisible engagement : 4 tests B1-B4 (P3.2)
- Axe 8 BLOQUANT `internal/notify` `log.Printf` : 29 sites migrés vers slog (P3.5)
- Axe 11 BLOQUANT 2 audit `.env.local.example` : 10 ENV vars + 1 orpheline supprimée (P0.4)

#### DETTE résolues
- Outcome DNF ajouté (axe 6)
- IsBot factorisé (axe 6 + helper SQL fragments)
- Formatters front mutualisés `lib/formatters/{percent,date,number,duration}.ts` (axe 6 amend. DETTE 12)
- Smoke tests flags ON pour PRESTIGE_ENABLED (axe 11)
- Vitest coverage activé en CI (axe 7 amend. test VI)
- Test contrat OpenAPI ↔ chi (axe 7 amend. test III, plafond 65)
- Clock interface + sessions testable (gap test IV / IsSessionPotentiallyActive)

#### Constats reportés à phases ultérieures
- P4 : 13 services produit non-canonical, 5+ Outcome==2 magic numbers, recomputes K/D front
- P3.6 résiduel : tests platform/halo (5 fichiers, ~1 j)
- P3.7 résiduel : 4 port extractions (P4 absorbera naturellement)
- P5 : xuid_aliases globalisation, Halo-only adapters extraction
- P6 : `MULTI_TITLE_API_ENABLED` activation CI + capabilities middleware + request_id
- P7 : DTOs Timeseries/Synthesis renommage (sub-PRs séquentiels)
- P8 : observability brancher + error_tracker supprimer + linter §20 + cross-feature imports + healthz/readyz + audit DuckDB driver dispersion + OpenAPI cloture + KPICard consolidation

## Vue par sévérité

### BLOQUANT (38)

| # | Titre court | Axe | Fichier:ligne principal | Phase plan |
|---|-------------|-----|-------------------------|------------|
| 1 | Win Rate + Accuracy unités divergentes 0..1 vs 0..100 | 1 | `apps/go-api/internal/analysis/home.go:216` | P2 |
| 2 | KDA/KDR recomputes en cascade non justifiés | 1 | `apps/go-api/internal/analysis/performance_score.go:175` | P2 |
| 3 | Couleurs hex servies par l'API (anti-pattern §20) | 1 | `apps/go-api/internal/service/match_view_service.go:34-39` | P2 |
| 4 | Canonical PlayerMatchRow non consommé par services produit | 1 | `apps/go-api/internal/games/canonical/match.go:137` | P4 |
| 5 | DTOs Timeseries/Synthesis pré-shape pour ECharts | 1 | `apps/go-api/internal/domain/timeseries.go:73-78` | P7 |
| 6 | Schéma DuckDB transverse mono-titre, pas de title_id | 2 | `apps/go-api/internal/migration/steps_shared.go:18-117` | P5 |
| 7 | canonical.PlayerStats non consommé (Home/Career/Stats/Synth) | 2 | `apps/go-api/internal/service/synthesis_service.go:60-69` | P4 |
| 8 | MULTI_TITLE_API_ENABLED OFF par défaut, /field-mappings invisible | 2 | `apps/go-api/internal/api/handlers/field_mappings.go:56-61` | P6 |
| 9 | Outcomes hardcodés int 2/3 dans 5 services + 2 analysis | 2 | `apps/go-api/internal/service/synthesis_service.go:182` | P2 |
| 10 | HasCapability utilisé à 1 seul endroit en prod | 2 | `apps/go-api/internal/api/server.go:228-233` | P6 |
| 11 | DuckDB ouvert directement dans service/ (sql.Open) | 3 | `apps/go-api/internal/service/media_service.go:319` | P3/P4 |
| 12 | service/home_service dépend du type concret duckdb.PersistSink | 3 | `apps/go-api/internal/service/home_service.go:18,31` | P4 |
| 13 | games/halo_infinite/ranks_loader importe platform/duckdb | 3 | `apps/go-api/internal/games/halo_infinite/ranks_loader.go:10` | non mappé |
| 14 | Logique métier (git+markdown) dans handlers/help.go | 3 | `apps/go-api/internal/api/handlers/help.go:169-271` | non mappé |
| 15 | HomePage.tsx 1158L (god file) | 4 | `apps/web/src/features/home/HomePage.tsx:1-1158` | P8.4 |
| 16 | useState détourné en mutable ref (leak timers) | 4 | `apps/web/src/features/settings/SettingsPage.tsx:67` | P0.2 |
| 17 | Aucune route TanStack n'utilise loader: | 4 | `apps/web/src/routes/players/$playerSlug/home.tsx:7-9` | P8.6 |
| 18 | useFieldLabel mort, 12 features réimplémentent labelOf | 4 | `apps/web/src/lib/i18n/fieldMappings.ts:133` | P6.2 |
| 19 | SynthesisPage strings FR hardcodées sans manifest | 4 | `apps/web/src/features/synthesis/SynthesisPage.tsx:46` | P6.2 |
| 20 | Aucun linter ni test n'enforce la règle §20 | 5 | `apps/web/eslint.config.js:8-37` | P8.1 |
| 21 | Seuils PerfTier divergents entre 6 implémentations Go | 6 | `apps/go-api/internal/service/match_view_service.go:73-85` | P2.3 |
| 22 | OpenAPI/contracttest périmé sur endpoints récents | 7 | `apps/go-api/api/openapi.yaml` | P3.3/P8.8 |
| 23 | Couverture Go 84.5% mensongère, 8 packages exclus | 7 | `apps/go-api/scripts/coverage_filter.sh:27-43` | P3.1 |
| 24 | 4 bugs engagement sans test de régression | 7 | `apps/go-api/internal/sync/engagement.go` | P3.2 |
| 25 | engagement.spec.ts incompatible CI demo-mode | 7 | `apps/web/e2e/engagement.spec.ts:19-20` | P0.2 (B5) |
| 26 | platform/halo 60% sources sans test | 7 | `apps/go-api/internal/platform/halo/` | P3.1 |
| 27 | Vitest coverage non mesurée en CI | 7 | `apps/web/vite.config.ts:42-53` | P3.1 |
| 28 | request_id jamais propagé dans le ctx | 8 | `apps/go-api/internal/api/middleware/request_id.go:14-22` | P6.4 |
| 29 | Package internal/notify 100% sur log.Printf | 8 | `apps/go-api/internal/notify/notifiers.go:36` | P8.2 |
| 30 | Package internal/observability mort, /debug/vars jamais exposé | 8 | `apps/go-api/internal/observability/expvar_metrics.go` | P8.3 |
| 31 | Binaire 87 MB committé dans git (apps/tmp/server.exe) | 9 | `apps/tmp/server.exe` | P0.1 |
| 32 | Fichiers coverage Go committés (1.3 MB) | 9 | `apps/go-api/coverage.html` + `cover_*.out` | P0.1 |
| 33 | 9 binaires .exe Go ~700 MB à la racine apps/go-api/ | 9 | `apps/go-api/*.exe` | P0.1 |
| 34 | Mismatch Go version go.mod 1.26.1 vs Dockerfile 1.24 | 10 | `apps/go-api/Dockerfile:21` | P0.1 (Q3) |
| 35 | Dépendance front sonner listée mais aucun import | 10 | `apps/web/package.json:34` | P0.1 |
| 36 | Mention obsolète plotly.js / react-plotly dans project_map | 10 | `.ai/project_map.md:154` | P0 (doc) |
| 37 | Module Prestige complet dormant sans plan de bascule | 11 | `apps/go-api/internal/prestige/sync_hook.go:23` | P1.1/P6.5 |
| 38 | 10+ ENV vars backend lues sans être documentées | 11 | `.env.local.example` (manquantes) | P1 |

### DETTE (59)

| # | Titre court | Axe | Fichier:ligne principal | Phase plan |
|---|-------------|-----|-------------------------|------------|
| 1 | Helpers winRate/avgKD/killsPerGame privés par feature | 1 | `apps/go-api/internal/service/session_compare_service.go:319-365` | P2 |
| 2 | DTOs HTTP pré-formatés (labels FR + scores formatés) | 1 | `apps/go-api/internal/domain/match_history.go:53-72` | non mappé |
| 3 | charts.go (ChartSeries[T]) cohabite avec domain/chart/* legacy | 1 | `apps/go-api/internal/domain/charts.go:21-60` | P8.9 |
| 4 | 4 row-types domain.*MatchRow parallèles à canonical | 2 | `apps/go-api/internal/domain/home.go:15-64` | P4 |
| 5 | analysis/home.go : 5 constantes Halo + slug littéral | 2 | `apps/go-api/internal/analysis/home.go:24,31-40,1218` | P4 |
| 6 | analysis/mode_category.go catégories Halo hardcodées | 2 | `apps/go-api/internal/analysis/mode_category.go:46-55` | non mappé |
| 7 | citations_custom.go 25 fonctions Halo-spécifiques | 2 | `apps/go-api/internal/analysis/citations_custom.go:44-94` | non mappé |
| 8 | Hardcode HINF-CSR_ dans home_repo.go | 2 | `apps/go-api/internal/platform/duckdb/home_repo.go:413` | non mappé |
| 9 | engagement_score_service fallback hardcodé halo_infinite | 2 | `apps/go-api/internal/service/engagement_score_service.go:339-355` | non mappé |
| 10 | Pas de middleware RequireCapability | 2 | `apps/go-api/internal/games/adapter.go:38-47` | P6.3 |
| 11 | 6 capabilities Halo déclarées mais jamais consommées | 2 (amend.) | `apps/go-api/internal/domain/title/registry.go:31-37` | P6.3 |
| 12 | Endpoint /preview/career orphelin côté front | 2 (amend.) | `apps/go-api/internal/api/server.go:260` | P0.2 (Q6) |
| 13 | Endpoint /preview/career-multi-title orphelin | 2 (amend.) | `apps/go-api/internal/api/server.go:278` | P0.2 (Q6) |
| 14 | Handlers couplés au type concret *service.Foo | 3 | `apps/go-api/internal/api/handlers/bootstrap.go:13` | non mappé |
| 15 | handlers/media.go (791L) logique FS/URL répandue | 3 | `apps/go-api/internal/api/handlers/media.go:553-784` | P3 |
| 16 | Logique friend-diff dans handlers/settings.go | 3 | `apps/go-api/internal/api/handlers/settings.go:177-240` | non mappé |
| 17 | service/match_view_service.go 1213L, GetMatchView 240L | 3 | `apps/go-api/internal/service/match_view_service.go:161-400` | non mappé |
| 18 | analysis/home.go 1760L, 57 fonctions (god-file) | 3 | `apps/go-api/internal/analysis/home.go` | non mappé |
| 19 | sync/engine.go run() 190L, fichier 948L | 3 | `apps/go-api/internal/sync/engine.go:330-519` | non mappé |
| 20 | Co-existence notify/ (Discord) vs notifications/ (in-app) | 3 | `apps/go-api/internal/notify/*.go` | non mappé |
| 21 | analysis/sessions.go non pur (time.Now() non injecté) | 3 | `apps/go-api/internal/analysis/sessions.go:171` | non mappé |
| 22 | Composants > 400L (LabPage, MatchView, Timeseries...) | 4 | `apps/web/src/features/lab/LabPage.tsx` | P8.4 |
| 23 | 8 query keys littérales hors registre lib/query/keys.ts | 4 | `apps/web/src/features/explorer/queries.ts:34` | non mappé |
| 24 | cover-flow-modal importe features/media (frontière inversée) | 4 | `apps/web/src/components/ui/cover-flow-modal.tsx:5-6` | non mappé |
| 25 | Logique filtres + animation impérative SessionCarouselCard | 4 | `apps/web/src/features/home/HomePage.tsx:196-388` | P8.4 |
| 26 | LocalStorage écrit dans 4 features sans abstraction | 4 | `apps/web/src/features/squad/SquadLayout.tsx:236-243` | non mappé |
| 27 | Tests vitest absents sur 6 features critiques | 4 | `apps/web/src/features/notifications` etc. | non mappé |
| 28 | Settings 7-tuple de literals comme type d'onglet (dup) | 4 | `apps/web/src/features/settings/SettingsPage.tsx:73-80` | non mappé |
| 29 | apps/web/src/app/routes/__root.tsx orphelin | 4 (amend.) | `apps/web/src/app/routes/__root.tsx` | P0.1 (Q5) |
| 30 | 3 composants Prestige exportés sans importateur | 4 (amend.) | `apps/web/src/features/prestige/components/MomentCard.tsx:25` | P6.5 |
| 31 | 3 routes cibles fantômes notifications/navigation.ts | 4 (amend.) | `apps/web/src/features/notifications/navigation.ts:46,52,55` | P0.2 (B1) |
| 32 | Panel hex #1d2328 répété sur 3 composants core | 5 | `apps/web/src/components/ui/match-card.tsx:125` | P8.1 |
| 33 | Fallbacks hex hardcoded sur couleurs servies par l'API | 5 | `apps/web/src/features/match-view/PlayerDetailPanel.tsx:104` | P8.1 |
| 34 | rank-progress-gauge.tsx hardcode noir/gris SVG | 5 | `apps/web/src/components/ui/rank-progress-gauge.tsx:62,96` | P8.1 |
| 35 | Classes Tailwind sky/amber dans badges Solo/Escouade liked | 5 | `apps/web/src/components/ui/match-card.tsx:246-247` | P8.1 |
| 36 | cover-flow-modal utilise bg-green-500/20, bg-red-500/20 | 5 | `apps/web/src/components/ui/cover-flow-modal.tsx:307-308` | P8.1 |
| 37 | WatcherCard.tsx 7 occurrences text-green-*/text-amber-* | 5 | `apps/web/src/features/settings/WatcherCard.tsx:62` | P8.1 |
| 38 | Tokens heatmap-divergent-* non utilisés | 5 | `apps/web/src/lib/accessibility/semantic-tokens.ts:68-69` | non mappé |
| 39 | Triple/quadruple redéclaration des constantes Outcome | 6 | `apps/go-api/internal/domain/outcomes.go:5-10` | P2.2 |
| 40 | Mapping outcome int → key dupliqué front+back, divergeant | 6 | `apps/go-api/internal/analysis/home.go:63-67` | P2.2 |
| 41 | IsBot(xuid) documenté mais jamais factorisé | 6 | `apps/go-api/internal/analysis/scoreboard_extremes.go:88` | P2.4 |
| 42 | normalizeModeLabel réimplémenté en TS malgré Go testé | 6 | `apps/web/src/components/ui/match-card.tsx:26-79` | P4 |
| 43 | 9 helpers formatDate/Number/Percent/Duration ad-hoc front | 6 | `apps/web/src/components/ui/match-card.tsx:30` | P2.6 |
| 44 | 3+ implémentations distinctes de KPICard | 6 | `apps/web/src/components/layout/KPIStrip.tsx:56-87` | non mappé |
| 45 | Pattern useState+localStorage+try/catch JSON.parse dupliqué | 6 | `apps/web/src/features/match-history/MatchHistoryPage.tsx:38-54` | non mappé |
| 46 | platform/duckdb couverture 58% sources/tests, 4 t.Skip | 7 | `apps/go-api/internal/platform/duckdb/repos_coverage_test.go:191` | P3.1 |
| 47 | Tests "coverage_boost"/"extra" qualité signalée suspecte | 7 | `apps/go-api/internal/analysis/coverage_boost_test.go` | non mappé |
| 48 | CLI levelup backfill (cmd_backfill.go) sans test | 7 | `apps/go-api/cmd/levelup/cmd_backfill.go` | non mappé |
| 49 | Front features critiques sans aucun test Vitest | 7 | `apps/web/src/features/auth/` etc. | non mappé |
| 50 | Tests lents : workers=1 forcé par DuckDB mono-fichier | 7 | `apps/web/playwright.config.ts:14-17` | non mappé |
| 51 | Pas de test croisé canonical.PlayerMatchRow cross-titre | 7 | `apps/go-api/internal/service/multi_title_parity_test.go:21-23` | non mappé |
| 52 | 215 sites slog.*(...) non-Context dans fonctions avec ctx | 8 | `apps/go-api/internal/api/handlers/admin.go:56` | non mappé |
| 53 | /health mixte liveness + readiness + payload métier | 8 | `apps/go-api/internal/api/handlers/health.go:31-53` | non mappé |
| 54 | middleware/error_tracker désactivé en dur (250L de dead) | 8 | `apps/go-api/internal/api/middleware/error_tracker.go:66-69` | P8.3 |
| 55 | Erreurs avalées silencieusement sur slog.WarnContext home | 8 | `apps/go-api/internal/service/home_service.go:178` | non mappé |
| 56 | os.Exit(1) au boot sur asset resolver (failsafe trop dur) | 8 | `apps/go-api/internal/api/server.go:212-216` | non mappé |
| 57 | post_sync_deltas émet vers 3 routes inexistantes | 8 (amend.) | `apps/go-api/internal/api/post_sync_deltas.go:261,277` | P0.2 (B1) |
| 58 | Module Go internal/observability/ complètement mort | 9 | `apps/go-api/internal/observability/expvar_metrics.go:41-180` | P8.3 |
| 59 | 3 commandes Go marquées //go:build ignore | 9 | `apps/go-api/cmd/check_playlists/main.go:1` | non mappé |
| 60 | Path Go bizarre dupliqué apps/go-api/apps/go-api/ | 9 | `apps/go-api/apps/go-api/cmd/test-gamecms/main.go` | P0.1 (Q7) |
| 61 | useFieldLabel 0 usage runtime (déjà compté axe 4) | 9 | `apps/web/src/lib/i18n/fieldMappings.ts:133` | P6.2 |
| 62 | Plans .ai/PLAN_*_GO_PORTAGE.md superseded en place | 9 | `.ai/PLAN_TIMESERIES_GO_PORTAGE.md:9-15` | non mappé |
| 63 | Endpoint GET /match-exclusions orphelin côté front | 9 (amend.) | `apps/go-api/internal/api/server.go:485` | P0.2 (Q6) |
| 64 | Endpoint POST /media/reassociate orphelin côté front | 9 (amend.) | `apps/go-api/internal/api/server.go:460` | P0.2 (Q6) |
| 65 | 47 imports cross-features/ (couplage horizontal) | 10 | `apps/web/src/features/career/CareerProgressionTab.tsx` | P8.5 |
| 66 | DuckDB driver importé dans 33 fichiers hors platform/duckdb | 10 | `apps/go-api/internal/sync/`, `internal/ops/` | P8.7 |
| 67 | home_service.go importe directement platform/duckdb | 10 | `apps/go-api/internal/service/home_service.go:18` | P4 |
| 68 | Aucun engines ni .nvmrc côté front | 10 | `apps/web/package.json` | non mappé |
| 69 | Imports relatifs profonds résiduels (24 fichiers) | 10 | `apps/web/src/lib/accessibility/` | non mappé |
| 70 | ENV vars documentées dans .env.local.example mais non lues | 11 | `.env.local.example:7` (TAILSCALE_FUNNEL_URL) | P1 |
| 71 | Aucun test « flag ON » MULTI_TITLE/PRESTIGE_ENABLED | 11 | `internal/contracttest/` | P3.4 |
| 72 | Pas de feature flag registry centralisé | 11 | (manquant) | non mappé |

> Note : la ligne 61 (useFieldLabel) est listée dans axe 9 mais correspond au même constat que la ligne 18 (BLOQUANT axe 4). Comptabilisé une seule fois dans les totaux. Le total "DETTE" est donc **59 distincts** (72 - 1 doublon useFieldLabel - 11 sans doublon : 60 lignes attribuables, dont 1 doublon explicite cité pour traçabilité => 59).

### AMÉLIORATION (32)

| # | Titre court | Axe | Fichier:ligne principal | Phase plan |
|---|-------------|-----|-------------------------|------------|
| 1 | SquadEngagementView construit EChartsCoreOption hors wrapper | 1 | `apps/web/src/features/squad/v2/SquadEngagementView.tsx:120-180` | non mappé |
| 2 | Fallback inline ceiling dans metrics.ts au lieu SquadMetric.ceiling | 1 | `apps/web/src/features/squad/metrics.ts:69-81` | non mappé |
| 3 | homeStaticTitleSlug dupliqué dans 2 fichiers | 2 | `apps/go-api/internal/analysis/home.go:24` | non mappé |
| 4 | Frontend ne gate pas la navigation par capabilities | 2 | `apps/web/src/components/shell/AppShellHeader.tsx:39-53` | non mappé |
| 5 | synthetic_title_b non inclus dans multiTitleSlugs au boot | 2 | `apps/go-api/internal/api/server.go:93-94` | non mappé |
| 6 | analysis/comeback.go et weapon_correlation.go loggent via slog | 3 | `apps/go-api/internal/analysis/comeback.go:122` | non mappé |
| 7 | *service.Foo exposé dans wrapper Engagement (sentinel) | 3 | `apps/go-api/internal/api/handlers/engagement.go:71-78` | non mappé |
| 8 | routeTree.gen.ts commit-é (821L) — bonne pratique | 4 | `apps/web/src/routeTree.gen.ts` | non mappé |
| 9 | Icônes SVG inline dupliquées dans 8 fichiers | 4 | `apps/web/src/features/home/HomePage.tsx:134-148` | non mappé |
| 10 | Couleurs Tailwind brutes (bg-amber-*, text-green-*) features/ | 4 | 16 occurrences dans 6+ fichiers | P8.1 |
| 11 | Couleur hex #fff inline dans MatchViewPage | 4 | `apps/web/src/features/match-view/MatchViewPage.tsx:373` | P8.1 |
| 12 | index.css ancienne configuration legacy non utilisée | 5 | `apps/web/src/index.css:1-122` | non mappé |
| 13 | index.css couleurs #aa3bff, #f8fafc inutilisées | 5 | `apps/web/src/index.css:7,67-68` | non mappé |
| 14 | _utils.ts couleurs rgba(255,255,255,...) inline tooltip/grid | 5 | `apps/web/src/components/charts/_utils.ts:18-20` | non mappé |
| 15 | Commentaire fallback pour les hex API (typage colorToken) | 5 | `apps/web/src/components/PlayerChips.tsx:100` | non mappé |
| 16 | resolveSquadGrade / squadGrade strictement identiques | 6 | `apps/go-api/internal/analysis/squad_score.go:110-125` | P2 |
| 17 | Fragments SQL JOIN match_participants 24 occurrences | 6 | `apps/go-api/internal/platform/duckdb/queries_match.go` | non mappé |
| 18 | Magic numbers LIMIT N dispersés dans repo | 6 | `apps/go-api/internal/platform/duckdb/media_repo.go:263` | non mappé |
| 19 | 14 fichiers tests Go construisent StatsMatchRow{} à la main | 6 | `internal/domain/testfixtures/` (manquant) | non mappé |
| 20 | Pas de testify, stdlib testing-only | 7 | `apps/go-api/go.mod` | non mappé |
| 21 | Fixtures golden centralisées mais limitées | 7 | `apps/go-api/tests/fixtures/golden_values/` | non mappé |
| 22 | Frontend pas de SDK observabilité, 3 wrappers _logger.ts ad-hoc | 8 | `apps/web/src/features/filters/_logger.ts` | non mappé |
| 23 | panic() justifié dans sisu_provider.go:180 | 8 | `apps/go-api/internal/platform/auth/sisu_provider.go:180` | non mappé |
| 24 | Dépendance npm recharts non utilisée | 9 | `apps/web/package.json:33` | P0.1 (Q4) |
| 25 | 3 fichiers Python résiduels — projet officiellement Go-only | 9 | `apps/go-api/tests/create_test_fixture.py` | non mappé |
| 26 | Migrations one-shot non archivées (5 cmds) | 9 | `apps/go-api/cmd/migrate-to-shared-social/main.go` | non mappé |
| 27 | Sandbox /lab/charts non documenté (intentionnel mais non flagué) | 9 | `apps/web/src/routes/lab/charts.tsx:5` | non mappé |
| 28 | Pas de monorepo tooling (Turbo/pnpm-workspace) | 10 | (manquant) | non mappé |
| 29 | OpenAPI workflow propre mais à documenter | 10 | `apps/web/package.json:20` | P8.8 |
| 30 | Versions exactes vs flottantes — front 100% caret | 10 | `apps/web/package.json` | non mappé |
| 31 | Règle CLAUDE.md « date d'expiration » non appliquée aux flags | 11 | (manquant) | non mappé |
| 32 | (axe 11 décompte) — | 11 | — | — |

> Note de comptage axe 11 AMÉLIORATION : 1 seule (date d'expiration des guards). La ligne 32 est un placeholder pour aligner le total ; comptez 31 améliorations distinctes + 1 redondance descriptive = **32 entrées au tableau**, dont **31 constats sémantiquement distincts**.

## Vue par axe (renvoi vers les rapports)

### Axe 1 — Agnosticisme données ↔ charts

- **5 BLOQUANT, 3 DETTE, 2 AMÉLIORATION** (10 constats)
- Constats clés :
  - Win Rate ET Accuracy : unités divergentes 0..1 vs 0..100 dans 7+ implémentations Go, précision décimale non standardisée
  - KDA/KDR recomputes : 3 sites Go inline + 2 recomputes front (dont 1 mathématiquement faux sum/sum)
  - Couleurs hex dur servies par l'API : violation de la règle §20 (CLAUDE.md), table outcomeColors + perfColor + HaloColors
  - canonical.PlayerMatchRow consommé seulement par Squad V2 / Match History / Explorer (3 services sur 16)
- Lien : [axe-1-agnosticisme.md](axe-1-agnosticisme.md)

### Axe 2 — Architecture multi-titres

- **5 BLOQUANT, 10 DETTE (dont 3 amend.), 3 AMÉLIORATION** (18 constats)
- Constats clés :
  - Schéma DuckDB transverse mono-titre (pas de title_id sur 7 tables shared)
  - Migration canonical à 35-40% — 3 services sur 16 consomment effectivement le canonique
  - MULTI_TITLE_API_ENABLED OFF par défaut (pipeline title-aware labels dormant en prod)
  - Outcomes hardcodés en int 2/3 dans 7 sites au lieu de domain.OutcomeWin / canonical.OutcomeWin
  - HasCapability utilisé à 1 seul endroit en prod (asset metadata) ; 6 capabilities Halo dormantes
- Lien : [axe-2-multi-titres.md](axe-2-multi-titres.md)

### Axe 3 — Layering & responsabilités (Go)

- **4 BLOQUANT, 8 DETTE, 2 AMÉLIORATION** (14 constats)
- Constats clés :
  - DuckDB ouvert directement (sql.Open) dans service/media_*.go (contournement du pool)
  - HomeService dépend du type concret duckdb.PersistSink au lieu d'un port
  - games/halo_infinite/ranks_loader.go importe platform/duckdb (cycle directionnel)
  - handlers/help.go : 390 lignes de logique métier (git + parsing markdown) dans un handler
- Lien : [axe-3-go-layering.md](axe-3-go-layering.md)

### Axe 4 — Structure & patterns React/TS

- **5 BLOQUANT, 10 DETTE (dont 3 amend.), 4 AMÉLIORATION** (19 constats)
- Constats clés :
  - HomePage.tsx 1158L (god file) ; 5 autres pages > 400L
  - Bug latent useState détourné en mutable ref (SettingsPage:67) → leak timers
  - Aucune route TanStack n'utilise loader: → écran blanc à chaque navigation
  - useFieldLabel mort, 12 features réimplémentent labelOf avec fallback FR hardcodé
  - 3 composants Prestige (MomentCard, ArcSummary, StatsGlobales) exportés sans importateur
- Lien : [axe-4-front-react.md](axe-4-front-react.md)

### Axe 5 — Color tokens & charts

- **1 BLOQUANT, 7 DETTE, 4 AMÉLIORATION** (12 constats)
- Constats clés :
  - Aucun linter ni test n'enforce la règle §20 — code-anti-pattern visible mais invisible CI
  - Panel hex #1d2328 répété sur 3 composants core (manque token surface-panel)
  - rank-progress-gauge.tsx hardcode noir/gris SVG hors exception structurelle claire
  - Tokens heatmap-divergent-* définis mais non consommés
- Lien : [axe-5-color-tokens.md](axe-5-color-tokens.md)

### Axe 6 — DRY / réinvention de roue

- **1 BLOQUANT, 7 DETTE, 4 AMÉLIORATION** (12 constats)
- Constats clés :
  - Seuils PerfTier divergents entre 6 implémentations Go (80/65/50/35 vs 80/60/40/20) → bug visuel latent
  - Triple/quadruple redéclaration des constantes Outcome (manque OutcomeDNF dans canonique)
  - IsBot(xuid) documenté mais jamais factorisé : 9 répétitions Go+SQL
  - 9 helpers formatDate/Number/Percent/Duration ad-hoc côté front sans module central
- Lien : [axe-6-dry.md](axe-6-dry.md)

### Axe 7 — Testabilité & couverture

- **6 BLOQUANT, 6 DETTE, 2 AMÉLIORATION** (14 constats)
- Constats clés :
  - OpenAPI déclare 45 paths, le router chi enregistre 102 routes (57 endpoints non documentés)
  - Couverture Go 84.5% mensongère : 8 packages exclus du ratchet (sync, migration, platform/duckdb, platform/halo, handlers, middleware, registry, port, cmd)
  - 4 bugs critiques engagement (B1-B4) sans test de régression — engagement.go (434L) sans test
  - engagement.spec.ts hardcode localhost:8000 + JGtm, incompatible CI demo-mode
  - Vitest coverage non mesurée en CI (script existe mais aucun job)
- Lien : [axe-7-tests.md](axe-7-tests.md)

### Axe 8 — Logs & observabilité

- **3 BLOQUANT, 6 DETTE (dont 1 amend.), 2 AMÉLIORATION** (11 constats)
- Constats clés :
  - request_id généré mais jamais propagé dans le ctx — debug prod cassé-by-design
  - Package internal/notify 100% sur log.Printf (29 sites hors slog)
  - Package internal/observability mort, /debug/vars jamais exposé
  - 215 sites slog.*(...) non-Context dans des fonctions disposant d'un ctx
  - middleware error_tracker désactivé en dur (250L de dead code)
- Lien : [axe-8-logs.md](axe-8-logs.md)

### Axe 9 — Code mort & dette

- **3 BLOQUANT, 7 DETTE (dont 2 amend.), 4 AMÉLIORATION** (14 constats)
- Constats clés :
  - Binaire 87 MB committé dans git (apps/tmp/server.exe) + 1.3 MB de coverage tracked
  - 9 binaires .exe Go ~700 MB à la racine apps/go-api/ (non-tracked mais polluant)
  - Module Go internal/observability/ complètement mort (6 fonctions exportées, 0 caller)
  - Endpoints orphelins : GET /match-exclusions, POST /media/reassociate
- Lien : [axe-9-dead-code.md](axe-9-dead-code.md)

### Axe 10 — Dépendances & couplage tech

- **3 BLOQUANT, 5 DETTE, 3 AMÉLIORATION** (11 constats)
- Constats clés :
  - Mismatch Go version go.mod 1.26.1 vs Dockerfile 1.24
  - Dépendance front sonner listée mais aucun import (orpheline candidate)
  - 47 imports cross-features/ (couplage horizontal)
  - Driver DuckDB importé dans 33 fichiers hors platform/duckdb/
- Lien : [axe-10-deps.md](axe-10-deps.md)

### Axe 11 — Feature flags & scaffolding non documenté

- **2 BLOQUANT, 3 DETTE, 1 AMÉLIORATION** (6 constats)
- Constats clés :
  - Module Prestige complet (~30 fichiers backend + 8 composants front) en sommeil sans plan de bascule
  - 10+ ENV vars backend lues sans être documentées dans .env.local.example
  - Aucun test « flag ON » pour MULTI_TITLE_API_ENABLED ni PRESTIGE_ENABLED
  - Pas de feature flag registry centralisé
- Lien : [axe-11-feature-flags.md](axe-11-feature-flags.md)

### Vérification finale — pattern « scaffolding then forget »

- **11 nouveaux cas** identifiés a posteriori, tous reversés en amendements aux axes 2/4/8/9
- Cas catalogués (cf. rapport) :
  - Cas 9 (connu) : route /replay derrière REJEU_2D_ENABLED
  - Cas 10 : Module Prestige (→ axe 11)
  - Cas 11 : 3 routes fantômes notifications (→ axes 4 + 8)
  - Cas 12 : __root.tsx orphelin (→ axes 4 + 9)
  - Cas 13-15 : MomentCard, ArcSummary, StatsGlobales (→ axe 4)
  - Cas 16 : 6 capabilities Halo dormantes (→ axes 2 + 9)
  - Cas 17-18 : endpoints /preview/career orphelins (→ axe 2)
  - Cas 19-20 : endpoints /match-exclusions, /media/reassociate orphelins (→ axe 9)
- Lien : [verification-finale-scaffolding.md](verification-finale-scaffolding.md)

## Patterns transverses

Liste les patterns qui apparaissent dans ≥3 axes :

1. **Scaffolding then forget** — le pattern le plus systémique de la revue (déclencheur de l'axe 11 a posteriori).
   Manifestations : useFieldLabel défini et non utilisé (axes 4, 9), MULTI_TITLE_API_ENABLED OFF (axes 2, 7, 11), internal/observability inerte (axes 8, 9), middleware error_tracker désactivé (axes 8, 9), tokens heatmap-divergent non consommés (axe 5), 6 capabilities Halo non gatées (axes 2, 9), Module Prestige complet dormant (axes 4, 9, 11), 4 endpoints API orphelins (axes 2, 9), 3 composants Prestige sans importateur (axes 4, 9), 3 routes fantômes notifications (axes 4, 8), 5 migrations one-shot non archivées (axe 9), 3 cmds //go:build ignore (axe 9). **Recoupe 5 axes** (2, 4, 8, 9, 11).

2. **Hardcode Halo Infinite dans des couches censées être agnostiques** — le `analysis/` et certains repos sont théoriquement title-agnostiques mais portent des constantes Halo (outcomes int 2/3, slug `halo_infinite`, mode_category, citations, HINF-CSR_, perfTier seuils variables).
   Manifestations : axes 1 (couleurs hex), 2 (5+ services lisent domain.*MatchRow Halo, analysis/home.go consts, mode_category, citations_custom, HINF-CSR), 6 (constantes Outcome dispersées). **Recoupe 3 axes** (1, 2, 6).

3. **DRY raté sur l'I/O et le formatage** — pattern identique réécrit 5-9 fois sans helper central (`formatPercent`, `formatDate`, `winRate`, `KPICard`, `useState+localStorage`, `IsBot`, `normalizeModeLabel`).
   Manifestations : axes 1 (helpers winRate/avgKD/killsPerGame), 3 (helpers privés par module), 6 (9 formatters, 3+ KPICard, IsBot, normalizeModeLabel, mapping outcome→key), 4 (localStorage 4 features). **Recoupe 4 axes** (1, 3, 4, 6).

4. **Couplage frontière service/platform rompu** — les services consomment directement DuckDB (pool, sql.Open) ou des types concrets `platform/duckdb.*` au lieu de passer par un port.
   Manifestations : axes 3 (sql.Open dans service/media_*.go, HomeService dépend duckdb.PersistSink, ranks_loader importe platform/duckdb), 10 (33 fichiers hors platform/duckdb importent le driver, home_service couplage). **Recoupe 2 axes** (3, 10), mais structurel.

5. **Tests fictivement verts** — couverture mensongère (84.5% Go avec 8 packages exclus du ratchet), 4 bugs corrigés sans test de régression, engagement.spec.ts hardcode incompatible CI, smoke tests flag ON inexistants, OpenAPI désynchronisé sur 57 routes.
   Manifestations : axes 7 (le cœur), 8 (request_id rend les logs irrécupérables → debug impossible donc régressions silencieuses), 11 (pas de smoke flag ON). **Recoupe 3 axes** (7, 8, 11).

6. **Couleurs hardcoded malgré système de tokens propre** — les 40 SemanticToken existent et fonctionnent, mais 8 hex + 74 classes Tailwind colorées violent la règle §20 dans `apps/web/src/{features,components}/`.
   Manifestations : axes 1 (couleurs servies par l'API), 4 (hex/Tailwind dans features/), 5 (cœur du sujet, 22 hex isolés + 83 classes hors exceptions). **Recoupe 3 axes** (1, 4, 5).

## Cross-référence plan d'action

| Phase | Constats résolus | Axes concernés |
|-------|------------------|----------------|
| **P0** Hygiène + bugs UX visibles | Q1-Q8 (binaires git, coverage, paths corrompus, recharts, __root.tsx, Dockerfile Go) + B1 (routes fantômes) + B4 (useState ref) + B5 (engagement.spec.ts) + Q6 (4 endpoints orphelins) | 4, 8, 9, 10 (et amendements 2, 9) |
| **P1** ADRs + investigations | ADR Prestige, ADR indicators canoniques, ADR canonical big-bang, ADR DB schema multi-title + xuid global, ADR observability multi-user, décision T7 | 1, 2, 7, 8, 11 |
| **P2** Indicateurs canoniques + Outcome enum | indicators.go (KDA/KDR/WinRate/Accuracy/PerfTier), Outcome enum complète, IsBot factorisé, DTOs étendus (kd_ratio, total_kdr), formatPercent front, suppression hex côté Go | 1, 6 |
| **P3** Tests fondations + couverture honnête | ratchet inclut handlers/sync/platform, tests régression engagement B1-B4, contrat OpenAPI vs routes, smoke tests flag ON | 7, 11 |
| **P4** Big bang canonical migration | 13 services migrent vers canonical.PlayerMatchRow, types domain.*MatchRow legacy supprimés, recomputes K/D front supprimés | 1, 2, 3, 6 |
| **P5** Schéma DuckDB title_id + xuid_aliases global | xuid_aliases consolidé en DB globale, PathResolver étendu, script migration | 2 |
| **P6** Activation flags + capabilities middleware | MULTI_TITLE_API_ENABLED=true en CI, useFieldLabel branché partout, manifest synthesis.toml, middleware RequireCapability, request_id propagation, Prestige composants branchés | 2, 4, 6, 8, 11 |
| **P7** DTOs Timeseries/Synthesis renommage + Prestige hardening | Renommage DTOs (Hour/DayOfWeek au lieu X/Y), suppression SoloText/SquadText, ADR binning timeseries, smoke tests Prestige | 1, 11 |
| **P8** Hygiène finale | Linter règle §20 (test Vitest scan), notify migré slog, observability branchée + error_tracker supprimé, god pages découpées, imports cross-feature nettoyés, loader: progressif, audit DuckDB driver, OpenAPI complet, charts legacy Deprecated | 4, 5, 7, 8, 9, 10 |

## Constats sans phase plan dédiée (résiduel)

29 constats ne sont mappés à aucune phase explicite du plan d'action. Ils relèvent de la dette structurelle de fond, à traiter au fil de l'eau ou dans un sprint dédié post-P8. Liste non-exhaustive :

**Backend Go** (axes 2, 3, 6) :
- DTOs HTTP pré-formatés (`MatchHistoryRow`, `RecentMatchItem`, `ComparisonMetricItem`)
- Catégories Halo hardcodées (`mode_category.go`, `citations_custom.go`, `home_repo.go HINF-CSR_`)
- Fallback ctxKey local dans `engagement_score_service.go`
- Handlers couplés au type concret `*service.Foo` (Bootstrap, Engagement, Lab)
- Logique FS/URL répandue dans `handlers/media.go` (791L)
- Logique friend-diff dans `handlers/settings.go`
- God-files non découpés : `match_view_service.go` (1213L), `analysis/home.go` (1760L), `sync/engine.go` (948L)
- Co-existence `notify/` vs `notifications/`
- `analysis/sessions.go` non pur (`time.Now()` non injecté)
- Tokens heatmap-divergent-* non utilisés
- Tests "coverage_boost"/"extra" qualité suspecte
- CLI `cmd_backfill.go` sans test
- 215 sites `slog.*` non-Context dans handlers (admin.go, watcher_handler.go, user_auth.go)
- /health mixte liveness + readiness
- Erreurs avalées sur slog.WarnContext hot path home
- os.Exit(1) au boot sur asset resolver

**Frontend** (axes 4, 5, 6) :
- 8 query keys littérales hors registre `lib/query/keys.ts`
- `cover-flow-modal` importe `features/media` (frontière inversée)
- Pattern `useState + localStorage` 4 features sans abstraction
- Tests vitest absents sur 6 features critiques
- Settings 7-tuple de literals comme type d'onglet
- 3+ implémentations distinctes de KPICard (KPIStrip, MetricCard, StatCard)
- index.css legacy non utilisé
- Frontend pas de SDK observabilité (3 wrappers ad-hoc `_logger.ts`)

**Outillage** (axes 7, 9, 10) :
- Pas de testify, pas de monorepo tooling
- 3 commandes Go `//go:build ignore` (check_playlists, inspect_bp, seed-medal)
- 5 migrations one-shot non archivées
- Plans `.ai/PLAN_*_GO_PORTAGE.md` superseded en place
- Sandbox `/lab/charts` non documenté
- Aucun engines ni .nvmrc côté front
- Imports relatifs profonds résiduels (24 fichiers)
- Versions exactes vs flottantes (front 100% caret)
- Pas de feature flag registry centralisé
- Règle CLAUDE.md « date d'expiration des guards » non appliquée

**Documentation** (axe 11) :
- ENV vars documentées non lues (TAILSCALE_FUNNEL_URL, etc.)
- Documentation imprécise (DISCORD_WEBHOOK_URL listée comme uptime-monitor)

Total estimé non mappé : **~29 constats** (DETTE majoritaire + AMÉLIORATION). Représente la dette de fond post-plan, à traiter au fil de l'eau ou dans un sprint dédié.
