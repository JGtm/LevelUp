package migration

// steps_metadata.go — migrations ciblant metadata.duckdb.
// Portage de src/data/migration/steps/add_asset_translations.py, add_battlepass_*.py,
// add_challenge_metadata.py, add_medal_definitions.py, add_weapon_labels.py,
// drop_legacy_translation_tables.py.

import (
	"database/sql"
)

// Mode canoniques (EN) — utilisés comme clés dans mode_name_tr et comme
// labels FR identiques (Halo n'a pas de traduction officielle pour ces modes).
const (
	modeAttrition  = "Attrition"
	modeExtraction = "Extraction"
	modeOddball    = "Oddball"
)

// Labels mode (cross-fichier metadata + playlist_fr).
const (
	modeTeamSlayer    = "Team Slayer"
	modeTeamSnipers   = "Team Snipers"
	modeTeamSlayerFR  = "Assassin en équipe"
	modeTeamSnipersFR = "Snipers en équipe"
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

	// add_battlepass_asset_refs / add_battlepass_metadata / add_challenge_metadata
	// → migrés vers games/halo_infinite/migrations (b5). Noms gardés dans canonicalOrder.

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

	// add_weapon_labels → migré vers games/halo_infinite/migrations/weapon_labels.go (b6).

	// drop_legacy_translation_tables → migré vers games/halo_infinite/migrations (b5).

	// add_waypoint_assets_raw → migré vers internal/games/halo_infinite/migrations/
	// steps.go (Phase 1.5 voie B). Le nom reste dans canonicalOrder.

	// add_map_images_registry → migré vers internal/games/halo_infinite/migrations/
	// steps.go (Phase 1.5 voie B, ADR 0025). Le nom reste dans canonicalOrder.

	Register(Migration{
		Name:        "add_mode_name_tr",
		TargetDB:    TargetMetadata,
		Description: "Table mode_name_tr : traductions des modes de jeu (FR/EN), portage depuis metadata-prebuilt",
		ApplySchema: applyModeNameTr,
	})

	// Famille citation_mappings (base→pk→v2_fields) → migrée ATOMIQUEMENT vers
	// games/halo_infinite/migrations (b5). Noms gardés dans canonicalOrder.

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

	// add_career_rank_translations → migré vers games/halo_infinite/migrations (b5).

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
			_, err := db.ExecContext(bootCtx(), `ALTER TABLE xbox_achievement_definitions ADD COLUMN IF NOT EXISTS title_id VARCHAR DEFAULT ''`)
			return err
		},
	})

	Register(Migration{
		Name:        "cleanup_xbox_achievement_definitions_unknown_title",
		TargetDB:    TargetMetadata,
		Description: "Supprime les succès Xbox sans title_id connu (insertés avant le filtre halo_infinite). L'utilisateur doit relancer sync-achievements après cette migration.",
		ApplySchema: func(db *sql.DB) error {
			_, err := db.ExecContext(bootCtx(), `DELETE FROM xbox_achievement_definitions WHERE title_id = '' OR title_id IS NULL`)
			return err
		},
	})

	Register(Migration{
		Name:        "add_xbox_title_id_to_xbox_achievement_definitions",
		TargetDB:    TargetMetadata,
		Description: "Colonne xbox_title_id sur xbox_achievement_definitions : identifiant Xbox numérique du titre source (ex: '1144039928' pour Halo Infinite). Peuplée lors du prochain sync-achievements. Permet au frontend de filtrer les succès cross-titres sans DELETE.",
		ApplySchema: func(db *sql.DB) error {
			_, err := db.ExecContext(bootCtx(), `ALTER TABLE xbox_achievement_definitions ADD COLUMN IF NOT EXISTS xbox_title_id VARCHAR DEFAULT ''`)
			return err
		},
	})

	Register(Migration{
		Name:        "add_service_config_id_to_xbox_achievement_definitions",
		TargetDB:    TargetMetadata,
		Description: "Colonne service_config_id (SCID Xbox) sur xbox_achievement_definitions. Le SCID est le seul discriminateur fiable par jeu : l'API Xbox retourne tous les achievements de la franchise quand on filtre par titleId. Peuplée lors du prochain sync-achievements.",
		ApplySchema: func(db *sql.DB) error {
			_, err := db.ExecContext(bootCtx(), `ALTER TABLE xbox_achievement_definitions ADD COLUMN IF NOT EXISTS service_config_id VARCHAR DEFAULT ''`)
			return err
		},
	})
}

// applyModeNameTr crée et peuple mode_name_tr avec les traductions connues.
func applyModeNameTr(db *sql.DB) error {
	if _, err := db.ExecContext(bootCtx(), `
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
		{modeAttrition, "en", modeAttrition},
		{"CTF", "en", "CTF"},
		{"CTF 3 Captures", "en", "CTF (3 Captures)"},
		{"Escalation Slayer", "en", "Escalation Slayer"},
		{modeExtraction, "en", modeExtraction},
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
		{modeOddball, "en", modeOddball},
		{"One Bomb", "en", "One Bomb"},
		{"One Flag CTF", "en", "One Flag CTF"},
		{"Sentry Defense", "en", "Sentry Defense"},
		{"Shotty Snipe Slayer FFA", "en", "Shotty Snipers FFA"},
		{"Shotty Snipes Slayer", "en", "Shotty Snipers"},
		{"Slayer", "en", "Slayer"},
		{"Stockpile", "en", "Stockpile"},
		{"Strongholds", "en", "Strongholds"},
		{modeTeamSlayer, "en", modeTeamSlayer},
		{modeTeamSnipers, "en", modeTeamSnipers},
		{"Total Control", "en", "Total Control"},
		{"VIP", "en", "VIP"},
		// FR
		{"Assault", "fr", "Assaut"},
		{modeAttrition, "fr", modeAttrition},
		{"CTF", "fr", "Capture du drapeau"},
		{"CTF 3 Captures", "fr", "CDD 3 captures"},
		{"Escalation Slayer", "fr", "Escalade"},
		{modeExtraction, "fr", modeExtraction},
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
		{modeOddball, "fr", modeOddball},
		{"One Bomb", "fr", "Bombe neutre"},
		{"One Flag CTF", "fr", "Drapeau neutre"},
		{"Sentry Defense", "fr", "Défense sentinelle"},
		{"Shotty Snipe Slayer FFA", "fr", "Fusils snipers à grenaille FFA"},
		{"Shotty Snipes Slayer", "fr", "Fusils snipers à grenaille"},
		{"Slayer", "fr", "Assassin"},
		{"Stockpile", "fr", "Stockage"},
		{"Strongholds", "fr", "Bases"},
		{modeTeamSlayer, "fr", modeTeamSlayerFR},
		{modeTeamSnipers, "fr", modeTeamSnipersFR},
		{"Total Control", "fr", "Contrôle total"},
		{"VIP", "fr", "VIP"},
	}

	for _, r := range rows {
		if _, err := db.ExecContext(bootCtx(),
			"INSERT OR IGNORE INTO mode_name_tr (mode_en, lang, name) VALUES (?, ?, ?)",
			r.modeEN, r.lang, r.name,
		); err != nil {
			return err
		}
	}
	return nil
}

// applyWeaponLabels / ApplyWeaponLabels + labelEnergySwordFR → migrés vers
// games/halo_infinite/migrations/weapon_labels.go (Phase 1.5 b6, voie B).
