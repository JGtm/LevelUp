// Package duckdb — FiltersRepo : résolution du contexte de filtres.
package duckdb

import (
	"context"
	"fmt"
	"time"

	"levelup/go-api/internal/domain"
)

// FiltersRepo implémente port.FiltersRepository.
type FiltersRepo struct {
	pdb *PlayerDB
}

// NewFiltersRepo crée un FiltersRepo depuis un PlayerDB ouvert.
func NewFiltersRepo(pdb *PlayerDB) *FiltersRepo {
	return &FiltersRepo{pdb: pdb}
}

// LoadMatchesForFilters charge tous les matchs du joueur pour la résolution cascade.
// Utilise mv_player_matches si disponible, sinon fallback sur match_registry.
func (r *FiltersRepo) LoadMatchesForFilters(ctx context.Context) ([]domain.FilterMatchRow, error) {
	ctx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()

	hasMV := r.hasMVPlayerMatches(ctx)
	var query string
	if hasMV {
		query = Q4MVMatchesForFilters
	} else {
		query = Q4MatchesForFilters
	}

	rows, err := r.pdb.Player.Query(ctx, query, r.pdb.XUID)
	if err != nil {
		return nil, fmt.Errorf("FiltersRepo.LoadMatchesForFilters: %w", err)
	}
	defer rows.Close()

	var results []domain.FilterMatchRow
	for rows.Next() {
		var m domain.FilterMatchRow
		if err := rows.Scan(
			&m.MatchID,
			&m.StartTime,
			&m.MapName,
			&m.MapNameFR,
			&m.PairName,
			&m.PairNameFR,
			&m.PlaylistName,
			&m.IsFirefight,
			&m.IsRanked,
			&m.SessionID,
			&m.SessionLabel,
			&m.IsWithFriends,
		); err != nil {
			return nil, fmt.Errorf("FiltersRepo.LoadMatchesForFilters scan: %w", err)
		}
		results = append(results, m)
	}
	return results, rows.Err()
}

// GetMatchCount retourne le nombre total de matchs dans shared_matches_v2.
func (r *FiltersRepo) GetMatchCount(ctx context.Context) (int, error) {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	var count int
	err := r.pdb.Shared.QueryRow(ctx, Q1MatchCount).Scan(&count)
	return count, err
}

// GetPlayerMatchCount retourne le nombre de matchs du joueur dans shared.
func (r *FiltersRepo) GetPlayerMatchCount(ctx context.Context) (int, error) {
	ctx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()
	var count int
	q := `SELECT COUNT(*) FROM shared.match_participants WHERE xuid = ?`
	err := r.pdb.Shared.QueryRow(ctx, q, r.pdb.XUID).Scan(&count)
	return count, err
}

// hasMVPlayerMatches vérifie si la vue matérialisée shared.mv_player_matches existe.
func (r *FiltersRepo) hasMVPlayerMatches(ctx context.Context) bool {
	ctx2, cancel := context.WithTimeout(ctx, 3*time.Second)
	defer cancel()
	rows, err := r.pdb.Player.Query(ctx2, "SELECT 1 FROM shared.mv_player_matches LIMIT 0")
	if err != nil {
		return false
	}
	rows.Close()
	return true
}

// GetAvailablePlaylists retourne les playlists uniques du joueur.
func (r *FiltersRepo) GetAvailablePlaylists(ctx context.Context) ([]domain.LabelValue, error) {
	ctx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	q := `
	SELECT DISTINCT
	    COALESCE(r.playlist_name_fr, r.playlist_name, '') AS label,
	    COALESCE(r.playlist_name, '')                     AS value
	FROM shared.match_registry r
	JOIN shared.match_participants p ON r.match_id = p.match_id
	WHERE p.xuid = ?
	  AND r.playlist_name IS NOT NULL
	  AND r.playlist_name != ''
	ORDER BY label ASC`

	rows, err := r.pdb.Shared.Query(ctx, q, r.pdb.XUID)
	if err != nil {
		return nil, fmt.Errorf("GetAvailablePlaylists: %w", err)
	}
	defer rows.Close()

	var results []domain.LabelValue
	for rows.Next() {
		var lv domain.LabelValue
		if err := rows.Scan(&lv.Label, &lv.Value); err != nil {
			return nil, err
		}
		results = append(results, lv)
	}
	return results, rows.Err()
}

// GetAvailableMaps retourne les cartes uniques jouées.
func (r *FiltersRepo) GetAvailableMaps(ctx context.Context) ([]domain.LabelValue, error) {
	ctx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	q := `
	SELECT DISTINCT
	    COALESCE(r.map_name_fr, r.map_name, '') AS label,
	    COALESCE(r.map_name, '')                AS value
	FROM shared.match_registry r
	JOIN shared.match_participants p ON r.match_id = p.match_id
	WHERE p.xuid = ?
	  AND r.map_name IS NOT NULL
	ORDER BY label ASC`

	rows, err := r.pdb.Shared.Query(ctx, q, r.pdb.XUID)
	if err != nil {
		return nil, fmt.Errorf("GetAvailableMaps: %w", err)
	}
	defer rows.Close()

	var results []domain.LabelValue
	for rows.Next() {
		var lv domain.LabelValue
		if err := rows.Scan(&lv.Label, &lv.Value); err != nil {
			return nil, err
		}
		results = append(results, lv)
	}
	return results, rows.Err()
}
