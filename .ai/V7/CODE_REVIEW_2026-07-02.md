# Revue de code — LevelUp — 2026-07-02

> Audit qualité/maintenabilité/réutilisation multi-agents (4 passes parallèles : architecture Go,
> composants React, i18n, duplication/code mort) + scans mécaniques (tailles, lint, couleurs,
> query keys). Chaque finding vérifié sur le code réel ; les affirmations « jamais appelé »
> contre-vérifiées avec plusieurs patterns de recherche.
> Périmètre : `apps/go-api/internal/` + `apps/web/src/`.
> Complémentaire de `.ai/V7/CODE_REVIEW_2026-06-02.md` (sécurité/déploiement) et de
> `.ai/AUDIT_ARCHI_GO_API_2026-07.md` (architecture Go approfondie — recoupe A2/A5 ci-dessous).

---

## 1. Résumé exécutif

La qualité ligne-à-ligne est **élevée** des deux côtés : logging slog irréprochable, zéro `fetch`
sauvage côté web, discipline title-agnostic réellement tenue (aucun `slug == "halo_infinite"`
littéral en prod), couleurs sémantiques bien canalisées, commentaires forensiques de grande valeur.
Le problème n'est pas la qualité moyenne mais la **concentration et la non-repropagation** :

1. **2 bugs utilisateur actifs** — l'échelle de couleurs performance est *inversée* sur le tab
   Forme (pires matchs en vert), et les badges de résultat de la page Carrière tombent tous en
   gris en locale EN.
2. **1 risque d'intégrité de données** — le recompute LUSR v1 (documenté « ne jamais utiliser »)
   est toujours exposé en HTTP et CLI à côté du chemin v2 canonique.
3. **3 God-functions Go** (`NewRouter` 1472L, `engine.run` 486L, `handleStartBackfill` 368L)
   portent un risque de modification disproportionné, et la couche `api` absorbe du SQL qui
   appartient à `platform/duckdb`.
4. **~40 fichiers de code mort vérifié** (cluster `home_*` legacy Go, feature `session-compare`
   entière web+Go, `SquadV2`, chaîne `NotifyNewMedia`) — le « dead code museum » que CLAUDE.md
   interdit, avec des tests qui entretiennent l'illusion de vie.
5. **Des patterns déjà factorisés mais non repropagés** : les helpers canoniques existent
   (`sql_fragments.go`, `lib/formatters/`, `useLocalFilterBar`, `perfScale`) mais leurs copies
   locales survivent — 87 copies du COALESCE timezone, 36 littéraux bot, 4 formatters de durée.
   C'est la duplication la plus dangereuse : celle qui diverge d'une source de vérité existante.
6. **La gouvernance des seuils a dérivé** : le `.golangci.yml` a relaxé 80→100L puis a
   **désactivé** `funlen`/`gocyclo` sur des répertoires entiers (`sync/`, `analysis/`,
   `service/`, `handlers/`), sans baseline gelée — contrairement au ratchet Python.

## 2. Conformité checklist CLAUDE.md (état factuel)

| Item | État | Mesure |
|---|---|---|
| Fichier <= 500L | NON | 66 fichiers Go (~25 en prod hors migrations/seeds), 17 TS/TSX hors i18n/tests |
| Fonction <= 80L | NON | 182 fonctions Go (168 hors migrations) ; lint relaxé à 100L puis désactivé par répertoire |
| Pattern <= 2 copies | NON | COALESCE TZ x87, prédicat bot x36, icône SVG x9, socle ECharts x7, `strPtr` x5, formatters date/percent x4 |
| Magic numbers = 0 | PARTIEL | Résiduels ciblés (seuils deltas 0.05/0.01, `outcome = 2` en dur x1, timeouts inline) — pas systémique |
| Code mort = 0 | NON | ~40 fichiers sur 4 clusters vérifiés (détail section 4) |
| Nommage 1 verbe + 1 complément | QUASI-OUI | 1 seul cas trouvé (`joinAndSort`) |
| Couleurs via tokens | OUI | Exceptions toutes annotées `color-allow` §20 ; mais 1 inversion sémantique (C1) |
| i18n FR/EN complet | PARTIEL | Parité garantie par le typage `Record<Locale, T>` ; trous concentrés sur les chemins d'erreur + famille « streak » |

---

## 3. Findings CRITIQUES

