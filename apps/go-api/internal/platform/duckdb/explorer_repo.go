// Package duckdb — ExplorerRepo : matchs communs entre deux joueurs.
//
// Port Go de apps/api/app/routers/explorer.py + services/explorer.
package duckdb

import (
	"context"
	"fmt"

	"levelup/go-api/internal/domain"
)

// ExplorerRepo implémente port.ExplorerRepository sur DuckDB.
type ExplorerRepo struct {
	pdb  *PlayerDB
	xuid string
}

// NewExplorerRepo crée un ExplorerRepo.
func NewExplorerRepo(pdb *PlayerDB, xuid string) *ExplorerRepo {
	return &ExplorerRepo{pdb: pdb, xuid: xuid}
}

// GetCommonMatches retourne les matchs communs entre xuid1 et xuid2 (max 100).
// Q19 retourne 10 colonnes : match_id, start_time, map_ui, mode_ui,
// player1_team_id, player2_team_id, player1_outcome, player1_kills, player1_deaths, player1_kda.
func (r *ExplorerRepo) GetCommonMatches(ctx context.Context, xuid1, xuid2 string) ([]domain.CommonMatchRaw, error) {
	rows, err := r.pdb.Player.Query(ctx, Q19CommonMatches, xuid1, xuid2)
	if err != nil {
		return nil, fmt.Errorf("ExplorerRepo.GetCommonMatches: query: %w", err)
	}
	defer rows.Close()

	var results []domain.CommonMatchRaw
	for rows.Next() {
		var m domain.CommonMatchRaw
		if err := rows.Scan(
			&m.MatchID,
			&m.StartTime,
			&m.MapUI,
			&m.ModeUI,
			&m.Player1TeamID,
			&m.Player2TeamID,
			&m.Player1Outcome,
			&m.Player1Kills,
			&m.Player1Deaths,
			&m.Player1KDA,
		); err != nil {
			return nil, fmt.Errorf("ExplorerRepo.GetCommonMatches: scan: %w", err)
		}
		results = append(results, m)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("ExplorerRepo.GetCommonMatches: rows: %w", err)
	}
	return results, nil
}

// ResolveXUIDByGamertag résout un gamertag en xuid via xuid_aliases (ILIKE).
func (r *ExplorerRepo) ResolveXUIDByGamertag(ctx context.Context, gamertag string) (string, error) {
	const q = `SELECT xuid FROM shared.xuid_aliases WHERE gamertag ILIKE ? LIMIT 1`
	row := r.pdb.Player.QueryRow(ctx, q, gamertag)
	var xuid string
	if err := row.Scan(&xuid); err != nil {
		return "", fmt.Errorf("ExplorerRepo.ResolveXUIDByGamertag(%q): %w", gamertag, err)
	}
	return xuid, nil
}
