package migration

// steps_metadata.go — migrations ciblant metadata.duckdb.
// Portage de src/data/migration/steps/add_asset_translations.py, add_battlepass_*.py,
// add_challenge_metadata.py, add_medal_definitions.py, add_weapon_labels.py,
// drop_legacy_translation_tables.py.

import (
	"database/sql"
)

// consts mode* → déplacées avec applyModeNameTr/applyPlaylistFRSeeds vers
// games/halo_infinite/migrations/mode_playlist_fr.go (Phase 1.5 b7).

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

	// Famille medal_definitions (base + indices/enrich/personal_score) → migrée
	// ATOMIQUEMENT vers games/halo_infinite/migrations (b8).

	// add_weapon_labels → migré vers games/halo_infinite/migrations/weapon_labels.go (b6).

	// drop_legacy_translation_tables → migré vers games/halo_infinite/migrations (b5).

	// add_waypoint_assets_raw → migré vers internal/games/halo_infinite/migrations/
	// steps.go (Phase 1.5 voie B). Le nom reste dans canonicalOrder.

	// add_map_images_registry → migré vers internal/games/halo_infinite/migrations/
	// steps.go (Phase 1.5 voie B, ADR 0025). Le nom reste dans canonicalOrder.

	// add_mode_name_tr → migré vers games/halo_infinite/migrations/mode_playlist_fr.go (b7).

	// Famille citation_mappings (base→pk→v2_fields) → migrée ATOMIQUEMENT vers
	// games/halo_infinite/migrations (b5). Noms gardés dans canonicalOrder.

	// Famille xbox_achievement_definitions (base + 4 ALTER/DELETE) → migrée
	// ATOMIQUEMENT vers games/halo_infinite/migrations (b8).

	// add_career_rank_translations → migré vers games/halo_infinite/migrations (b5).

	// medal_definitions_add_indices / enrich_medal_definitions_v2 /
	// medal_definitions_add_personal_score → migrés avec la famille medal (b8).

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

	// seed_playlist_fr_translations → migré vers games/halo_infinite/migrations/mode_playlist_fr.go (b7).

	// add_title_id / cleanup / add_xbox_title_id / add_service_config_id (xbox
	// achievement) → migrés avec la famille xbox_achievement (b8).
}

// applyWeaponLabels / ApplyWeaponLabels + labelEnergySwordFR → migrés vers
// games/halo_infinite/migrations/weapon_labels.go (Phase 1.5 b6, voie B).