**C1 — Échelle perf-tier inversée : les pires matchs s'affichent en vert, les meilleurs en rouge.**
`apps/web/src/features/timeseries/TimeseriesFormCharts.tsx:51-57` — le `perfTier()` local mappe
`score < 20 -> 'perf-tier-1'` alors que la source de vérité
`apps/web/src/lib/accessibility/scales/instances.ts:20-23` (`perfScale`, protégée par snapshot CI)
mappe `>= 80 -> perf-tier-1`, et tier-1 = vert dans les tokens CSS. Les seuils divergent aussi
(20/40/60/80 vs 35/50/65/80). Tous les autres consommateurs sont cohérents
(`lib/perf-color.ts:19`, `ExplorerMatchesTable.tsx:456`).
-> Supprimer la copie locale, importer `perfScale`. Fix de 5 minutes, vérification visuelle du
graphe « Performance » du tab Forme.

**C2 — Badges de résultat décidés par matching de sous-chaîne française : cassés en locale EN.**
`apps/web/src/features/career/CareerTopMatchesTable.tsx:124-129` —
`outcome_label.toLowerCase().includes('victoire')` : en EN (« Victory »/« Defeat »), tous les
badges retombent en `secondary` gris. Même fichier, l.99 : `toLocaleDateString('fr-FR')` figé.
-> Consommer un code outcome + `outcomeKey`/`getOutcomeColor` comme le fait
`ExplorerMatchesTable.tsx:317-327`. Jamais de logique sur un label localisé.

**C3 — Le recompute LUSR v1 est toujours câblé en HTTP et CLI, en concurrence avec le chemin v2.**
`RunBackfillLUSR` (v1, `apps/go-api/internal/sync/engine_backfills.go:182`) reste exposé via
`apps/go-api/internal/api/handlers/backfill.go:334` et `cmd/levelup/cmd_backfill.go:411,431`,
alors que le chemin canonique est `RecomputeLUSRCanonicalForPlayer` (v2). Deux chemins concurrents
écrivent `match_skill_rank` ; déclencher le backfill HTTP peut réintroduire un état v1 périmé —
la connaissance projet dit explicitement « jamais RunBackfillLUSR ».
-> Rerouter le handler et la CLI vers le chemin v2, ou retirer l'option et supprimer v1.

**C4 — Documentation inversée sur deux flags critiques pour l'anti-corruption ART.**
- `apps/go-api/internal/sync/engine_options.go:130-131` affirme que `LEVELUP_PERSIST_BATCH` est
  « par défaut désactivé » — faux : `scheduler/auto_sync.go:336` et `handlers/sync_handler.go:210`
  font `!= "0"` (défaut **ON**). Docs identiquement périmées dans `engine.go:128-133` et
  `engine_batch_path.go:16`.
- `apps/go-api/internal/sync/v2/doc.go:17` : « défaut v1 » — contredit `cmd/server/main.go:1141`
  (V2 par défaut depuis ADR 0027).
Sur le chemin anti-ART, une doc inversée est un risque opérationnel réel (un dev qui « remet le
défaut » réactive le chemin ART-unsafe).
-> Réécrire les 4 commentaires (défaut ON, `=0`/`=v1` = kill-switch) et dater le retrait des
chemins legacy.

---

## 4. Findings MAJEURS

### 4.1 Architecture Go

**A1 — `NewRouter` : 1472 lignes, 9 paramètres, mélange DI + I/O + métier.**
`apps/go-api/internal/api/server.go:245-1717` — composition root, ouverture de handles DuckDB,
purge de sessions en goroutine (l.269-285), retry-loop metadata (l.739-800), et une closure
métier de 46L (`waypointExplore`, l.1045-1091). S'y ajoute un double chargement identique de la
metadata H5 au boot (l.773 vs l.816 — 2 ouvertures DB + 6 requêtes, alors que le commentaire
l.805 revendique « chargé UNE fois »).
-> Extraire des builders par domaine (`buildAuthRoutes`, `mountPlayerRoutes`,
`buildTitleAdapters`...) ; `waypointExplore` va dans un service.

