# Plan : compaction des passes de décodage (lot C) puis lectures bornées aux matchs du joueur (lot B) — 2026-09-26

> Suite de `.ai/PLAN_PERF_LECTURES_PERIMETRE_2026-09-26.md` (ADR 0036, lot A clos) et du handoff
> `.ai/HANDOFF_PERF_CHARGEMENTS_2026-09-24.md` §4 / §6. Go utilisateur du 2026-09-26 (« je suis
> d'accord avec toi pour le lot B et le lot C, vas-y écris le plan »). Contrat d'exécution : skill
> `plan-execution` (ordre strict, aucun item sans statut, zéro fix hors périmètre, découvertes au
> §6). Le plan fait foi en cas de divergence.

## 0. Faits mesurés et décisions tranchées (superviseur, 2026-09-26)

**Mesure** (copie de `shared_matches_v2.duckdb` du 2026-09-26 14:07, lecture seule, fichier de
1,2 Gio) — lignes brutes contre lignes servies par la vue `_latest` :

| Table | Brutes | `_latest` | Rapport |
|---|---|---|---|
| `match_kill_events` | 3 953 799 | 412 216 | 9,6 |
| `match_lives` | 1 725 664 | 163 092 | 10,6 |
| `match_death_context` | 1 516 693 | 146 654 | 10,3 |
| `kill_openings` | 1 495 845 | 147 225 | 10,2 |
| `kill_positions` | 1 495 728 | 147 360 | 10,2 |
| `match_weapon_shots` | 896 590 | 74 882 | 12,0 |
| `match_player_positions` | 167 189 | 22 198 | 7,5 |
| `match_usage_players` | 11 465 | 1 420 | 8,1 |
| `match_pad_pickups_by_tier` | 5 590 | 1 502 | 3,7 |
| `match_usage_films` | 1 262 | 152 | 8,3 |
| `match_flag_grabs_net` | 353 | 147 | 2,4 |
| `match_csrs`, `match_objective_stats` | 30 105 / 26 499 | identiques | 1,0 |

Environ 90 % des lignes des tables issues du film sont des passes de décodage SUPERSÉDÉES : chaque
redécodage d'un film ajoute une passe complète (écriture INSERT-only, ADR 0019 / 0026) et la vue
`_latest` ne garde que la dernière passe de chaque match. Toute lecture qui évalue une de ces vues
parcourt les passes mortes. L'utilisateur confirme qu'aucune ancienne passe n'a d'usage.

Décisions :
- **Ordre : C avant B.** La compaction divise le coût de TOUTES les fenêtres `_latest` des tables
  du film (Carrière, Relations, rencontres de la vue match, repli de localisation du lot A,
  onglet Tactique) ; le lot B ne traite ensuite que ce qui reste lent, mesuré sur la copie
  compactée.
- **Lot B = borner la lecture aux matchs du joueur**, sous la fenêtre (patron L5a / ADR 0036 I2).
  PAS de cache de lecture (décision utilisateur : le cache ne sert qu'aux visites répétées, la
  première ouverture après chaque sync paie le prix plein, et son invalidation est une source
  d'erreurs).
- **Branche** : `feat/perf-perimetre` (worktree `C:\Users\Guillaume\Projects\LevelUp-wt-perf-perimetre`),
  à la suite du lot A — même chantier, pas encore fusionné. Fusion dans `feat/v75` : sur signal
  de l'utilisateur seulement (consigne du 2026-09-26).
- **Agents** : Opus au plus, type `opus-high` pour l'exécution (C touche aux invariants anti-ART,
  B à la parité des chiffres), au plus quatre en parallèle, mais les étapes restent séquentielles
  (un exécuteur à la fois dans le worktree ; relecteurs en parallèle entre eux).
- **Données réelles** : AUCUN agent ne compacte une base sous `data/`. Local : le superviseur,
  serveur arrêté, sauvegarde faite, après accord de l'utilisateur (étape F). Prod : geste de
  l'utilisateur, étape Ops de la v7.5 après les recuissons (qui AJOUTENT des passes).

## 1. Règles communes

Celles du §1 du plan du lot A s'appliquent telles quelles (worktree seul, copie de base dans le
scratchpad ouverte `access_mode=read_only` pour mesurer — une compaction de test se fait sur une
SECONDE copie ouverte en écriture dans le scratchpad, jamais sous `data/` ; sonde temporaire
`cmd/*_probe_tmp/` supprimée avant commit ; commandes `go` une à la fois, `CGO_ENABLED=1`, GOCACHE
privé `.gocache-perimetre`, `CC` de l'environnement ; interdits : push, stash, `--no-verify`,
`--force`, `git add -A`, `.ai/thought_log.md`, serveur, navigateur, sous-agent, arrière-plan ;
baseline de tests tenue ; artefacts `data/` du worktree supprimés avant commit). En plus :
`go test -tags=integration -p 1 ./...` COMPLET est obligatoire à la clôture de C (le diff touche
migration / persist / sync).

