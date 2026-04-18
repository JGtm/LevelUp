// Package duckdb — MatchHistoryRepo : chargement paginé de l'historique des parties.
package duckdb

import (
	"context"
	"fmt"
	"time"

	"levelup/go-api/internal/domain"
)

// MatchHistoryRepo implémente port.MatchHistoryRepository.
type MatchHistoryRepo struct {
	pdb *PlayerDB
}

// NewMatchHistoryRepo crée un MatchHistoryRepo depuis un PlayerDB.
func NewMatchHistoryRepo(pdb *PlayerDB) *MatchHistoryRepo {
	return &MatchHistoryRepo{pdb: pdb}
}

// LoadAll charge tous les matchs du joueur avec les stats de participation.
func (r *MatchHistoryRepo) LoadAll(ctx context.Context) ([]domain.MatchHistoryRawRow, error) {
	ctx, cancel := context.WithTimeout(ctx, 60*time.Second)
	defer cancel()

	rows, err := r.pdb.Player.Query(ctx, Q5MatchHistory, r.pdb.XUID, r.pdb.XUID)
	if err != nil {
		return nil, fmt.Errorf("MatchHistoryRepo.LoadAll: %w", err)
	}
	defer rows.Close()

	var results []domain.MatchHistoryRawRow
	for rows.Next() {
		var m domain.MatchHistoryRawRow
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
			&m.Outcome,
			&m.TeamMMR,
			&m.EnemyMMR,
			&m.Kills,
			&m.Deaths,
			&m.Assists,
			&m.KDA,
			&m.Accuracy,
			&m.PersonalScore,
			&m.AverageLifeSeconds,
			&m.TimePlayedSeconds,
			&m.IsExcluded,
		); err != nil {
			return nil, fmt.Errorf("MatchHistoryRepo.LoadAll scan: %w", err)
		}
		results = append(results, m)
	}
	return results, rows.Err()
}

// LoadMapWinRates calcule le win_rate historique par carte (sur tous les matchs).
// Retourne map[map_name] -> {wins, total}.
func (r *MatchHistoryRepo) LoadMapWinRates(ctx context.Context) (map[string][2]int, error) {
	ctx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()

	q := `
	SELECT r.map_name,
	       SUM(CASE WHEN p.outcome = 2 THEN 1 ELSE 0 END) AS wins,
	       COUNT(*) AS total
	FROM shared.match_registry r
	JOIN shared.match_participants p ON r.match_id = p.match_id
	WHERE p.xuid = ? AND r.map_name IS NOT NULL
	GROUP BY r.map_name`

	rows, err := r.pdb.Shared.Query(ctx, q, r.pdb.XUID)
	if err != nil {
		return nil, fmt.Errorf("LoadMapWinRates: %w", err)
	}
	defer rows.Close()

	result := make(map[string][2]int)
	for rows.Next() {
		var mapName string
		var wins, total int
		if err := rows.Scan(&mapName, &wins, &total); err != nil {
			return nil, err
		}
		result[mapName] = [2]int{wins, total}
	}
	return result, rows.Err()
}