**A2 — SQL brut dans la couche `api`.**
`apps/go-api/internal/api/server.go:137-222` (`loadTitleAssetDrawerData`,
`loadCSRBadgeResolver`) et `apps/go-api/internal/api/post_sync_deltas_records.go:21-48`
(`loadPlayerRecord`/`upsertPlayerRecord`) exécutent du SELECT/UPSERT directement depuis `api`,
en violation de la frontière `platform/duckdb`. Le voisin `loadTitleRankImageURLs` (l.231)
montre le bon pattern. (Recoupe le foyer « racine api/ = 2e couche service de facto » de
`.ai/AUDIT_ARCHI_GO_API_2026-07.md`.)
-> Déplacer ces loaders dans `platform/duckdb`.

**A3 — `handleStartBackfill` : pipeline métier de 368L dans un handler.**
`apps/go-api/internal/api/handlers/backfill.go:76-443` — goroutine de 310L orchestrant 10 phases,
construit `go_sync.NewSyncEngine` en direct (l.135), avec ~10 copies du bloc
`SetStatus -> Run -> append(Warnings)`.
-> Extraire un `service.BackfillOrchestrator` table-driven (`{nom, gate, fn}`) ; le handler ne
garde que validation + 202.

**A4 — `engine.run()` : 486L au control-flow defer/LIFO le plus fragile du codebase.**
`apps/go-api/internal/sync/engine.go:180-665` — finalizer armé par flag, closure-swap de
`releaseShared` anti double-release (l.524-526), release/ré-acquisition de leases autour du Drain
(l.514-586). Tout est documenté, mais la correction dépend de l'ordre de 6+ defers sur 486L.
-> Extraire au minimum le bloc drain/ré-acquisition (l.505-597) et la boucle de pagination
(l.346-497) en méthodes nommées.

**A5 — Handlers qui construisent des repos DuckDB et assemblent le métier, sans couche service.**
`apps/go-api/internal/api/handlers/progression.go:162-251` — `handleStreaks`/`handleRecords`/
`handleMilestones` instancient 4 repos et font la jointure catalog x earned dans le handler.
Import direct `platform/duckdb` aussi dans `player_profile.go`, `campaign.go`, `home.go`,
`admin_auto_sync.go`. Couplage horizontal service->service en miroir :
`apps/go-api/internal/service/home_service.go:55` détient un `*CareerLiveService` concret ;
idem `career_service.go:74`, `filters_service.go:26`.
-> Un `ProgressionService` + interfaces port (`port.SpartanIdentityProvider`...) ; câblage
concret au registry uniquement.

**A6 — 41 `os.Getenv` hors `internal/config` : flags indécouvrables et relus à plusieurs endroits.**
`LEVELUP_PERSIST_BATCH` est lu dans le scheduler ET le handler sync (risque de divergence) ;
6 flags dans `auto_sync.go`, d'autres au fond de `skill_v2_*.go`.
-> Centraliser dans `config.AppConfig` au boot, injecter.

### 4.2 Code mort (« dead code museum »)

**A7 — Cluster home legacy : 10 exports sans aucun appelant de prod, doc de package fausse.**
`ComputeKPIs`, `ComputeTrend`, `BuildHeroCard`, `BuildHighlights`, `BuildSessionSummaries`,
`BuildRecentMatches*` (x4) dans `apps/go-api/internal/analysis/home_kpis.go`,
`home_highlights.go`, `home_sessions.go`, `home_recent.go` — la prod n'appelle que les variantes
`*FromCanonical` (vérifié, y compris transitifs `home_highlights_tiles.go`). Aggravant :
`home_canonical.go:4-18` affirme « chaque *FromCanonical délègue à la version legacy... pas de
duplication, pas de risque de drift » — c'est faux, ce sont des réimplémentations complètes ;
le drift « impossible » est déjà réel.
-> Supprimer les 10 exports + tiles legacy + leurs tests ; conserver les 4 helpers partagés
(`mapImageURLFromRegistry`, `mmrDelta`, `float64PtrVal`, `intPtrIfPos`) ; corriger le doc.

**A8 — Feature `session-compare` entière morte, avec chaîne Go zombie end-to-end.**
`apps/web/src/features/session-compare/SessionComparePage.tsx` (761L) n'est montée par aucune
route (vérifié `routes/` + imports dynamiques ; seules refs = un commentaire et la query key).
17 fichiers front + côté Go : `handlers/session_compare.go`, `service/session_compare_service.go`
+ 3 helpers, `domain/session_compare.go`, entrée `openapi.yaml` — un endpoint complet compilé,
servi, testé, sans consommateur atteignable (~25 fichiers cumulés).
-> Décision produit : router la page ou tout supprimer (front + endpoint + service + manifest
§compare).

