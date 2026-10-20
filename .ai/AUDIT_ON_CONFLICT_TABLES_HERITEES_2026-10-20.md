# Audit `ON CONFLICT` sur tables héritées sans clé primaire — 2026-10-20

Étape 7.3 du plan de release v8. Audit en lecture seule : aucun code n'a été modifié.

## Cadrage

**Périmètre** : tous les `ON CONFLICT` exécutables de `apps/go-api` (hors commentaires et hors `_test.go`) : 22 instructions. Périmètre voisin, relevé à part car il casse de la même façon (voir « Famille voisine ») : les 37 `INSERT OR IGNORE` / `INSERT OR REPLACE` exécutables.

**Axe principal** : A2 (garde-rails et invariants ART), sous-axe « la clé qui porte le conflit existe-t-elle sur une base héritée ? ». Axe secondaire : ART (`DO UPDATE` sur table critique).

**Axes NON passés** : A1, A3 à A10 (hors sujet ; une remarque de doc inversée est notée en « Constats écartés »).

**Doctrine de référence** (extraits) :
- CLAUDE.md, règle critique n°6 : « `CREATE TABLE IF NOT EXISTS` n'ajoute jamais une PK à une table existante → `ON CONFLICT` échoue. Player DBs legacy sans contraintes : pattern `SELECT-then-UPDATE-or-INSERT`. »
- CLAUDE.md, règles ART 1 à 3 : pas d'UPSERT / `ON CONFLICT DO UPDATE` concurrent sur les tables critiques ; garde-rail `internal/sync/no_art_patterns_test.go`.
- Précédent : `3d83ca75e` / `6d71a8acb` (14/09/2026), `medal_definitions` sans PK, seeds de médailles cassés.

**Dette assumée, non remontée** : pattern `SELECT-then-UPDATE-or-INSERT` sur les player DB legacy ; seeds de migration à usage unique documentés.

### Méthode et limites de la preuve

1. Inventaire par `grep` (voir « Reproduction »), contrôle multi-lignes (`ON\s*\n\s*CONFLICT` : 0 résultat), contrôle des fichiers non-Go (aucun `.sql` embarqué).
2. Comportement vérifié par une sonde Go jetable (supprimée, non commitée) : sur une table sans PK, `ON CONFLICT (id) DO NOTHING`, `ON CONFLICT (id) DO UPDATE`, `INSERT OR IGNORE` et `INSERT OR REPLACE` échouent TOUS avec `Binder Error` (« conflict target ... not referenced by a UNIQUE/PRIMARY KEY CONSTRAINT » ou « There are no UNIQUE/PRIMARY KEY constraints that refer to this table »). L'échec est bruyant, jamais silencieux, mais la ligne n'est pas écrite.
3. Contraintes lues sur les vraies bases avec `cmd/diag_q` (ouverture `access_mode=read_only`), sur COPIES : le serveur local tient en écriture, avec verrou Windows exclusif, `metadata`, `shared_matches_v2` et `shared_social` de `halo_infinite`, `metadata` de `halo_5` et les `stats.duckdb` de 5 joueurs `halo_infinite`. Faute de pouvoir copier ces fichiers vivants, ont été lus à leur place les sauvegardes du `data/backups/` : `metadata` du 08/10, `shared_matches_v2` du 16/10, 5 player DB du 16/10.
4. Bases lues (22) : `halo_infinite` shared (sauvegarde 16/10), metadata (sauvegarde 08/10), pve, 10 player ; `halo_5` shared, social, 6 player ; `data/global/xbox_aliases.duckdb`.
5. **NON lues** : `shared_social` de `halo_infinite` (verrouillée, aucune sauvegarde), `metadata` de `halo_5` (verrouillée), état du 20/10 des bases sauvegardées, et surtout **toute base de production** (VPS). Tout verdict SÛR vaut pour les bases locales et pour le DDL ; il ne vaut pas pour la prod. Le précédent `medal_definitions` le montre : les bases locales portent la PK (`PRIMARY KEY` présent dès la sauvegarde du 01/09), alors que la base du 14/09 ne l'avait pas. Les bases locales ne reproduisent donc pas toutes les bases du parc.
6. Barème : **SÛR** = le DDL de création pose la PK, la table n'a pas d'ancêtre Python sans PK, la PK est observée sur au moins une base réelle de la même famille, aucun chemin connu de perte de PK ne la touche. **RISQUE** = PK absente sur une base réelle, ou perte de PK possible par un chemin identifié. **INCONNU** = non tranchable.

