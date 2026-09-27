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
  **Amendé le 2026-09-26 (C.9, revue L6)** : pour la compaction, `recoverOrphan` devient un REFUS
  (`refuserOrphelin`, en dry-run comme pour de bon) — la table de construction ne porte jamais les
  index secondaires (posés après le RENAME) et leur seule source a disparu avec la table ; les
  reposer exigerait de dupliquer le DDL d'index des migrations ou de le faire voyager dans un
  commentaire de catalogue, pour un état que l'échange transactionnel ne produit pas. L'exploitant
  remet la sauvegarde `*.avant-compaction-*` prise par l'exécution INTERROMPUE (la plus récente
  antérieure à l'orphelin) ; les écritures postérieures à cette sauvegarde sont perdues. Corrigé en
  C.10 : ce refus passe AVANT toute sauvegarde (et avant les migrations), en dry-run comme pour de
  bon — auparavant la sauvegarde de l'exécution courante copiait l'état déjà orphelin. La
  conversion append-only garde `recoverOrphan`.
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
  **Amendé le 2026-09-26 (C.8, revue adversariale L1)** : JAMAIS de fenêtre sans fichier au
  chemin de la base — l'ancien fichier est COPIÉ (sauvegarde `<nom>.avant-reecriture-<UTC>`),
  puis UN `rename(neuf, base)` remplace la base (POSIX atomique ; `os.Rename` remplace sous
  Windows). Juste avant ce rename : la base n'a pas changé depuis la copie (taille + date de
  modification relevées à la fermeture du handle de copie ; pas d'empreinte : relire 1,2 Gio
  n'est pas bon marché) et personne ne la tient (ouverture exclusive en écriture qui réussit,
  puis fermeture) ; sinon refus. Toute erreur après la création de `<base>.reecriture` le retire,
  avec la sauvegarde devenue inutile, journalisée en `slog`. `--backup-dir` sur un autre volume :
  CHOIX = la sauvegarde est une copie (io.Copy) écrite directement dans ce dossier AVANT
  l'échange, et le seul rename reste dans le dossier de la base (`<base>.reecriture` y est
  écrit) — ni recopie après coup, ni refus : la combinaison marche sur tout volume, et la
  sauvegarde existe avant que la base ne change.
  **Amendé de nouveau le 2026-09-26 (C.10, seconde revue ; méthode décidée par le superviseur,
  VOIE A)** : la vérification « inchangée » (taille + date) de C.8 ne voyait pas un tiers qui
  écrit dans le `.wal` puis meurt avant son CHECKPOINT, et l'ouverture « libre » REJOUAIT ce WAL
  étranger avant le remplacement par la copie d'avant (P1). Nouvelle méthode : UNE connexion en
  écriture TIENT le verrou DuckDB de la base du début à la fin ; elle fait le CHECKPOINT, la copie
  vers le fichier neuf et la SAUVEGARDE par `COPY FROM DATABASE` (lire les octets d'un fichier
  tenu est impossible sous Windows, mesuré), les inventaires des deux copies se font sous ce
  verrou ; les vérifications « inchangée » et « libre » disparaissent avec leur code. Un `.wal`
  non vide au lancement est refusé d'entrée. Remplacement par un point de variation
  (`cmd_compact_passes_remplacement_{windows,other}.go`) : POSIX — rename PENDANT que la connexion
  est ouverte, puis fermeture ; Windows — un fichier tenu ne s'y remplace pas (mesuré : « Accès
  refusé ») : fermeture puis rename IMMÉDIAT, sans travail entre les deux. Pourquoi la voie A ne
  rouvre aucune fenêtre de perte sous Windows : un tiers qui ouvre entre la fermeture et le rename
  TIENT la base au moment du rename, le rename échoue, refus propre (fichier neuf retiré, base
  intacte, le tiers continue sur son fichier) ; un tiers qui ouvre après le rename ouvre le fichier
  neuf ; seule borne théorique, nommée dans la doc : un tiers qui ouvrirait, écrirait, ferait son
  CHECKPOINT et fermerait ENTIÈREMENT dans ces quelques microsecondes. La voie C (refuser
  `--rewrite-file` sous Windows) est écartée : l'utilisateur veut réduire sa base locale, sous
  Windows. Toute copie partielle (sauvegarde de la compaction ou de la réécriture, fichier neuf)
  est retirée sur erreur, journalisée.
- DC.6 **Pas d'automatisme** : pas de compaction dans le post-sync ni au boot (elle exige
  l'écrivain exclusif et le serveur arrêté). C'est une opération d'entretien, documentée comme
  telle, à jouer après chaque campagne de redécodage (recuisson, backfill killsource).

Items :
- [x] C.1 inventaire (DC.1) : tables, règle de leur vue, rapport mesuré, lecteurs bruts ; liste
      retenue et exclusions justifiées écrites ici — Journal C, « C.1 » : 13 tables retenues
      (12 à passe + `match_bomb_stats`), toutes les autres vues `_latest` exclues avec leur raison
- [x] C.2 cœur de compaction (DC.2, DC.3) + tests d'intégration : passes multiples, match à une
      seule passe, table vide, crash simulé en cours de swap (`recoverOrphan`), idempotence,
      séquence qui continue, DDL identique, vue identique avant / après — `migration/table_swap.go`
      (cœur extrait d'`append_only_rebuild.go`, qui l'appelle), `compaction_registry.go`,
      `compaction.go` ; tests `compaction_test.go` (9) et
      `games/halo_infinite/migrations/compaction_e2e_test.go` (schéma partagé réel) — Journal C
- [x] C.3 commande CLI (DC.4) : `--dry-run`, refus si la base est tenue, sauvegarde, journal ;
      tests de la commande — `levelup compact-passes` (`cmd/levelup/cmd_compact_passes.go`), tests
      `cmd_compact_passes_test.go` (6, dont un refus INTER-PROCESSUS par processus auxiliaire) — Journal C
- [x] C.4 `--rewrite-file` (DC.5), ou `[!]` prouvé — livré : DuckDB 1.5.5 embarqué préserve tables,
      vues, index, macros et la PROCHAINE valeur des séquences (`compaction_rewrite_test.go`) ;
      `cmd_compact_passes_rewrite.go`, échange seulement si l'inventaire relu SEUL est identique
- [x] C.5 preuve sur copie : seconde copie de la base compactée par la commande ; empreintes des
      vues avant / après identiques pour CHAQUE table retenue ; taille du fichier avant / après ;
      chrono avant / après sur copie (2 threads / 512 Mo) : rencontres et rivaux de la Carrière,
      liste Relations, rencontres de la vue match (Q23b), repli de localisation du lot A, un bloc
      de l'onglet Tactique ; cinq joueurs suivis — 13 / 13 vues identiques (SHA-256 d'une lecture
      ordonnée complète), catalogue identique, 11,27 M -> 1,12 M lignes, 1 264 -> 1 215 Mio puis
      351 Mio par `--rewrite-file` ; chronos : Journal C