**A9 — Autres clusters morts vérifiés.**
- `apps/web/src/features/squad/v2/SquadV2RouteHost.tsx:32` + `SquadV2Page.tsx` : 0 import réel
  (routes squad = `SquadLayout`/`SquadSynergiesPage`) — attention, `squad/v2/types.ts` reste
  vivant.
- Chaîne `NotifyNewMedia` -> `queryUnnotifiedMedia` -> `markMediaNotified`
  (`apps/go-api/internal/notify/notifiers.go:88-190`) : appelée uniquement par ses propres
  tests ; en plus elle ouvre la DB en RW direct hors lease/persister (violerait ADR 0013/0022
  si ravivée) et une migration de colonne `discord_notified_at` vit pour elle.
- `upsertLUSRRatingsLegacy` (`apps/go-api/internal/sync/skill_rating_loaders.go:232`) :
  0 caller, 0 test.
- `apps/web/src/features/session-detail/SessionKDATimeline.tsx` et `SessionOcdrScatter.tsx` :
  charts ECharts finis, jamais montés.
- `MapMuToLegacyRating`/`MapTierSubToLegacyRating` (skill_v2) : exports test-only, à dé-exporter.
- `processMatch` (`apps/go-api/internal/sync/engine_batch_path.go:54`) : « legacy test-only » —
  les tests V1 exercent un chemin que la prod n'exécute plus et qui divergeait déjà.

### 4.3 Duplication contre source de vérité existante

**A10 — Le COALESCE timezone canonique est copié 87 fois dans 33 fichiers.**
Le pattern `COALESCE(x.start_time_utc, x.start_time AT TIME ZONE 'UTC')` (règle projet
documentée) n'a jamais été ajouté à `apps/go-api/internal/analysis/sql_fragments.go` — créé
pourtant exactement pour ça. `queries_squad.go` x10, `queries_match.go` x9,
`queries_career_encounters.go` x9... Un seul site qui oublie le COALESCE recrée le bug TZ connu.
-> Helper `SQLStartTimeUTC(alias)`, migration mécanique, garde-rail grep en test.

**A11 — `SQLIsBot` centralisé puis ignoré : 36 littéraux résiduels dans 19 fichiers.**
`sql_fragments.go:26-30` date de la revue « DETTE 11 — IsBot répété 8 fois » ; la dette a été
x4 depuis, avec 1 seul consommateur réel de la constante.
-> Substitution mécanique + test grep interdisant le littéral `LIKE 'bid(%` hors
`sql_fragments.go`.

**A12 — SynthesisPage n'a jamais été migrée sur le hook factorisé *depuis elle-même*.**
`apps/web/src/features/_shared/useLocalFilterBar.tsx` documente « pattern factorisé depuis
SynthesisPage » ; la page d'origine garde sa copie intégrale (8 useState pending/committed,
`EXPERIENCE_TO_CASCADE` et `setsEqual` identiques mot pour mot, `SynthesisPage.tsx:51-61` ===
`useLocalFilterBar.tsx:23-33`). ~250L supprimables.

**A13 — Formatters et helpers dupliqués malgré modules canoniques.**
4 définitions de `formatDate`, 4 de `formatPercent`, 3 sorties de durée concurrentes malgré
`lib/formatters/{date,duration,percent}.ts` ; côté Go, `strPtr` redéfini 5x hors tests (aucun
`Ptr[T]` générique) et `safeDiv` (`service/teammates_service_kpis.go:224`) réimplémente
`analysis.SafeRatio`. Icône SVG « ouvrir le match » copiée 9x dans 8 fichiers ; socle d'option
ECharts répété 7x dans `TimeseriesFormCharts.tsx` ; helpers couleur win/loss/ratio recodés 4x
dans palmares/career.
-> Compléter les modules canoniques et purger les copies ; `Ptr[T]` générique côté Go ;
composant `<OpenMatchIcon/>`.

### 4.4 React / conformité front

**A14 — `LeaderboardBlock` : table native de 13 colonnes triables avec ~60L de plomberie manuelle.**
`apps/web/src/features/leaderboard/LeaderboardBlock.tsx:325` — violation directe de la règle
TanStack Table (8 autres tables du repo font correctement).

