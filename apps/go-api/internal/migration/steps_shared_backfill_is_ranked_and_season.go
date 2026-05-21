package migration

// steps_shared_backfill_is_ranked_and_season.go — Phase 1 du plan pipeline CSR.
//
// Régression connue (2026-05-10) : tous les match_registry.is_ranked sont à
// FALSE alors que des playlists "Ranked Arena" / "Ranked Slayer" existent. Le
// sync écrit FALSE par défaut à cause d'un bug dans isRankedPlaylist ou parce
// que les rows sont antérieures à l'activation du calcul.
//
// Cette migration :
//   1. Ajoute la colonne match_registry.season_id (VARCHAR nullable).
//   2. ApplyBackfill : UPDATE is_ranked=TRUE pour les rows dont playlist_name
//      ou pair_name contient "ranked" (case-insensitive).
//   3. UPDATE season_id pour les rows ranked, en dérivant depuis start_time
//      via les bornes du seasons catalog (sync avec
//      config/titles/halo_infinite/mappings/assets.toml et
//      metadata.csr_placement_thresholds).
//
// Idempotente : ALTER ... IF NOT EXISTS + UPDATE conditionnels (WHERE
// is_ranked=FALSE et WHERE season_id IS NULL). Le framework de migration
// trace l'application via schema_migrations.backfill_done — un second run
// sera no-op.

import "database/sql"

func init() {
	Register(Migration{
		Name:        "shared_backfill_is_ranked_and_season",
		TargetDB:    TargetShared,
		Description: "Phase 1 plan CSR : ALTER +season_id, backfill is_ranked + season_id via heuristique nom playlist + bornes saison",
		ApplySchema: func(db *sql.DB) error {
			return execScript(db, `
				ALTER TABLE match_registry ADD COLUMN IF NOT EXISTS season_id VARCHAR;
				CREATE INDEX IF NOT EXISTS idx_match_registry_season_id ON match_registry(season_id);
			`)
		},
		ApplyBackfill: func(db *sql.DB) error {
			// Étape 1 : marquer ranked les rows dont le nom playlist ou pair contient "ranked".
			// (Halo retourne les noms en EN dans le payload — testé : 0 occurrence de FR
			//  "classé"/"classée" dans la base ; l'heuristique EN couvre 100% des cas.)
			if _, err := db.Exec(`
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

			// Étape 2 : dériver season_id depuis start_time via les bornes officielles
			// du seasons catalog Halo Infinite. Sync avec :
			//   - config/titles/halo_infinite/mappings/assets.toml
			//   - metadata.csr_placement_thresholds (Phase 5)
			// Évalué uniquement pour les rows ranked sans season_id (premier run + cas
			// de drift). Idempotent : 2e run = 0 rows mises à jour.
			_, err := db.Exec(`
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
	})
}