- [x] C.6 docs : `docs/COMMANDS.md` + `docs/FR/COMMANDS.md` (commande, quand la jouer, serveur
      arrêté, sauvegarde) ; ligne Ops du changelog 7.5.0 EN + FR (après les recuissons et le
      backfill killsource, serveur arrêté) ; ADR 0036 (section Exceptions : le coût des fenêtres
      suit le nombre de passes, la compaction est l'entretien qui le borne) et renvoi dans l'ADR
      0026 si une règle y change (sinon rien) — sous-section « Compacting superseded decode passes »
      EN + FR, Ops (5) EN + FR, paragraphe sous la table I2 de l'ADR 0036 ; ADR 0026 : aucune règle
      ne change (sa section Conséquences annonçait déjà « un compactage périodique pourra être
      ajouté »), rien
- [x] C.7 mutations jouées : garde de cardinalité retirée, `DELETE` réintroduit (garde-rail
      anti-ART rouge), passe gardée = la première au lieu de la dernière, séquence remise à zéro ;
      chacune rouge puis restaurée — quatre mutations (six variantes), toutes rouges, `cmp` à
      l'appui — Journal C
- [x] C.8 (revue L1, 2026-09-26) échange de fichiers de `--rewrite-file` : jamais de fenêtre sans
      base au chemin, source revérifiée (inchangée, libre) avant le rename unique, fichier neuf
      retiré sur toute erreur ; tests des trois cas rouges sur l'ancien code puis verts — Journal C
- [x] C.9 (revue L6, 2026-09-26) `verifierApresEchange` verrouillée par des tests qui produisent
      les écarts ; orphelin de compaction REFUSÉ (les index secondaires seraient perdus) — Journal C
- [x] C.10 (seconde revue, 2026-09-26) `--rewrite-file` sous verrou TENU du début à la fin (voie A),
      WAL étranger refusé d'entrée, sauvegardes partielles retirées, orphelin refusé avant toute
      sauvegarde — Journal C

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
- **C.2 — mécanisme** (`d20d2d5d4`). Cœur commun extrait d'`append_only_rebuild.go` vers
  `migration/table_swap.go` (`swapTableTx` : cardinalité attendue lue DANS la transaction, DROP
  d'une table de construction périmée, Build, garde `rebuilt == attendu` AVANT le DROP, DROP,
  RENAME, PostRename, `Verify` optionnel avant COMMIT, rollback intégral ; `recoverOrphanTable`
  générique) ; `rebuildAppendOnlyTx` et `recoverOrphanAppendOnly` l'appellent (règle des deux
  copies : aucune copie). Compaction (`compaction.go`) par table : `recoverOrphan` (suffixe
  `__compact`), table et vue présentes (sinon `absente`), SIGNATURE de la règle retrouvée dans le
  SQL de la vue en base (sinon REFUS de la table), comptes brut / gardé, empreinte de la vue
  (`count` + somme des `hash` de ligne), `gardées == brutes` -> `deja-compacte` (idempotence),
  sinon échange : table neuve au DDL EXACT (`duckdb_tables().sql`, seul le nom change), `INSERT …
  SELECT * … <règle>`, DROP, RENAME, index recréés depuis `duckdb_indexes().sql` (DuckDB refuse de
  renommer une table qui porte un index : mesuré), puis vérification avant COMMIT : DDL identique,
  index identiques, empreinte de la vue identique. Séquences jamais touchées (les `id` gardés sont
  recopiés, aucun `nextval`). `CHECKPOINT` final. Tests (`cgo`) : `compaction_test.go` — table
  réelle `match_bomb_stats` compactée 6 -> 3 à vue identique et 12 tables absentes sautées, dry-run
  sans écriture, vue dont la règle a changé refusée, garde de cardinalité (rollback), panne APRÈS le
  DROP et le RENAME (rollback, DDL et index intacts), orphelin `__compact` récupéré puis compaction
  terminée, `ddlDeConstruction`, signatures sur le SQL tel que DuckDB le rend (et une règle
  modifiée qui ne passe plus), garde-rail source `TestCompaction_AucuneMutationDeLigne` ;
  `games/halo_infinite/migrations/compaction_e2e_test.go` — schéma partagé RÉEL (chaîne de migration
  complète) : les 13 tables semées (passes multiples, match à une passe, passe la plus récente au
  nom qui trie avant, doublon dans la passe retenue de `kill_positions`, films à une ligne par passe,
  table vide), dry-run, compaction, sortie de chaque vue identique (lecture ordonnée), tables /
  index / vues / contraintes / séquences identiques, id suivant = max + 1, seconde passe
  `deja-compacte`.