**A15 — `SquadLayout` : god component de ~630L.**
`apps/web/src/features/squad/SquadLayout.tsx:97-729` — deep-links, localStorage, double sync
store, pending/committed, ré-ancrage, rendu (7 useEffect dont 6 avec `eslint-disable
exhaustive-deps`). Y compris un side effect (écriture localStorage) **dans un updater setState**
(l.143-149 — impur, rejoué en StrictMode).
-> 3 hooks extraits (`useSquadSessionSync`, `useSquadCompositionAnchor`,
`useSquadPendingFilters`) + `SquadFilterBar` ; l'écriture localStorage passe en `useEffect`.

**A16 — Query keys hors registre.**
11 clés inline en littéral (`['combatYieldHistory',...]` dans un fichier qui importe déjà
`queryKeys` ; invalidation par littéral `admin/data-quality/mutations.ts:22` — désynchronisable
silencieusement), 26 `invalidateQueries` sans clés centralisées, 7 registres locaux dispersés
(`squadKeys`, `prestigeKeys`, `watcherKeys`...) consommés cross-feature.
-> Rapatrier dans `lib/query/keys.ts` ou documenter l'exception dans le skill frontend-patterns.

### 4.5 i18n

**A17 — Chemins d'erreur auth/setup/onboarding massivement FR monolingues.**
`features/auth/XboxLoginPage.tsx` (~12 strings, l.106-418, alors que le manifest est importé et
utilisé 10 lignes plus loin), `features/setup/StepDeviceCode.tsx:129-142`,
`StepInitialSync.tsx:81-87`, `features/auth/RegisterPage.tsx:71-79`,
`features/onboarding/OpenSpartanImportCard.tsx:346-365` (doc assumant « French sentence »).

**A18 — Surfaces de données FR monolingues.**
9 colonnes du scoreboard hardcodées (« Folie meurt. », « Tirs à la Tête »...) dans
`features/match-view/MatchScoreboard.tsx:51-76` — la moitié des colonnes passe par `t.sbCol*`,
migration à finir ; heatmap Explorer entière (`ExplorerActivityHeatmapChart.tsx:18-84` : jours,
axes, tooltip) ; 24 `toLocaleString('fr-FR')` figés dans 7 fichiers synthesis/career/session ;
« Par carte »/« Par mode »/« Analyser » dans SynthesisPage et SquadLayout.

**A19 — Anglicisme « streak » dans les textes FR** (préférence utilisateur documentée).
`features/notifications/i18n.ts:171,201,283,286` (« ta streak shield est prête »),
`features/ascension/i18n.ts:162-169` (« Aucune streak en cours... pour démarrer une **série** ! »
— le bon terme est dans la même phrase), + `settings/i18n.ts:456` (« ranked/casual »),
`match-view/i18n.ts:239`, `squad/i18n.ts:429`, `help/i18n.ts:200`.
-> « série » partout ; le glossaire officiel (`help/i18n.ts:326`) le confirme déjà.

### 4.6 Gouvernance

**A20 — Les seuils CLAUDE.md ne sont plus protégés côté Go.**
`apps/go-api/.golangci.yml` : funlen 80->100, lll 100->220, argument-limit 5->7 **puis
neutralisé** par l'exclusion `text: 'argument-limit'` (l.117-118 — la règle est morte), et
`funlen`/`gocyclo` désactivés sur `internal/sync/`, `internal/analysis/`, `internal/service/`,
`internal/api/handlers/`, `server.go` — précisément les packages qui concentrent les 168
fonctions > 80L. Contrairement au Python (`size_baseline.txt` = dette gelée fichier par fichier),
l'exemption par répertoire laisse la dette croître silencieusement. Le commentaire l.92
(« dossier dupliqué accidentel ») est en outre périmé — le dossier n'existe plus.
-> Remplacer les exclusions par répertoire par une baseline gelée (`--new-from-rev` ou baseline
explicite) + réactiver argument-limit.

---

## 5. Findings MINEURS (condensés)

- **Débris de splits de fichiers Go** : doc + `//nolint` orphelins en fin de
  `service/teammates_squad_charts_impact_events.go:444-463` (la fonction documentée vit dans
  `teammates_squad_charts_weapons_perf.go:18`, sans doc) ; doc-comments désassortis dans
  `api/post_sync_deltas.go:133,404` ; commentaires nolint « brouillés » (ordre de lignes
  inversé) en 5+ sites dont le doc de `NewRouter`.
