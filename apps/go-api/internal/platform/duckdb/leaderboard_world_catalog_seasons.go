// Package duckdb — leaderboard_world_catalog_seasons.go : ce que le catalogue du
// classement mondial sait dire d'une SAISON, au-delà de son libellé.
//
//   - a-t-elle des stats détaillées (world_player_season_stats) ou seulement un
//     classement CSR scrappé ?
//   - sur QUELLES playlists a-t-elle réellement un relevé servi ?
//
// Le second point existe parce que les deux listes plates du catalogue (saisons,
// playlists) ne se croisent pas librement : une saison n'a été relevée que sur les
// playlists actives à l'époque, et un relevé a pu être refusé par le garde-fou
// qualité. Croiser les listes à l'aveugle donne des couples vides à l'écran ; le
// front couple donc ses sélecteurs avec les playlists de la saison choisie.
package duckdb

import (
	"context"
	"database/sql"
	"fmt"
	"log/slog"

	"levelup/go-api/internal/domain"
)

// WorldCSRSeasonPlaylistIDs retourne, pour chaque saison, les playlists qui ont un
// relevé SERVI (vue world_csr_leaderboard_latest — ce que la page lit réellement,
// pas l'archive complète des snapshots). Playlists triées par id, pour un ordre
// stable d'un appel à l'autre.
//
// Le shared DB est isolé par titre (ADR 0008), donc pas de filtre title_slug ici —
// même convention que les autres lectures du catalogue.
func WorldCSRSeasonPlaylistIDs(ctx context.Context, db *sql.DB) (map[string][]string, error) {
	const q = `
		SELECT DISTINCT season_id, playlist_id
		FROM world_csr_leaderboard_latest
		WHERE season_id <> '' AND playlist_id <> ''
		ORDER BY season_id, playlist_id`
	rows, err := db.QueryContext(ctx, q)
	if err != nil {
		return nil, fmt.Errorf("WorldCSRSeasonPlaylistIDs: %w", err)
	}
	defer rows.Close()
	out := make(map[string][]string)
	for rows.Next() {
		var season, playlist string
		if err := rows.Scan(&season, &playlist); err != nil {
			return nil, fmt.Errorf("WorldCSRSeasonPlaylistIDs: scan: %w", err)
		}
		out[season] = append(out[season], playlist)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("WorldCSRSeasonPlaylistIDs: %w", err)
	}
	return out, nil
}

// decorateCatalogSeasons renseigne, sur chaque saison du catalogue, le flag
// Enriched et la liste des playlists réellement relevées.
//
// Enriched est une erreur dure (la lecture porte sur la même DB que la liste des
// saisons : si elle échoue, le catalogue n'est pas fiable). Les couples, eux, sont
// best-effort AVEC log : leur absence fait retomber le front sur la liste plate de
// playlists — dégradation visible côté produit, pas une panne.
func decorateCatalogSeasons(ctx context.Context, db *sql.DB, seasons []domain.LeaderboardCatalogRef) error {
	enrichedIDs, err := scanIDColumn(ctx, db,
		`SELECT DISTINCT season_id FROM world_player_season_stats_latest WHERE season_id <> ''`)
	if err != nil {
		return fmt.Errorf("enriched seasons: %w", err)
	}
	enrichedSet := make(map[string]bool, len(enrichedIDs))
	for _, id := range enrichedIDs {
		enrichedSet[id] = true
	}
	pairs, err := WorldCSRSeasonPlaylistIDs(ctx, db)
	if err != nil {
		slog.WarnContext(ctx, "catalogue classement mondial : couples saison/playlist illisibles — sélecteurs non couplés",
			"module", logModuleLeaderboard, "err", err)
		pairs = nil
	}
	for i := range seasons {
		seasons[i].Enriched = enrichedSet[seasons[i].ID]
		seasons[i].PlaylistIDs = pairs[seasons[i].ID]
	}
	return nil
}