## 2. Étape C — Compaction des passes supersédées (Go, lot sensible)

Décisions :
- DC.1 **Tables éligibles** : une table append-only est compactable si et seulement si (a) sa vue
  `_latest` retient « toutes les lignes de la dernière passe par match » (`decode_pass`,
  `positions_pass`, `summary_pass`) ou « la dernière ligne par clé » sans fusion de colonnes, et
  (b) AUCUN lecteur ne lit ses anciennes versions. Exclues d'office : `match_skill_rank` (lecture
  brute VOLONTAIRE de Q24LUSRHistory et de `queries_squad.go`, allowlist de
  `no_raw_rating_reads_test.go`), `player_match_enrichment` (vue fusionnée par colonne), toute
  table dont un lecteur brut n'est pas une lecture de localisation. Item C.1 = inventaire exhaustif
  (bases partagées de chaque titre ET bases joueur), rapport brut / `_latest` mesuré sur copie,
  lecteurs bruts grepés sur `internal/` ET `cmd/` : la liste retenue est écrite dans ce plan avant
  tout code.
- DC.2 **Ce qui est gardé** : exactement les lignes de la passe (ou version) que la vue retient,
  définie par la règle de la vue elle-même (même clé de partition, même ordre) — toutes les lignes
  de cette passe, y compris celles qu'un filtre supplémentaire de la vue écarte (ex. le
  `row_number` de `kill_positions_latest`) : la vue reste l'unique chemin de lecture et rend le
  même résultat. Preuve obligatoire par table : sortie de la vue AVANT == APRÈS (empreinte d'une
  lecture ordonnée complète), et lignes brutes après == lignes de la dernière passe avant.
- DC.3 **Mécanisme** : reconstruction transactionnelle par CTAS + swap, JAMAIS de `DELETE` (bug
  ART #23645) — réutiliser la mécanique de `internal/migration/append_only_rebuild.go` (TX, garde
  de cardinalité AVANT le DROP, `recoverOrphan` pour un crash en cours de swap, idempotence) sans
  la copier (extraire le cœur commun si nécessaire, règle des 2 copies). Schéma, PK technique
  `id`, séquences (la valeur courante CONTINUE : aucun `id` réutilisé), index, contraintes et vues
  `_latest` identiques après le swap — vérifié par test sur le DDL.
- DC.4 **Exécution** : commande CLI `levelup` (placement cohérent avec les commandes existantes de
  `cmd/levelup/`, nom à justifier au rapport), par titre via `PathResolver` et le registre des
  titres (aucun `slug == ...`), écrivain exclusif par le bail (ADR 0013), REFUS si le serveur ou un
  autre processus tient la base. `--dry-run` : lecture seule, rapport brut / `_latest` par table
  et estimation du gain ; sans `--dry-run` : sauvegarde préalable exigée (chemin affiché, reprise
  de l'outillage de sauvegarde existant s'il y en a un), compaction table par table, journal
  structuré (`slog`) par table (avant, après, durée). Idempotente (une seconde passe ne fait rien).
