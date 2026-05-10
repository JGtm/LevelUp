package migration

// steps_metadata.go — migrations ciblant metadata.duckdb.
// Portage de src/data/migration/steps/add_asset_translations.py, add_battlepass_*.py,
// add_challenge_metadata.py, add_medal_definitions.py, add_weapon_labels.py,
// drop_legacy_translation_tables.py.

import (
	"database/sql"
	"fmt"
)

func init() {
	Register(Migration{
		Name:        "add_asset_translations",
		TargetDB:    TargetMetadata,
		Description: "Tables asset_translations + medal_translations (pivot multi-langue)",
		ApplySchema: func(db *sql.DB) error {
			return execScript(db, `
				CREATE TABLE IF NOT EXISTS asset_translations (
					asset_id    VARCHAR NOT NULL,
					asset_type  VARCHAR NOT NULL,
					lang        VARCHAR NOT NULL,
					name        VARCHAR NOT NULL,
					description VARCHAR,
					fetched_at  TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
					PRIMARY KEY (asset_id, asset_type, lang)
				);
				CREATE INDEX IF NOT EXISTS idx_asset_tr_id_type ON asset_translations(asset_id, asset_type);
				CREATE TABLE IF NOT EXISTS medal_translations (
					medal_name_id BIGINT NOT NULL,
					lang          VARCHAR NOT NULL,
					name          VARCHAR NOT NULL,
					description   VARCHAR,
					PRIMARY KEY (medal_name_id, lang)
				);
			`)
		},
	})

	Register(Migration{
		Name:        "add_battlepass_asset_refs",
		TargetDB:    TargetMetadata,
		Description: "Table battlepass_asset_refs pour visuels battle pass",
		ApplySchema: func(db *sql.DB) error {
			return execScript(db, `
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
	})

	Register(Migration{
		Name:        "add_battlepass_metadata",
		TargetDB:    TargetMetadata,
		Description: "Tables battlepass_track_definitions/translations + battlepass_item_definitions/translations",
		ApplySchema: func(db *sql.DB) error {
			return execScript(db, `
				CREATE TABLE IF NOT EXISTS battlepass_track_definitions (
					reward_track_path VARCHAR NOT NULL, content_hash VARCHAR NOT NULL,
					xp_per_rank INTEGER, battlepass_image_path VARCHAR, background_image_path VARCHAR,
					raw_payload_json VARCHAR NOT NULL, first_seen_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
					last_seen_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP, is_current BOOLEAN NOT NULL DEFAULT TRUE,
					PRIMARY KEY (reward_track_path, content_hash)
				);
				CREATE INDEX IF NOT EXISTS idx_battlepass_track_definitions_lookup ON battlepass_track_definitions(reward_track_path, is_current);
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
				CREATE INDEX IF NOT EXISTS idx_battlepass_item_definitions_lookup ON battlepass_item_definitions(inventory_item_path, is_current);
				CREATE TABLE IF NOT EXISTS battlepass_item_translations (
					inventory_item_path VARCHAR NOT NULL, content_hash VARCHAR NOT NULL,
					lang VARCHAR NOT NULL, title VARCHAR, description VARCHAR,
					first_seen_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP, last_seen_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
					PRIMARY KEY (inventory_item_path, content_hash, lang)
				);
				CREATE INDEX IF NOT EXISTS idx_battlepass_item_translations_lookup ON battlepass_item_translations(inventory_item_path, lang);
			`)
		},
	})

	Register(Migration{
		Name:        "add_challenge_metadata",
		TargetDB:    TargetMetadata,
		Description: "Tables challenge_definitions + challenge_translations",
		ApplySchema: func(db *sql.DB) error {
			return execScript(db, `
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
	})

	Register(Migration{
		Name:        "add_medal_definitions",
		TargetDB:    TargetMetadata,
		Description: "Table medal_definitions (medal_name_id BIGINT, noms, descriptions)",
		ApplySchema: func(db *sql.DB) error {
			return execScript(db, `
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
	})

	Register(Migration{
		Name:        "add_weapon_labels",
		TargetDB:    TargetMetadata,
		Description: "Table weapon_labels (weapon_id UBIGINT, name_en, name_fr)",
		ApplySchema: applyWeaponLabels,
	})

	Register(Migration{
		Name:        "drop_legacy_translation_tables",
		TargetDB:    TargetMetadata,
		Description: "DROP mode_translations + playlist_translations",
		ApplySchema: func(db *sql.DB) error {
			return execScript(db, `
				DROP TABLE IF EXISTS mode_translations;
				DROP TABLE IF EXISTS playlist_translations;
			`)
		},
	})

	Register(Migration{
		Name:        "add_waypoint_assets_raw",
		TargetDB:    TargetMetadata,
		Description: "Table waypoint_assets_raw : cache générique de blobs JSON Waypoint",
		ApplySchema: func(db *sql.DB) error {
			return execScript(db, `
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
	})

	Register(Migration{
		Name:        "add_map_images_registry",
		TargetDB:    TargetMetadata,
		Description: "Table map_images_registry : cache-aside des images de maps avec local_path optionnel",
		ApplySchema: func(db *sql.DB) error {
			return execScript(db, `
				CREATE TABLE IF NOT EXISTS map_images_registry (
					title_id     VARCHAR NOT NULL,
					map_id       VARCHAR NOT NULL,
					image_url    VARCHAR NOT NULL DEFAULT '',
					local_path   VARCHAR NOT NULL DEFAULT '',
					fetched_at   TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT now(),
					content_hash VARCHAR NOT NULL DEFAULT '',
					PRIMARY KEY (title_id, map_id)
				);
				CREATE INDEX IF NOT EXISTS idx_map_images_registry_fetched ON map_images_registry(fetched_at);
			`)
		},
	})

	Register(Migration{
		Name:        "add_mode_name_tr",
		TargetDB:    TargetMetadata,
		Description: "Table mode_name_tr : traductions des modes de jeu (FR/EN), portage depuis metadata-prebuilt",
		ApplySchema: applyModeNameTr,
	})

	Register(Migration{
		Name:        "add_citation_mappings",
		TargetDB:    TargetMetadata,
		Description: "Table citation_mappings : mappings médaille→citation avec paliers, images et flags (portage populate_citation_mappings.py)",
		ApplySchema: func(db *sql.DB) error {
			return execScript(db, `
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
				CREATE INDEX IF NOT EXISTS idx_citation_mappings_medal ON citation_mappings(medal_id);
			`)
		},
	})

	Register(Migration{
		Name:        "add_citation_mappings_pk",
		TargetDB:    TargetMetadata,
		Description: "Ajout PRIMARY KEY (citation_name_norm, medal_id) sur citation_mappings (nécessaire pour ON CONFLICT DO NOTHING)",
		ApplySchema: func(db *sql.DB) error {
			// DuckDB ne supporte pas ALTER TABLE ADD CONSTRAINT PK.
			// On recrée la table avec déduplication. La PK est sur citation_name_norm
			// uniquement : medal_id peut être NULL (citations non liées à une médaille
			// spécifique), ce qui interdit de l'inclure dans une PRIMARY KEY.
			return execScript(db, `
				CREATE TABLE IF NOT EXISTS citation_mappings_v2 AS
					SELECT DISTINCT ON (citation_name_norm)
						citation_name_norm, citation_name_display, mapping_type,
						category, image_path, description, tier_targets, medal_id, enabled
					FROM citation_mappings;
				DROP TABLE IF EXISTS citation_mappings;
				ALTER TABLE citation_mappings_v2 RENAME TO citation_mappings;
				ALTER TABLE citation_mappings ADD PRIMARY KEY (citation_name_norm);
				CREATE INDEX IF NOT EXISTS idx_citation_mappings_norm ON citation_mappings(citation_name_norm);
				CREATE INDEX IF NOT EXISTS idx_citation_mappings_medal ON citation_mappings(medal_id);
			`)
		},
	})

	Register(Migration{
		Name:        "add_citation_mappings_v2_fields",
		TargetDB:    TargetMetadata,
		Description: "citation_mappings : ajout medal_ids/stat_name/award_name/award_category/custom_function/composite_children/subcategory pour le moteur complet (parité avec scripts/populate_citation_mappings.py main)",
		ApplySchema: func(db *sql.DB) error {
			return execScript(db, `
				ALTER TABLE citation_mappings ADD COLUMN IF NOT EXISTS medal_ids          VARCHAR;
				ALTER TABLE citation_mappings ADD COLUMN IF NOT EXISTS stat_name          VARCHAR;
				ALTER TABLE citation_mappings ADD COLUMN IF NOT EXISTS award_name         VARCHAR;
				ALTER TABLE citation_mappings ADD COLUMN IF NOT EXISTS award_category     VARCHAR;
				ALTER TABLE citation_mappings ADD COLUMN IF NOT EXISTS custom_function    VARCHAR;
				ALTER TABLE citation_mappings ADD COLUMN IF NOT EXISTS composite_children VARCHAR;
				ALTER TABLE citation_mappings ADD COLUMN IF NOT EXISTS subcategory        VARCHAR;
				CREATE INDEX IF NOT EXISTS idx_citation_mappings_type ON citation_mappings(mapping_type);
			`)
		},
	})

	Register(Migration{
		Name:        "add_xbox_achievement_definitions",
		TargetDB:    TargetMetadata,
		Description: "Table xbox_achievement_definitions : référentiel achievements Halo Infinite (bilingue EN/FR)",
		ApplySchema: func(db *sql.DB) error {
			return execScript(db, `
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
	})

	Register(Migration{
		Name:        "add_career_rank_translations",
		TargetDB:    TargetMetadata,
		Description: "Table career_rank_translations : libellés rangs de carrière dans toutes les langues exposées par GameCMS",
		ApplySchema: func(db *sql.DB) error {
			return execScript(db, `
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
	})

	Register(Migration{
		Name:        "medal_definitions_add_indices",
		TargetDB:    TargetMetadata,
		Description: "medal_definitions : ajout difficulty_index + type_index (entiers Waypoint) — idempotent via IF NOT EXISTS",
		ApplySchema: func(db *sql.DB) error {
			return execScript(db, `
				ALTER TABLE medal_definitions ADD COLUMN IF NOT EXISTS difficulty_index TINYINT DEFAULT 0;
				ALTER TABLE medal_definitions ADD COLUMN IF NOT EXISTS type_index TINYINT DEFAULT 0;
			`)
		},
	})

	Register(Migration{
		Name:        "enrich_medal_definitions_v2",
		TargetDB:    TargetMetadata,
		Description: "medal_definitions : ajout difficulty (Normal/Heroic/Legendary/Mythic) + medal_type (multikill/spree/…) depuis difficulty_index/type_index existants",
		ApplySchema: func(db *sql.DB) error {
			return execScript(db, `
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
	})

	Register(Migration{
		Name:        "medal_definitions_add_personal_score",
		TargetDB:    TargetMetadata,
		Description: "medal_definitions : ajout personal_score (XP de carrière accordé par médaille, 0 par défaut)",
		ApplySchema: func(db *sql.DB) error {
			return execScript(db, `
				ALTER TABLE medal_definitions ADD COLUMN IF NOT EXISTS personal_score INTEGER DEFAULT 0;
			`)
		},
	})

	Register(Migration{
		Name:        "fix_super_fiesta_fr_label",
		TargetDB:    TargetMetadata,
		Description: "asset_translations : retire le point parasite de 'Méga fiesta.' (playlist Super Fiesta, lang fr/fr-FR)",
		ApplySchema: func(db *sql.DB) error {
			return execScript(db, `
				UPDATE asset_translations
				SET name = 'Méga fiesta'
				WHERE asset_type = 'playlist'
				  AND asset_id = '4829f027-a9af-4b2f-86dd-7b290d6bb0a4'
				  AND lang IN ('fr', 'fr-FR')
				  AND name = 'Méga fiesta.';
			`)
		},
	})

	Register(Migration{
		Name:        "seed_playlist_fr_translations",
		TargetDB:    TargetMetadata,
		Description: "asset_translations : seed FR canoniques pour playlists Halo Infinite dont l'API a renvoyé l'EN raw en lang fr-FR (cf. thought_log 2026-05-09)",
		ApplySchema: applyPlaylistFRSeeds,
	})

	Register(Migration{
		Name:        "add_title_id_to_xbox_achievement_definitions",
		TargetDB:    TargetMetadata,
		Description: "Colonne title_id sur xbox_achievement_definitions (filtre par jeu — halo_infinite) pour exclure les succès d'autres titres Xbox stockés avant l'introduction du filtre titleId.",
		ApplySchema: func(db *sql.DB) error {
			_, err := db.Exec(`ALTER TABLE xbox_achievement_definitions ADD COLUMN IF NOT EXISTS title_id VARCHAR DEFAULT ''`)
			return err
		},
	})

	Register(Migration{
		Name:        "cleanup_xbox_achievement_definitions_unknown_title",
		TargetDB:    TargetMetadata,
		Description: "Supprime les succès Xbox sans title_id connu (insertés avant le filtre halo_infinite). L'utilisateur doit relancer sync-achievements après cette migration.",
		ApplySchema: func(db *sql.DB) error {
			_, err := db.Exec(`DELETE FROM xbox_achievement_definitions WHERE title_id = '' OR title_id IS NULL`)
			return err
		},
	})

	Register(Migration{
		Name:        "add_xbox_title_id_to_xbox_achievement_definitions",
		TargetDB:    TargetMetadata,
		Description: "Colonne xbox_title_id sur xbox_achievement_definitions : identifiant Xbox numérique du titre source (ex: '1144039928' pour Halo Infinite). Peuplée lors du prochain sync-achievements. Permet au frontend de filtrer les succès cross-titres sans DELETE.",
		ApplySchema: func(db *sql.DB) error {
			_, err := db.Exec(`ALTER TABLE xbox_achievement_definitions ADD COLUMN IF NOT EXISTS xbox_title_id VARCHAR DEFAULT ''`)
			return err
		},
	})

	Register(Migration{
		Name:        "add_service_config_id_to_xbox_achievement_definitions",
		TargetDB:    TargetMetadata,
		Description: "Colonne service_config_id (SCID Xbox) sur xbox_achievement_definitions. Le SCID est le seul discriminateur fiable par jeu : l'API Xbox retourne tous les achievements de la franchise quand on filtre par titleId. Peuplée lors du prochain sync-achievements.",
		ApplySchema: func(db *sql.DB) error {
			_, err := db.Exec(`ALTER TABLE xbox_achievement_definitions ADD COLUMN IF NOT EXISTS service_config_id VARCHAR DEFAULT ''`)
			return err
		},
	})
}

// applyModeNameTr crée et peuple mode_name_tr avec les traductions connues.
func applyModeNameTr(db *sql.DB) error {
	if _, err := db.Exec(`
		CREATE TABLE IF NOT EXISTS mode_name_tr (
			mode_en VARCHAR NOT NULL,
			lang    VARCHAR NOT NULL,
			name    VARCHAR NOT NULL,
			PRIMARY KEY (mode_en, lang)
		)
	`); err != nil {
		return err
	}

	type modeRow struct{ modeEN, lang, name string }
	rows := []modeRow{
		{"Assault", "en", "Assault"},
		{"Attrition", "en", "Attrition"},
		{"CTF", "en", "CTF"},
		{"CTF 3 Captures", "en", "CTF (3 Captures)"},
		{"Escalation Slayer", "en", "Escalation Slayer"},
		{"Extraction", "en", "Extraction"},
		{"FFA Slayer", "en", "FFA Slayer"},
		{"Fiesta CTF", "en", "Fiesta CTF"},
		{"Fiesta Slayer", "en", "Fiesta Slayer"},
		{"Fiesta Total Control", "en", "Fiesta Total Control"},
		{"Heroic KOTH", "en", "King of the Hill (Heroic)"},
		{"Heroic King of the Hill", "en", "King of the Hill (Heroic)"},
		{"King of the Hill", "en", "King of the Hill"},
		{"Land Grab", "en", "Land Grab"},
		{"Legendary King of the Hill", "en", "King of the Hill (Legendary)"},
		{"Neutral Bomb", "en", "Neutral Bomb"},
		{"Neutral Bomb Squad", "en", "Neutral Bomb Squad"},
		{"Neutral Flag CTF", "en", "Neutral Flag CTF"},
		{"Oddball", "en", "Oddball"},
		{"One Bomb", "en", "One Bomb"},
		{"One Flag CTF", "en", "One Flag CTF"},
		{"Sentry Defense", "en", "Sentry Defense"},
		{"Shotty Snipe Slayer FFA", "en", "Shotty Snipers FFA"},
		{"Shotty Snipes Slayer", "en", "Shotty Snipers"},
		{"Slayer", "en", "Slayer"},
		{"Stockpile", "en", "Stockpile"},
		{"Strongholds", "en", "Strongholds"},
		{"Team Slayer", "en", "Team Slayer"},
		{"Team Snipers", "en", "Team Snipers"},
		{"Total Control", "en", "Total Control"},
		{"VIP", "en", "VIP"},
		// FR
		{"Assault", "fr", "Assaut"},
		{"Attrition", "fr", "Attrition"},
		{"CTF", "fr", "Capture du drapeau"},
		{"CTF 3 Captures", "fr", "CDD 3 captures"},
		{"Escalation Slayer", "fr", "Escalade"},
		{"Extraction", "fr", "Extraction"},
		{"FFA Slayer", "fr", "Chacun pour soi"},
		{"Fiesta CTF", "fr", "Fiesta CDD"},
		{"Fiesta Slayer", "fr", "Fiesta"},
		{"Fiesta Total Control", "fr", "Fiesta Contrôle total"},
		{"Heroic KOTH", "fr", "Roi de la colline héroïque"},
		{"Heroic King of the Hill", "fr", "Roi de la colline héroïque"},
		{"King of the Hill", "fr", "Roi de la colline"},
		{"Land Grab", "fr", "Bases"},
		{"Legendary King of the Hill", "fr", "Roi de la colline légendaire"},
		{"Neutral Bomb", "fr", "Bombe neutre"},
		{"Neutral Bomb Squad", "fr", "Escouade bombe neutre"},
		{"Neutral Flag CTF", "fr", "Drapeau neutre"},
		{"Oddball", "fr", "Oddball"},
		{"One Bomb", "fr", "Bombe neutre"},
		{"One Flag CTF", "fr", "Drapeau neutre"},
		{"Sentry Defense", "fr", "Défense sentinelle"},
		{"Shotty Snipe Slayer FFA", "fr", "Fusils snipers à grenaille FFA"},
		{"Shotty Snipes Slayer", "fr", "Fusils snipers à grenaille"},
		{"Slayer", "fr", "Assassin"},
		{"Stockpile", "fr", "Stockage"},
		{"Strongholds", "fr", "Bases"},
		{"Team Slayer", "fr", "Assassin en équipe"},
		{"Team Snipers", "fr", "Snipers en équipe"},
		{"Total Control", "fr", "Contrôle total"},
		{"VIP", "fr", "VIP"},
	}

	for _, r := range rows {
		if _, err := db.Exec(
			"INSERT OR IGNORE INTO mode_name_tr (mode_en, lang, name) VALUES (?, ?, ?)",
			r.modeEN, r.lang, r.name,
		); err != nil {
			return err
		}
	}
	return nil
}

// ApplyWeaponLabels expose applyWeaponLabels pour les outils CLI de reseed
// (cf. cmd/seed-weapon-labels). Idempotent via INSERT OR IGNORE — peut etre
// appele meme si schema_migrations marque la migration comme done.
func ApplyWeaponLabels(db *sql.DB) error {
	return applyWeaponLabels(db)
}

// applyWeaponLabels crée et peuple weapon_labels avec tous les IDs connus.
func applyWeaponLabels(db *sql.DB) error {
	if _, err := db.Exec(`
		CREATE TABLE IF NOT EXISTS weapon_labels (
			weapon_id UBIGINT PRIMARY KEY,
			name_en   VARCHAR NOT NULL,
			name_fr   VARCHAR NOT NULL
		)
	`); err != nil {
		return err
	}

	// Sentinels + confirmed weapons (portage de WEAPON_INT_TO_NAME + WEAPON_NAME_FR)
	//
	// Contrainte driver : database/sql ne supporte pas uint64 avec bit63=1.
	// Contournement : injecter weapon_id comme littéral décimal (valeur constante),
	// les noms restent paramétrisés.
	type label struct {
		id uint64
		en string
		fr string
	}
	labels := []label{
		{0, "Grenade", "Grenade"},
		{1, "Melee", "Corps à corps"},
		{2, "Vehicle", "Véhicule"},
		{0x6acdc44d42c9679f, "Bandit Evo", "Bandit EVO"},
		{0x2b1824d542c9679f, "BR75", "BR75"},
		{0x230447b142c9679f, "Cindershot", "Crémateur"},
		{0xb619d84a42c9679f, "CQS48 Bulldog", "CQS48 Bulldog"},
		{0x84bd29ed42c9679f, "Disruptor", "Disrupteur"},
		{0x9d6aaed242c9679f, "Fuel Rod SPNKr", "M41 SPNKr"},
		{0x841ac5e542c9679f, "Gravity Hammer", "Marteau antigravité"},
		{0x2ac9c2ff42c9679f, "Heatwave", "Calcineur"},
		{0x71ab0a2c42c9679f, "M41 SPNKr", "M41 SPNKr"},
		{0x2fb21c8742c9679f, "M392 Bandit", "Bandit EVO"},
		{0x48c19d2d42c9679f, "MA40 AR", "MA40 AR"},
		{0xf5c335dfe7232c0f, "MA5K Avenger", "MA5K Avenger"},
		{0x80977ba542c9679f, "Mangler", "Déchiqueteur"},
		{0x767db96d42c9679f, "MLRS-2 Hydra", "Hydra"},
		{0xf408190f42c9679f, "Mk51 Sidekick", "MK50 Sidekick"},
		{0xd791556542c9679f, "Mutilator", "Mutilateur"},
		{0xb7262ca1c8fb11d0, "Mythic Sandwich", "Mythic Sandwich"},
		{0xb533957e42c9679f, "Needler", "Needler"},
		{0xc354294642c9679f, "Plasma Pistol", "Pistolet à plasma"},
		{0x30484ea642c9679f, "Pulse Carbine", "Carabine à impulsion"},
		{0xc30d87c742c9679f, "Ravager", "Ravageur"},
		{0x0a1992bc42c9679f, "S7 Sniper", "S7 Sniper"},
		{0x880fe0bc42c9679f, "Sandwich", "Sandwich"},
		{0xa0955e9e42c9679f, "Sentinel Beam", "Rayon de Sentinelle"},
		{0x9387a8b942c9679f, "Shock Rifle", "Fusil électrique"},
		{0x1a22fee642c9679f, "Shock Rifle (Ranked)", "Fusil électrique"},
		{0x0d20c46942c9679f, "Skewer", "Empaleur"},
		{0xdaf193c742c9679f, "Stalker Rifle", "Fusil traqueur"},
		{0x3e07021742c9679f, "Vestige Carbine", "Carabine Vestige"},
		{0xfd98554c42c9679f, "VK78 Commando", "VK78 Commando"},
		{0x4ff3937e42c9679f, "Energy Sword", "Épée à énergie"},
		{0x4ff3937e8978aa7a, "Duelist Energy Sword", "Épée à énergie"},
		{0x4ff3937e1ec48c7a, "Elite Bloodblade", "Épée à énergie"},
		{0x0c55765f7a9376a0, "Infected Energy Sword", "Épée à énergie"},
		{0x841ac5e5a730e49f, "Diminisher of Hope", "Marteau antigravité"},
		{0x841ac5e5d8d07ca1, "Rushdown Hammer", "Marteau antigravité"},
		{0xb6dbead842c9679f, "Frag Grenade", "Grenade frag"},
		{0xc1e1bab042c9679f, "Plasma Grenade", "Grenade plasma"},
		{0x3ad55da442c9679f, "Dynamo Grenade", "Grenade dynamo"},
	}

	for _, l := range labels {
		// Contournement driver : database/sql ne supporte pas uint64 avec bit63=1.
		// weapon_id est une constante interne (pas user input) → littéral décimal sûr.
		q := fmt.Sprintf( //nolint:gosec
			"INSERT OR IGNORE INTO weapon_labels (weapon_id, name_en, name_fr) VALUES (%d, ?, ?)",
			l.id,
		)
		if _, err := db.Exec(q, l.en, l.fr); err != nil {
			return err
		}
	}
	return nil
}
