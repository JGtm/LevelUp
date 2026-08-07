package migrations

// steps_metadata_root.go — RACINE metadata (asset_translations), déplacée depuis
// internal/migration/steps_metadata.go (Phase 1.5 b26, voie B — DERNIER root metadata).
//
// add_asset_translations crée asset_translations + medal_translations (pivot multi-langue) :
// table écrite par de nombreux seeds (seed_ranked_playlists_catalog, playlist FR, mode_name_tr,
// ReconcileMetadataSeeds…) tous déjà title-owned → la racine peut partir. fix_super_fiesta_fr_label
// (UPDATE asset_translations) suit, atomique. Plus de hazard de cycle : applyModeNameTr /
// ReconcileMetadataSeeds ont été déplacés au titre en b7.

import (
	"database/sql"

	"levelup/go-api/internal/migration"
)

// metadataRootSteps retourne la racine metadata title-owned (b26).
func metadataRootSteps() []migration.Migration {
	return []migration.Migration{
		{
			Name:        "add_asset_translations",
			TargetDB:    migration.TargetMetadata,
			Description: "Tables asset_translations + medal_translations (pivot multi-langue)",
			ApplySchema: func(db *sql.DB) error {
				return migration.ExecScript(db, `
					CREATE TABLE IF NOT EXISTS asset_translations (
						asset_id    VARCHAR NOT NULL,
						asset_type  VARCHAR NOT NULL,
						lang        VARCHAR NOT NULL,
						name        VARCHAR NOT NULL,
						description VARCHAR,
						fetched_at  TIMESTAMP DEFAULT CAST(now() AT TIME ZONE 'UTC' AS TIMESTAMP),
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
		},
		{
			Name:        "fix_super_fiesta_fr_label",
			TargetDB:    migration.TargetMetadata,
			Description: "asset_translations : retire le point parasite de 'Méga fiesta.' (playlist Super Fiesta, lang fr/fr-FR)",
			ApplySchema: func(db *sql.DB) error {
				return migration.ExecScript(db, `
					UPDATE asset_translations
					SET name = 'Méga fiesta'
					WHERE asset_type = 'playlist'
					  AND asset_id = '4829f027-a9af-4b2f-86dd-7b290d6bb0a4'
					  AND lang IN ('fr', 'fr-FR')
					  AND name = 'Méga fiesta.';
				`)
			},
		},
	}
}
