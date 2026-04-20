// Package duckdb — home_repo.go : accès DB pour la page d'accueil Mission Control.
package duckdb

import (
	"context"
	"database/sql"
	"strings"

	"levelup/go-api/internal/domain"
)

// HomeRepo fournit les données de la page d'accueil depuis DuckDB.
type HomeRepo struct {
	pdb *PlayerDB
}

// NewHomeRepo crée un HomeRepo pour un joueur.
func NewHomeRepo(pdb *PlayerDB) *HomeRepo {
	return &HomeRepo{pdb: pdb}
}

// LoadHomeMatches charge les 200 derniers matchs du joueur (Q26).
func (r *HomeRepo) LoadHomeMatches(ctx context.Context) ([]domain.HomeMatchRow, error) {
	rows, err := r.pdb.Player.Query(ctx, Q26HomeMatches, r.pdb.XUID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var result []domain.HomeMatchRow
	for rows.Next() {
		var row domain.HomeMatchRow
		if err := rows.Scan(
			&row.MatchID,
			&row.StartTime,
			&row.MapName,
			&row.MapNameFR,
			&row.PairName,
			&row.PairNameFR,
			&row.PlaylistName,
			&row.IsFirefight,
			&row.IsRanked,
			&row.SessionLabel,
			&row.IsWithFriends,
			&row.Outcome,
			&row.Kills,
			&row.Deaths,
			&row.Assists,
			&row.KDA,
			&row.Ratio,
			&row.Accuracy,
			&row.TimePlayedSecs,
		); err != nil {
			return nil, err
		}
		result = append(result, row)
	}
	return result, rows.Err()
}

// CountPlayerMatches retourne le nombre total de matchs du joueur (Q26b).
func (r *HomeRepo) CountPlayerMatches(ctx context.Context) (int, error) {
	var count int
	err := r.pdb.Player.QueryRow(ctx, Q26bCountPlayerMatches, r.pdb.XUID).Scan(&count)
	return count, err
}

// LoadHomeSessions charge les sessions avec label depuis player_match_enrichment (Q27).
func (r *HomeRepo) LoadHomeSessions(ctx context.Context) ([]domain.HomeSessionRow, error) {
	rows, err := r.pdb.Player.Query(ctx, Q27HomeSessions)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var result []domain.HomeSessionRow
	for rows.Next() {
		var row domain.HomeSessionRow
		if err := rows.Scan(
			&row.MatchID,
			&row.SessionID,
			&row.SessionLabel,
			&row.IsWithFriends,
			&row.StartTime,
		); err != nil {
			return nil, err
		}
		result = append(result, row)
	}
	return result, rows.Err()
}

// LoadRecentMedia charge les médias récents du joueur (Q28).
// Retourne une liste vide si la table media_files n'existe pas.
func (r *HomeRepo) LoadRecentMedia(ctx context.Context, limit int) ([]domain.HomeMediaRow, error) {
	rows, err := r.pdb.Player.Query(ctx, Q28RecentMedia, limit)
	if err != nil {
		// La table media_files peut ne pas exister — dégradation silencieuse.
		if isTableNotFoundErr(err) {
			return nil, nil
		}
		return nil, err
	}
	defer rows.Close()

	var result []domain.HomeMediaRow
	for rows.Next() {
		var row domain.HomeMediaRow
		var matchID sql.NullString
		var matchStartTime sql.NullTime
		if err := rows.Scan(&row.FileName, &matchID, &matchStartTime); err != nil {
			return nil, err
		}
		if matchID.Valid {
			row.MatchID = &matchID.String
		}
		if matchStartTime.Valid {
			row.MatchStartTime = &matchStartTime.Time
		}
		result = append(result, row)
	}
	return result, rows.Err()
}

// isTableNotFoundErr détecte les erreurs "table not found" DuckDB.
func isTableNotFoundErr(err error) bool {
	if err == nil {
		return false
	}
	msg := err.Error()
	return strings.Contains(msg, "Table with name") ||
		strings.Contains(msg, "does not exist") ||
		strings.Contains(msg, "no such table")
}
