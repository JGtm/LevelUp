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
		Name:     "add_asset_translations",
		TargetDB: TargetMetadata,
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
		Name:     "add_battlepass_asset_refs",
		TargetDB: TargetMetadata,
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
		Name:     "add_battlepass_metadata",
		TargetDB: TargetMetadata,
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
		Name:     "add_challenge_metadata",
		TargetDB: TargetMetadata,
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
		id   uint64
		en   string
		fr   string
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