- **`strings.Title` déprécié** — `sync/engine.go:191`, remplaçable par un map à 2 entrées.
- **`EmitPostSyncDeltas` 247L** = 12 blocs quasi identiques -> table-driven (~60L) :
  `api/post_sync_deltas.go:156-402`.
- **~15 blocs `g.Go` copiés** dans `service/match_view_data_loaders.go:68-256`, avec mélange
  `slog.Warn`/`slog.WarnContext` alors que `gctx` est disponible.
- **Magic numbers ciblés** : paliers 0.05/0.01 des deltas (`post_sync_deltas.go:347-396`),
  `rows[:50]` (`home_canonical_highlights.go:214`), fenêtre `15` min
  (`media_repo_filters.go:183`), `outcome = 2` en dur (1 site :
  `api/post_sync_progression_queries.go:301`, bypass du seam `outcomeSQLEq`).
- **Branchement slug résiduel** : `sync/comeback.go:33-38` route un ID de médaille par
  comparaison `DefaultSlug` au lieu du mapping TOML — dette connue, à basculer à l'activation
  du 2e titre.
- **Web** : bypass ECharts unique (`palmares/CumulativeFragGapChart.tsx:23`) ;
  `isLoading -> return null` (flash blanc) sur SynthesisPage:631/HomePage:123 ; listener clavier
  aux deps approximées (`media/CoverFlowModal.tsx:474-482`) ; `MatchCard` ~470L
  (`components/ui/match-card.tsx:65-537`) ; SessionComparePage 15 sections Card répétées
  (absorbé par A8 si suppression) ; dictionnaire i18n de 60L inline dans
  `components/shell/PeriodSessionRail.tsx:55-114` + `formatDateShort` homonyme du canonique ;
  `joinAndSort` à renommer (`squad/charts/mapPerfVsHistoryChart.ts:56`) ; aria-label FR figé
  (`notifications/NotificationItem.tsx:72`) ; ~33 ternaires `locale === 'en' ? ... : ...` inline
  contournant les i18n.ts existants ; matching de filtres couplé aux libellés FR
  (`_shared/useLocalFilterBar.tsx:234-236`) ; labels de série/titre hardcodés
  (`TimeseriesFormCharts.tsx:207`, `LeaderboardBlock.tsx:435`, `prestige/LeaderboardPP.tsx:82-87`).
- **Guards legacy sans date d'expiration** : tokens Phase 5 (`platform/auth/cli_refresh.go:29-53`,
  `migration.go:93-103`), `LEVELUP_SYNC_PIPELINE=v1` (`scheduler/auto_sync.go:433`),
  `PERSIST_BATCH=0` (`sync/engine_batch_path.go:16-18`) — le modèle à copier existe :
  `platform/duckdb/shared_reader_legacy.go:30-34` (retrait daté >= 2026-Q3 + critère expvar
  mesurable).

---

## 6. TOP 10 priorisé (impact x effort)

| # | Action | Réf | Effort |
|---|---|---|---|
| 1 | Corriger l'échelle perf-tier inversée (bug visuel actif) | C1 | ~15 min |
| 2 | Corriger les badges outcome cassés en EN (+ `fr-FR` figé) | C2 | ~30 min |
| 3 | Rerouter/retirer `RunBackfillLUSR` v1 (intégrité `match_skill_rank`) | C3 | ~1 h |
| 4 | Réécrire les 4 docs inversées des flags `PERSIST_BATCH`/`SYNC_PIPELINE` | C4 | ~15 min |
| 5 | Purge code mort : home legacy + session-compare + SquadV2 + NotifyNewMedia (~40 fichiers, décision produit pour session-compare) | A7-A9 | 1-2 j |
| 6 | Campagne i18n « chemins d'erreur » (auth/setup/onboarding) + « streak -> série » — purge des warns avant le passage de `no-hardcoded-strings` en `error` | A17-A19 | 1 j |
| 7 | `SQLStartTimeUTC` + adoption `SQLIsBot` + garde-rails grep | A10-A11 | 1/2 j |
| 8 | Migrer SynthesisPage sur `useLocalFilterBar` (~250L) + compléter `lib/formatters` et purger les copies | A12-A13 | 1 j |
| 9 | Extraire `BackfillOrchestrator` du handler (368L -> service table-driven) | A3 | 1 j |
| 10 | Rétablir la gouvernance lint Go (baseline gelée au lieu d'exclusions par répertoire) | A20 | 1/2 j |

Le chantier `NewRouter`/`engine.run` (A1, A4) est volontairement hors TOP 10 : impact élevé mais
risque de régression réel — à planifier comme tâche dédiée avec la grille `plan-review`, pas en
opportuniste.

## 7. Recommandations

1. **Traiter la « non-repropagation » comme une classe de dette à part.** Le pattern dominant
   n'est pas l'absence de factorisation mais l'abandon à mi-chemin : `sql_fragments.go`,
   `lib/formatters/`, `useLocalFilterBar`, la migration des colonnes du scoreboard — chaque
   helper canonique créé devrait embarquer son garde-rail (test grep interdisant l'ancien
   littéral) le jour de sa création, sinon la dette re-croît (le prédicat bot est passé de 8 à
   36 copies *après* centralisation).
2. **Un code mort testé est le pire code mort.** Les 4 clusters morts ont tous des tests verts
   qui entretiennent l'illusion (et `processMatch` fait tester un chemin que la prod n'exécute
   plus). Ajouter à la `delivery-checklist` une vérification « ce que je supprime du routing/des
   callers, je le supprime du code ».