- DC.5 **Taille du fichier** : DuckDB réutilise les blocs libérés mais ne rétrécit pas le fichier.
  Option `--rewrite-file` : recopie de la base dans un fichier neuf (tables, vues, séquences avec
  leur valeur courante, macros), vérification (liste des objets, comptes de toutes les tables,
  empreintes des vues compactées), échange des fichiers en gardant l'ancien en sauvegarde. Si
  DuckDB 1.5.5 ne préserve pas la valeur courante des séquences ou un autre objet : l'option n'est
  PAS livrée, `[!]` avec la preuve (le gain de lecture ne dépend que du nombre de lignes).
- DC.6 **Pas d'automatisme** : pas de compaction dans le post-sync ni au boot (elle exige
  l'écrivain exclusif et le serveur arrêté). C'est une opération d'entretien, documentée comme
  telle, à jouer après chaque campagne de redécodage (recuisson, backfill killsource).

Items :
- [x] C.1 inventaire (DC.1) : tables, règle de leur vue, rapport mesuré, lecteurs bruts ; liste
      retenue et exclusions justifiées écrites ici — Journal C, « C.1 » : 13 tables retenues
      (12 à passe + `match_bomb_stats`), toutes les autres vues `_latest` exclues avec leur raison
- [ ] C.2 cœur de compaction (DC.2, DC.3) + tests d'intégration : passes multiples, match à une
      seule passe, table vide, crash simulé en cours de swap (`recoverOrphan`), idempotence,
      séquence qui continue, DDL identique, vue identique avant / après
- [ ] C.3 commande CLI (DC.4) : `--dry-run`, refus si la base est tenue, sauvegarde, journal ;
      tests de la commande
- [ ] C.4 `--rewrite-file` (DC.5), ou `[!]` prouvé
- [ ] C.5 preuve sur copie : seconde copie de la base compactée par la commande ; empreintes des
      vues avant / après identiques pour CHAQUE table retenue ; taille du fichier avant / après ;
      chrono avant / après sur copie (2 threads / 512 Mo) : rencontres et rivaux de la Carrière,
      liste Relations, rencontres de la vue match (Q23b), repli de localisation du lot A, un bloc
      de l'onglet Tactique ; cinq joueurs suivis