- **C.3 — commande** `levelup compact-passes` (`a74c20a34`) : nom sur le modèle de `rebuild-pme-art`
  (entretien serveur arrêté, verbe + objet ; l'objet est la PASSE). Titres : `--title` (vérifié au
  registre chargé de la config) ou chaque titre du registre dont la base partagée existe ; chemin
  par `PathResolver`. Réel : bail `dblease` (ADR 0013) puis ouverture RW (refus si un autre processus
  tient le fichier), `RunForTitleDB(slug, shared)`, `CHECKPOINT`, fermeture, SAUVEGARDE octet pour
  octet (`<nom>.avant-compaction-<UTC>.duckdb`, refus si WAL non vide), réouverture RW (refus si prise
  entre-temps), compaction, rapport par table + taille du fichier. Écart assumé : la sauvegarde se
  fait fichier FERMÉ — sous Windows un fichier tenu en écriture par DuckDB ne s'ouvre pas en lecture
  (mesuré : « utilisé par un autre processus ») ; l'outillage existant (`backup`, pkg/duckdbbackup)
  exporte en Parquet et perd séquences / vues / index : non réutilisable ici. Dry-run : ouverture en
  LECTURE SEULE, ni migration ni sauvegarde. Tests `cmd_compact_passes_test.go` : dry-run puis
  sauvegarde (qui porte la base d'avant) et compaction, idempotence, refus sur bail tenu, refus quand
  un AUTRE PROCESSUS tient la base (processus auxiliaire `TestAideTenirLaBase`, dry-run et réel),
  `--rewrite-file`, options incompatibles, WAL non vide.
- **C.4 — `--rewrite-file`** livré (`a74c20a34`) : `migration.CopierBaseVers` (ATTACH + `COPY FROM
  DATABASE` + DETACH), `LireInventaire` / `Ecarts` (catalogue : tables, vues, index, séquences par
  leur `sql` qui porte `START <prochaine valeur>`, macros, types ; compte de chaque table ; empreinte
  des vues du registre), relus sur CHAQUE fichier ouvert SEUL ; échange seulement sans écart, ancien
  fichier gardé (`<nom>.avant-reecriture-<UTC>.duckdb`), WAL vides retirés avant l'échange, retour
  de l'ancien si la mise en place échoue. Mesuré sur DuckDB 1.5.5 embarqué
  (`compaction_rewrite_test.go`) : séquence tirée, jamais tirée, `START 10` -> prochaines valeurs
  identiques ; vue sur vue, index, macro préservés. Seul écart constaté, sans lecteur :
  `duckdb_sequences().last_value` affiche la prochaine valeur dans le fichier neuf (aucun code ne
  lit `last_value` ni `currval`, grep).
- **C.5 — preuve sur copie** : seconde copie de la copie du 2026-09-26 14:07 dans
  `scratchpad/compactC/c5root/data/titles/halo_infinite/warehouse/`, commande jouée par
  `LEVELUP_REPO_ROOT` (binaire du worktree). `--dry-run` 10,8 s ; compaction 26,0 s (dont
  `match_kill_events` 12,1 s), migrations 0 appliquée ; seconde passe avec `--rewrite-file` :
  13 / 13 `deja-compacte` (idempotence) puis réécriture en ~10 s. Lignes brutes avant -> après :
  `match_kill_events` 3 953 799 -> 412 216, `match_lives` 1 725 664 -> 163 092,
  `match_death_context` 1 516 693 -> 146 654, `kill_openings` 1 495 845 -> 147 225,
  `kill_positions` 1 495 728 -> 147 360, `match_weapon_shots` 896 590 -> 74 882,
  `match_player_positions` 167 189 -> 22 198, `match_usage_films` 1 262 -> 152,
  `match_usage_players` 11 465 -> 1 420, `match_pad_pickups_by_tier` 5 590 -> 1 502,
  `match_flag_grabs_net` 353 -> 147, `match_weapon_hit_distance` et `match_bomb_stats` 0 ; total
  11 270 178 -> 1 116 848. Empreintes (SHA-256 d'une lecture ordonnée complète `SELECT * FROM
  <vue> ORDER BY id`, sonde temporaire supprimée) : 13 / 13 IDENTIQUES avant, après compaction et
  après réécriture (ex. `match_kill_events_latest` 412 216 lignes `95b4279b…5fd8dd`,
  `kill_positions_latest` 147 360 `9012c713…735f33`). Catalogue (tables, index, vues, séquences,
  contraintes : 356 lignes) identique avant / après / réécrit ; séquence `match_kill_events_id_seq`
  intouchée (dernier tiré 3 953 800, `max(id)` 3 953 799). Fichier : 1 264,3 Mio -> 1 215,0 Mio
  (compaction seule) -> 350,8 Mio (`--rewrite-file`). Chronos (2 threads / 512 Mo, machine
  partagée, deux tours, avant -> après compaction ; le fichier réécrit donne les mêmes ordres) :
  Carrière rencontres Q26 JGtm 1 829-1 870 -> 267-275 ms, Madina97294 1 622-1 909 -> 364-422,
  Chocoboflor 2 171-2 449 -> 238-263, XxDaemonGamerxX 1 829-2 131 -> 277-293, Nuzzles 2 192-2 349
  -> 609-761 ; rivaux Q27 JGtm 3 066-3 806 -> 343-425, Madina 3 727-3 906 -> 381-486, Chocoboflor
  3 513-4 277 -> 411-418, Xx 3 391-3 685 -> 282-439, Nuzzles 3 306-3 935 -> 604-642 ; Relations
  Q28 JGtm 1 916-2 428 -> 374-462, Madina 2 463-2 894 -> 479-505, Chocoboflor 1 846-2 077 ->
  372-383, Xx 1 770-1 942 -> 164-277, Nuzzles 3 309-3 688 -> 1 597-1 688 ; vue match Q23b (médiane
  de 11 matchs) JGtm 1 909-2 466 -> 236-296, Nuzzles 1 922-2 159 -> 230-242 ; Tactique
  `MortsParCarte` JGtm 2 696-2 933 -> 397-499, Madina 3 506-3 569 -> 355-493, Chocoboflor 3 025-3 252
  -> 601-641, Xx 2 899-3 017 -> 346-492, Nuzzles 3 715-4 972 -> 451-517 ; repli de localisation du
  lot A (pas 1 seul, 2 973 xuids de Nuzzles) 192-252 -> 61-90 ms, mêmes 2 matchs candidats ; Q12
  des 23 matchs à repli de JGtm, en alternance avant / après (trois tours chacun) : médiane
  13-24 -> 13-15 ms, max 133-292 -> 39-78 ms.
- **C.7 — mutations** (chacune appliquée, test rouge, fichier restauré, `cmp` identique) :
  (1) garde de cardinalité retirée de `swapTableTx` -> `TestSwapTableTx_GardeDeCardinalite_Rollback`
  rouge (« attendu l'abandon de la garde, got <nil> ») ; (2) `DELETE FROM <table>__compact WHERE id
  NOT IN (…)` réintroduit dans l'échange -> `TestCompaction_AucuneMutationDeLigne` rouge (le
  résultat serait le même : seul le garde-rail le voit, et c'est son rôle — les scans de
  `internal/sync` excluent `internal/migration`) ; (3) passe gardée = la PREMIÈRE (`keep` en `ASC`,
  signature inchangée) -> `TestCompaction_SchemaReel_BoutABout` rouge (dry-run : `kill_positions`
  garderait 7 lignes et non 8), et avec la vérification avant COMMIT neutralisée, rouge de même ;
  variante sur la dernière ligne par clé (`match_bomb_stats`) -> la vérification avant COMMIT
  refuse (« la vue match_bomb_stats_latest ne rend plus le même résultat », rollback), et
  vérification neutralisée -> `TestCompaction_BombStats_VueIdentiqueEtTablesAbsentes` rouge
  (« vue changée ») ; (4) séquence remise à zéro (séquence neuve `START 1` posée en défaut de
  `id` après le RENAME) -> la vérification avant COMMIT refuse (« DDL de match_kill_events
  changé », rollback) et, vérification neutralisée, `TestCompaction_SchemaReel_BoutABout` rouge
  (« schéma changé »).
- **Gate C** (code `a74c20a34`, depuis `apps/go-api`, `CGO_ENABLED=1`, GOCACHE privé) :
  `gofmt -l ./internal ./cmd` vide ; `go build ./...` 0 ; `go vet ./...` 0 ; `go test ./...` en
  quatre tranches de `go list ./...` (343 paquets ; la limite de 10 min par commande interdit un
  seul appel) : 4 codes de sortie 0, 190 `ok`, 153 sans test, 0 `FAIL` ; `go test -tags=integration
  -p 1` sur les 344 paquets de `go list -tags=integration ./...`, en tranches (codes de sortie lus
  un par un, pas une sortie filtrée) : toutes à 0 SAUF une, `internal/service` rouge sur
  `TestRelationsSegmentation_SoloVsSquad_CrossDB` et `_PlaylistFilter_CrossDB` — PRÉEXISTANT,
  rouge à l'identique sur l'arbre exporté de `34edf29af`, hors du diff de C (découverte (9)) ;
  seconde passe complète `-json -count=1` pour la baseline : mêmes deux rouges, plus
  `TestLUSRV2Shadow_RafalesBornees_300Candidats` (seuil de 2 s dépassé de 2 à 62 ms, machine
  chargée) — rejoué seul : vert ; garde-rails `internal/sync` (`NoART|NoRaw|Allowlist|Bulk|
  Interpolated|Legacy|Sentinel`) verts (dont `TestNoARTPatternsOnProtectedTables`,
  `TestNoRawDeleteOnAppendOnlyTables`, `TestNoInterpolatedWriteOnProtectedTables`),
  `TestNoRawAppendOnlyReads` vert, sentinelle `platform/auth` verte ; golangci-lint
  `--new-from-rev=f04b9fb78 ./...` 0 issue, idem `--build-tags=integration` ; baseline
  (`scripts/check_test_baseline.sh tests --from-jsonl` sur le JSONL complet) : « Tous les tests
  baseline présents » (9 693 / 18 454), verdict d'échec = les trois tests ci-dessus seulement ;
  aucun test renommé ni supprimé (baseline JSONL inchangée) ; sonde `cmd/compaction_probe_tmp`
  absente ; artefacts de test sous `data/` (rasters, `halo_5/warehouse/metadata.duckdb`) retirés.
- **Revue C** (deux relecteurs aveugles) : NON FAITE par l'exécuteur — sous-agents interdits par
  le brief ; au superviseur.
- **C.8 — échange de fichiers de `--rewrite-file`** (revue adversariale L1, trois constats retenus,
  les deux derniers bloquants). Constats : (1) `os.Rename(base, ancien)` vers un `--backup-dir` sur
  un autre volume échoue et laissait `<base>.reecriture`, qui bloquait toute réécriture suivante ;
  (2) entre les deux renames, plus aucun fichier au chemin de la base (un redémarrage aurait créé
  une base vide) ; (3) rien ne revérifiait la source entre la copie et l'échange (sous Linux, un
  serveur ouvert dans cette fenêtre écrit dans l'inode mis de côté). Correctif
  (`cmd/levelup/cmd_compact_passes_rewrite.go`, DC.5 amendé ci-dessus) : copie de sauvegarde,
  vérifications « inchangée » (taille + date) et « libre » (OpenReadWrite exclusif puis
  fermeture), un seul `rename(neuf, base)` ; abandon qui retire `<base>.reecriture` et la
  sauvegarde, journalisé ; un reste de réécriture interrompue trouvé au départ est retiré (WARN)
  — la base est complète par construction. Points d'observation `etapeReecriture` (nil en prod) et
  `renommer` (os.Rename) pour les tests. Tests (`cmd_compact_passes_rewrite_test.go`) :
  `TestReecriture_SauvegardeSurUnAutreVolume` (aucun second volume sur le poste : `renommer` échoue
  comme EXDEV / `MoveFileEx` sans COPY_ALLOWED dès que source et cible sont dans deux dossiers),
  `TestReecriture_JamaisSansBaseAuChemin` (à chaque étape, une base complète et lisible au chemin ;
  arrêt simulé à chacune des quatre étapes : base complète, aucun reste),
  `TestReecriture_SourceModifieeApresLaCopie` (INSERT entre sauvegarde et échange : refus « a
  changé depuis la copie », la ligne est conservée, aucun reste),
  `TestReecriture_SourceTenueAvantLEchange` (processus auxiliaire qui tient la base : refus « tenue
  par un autre processus », aucun reste ; l'ouverture de l'auxiliaire ne change pas la date du
  fichier, c'est bien la vérification « libre » qui refuse). Démonstration sur l'ANCIEN code (celui
  de `a74c20a34`, instrumenté des seuls points d'observation, sans autre changement) : les quatre
  tests ROUGES — autre volume : « mise de côté … volume différent » ; jamais sans base : rouge aux
  arrêts `apres-sauvegarde` (reste `.reecriture`) et `apres-verification` (« aucun fichier au
  chemin de la base ») et au passage sans arrêt ; source modifiée : échange fait, got <nil> ;
  source tenue : échange FAIT sous Windows aussi (got <nil> : DuckDB ouvre avec partage de
  suppression, l'écriture de l'autre processus aurait été perdue). Nouveau code : verts.
  Mutations sur le nouveau code : vérification « inchangée » retirée -> `…SourceModifiee…` rouge ;
  vérification « libre » retirée -> `…SourceTenue…` rouge (sous Windows le rename échoue alors sur
  le fichier tenu, sous Linux il réussirait). Doc `docs/COMMANDS.md` + FR mise à jour
  (`--backup-dir` : sauvegardes = copies, tout volume ; rename unique ; refus). Le test
  `TestCompactPasses_RefusSiUnAutreProcessusTientLaBase` partage désormais l'aide `tenirLaBase`
  (nom inchangé).
  Gate C.8 : `gofmt` vide ; `go vet ./cmd/levelup/ ./internal/migration/...` 0 ; `go test
  ./cmd/levelup/ ./internal/migration/...` 0 (2 ok) ; `go test -tags=integration -p 1 -count=1
  ./cmd/levelup/ ./internal/migration/... ./internal/games/halo_infinite/migrations/...` 0 (3 ok) ;
  golangci-lint `--new-from-rev=34edf29af ./...` 0 issue, idem `--build-tags=integration`.
- **C.9 — couverture de la vérification et orphelin** (revue L6, deux constats P2 retenus).
  (1) `verifierApresEchange` n'était verrouillée par rien (corps remplacé par `return nil` : suite
  verte). Tests ajoutés (`internal/migration/compaction_test.go`) :
  `TestCompaction_VueQuiRetientPlusQueLaRegle_RollbackIntegral` — vue réelle qui porte la
  signature mais retient plus que la règle (`… = 1 OR bomb_arms = 1` garde une version ancienne) :
  la compaction retirerait une ligne servie, la vérification refuse (« ne rend plus le même
  résultat »), rollback intégral (6 lignes, DDL, index, sortie de la vue identiques, aucune
  `__compact`) ; `TestVerifierApresEchange_EcartsDeDDLEtDIndex` — la vérification lit le vrai
  catalogue dans une transaction et refuse un DDL qui n'est plus celui d'avant (un `NOT NULL` en
  moins) et une liste d'index qui a perdu un index, accepte le schéma identique (un écart de DDL
  ou d'index « réel » exigerait une construction défectueuse : pas de moyen propre de le produire
  sans point d'injection). Rouges : corps `return nil` -> les deux rouges (« got <nil> ») ;
  comparaison d'index seule neutralisée -> cas « index » rouge. (2) Orphelin : la table de
  construction ne porte pas les index secondaires, la récupération les perdait pour de bon.
  CHOIX = refus (`refuserOrphelin`, DC.3 amendé, justification ci-dessus) plutôt que reposer les
  index. `TestCompaction_RecupereUnOrphelin` (test de ce lot, absent de la baseline) devient
  `TestCompaction_OrphelinRefuse` : refus en dry-run et pour de bon, orpheline et absence de la
  table laissées telles quelles, et — si une version récupère quand même — comparaison de
  `duckdb_indexes()` avant / après. Rouge sur l'ancien code (récupération par
  `recoverOrphanTable`) : « orpheline récupérée SANS ses index : avant [CREATE INDEX
  idx_match_bomb_stats_match …], après [] ».
  Gate C.9 : `gofmt` vide ; `go vet` 0 ; `go test ./cmd/levelup/ ./internal/migration/...` 0 ;
  `go test -tags=integration -p 1 -count=1 ./cmd/levelup/ ./internal/migration/...
  ./internal/games/halo_infinite/migrations/...` 0 (3 ok) ; golangci-lint `--new-from-rev=34edf29af`
  0 issue, idem `--build-tags=integration`.
- **C.10 — `--rewrite-file` sous verrou tenu** (seconde revue : un P1, deux P2 ; méthode décidée
  par le superviseur, voie A, DC.5 amendé de nouveau). Mesures Windows préalables (sonde retirée),
  NOTRE connexion DuckDB tenant la base après CHECKPOINT : lecture des octets du fichier ÉCHOUE
  (« utilisé par un autre processus ») ; `os.Rename(neuf, base)` ÉCHOUE (« Accès refusé ») ; le
  même rename après fermeture réussit. Code : `cmd_compact_passes_rewrite.go` (une connexion
  `r.source` tient la base : CHECKPOINT, inventaire, `CopierBaseVers` vers le fichier neuf puis
  vers la sauvegarde, inventaire des deux copies relues seules, sous le verrou ; refus d'entrée
  d'un `.wal` non vide ; `abandonner` ferme, retire fichier neuf et sauvegarde, journalise ; une
  erreur APRÈS le remplacement ne retire rien), `cmd_compact_passes_remplacement_windows.go`
  (fermeture puis rename immédiat, refus si le rename échoue) et `..._other.go` (rename sous
  verrou puis fermeture). Retirés avec leur code (zéro code mort) : `etatFichier`,
  `lireEtatFichier`, `verifierSourceInchangee`, `verifierSourceLibre`, `echanger`. P2 :
  `copierFichier` (sauvegarde de la compaction) retire sa cible sur toute erreur après sa création
  (journalisé), `cheminDeSauvegarde` extrait ; `migration.VerifierAucunOrphelin` appelée par la
  commande AVANT les migrations et la sauvegarde ; message et commentaire de `refuserOrphelin`
  corrigés (sauvegarde de l'exécution interrompue, écritures postérieures perdues). Tests
  (`cmd_compact_passes_rewrite_test.go`, `cmd_compact_passes_test.go`) : AJOUTÉS
  `TestReecriture_TiersRefuseDuDebutALaFin` (processus auxiliaire `TestAideEssayerLaBase` à chacune
  des quatre étapes sous verrou), `TestReecriture_TiersTientLaBaseAuRemplacement_Windows`,
  `TestReecriture_RemplacementSousVerrou_POSIX` (`t.Skip` sous Windows, raison écrite : prouvé par
  la CI Linux seulement), `TestReecriture_WALNonVideAuLancement` (WAL laissé par
  `disable_checkpoint_on_shutdown` : refus, WAL intact, la ligne revient à la réouverture),
  `TestCopierFichier_EchecEnCoursDeCopieNeLaisseRien` (la « base » est un dossier : la lecture
  échoue après la création de la cible), `TestCompactPasses_OrphelinRefuseAvantTouteSauvegarde`
  (réel, dry-run, `--rewrite-file`) ; GARDÉS et adaptés `TestReecriture_JamaisSansBaseAuChemin`
  (étapes observées puis arrêt à chacune, aucun WAL resté à côté de la base réécrite) et
  `TestReecriture_SauvegardeSurUnAutreVolume` ; RETIRÉS (leur objet disparaît avec les
  vérifications) `TestReecriture_SourceModifieeApresLaCopie` et
  `TestReecriture_SourceTenueAvantLEchange` — tests de ce lot, absents de la baseline.
  Démonstration sur le code de C.9 (instrumenté des seuls points d'observation et de la signature
  de `copierFichier`) : ROUGES — `TiersRefuseDuDebutALaFin` (un tiers ouvre la base aux quatre
  étapes), `TiersTientLaBaseAuRemplacement_Windows` (got <nil> : le point `apres-fermeture` n'existe
  pas, l'échange se fait — comportement nouveau, le C.9 refusait aussi par sa vérification
  « libre » dans ce cas), `WALNonVideAuLancement` (got <nil>), `CopierFichier_EchecEnCours…`
  (copie partielle laissée), `OrphelinRefuseAvantTouteSauvegarde` (« sauvegarde prise malgré
  l'orphelin ») ; verts sur C.9 : `JamaisSansBaseAuChemin` et `SauvegardeSurUnAutreVolume`
  (conservés, déjà acquis en C.8). Nouveau code : tous verts sous Windows, le test POSIX saute.
  Limites : la variante `_other.go` ne compile ni ne se teste sur ce poste (pas de chaîne cgo
  Linux) : compilation, lint et `TestReecriture_RemplacementSousVerrou_POSIX` relèvent de la CI.
  Gate C.10 : `gofmt` vide ; `go vet` 0 ; `go test ./cmd/levelup/ ./internal/migration/...` 0 ;
  `go test -tags=integration -p 1 -count=1 ./cmd/levelup/ ./internal/migration/...
  ./internal/games/halo_infinite/migrations/...` 0 (3 ok) ; golangci-lint `--new-from-rev=34edf29af`
  0 issue, idem `--build-tags=integration` (lint sous GOOS=windows).

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
- [x] B.0 re-mesure des cibles sur la copie compactée (DB.1), liste traitée écrite ici — Journal B,
      « B.0 » : traitées Q26, Q27, Q28 (non scopé), `MortsParCarte` ; `[~]` Q28 scopé et Accueil
      (arme favorite) ; `[!]` (DB.5) Synthèse `weapon_records` et l'annuaire de Relations de Nuzzles
- [x] B.1 Carrière : rencontres et rivaux bornés, rivaux en une lecture ; parité ; chrono —
      `f81a8a09c` (`career_repo_encounters.go`, `career_repo_rivals.go`, `perimetre_liste.go`) ;
      Journal B
- [x] B.2 Relations : Q28 et Q28 scopé bornés ; parité ; chrono — `f81a8a09c` : Q28 sans liste
      RETIRÉ, l'historique passe par la requête scopée (même liste, un paramètre `VARCHAR[]`) ; Q28
      scopé était déjà borné (B.0), il change seulement de forme de liaison ; Journal B
- [x] B.3 Tactique `MortsParCarte` : fenêtres du journal des morts et des positions bornées ;
      parité ; chrono — `f81a8a09c` ; Journal B
- [!] B.4 Accueil et Synthèse (`weapon_records`) : selon B.0 et DB.5 — Accueil `[~]` résolu par C
      (sa seule fenêtre `_latest` d'historique complet, l'arme favorite : 0,10-0,19 s ; la page
      entière ne se re-mesure pas sans serveur) ; `weapon_records` `[!]` DB.5 (fenêtres déjà
      bornées, coût = volume des matchs du joueur, 0,05-0,68 s) ; rien codé ; à revoir avec
      l'utilisateur (§6)
- [x] B.5 garde-rails (DB.4), sections de durée, ADR 0036 mis à jour — `f81a8a09c` (tests),
      `3e1683842` (ADR) ; sections `top_encounters`, `rivals`, `relations` et leurs `_annuaire`
      conservées (`rivals` : un appel au lieu de deux) ; Journal B
- [x] B.6 mutations : liste retirée de sous la fenêtre (test de fenêtres rouge), rivaux relus deux
      fois (test de comptage rouge), exclusion Campagne retirée (parité rouge) — six mutations,
      toutes rouges ; Journal B

Gate B : comme C sans l'intégration complète : `gofmt`, build, vet, `go test ./internal/service/...
./internal/platform/duckdb/... ./internal/analysis/... ./internal/api/... ./internal/archlint/...`,
`go test -tags=integration -p 1 ./internal/platform/duckdb/...`, golangci-lint.

Revue B : un relecteur aveugle (lentilles SQL / périmètre et parité des chiffres).

Journal B (2026-09-27, exécuteur Opus, worktree `LevelUp-wt-perf-perimetre`, base `fb1e65d67`) :

- **B.0 — re-mesure sur la copie COMPACTÉE ET RÉÉCRITE de C.5** (copie privée du fichier de
  `compactC/c5root`, 351 Mio, lecture seule, 2 threads / 512 Mo ; sonde temporaire qui appelle les
  vrais lecteurs, sections de durée relevées ; deux tours, min-max ; machine moins chargée qu'en
  C.5). Règle de décision : une cible est TRAITÉE si au moins un des cinq joueurs dépasse 300 ms
  aux deux tours.
  - Carrière rencontres (`GetTopEncountersGlobal`, Q26 + annuaire) : JGtm 217-281 ms, Madina97294
    228-336, Chocoboflor 160-178, XxDaemonGamerxX 142-145, Nuzzles 435-513 (Q26 214-279, annuaire
    220-233) -> **TRAITÉE** (B.1).
  - Carrière rivaux (`GetRivals`, Q27 lu deux fois + annuaire) : JGtm 287-461, Madina 291-451,
    Chocoboflor 249-296, Xx 233-273, Nuzzles 333-375 -> **TRAITÉE** (B.1).
  - Relations Q28 non scopé (`GetRelations(nil)`) : JGtm 299-373 (Q28 165-240, annuaire 133),
    Madina 376-417 (Q28 210-245, annuaire 165-172), Chocoboflor 213-229, Xx 134-136, Nuzzles
    982-1 030 (Q28 230-254, **annuaire 752-775**) -> **TRAITÉE** (B.2) pour la fenêtre du
    kill-feed de Q28 ; l'annuaire de Nuzzles n'est PAS une fenêtre sur tout l'historique (ses
    lectures sont des tables et des fenêtres déjà bornées par liste, lot A) : DB.5, découverte au §6.
  - Relations Q28 scopé (30 derniers matchs) : 42-59 ms pour les cinq -> `[~]` : la variante
    scopée lie déjà la liste en constantes sous la fenêtre du kill-feed (`kv.match_id IN (…)`).
  - Tactique `MortsParCarte`, appelée comme par la page (liste blanche = tous les matchs du
    joueur) : JGtm 301-386, Madina 331-343, Chocoboflor 273-278, Xx 229-246, Nuzzles 318-326 ;
    liste de 30 matchs : 225-396 ; sans liste : 224-572 -> **TRAITÉE** (B.3) : la liste n'est
    posée que sur `match_registry`, les deux fenêtres (positions, journal) voient toute la table.
  - Accueil : la seule fenêtre `_latest` du film sur tout l'historique est l'arme favorite
    (`LoadFavoriteWeapon`, source de dégât) : JGtm 119-189, Madina 103-124, Chocoboflor 99-118,
    Xx 98-103, Nuzzles 100-102 -> `[~]` résolu par C ; l'historique canonique de la page
    (`PlayerMatchesRepo.Load`, bases joueur copiées, Nuzzles sans base) : 43-214 ms, pas une
    fenêtre. La page entière (1,6 s pour 310 Ko en septembre) ne se re-mesure pas sans serveur
    (interdit) : §6.
  - Synthèse `weapon_records` (`LoadWeaponRange`, liste = tous les matchs du joueur) : JGtm
    335-665, Madina 269-299, Chocoboflor 177-182, Xx 49-62, Nuzzles 665-677 -> `[!]` DB.5 : ses
    fenêtres sont DÉJÀ bornées à la liste, profil JSON de la requête relevé sur la copie
    (fenêtres vues : JGtm 102 235 kill-feed / 96 633 positions = exactement les lignes de ses
    matchs, Nuzzles 272 132 / 19 978) ; le coût est le volume des matchs du joueur, pas une
    fenêtre sur tout l'historique.
  - Coût de la liste elle-même (lecture du kill-feed d'un joueur, `COUNT(*)`, deux tours) : sans
    liste 109-167 ms (fenêtre de 412 216 lignes) ; liste de ses matchs en constantes : JGtm 66-292
    (102 235 lignes vues), Madina 48-55, Chocoboflor 35-39, Xx 16-18, **Nuzzles 155-173** (7 190
    matchs, 272 132 lignes vues : la liste coûte plus que la fenêtre entière, déjà écrit à l'ADR
    0036 pour les longues listes). Le mécanisme DB.2 reste celui du plan ; sa FORME de liaison est tranchée en B.1.
- **B.1 à B.3 — mécanisme** (`f81a8a09c`). Chaque lecture traitée lit d'abord la liste (les
  matchs du joueur, `QMatchsDuJoueurTpl`, Campagne exclue ; ou la liste blanche de la page pour
  `MortsParCarte` et Q28 scopé), la lie sous CHAQUE vue `_latest` qu'elle lit, puis passe la même
  liste à son annuaire (une lecture de la liste par appel ; l'annuaire la relisait déjà).
  FORME DE LIAISON, décidée sur mesure : un premier jet en `IN (?, ?, …)` faisait RÉGRESSER Nuzzles
  (Q26 615-729 -> 726-809 ms, Q28 1 543-1 586 -> 1 548-1 788, `MortsParCarte` 492-598 -> 613-743,
  trois tours alternés) ; sonde sur la lecture du kill-feed d'un joueur (trois tours) : 7 190
  paramètres en `IN` 304-341 ms, la même liste en littéraux 196-220, en UN paramètre `VARCHAR[]`
  (`list_contains(?::VARCHAR[], match_id)`) 192-249 (fenêtre bornée : 272 132 lignes vues), sans
  liste 195-214 ; JGtm (1 160) : `IN` 98-107, `VARCHAR[]` 64-87, sans liste 223-274. Le coût d'un
  long `IN` est la liaison de ses paramètres : la liste est donc liée en UN paramètre par vue
  (`clauseListeMatchs`, `platform/duckdb/perimetre_liste.go`, helper unique), c'est toujours une
  constante sous la fenêtre (vérifié par EXPLAIN ANALYZE). Détail par lecture : Q26 — `kv_stats`
  borné ; Q27 — UNE lecture de l'agrégat de tous les adversaires, `classerRivaux` trie en Go
  (clé DESC, matchs DESC, xuid ASC — l'ordre total de l'ancien `ORDER BY`, octet par octet) et
  prend dix lignes par classement, `career_repo_rivals.go` (sorti de `career_repo_encounters.go`) ;
  Q28 — la variante sans liste (`Q28RelationsTpl`) est RETIRÉE, l'historique passe par
  `Q28RelationsScopedTpl` (la clause `my_history` y est redondante avec la liste de l'historique,
  sans effet sur les lignes) ; `MortsParCarte` — la liste sur `kp.match_id` ET `e.match_id`, plus
  sur `mr.match_id` (la jointure du registre sur `kp` la porte déjà) ; sans liste blanche (aucun
  appelant de production), la liste des matchs du joueur. `queries_career_encounters.go` passe de
  565 à 484 lignes (gelé au-delà de 500 avant). Une liste vide rend la main sans requête, avec le
  même résultat qu'avant (nil / carte vide / `[]` pour un scope vide).
- **Parité (DB.3)** : sortie des vrais lecteurs, sérialisée en JSON, AVANT (code `fb1e65d67`) /
  APRÈS sur la copie compactée, cinq joueurs, neuf lectures (49 comparaisons) : 29 IDENTIQUES à
  l'octet, ordre compris (Q26 : 10 rencontres + 10 stats par joueur ; Q27 : 10 némésis + 10
  souffre-douleur ; Q28 : 1 231 / 2 938 / 490 / 21 / 7 450 lignes = 12 130 ; Q28 scopé 30 matchs :
  159 lignes ; arme favorite ; historique canonique), 20 identiques EN ENSEMBLES là où l'ordre
  n'était pas total (`MortsParCarte` sans `ORDER BY` : 26 784 morts sur liste complète, 1 364 sur
  30 matchs, idem sans liste ; `weapon_records`, non modifié, agrégat par hachage) ; noms compris.
  Données : 0 ligne du kill-feed des cinq joueurs hors de leurs matchs (HI, pas de Campagne). SEUL
  ÉCART VOULU, Halo 5 : les jambes kill-feed de Q26 / Q27 / Q28 n'excluaient pas la Campagne, la
  liste l'exclut (comme l'historique dont ces lignes relèvent) — copie H5 : 118 morts entre deux
  joueurs dans 63 matchs de Campagne ne comptent plus (test `…CampagneHorsDeLaListe`).
- **Chronos** (copie compactée, 2 threads / 512 Mo, lecteur complet avec annuaire, trois tours
  ALTERNÉS avant / après, machine chargée, min-max) : Q26 JGtm 282-301 -> 169-287 ms (cinq tours :
  228-295 -> 164-187), Madina 324-395 -> 193-267, Chocoboflor 180-298 -> 100-107, Xx 144-161 ->
  42-46, Nuzzles 497-575 -> 533-712 (cinq tours : médiane 519 -> 579 ; requête seule, quatre tours
  alternés : 328-530 -> 317-638, neutre au bruit près) ; Q27 JGtm 460-465 -> 133-247, Madina
  371-500 -> 122-163, Chocoboflor 303-380 -> 75-90, Xx 235-257 -> 25-30, Nuzzles 388-457 -> 306-354 ;
  Q28 JGtm 371-421 -> 253-437 (cinq tours : 302-391 -> 233-272), Madina 409-451 -> 308-333,
  Chocoboflor 242-392 -> 150-168, Xx 139-168 -> 43-44, Nuzzles 1 097-1 296 -> 1 144-1 224 (cinq
  tours : médiane 1 061 -> 1 158 ; l'annuaire en fait 0,8-0,9 s) ; Q28 scopé 30 matchs 43-128 ->
  31-55 ; `MortsParCarte` liste complète JGtm 371-622 -> 227-383, Madina 351-507 -> 162-241,
  Chocoboflor 308-471 -> 115-150, Xx 269-283 -> 28-35, Nuzzles 356-588 -> 238-260 ; liste de 30
  matchs 237-480 -> 23-53 pour les cinq. Nuzzles : sa liste (7 190 matchs) couvre 66 % des lignes
  du kill-feed, la borne n'y gagne presque rien sur Q26 / Q28.
- **B.5 — garde-rails** (`f81a8a09c`, suite par défaut) : `career_repo_fenetres_test.go` —
  `TestLecturesHistorique_FenetresBorneesAuxMatchsDuJoueur` (quatre matchs du joueur parmi dix :
  Q26, Q27, Q28 historique ET scopé, `MortsParCarte` sans liste blanche, chaque fenêtre rejouée
  sous EXPLAIN ANALYZE par `exigerFenetresBornees` ; plus UNE seule lecture de l'agrégat Q27, par
  le carnet de requêtes) et `TestLecturesHistorique_CampagneHorsDeLaListe` (Halo 5, duel de
  Campagne hors des rivaux et des frags échangés) ; `tactical_repo_fenetres_test.go` —
  `MortsParCarte` restreinte à deux matchs ; `career_repo_rivals_test.go` —
  `TestClasserRivaux_OrdreTotalEtLimite` (ex aequo, limite, agrégat non modifié). Démonstration
  sur le code de `fb1e65d67` (arbre exporté, mêmes tests) : ROUGES — fenêtre de 30 lignes pour une
  borne de 12 (Q26) et de 6 (`MortsParCarte`), agrégat Q27 lu 2 fois, souffre-douleur à 2 frags
  sur 2 matchs (Campagne comptée). `TestCareerRepo_Annuaire_SectionsDeDuree` (intégration) attend
  désormais `rivals` : 1 appel. `campaign_exclusion_guard_test.go` : l'entrée du template retiré
  sort de la liste. ADR 0036 (`3e1683842`) : trois lignes retirées de la table I2 avec leur coût
  après, lignes I3 (Accueil, `weapon_records`) re-mesurées, garde-rails I2 complétés, conséquence
  sur la liaison d'une longue liste.
- **B.6 — mutations** (chacune appliquée, test rouge, fichier restauré, `cmp` identique) :
  (1a) liste liée en semi-jointure (`col IN (SELECT unnest(?::VARCHAR[]))` dans `clauseListeMatchs`)
  -> `…FenetresBorneesAuxMatchsDuJoueur` rouge sur les cinq lectures (fenêtres de 30 lignes pour
  12 et 6) et `TestTacticalRepo_PerimetreRestreint_FenetresBornees` rouge (`MortsParCarte`) ;
  (1b) liste retirée de sous la fenêtre du journal de `MortsParCarte` (`OR TRUE`) -> les deux tests
  de fenêtres rouges ; (1c) liste retirée de sous la fenêtre de Q27 -> rouge (« GetRivals : une
  fenêtre a vu 30 lignes, borne 12 ») ; (2) rivaux relus deux fois (un agrégat par classement) ->
  rouge (« agrégat Q27 lu 2 fois, want 1 ») ; (3) exclusion Campagne retirée de
  `QMatchsDuJoueurTpl` -> `…CampagneHorsDeLaListe` rouge (2 frags sur 2 matchs) ; (4) départage par
  xuid retiré de `classerRivaux` -> `TestClasserRivaux_OrdreTotalEtLimite` rouge.
- **Gate B** (code `f81a8a09c`, depuis `apps/go-api`, `CGO_ENABLED=1`, GOCACHE privé) : `gofmt -l
  ./internal ./cmd` vide ; `go build ./...` 0 ; `go vet ./...` 0 ; `go test ./internal/service/...
  ./internal/platform/duckdb/... ./internal/analysis/... ./internal/api/... ./internal/archlint/...`
  : 32 `ok`, 2 sans test, UN rouge — `internal/archlint` `TestAucunSuffixeDePlateformeAccidentel` sur
  `cmd/levelup/cmd_compact_passes_remplacement_windows.go` (fichier de C.10), rouge à l'identique
  sur l'arbre exporté de `fb1e65d67`, hors du diff de B (découverte (11)) ; le reste d'`archlint`
  vert (`-skip` de ce seul test) ; `go test -tags=integration -p 1 -timeout 30m
  ./internal/platform/duckdb/...` 0 (5 `ok`, `duckdb` 470 s) ; golangci-lint
  `--new-from-rev=fb1e65d67 ./...` 0 issue, idem `--build-tags=integration` (lancés avec
  `--allow-parallel-runners` : un autre golangci-lint tenait le verrou du poste) ; aucun test retiré
  ni renommé (`git diff` : 0 `func Test` supprimé) ; sonde `cmd/perimetre_probe_tmp` absente ;
  artefact `data/titles/halo_5/warehouse/metadata.duckdb` créé par les tests, retiré. Hors gate,
  pour contrôle : `go test -tags=integration -run 'Relations|Career|Rival|Tactical|Encounter'
  ./internal/service/...` vert.
- **Revue B** (un relecteur aveugle) : NON FAITE par l'exécuteur — sous-agents interdits par le
  brief ; au superviseur.

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

- (C, 2026-09-26) (1) Au moins quinze swaps « table neuve + RENAME » écrits à la main hors du cœur
  commun (`grep "RENAME TO"` : `games/halo_infinite/migrations/steps*.go` x8,
  `migration/steps_metadata_*`, `steps_player_append_only_match_enrichment.go`,
  `steps_shared_rebuild_match_participants.go`, `steps_shared_social_media_files_*`,
  `ops/records_purge.go`, `cmd/purge_foreign_lusr_chain`) : dette antérieure, le cœur
  `migration/table_swap.go` existe désormais pour eux ; non migrés (hors périmètre), aucun
  garde-rail n'interdit une nouvelle copie. (2) `migrerSchemaPartage` (`cmd/levelup/
  cmd_backfill_killsource.go`, appelé aussi par `backfill-flag-grabs-net`, `-pad-tiers`, …) reçoit
  le slug mais appelle `migration.RunForDB`, qui force `DefaultSlug` : sur `--title halo_5`, ce
  serait le jeu de migrations de Halo Infinite (le commentaire d'`applySharedMigrationsForTitle`
  dit exactement ce piège) ; `compact-passes` appelle `RunForTitleDB`. Non vérifié en exécution.
  (3) Dans UN MÊME processus, une seconde instance DuckDB (`sql.Open` hors du cache du paquet
  `duckdb`) ouvre en écriture un fichier qu'une autre instance tient déjà en écriture, sans refus
  (mesuré par le premier jet du test de refus) : seul le verrou INTER-processus protège, le bail
  `dblease` étant le seul garde intra-processus. (4) Sous Windows, un fichier que DuckDB tient en
  écriture ne s'ouvre pas en lecture par un autre handle (« utilisé par un autre processus ») :
  toute copie de sauvegarde « à chaud » d'une base tenue échoue sur le poste local (la
  sauvegarde de `compact-passes` se fait donc fichier fermé). (5) Les scans anti-ART de
  `internal/sync` (`dansLePerimetreART`) excluent `internal/migration` : du code de maintenance
  qui y vit et tourne hors du boot (la compaction) n'est gardé que par son propre test
  (`TestCompaction_AucuneMutationDeLigne`). (6) Halo 5 : `kill_positions` porte une passe
  synthétique `legacy-<match>` par match, et sa vue écarte 2 606 doublons (297 963 brutes /
  295 357 servies) À L'INTÉRIEUR de ces passes — la compaction les garde (DC.2), rien à gagner
  sur cette table. (7) Après compaction, la liste Relations de Nuzzles reste à 1,4-1,7 s (les
  autres joueurs 0,16-0,62 s) : le coût restant n'est pas celui des passes — à mesurer en B.0.
  (8) Après `COPY FROM DATABASE`, `duckdb_sequences().last_value` affiche la PROCHAINE valeur
  (et non la dernière tirée) dans le fichier neuf ; `nextval` continue juste, aucun code ne lit
  `last_value` ni `currval`.
  (10) (C.9) La conversion append-only garde `recoverOrphanAppendOnly` : après la récupération
  d'un orphelin `__appendonly`, le marqueur étant présent, ni la clé primaire, ni le défaut
  `nextval`, ni les index de `PostSwap` ne sont reposés — même perte latente que celle corrigée
  pour la compaction. Non traité (conversions one-shot déjà appliquées ; hors périmètre).
  (9) Gate C, `go test -tags=integration -p 1` : `internal/service`
  `TestRelationsSegmentation_SoloVsSquad_CrossDB` et `_PlaylistFilter_CrossDB` ROUGES, de façon
  déterministe (rejoués seuls), avec « Binder Error: Referenced column "gamertag" not found » dans
  l'annuaire de `CareerRepo.GetRelations` : leur fixture crée `match_participants` sans la colonne
  `gamertag` que l'annuaire du lot A (`f576df10e`, `5ab3c13c7`) lit. PRÉEXISTANT : rouge à
  l'identique sur l'arbre exporté de `34edf29af` (avant tout code de l'étape C) ; le diff de C ne
  touche ni `internal/service`, ni `internal/platform`, ni `internal/analysis`. Le gate du lot A ne
  jouait l'intégration que sur `./internal/platform/duckdb/...`. Non traité (hors périmètre de C :
  fixture d'un test du lot A).
- (B, 2026-09-27) (11) `internal/archlint` `TestAucunSuffixeDePlateformeAccidentel` est ROUGE sur
  `fb1e65d67` : `cmd/levelup/cmd_compact_passes_remplacement_windows.go` (C.10) porte un suffixe
  de plateforme que le garde-rail n'autorise pas (le renommer, ou l'inscrire dans
  `goosSuffixAllowed` avec sa justification). Fichier de la compaction : non touché (brief).
  (12) L'annuaire en portée base (lot A) lie ses listes en `IN (?, …)` : pour les Relations de
  Nuzzles, 14 900, 10 163 et 17 741 paramètres (alias/participants, matchs des restants,
  kill-feed), 0,8-0,9 s des 1,1-1,3 s de la lecture — la même cause que celle mesurée en B.1 (la
  liaison de milliers de paramètres, pas le filtre) ; `clauseListeMatchs` s'y appliquerait. Non
  traité (DB.5 : ce ne sont pas des fenêtres sur tout l'historique). Idem, à moindre échelle, pour
  l'annuaire de Q26 (7 210 et 7 192 paramètres, ~0,25 s chez Nuzzles). (13) Les autres lectures
  tactiques (`KillEvents`, `MortsAvecContexte`, `KillPositions`, l'univers) lient la liste blanche
  en `IN` jusqu'à trois fois (et `clausePerimetre` sur `mr.match_id`) : à liste longue (onglet
  Tactique sans filtre, 7 190 matchs), des dizaines de milliers de paramètres. Non mesuré, non
  traité. (14) Accueil : la page entière (1,6 s pour 310 Ko en septembre) ne se re-mesure pas sans
  serveur ; ses lectures de la base partagée mesurées sur la copie restent sous 0,22 s (arme
  favorite 0,10-0,19 s, historique canonique 0,04-0,21 s) : si la page reste lente, la cause est
  ailleurs (taille de la réponse, lectures joueur ou sociales, concurrence). Synthèse
  `weapon_records` : 0,05-0,68 s, fenêtres déjà bornées ; le coût suit le volume des matchs du
  joueur (Nuzzles, JGtm). À décider avec l'utilisateur (un cache est exclu par décision). (15)
  Halo 5 : la liste des matchs du joueur exclut la Campagne, les jambes kill-feed de Q26 / Q27 /
  Q28 ne l'excluaient pas — 118 morts entre deux joueurs dans 63 matchs de Campagne de la copie H5
  sortent des rivaux et des frags échangés (écart voulu par DB.2, figé par
  `TestLecturesHistorique_CampagneHorsDeLaListe`) ; non mesuré sur les joueurs H5 un par un.
  (16) La découverte (9) ne se reproduit plus : `go test -tags=integration -run
  TestRelationsSegmentation ./internal/service/` est VERT sur l'arbre exporté de `fb1e65d67` comme
  après B (les journaux montrent encore une erreur de liaison `publishable` dans l'enrichissement
  des assistances Q28c, journalisée en WARN et sans effet sur le verdict). (17) Sur ce poste
  partagé, les chronos varient du simple au double d'un tour à l'autre (charge) : seuls des tours
  ALTERNÉS avant / après sont comparables ; un golangci-lint concurrent tenait le verrou du poste.

## 7. Journal (superviseur)

- 2026-09-26 : mesure des passes supersédées sur copie (tableau §0) ; lot C passé d'« abandonné »
  à « à faire » et lot B restreint au bornage (sans cache) sur décision de l'utilisateur ; plan écrit.