## Synthèse

| Famille | SÛR | RISQUE | INCONNU |
|---|---|---|---|
| `ON CONFLICT` (22 instructions) | 19 | 3 | 0 |
| `INSERT OR IGNORE / REPLACE` (37 instructions, voisin) | 37 | 0 (mais tous sous C1) | 0 |

Constats retenus : 1 P1 (systémique), 2 P2, 1 P2 à trancher (ART). Constats écartés : 5.

Réserve sur les 19 SÛR : 4 reposent sur le DDL et une base de la même famille faute de lecture directe (`user_prestige` et `prestige_events` de `halo_infinite` social, `milestone_catalog` de `halo_5` metadata, tables `h5-metadata-fetch`).

## Constats retenus

### [P1] C1 — La restauration depuis Parquet recrée toutes les tables sans clé primaire, et rien ne les rétablit
- Où : `apps/go-api/pkg/duckdbbackup/importer.go:157` (`CREATE OR REPLACE TABLE %q AS SELECT * FROM read_parquet(...)`), appelé par `cmd/restore/main.go:107` et `cmd/restore_one_player/main.go`.
- Règle enfreinte : CLAUDE.md règle critique n°6 (une table sans PK fait échouer tout `ON CONFLICT`) ; aucune garantie de contrainte après import, aucun test (`grep -n "PRIMARY KEY\|duckdb_constraints" apps/go-api/pkg/duckdbbackup/*_test.go` : vide).
- Preuve sur pièce : la player DB `Chocoboflor` de `halo_infinite` (restaurée par `restore_one_player`, créé selon son en-tête après l'incident du 27/05) a 23 tables sans PK ni UNIQUE, dont `schema_migrations` elle-même (son DDL, `migration/registry.go:150`, pose pourtant `name VARCHAR PRIMARY KEY`) ainsi que `arc`, `challenge`, `sessions`, `milestone_earned`. Les autres player DB n'en ont que 4 à 6 (`career_progression`, `mv_*`, `*_snapshots`, sans PK par conception). La PK de `schema_migrations` ne peut pas revenir : `CREATE TABLE IF NOT EXISTS` ne l'ajoute jamais, et tous les steps sont déjà inscrits donc ne rejouent pas.
- Conséquence concrète : sur une base partagée restaurée par `cmd/restore` (reprise sur sinistre), le chemin chaud de la synchronisation échoue à chaque match, dans la transaction de persistance, avec `Binder Error` : `persist/shared_persister.go:264` (`medals_earned`), `:486` (`xuid_aliases`), `:533` (`match_commendations`), `persist/events_completion_persister.go:241` et `sync/writes.go:251` (`highlight_events`). Bruyant, mais bloque toute nouvelle écriture de match jusqu'à réparation manuelle. Les migrations qui font `ON CONFLICT (key)` sur `sync_meta` (`steps_player_repairs.go:232`, `steps_shared_rebuild_match_participants.go:340`) sont exposées de la même manière si la table y perd sa PK. Inféré (le mécanisme est prouvé par la sonde et par `Chocoboflor`) ; non rejoué de bout en bout sur une base partagée restaurée.
- Reproduction : `diag_q dbcopy/bkp_Chocoboflor/stats.duckdb "SELECT table_name FROM duckdb_tables() t WHERE NOT EXISTS (SELECT 1 FROM duckdb_constraints() c WHERE c.table_name=t.table_name AND c.constraint_type IN ('PRIMARY KEY','UNIQUE'))"` (copie de `data/backups/avant_recuisson_98_2026-10-16/players/Chocoboflor/`) ; `grep -n "CREATE OR REPLACE TABLE" apps/go-api/pkg/duckdbbackup/importer.go`.
- Vérification adverse : non passée en sous-agent (périmètre étroit, preuves lues directement). Tentative de réfutation faite à la main : la PK de `sync_meta` est présente dans la même base alors que l'import l'aurait supprimée. Cela ne réfute pas le constat (la table a pu être recréée depuis son DDL, `purge_sync_meta_legacy_auth_keys_v1` du 09/10 l'a reconstruite), mais montre que la cause de la perte de PK de `Chocoboflor` reste une hypothèse très probable, pas une preuve.
- Piste (non appliquée) : après `importKeyDB`, rejouer le DDL canonique des tables à PK (ou refuser l'import tant que `duckdb_constraints()` ne retrouve pas les PK du schéma attendu) ; test d'import qui vérifie les PK d'un échantillon de tables. Décision : à trancher (touche à la procédure de reprise sur sinistre).

### [P2] C2 — `challenge_template` n'a pas de PK sur la base metadata locale ; le seed Prestige fait `ON CONFLICT (id)`
- Où : `apps/go-api/internal/games/halo_infinite/migrations/prestige.go:130` (`ON CONFLICT (id) DO UPDATE`), table `challenge_template`, base `metadata.duckdb` de `halo_infinite`.
- Règle enfreinte : règle critique n°6 (le DDL `steps.go:118` pose bien `id VARCHAR PRIMARY KEY`, mais la base locale a une table sans PK ni NOT NULL, signature d'un `CREATE TABLE AS SELECT`).
- Preuve : sauvegarde `data/backups/avant_rattrapage_2026-10-08/metadata.duckdb` : `CREATE TABLE challenge_template(id VARCHAR, title_slug VARCHAR, ...)` sans contrainte, 27 lignes, 27 ids distincts (une PK serait donc posable sans nettoyage préalable). Les deux autres tables de ce seed (`preset_arc`, `preset_arc_step`) ont leur PK.
- Conséquence concrète : aujourd'hui aucune, car `seed_prestige_catalog_v1` est déjà inscrit `backfill_done = true` (appliqué le 01/05 sans erreur : la PK existait donc alors). À la prochaine montée de version du seed (même geste que `seed_milestone_catalog_v2`) ou sur une table à rejouer, `seedTemplates` échoue en `Binder Error`. Comme c'est un `ApplyBackfill`, l'échec n'arrête pas les migrations suivantes (`registry.go:403`, simple WARN `migration: backfill failed`) mais il se répète à chaque démarrage : les modèles de défis du catalogue ne sont pas rafraîchis, sans autre signal que ce WARN. Les écritures courantes de la table (`prestige_metadata_repo.go:170`) sont en `SELECT-then-UPDATE-or-INSERT` et ne sont pas concernées.
- Origine de la perte de PK : non établie. Deux pistes : le chemin de repli de `cmd/repair-metadata/main.go` (`repairTable`, étape 3, `ALTER TABLE tmp RENAME TO table` restitue la copie CTAS sans PK si la recréation échoue) ; ou une restauration Parquet (C1).
- Reproduction : `diag_q <copie de la sauvegarde> "SELECT sql FROM duckdb_tables() WHERE table_name='challenge_template'"`.
- Vérification adverse : voir C1 (non passée).
- Piste : écrire le seed via `SELECT-then-write` comme le repo (cohérent avec `UpsertNoConflict`), ou un step de réparation qui reconstruit la table avec sa PK (modèle : `repair_match_citations_primary_key`). Décision : à trancher. Contrôler d'abord l'état du fichier vivant (verrouillé pendant l'audit).

### [P2] C3 — `cmd/prestige-seed` : `arc` et `challenge` sans PK sur la player DB `Chocoboflor`
- Où : `apps/go-api/cmd/prestige-seed/main.go:161` (`arc`, `ON CONFLICT (id) DO NOTHING`) et `:175` (`challenge`, même forme), base `players/Chocoboflor/stats.duckdb` de `halo_infinite`.
- Règle enfreinte : règle critique n°6.
- Preuve : sauvegarde du 16/10, `arc` et `challenge` de `Chocoboflor` : aucune contrainte (0 ligne chacune) ; toutes les autres player DB les ont avec PK (`id`). Cause probable : C1.
- Conséquence concrète : `go run ./cmd/prestige-seed/main.go --player Chocoboflor` (l'exemple même de son en-tête) échoue en `Binder Error` à l'insertion de `arc`, après avoir déjà écrit `user_prestige` et `prestige_events` dans `shared_social` (pas de transaction commune) : jeu de données de test à moitié posé. Outil de développement, `//go:build ignore`, serveur arrêté : impact limité à ce poste.
- Reproduction : `diag_q <copie> "SELECT table_name, constraint_type FROM duckdb_constraints() WHERE table_name IN ('arc','challenge')"` (résultat vide sur `Chocoboflor`).
- Piste : `WHERE NOT EXISTS` au lieu de `ON CONFLICT` (pattern du dépôt), ou retirer l'outil s'il n'est plus utilisé. Décision : à trancher.

### [P2, à trancher] C4 — `ON CONFLICT DO UPDATE` sur `playlists_catalog` dans une migration, hors portée du garde-rail ART
- Où : `apps/go-api/internal/games/halo_infinite/migrations/ranked_playlists.go:41` (`playlists_catalog`), `:71` (`asset_translations`, non protégée).
- Règle : `no_art_patterns_test.go` range `playlists_catalog` parmi `tablesProtegees` (commentaire : « un `ON CONFLICT DO UPDATE` sur ces tables FATAL-invalide le handle metadata »), mais `dansLePerimetreART` exclut tout chemin contenant `/migrations/`, `/cmd/`, `/scripts/` (`internal/sync/no_art_patterns_test.go:703`). L'instruction passe donc sous le garde-rail par construction.
- Conséquence : l'exécution est unique (step inscrit dans `schema_migrations`, base locale lue : appliqué) et mono-process, et la PK n'est pas modifiée par le `DO UPDATE` ; le risque ne vaut donc qu'au premier démarrage d'une base neuve. À trancher : accepter (usage unique documenté) ou aligner sur `SELECT-then-write`. La même remarque vaut, sans table protégée, pour les autres `DO UPDATE` de migration (`milestones.go` x2, `prestige.go` x3, `steps_player_repairs.go:232`, `steps_shared_rebuild_match_participants.go:340`).
- Reproduction : `grep -n "ON CONFLICT" apps/go-api/internal/games/halo_infinite/migrations/ranked_playlists.go` ; `sed -n 703,713p apps/go-api/internal/sync/no_art_patterns_test.go`.
- Décision : à trancher (touche à la doctrine du garde-rail).

## Inventaire des 22 `ON CONFLICT`

| # | Fichier:ligne | Table | Base | Conflit | Action | Verdict | Preuve |
|---|---|---|---|---|---|---|---|
| 1 | `cmd/levelup/cmd_consolidate_aliases.go:75` | `xuid_aliases` | shared (hi) | `xuid` | DO NOTHING | SÛR | PK `xuid` (shared hi 16/10) ; DDL Python et Go avec PK |
| 2 | `cmd/migrate-xuid-aliases-global/main.go:227` | `xuid_aliases` | global | `xuid` | DO UPDATE | SÛR | crée sa table avec PK ; PK observée sur `xbox_aliases.duckdb` |
| 3 | `cmd/world-aliases-persist/main.go:161` | `xuid_aliases` | shared (hi) | `xuid` | DO NOTHING | SÛR | idem 1 |
| 4 | `cmd/prestige-seed/main.go:82` | `user_prestige` | shared_social | `(user_id, title_slug)` | DO UPDATE | SÛR (réserve) | PK observée sur social h5 ; social hi non lisible. Écrit la table figée par l'append-only, voir « Constats écartés » |
| 5 | `cmd/prestige-seed/main.go:96` | `prestige_events` | shared_social | `id` | DO NOTHING | SÛR (réserve) | idem 4 |
| 6 | `cmd/prestige-seed/main.go:161` | `arc` | player | `id` | DO NOTHING | **RISQUE** (C3) | sans PK sur `Chocoboflor` |
| 7 | `cmd/prestige-seed/main.go:175` | `challenge` | player | `id` | DO NOTHING | **RISQUE** (C3) | idem |
| 8 | `cmd/seed-ranked-playlists/main.go:91` | `playlists_catalog` | metadata | `(title_slug, playlist_asset_id)` | DO NOTHING | SÛR | PK observée (metadata hi 08/10) ; DDL Go `steps.go:281` |
| 9 | `games/halo_5/migrations/milestones.go:140` | `milestone_catalog` | metadata (h5) | `id` | DO UPDATE | SÛR (réserve) | DDL h5 avec PK, table née Go ; h5 metadata non lisible ; PK observée sur la même table côté hi |
| 10 | `games/halo_infinite/migrations/milestones.go:98` | `milestone_catalog` | metadata | `id` | DO UPDATE | SÛR | PK observée (hi 08/10) |
| 11 | `games/halo_infinite/migrations/prestige.go:130` | `challenge_template` | metadata | `id` | DO UPDATE | **RISQUE** (C2) | sans PK (hi 08/10), latent |
| 12 | `.../prestige.go:179` | `preset_arc` | metadata | `id` | DO UPDATE | SÛR | PK observée |
| 13 | `.../prestige.go:197` | `preset_arc_step` | metadata | `(preset_arc_id, position)` | DO UPDATE | SÛR | PK observée |
| 14 | `.../ranked_playlists.go:41` | `playlists_catalog` | metadata | `(title_slug, playlist_asset_id)` | DO UPDATE | SÛR (voir C4 pour l'axe ART) | PK observée |
| 15 | `.../ranked_playlists.go:71` | `asset_translations` | metadata | `(asset_id, asset_type, lang)` | DO UPDATE | SÛR | PK observée ; table née Go (30/03) |
| 16 | `.../steps_player_repairs.go:232` | `sync_meta` | player | `key` | DO UPDATE | SÛR | PK observée sur les 16 player DB lues ; DDL Python (`engine.py`) et Go avec PK. Exposé à C1 |
| 17 | `internal/migration/registry.go:194` | `title_schema_version` | toutes | `(title_slug, target)` | DO UPDATE | SÛR | table née Go, PK posée à la création, observée sur toutes les bases lues qui en portent une ; échec non fatal (WARN) |
| 18 | `internal/migration/steps_shared_rebuild_match_participants.go:340` | `sync_meta` | shared | `key` | DO UPDATE | SÛR | PK observée (hi 16/10, h5). Exposé à C1 |
| 19 | `internal/ops/seed_demo_media.go:373` | `media_files` | social démo | `id` | DO NOTHING | SÛR | base démo générée à neuf par les migrations (`seed_demo_synthetic.go` en-tête) ; PK observée sur social h5 |
| 20 | `internal/ops/seed_demo_synthetic_meta.go:104` | `medal_definitions` | metadata démo | `medal_name_id` | DO NOTHING | SÛR | idem 19 ; piège hérité du 14/09 inopérant sur une base neuve |
| 21 | `scripts/import_bp_items/main.go:111` | `battlepass_item_definitions` | metadata | `(inventory_item_path, content_hash)` | DO UPDATE | SÛR | PK observée |
| 22 | `scripts/import_bp_items/main.go:147` | `battlepass_item_translations` | metadata | `(inventory_item_path, content_hash, lang)` | DO UPDATE | SÛR | PK observée |

## Famille voisine : `INSERT OR IGNORE` / `INSERT OR REPLACE` (37 instructions)

Même mécanisme, même erreur, prouvé par la sonde. Non demandés dans le périmètre, relevés car un correctif de classe doit les couvrir.

| Groupe | Fichiers (extrait) | Tables | Verdict |
|---|---|---|---|
| Chemin chaud de la synchronisation (shared) | `persist/shared_persister.go:264,486,533`, `persist/events_completion_persister.go:241`, `sync/writes.go:251` | `medals_earned`, `xuid_aliases`, `match_commendations`, `highlight_events` | SÛR (PK observée hi et h5) ; exposé à C1 |
| Référentiels metadata hi | `persist/metadata_persister.go:76`, `migrations/mode_playlist_fr.go:121,208`, `migrations/steps.go:206`, `games/weapons/{labels,name_labels,registry}.go` (6), `ops/seed.go:218`, `cmd/{refresh-career-ranks,seed-rank-translations,seed-assists-model}` | `mode_name_tr`, `asset_translations`, `csr_placement_thresholds`, `weapon_*`, `weapons`, `career_ranks`, `career_rank_translations`, `assists_model_coefs` | SÛR (PK observée, metadata hi 08/10) |
| Outils `halo_5` | `cmd/h5-metadata-fetch/main.go` (8), `cmd/h5-roster-{refetch,topup}` (2) | tables metadata h5, `xuid_aliases` | SÛR sur DDL (réserve : metadata h5 non lisible) |
| Migration ponctuelle | `cmd/migrate-to-shared-social` (3) | `media_files`, `media_match_associations`, `media_likes` | SÛR |
| Génération démo | `ops/seed_demo_synthetic_*.go` (5) | `career_ranks`, `sessions`, `match_citations`, `xuid_aliases`, `medals_earned` | SÛR (base neuve) |

## Constats écartés

| Constat | Axe | Motif d'écart |
|---|---|---|
| `title_schema_version` : `DO UPDATE` à chaque démarrage sur chaque base (`registry.go:194`) | ART | Table d'une ligne, non critique, écriture mono-process au démarrage et non fatale ; le commentaire de `registry.go:168` le documente. Aucune règle écrite enfreinte. |
| `prestige-seed` écrit `user_prestige` en `DO UPDATE` alors que la table a été figée par l'append-only (`shared_social_user_prestige_append_only_v1`) | ART / A1 | Outil jetable `//go:build ignore`, hors `go build` ; hors périmètre PK. À noter pour un éventuel nettoyage de l'outil. |
| `sync_meta` : deux migrations en `ON CONFLICT` alors que `steps_player.go:288` utilise `SELECT-then-UPDATE-or-INSERT` pour la même table | A5 | PK observée sur toutes les bases lues, DDL Python et Go avec PK ; incohérence de style sans conséquence mesurée. |
| Le journal `legacy_player_records_upsert_used` (`shared_social_persister_batch.go:301`) annonce « player_records ON CONFLICT DO UPDATE » alors que le repli (`:365`) est un `SELECT-then-UPDATE-or-INSERT` | A7 (doc inversée) | Hors axe de cet audit ; signalé tel quel, aucune conséquence sur les données. |
| Tables sans PK de `halo_5` shared (`killer_victim_pairs`, `weapon_accuracy`, `weapon_kills*`) | A2 | Aucun `ON CONFLICT` / `OR IGNORE` ne les vise (`grep`), écritures INSERT pur. |

## Axes sans constat
Aucun axe n'a été passé sans constat : l'axe unique passé a produit les constats ci-dessus. Aucun `ON CONFLICT` n'est sur le chemin chaud d'une table append-only protégée (règles ART 1 et 2 respectées sur ce point).

## Statut des correctifs (2026-10-21, branche `fix/v802-cles-primaires`, cible v8.0.2)

| Constat | Statut | Commits | Preuve |
|---|---|---|---|
| C1 restauration Parquet | **Corrigé** | `64d3dbe00` (+ `7bb9d4831`, `d4a8a1a39` : ratchets) | `pkg/duckdbbackup/importer_test.go` (export puis import : PK/UNIQUE présentes dans `duckdb_constraints()`, `ON CONFLICT` et `INSERT OR IGNORE` réussissent), `internal/ops/restore_schema_cgo_test.go` (base joueur migrée, exportée, restaurée : clés, vue `_latest`, cycle de migrations suivant) |
| C2 `challenge_template` + séquelles de C1 | **Corrigé** (sauf `streak` de Chocoboflor, voir ci-dessous) | `ba46b8cdd` | `internal/migration/declared_keys_repair_test.go`, `internal/games/halo_infinite/migrations/declared_keys_repair_test.go` (dont le seed Prestige en `ON CONFLICT` sur la table réparée) |
| C3 `cmd/prestige-seed` | **Corrigé** (outil supprimé) | `f30a7073d` | ratchets `no_bare_duckdb_rw_open` et `no_attach_on_social` verts sans l'entrée |
| C4 garde-rail ART et migrations | **Corrigé** | `ce529939e` | `internal/sync/no_art_upsert_migrations_test.go` : rouge sur `ranked_playlists.go` avant correctif, vert sur tout le module après ; témoin `TestUpsertProtegeDetection_Sanity` |

**C1** : chaque base restaurée est d'abord créée par le jeu de migrations de son titre et de sa cible (`ops.RestoreSchema`), puis remplie table par table par colonnes nommées (`pkg/duckdbkeys`). Une table hors schéma canonique est recréée depuis le DDL que l'export écrit désormais (`_ddl_<horodatage>.json`) ; repli `CREATE TABLE AS` journalisé et compté seulement sans DDL (instantanés antérieurs). Mesure (export puis restauration de copies) : Chocoboflor 60 272 lignes identiques, tables sans clé 23 → 9 ; JGtm 105 713 lignes identiques ; metadata 44 153 lignes identiques (11 tables recréées depuis le DDL, `challenge_template` retrouve sa clé) ; shared 20 461 480 lignes identiques en 46 s, clés intactes.

**C2** : passe générique en fin de cycle de migrations (`internal/migration/declared_keys_repair.go`). Mesure sur copies du `data/backups/` : metadata (08/10) 1 table réparée (`challenge_template`, 27 lignes, 29 ms) ; Chocoboflor (16/10) 13 tables réparées (`arc`, `baseline_state`, `challenge`, `coach_proposal`, `improvement_campaign`, `milestone_earned`, `moment_card`, `player_achievements`, `player_assists_model`, `player_privacy_state`, `prestige_telemetry`, `schema_migrations`, `sessions`), 0 doublon, lignes identiques, cycle 0,43 s référence comprise ; JGtm, Madina97294, Nuzzles, XxDaemonGamerxX, shared Halo Infinite et shared Halo 5 : rien à réparer.

**Reste à décider** : `streak` de Chocoboflor (table append-only, vue `streak_latest`, 1 ligne) n'a pas sa clé technique `id` ; la passe refuse par principe les tables append-only (WARN `migration_key_repair_skipped_total`). Écritures non bloquées (INSERT pur). Les tables `mv_global_stats`, `mv_mode_category_stats`, `mv_session_stats` et `spartan_identity` de Chocoboflor, héritées hors migrations et sans écrivain Go, restent sans clé. La vérification de production (tables sans clé de la shared et de la metadata de prod) reste à faire : la passe C2 les réparera au premier démarrage de la v8.0.2, et journalisera ce qu'elle refuse.

## Suite
- C1 : à escalader (procédure de reprise sur sinistre) ; vérifier d'abord, avant toute restauration réelle, que `cmd/restore` re-pose les clés. Sans dépendre de C1, une vérification de production reste à faire : lister les tables sans PK de la base partagée et de la metadata de prod (sauvegarde restic ou arrêt du serveur ; les fichiers vivants sont verrouillés), avec la requête de « Reproduction ».
- C2 : à planifier avant la prochaine montée de version du seed Prestige ; contrôle préalable de l'état du fichier vivant `metadata.duckdb` (verrouillé pendant l'audit).
- C3 : à classer (outil jetable) ou à retirer.
- C4 : décision de doctrine sur la portée du garde-rail pour les migrations.
- Toute correction passe par `plan-review`, `plan-execution`, `adversarial-review` ; cet audit ne corrige rien.

## Reproduction

```
# Outil de lecture seule (CGO, gcc msys64)
CGO_ENABLED=1 CC=C:/msys64/ucrt64/bin/gcc.exe go -C apps/go-api build -o diag_q.exe ./cmd/diag_q
# Tables sans PK ni UNIQUE d'une base (travailler sur une COPIE si le serveur tourne)
diag_q.exe <copie.duckdb> "SELECT t.table_name FROM duckdb_tables() t WHERE NOT EXISTS (SELECT 1 FROM duckdb_constraints() c WHERE c.table_name=t.table_name AND c.constraint_type IN ('PRIMARY KEY','UNIQUE')) AND NOT EXISTS (SELECT 1 FROM duckdb_indexes() i WHERE i.table_name=t.table_name AND i.is_unique)"
# Inventaire
grep -rn -i "ON CONFLICT" --include=*.go apps/go-api | grep -v _test.go | grep -v -E ':[0-9]+:\s*(//|\*)'
grep -rn -i -E "INSERT OR (IGNORE|REPLACE)" --include=*.go apps/go-api | grep -v _test.go | grep -v -E ':[0-9]+:\s*(//|\*)'
```
