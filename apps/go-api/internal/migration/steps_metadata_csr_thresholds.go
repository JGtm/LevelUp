package migration

// steps_metadata_csr_thresholds.go — Phase 5 du plan pipeline CSR.
//
// Table de mapping season_id → nombre de matchs de placement requis. Halo Infinite
// a baissé le seuil de 10 → 5 à partir de Season 3 (2023-03-07). Cette table
// permet au backend display de calculer le bon "X/N" en fonction de la saison
// du match/snapshot consulté.
//
// Source dates : config/titles/halo_infinite/mappings/assets.toml (seasons catalog).

import "database/sql"

func init() {
	Register(Migration{
		Name:        "add_csr_placement_thresholds",
		TargetDB:    TargetMetadata,
		Description: "Table csr_placement_thresholds (mapping season_id → seuil placement, 10 pré-S3 / 5 depuis S3)",
		ApplySchema: func(db *sql.DB) error {
			return execScript(db, `
				CREATE TABLE IF NOT EXISTS csr_placement_thresholds (
					season_id  VARCHAR PRIMARY KEY,
					threshold  INTEGER NOT NULL CHECK (threshold > 0 AND threshold <= 20),
					valid_from DATE,
					notes      VARCHAR
				);

				-- Seed initial : 13 saisons connues (S1-S13).
				-- INSERT OR REPLACE pour rester idempotent (peut tourner à chaque boot
				-- sans dupliquer ni écraser les éventuelles surcharges manuelles
				-- ajoutées par admin).
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
	})
}
