package migration

// steps_world_csr_leaderboard_latest_by_batch.go — durcissement de la vue
// world_csr_leaderboard_latest.
//
// Problème de la v1 (création initiale) : la vue prenait le dernier snapshot
// PAR RANG (PARTITION BY season_id, playlist_id, rank). Si un nouveau snapshot
// était plus court que le précédent (page tronquée, fin de saison, glitch), la
// vue affichait un classement « Frankenstein » : les rangs du nouveau snapshot +
// la queue (rangs supérieurs) de l'ANCIEN snapshot, jamais réécrits.
//
// Fix : grouper par BATCH de scrape via `fetched_at` (fixé une seule fois par le
// scraper pour toutes les entrées d'une même capture de playlist). La vue ne
// retient que les lignes du dernier `fetched_at` pour chaque (season, playlist)
// → exactement un snapshot cohérent, jamais un mélange. Combiné au plancher de
// cohérence du cron + à l'insert atomique, garantit qu'on n'affiche jamais un
// classement partiel.
//
// Compatible avec les données existantes (per-row ou atomiques) : `fetched_at`
// est porté par les lignes depuis l'origine, indépendant du mode d'insertion.

import "database/sql"

func init() {
	Register(Migration{
		Name:        "world_csr_leaderboard_latest_by_batch",
		TargetDB:    TargetShared,
		Description: "Remplace world_csr_leaderboard_latest : dernier batch (fetched_at) par (season, playlist) au lieu du dernier par rang (fix snapshot Frankenstein)",
		ApplySchema: func(db *sql.DB) error {
			return execScript(db, `
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
	})
}