- [ ] C.6 docs : `docs/COMMANDS.md` + `docs/FR/COMMANDS.md` (commande, quand la jouer, serveur
      arrêté, sauvegarde) ; ligne Ops du changelog 7.5.0 EN + FR (après les recuissons et le
      backfill killsource, serveur arrêté) ; ADR 0036 (section Exceptions : le coût des fenêtres
      suit le nombre de passes, la compaction est l'entretien qui le borne) et renvoi dans l'ADR
      0026 si une règle y change (sinon rien)
- [ ] C.7 mutations jouées : garde de cardinalité retirée, `DELETE` réintroduit (garde-rail
      anti-ART rouge), passe gardée = la première au lieu de la dernière, séquence remise à zéro ;
      chacune rouge puis restaurée

Gate C (depuis `apps/go-api`) : `gofmt -l ./internal ./cmd` vide ; `go build ./...` ; `go vet
./...` ; `go test ./...` ; `go test -tags=integration -p 1 ./...` (code de sortie 0 vérifié, pas
la sortie filtrée) ; garde-rails `internal/sync` (`NoART|Legacy|Sentinel`) ; golangci-lint
`--new-from-rev=f04b9fb78` avec et sans `--build-tags=integration` ; baseline de tests.

Revue C : OBLIGATOIRE, deux relecteurs aveugles en parallèle (skill `adversarial-review` : lentille
L1 anti-ART / écritures, lentille L6 tests), deux rondes au plus.

Journal C (2026-09-26, exécuteur Opus, worktree `LevelUp-wt-perf-perimetre`, base `34edf29af`) :

- **C.1 — inventaire, écrit avant tout code.** Sources : `duckdb_views()` de COPIES (lecture
  seule, CLI 1.5.4) de chaque base de chaque titre — partagée HI (copie du 2026-09-26 14:07) et H5,
  `shared_pve`, `shared_social` HI et H5, `metadata` HI et H5 (aucune vue `_latest`), bases joueur
  HI (JGtm, Madina97294, Chocoboflor, XxDaemonGamerxX) et H5 (JGtm, Madina97294) ; lecteurs bruts
  grepés sur `internal/` ET `cmd/` (FROM / JOIN, littéraux de nom de table, SQL construit par
  concaténation), hors commentaires. Les 20 vues `_latest` de la base partagée ont le MÊME SQL sur
  HI et H5 (diff vide).
  - Règle « dernière passe entière par match » (`QUALIFY <passe> = first_value(<passe>) OVER
    (PARTITION BY match_id ORDER BY written_at DESC, id DESC)`), gardées = toutes les lignes de
    cette passe (DC.2), mesuré HI (brutes / `_latest` / gardées / matchs) :
    `match_kill_events` 3 953 799 / 412 216 / 412 216 / 3 631 ; `match_lives` 1 725 664 / 163 092 /
    163 092 / 1 519 ; `match_death_context` 1 516 693 / 146 654 / 146 654 / 1 516 ; `kill_openings`
    1 495 845 / 147 225 / 147 225 / 1 519 ; `kill_positions` 1 495 728 / 147 360 / 147 360 / 1 519
    (sa vue ajoute un `row_number() = 1` DANS la passe : les gardées sont la passe entière) ;
    `match_weapon_shots` 896 590 / 74 882 / 74 882 / 1 593 ; `match_player_positions`
    (`positions_pass`) 167 189 / 22 198 / 22 198 / 99 ; `match_usage_films` (`summary_pass`) 1 262 /
    152 / 152 / 152 ; `match_pad_pickups_by_tier` 5 590 / 1 502 / 1 502 / 99 ;
    `match_flag_grabs_net` 353 / 147 / 147 / 17 ; `match_weapon_hit_distance` 0 / 0. H5 :
    `match_kill_events` 536 674 / 268 337 / 268 337 ; `kill_positions` 297 963 / 295 357 / 297 963
    (une passe `legacy-<match>` par match : rien à compacter, l'écart est le `row_number` de la
    vue) ; les autres tables vides.
  - Règle « passe de la vue des films » : `match_usage_players_latest` = jointure sur
    `match_usage_films_latest` (`match_id`, `summary_pass`) ; gardées = les lignes dont le couple
    est dans cette vue : 11 465 / 1 420 / 1 420 ; 0 ligne joueur sans ligne film.
  - Règle « dernière ligne par clé » (`row_number() OVER (PARTITION BY match_id, xuid ORDER BY
    written_at DESC, id DESC) = 1`), produit du film (statistiques d'Assaut, `backfill-bomb-stats
    --force` réécrit) : `match_bomb_stats` 0 / 0 aujourd'hui.
  - **Lecteurs bruts de ces 13 tables** (hors écrivains `internal/persist`, DDL des migrations et
    commentaires) : (1) `analysis/identity_annuaire.go` `AnnuaireKillFeedLocaliserSQL` (lecture de
    localisation, DA.10 du lot A) ; (2) la migration `shared_kill_events_credit_base_v1`
    (`decode_pass LIKE 'creditbase-%'` comme marque d'idempotence) ; (3) `ops/seed_demo*.go` et
    `ops/snapshot_export.go` COPIENT les lignes brutes d'un sous-ensemble de matchs vers une autre
    base qui les relit par la même vue ; (4) `cmd/h5-backfill`, `cmd/h5-sync` impriment un
    `count(*)` brut de `kill_positions` (diagnostic). Aucun ne lit une ancienne passe pour sa
    valeur : (1) après compaction la brute ne porte que les passes que la vue retient — la
    localisation rend les matchs où la DERNIÈRE passe porte le xuid avec un nom, ce qui reste un
    sur-ensemble des partitions `_latest` qui le montrent (égalité exacte pour la jambe kill-feed :
    la vue de `match_kill_events` rend la passe entière) ; le pas 2 relit ces matchs par `_latest` :
    même nom, même absence de nom, la lecture reste CORRECTE (et plus courte) ; (2) migration
    name-keyed, appliquée le 2026-08-19 sur la base locale, jamais rejouée (`schema_migrations`), et
    la commande joue les migrations AVANT de compacter : une base où elle n'a pas encore tourné la
    voit tourner d'abord ; (3) la copie ne porte plus que la dernière passe, la vue de la base
    cible rend le même résultat ; (4) le compte baisse, c'est ce qu'il doit montrer. Les écrivains
    tirent `decode_pass` au hasard (`newDecodePassID`) et ne lisent jamais les passes précédentes ;
    les sélections des backfills (`backfill-killsource`, `-usage-summary`, `-pad-tiers`,
    `-flag-grabs-net`) lisent les vues `_latest`.
  - **Retenues (13)** : `match_kill_events`, `match_lives`, `match_death_context`,
    `kill_openings`, `kill_positions`, `match_weapon_shots`, `match_weapon_hit_distance`,
    `match_player_positions`, `match_usage_films`, `match_usage_players`,
    `match_pad_pickups_by_tier`, `match_flag_grabs_net`, `match_bomb_stats` — base partagée des
    matchs, même liste pour chaque titre (tables absentes ou vides : rien à faire).
  - **Exclues**, et pourquoi : `match_skill_rank` (bases joueur ; d'office, lectures brutes
    volontaires, allowlist de `no_raw_rating_reads_test.go`) ; `player_match_enrichment` (vue
    fusionnée par colonne) ; `player_skill_state_v2` (59 383 / 10 140 HI, 7 759 / 4 H5 : lecteur
    brut de l'historique `SkillV2Repo.LoadStateHistory`, lissage TTT, et `cmd/diag_lusr_volatility`,
    `cmd/lusr_v2_ttt_batch`) ; `world_csr_leaderboard_snapshots` (24 007 / 4 393 : lecteurs bruts
    `leaderboard_world_batch_stats.go` x3, `leaderboard_world_repo.go` x2 — l'historique des
    instantanés est lu) ; `match_csrs` (30 105 / 30 105 : lecteurs bruts
    `career_repo_csr_seasons.go`, `sync/schema.go`, `ops/snapshot_read.go`) ; `player_csr_snapshots`
    (joueur, 16 798 / 7 : lecteur brut `cmd/seed-ranked-playlists`, historique CSR) ;
    `weapon_kills` (H5 seulement, vue `v_weapon_kills` par génération, 550 926 / 270 585 : donnée
    d'API, pas une passe de décodage, et lecteurs bruts `ops/snapshot_read.go`, `ops/seed_demo.go`,
    `cmd/diag_*`) ;
    `match_citations`, `personal_score_awards` (lecteurs bruts `sync/citations_backfill.go`,
    `sync/invariants`, `sync/convergence.go`, `cmd/backfill_all`, …) ; `player_records_history`
    (lecteur brut `ops/records_purge.go`) ; `pve_match_stats` (20 / 20, lecture brute
    `pve_persister.go`) ; `match_objective_stats` (26 499 / 26 499), `world_player_season_stats`
    (13 007 / 7 047), `lusr_hyperparams_v2` (24 / 24), `player_squad_offset` (0),
    `lusr_component_history` (joueur, ≤ 8 060 / 7 137), `streak_history` (≤ 57 / 22) et les huit
    `*_history` de `shared_social` (≤ 254 / 191) : versions d'une donnée d'API ou d'une donnée
    utilisateur, pas des passes de décodage — la confirmation de l'utilisateur (« aucune ancienne
    passe n'a d'usage ») ne les couvre pas, et leur rapport (≤ 1,9 hors tables à lecteur brut) ne
    rend aucun gain de lecture mesurable.

## 3. Étape B — Lectures bornées aux matchs du joueur (Go)

Décisions :
- DB.1 **Cibles candidates** (handoff §4, ADR 0036 Exceptions) : Carrière rencontres (Q26) et
  rivaux (Q27, lus deux fois), liste Relations (Q28, Q28 scopé), onglet Tactique « morts par
  carte » (`MortsParCarte`, liste bornée sur `match_registry` seulement), Accueil (`GET
  /pages/home`, 1,6 s pour 310 Ko), Synthèse bloc « records de distance par arme »
  (`weapon_records`, 1,6 s). Item B.0 : re-mesurer chacune sur la copie COMPACTÉE (C.5) ; seules
  celles qui dépassent 300 ms sont traitées, les autres statuées `[~]` « résolu par C » avec leur
  chiffre.
- DB.2 **Mécanisme** : la liste des matchs du joueur (`QMatchsDuJoueurTpl`, exclusion Campagne
  comprise, ou le périmètre de filtres de la page quand il existe) est liée SOUS la fenêtre des
  vues `_latest` (seul `match_id` se pousse), comme `tactical_repo_fenetres_test.go` le fige déjà.
  Rivaux : UNE lecture de l'agrégat, les deux classements (morts subies, frags infligés) triés en
  Go. Aucun cache.
- DB.3 **Parité stricte** : lignes, compteurs, noms et ordre servis identiques avant / après sur
  les cinq joueurs suivis (JGtm, Madina97294, Chocoboflor, XxDaemonGamerxX, Nuzzles) ; ex aequo
  comparés en ensembles là où l'ordre n'était pas total (dette connue, non traitée).
- DB.4 **Garde-rails** : chaque lecture traitée entre dans un test de fenêtres bornées par
  EXPLAIN ANALYZE (même helper `exigerFenetresBornees`) : rouge si la fenêtre voit plus que les
  matchs demandés. ADR 0036 : lignes retirées de la section Exceptions avec leur coût après.
- DB.5 Si la cause d'une cible n'est PAS une fenêtre sur tout l'historique (Accueil, par exemple,
  peut être la taille de la réponse) : la cible est statuée `[!]` avec la mesure qui le montre et
  consignée au §6 ; pas de changement d'une autre nature dans ce lot.

Items :
- [ ] B.0 re-mesure des cibles sur la copie compactée (DB.1), liste traitée écrite ici
- [ ] B.1 Carrière : rencontres et rivaux bornés, rivaux en une lecture ; parité ; chrono
- [ ] B.2 Relations : Q28 et Q28 scopé bornés ; parité ; chrono
- [ ] B.3 Tactique `MortsParCarte` : fenêtres du journal des morts et des positions bornées ;
      parité ; chrono
- [ ] B.4 Accueil et Synthèse (`weapon_records`) : selon B.0 et DB.5
- [ ] B.5 garde-rails (DB.4), sections de durée, ADR 0036 mis à jour
- [ ] B.6 mutations : liste retirée de sous la fenêtre (test de fenêtres rouge), rivaux relus deux
      fois (test de comptage rouge), exclusion Campagne retirée (parité rouge)

Gate B : comme C sans l'intégration complète : `gofmt`, build, vet, `go test ./internal/service/...
./internal/platform/duckdb/... ./internal/analysis/... ./internal/api/... ./internal/archlint/...`,
`go test -tags=integration -p 1 ./internal/platform/duckdb/...`, golangci-lint.

Revue B : un relecteur aveugle (lentilles SQL / périmètre et parité des chiffres).

Journal B : (vide)

## 4. Étape F — Clôture (superviseur)

- [ ] F.1 gates rejoués par le superviseur sur la tête de la branche
- [ ] F.2 compaction de la base LOCALE réelle, après accord de l'utilisateur : serveur arrêté,
      `--dry-run` puis compaction avec sauvegarde, vues vérifiées, serveur relancé ; taille du
      fichier et durées des pages avant / après (l'utilisateur ouvre Carrière, Relations, une vue
      match ; durées lues dans `logs/http.log`)
- [ ] F.3 changelog 7.5.0 EN + FR complété (chiffres C et B) ; handoff et thought_log
- [ ] F.4 push de la branche et CI : sur signal de l'utilisateur
- [ ] F.5 fusion dans `feat/v75` : sur signal de l'utilisateur ; la compaction PROD reste son geste,
      dans la séquence Ops de la v7.5, après les recuissons

## 5. Reprise de session

Ce fichier fait foi. `git log --oneline f04b9fb78..feat/perf-perimetre` ; reprendre à la première
case non statuée de l'étape courante (C, puis B, puis F).

## 6. Découvertes (à consigner, pas à traiter)

(vide)

## 7. Journal (superviseur)

- 2026-09-26 : mesure des passes supersédées sur copie (tableau §0) ; lot C passé d'« abandonné »
  à « à faire » et lot B restreint au bornage (sans cache) sur décision de l'utilisateur ; plan écrit.