3. **Documenter les kill-switches comme des kill-switches.** C4 montre le mécanisme de drift :
   le flag naît opt-in, devient default-ON, la doc reste. Convention suggérée : tout flag
   d'activation porte dans son commentaire la date du basculement de défaut + la date cible de
   retrait (le modèle `shared_reader_legacy.go` est excellent — le généraliser).
4. **Côté front, verrouiller les deux règles déjà outillées** : passer
   `@levelup/no-hardcoded-strings` en `error` (après purge du TOP 10 #6) et ajouter une règle
   ESLint sur les `queryKey:` littéraux hors `lib/query/keys.ts`.
5. **Points forts à préserver explicitement** (ils rendent l'audit possible) : commentaires
   forensiques datés citant ADR/incidents, event IDs slog corrélables, seam `outcomeSQLEq`,
   capability-gating sans slug, exceptions couleur annotées `color-allow`, typage
   `Record<Locale, T>` qui garantit la parité FR/EN par compilation. La culture de refactoring
   traçable (HomePage P8.4, promotion CoverFlowModal) est réelle — c'est l'étape « repropager
   puis supprimer l'original » qui manque au rituel.

## 8. Observations positives (calibrage)

1. **Discipline slog exemplaire** : zéro `fmt.Print*`/`log.Printf` dans `internal/` (prod) ;
   event IDs corrélables cross-module (`logging.WithEvent`).
2. **Title-agnostic réellement tenu dans les data-paths** : pas un seul `slug == "halo_infinite"`
   littéral en prod ; gating par capabilities, sélection registry-driven.
3. **Zéro fetch sauvage côté web** : les 4 `fetch()` hors client API sont justifiés ET documentés
   (dont le commentaire sécurité de feedback-drawer sur le leak cookies vers GitHub).
4. **Système de couleurs très bien tenu** : quasi aucun hex hors exceptions annotées
   `color-allow` ; seuils métier centralisés avec snapshot CI anti-dérive (c'est précisément ce
   qui rend C1 détectable et grave : une seule copie locale a suffi à inverser une sémantique).
5. **Discipline de types front** : 331 imports de `lib/api/types` dans `features/`, zéro
   redéclaration de DTO API ; parité i18n FR/EN garantie par le compilateur.
6. **Robustesse défensive du pipeline sync** : recover+stack post-sync, réconciliation
   « bénéfice du doute » post-drain, fail-fast RC-A — les modes de défaillance sont anticipés.
7. **Modèle de retrait de dette à généraliser** : `shared_reader_legacy.go:30-34` (plan de
   suppression daté + critère expvar mesurable) — exactement ce que la règle projet demande.

---

*Méthodologie : 4 agents parallèles (Go ~56 lectures/greps, React ~47, i18n ~74, duplication ~76
vérifications croisées) + scans mécaniques awk/grep (tailles fichiers/fonctions, couleurs, query
keys, sql.Open, os.Getenv). Findings dédupliqués et contre-vérifiés avant consolidation ;
2 claims centraux (session-compare morte, cluster home legacy mort) revalidés indépendamment.*
