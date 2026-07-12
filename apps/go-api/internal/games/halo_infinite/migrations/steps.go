// Package migrations regroupe les migrations DDL APPARTENANT à Halo Infinite
// (Phase 1.5.1 voie B, ADR 0025). Elles vivaient dans internal/migration/
// (couplées au registre global via init()) ; elles en sortent ici, fournies au
// runner via migration.SetTitleStepsProvider(StepsFor).
//
// Construites avec les helpers exportés du package migration (TableExists,
// AddColumnIfMissing, ExecScript, …). L'ordre d'exécution reste imposé par
// migration.canonicalOrder (order.go) — déplacer un step ici ne le réordonne
// pas.
//
// État transition : Steps() se remplit au fur et à mesure des déplacements
// (b3). Vide = aucun step encore migré (le registre global legacy fournit
// tout, comportement inchangé).
package migrations

import (
	"database/sql"
	"fmt"

	"levelup/go-api/internal/migration"
)

// Steps retourne toutes les migrations title-owned de Halo Infinite, tous
// targets confondus. Se remplit au fur et à mesure des déplacements (b3) ;
// chaque entrée doit être listée dans migration.canonicalOrder (vérifié par
// order_audit_test.go).
func Steps() []migration.Migration {
	steps := []migration.Migration{
		// Déplacés depuis internal/migration/steps_metadata.go (b4 — leaves additifs).
		{
			Name:        "add_waypoint_assets_raw",
			TargetDB:    migration.TargetMetadata,
			Description: "Table waypoint_assets_raw : cache générique de blobs JSON Waypoint",
			ApplySchema: func(db *sql.DB) error {
				return migration.ExecScript(db, `
					CREATE TABLE IF NOT EXISTS waypoint_assets_raw (
						title_id     VARCHAR NOT NULL,
						asset_id     VARCHAR NOT NULL,
						asset_type   VARCHAR NOT NULL DEFAULT '',
						version_id   VARCHAR NOT NULL DEFAULT '',
						name         VARCHAR NOT NULL DEFAULT '',
						description  VARCHAR NOT NULL DEFAULT '',
						raw_json     VARCHAR NOT NULL DEFAULT '',
						fetched_at   TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT now(),
						content_hash VARCHAR NOT NULL DEFAULT '',
						PRIMARY KEY (title_id, asset_id, version_id)
					);
				`)
			},
		},
		// Décision MT-16 (EXT-1.5, PMT-13) — stratégie title_id NUANCÉE :
		// map_images_registry ET waypoint_assets_raw PORTENT `title_id` (PK) car ce
		// sont des caches d'assets GÉNÉRIQUES (blobs Waypoint indexés par title_id +
		// asset_id) — défense en profondeur contre une collision d'asset_id entre
		// titres dans une même metadata.duckdb. À l'INVERSE, les référentiels
		// canoniques (weapon_labels, mode_name_tr, citation_mappings,
		// career_rank_translations) RESTENT SANS title_id : isolés PAR CHEMIN
		// (data/titles/<slug>/warehouse/metadata.duckdb, cf. ADR 0008). La reco
		// EXT-1.5 « path suffit » tient pour le canonique ; title_id n'est ajouté
		// que là où l'asset_id seul n'est pas globalement unique.
		{
			Name:        "add_map_images_registry",
			TargetDB:    migration.TargetMetadata,
			Description: "Table map_images_registry : cache-aside des images de maps avec local_path optionnel",
			ApplySchema: func(db *sql.DB) error {
				return migration.ExecScript(db, `
					CREATE TABLE IF NOT EXISTS map_images_registry (
						title_id     VARCHAR NOT NULL,
						map_id       VARCHAR NOT NULL,
						image_url    VARCHAR NOT NULL DEFAULT '',
						local_path   VARCHAR NOT NULL DEFAULT '',
						fetched_at   TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT now(),
						content_hash VARCHAR NOT NULL DEFAULT '',
						PRIMARY KEY (title_id, map_id)
					);
				`)
			},
		},
		// Déplacés depuis internal/migration (b6/b7 — named-func, cf. weapon_labels.go
		// + mode_playlist_fr.go).
		{
			Name:        "add_weapon_labels",
			TargetDB:    migration.TargetMetadata,
			Description: "Table weapon_labels (weapon_id UBIGINT, name_en, name_fr)",
			ApplySchema: applyWeaponLabels,
		},
		{
			Name:        "add_weapon_registry",
			TargetDB:    migration.TargetMetadata,
			Description: "Registre d'armes canonique (weapons/weapon_ids/weapon_families) : passage principal de la résolution d'arme, seed §6 vérifiée + filmshell ids Infinite",
			ApplySchema: applyWeaponRegistry,
		},
		{
			Name:        "add_mode_name_tr",
			TargetDB:    migration.TargetMetadata,
			Description: "Table mode_name_tr : traductions des modes de jeu (FR/EN), portage depuis metadata-prebuilt",
			ApplySchema: applyModeNameTr,
		},
		{
			Name:        "seed_playlist_fr_translations",
			TargetDB:    migration.TargetMetadata,
			Description: "asset_translations : seed FR canoniques pour playlists Halo Infinite dont l'API a renvoyé l'EN raw en lang fr-FR (cf. thought_log 2026-05-09)",
			ApplySchema: applyPlaylistFRSeeds,
		},
		// Famille prestige (schéma + 2 ALTER challenge_template) → migrée (b10).
		// Le seed TOML seed_prestige_catalog_v1 est enregistré dynamiquement par boot
		// via RegisterPrestigeSeedMigration (cf. prestige.go).
		{
			Name:        "create_prestige_metadata_schema",
			TargetDB:    migration.TargetMetadata,
			Description: "Tables Prestige metadata (challenge_template, preset_arc, preset_arc_step)",
			ApplySchema: func(db *sql.DB) error {
				return migration.ExecScript(db, `
					CREATE TABLE IF NOT EXISTS challenge_template (
						id                  VARCHAR PRIMARY KEY,
						title_slug          VARCHAR NOT NULL,
						metric              VARCHAR NOT NULL,
						window_type         VARCHAR NOT NULL,
						window_value        VARCHAR,
						cadence             VARCHAR NOT NULL,
						eval_type           VARCHAR NOT NULL,
						mode_filter         VARCHAR NOT NULL DEFAULT 'universal',
						label_en            VARCHAR NOT NULL,
						label_fr            VARCHAR NOT NULL,
						description_en      VARCHAR,
						description_fr      VARCHAR,
						normal_target       DOUBLE NOT NULL,
						heroic_target       DOUBLE NOT NULL,
						legendary_target    DOUBLE NOT NULL,
						mythic_target       DOUBLE NOT NULL,
						schema_version      INTEGER NOT NULL DEFAULT 1,
						updated_at          TIMESTAMP DEFAULT CURRENT_TIMESTAMP
					);
					-- PAS d'index secondaire : title_slug/cadence/metric sont MUTÉS par
					-- PrestigeChallengeTemplateRepo.Replace (SELECT-then-write) → surface ART.
					-- Catalogue minuscule (TOML). Drop DBs existantes : drop_metadata_art_surface_indexes_v3.

					CREATE TABLE IF NOT EXISTS preset_arc (
						id              VARCHAR PRIMARY KEY,
						title_slug      VARCHAR NOT NULL,
						title_en        VARCHAR NOT NULL,
						title_fr        VARCHAR NOT NULL,
						description_en  VARCHAR,
						description_fr  VARCHAR,
						schema_version  INTEGER NOT NULL DEFAULT 1,
						updated_at      TIMESTAMP DEFAULT CURRENT_TIMESTAMP
					);
					-- PAS d'index sur title_slug : muté par PrestigePresetArcRepo.Replace
					-- (SELECT-then-write) → surface ART. Drop : drop_metadata_art_surface_indexes_v3.

					CREATE TABLE IF NOT EXISTS preset_arc_step (
						preset_arc_id   VARCHAR NOT NULL,
						position        INTEGER NOT NULL,
						template_id     VARCHAR NOT NULL,
						target_tier     VARCHAR NOT NULL,
						PRIMARY KEY (preset_arc_id, position)
					);
				`)
			},
		},
		{
			Name:        "challenge_template_add_source_column",
			TargetDB:    migration.TargetMetadata,
			Description: "Ajoute challenge_template.source pour distinguer catalog vs coach_synthesized (ADR 0028)",
			ApplySchema: func(db *sql.DB) error {
				return migration.ExecScript(db, `
					ALTER TABLE challenge_template
						ADD COLUMN IF NOT EXISTS source VARCHAR DEFAULT 'catalog';
					UPDATE challenge_template SET source = 'catalog' WHERE source IS NULL;
				`)
			},
		},
		{
			Name:        "add_template_tagging_columns",
			TargetDB:    migration.TargetMetadata,
			Description: "Ajoute lusr_components, radar_axes, is_long_term à challenge_template pour le tagging V1 PlayerProfile.",
			ApplySchema: func(db *sql.DB) error {
				if err := migration.AddColumnIfMissing(db, "challenge_template", "lusr_components", "VARCHAR"); err != nil {
					return err
				}
				if err := migration.AddColumnIfMissing(db, "challenge_template", "radar_axes", "VARCHAR"); err != nil {
					return err
				}
				return migration.AddColumnIfMissing(db, "challenge_template", "is_long_term", "BOOLEAN DEFAULT FALSE")
			},
		},
		// Leaves additifs sibling-files → migrés (b9).
		{
			Name:        "add_csr_placement_thresholds",
			TargetDB:    migration.TargetMetadata,
			Description: "Table csr_placement_thresholds (mapping season_id → seuil placement, 10 pré-S3 / 5 depuis S3)",
			ApplySchema: func(db *sql.DB) error {
				return migration.ExecScript(db, `
					CREATE TABLE IF NOT EXISTS csr_placement_thresholds (
						season_id  VARCHAR PRIMARY KEY,
						threshold  INTEGER NOT NULL CHECK (threshold > 0 AND threshold <= 20),
						valid_from DATE,
						notes      VARCHAR
					);

					-- Seed initial : 13 saisons connues (S1-S13). INSERT OR REPLACE idempotent.
					INSERT OR REPLACE INTO csr_placement_thresholds VALUES
						('CsrSeason1',    10, DATE '2021-12-08', 'S1 Launch — seuil historique 10'),
						('CsrSeason2',    10, DATE '2022-05-03', 'S2 Lone Wolves — seuil 10'),
						('CsrSeason2-2', 10, DATE '2022-11-08', 'Winter 22 — seuil 10'),
						('CsrSeason3-1',  5, DATE '2023-03-07', 'S3 Echoes Within — seuil baissé à 5'),
						('CsrSeason4-1',  5, DATE '2023-06-20', 'S4 Infection'),
						('CsrSeason5-1',  5, DATE '2023-10-17', 'S5 Reckoning'),
						('CsrSeason6-1',  5, DATE '2024-01-30', 'S6'),
						('CsrSeason7-1',  5, DATE '2024-04-30', 'S7'),
						('CsrSeason8-1',  5, DATE '2024-07-30', 'S8'),
						('CsrSeason9-1',  5, DATE '2024-11-05', 'S9'),
						('CsrSeason10-1', 5, DATE '2025-02-04', 'S10'),
						('CsrSeason11-1', 5, DATE '2025-05-06', 'S11'),
						('CsrSeason12-1', 5, DATE '2025-08-05', 'S12'),
						('CsrSeason13-1', 5, DATE '2025-11-18', 'S13 Infinite — saison courante (mai 2026)');
				`)
			},
		},
		{
			Name:        "add_assists_model_coefs",
			TargetDB:    migration.TargetMetadata,
			Description: "Table assists_model_coefs : coefs régressions expected_assists par mode",
			ApplySchema: func(db *sql.DB) error {
				return migration.ExecScript(db, `
					CREATE TABLE IF NOT EXISTS assists_model_coefs (
						game_variant_name  VARCHAR PRIMARY KEY,
						slope              DOUBLE  NOT NULL,
						intercept          DOUBLE  NOT NULL,
						r2                 DOUBLE  NOT NULL,
						n_samples          INTEGER NOT NULL,
						computed_at        TIMESTAMP DEFAULT CURRENT_TIMESTAMP
					);
				`)
			},
		},
		// Famille milestones (schéma) → migrée (b11). Le seed TOML
		// seed_milestone_catalog_v1 est enregistré dynamiquement par boot via
		// RegisterMilestonesSeedMigration (cf. milestones.go).
		{
			Name:        "create_milestone_catalog_metadata",
			TargetDB:    migration.TargetMetadata,
			Description: "Table milestone_catalog (référentiel des milestones cross-titres, chargée du TOML)",
			ApplySchema: func(db *sql.DB) error {
				return migration.ExecScript(db, `
					CREATE TABLE IF NOT EXISTS milestone_catalog (
						id          VARCHAR PRIMARY KEY,
						title_slug  VARCHAR NOT NULL,
						metric      VARCHAR NOT NULL,
						threshold   DOUBLE NOT NULL,
						title_en    VARCHAR NOT NULL,
						title_fr    VARCHAR NOT NULL,
						icon        VARCHAR,
						condition   VARCHAR,
						updated_at  TIMESTAMP DEFAULT CURRENT_TIMESTAMP
					);
					-- PAS d'index secondaire : title_slug et metric sont MUTÉS par
					-- MilestoneCatalogRepo.Upsert (SELECT-then-write) → un index ART sur une
					-- colonne mutée FATAL-invalide metadata.duckdb. Table minuscule (TOML) →
					-- scan séquentiel instantané. Drop DBs existantes :
					-- drop_metadata_art_surface_indexes_v2.
				`)
			},
		},
		// Famille catalogue Playlists/Pairs/Maps → migrée (b12). Schéma 8 tables
		// title-aware + drop des index ART-corrupteurs (même table playlists_catalog,
		// atomique) + seed_ranked_playlists_catalog (named-func, ranked_playlists.go).
		{
			Name:        "add_catalog_playlists",
			TargetDB:    migration.TargetMetadata,
			Description: "Catalogue global Playlists/Pairs/Maps (8 tables title-aware) — Phase A plan catalogue",
			ApplySchema: func(db *sql.DB) error {
				return migration.ExecScript(db, `
					-- 1. Référentiel des playlists, par titre
					CREATE TABLE IF NOT EXISTS playlists_catalog (
						title_slug          VARCHAR NOT NULL,
						playlist_asset_id   VARCHAR NOT NULL,
						current_version_id  VARCHAR,
						name_canonical      VARCHAR,
						experience          VARCHAR,
						is_ranked           BOOLEAN DEFAULT FALSE,
						is_active           BOOLEAN DEFAULT TRUE,
						first_seen_at       TIMESTAMP,
						last_seen_at        TIMESTAMP,
						last_fetched_at     TIMESTAMP,
						PRIMARY KEY (title_slug, playlist_asset_id)
					);

					-- 2. Référentiel des maps. NB (fix RC-E 2026-06-01) : aucun index
					-- secondaire sur playlists_catalog ci-dessus. seedPlaylistsCatalog UPDATE
					-- ses colonnes experience/is_active a chaque cycle, et un UPDATE sur une
					-- colonne ART-indexee corrompt l index DuckDB puis FATAL-invalide
					-- metadata.duckdb (cascade shared RO). A 34 lignes un scan est instantane.
					-- Cf. drop_playlists_catalog_secondary_indexes (drop sur DB existantes).
					CREATE TABLE IF NOT EXISTS maps_catalog (
						title_slug          VARCHAR NOT NULL,
						map_asset_id        VARCHAR NOT NULL,
						current_version_id  VARCHAR,
						name_canonical      VARCHAR,
						image_url           VARCHAR,
						last_fetched_at     TIMESTAMP,
						PRIMARY KEY (title_slug, map_asset_id)
					);

					-- 3. Référentiel des game variants (un variant peut être dans N pairs)
					CREATE TABLE IF NOT EXISTS game_variants_catalog (
						title_slug              VARCHAR NOT NULL,
						game_variant_asset_id   VARCHAR NOT NULL,
						current_version_id      VARCHAR,
						name_canonical          VARCHAR,
						mode_canonical          VARCHAR,
						game_variant_category   INTEGER,
						last_fetched_at         TIMESTAMP,
						PRIMARY KEY (title_slug, game_variant_asset_id)
					);
					-- AUCUN index secondaire : upsertGameVariant UPDATE mode_canonical →
					-- un index ART sur une colonne mutée corrompt metadata.duckdb (FATAL
					-- invalidated). PK-only. Cf. drop_metadata_art_surface_indexes_v1.

					-- 4. Pair = jonction map + game_variant + nom composite, par titre
					CREATE TABLE IF NOT EXISTS map_mode_pair_definitions (
						title_slug             VARCHAR NOT NULL,
						pair_asset_id          VARCHAR NOT NULL,
						current_version_id     VARCHAR,
						name_canonical         VARCHAR,
						map_asset_id           VARCHAR,
						game_variant_asset_id  VARCHAR,
						mode_category          VARCHAR,
						last_fetched_at        TIMESTAMP,
						PRIMARY KEY (title_slug, pair_asset_id)
					);
					-- AUCUN index secondaire : upsertPair UPDATE map_asset_id /
					-- game_variant_asset_id / mode_category (ex-colonnes indexées) →
					-- surface ART corruptrice. PK-only. Cf. drop_metadata_art_surface_indexes_v1.

					-- 5. Relation N-N playlist <-> pair, avec poids de tirage
					CREATE TABLE IF NOT EXISTS playlist_pair_links (
						title_slug         VARCHAR NOT NULL,
						playlist_asset_id  VARCHAR NOT NULL,
						pair_asset_id      VARCHAR NOT NULL,
						weight             DOUBLE,
						PRIMARY KEY (title_slug, playlist_asset_id, pair_asset_id)
					);
					CREATE INDEX IF NOT EXISTS idx_playlist_pair_links_pair ON playlist_pair_links(title_slug, pair_asset_id);

					-- 6. File d'attente du fetcher (pattern Kinds, drain mensuel)
					CREATE TABLE IF NOT EXISTS catalog_fetch_queue (
						title_slug    VARCHAR NOT NULL,
						asset_type    VARCHAR NOT NULL,
						asset_id      VARCHAR NOT NULL,
						version_id    VARCHAR,
						enqueued_at   TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
						attempts      INTEGER DEFAULT 0,
						last_error    VARCHAR,
						PRIMARY KEY (title_slug, asset_type, asset_id)
					);
					-- PAS d'index secondaire : catalog_fetch_queue est DELETE/UPDATE per-row
					-- par le drain → toute surface ART la corrompt. La PK elle-même est retirée
					-- par rebuild_catalog_fetch_queue_drop_art_indexes (rebuild sans PK).

					-- 7. Labels normalisés multi-langues (sortie NormalizeModeLabel par langue)
					CREATE TABLE IF NOT EXISTS pair_mode_label_translations (
						title_slug      VARCHAR NOT NULL,
						pair_asset_id   VARCHAR NOT NULL,
						lang            VARCHAR NOT NULL,
						label           VARCHAR,
						PRIMARY KEY (title_slug, pair_asset_id, lang)
					);
					CREATE INDEX IF NOT EXISTS idx_pair_mode_label_lang ON pair_mode_label_translations(title_slug, lang);

					-- 8. Auto-détection des préfixes inconnus (alerting sur nouvelles catégories candidates)
					CREATE TABLE IF NOT EXISTS unknown_prefix_candidates (
						title_slug    VARCHAR NOT NULL,
						prefix        VARCHAR NOT NULL,
						n_matches     INTEGER DEFAULT 1,
						first_seen_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
						last_seen_at  TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
						pair_examples VARCHAR[],
						PRIMARY KEY (title_slug, prefix)
					);
					CREATE INDEX IF NOT EXISTS idx_unknown_prefix_n ON unknown_prefix_candidates(title_slug, n_matches DESC);
				`)
			},
		},
		{
			Name:        "drop_playlists_catalog_secondary_indexes",
			TargetDB:    migration.TargetMetadata,
			Description: "Supprime idx_playlists_catalog_active/_experience (source RC-E : UPDATE sur colonne ART-indexée corrompt metadata.duckdb)",
			ApplySchema: func(db *sql.DB) error {
				return migration.ExecScript(db, `
					DROP INDEX IF EXISTS idx_playlists_catalog_active;
					DROP INDEX IF EXISTS idx_playlists_catalog_experience;
				`)
			},
		},
		{
			Name:        "seed_ranked_playlists_catalog",
			TargetDB:    migration.TargetMetadata,
			Description: "Seed autoritatif des playlists classées (is_ranked=TRUE) depuis la référence rankedplaylists — corrige le bug récurrent is_ranked=false",
			ApplySchema: applyRankedPlaylistSeeds,
		},
		// Famille medal_definitions (base + 3 ALTER) → migrée ATOMIQUEMENT (b8).
		{
			Name:        "add_medal_definitions",
			TargetDB:    migration.TargetMetadata,
			Description: "Table medal_definitions (medal_name_id BIGINT, noms, descriptions)",
			ApplySchema: func(db *sql.DB) error {
				return migration.ExecScript(db, `
					CREATE TABLE IF NOT EXISTS medal_definitions (
						medal_name_id  BIGINT PRIMARY KEY,
						name_fr        VARCHAR NOT NULL,
						name_en        VARCHAR NOT NULL,
						description_fr VARCHAR DEFAULT '',
						description_en VARCHAR DEFAULT '',
						is_custom      BOOLEAN DEFAULT FALSE
					);
				`)
			},
		},
		{
			Name:        "medal_definitions_add_indices",
			TargetDB:    migration.TargetMetadata,
			Description: "medal_definitions : ajout difficulty_index + type_index (entiers Waypoint) — idempotent via IF NOT EXISTS",
			ApplySchema: func(db *sql.DB) error {
				return migration.ExecScript(db, `
					ALTER TABLE medal_definitions ADD COLUMN IF NOT EXISTS difficulty_index TINYINT DEFAULT 0;
					ALTER TABLE medal_definitions ADD COLUMN IF NOT EXISTS type_index TINYINT DEFAULT 0;
				`)
			},
		},
		{
			Name:        "enrich_medal_definitions_v2",
			TargetDB:    migration.TargetMetadata,
			Description: "medal_definitions : ajout difficulty (Normal/Heroic/Legendary/Mythic) + medal_type (multikill/spree/…) depuis difficulty_index/type_index existants",
			ApplySchema: func(db *sql.DB) error {
				return migration.ExecScript(db, `
					ALTER TABLE medal_definitions ADD COLUMN IF NOT EXISTS difficulty VARCHAR;
					ALTER TABLE medal_definitions ADD COLUMN IF NOT EXISTS medal_type VARCHAR;
					UPDATE medal_definitions SET
						difficulty = CASE difficulty_index
							WHEN 0 THEN 'Normal'
							WHEN 1 THEN 'Heroic'
							WHEN 2 THEN 'Legendary'
							WHEN 3 THEN 'Mythic'
							ELSE 'Normal'
						END,
						medal_type = CASE type_index
							WHEN 0 THEN 'spree'
							WHEN 1 THEN 'mode'
							WHEN 2 THEN 'multikill'
							WHEN 3 THEN 'proficiency'
							WHEN 4 THEN 'skill'
							WHEN 5 THEN 'style'
							ELSE ''
						END;
				`)
			},
		},
		{
			Name:        "medal_definitions_add_personal_score",
			TargetDB:    migration.TargetMetadata,
			Description: "medal_definitions : ajout personal_score (XP de carrière accordé par médaille, 0 par défaut)",
			ApplySchema: func(db *sql.DB) error {
				return migration.ExecScript(db, `
					ALTER TABLE medal_definitions ADD COLUMN IF NOT EXISTS personal_score INTEGER DEFAULT 0;
				`)
			},
		},
		{
			// Médaille custom « Vengeur » (id 9000000001) : native Halo 5, ré-exposée
			// pour Halo Infinite (LevelUp la calcule via citation_mappings). Sans cette
			// ligne, elle n'apparaît pas dans le catalogue médailles (tab Asset Drawer)
			// d'Infinite. INSERT idempotent (ON CONFLICT DO NOTHING) ; NE repeuple PAS
			// le reste du catalogue Infinite (déjà complet via refresh-metadata).
			Name:        "seed_custom_vengeur_medal",
			TargetDB:    migration.TargetMetadata,
			Description: "medal_definitions : seed médaille custom Vengeur (9000000001) pour Halo Infinite (native H5)",
			ApplySchema: func(db *sql.DB) error {
				return migration.ExecScript(db, `
					INSERT INTO medal_definitions
						(medal_name_id, name_fr, name_en, description_fr, description_en,
						 is_custom, difficulty_index, type_index, difficulty, medal_type, personal_score)
					VALUES
						(9000000001, 'Vengeur', 'Avenger',
						 'Tuez l''ennemi responsable de votre mort précédente.',
						 'Kill the enemy responsible for your previous death.',
						 TRUE, 0, 4, 'Normal', 'skill', 0)
					ON CONFLICT (medal_name_id) DO NOTHING;
				`)
			},
		},
		// Famille xbox_achievement_definitions (base + 4 ALTER/DELETE) → migrée ATOMIQUEMENT (b8).
		{
			Name:        "add_xbox_achievement_definitions",
			TargetDB:    migration.TargetMetadata,
			Description: "Table xbox_achievement_definitions : référentiel achievements Halo Infinite (bilingue EN/FR)",
			ApplySchema: func(db *sql.DB) error {
				return migration.ExecScript(db, `
					CREATE TABLE IF NOT EXISTS xbox_achievement_definitions (
						achievement_id   VARCHAR PRIMARY KEY,
						name_en          VARCHAR NOT NULL DEFAULT '',
						name_fr          VARCHAR NOT NULL DEFAULT '',
						description_en   VARCHAR,
						description_fr   VARCHAR,
						locked_desc_en   VARCHAR,
						locked_desc_fr   VARCHAR,
						gamerscore       INTEGER NOT NULL DEFAULT 0,
						image_url        VARCHAR,
						is_secret        BOOLEAN NOT NULL DEFAULT FALSE,
						rarity_category  VARCHAR,
						rarity_percent   FLOAT,
						fetched_at       TIMESTAMP DEFAULT CURRENT_TIMESTAMP
					);
				`)
			},
		},
		{
			Name:        "add_title_id_to_xbox_achievement_definitions",
			TargetDB:    migration.TargetMetadata,
			Description: "Colonne title_id sur xbox_achievement_definitions (filtre par jeu — halo_infinite) pour exclure les succès d'autres titres Xbox stockés avant l'introduction du filtre titleId.",
			ApplySchema: func(db *sql.DB) error {
				_, err := db.ExecContext(migration.BootCtx(), `ALTER TABLE xbox_achievement_definitions ADD COLUMN IF NOT EXISTS title_id VARCHAR DEFAULT ''`)
				return err
			},
		},
		{
			Name:        "cleanup_xbox_achievement_definitions_unknown_title",
			TargetDB:    migration.TargetMetadata,
			Description: "Supprime les succès Xbox sans title_id connu (insertés avant le filtre halo_infinite). L'utilisateur doit relancer sync-achievements après cette migration.",
			ApplySchema: func(db *sql.DB) error {
				_, err := db.ExecContext(migration.BootCtx(), `DELETE FROM xbox_achievement_definitions WHERE title_id = '' OR title_id IS NULL`)
				return err
			},
		},
		{
			Name:        "add_xbox_title_id_to_xbox_achievement_definitions",
			TargetDB:    migration.TargetMetadata,
			Description: "Colonne xbox_title_id sur xbox_achievement_definitions : identifiant Xbox numérique du titre source (ex: '1144039928' pour Halo Infinite). Peuplée lors du prochain sync-achievements. Permet au frontend de filtrer les succès cross-titres sans DELETE.",
			ApplySchema: func(db *sql.DB) error {
				_, err := db.ExecContext(migration.BootCtx(), `ALTER TABLE xbox_achievement_definitions ADD COLUMN IF NOT EXISTS xbox_title_id VARCHAR DEFAULT ''`)
				return err
			},
		},
		{
			Name:        "add_service_config_id_to_xbox_achievement_definitions",
			TargetDB:    migration.TargetMetadata,
			Description: "Colonne service_config_id (SCID Xbox) sur xbox_achievement_definitions. Le SCID est le seul discriminateur fiable par jeu : l'API Xbox retourne tous les achievements de la franchise quand on filtre par titleId. Peuplée lors du prochain sync-achievements.",
			ApplySchema: func(db *sql.DB) error {
				_, err := db.ExecContext(migration.BootCtx(), `ALTER TABLE xbox_achievement_definitions ADD COLUMN IF NOT EXISTS service_config_id VARCHAR DEFAULT ''`)
				return err
			},
		},
		// Déplacés depuis internal/migration/steps_metadata.go (b5 — leaves additifs).
		{
			Name:        "add_battlepass_asset_refs",
			TargetDB:    migration.TargetMetadata,
			Description: "Table battlepass_asset_refs pour visuels battle pass",
			ApplySchema: func(db *sql.DB) error {
				return migration.ExecScript(db, `
					CREATE TABLE IF NOT EXISTS battlepass_asset_refs (
						asset_key         VARCHAR PRIMARY KEY,
						asset_kind        VARCHAR NOT NULL,
						source_path       VARCHAR NOT NULL,
						cache_rel_path    VARCHAR NOT NULL,
						mime_type         VARCHAR NOT NULL DEFAULT 'image/png',
						image_source_path VARCHAR,
						source_origin     VARCHAR NOT NULL DEFAULT 'cms',
						first_cached_at   TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
						last_cached_at    TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
						last_accessed_at  TIMESTAMP DEFAULT CURRENT_TIMESTAMP
					);
					CREATE UNIQUE INDEX IF NOT EXISTS idx_battlepass_asset_refs_kind_source ON battlepass_asset_refs(asset_kind, source_path);
					CREATE INDEX IF NOT EXISTS idx_battlepass_asset_refs_accessed ON battlepass_asset_refs(last_accessed_at);
				`)
			},
		},
		{
			Name:        "add_battlepass_metadata",
			TargetDB:    migration.TargetMetadata,
			Description: "Tables battlepass_track_definitions/translations + battlepass_item_definitions/translations",
			ApplySchema: func(db *sql.DB) error {
				return migration.ExecScript(db, `
					CREATE TABLE IF NOT EXISTS battlepass_track_definitions (
						reward_track_path VARCHAR NOT NULL, content_hash VARCHAR NOT NULL,
						xp_per_rank INTEGER, battlepass_image_path VARCHAR, background_image_path VARCHAR,
						raw_payload_json VARCHAR NOT NULL, first_seen_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
						last_seen_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP, is_current BOOLEAN NOT NULL DEFAULT TRUE,
						PRIMARY KEY (reward_track_path, content_hash)
					);
					CREATE TABLE IF NOT EXISTS battlepass_track_translations (
						reward_track_path VARCHAR NOT NULL, content_hash VARCHAR NOT NULL,
						lang VARCHAR NOT NULL, track_name VARCHAR,
						first_seen_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP, last_seen_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
						PRIMARY KEY (reward_track_path, content_hash, lang)
					);
					CREATE INDEX IF NOT EXISTS idx_battlepass_track_translations_lookup ON battlepass_track_translations(reward_track_path, lang);
					CREATE TABLE IF NOT EXISTS battlepass_item_definitions (
						inventory_item_path VARCHAR NOT NULL, content_hash VARCHAR NOT NULL,
						quality VARCHAR, item_type VARCHAR, display_path VARCHAR,
						raw_payload_json VARCHAR NOT NULL, first_seen_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
						last_seen_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP, is_current BOOLEAN NOT NULL DEFAULT TRUE,
						PRIMARY KEY (inventory_item_path, content_hash)
					);
					CREATE TABLE IF NOT EXISTS battlepass_item_translations (
						inventory_item_path VARCHAR NOT NULL, content_hash VARCHAR NOT NULL,
						lang VARCHAR NOT NULL, title VARCHAR, description VARCHAR,
						first_seen_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP, last_seen_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
						PRIMARY KEY (inventory_item_path, content_hash, lang)
					);
					CREATE INDEX IF NOT EXISTS idx_battlepass_item_translations_lookup ON battlepass_item_translations(inventory_item_path, lang);
				`)
			},
		},
		{
			Name:        "add_challenge_metadata",
			TargetDB:    migration.TargetMetadata,
			Description: "Tables challenge_definitions + challenge_translations",
			ApplySchema: func(db *sql.DB) error {
				return migration.ExecScript(db, `
					CREATE TABLE IF NOT EXISTS challenge_definitions (
						challenge_path VARCHAR NOT NULL, content_hash VARCHAR NOT NULL,
						category VARCHAR, difficulty VARCHAR, threshold_for_success INTEGER,
						reward_xp INTEGER DEFAULT 0, secondary_reward_xp INTEGER DEFAULT 0,
						first_seen_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP, last_seen_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
						is_current BOOLEAN DEFAULT TRUE,
						PRIMARY KEY (challenge_path, content_hash)
					);
					CREATE INDEX IF NOT EXISTS idx_challenge_definitions_current ON challenge_definitions(challenge_path, is_current);
					CREATE INDEX IF NOT EXISTS idx_challenge_definitions_category ON challenge_definitions(category, difficulty);
					CREATE TABLE IF NOT EXISTS challenge_translations (
						challenge_path VARCHAR NOT NULL, content_hash VARCHAR NOT NULL,
						lang VARCHAR NOT NULL, title VARCHAR, description VARCHAR,
						first_seen_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP, last_seen_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
						PRIMARY KEY (challenge_path, content_hash, lang)
					);
					CREATE INDEX IF NOT EXISTS idx_challenge_translations_lookup ON challenge_translations(challenge_path, lang);
				`)
			},
		},
		{
			Name:        "drop_legacy_translation_tables",
			TargetDB:    migration.TargetMetadata,
			Description: "DROP mode_translations + playlist_translations",
			ApplySchema: func(db *sql.DB) error {
				return migration.ExecScript(db, `
					DROP TABLE IF EXISTS mode_translations;
					DROP TABLE IF EXISTS playlist_translations;
				`)
			},
		},
		{
			Name:        "add_career_rank_translations",
			TargetDB:    migration.TargetMetadata,
			Description: "Table career_rank_translations : libellés rangs de carrière dans toutes les langues exposées par GameCMS",
			ApplySchema: func(db *sql.DB) error {
				return migration.ExecScript(db, `
					CREATE TABLE IF NOT EXISTS career_rank_translations (
						rank_id  INTEGER NOT NULL,
						lang     VARCHAR NOT NULL,
						title    VARCHAR,
						subtitle VARCHAR,
						tier     VARCHAR,
						fetched_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
						PRIMARY KEY (rank_id, lang)
					);
					CREATE INDEX IF NOT EXISTS idx_career_rank_translations_lookup ON career_rank_translations(rank_id, lang);
				`)
			},
		},
		// Famille citation_mappings (base→pk→v2_fields) — relocalisée ATOMIQUEMENT (b5).
		{
			Name:        "add_citation_mappings",
			TargetDB:    migration.TargetMetadata,
			Description: "Table citation_mappings : mappings médaille→citation avec paliers, images et flags (portage populate_citation_mappings.py)",
			ApplySchema: func(db *sql.DB) error {
				return migration.ExecScript(db, `
					CREATE TABLE IF NOT EXISTS citation_mappings (
						citation_name_norm    VARCHAR NOT NULL,
						citation_name_display VARCHAR NOT NULL,
						mapping_type          VARCHAR NOT NULL DEFAULT 'medal',
						category              VARCHAR,
						image_path            VARCHAR,
						description           VARCHAR,
						tier_targets          VARCHAR,
						medal_id              UBIGINT,
						enabled               BOOLEAN NOT NULL DEFAULT TRUE
					);
					CREATE INDEX IF NOT EXISTS idx_citation_mappings_norm ON citation_mappings(citation_name_norm);
				`)
			},
		},
		{
			Name:        "add_citation_mappings_pk",
			TargetDB:    migration.TargetMetadata,
			Description: "Ajout PRIMARY KEY (citation_name_norm, medal_id) sur citation_mappings (nécessaire pour ON CONFLICT DO NOTHING)",
			ApplySchema: func(db *sql.DB) error {
				// DuckDB ne supporte pas ALTER TABLE ADD CONSTRAINT PK.
				// On recrée la table avec déduplication. La PK est sur citation_name_norm
				// uniquement : medal_id peut être NULL (citations non liées à une médaille
				// spécifique), ce qui interdit de l'inclure dans une PRIMARY KEY.
				return migration.ExecScript(db, `
					CREATE TABLE IF NOT EXISTS citation_mappings_v2 AS
						SELECT DISTINCT ON (citation_name_norm)
							citation_name_norm, citation_name_display, mapping_type,
							category, image_path, description, tier_targets, medal_id, enabled
						FROM citation_mappings;
					DROP TABLE IF EXISTS citation_mappings;
					ALTER TABLE citation_mappings_v2 RENAME TO citation_mappings;
					ALTER TABLE citation_mappings ADD PRIMARY KEY (citation_name_norm);
					CREATE INDEX IF NOT EXISTS idx_citation_mappings_norm ON citation_mappings(citation_name_norm);
				`)
			},
		},
		{
			Name:        "add_citation_mappings_v2_fields",
			TargetDB:    migration.TargetMetadata,
			Description: "citation_mappings : ajout medal_ids/stat_name/award_name/award_category/custom_function/composite_children/subcategory pour le moteur complet (parité avec scripts/populate_citation_mappings.py main)",
			ApplySchema: func(db *sql.DB) error {
				return migration.ExecScript(db, `
					ALTER TABLE citation_mappings ADD COLUMN IF NOT EXISTS medal_ids          VARCHAR;
					ALTER TABLE citation_mappings ADD COLUMN IF NOT EXISTS stat_name          VARCHAR;
					ALTER TABLE citation_mappings ADD COLUMN IF NOT EXISTS award_name         VARCHAR;
					ALTER TABLE citation_mappings ADD COLUMN IF NOT EXISTS award_category     VARCHAR;
					ALTER TABLE citation_mappings ADD COLUMN IF NOT EXISTS custom_function    VARCHAR;
					ALTER TABLE citation_mappings ADD COLUMN IF NOT EXISTS composite_children VARCHAR;
					ALTER TABLE citation_mappings ADD COLUMN IF NOT EXISTS subcategory        VARCHAR;
				`)
			},
		},
		{
			Name:        "add_citation_name_display_en",
			TargetDB:    migration.TargetMetadata,
			Description: "citation_mappings : ajout citation_name_display_en (nom anglais ; citations Infinite = copies de commendations H5, seul le calcul diffère). Servi locale-aware au read ; NULL → fallback FR.",
			ApplySchema: func(db *sql.DB) error {
				return migration.ExecScript(db, `
					ALTER TABLE citation_mappings ADD COLUMN IF NOT EXISTS citation_name_display_en VARCHAR;
				`)
			},
		},
		{
			Name:        "add_citation_description_en",
			TargetDB:    migration.TargetMetadata,
			Description: "citation_mappings : ajout description_en (description anglaise ; source = commendations Halo 5 officielles via l'API Metadata + traduction fidèle des maîtrises d'armes Infinite). Symétrique de citation_name_display_en. Servie locale-aware au read ; NULL → tooltip = nom seul (GH4).",
			ApplySchema: func(db *sql.DB) error {
				return migration.ExecScript(db, `
					ALTER TABLE citation_mappings ADD COLUMN IF NOT EXISTS description_en VARCHAR;
				`)
			},
		},
		{
			// Downstream de la famille citation (data-fix). DOIT rester title-owned
			// avec la chaîne : il UPDATE citation_mappings, absente en run global-only.
			Name:        "fix_citation_image_paths_double_encoded",
			TargetDB:    migration.TargetMetadata,
			Description: "Remplace les %XX URL-encodés par leurs caractères littéraux dans citation_mappings.image_path (16 entrées : Zéro défaut, Œil de lynx, etc.) — voir seed.go pour la liste canonique",
			ApplySchema: func(db *sql.DB) error {
				return migration.ExecScript(db, `
					UPDATE citation_mappings SET image_path = 'static/commendations/halo_5_guardians/H5G_citation_À_la_charge.png'                  WHERE citation_name_norm = 'charge';
					UPDATE citation_mappings SET image_path = 'static/commendations/halo_5_guardians/H5G_citation_Annexion_forcée.png'              WHERE citation_name_norm = 'forced_annexation';
					UPDATE citation_mappings SET image_path = 'static/commendations/halo_5_guardians/H5G_citation_Défenseur_du_drapeau.png'         WHERE citation_name_norm = 'flag_defender';
					UPDATE citation_mappings SET image_path = 'static/commendations/halo_5_guardians/H5G_citation_Je_te_tiens_!.png'                WHERE citation_name_norm = 'got_you';
					UPDATE citation_mappings SET image_path = 'static/commendations/halo_5_guardians/H5G_citation_Écrasement.png'                   WHERE citation_name_norm = 'splatter';
					UPDATE citation_mappings SET image_path = 'static/commendations/halo_5_guardians/H5G_citation_Grenade_à_fragmentation.png'      WHERE citation_name_norm = 'frag_grenade';
					UPDATE citation_mappings SET image_path = 'static/commendations/halo_5_guardians/H5G_citation_Grenade_à_plasma.png'             WHERE citation_name_norm = 'plasma_grenade';
					UPDATE citation_mappings SET image_path = 'static/commendations/halo_5_guardians/H5G_citation_Combat_rapproché.png'             WHERE citation_name_norm = 'close_combat';
					UPDATE citation_mappings SET image_path = 'static/commendations/halo_5_guardians/H5G_citation_Tir_à_la_tête.png'                WHERE citation_name_norm = 'headshot';
					UPDATE citation_mappings SET image_path = 'static/commendations/halo_5_guardians/H5G_citation_Œil_de_lynx.png'                  WHERE citation_name_norm = 'eagle_eye';
					UPDATE citation_mappings SET image_path = 'static/commendations/halo_5_guardians/H5G_citation_Flag_''em_down.png'               WHERE citation_name_norm = 'flag_em_down';
					UPDATE citation_mappings SET image_path = 'static/commendations/halo_5_guardians/H5G_citation_I''m_just_perfect.png'            WHERE citation_name_norm = 'im_just_perfect';
					UPDATE citation_mappings SET image_path = 'static/commendations/halo_5_guardians/H5G_citation_Destructeur_d''apparitions.png'   WHERE citation_name_norm = 'wraith_destroyer';
					UPDATE citation_mappings SET image_path = 'static/commendations/halo_5_guardians/H5G_citation_Tueur_d''Élites.png'              WHERE citation_name_norm = 'elite_slayer';
					UPDATE citation_mappings SET image_path = 'static/commendations/halo_5_guardians/H5G_citation_Tueur_de_répliques_de_Marines.png' WHERE citation_name_norm = 'marine_slayer';
					UPDATE citation_mappings SET image_path = 'static/commendations/halo_5_guardians/H5G_citation_Épée_à_énergie.png'               WHERE citation_name_norm = 'energy_sword_mastery';
				`)
			},
		},
		// Déplacé depuis internal/migration/steps_shared_pve.go (b3 pilote).
		{
			Name:        "add_pve_schema",
			TargetDB:    migration.TargetSharedPvE,
			Description: "Table pve_match_stats pour stats Firefight",
			ApplySchema: func(db *sql.DB) error {
				return migration.ExecScript(db, `
					CREATE TABLE IF NOT EXISTS pve_match_stats (
						match_id            VARCHAR NOT NULL,
						xuid                VARCHAR NOT NULL,
						waves_completed     INTEGER,
						boss_kills          INTEGER,
						grunt_kills         INTEGER,
						elite_kills         INTEGER,
						jackal_kills        INTEGER,
						brute_kills         INTEGER,
						hunter_kills        INTEGER,
						skimmer_kills       INTEGER,
						crawler_kills       INTEGER,
						soldier_kills       INTEGER,
						knight_kills        INTEGER,
						warden_kills        INTEGER,
						total_kills         INTEGER,
						deaths              INTEGER,
						created_at          TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
						PRIMARY KEY (match_id, xuid)
					);
					CREATE INDEX IF NOT EXISTS idx_pve_match ON pve_match_stats(match_id);
					CREATE INDEX IF NOT EXISTS idx_pve_xuid ON pve_match_stats(xuid);
				`)
			},
		},
		// Déplacés depuis internal/migration/steps_shared_*.go (b3 batch Shared).
		{
			Name:        "shared_add_t0_quality",
			TargetDB:    migration.TargetShared,
			Description: "Colonne t0_quality sur match_registry + repurpose real_start_time en début gameplay UTC (Match Timeline T0 Phase 2)",
			ApplySchema: func(db *sql.DB) error {
				return migration.ExecScript(db, `
					ALTER TABLE match_registry ADD COLUMN IF NOT EXISTS t0_quality VARCHAR;
				`)
			},
		},
		// Déplacé depuis internal/migration/steps_engagement.go (b17, partie shared de
		// la famille engagement). ALTER match_registry (créée par le god-file shared,
		// racine globale). Guard tableExists : ordre canonicalOrder peut l'évaluer avant.
		{
			Name:        "add_match_intensity_to_match_registry",
			TargetDB:    migration.TargetShared,
			Description: "Ajoute match_intensity (DOUBLE) a shared.match_registry (events/min/joueur du lobby, caracteristique permanente du match)",
			ApplySchema: func(db *sql.DB) error {
				exists, err := migration.TableExists(db, "match_registry")
				if err != nil {
					return fmt.Errorf("add_match_intensity: check table: %w", err)
				}
				if !exists {
					return nil
				}
				return migration.ExecScript(db, `
					ALTER TABLE match_registry ADD COLUMN IF NOT EXISTS match_intensity DOUBLE;
				`)
			},
		},
		// ─── Leaves shared additifs (tables/vues append-only autonomes + backfill) → b18.
		// world_csr_leaderboard : table snapshots + vue _latest (paire ATOMIQUE, la vue
		// est créée en v1 puis remplacée par _latest_by_batch).
		{
			Name:        "create_world_csr_leaderboard_snapshots",
			TargetDB:    migration.TargetShared,
			Description: "Crée world_csr_leaderboard_snapshots (append-only) + vue _latest pour le classement CSR mondial scrapé depuis Halo Waypoint",
			ApplySchema: func(db *sql.DB) error {
				return migration.ExecScript(db, `
					CREATE SEQUENCE IF NOT EXISTS wcl_seq START 1;

					CREATE TABLE IF NOT EXISTS world_csr_leaderboard_snapshots (
						id            BIGINT PRIMARY KEY DEFAULT nextval('wcl_seq'),
						season_id     VARCHAR NOT NULL,
						playlist_id   VARCHAR NOT NULL,
						rank          INTEGER NOT NULL,
						gamertag      VARCHAR NOT NULL,
						csr_value     INTEGER NOT NULL,
						tier_derived  VARCHAR,
						fetched_at    TIMESTAMP,
						written_at    TIMESTAMP DEFAULT CURRENT_TIMESTAMP
					);

					CREATE INDEX IF NOT EXISTS idx_wcl_lookup
						ON world_csr_leaderboard_snapshots(season_id, playlist_id, rank, written_at);

					CREATE OR REPLACE VIEW world_csr_leaderboard_latest AS
						SELECT *
						FROM world_csr_leaderboard_snapshots
						QUALIFY ROW_NUMBER() OVER (
							PARTITION BY season_id, playlist_id, rank
							ORDER BY written_at DESC, id DESC
						) = 1;
				`)
			},
		},
		{
			Name:        "world_csr_leaderboard_latest_by_batch",
			TargetDB:    migration.TargetShared,
			Description: "Remplace world_csr_leaderboard_latest : dernier batch (fetched_at) par (season, playlist) au lieu du dernier par rang (fix snapshot Frankenstein)",
			ApplySchema: func(db *sql.DB) error {
				return migration.ExecScript(db, `
					CREATE OR REPLACE VIEW world_csr_leaderboard_latest AS
						SELECT s.*
						FROM world_csr_leaderboard_snapshots s
						WHERE s.fetched_at = (
							SELECT max(s2.fetched_at)
							FROM world_csr_leaderboard_snapshots s2
							WHERE s2.season_id = s.season_id
							  AND s2.playlist_id = s.playlist_id
						);
				`)
			},
		},
		{
			// PMT-7 (MT-03) : dimension titre sur le leaderboard CSR mondial. Colonne
			// title_slug (défaut halo_infinite → rétro-compatible byte-identique) + vue
			// _latest partitionnée par titre + index lookup titré. Permet à
			// GetCSRWorldLeaderboard(titleSlug, …) de filtrer par titre.
			Name:        "add_title_slug_to_world_csr_leaderboard",
			TargetDB:    migration.TargetShared,
			Description: "PMT-7 : colonne title_slug sur world_csr_leaderboard_snapshots + vue _latest par titre",
			ApplySchema: func(db *sql.DB) error {
				// DuckDB ne supporte pas NOT NULL sur ADD COLUMN ; le DEFAULT rétro-remplit
				// les lignes existantes ET les nouvelles (les writers posent toujours le slug).
				return migration.ExecScript(db, `
					ALTER TABLE world_csr_leaderboard_snapshots
						ADD COLUMN IF NOT EXISTS title_slug VARCHAR DEFAULT 'halo_infinite';

					CREATE INDEX IF NOT EXISTS idx_wcl_lookup_title
						ON world_csr_leaderboard_snapshots(title_slug, season_id, playlist_id, rank, written_at);

					CREATE OR REPLACE VIEW world_csr_leaderboard_latest AS
						SELECT s.*
						FROM world_csr_leaderboard_snapshots s
						WHERE s.fetched_at = (
							SELECT max(s2.fetched_at)
							FROM world_csr_leaderboard_snapshots s2
							WHERE s2.title_slug = s.title_slug
							  AND s2.season_id = s.season_id
							  AND s2.playlist_id = s.playlist_id
						);
				`)
			},
		},
		{
			// B1 (leaderboard sans trous) : le scraper Waypoint parse DÉJÀ le xuid de
			// chaque joueur (leaderboard_scraper.go) mais il était jeté à la persistance
			// (table sans colonne xuid). On l'ajoute (nullable — les lignes pré-migration
			// restent NULL, le prochain scrape les remplit) pour alimenter l'enrichissement
			// mondial SANS re-résolution PeopleHub, et pour activer la mise en évidence du
			// joueur courant (isLocalXUID) sur le classement mondial. Colonne non indexée /
			// non-PK → aucun risque ART. La vue _latest est recréée : DuckDB fige
			// l'expansion de `s.*` à la création, il faut donc la reconstruire pour exposer
			// la nouvelle colonne.
			Name:        "add_xuid_to_world_csr_leaderboard",
			TargetDB:    migration.TargetShared,
			Description: "B1 : colonne xuid sur world_csr_leaderboard_snapshots (déjà scrapé de Waypoint) + vue _latest exposant xuid",
			ApplySchema: func(db *sql.DB) error {
				return migration.ExecScript(db, `
					ALTER TABLE world_csr_leaderboard_snapshots
						ADD COLUMN IF NOT EXISTS xuid VARCHAR;

					CREATE OR REPLACE VIEW world_csr_leaderboard_latest AS
						SELECT s.*
						FROM world_csr_leaderboard_snapshots s
						WHERE s.fetched_at = (
							SELECT max(s2.fetched_at)
							FROM world_csr_leaderboard_snapshots s2
							WHERE s2.title_slug = s.title_slug
							  AND s2.season_id = s.season_id
							  AND s2.playlist_id = s.playlist_id
						);
				`)
			},
		},
		{
			// C2 (saisons) : la page classement Waypoint expose la liste des saisons CSR
			// avec leur nom d'Operation (displayName EN + translations par locale, ex
			// fr-FR "Ombres"). Le scraper les parse (FetchCatalog) ; on les persiste ici
			// pour un libellé autoritatif "Saison N · Nom" dans les sélecteurs (page
			// classement + page player), au lieu du "Saison N" dérivé du seul season_id.
			//
			// TargetShared (et non metadata) : la SOURCE est le scrape Waypoint et le
			// SEUL writer sanctionné détenu par world_leaderboard_cron est le writer
			// shared (provider.AcquireWriter). Écrire dans metadata depuis ce cron
			// contredirait le writer mono-process (contention avec la sync — cf. A3).
			// Co-localisé avec world_csr_leaderboard_snapshots (même cron, même scrape).
			Name:        "create_season_catalog",
			TargetDB:    migration.TargetShared,
			Description: "C2 : season_catalog (noms + traduction FR des saisons CSR Waypoint)",
			ApplySchema: func(db *sql.DB) error {
				return migration.ExecScript(db, `
					CREATE TABLE IF NOT EXISTS season_catalog (
						title_slug       VARCHAR NOT NULL,
						season_id        VARCHAR NOT NULL,
						display_name     VARCHAR,
						name_fr          VARCHAR,
						season_major     INTEGER,
						season_minor     INTEGER,
						first_seen_at    TIMESTAMP,
						last_fetched_at  TIMESTAMP,
						PRIMARY KEY (title_slug, season_id)
					);
					-- AUCUN index secondaire : display_name/name_fr sont MUTÉS par l'upsert
					-- (SELECT-then-write, ops.RefreshSeasonCatalog). PK-only → l'UPDATE ne
					-- touche pas d'index secondaire (surface ART #23046). Table minuscule
					-- (≤ ~15 saisons) → scan séquentiel instantané.
				`)
			},
		},
		{
			// Joueurs « privés / sans données » du classement mondial : un joueur dont
			// l'historique matchmade est inaccessible (privacy Xbox → 403/vide) ressort
			// de l'enrichissement avec 0 stat. On le marque ici pour (a) ne plus le
			// re-fetcher aux runs suivants (backfill -skip-existing), (b) le masquer du
			// classement affiché (anti-join). Marqueur par (titre, saison, gamertag).
			Name:        "create_world_player_no_data",
			TargetDB:    migration.TargetShared,
			Description: "Marqueur des joueurs privés/sans données du classement mondial (skip backfill + masquage affichage)",
			ApplySchema: func(db *sql.DB) error {
				return migration.ExecScript(db, `
					CREATE TABLE IF NOT EXISTS world_player_no_data (
						title_slug  VARCHAR NOT NULL,
						season_id   VARCHAR NOT NULL,
						gamertag    VARCHAR NOT NULL,
						marked_at   TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
						PRIMARY KEY (title_slug, season_id, gamertag)
					);
					-- AUCUN index secondaire (PK-only) : insert-or-ignore, jamais d'UPDATE
					-- sur colonne indexée → hors surface ART #23046. Table petite.
				`)
			},
		},
		{
			Name:        "shared_create_player_squad_offset",
			TargetDB:    migration.TargetShared,
			Description: "LUSR v2 Sprint 1.C — player_squad_offset (append-only) + vue _latest : offset synergie par paire de coéquipiers",
			ApplySchema: func(db *sql.DB) error {
				return migration.ExecScript(db, `
					CREATE SEQUENCE IF NOT EXISTS player_squad_offset_seq START 1;
					CREATE TABLE IF NOT EXISTS player_squad_offset (
						id              BIGINT DEFAULT nextval('player_squad_offset_seq') PRIMARY KEY,
						xuid            VARCHAR NOT NULL,
						partner_xuid    VARCHAR NOT NULL,
						playlist_group  VARCHAR NOT NULL,
						offset_value    DOUBLE  NOT NULL,
						match_count     INTEGER NOT NULL DEFAULT 0,
						source          VARCHAR NOT NULL,
						written_at      TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
					);
					CREATE INDEX IF NOT EXISTS idx_pso_lookup
						ON player_squad_offset(xuid, playlist_group, partner_xuid, written_at DESC);

					CREATE OR REPLACE VIEW player_squad_offset_latest AS
					SELECT o.*
					FROM player_squad_offset o
					JOIN (
						SELECT xuid, partner_xuid, playlist_group, MAX(written_at) AS max_written_at
						FROM player_squad_offset
						GROUP BY xuid, partner_xuid, playlist_group
					) m
						ON o.xuid = m.xuid
						AND o.partner_xuid = m.partner_xuid
						AND o.playlist_group = m.playlist_group
						AND o.written_at = m.max_written_at;
				`)
			},
		},
		{
			Name:        "create_world_player_season_stats",
			TargetDB:    migration.TargetShared,
			Description: "Crée world_player_season_stats (append-only) + vue _latest : stats joueur du classement mondial par saison CSR x playlist (compteurs bruts, ratios dérivés à la lecture)",
			ApplySchema: func(db *sql.DB) error {
				return migration.ExecScript(db, `
					CREATE SEQUENCE IF NOT EXISTS wpss_seq START 1;

					CREATE TABLE IF NOT EXISTS world_player_season_stats (
						id           BIGINT PRIMARY KEY DEFAULT nextval('wpss_seq'),
						title_slug   VARCHAR NOT NULL DEFAULT 'halo_infinite',
						gamertag     VARCHAR NOT NULL,
						season_id    VARCHAR NOT NULL,
						playlist_id  VARCHAR NOT NULL,
						match_count  INTEGER NOT NULL DEFAULT 0,
						win_count    INTEGER NOT NULL DEFAULT 0,
						loss_count   INTEGER NOT NULL DEFAULT 0,
						tie_count    INTEGER NOT NULL DEFAULT 0,
						dnf_count    INTEGER NOT NULL DEFAULT 0,
						kills        BIGINT NOT NULL DEFAULT 0,
						deaths       BIGINT NOT NULL DEFAULT 0,
						assists      BIGINT NOT NULL DEFAULT 0,
						playtime_s   BIGINT NOT NULL DEFAULT 0,
						medal_count  BIGINT NOT NULL DEFAULT 0,
						kda          DOUBLE NOT NULL DEFAULT 0,
						accuracy     DOUBLE NOT NULL DEFAULT 0,
						damage_dealt BIGINT NOT NULL DEFAULT 0,
						damage_taken BIGINT NOT NULL DEFAULT 0,
						computed_at  TIMESTAMP,
						written_at   TIMESTAMP DEFAULT CURRENT_TIMESTAMP
					);

					CREATE INDEX IF NOT EXISTS idx_wpss_lookup
						ON world_player_season_stats(title_slug, season_id, playlist_id, gamertag, written_at);

					CREATE OR REPLACE VIEW world_player_season_stats_latest AS
						SELECT *
						FROM world_player_season_stats
						QUALIFY ROW_NUMBER() OVER (
							PARTITION BY title_slug, gamertag, season_id, playlist_id
							ORDER BY written_at DESC, id DESC
						) = 1;
				`)
			},
		},
		{
			Name:        "shared_create_skill_v2_tables",
			TargetDB:    migration.TargetShared,
			Description: "LUSR v2 — player_skill_state_v2 (append-only) + lusr_hyperparams_v2 + vues _latest",
			ApplySchema: func(db *sql.DB) error {
				return migration.ExecScript(db, `
					CREATE SEQUENCE IF NOT EXISTS player_skill_state_v2_seq START 1;
					CREATE TABLE IF NOT EXISTS player_skill_state_v2 (
						id              BIGINT DEFAULT nextval('player_skill_state_v2_seq') PRIMARY KEY,
						xuid            VARCHAR NOT NULL,
						playlist_group  VARCHAR NOT NULL,
						mu              DOUBLE  NOT NULL,
						sigma           DOUBLE  NOT NULL,
						experience      INTEGER NOT NULL DEFAULT 0,
						last_match_id   VARCHAR,
						last_match_at   TIMESTAMP,
						written_at      TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
					);
					CREATE INDEX IF NOT EXISTS idx_pssv2_xuid_group_written
						ON player_skill_state_v2(xuid, playlist_group, written_at DESC);

					CREATE OR REPLACE VIEW player_skill_state_v2_latest AS
					SELECT s.*
					FROM player_skill_state_v2 s
					JOIN (
						SELECT xuid, playlist_group, MAX(written_at) AS max_written_at
						FROM player_skill_state_v2
						GROUP BY xuid, playlist_group
					) m
						ON s.xuid = m.xuid
						AND s.playlist_group = m.playlist_group
						AND s.written_at = m.max_written_at;

					CREATE SEQUENCE IF NOT EXISTS lusr_hyperparams_v2_seq START 1;
					CREATE TABLE IF NOT EXISTS lusr_hyperparams_v2 (
						id              BIGINT DEFAULT nextval('lusr_hyperparams_v2_seq') PRIMARY KEY,
						playlist_group  VARCHAR NOT NULL,
						name            VARCHAR NOT NULL,
						value           DOUBLE  NOT NULL,
						source          VARCHAR NOT NULL,
						written_at      TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
					);
					CREATE INDEX IF NOT EXISTS idx_lhv2_group_name_written
						ON lusr_hyperparams_v2(playlist_group, name, written_at DESC);

					CREATE OR REPLACE VIEW lusr_hyperparams_v2_latest AS
					SELECT h.*
					FROM lusr_hyperparams_v2 h
					JOIN (
						SELECT playlist_group, name, MAX(written_at) AS max_written_at
						FROM lusr_hyperparams_v2
						GROUP BY playlist_group, name
					) m
						ON h.playlist_group = m.playlist_group
						AND h.name = m.name
						AND h.written_at = m.max_written_at;
				`)
			},
		},
		{
			Name:        "shared_backfill_is_ranked_and_season",
			TargetDB:    migration.TargetShared,
			Description: "Phase 1 plan CSR : ALTER +season_id, backfill is_ranked + season_id via heuristique nom playlist + bornes saison",
			ApplySchema: func(db *sql.DB) error {
				return migration.ExecScript(db, `
					ALTER TABLE match_registry ADD COLUMN IF NOT EXISTS season_id VARCHAR;
					CREATE INDEX IF NOT EXISTS idx_match_registry_season_id ON match_registry(season_id);
				`)
			},
			ApplyBackfill: func(db *sql.DB) error {
				if _, err := db.ExecContext(migration.BootCtx(), `
					UPDATE match_registry
					SET is_ranked = TRUE
					WHERE COALESCE(is_ranked, FALSE) = FALSE
					  AND (
					      LOWER(COALESCE(playlist_name, '')) LIKE '%ranked%'
					      OR LOWER(COALESCE(pair_name, '')) LIKE 'ranked:%'
					  );
				`); err != nil {
					return err
				}
				_, err := db.ExecContext(migration.BootCtx(), `
					UPDATE match_registry
					SET season_id = CASE
						WHEN start_time >= TIMESTAMP '2025-11-18' THEN 'CsrSeason13-1'
						WHEN start_time >= TIMESTAMP '2025-08-05' THEN 'CsrSeason12-1'
						WHEN start_time >= TIMESTAMP '2025-05-06' THEN 'CsrSeason11-1'
						WHEN start_time >= TIMESTAMP '2025-02-04' THEN 'CsrSeason10-1'
						WHEN start_time >= TIMESTAMP '2024-11-05' THEN 'CsrSeason9-1'
						WHEN start_time >= TIMESTAMP '2024-07-30' THEN 'CsrSeason8-1'
						WHEN start_time >= TIMESTAMP '2024-04-30' THEN 'CsrSeason7-1'
						WHEN start_time >= TIMESTAMP '2024-01-30' THEN 'CsrSeason6-1'
						WHEN start_time >= TIMESTAMP '2023-10-17' THEN 'CsrSeason5-1'
						WHEN start_time >= TIMESTAMP '2023-06-20' THEN 'CsrSeason4-1'
						WHEN start_time >= TIMESTAMP '2023-03-07' THEN 'CsrSeason3-1'
						WHEN start_time >= TIMESTAMP '2022-11-08' THEN 'CsrSeason2-2'
						WHEN start_time >= TIMESTAMP '2022-05-03' THEN 'CsrSeason2'
						WHEN start_time >= TIMESTAMP '2021-12-08' THEN 'CsrSeason1'
						ELSE NULL
					END
					WHERE is_ranked = TRUE AND season_id IS NULL;
				`)
				return err
			},
		},
		{
			Name:        "shared_add_participation_info_booleans",
			TargetDB:    migration.TargetShared,
			Description: "ParticipationInfo booleans sur match_participants pour LUSR v2 §9 quit penalty",
			ApplySchema: func(db *sql.DB) error {
				return migration.ExecScript(db, `
					ALTER TABLE match_participants ADD COLUMN IF NOT EXISTS present_at_beginning BOOLEAN;
					ALTER TABLE match_participants ADD COLUMN IF NOT EXISTS present_at_completion BOOLEAN;
					ALTER TABLE match_participants ADD COLUMN IF NOT EXISTS joined_in_progress BOOLEAN;
					ALTER TABLE match_participants ADD COLUMN IF NOT EXISTS left_in_progress BOOLEAN;
				`)
			},
		},
		{
			Name:        "shared_add_participation_timestamps",
			TargetDB:    migration.TargetShared,
			Description: "ParticipationInfo timestamps (FirstJoinedTime, LastLeaveTime) sur match_participants pour LUSR v2 quit ordering",
			ApplySchema: func(db *sql.DB) error {
				return migration.ExecScript(db, `
					ALTER TABLE match_participants ADD COLUMN IF NOT EXISTS first_joined_time TIMESTAMPTZ;
					ALTER TABLE match_participants ADD COLUMN IF NOT EXISTS last_leave_time TIMESTAMPTZ;
				`)
			},
		},
		{
			Name:        "add_shared_match_csrs",
			TargetDB:    migration.TargetShared,
			Description: "Table shared.match_csrs : CSR par-match par-joueur (capture all participants)",
			ApplySchema: func(db *sql.DB) error {
				return migration.ExecScript(db, `
					CREATE TABLE IF NOT EXISTS match_csrs (
						match_id                     VARCHAR NOT NULL,
						xuid                         VARCHAR NOT NULL,
						rating_type                  VARCHAR NOT NULL DEFAULT 'CSR',
						rating_value                 FLOAT,
						tier                         VARCHAR,
						sub_tier                     SMALLINT DEFAULT 0,
						tier_label                   VARCHAR,
						rating_delta                 FLOAT,
						measurement_matches_remaining INTEGER DEFAULT 0,
						season_id                    VARCHAR,
						created_at                   TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
						updated_at                   TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
						PRIMARY KEY (match_id, xuid)
					);
					CREATE INDEX IF NOT EXISTS idx_match_csrs_xuid    ON match_csrs(xuid);
					CREATE INDEX IF NOT EXISTS idx_match_csrs_season  ON match_csrs(season_id);
					CREATE INDEX IF NOT EXISTS idx_match_csrs_match   ON match_csrs(match_id);
				`)
			},
		},
		// Déplacé depuis internal/migration/steps_shared_seed_tier_boundaries_v2.go.
		{
			Name:        "shared_seed_tier_boundaries_v2",
			TargetDB:    migration.TargetShared,
			Description: "LUSR v2 Phase 3e v2 — seed des tier_boundary_* (Bronze..Onyx) dans lusr_hyperparams_v2",
			ApplySchema: func(db *sql.DB) error {
				return nil // table déjà créée par shared_create_skill_v2_tables (reste dans le registre global)
			},
			ApplyBackfill: func(db *sql.DB) error {
				boundaries := []struct {
					name  string
					value float64
				}{
					{"tier_boundary_bronze", 0.0},
					{"tier_boundary_silver", 21.0},
					{"tier_boundary_gold", 22.0},
					{"tier_boundary_platinum", 25.0},
					{"tier_boundary_diamond", 25.8},
					{"tier_boundary_onyx", 27.0},
				}
				groups := []string{"arena_slayer", "arena_objectif", "btb", "chaos"}
				for _, g := range groups {
					for _, b := range boundaries {
						if _, err := db.ExecContext(migration.BootCtx(), `
							INSERT INTO lusr_hyperparams_v2 (playlist_group, name, value, source)
							SELECT ?, ?, ?, 'phase_3e_v2_default'
							WHERE NOT EXISTS (
								SELECT 1 FROM lusr_hyperparams_v2
								WHERE playlist_group = ? AND name = ?
							)`,
							g, b.name, b.value, g, b.name); err != nil {
							return err
						}
					}
				}
				return nil
			},
		},
		// Leaves player standalone (tables propres, aucune famille atomique) → migrés (b14).
		{
			Name:        "add_player_assists_model",
			TargetDB:    migration.TargetPlayer,
			Description: "Table player_assists_model : coefs OLS multi-variée expected_assists par mode",
			ApplySchema: func(db *sql.DB) error {
				return migration.ExecScript(db, `
					CREATE TABLE IF NOT EXISTS player_assists_model (
						game_variant_name VARCHAR PRIMARY KEY,
						coef_intercept    DOUBLE NOT NULL DEFAULT 0,
						coef_kills        DOUBLE NOT NULL DEFAULT 0,
						coef_deaths       DOUBLE NOT NULL DEFAULT 0,
						coef_damage_dealt DOUBLE NOT NULL DEFAULT 0,
						coef_damage_taken DOUBLE NOT NULL DEFAULT 0,
						coef_mmr_delta    DOUBLE NOT NULL DEFAULT 0,
						r2                DOUBLE NOT NULL DEFAULT 0,
						n_samples         INTEGER NOT NULL DEFAULT 0,
						computed_at       TIMESTAMP DEFAULT CURRENT_TIMESTAMP
					);
				`)
			},
		},
		{
			Name:        "create_lusr_component_history",
			TargetDB:    migration.TargetPlayer,
			Description: "Table lusr_component_history (V2 §1 — alimentation live + backfill)",
			ApplySchema: func(db *sql.DB) error {
				return migration.ExecScript(db, `
					CREATE TABLE IF NOT EXISTS lusr_component_history (
						match_id        VARCHAR NOT NULL,
						component_name  VARCHAR NOT NULL,
						value           DOUBLE  NOT NULL,
						weight          DOUBLE  NOT NULL,
						computed_at     TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
						PRIMARY KEY (match_id, component_name)
					);
					CREATE INDEX IF NOT EXISTS idx_lch_component ON lusr_component_history(component_name);
					CREATE INDEX IF NOT EXISTS idx_lch_match ON lusr_component_history(match_id);
				`)
			},
		},
		{
			Name:        "create_coach_proposal_player_schema",
			TargetDB:    migration.TargetPlayer,
			Description: "Table coach_proposal pour le pont coach_advisor → Prestige (ADR 0020)",
			ApplySchema: func(db *sql.DB) error {
				return migration.ExecScript(db, `
					CREATE TABLE IF NOT EXISTS coach_proposal (
						id                    VARCHAR PRIMARY KEY,
						user_id               VARCHAR NOT NULL,
						title_slug            VARCHAR NOT NULL,
						kind                  VARCHAR NOT NULL,
						template_id           VARCHAR,
						challenges_spec_json  VARCHAR,
						suggested_tier        VARCHAR,
						source_signal         VARCHAR NOT NULL,
						source_metric         VARCHAR,
						radar_axis            VARCHAR,
						strength              DOUBLE,
						origin                VARCHAR NOT NULL,
						reason_key_en         VARCHAR,
						reason_key_fr         VARCHAR,
						reason_params         VARCHAR,
						status                VARCHAR NOT NULL DEFAULT 'pending',
						created_at            TIMESTAMP NOT NULL,
						expires_at            TIMESTAMP,
						resolved_at           TIMESTAMP,
						resolved_ref          VARCHAR,
						superseded_by         VARCHAR,
						superseded_at         TIMESTAMP,
						obsoleted_at          TIMESTAMP
					);
					-- PAS d'index sur (user_id, title_slug, status) : status est muté par
					-- MarkAccepted/Dismissed/Superseded/Obsoleted → surface ART #23046 sur la
					-- player DB. La query GET pending scanne (table minuscule). Drop sur DB
					-- existantes : drop_coach_proposal_status_art_index_v1.
					CREATE INDEX IF NOT EXISTS idx_coach_proposal_metric_axis
						ON coach_proposal(user_id, title_slug, source_metric, radar_axis);
				`)
			},
		},
	}
	// Baseline squashée v1 : racine player title-owned « à plat » (remplace les 33
	// steps create_base_player_schema..player_append_only_csr_snapshots_v1 sur DB vierge).
	// Cf. steps_player_baseline.go + plan PLAN_MIGRATION_SQUASH_BASELINE_2026-07 (M3).
	steps = append(steps, playerBaselineSteps()...)
	// Steps player CONSOMMATEURS restants (perf_chain, psa_checked, fix_career_xp,
	// dedup/streak) — post-baseline dans canonicalOrder.
	steps = append(steps, playerSteps()...)
	// Chaîne match_skill_rank player CONSOMMATRICE (append-only + vues) → b20.
	steps = append(steps, playerMatchSkillRankSteps()...)
	// Repairs/rebuilds player CONSOMMATEURS (repair PK pme/citations, rebuild career) → b21.
	steps = append(steps, playerRepairSteps()...)
	// Conversions append-only CONSOMMATRICES (csr_snapshots, match_csrs, pve_match_stats) → b22.
	steps = append(steps, appendOnlyMiscSteps()...)
	// RACINE player (god-file base + prestige/campaign/progression + drop_notifications) → b25.
	steps = append(steps, playerBaseSteps()...)
	// RACINE metadata (asset_translations + fix_super_fiesta) — dernier root → b26.
	steps = append(steps, metadataRootSteps()...)
	// God-file shared (34 steps, RACINE shared_matches_v2 : match_registry/participants/…) → b23.
	steps = append(steps, sharedCoreSteps()...)
	// Schéma de référence inter-titres : positions monde par kill (Halo 5 natif,
	// Infinite plus tard). Cf. steps_shared_kill_positions.go.
	steps = append(steps, sharedKillPositionsSteps()...)
	// Schéma de référence inter-titres : compteur par-match des commendations natives
	// (Halo 5 natif, AXE B prod-gate). Cf. steps_shared_commendations.go.
	steps = append(steps, sharedCommendationsSteps()...)
	// Steps social CONSOMMATEURS (media ALTERs, records family, purge, rekey) → b19.
	steps = append(steps, sharedSocialSteps()...)
	// Racines du tier social (schémas de base media/notifications/prestige) → b24.
	steps = append(steps, sharedSocialRootSteps()...)
	return steps
}

// StepsFor filtre Steps() par target — c'est la fonction enregistrée comme
// provider via migration.SetTitleStepsProvider.
func StepsFor(target migration.TargetDB) []migration.Migration {
	all := Steps()
	if len(all) == 0 {
		return nil
	}
	out := make([]migration.Migration, 0, len(all))
	for _, m := range all {
		if m.TargetDB == target {
			out = append(out, m)
		}
	}
	return out
}
