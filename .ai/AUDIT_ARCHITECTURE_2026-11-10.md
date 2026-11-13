# Audit architecture et état du code — 2026-11-10

> Registre d'audit (skill `adversarial-audit`). L'audit ne corrige rien : il produit ce registre,
> qui devient un plan. Commit audité : `afb963994c` (tête de `release/v8.1.2`).
> **Statuts à la clôture (2026-11-13)** : chaque constat est statué dans la section « Statut des constats »
> en fin de fichier (plan `.ai/PLAN_ARCHITECTURE_FRONTIERES_2026-11-10.md`, étape 8.5). Les champs
> « Décision » des constats ci-dessous sont ceux du jour de l'audit.

## Cadrage

- **Demande** : (1) revoir ce qui a été fait depuis la dernière grosse revue ; (2) architecture,
  séparation des couches et des responsabilités, facilité à changer l'UI ou une technologie,
  organisation du code, des dossiers et du traitement.
- **Dernières grosses revues retrouvées** : 2026-07-02 (architecture Go), 2026-07-17 (revue de
  code), 2026-08-06 (audit de `main`, v7.2.0 vers `634e079f2`), 2026-09-05 (audit `feat/v75`
  depuis v7.3.0, 15 auditeurs). Référence retenue : 2026-09-05 (`736ccf3c3`).
- **Conséquence de méthode** : depuis `736ccf3c3`, 6 162 commits et la quasi-totalité du code
  touchée. Une relecture de diff n'a pas de sens à cette échelle : la partie « depuis la dernière
  revue » est traitée comme trois audits de l'état actuel sur des axes transverses (A2, A7, A1),
  dont deux (A5, A7) n'avaient jamais eu d'auditeur dédié.
- **Méthode** : campagne « périmètre × axe », 8 auditeurs Sonnet en contexte frais, lecture seule,
  aveugles les uns aux autres. Vérification des constats P1 par le superviseur, sur pièces
  (fichier et ligne rouverts, commande de reproduction rejouée). Aucun `go test`, `go build`,
  lint ni `vitest` exécuté : tout est établi par lecture, `git grep` et `go list`.
- **Périmètres et axes passés**

| Réf. | Périmètre | Axe |
|---|---|---|
| API | `internal/api/` (handlers, middleware, wire, racine) | A4 frontières de couches côté HTTP |
| STO | `service`, `analysis`, `domain`, `port`, `prestige`, `progression`, `notifications`, `notify`, `media`, `scheduler` | A4 étanchéité vis-à-vis du moteur de stockage |
| DEP | graphe d'imports des paquets de premier niveau de `internal/` et des modules `libs/` | A4 direction des dépendances |
| WEB | `apps/web/src/{features,components,routes}` | A4 frontières entre modules du front |
| TEC | `apps/web/src` | un point d'entrée par technologie tierce |
| ART | `persist`, `sync`, `migration`, `media` | A2 invariants des écritures DuckDB |
| FLG | `internal/`, `cmd/server`, `cmd/levelup`, `web/lib`, `web/app` | A7 flags et compatibilité |
| MORT | `cmd/` (180 binaires) et 16 paquets satellites | A1 code mort |

- **Axes NON passés** : A3 multi-titre (pas d'auditeur dédié ; vu en partie par DEP-1), A5
  duplication, A6 erreurs avalées (vu seulement sur les chemins d'écriture), A8 couverture de
  tests, A9 correction des données, A10 couleurs et i18n (tenus par des lints et garde-rails).
  **Périmètres non passés** : intérieur de `libs/film` (audit du 2026-09-24, revues de clôture du
  2026-11-09), coquille Rust `apps/desktop` (revues du 02 au 08/11), les quatre branches 8.1.2
  non fusionnées, la sécurité (audit du 2026-10-15).
- **Doctrine de référence** : CLAUDE.md (règles 5, 6, 7, 10, 11, 13 ; règles ART 1 à 7 ;
  anti-patterns 1, 2, 5, 7, 9, 10), skills `arch-rules` et `frontend-patterns`, ADR 0001, 0023,
  0030, 0034, 0039, 0041, 0042, 0044, 0045, 0046, 0047.
- **Dette assumée, non remontée** : baseline lint (tailles), exemptions commentées, entrées
  d'allowlist datées et justifiées, pattern SELECT-then-UPDATE-or-INSERT des bases joueur
  héritées, items de `.ai/REGISTRE_REPORTS.md` et `.ai/BACKLOG.md`.

## Bilan

| Gravité | Nombre | Lecture |
|---|---|---|
| P0 | 0 | aucune corruption, perte d'écriture ni faille démontrée |
| P1 | 25 | règle écrite violée ; un seul peut priver l'écran d'une donnée (ART-2, noms de joueurs) |
| P2 | 22 | dette réelle, coût futur |
| Écartés ou requalifiés | 12 | section dédiée |

Constat d'ensemble : là où un garde-rail existe, la règle tient (allowlists ART vides, 0 `ON
CONFLICT DO UPDATE`, `domain` et `port` propres, briques autonomes, pont Tauri fermé). Tous les
P1 se trouvent là où aucun garde-rail ne regarde. La liste des garde-rails manquants est en fin
de registre.

## Réponse à la question « changer l'UI ou une technologie »

| Si on change... | État | Ce qui rend le changement facile | Ce qui le renchérit |
|---|---|---|---|
| Tout le front (React vers autre chose) | Bon | Contrat OpenAPI généré et tenu par deux garde-rails ; le serveur renvoie des libellés résolus ; 0 logique métier dans la coquille | 31 types d'API écrits à la main doublent le contrat (TEC-2) |
| La bibliothèque de graphes | Moyen | 0 import d'exécution d'ECharts hors `components/charts/` ; un seul point de montage | 67 fichiers de `features/` construisent eux-mêmes des options ECharts, 72 importent son type (TEC-P2-3) : le changement touche environ 70 fichiers |
| Le moteur de base (DuckDB) | Moyen | `service/` n'importe jamais `platform/duckdb` (0 sur 299) ; `domain`, `port` sans type du moteur dans 130 interfaces sur 134 | SQL hors infrastructure : `progression/profile` (13 requêtes), `scheduler` (9 fichiers), `api/wire` (9 fichiers), 8 handlers, 2 fichiers d'`analysis` ; fragments SQL au dialecte DuckDB dans `analysis` |
| Le cadre HTTP (chi, Huma) | Moyen | 0 SQL et 0 calcul statistique dans `handlers/` ; `middleware/` propre | le cycle de vie des jobs de sync vit dans 4 handlers (API-2) ; `api/wire` porte du SQL, des écritures et un calcul (API-3) |
| Ajouter un titre | Moyen | adapters, capabilities, TOML, ratchet anti-slug | 15 fichiers de `service/` importent `games/halo_infinite` en direct (DEP-1), sans ratchet |
| La coquille bureau (Tauri) | Très bon | pont fermé dans `lib/desktop/` (2 fichiers), 0 import `@tauri-apps` ailleurs, aucune comparaison sur le profil de déploiement | rien |
| Les briques `libs/` | Très bon | 0 dépendance vers l'application, rang `film` → `himap` → `gamefiles` respecté, couches internes du décodeur fermées par le compilateur | rien |
| Le client HTTP du front | Bon | `lib/api/client` porte base, cookies, en-tête de titre | 7 `fetch` hors client, une base URL recopiée (TEC-P2-2) |

Organisation : le découpage en couches est le bon et il est documenté. Trois foyers de dérive :
(1) `api/wire` et huit handlers font office de seconde couche service (la « dérive n°1 » de
l'audit du 2026-07-02 a quitté la racine `api/` pour `wire/`) ; (2) `cmd/` porte 106 binaires sur
180 sans utilisateur, soit 30 % de ses lignes ; (3) 9 interrupteurs sans échéance.

## Constats retenus

### P1 sur les chemins de données

#### [P1] ART-5 — Probabilité de victoire : statistique abandonnée, code toujours en place
- Où : capability `CapExpectedWinProb` (`domain/title/registry.go:176-187`, « REMISÉE le 2026-09-07, non déclarée par aucun titre ») ; 34 fichiers Go non-test portent encore `ExpectedWinProb` / `expected_win_prob` ; côté web `match-view/MatchStatCards.tsx:222-395`, `squad/SquadSynergyHistoryTable.tsx:309`, `timeseries/TimeseriesPage.progression.tsx:77-99`, `session-detail/SessionMatchesTable.tsx:114`, `lib/winProbCategory.ts`, `lib/capabilities/`. La requête `queries_squad.go:10` (`MAX(expected_win_prob) FROM match_skill_rank`, lecture brute) tourne à chaque chargement d'Escouade pour une colonne que personne n'affiche.
- Règle : règle 11 (« pas de flag qui laisse une feature OFF pour plus tard ») ; règle 7 (0 code mort).
- Conséquence : rien de faux à l'écran (aucun titre n'accorde la capability). Coût : une requête inutile par chargement, une exemption de lecture brute non datée (`no_raw_rating_reads_test.go:81`), et du code d'affichage entretenu pour rien.
- Reproduction : `git grep -n 'expected_win_prob' -- config/` (vide) ; `git grep -lE 'ExpectedWinProb|expected_win_prob' -- apps/go-api/internal ':!*_test.go' | wc -l` (34)
- Vérification : tient. Requalifié le 2026-11-10 sur message de l'utilisateur (« stat abandonnée, pas fiable du tout à l'usage ») : le constat initial (« valeur périmée servie à Escouade ») est sans objet, la colonne n'étant affichée nulle part.
- Traitement proposé : retirer l'affichage, la capability, les lecteurs et l'exemption ; statuer sur le calcul et la colonne stockée. Décision : abandon acté par l'utilisateur, retrait à planifier (chantier « ménage »).

#### [P1] ART-2 — Alias de joueurs perdus sans trace quand leur écriture échoue
- Où : `apps/go-api/internal/sync/convergence.go:214` (marqueur terminal `psa_checked_at` posé avant le flush), `:232` (échec ni journalisé ni compté), `engine_postsync_scoring.go:238` (journal « retentés au prochain cycle »).
- Règle : anti-pattern 10, erreur avalée sur un chemin d'écriture.
- Conséquence : le match est marqué traité avant l'écriture des alias ; si la fenêtre d'écriture est indisponible ou si un upsert échoue, ces alias ne sont jamais retentés, contrairement à ce que dit le journal.
- Reproduction : `git grep -nE 'err == nil \{|retentés au prochain cycle|psa_checked_at, stage' -- apps/go-api/internal/sync/convergence.go apps/go-api/internal/sync/engine_postsync_scoring.go`
- Vérification : tient.
- Traitement proposé : compter et journaliser l'échec ; poser le marqueur après le flush ou corriger le message. Décision : à trancher.

#### [P1] ART-1 — Intensités de match non écrites sans trace
- Où : `apps/go-api/internal/sync/engagement.go:654` (`_ = persistMatchIntensity(...)`).
- Règle : anti-pattern 10.
- Conséquence : si l'écriture échoue, l'engagement se calcule sans ces intensités et rien ne le signale.
- Reproduction : `git grep -n '_ = persistMatchIntensity' -- apps/go-api/internal/sync`
- Vérification : tient. Le commentaire dit « best-effort » mais n'apporte ni log ni compteur.
- Traitement proposé : journal et compteur. Décision : planifiable.

#### [P1] STO-1 — L'import OpenSpartan écrit dans la base du joueur depuis un service
- Où : `apps/go-api/internal/service/openspartan_post_import_service.go:217` (`INSERT INTO player_match_enrichment`), appelé depuis le handler `openspartan_import.go:258`.
- Règle : règle ART 1 (écriture per-match par `persist.BatchBuilder`) ; « SQL inline dans un service ».
- Conséquence : le persister existant (`persist/post_sync_enrichment_persister.go`) est contourné ; la boucle s'arrête à la première erreur et laisse les matchs suivants sans ligne `live`.
- Reproduction : `git grep -nE 'INSERT INTO|QueryContext|PrepareContext' -- apps/go-api/internal/service/openspartan_post_import_service.go`
- Vérification : tient pour le contournement. La partie « sans bail, risque de corruption » est requalifiée (voir écartés).
- Traitement proposé : passer par le persister. Décision : planifiable.

### P1 de frontières de couches (Go)

#### [P1] API-1 — Des handlers construisent des dépôts DuckDB (trouvé par deux auditeurs)
- Où : `api/handlers/progression.go:155,180,187,237,242`, `campaign.go:287-288`, `player_profile.go:113,143,153`, `settings_data.go:98-99`.
- Règle : « le handler consomme l'interface `port.XService`, jamais le type concret » ; « accès DB dans un handler ».
- Conséquence : ces handlers ne se testent pas avec un faux port ; `progression.go:236-262` porte la jointure catalogue × jalons. Aucun garde-rail n'interdit `platform/duckdb` dans `handlers/`.
- Reproduction : `git grep -nE 'duckdb\.New[A-Z]|duckdb\.(ApplyMemoryLimit|SetMemoryLimit)' -- apps/go-api/internal/api/handlers ':!*_test.go'`
- Vérification : tient.

#### [P1] API-2 — Des handlers construisent le moteur de sync et le lancent
- Où : `api/handlers/backfill.go:148-159`, `sync_handler.go:192,242,465`, `sync_handler_align.go:133`, `settings_sessions_recalc.go:81`.
- Règle : « appelle service via port interface » ; pas de runner dans la couche HTTP.
- Conséquence : le cycle de vie des jobs (délais, libération du verrou, post-sync) vit dans la couche HTTP ; la recherche de profil y est recopiée trois fois (règle 6).
- Reproduction : `git grep -nE 'go_sync\.(NewSyncEngineForTitle|RecalculatePlayerSessions)|livesync\.RunnerForTitle|runner\.RunDelta' -- apps/go-api/internal/api/handlers ':!*_test.go'`
- Vérification : tient.

#### [P1] API-3 — SQL, écritures et calcul dans `api/wire`
- Où : 9 fichiers de `api/wire/` (39 occurrences), dont `registry_catalog_expand.go:79-139` (INSERT, CREATE TABLE, UPDATE), `post_sync_deltas_snapshot.go`, `registry_weapon_coverage.go` ; calcul KPM et PSPM dans `post_sync_progression_queries.go:231-236`.
- Règle : « racine `api/` réservée à la DI, ne JAMAIS y ajouter de SQL, de logique métier ou de runner » ; « calcul : `analysis/` ».
- Conséquence : le schéma est connu de la couche de câblage ; le KPM par match est calculé hors `analysis/`, une correction de formule peut oublier ce site.
- Reproduction : `git grep -cE '(SELECT |INSERT INTO|UPDATE .* SET|CREATE TABLE)' -- 'apps/go-api/internal/api/wire/*.go' ':!*_test.go'`
- Vérification : tient.

#### [P1] API-4 — La racine `api/` porte des handlers et une boucle de fond
- Où : `api/commendation_handler.go:19-89`, `spa_handler.go`, `static_files.go`, `media_audience.go:26-71`, `server.go:580-595` (purge des sessions toutes les 6 h).
- Règle : même règle que API-3.
- Conséquence : le garde anti-traversée de chemin de `commendation_handler.go` vit hors de `handlers/` et de ses tests de couche.
- Reproduction : `git grep -nE 'ServeHTTP|time\.NewTicker|go func' -- 'apps/go-api/internal/api/*.go' ':!*_test.go'`
- Vérification : tient.

#### [P1] DEP-1 — 15 fichiers de `service/` importent du code propre à Halo Infinite
- Où : `service/home_service.go:9,85` (`haloadapter.DefaultHaloProvider`, singleton), `catalog_fetcher_service.go:20`, `world_stats_enricher.go:17`, `replay_*_labels.go`, `tactical_callouts.go:20`, `replayview/convert_*.go`, `squadagg/*.go`.
- Règle : « un service obtient ses adapters via le Resolver, jamais en important `games/halo_infinite` ».
- Conséquence : `HomeService` est câblé sur le fournisseur Halo Infinite ; un nouveau titre oblige à retoucher chaque service. Le ratchet `no_title_package_in_analysis_test.go` ne couvre que `analysis/`.
- Reproduction : `git grep -n -E '"levelup/go-api/internal/games/(halo_infinite|halo_5)[/"]' -- apps/go-api/internal/service ':!*_test.go'`
- Vérification : tient (15 lignes).

#### [P1] STO-2 — `progression/profile` : 13 requêtes SQL dans un paquet de service
- Où : `internal/progression/profile/{queries,service,activity,first_events,lusr_components_query}.go` ; `service.go:34-35` tient des `*duckdb.DB` concrets.
- Règle : « SQL inline dans un service » ; « mock `port.Repository` via interface ».
- Conséquence : aucun port substituable ; tout test exige une base réelle ; `queries.go:508` recopie `scanTemplate` de `platform/duckdb`.
- Reproduction : `git grep -cE 'QueryContext\(|QueryRowContext\(' -- apps/go-api/internal/progression/profile ':!*_test.go'`

#### [P1] STO-3 — Les sondes de santé du scheduler portent leur SQL
- Où : `internal/scheduler/data_health_check.go:320-390`, `replay_purge_cron.go:285-299`, `data_health_{lusr,kill_summaries,passes}.go` ; 9 fichiers importent `platform/duckdb`.
- Règle : « accès DuckDB direct hors adapter/repo ».
- Conséquence : un renommage de table casse une sonde, dont le compteur reste alors à 0 (les erreurs vont dans `ProbeErrors`).
- Reproduction : `git grep -nE 'SELECT|OpenReadOnly|OpenReadForQuery' -- apps/go-api/internal/scheduler/data_health_check.go apps/go-api/internal/scheduler/replay_purge_cron.go`

#### [P1] STO-4 — `analysis` exécute des requêtes (trouvé par deux auditeurs)
- Où : `internal/analysis/weapon_evidence.go:24,40,47`, `internal/analysis/prestigetuning/collect.go:5,84-130`.
- Règle : « `analysis/` : 0 DB, 0 HTTP, 0 side-effect ».
- Conséquence : seul `analysis/tactical` est protégé par un test de pureté ; rien n'empêche la requête suivante.
- Reproduction : `git grep -lE '"database/sql"' -- apps/go-api/internal/analysis ':!*_test.go'`

### P1 d'écritures et de garde-rails

#### [P1] ART-3 — L'allowlist datée de `OpenReadWrite` (ADR 0030 D-3) n'est tenue par aucun test
- Où : `archlint/no_bare_duckdb_rw_open_test.go` ne recense que `sql.Open` et `NewConnector`. Sites : `persist/combined_persister.go:187`, `sync/schema.go:476,507`, `assets/store_duckdb.go`, `ops/monitoring_store.go`.
- Conséquence : un nouvel appel `duckdb.OpenReadWrite` sur une base critique passe la CI.
- Reproduction : `git grep -nE 'duckdb(pkg)?\.OpenReadWrite\(' -- apps/go-api/internal ':!*_test.go'`

#### [P1] ART-4 — Allowlists sans date, dont une au motif périmé
- Où : `sync/shared_write_guard_test.go:55-57,64`, `sync/append_only_state_guard_test.go:210-218`, `platform/duckdb/no_raw_rating_reads_test.go:78-82`.
- Règle : règle ART 3, « ne jamais allowlister sans justification datée ».
- Conséquence : `shared_write_guard_test.go:64` justifie `writes.go` par « DELETE+INSERT sérialisé » alors que l'écriture est devenue append-only ; `no_raw_rating_reads_test.go` exempte par nom de base de fichier, donc tout fichier homonyme.
- Reproduction : `git grep -n 'DELETE+INSERT sérialisé' -- apps/go-api/internal/sync`

### P1 d'interrupteurs

#### [P1] FLG-1 — `LEVELUP_USE_SHARED_PROVIDER` : échéance de retrait dépassée
- Où : `platform/duckdb/shared_reader_legacy.go:30` (« ≥ 2026-Q3 »), lu par `os.Getenv` dans `cmd/server/boot_providers.go:71`.
- Règle : règle 11. Ce fichier est le modèle cité par CLAUDE.md.
- Conséquence : le lecteur hérité reste branché avec le défaut qu'il devait faire disparaître.

#### [P1] FLG-2 — Réparation automatique des trous de note livrée éteinte depuis le 2026-07-21
- Où : `scheduler/data_health_lusr.go:36-48` (`LEVELUP_LUSR_AUTOHEAL_ENABLED`, défaut OFF, pas de date cible).
- Règle : règle 11, « pas de flag qui laisse une feature OFF pour plus tard ».
- Conséquence : seule l'alerte existe ; le rejeu reste manuel.

#### [P1] FLG-3 — Sept interrupteurs sans cycle de vie
- Où : `LEVELUP_POSTSYNC_BURST` (`sync/shared_access.go:33-39`, lu à deux endroits), `LEVELUP_LUSR_V2_MODE_COUPLING` et `LEVELUP_LUSR_V2_SQUAD_OFFSET` (`sync/skill/skill_v2_cross_mode.go:31`, `skill_v2_squad.go:35`, relus pendant le calcul), `LEVELUP_SYNC_V2_DRYRUN` (`cmd/server/sync_v2_wiring.go:86`), `LEVELUP_WATCHER_BROADCAST` (`cmd/server/boot_watcher.go:364`), `LEVELUP_PERSIST_NO_FETCH_CACHE` (`sync/engine.go:489`), `PRESTIGE_ENABLED` (`prestige/sync_hook.go:24-32`).
- Règle : règle 11 (date de bascule, date cible de retrait, critère) ; lecture au boot par `internal/config`.
- Reproduction : `git grep -nE 'os\.(Getenv|LookupEnv)\(' -- apps/go-api/internal ':!*_test.go' ':!apps/go-api/internal/config'`

#### [P1] FLG-4 — Deux commentaires décrivent l'ancien défaut
- Où : `scheduler/auto_sync.go:205` (« Activée via LEVELUP_PERSIST_BATCH_ASYNC=1 », alors que le défaut est ON), `cmd/server/main.go:16` (`LEVELUP_LOG_JSON` « défaut: text », le défaut réel est `compact`).
- Règle : anti-pattern 9.

#### [P1] FLG-5 — Code mort dont la date de péremption est dépassée
- Où : `platform/duckdb/home_repo_playlist_ranks.go:120,144,162` (`nolint:unused`, « péremption 2026-06-22 »), `service/career_live_service.go:539` (`errNoFallback`, « d'ici 2026-Q3 »), `api/handlers/auth.go:76` (`AuthHandler.WithUserStore` sans appelant).
- Règle : règle 7.

### P1 du front

#### [P1] WEB-1 — Le contrôle des imports entre features a trois angles morts
- Où : `tools/lint-cross-feature-imports.mjs:283` (le motif `[a-z0-9-]+` ignore `_shared`), `:306` (le marqueur `cross-feature-allow` exempte le fichier entier, 10 fichiers), et aucun contrôle des imports relatifs.
- Faits qui en découlent : `features/_shared/useLocalFilterBar.tsx:31,34,35` importe `filters`, `squad` et `explorer` ; `features/match-replay/model/replayTimelineTracksLogic.ts:26` importe `../../match-view/scoreTimelineKind`, hors des quatre modules autorisés ; `ascension/AscensionCoachingTab.tsx:20` importe `settings/queries` sous un marqueur qui ne le justifie pas.
- Règle : « feature = module auto-suffisant » ; import cross-feature = défaut hors allowlist.
- Conséquence : cycles `squad` → `_shared` → `squad` et `explorer` → `_shared` → `explorer` invisibles au lint.
- Reproduction : `git grep -nE "['\"]@/features/[a-z]" -- apps/web/src/features/_shared | grep -v "\.test\."`
- Vérification : tient.

#### [P1] WEB-2 — Deux hooks de données recopiés dans des composants
- Où : `features/home/HomePrestigeSection.tsx:76-81` (double de `prestige/hooks/useChallenges.ts`), `features/ascension/PrestigeSquadProgress.tsx:78-84` (double de `usePrestige`).
- Règle : structure de feature (`queries.ts`).
- Conséquence : même clé de cache, options divergentes ; la copie de l'Accueil n'a pas le garde `enabled` et peut partir avec un slug vide.
- Reproduction : `git grep -nE "\buse(Query|Queries)\(" -- 'apps/web/src/features/*.tsx'`

#### [P1] TEC-1 — Deux graphes ne suivent pas un changement de palette d'accessibilité
- Où : `features/palmares/CumulativeFragGapChart.tsx:117`, `components/charts/OutcomeSequenceTape.tsx:342` (9 consommateurs).
- Règle : ADR 0001 et README des wrappers (« Wrap in `<ChartCard>` »).
- Conséquence : ils montent `LazyECharts` sans `ChartCard` et n'écoutent que le thème ; après un changement de palette, leurs couleurs restent celles de l'ancienne jusqu'au remontage.
- Reproduction : `for f in $(git grep -lE "<LazyECharts" -- apps/web/src ':!*.test.*'); do echo "$f $(grep -c useColorPaletteVersion $f)"; done`
- Vérification : tient (0 écoute de la palette dans les deux fichiers).

#### [P1] TEC-2 — 31 types d'API écrits à la main doublent le contrat généré
- Où : `apps/web/src/lib/api/types.ts` (ex. `MatchScoreboardRow`, 53 importeurs ; `MatchHistoryRow`, 0 importeur).
- Règle : `generated.ts` est la source du contrat ; règle 6.
- Conséquence : le garde `response-types.guard.test.ts` ne regarde que les noms en `Response`. Écarts constatés : nullabilité différente, champs du contrat absents du type manuel, champs manuels absents du contrat.
- Reproduction : `cd apps/web/src/lib/api && comm -12 <(grep -oE "^export interface \w+" types.ts | awk '{print $3}' | sort) <(grep -oE "^        [A-Za-z0-9_]+: \{" generated.ts | sed -E 's/^ +//;s/: \{//' | sort -u) | grep -vc 'Response$'` (rend 31)

### P1 de code mort

#### [P1] MORT-1 — 106 binaires de `cmd/` sur 180 sans utilisateur
- Où : `apps/go-api/cmd/`. Familles : `diag_*` 38 (9 205 L), `h5-*` 14 (2 946 L), `backfill*` et `lusr_v2_*` 15 (5 025 L), réparation, purge, migration et divers 25 environ (4 600 L), instruments cartes et film 8 (7 500 L, P2).
- Règle : règle 7, anti-pattern 1.
- Conséquence : 30 291 lignes sur 99 432 compilées, vérifiées et lintées à chaque CI ; les binaires de réparation ouvrent des bases en écriture et gardent une dérogation au garde-rail anti-ART.
- Reproduction : `for d in $(ls apps/go-api/cmd); do n=$(git grep -lE "$d([^A-Za-z0-9_-]|$)" -- Makefile scripts .github docs apps/desktop README.md CLAUDE.md .claude Dockerfile ':!docs/adr' | wc -l); echo "$n $d"; done | sort -n | head -110`
- Vérification : 8 binaires tirés au sort, 8 sans citation. Réserve : `REGISTRE_REPORTS.md:274` indique que certains `diag_*` servent à des diagnostics ponctuels lancés par des agents.
- Décision : à trancher par l'utilisateur, famille par famille.

#### [P1] MORT-2 — Code testé que rien n'appelle
- Où : `internal/presence/steam_poller.go` (159 L, `NewSteamPoller` sans appelant, tests verts) ; `internal/domain/chart/` (2 fichiers `Deprecated`, 0 importeur).
- Règle : règle 7, anti-pattern 1.
- Reproduction : `git grep -n "NewSteamPoller" -- apps libs ':!*_test.go'` ; `git grep -n 'internal/domain/chart"' -- .`

### P2 — dette

| Réf. | Où | Constat |
|---|---|---|
| API-P2-1 | `handlers/bootstrap.go:23`, `presence.go:30`, `site_import.go:30`, `openspartan_import.go:55`, `engagement.go:45`, `title_sync.go:53` | handler tenant le type concret d'un service au lieu d'un port |
| STO-P2-1 | `port/dbexecutor.go:22-26`, `dbwriter.go:20-22`, `media_missing.go:21`, `repository_data.go:19` | 4 interfaces sur 134 exposent `*sql.Rows` ou `*sql.Tx` ; `MediaService` ouvre et valide lui-même la transaction |
| STO-P2-2 | `service/openspartan_*.go`, `site_import.go` (14 signatures) | `*sql.DB` passé de fonction en fonction ; champ `legacyDB` réservé aux tests |
| STO-P2-3 | `service/media_service.go:10`, `media_index_service.go:13` | import du pilote `duckdb-go` devenu inutile ; commentaire faux |
| STO-P2-4 | 6 sites (`service/`, `prestige/`) | `sql.ErrNoRows` comparé dans la couche métier : contrat implicite absent de `port/` |
| STO-P2-5 | `analysis/sql_fragments.go`, `sql_liste.go`, `identity*.go` | fragments SQL au dialecte DuckDB dans `analysis` (centralisation voulue par les règles 6 et 8) |
| DEP-P2-1 | `config/config.go:17-18`, `player_resolver.go:14` | `config` importe `platform/duckdb` et `prestige` ; `platform` importe `config` |
| DEP-P2-2 | `games/halo_5/livesync/wire.go:19` → `worldenrich/wiring.go:21` | un adapter de titre importe `service` par transitivité |
| ART-P2-1 | `persist/doc.go:49-58` et deux tables de stages | la recette « 3 étapes » d'un enrichissement omet la liste des colonnes de la vue ; l'échec est bruyant |
| FLG-P2-1 | `observability/logging/config.go:123` | `LEVELUP_LOG_JSON` hérité, sans date |
| FLG-P2-2 | `handlers/media_paths.go:110-160`, `web/lib/title-routing/buildLegacyRedirect.ts:8` | branches de compatibilité sans date de retrait |
| FLG-P2-3 | `LEVELUP_LOG_LEVEL`, `LEVELUP_OAUTH_CLIENT_ID`, `LEVELUP_LEGACY_FILM_CACHE_DIR`, `LEVELUP_NOTIFY_VERSIONS` | variables lues hors `internal/config`, certaines à deux endroits |
| WEB-P2-1 | `components/ui/match-card-assisted-frags.tsx:19-22` | `components/ui` importe `features/_shared` |
| WEB-P2-2 | `features/auth/JoinPage.tsx:22` | une feature importe son module de route (cycle) |
| WEB-P2-3 | `timeseries/TimeseriesKdaDensity.tsx:57-105`, 9 moyennes inline | statistiques dans des composants, copies multiples, traitement des valeurs nulles divergent |
| WEB-P2-4 | `routes/.../replay.tsx` (470 L), `routes/.../$titleSlug.tsx:55-90` | logique d'orchestration dans des fichiers de route |
| WEB-P2-5 | `settings/queries.ts:120,131`, `palmares/queries.ts:38,60` | clés de requête dérivées écrites hors `lib/query/keys.ts` |
| WEB-P2-6 | `auth/XboxLoginPage.tsx:88`, `routes/__root.tsx:58,109`, `lib/title-routing/applyActiveTitle.ts:59` | récupération du bootstrap écrite quatre fois, options différentes |
| TEC-P2-1 | 11 tableaux d'`admin/`, `career/`, `leaderboard/` | tri fait à la main (11 `toggleSort`) au lieu de TanStack Table |
| TEC-P2-2 | 7 fichiers ; `explorer/explorerTableRows.ts:54` | `fetch` hors `lib/api/client` ; base URL recopiée |
| TEC-P2-3 | 72 fichiers hors `components/charts/` | import du type d'option ECharts ; aucun réexport depuis `components/charts/` |
| MORT-P2-1 | `archlint/*_test.go` | 104 entrées d'allowlist pointent des binaires sans utilisateur ; `openspartan.IsOpenSpartanDB`, `campaign.MinMatchesForTrend` sans appelant |

## Constats écartés ou requalifiés

| Constat | Axe | Motif |
|---|---|---|
| STO-1 : écriture joueur « sans bail, risque de corruption » | STO | requalifié : le handle vient du cache du process (`sync.OpenPlayerDB`), pas d'un second process ; reste le contournement de `persist` |
| WEB : `components/ui` → `_shared` et `JoinPage` → route, proposés P1 | WEB | requalifiés P2 : le sens des dépendances venait du cadrage, pas d'une règle écrite ; `Route.useSearch` est un motif TanStack |
| API-3 : erreur ignorée dans `upsertPlaylistWeight` | API | écarté : commentaire de justification (poids recalculé au cycle suivant) |
| Taille de `server_apiv1.go` (1 749 L) | API | dette assumée (baseline lint) |
| 74 paires feature → feature et 8 cycles | WEB | déclarés dans l'allowlist du lint |
| 83 lectures de `.DemoMode` / `.AuthMode` | FLG | allowlist du ratchet ADR 0044 |
| `LEVELUP_SYNC_RESOLVE_ASSETS` échu | FLG | déjà à `REGISTRE_REPORTS.md:283` |
| `backfill_t0`, `backfill_t0_film`, `diag_weapons_v3` | MORT | déjà à `REGISTRE_REPORTS.md:42` et `:257` |
| 13 binaires cités par un plan actif (`diag_film`, `killsource`, `fetch_film_chunks`, ...) | MORT | non comptés comme morts |
| Lecture brute `Q24LUSRHistory` | ART | exemption datée du 2026-07-10 |
| État global zustand hors `stores/` (6 fichiers) | TEC | aucune règle écrite |
| `analysis` → `games/mappings` | DEP | autorisé par `no_title_package_in_analysis_test.go` |

## Points sains établis

- Écritures : allowlists ART vides ; 0 `ON CONFLICT DO UPDATE` ni `INSERT OR REPLACE` dans `persist`, `sync`, `media` ; aucun `UPDATE match_registry` sur une colonne indexée ; ADR 0039 et 0047 respectées.
- Couches Go : `domain`, `analysis`, `port` n'importent ni `config`, ni `platform`, ni `persist`, ni `sync`, ni `service`, ni `api` ; `service/` n'importe jamais `platform/duckdb` ; `middleware/` sans requête ; 0 SQL dans `handlers/` ; `platform/auth` sans métier ; aucun cycle de paquets.
- Briques : 0 dépendance vers l'application ; rang respecté ; `libs/film/internal/` inatteignable.
- Front : un seul importeur d'`echarts-for-react` ; clés de requête tenues par un garde ; pont Tauri fermé ; aucune comparaison sur le profil de déploiement ; aucun littéral de route ; fraîcheur du contrat généré tenue.
- Flags : `LEVELUP_PERSIST_BATCH_ASYNC` et `LEVELUP_EVENTS_CONVERGENCE` portent leurs trois mentions ; aucun résidu des flags supprimés.

## Garde-rails manquants

Chaque ligne est une règle écrite qu'aucun test ne tient ; c'est là que se trouvent les P1.

| Règle | Garde-rail à poser |
|---|---|
| handler sans `platform/duckdb` ni moteur de sync | ratchet d'imports sur `api/handlers` |
| `api/wire` et racine `api/` sans SQL | ratchet de littéraux SQL |
| service sans `games/{slug}` | étendre `no_title_package_in_analysis_test.go` à `service/` |
| `analysis` sans `database/sql` | étendre `tactical_pure_test.go` à tout `analysis/` |
| allowlist datée de `duckdb.OpenReadWrite` (ADR 0030 D-3) | étendre `no_bare_duckdb_rw_open_test.go` |
| date obligatoire sur toute entrée d'allowlist | test de forme sur les allowlists |
| pas d'`os.Getenv` hors `internal/config` | ratchet à plafond décroissant |
| imports entre features | corriger le motif `_shared`, contrôler les imports relatifs, marqueur par ligne |
| types d'API manuels homonymes du contrat | étendre `response-types.guard.test.ts` à tous les schémas |
| `fetch` hors `lib/api/client` | garde avec allowlist datée |
| graphe monté hors `ChartCard` | garde sur `<LazyECharts` |

## Suite

- **À trancher par l'utilisateur** : ART-2, MORT-1 (famille par famille), FLG-1 et FLG-2 (retirer ou activer), et pour ART-5 le sort du calcul et de la colonne stockée (l'abandon de l'affichage est acté).
- **Chantier candidat « frontières »** : API-1 à API-4, DEP-1, STO-1 à STO-4, avec leurs garde-rails. C'est le seul lot qui change la réponse à « changer une technologie ».
- **Chantier candidat « ménage »** : MORT-1, MORT-2, ART-5, FLG-3 à FLG-5, ART-3, ART-4.
- **Chantier candidat « front »** : WEB-1, WEB-2, TEC-1, TEC-2.
- Tout chantier se cadre sous `plan-review`, s'exécute sous `plan-execution`, se relit sous `adversarial-review`.

## Statut des constats (clôture, 2026-11-13)

Plan `.ai/PLAN_ARCHITECTURE_FRONTIERES_2026-11-10.md`. Statuts : **traité** (étape et item du plan), **traité
avec reste** (le reste est nommé), **couvert ailleurs**, **non traité** (justification). Bilan : 25 P1 et 22 P2
traités (dont 3 avec reste : TEC-2, FLG-P2-2, TEC-P2-2), 12 écartés ou requalifiés (dont 1 requalifié P1 vers P2 traité),
aucun constat sans statut.

### P1

| Réf. | Statut | Où | Note |
|---|---|---|---|
| ART-5 | traité | 1.6 (D2, D9) | capability, affichage, champs d'API et lecteurs retirés ; le calcul et la colonne `match_skill_rank.expected_win_prob` restent (D2) |
| ART-2 | traité | 3.1 | `convergePSACollect` / `flushAliasPairs` / `settlePSA`, marqueur après les alias, échecs comptés ; `convergePSA` sorti de la production le 2026-11-13 (8.5) |
| ART-1 | traité | 3.2 | journal et compteur `match_intensity_write_failed_total` |
| STO-1 | traité | 5.1 | amorçage par `PostSyncEnrichmentPersister.PrimeBaseline`, sans arrêt à la première erreur |
| API-1 | traité | 4.3 | progression, campagne, profil joueur et limite mémoire derrière des ports ; `TestHandlersSansStockageDuckDB` |
| API-2 | traité | 4.2 | `port.SyncJobService` et `service.SyncJobService` ; `TestHandlersSansMoteurDeSync` |
| API-3 | traité | 3.3 (erreurs de `rows.Scan`), 4.4 | SQL, écritures et KPM/PSPM sortis de `api/wire` ; `TestApiSansSQL` |
| API-4 | traité | 4.5 | handlers et fichiers statiques dans `api/handlers/`, règle d'audience en service, purge des sessions dans `scheduler/` |
| DEP-1 | traité | 6.1, 6.2 | 15 imports (plus 1 de `livesync`) à 0 ; `TestServiceImporteAucunPaquetDeTitre` |
| STO-2 | traité | 5.2 | `platform/duckdb/profilerepo` derrière `port.ProfileStore` |
| STO-3 | traité | 5.3 | `cronstore`, `worldrepo` ; `TestSchedulerSansStockageDuckDB` |
| STO-4 | traité | 5.4 | `TestAnalysisSansStockage` |
| ART-3 | traité | 3.4 | `TestOpenReadWriteHorsDuPointUniqueSurAllowlist` ; `duckdb.OpenReadWriteShared` reste non gardé (§12 du plan) |
| ART-4 | traité | 3.5 | dates, clé par chemin complet, tests de forme |
| FLG-1 | traité | 2.1 | interrupteur retiré (D3) |
| FLG-2 | traité | 2.2 | réparation automatique supprimée (D4) ; détection, jauge, alerte et recalcul manuel intacts |
| FLG-3 | traité | 2.3 | sept interrupteurs retirés (D5), `LUSRRev` inchangée |
| FLG-4 | traité | 1.7 | deux commentaires corrigés |
| FLG-5 | traité | 1.4 | code mort supprimé |
| WEB-1 | traité | 7.1 | `lint-cross-feature-imports.mjs` à plafond 0 |
| WEB-2 | traité | 7.2 | `featureTsxNoQuery.guard.test.ts` |
| TEC-1 | traité | 7.3 | `lazyEchartsMount.guard.test.ts` ; rendu sous changement de palette à contrôler à l'écran (utilisateur) |
| TEC-2 | traité avec reste | 1.8, 7.4 | 31 types manuels homonymes : 11 restent en listes datées (4 types d'entrée, 2 réponses où le contrat diffère du serveur, 5 hors `types.ts`), garde à plafond 0 hors listes |
| MORT-1 | traité | 1.1, 1.9 | 180 commandes, 98 restantes (D8 révisée, décisions du 2026-11-10 et du 2026-11-11) |
| MORT-2 | traité | 1.2 | `steam_poller.go` et `domain/chart/` supprimés |

### P2

| Réf. | Statut | Où | Note |
|---|---|---|---|
| API-P2-1 | traité | 4.6 | interfaces de `port/` pour les sept services |
| STO-P2-1 | traité | 5.5 | `DBExecutor` et `DBWriter` dans `platform/dblease` ; `TestPortSansTypeSQL` |
| STO-P2-2 | traité | 5.1 | `*sql.DB` hors des signatures des services d'import (`importstore`) |
| STO-P2-3 | traité | 1.5 | import du pilote retiré |
| STO-P2-4 | traité | 5.6 | `domain.ErrRecordNotFound` / `port.ErrNotFound`, cinq comparaisons (le plan en comptait six) |
| STO-P2-5 | traité | 5.7 | `platform/duckdb/sqlfrag` |
| DEP-P2-1 | traité | 5.8 | `playerresolve` ; `config` n'importe plus `platform/duckdb` ni `prestige` |
| DEP-P2-2 | traité | 4.1 | `TestWorldenrichSansService` |
| ART-P2-1 | traité | 3.6 | recette de `persist/doc.go` et `TestEnrichmentColumnStage_LieALaVueMergeOnRead` |
| FLG-P2-1 | traité | 2.3 | `LEVELUP_LOG_JSON` retirée, image sur `LEVELUP_LOG_FORMAT=json` |
| FLG-P2-2 | traité avec reste | 2.5 | les deux branches sont datées (retrait cible 2026-12-31, critère écrit), non retirées : la mesure de production ne les prouve pas inutilisées ; le retrait est à décider à l'échéance |
| FLG-P2-3 | traité | 2.4, 2.6 | six variables lues par `internal/config` ; `TestAucuneLectureEnvHorsConfig`, 13 fichiers en allowlist datée |
| WEB-P2-1 | traité | 7.11 | modules d'assistance dans `components/assists/` |
| WEB-P2-2 | traité | 7.11 | `noRouteImport.guard.test.ts` |
| WEB-P2-3 | traité | 7.8 | `lib/charts/stats` ; `medianSeconds` et `bucketMedian` non branchés (§12) |
| WEB-P2-4 | traité | 7.9 | `ReplayPage`, `useTitleConvergence` |
| WEB-P2-5 | traité | 7.10 | clés dérivées dans `queryKeys`, `keys.guard.test.ts` |
| WEB-P2-6 | traité | 7.10 | `lib/query/bootstrap.ts`, `bootstrap.guard.test.ts` |
| TEC-P2-1 | traité | 7.7 | `lib/table/useColumnSort`, `noLocalSortToggle.guard.test.ts` |
| TEC-P2-2 | traité avec reste | 7.6 | `fetch` hors client de 7 à 4, tous en liste datée (GitHub, deux sons du rejeu, un fichier statique) ; base URL une seule fois |
| TEC-P2-3 | traité | 7.5 | `chartsImportBoundary.guard.test.ts` ; imports d'`echarts` hors `components/charts/` de 72 à 0 |
| MORT-P2-1 | traité | 1.1, 1.3 | entrées d'allowlist retirées, deux fonctions supprimées |

### Écartés ou requalifiés

| Constat | Statut | Note |
|---|---|---|
| STO-1 « sans bail » | requalifié, sans objet | le contournement de `persist` est traité par 5.1 |
| WEB : `components/ui` vers `_shared`, `JoinPage` vers la route | requalifiés P2, traités | WEB-P2-1 et WEB-P2-2 (7.11) |
| API-3 : erreur de `upsertPlaylistWeight` | écarté, sans changement | commentaire de justification ; la revue de 4.4 note que `PlaylistExpansionService` ignore l'erreur de `RecordWeight` (§12 du plan, non traité) |
| Taille de `server_apiv1.go` | dette assumée, non traité | baseline lint |
| 74 paires feature vers feature et 8 cycles | couvert ailleurs | déclarés dans l'allowlist du lint, dont le marqueur d'exemption est par ligne depuis 7.1 |
| 83 lectures de `.DemoMode` / `.AuthMode` | dette assumée, non traité | allowlist du ratchet de l'ADR 0044 |
| `LEVELUP_SYNC_RESOLVE_ASSETS` échu | couvert ailleurs | `.ai/REGISTRE_REPORTS.md` ; lecture encore en allowlist datée de `TestAucuneLectureEnvHorsConfig` |
| `backfill_t0`, `backfill_t0_film`, `diag_weapons_v3` | couvert ailleurs | `.ai/REGISTRE_REPORTS.md` |
| 13 binaires cités par un plan actif | non traité | conservés (D8) |
| Lecture brute `Q24LUSRHistory` | non traité | exemption datée du 2026-07-10, « non revérifié » (§12 du plan, étape 3) |
| État global zustand hors `stores/` | non traité | aucune règle écrite |
| `analysis` vers `games/mappings` | dette assumée, non traité | autorisé par `no_title_package_in_analysis_test.go` |
