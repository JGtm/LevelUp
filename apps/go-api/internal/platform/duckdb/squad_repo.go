// Package duckdb — squad_repo.go : accès DB pour la page Escouade et Synthèse.
package duckdb

import (
	"context"
	"fmt"
	"strings"
	"time"

	"levelup/go-api/internal/domain"
)

// SquadRepo implémente port.SquadRepository.
type SquadRepo struct {
	pdb *PlayerDB
}

// NewSquadRepo crée un SquadRepo pour un joueur.
func NewSquadRepo(pdb *PlayerDB) *SquadRepo {
	return &SquadRepo{pdb: pdb}
}

// LoadTopTeammates charge les 10 meilleurs coéquipiers du joueur (Q29).
func (r *SquadRepo) LoadTopTeammates(ctx context.Context, xuid string) ([]domain.TopTeammateRow, error) {
	ctx, cancel := context.WithTimeout(ctx, 15*time.Second)
	defer cancel()

	rows, err := r.pdb.Player.Query(ctx, Q29TopTeammates, xuid, xuid)
	if err != nil {
		return nil, fmt.Errorf("LoadTopTeammates: %w", err)
	}
	defer rows.Close()

	var result []domain.TopTeammateRow
	for rows.Next() {
		var row domain.TopTeammateRow
		if err := rows.Scan(
			&row.XUID,
			&row.Gamertag,
			&row.GamesTogether,
			&row.WinsTogether,
			&row.WinRate,
			&row.AvgKills,
			&row.AvgDeaths,
			&row.AvgKDA,
		); err != nil {
			return nil, fmt.Errorf("LoadTopTeammates scan: %w", err)
		}
		result = append(result, row)
	}
	return result, rows.Err()
}

// LoadSquadMatches charge les matchs communs joueur+coéquipier (Q30).
func (r *SquadRepo) LoadSquadMatches(ctx context.Context, playerXUID, teammateXUID string) ([]domain.SquadMatchRow, error) {
	ctx, cancel := context.WithTimeout(ctx, 15*time.Second)
	defer cancel()

	rows, err := r.pdb.Player.Query(ctx, Q30SquadMatches, teammateXUID, playerXUID)
	if err != nil {
		return nil, fmt.Errorf("LoadSquadMatches: %w", err)
	}
	defer rows.Close()

	var result []domain.SquadMatchRow
	for rows.Next() {
		var row domain.SquadMatchRow
		if err := rows.Scan(
			&row.MatchID,
			&row.StartTime,
			&row.MapName,
			&row.MapUI,
			&row.PairName,
			&row.PlaylistName,
			&row.IsFirefight,
			&row.IsRanked,
			&row.Outcome,
			&row.Kills,
			&row.Deaths,
			&row.Assists,
			&row.KDA,
			&row.Accuracy,
			&row.TimePlayedSecs,
			&row.TeamMMR,
			&row.SessionID,
			&row.SessionLabel,
			&row.PerformanceScore,
			&row.IsWithFriends,
		); err != nil {
			return nil, fmt.Errorf("LoadSquadMatches scan: %w", err)
		}
		result = append(result, row)
	}
	return result, rows.Err()
}

// LoadTeammateMatches charge les stats du coéquipier sur les matchs communs (Q31).
func (r *SquadRepo) LoadTeammateMatches(ctx context.Context, playerXUID, teammateXUID string) ([]domain.TeammateMatchRow, error) {
	ctx, cancel := context.WithTimeout(ctx, 15*time.Second)
	defer cancel()

	rows, err := r.pdb.Player.Query(ctx, Q31TeammateMatches, playerXUID, teammateXUID)
	if err != nil {
		return nil, fmt.Errorf("LoadTeammateMatches: %w", err)
	}
	defer rows.Close()

	var result []domain.TeammateMatchRow
	for rows.Next() {
		var row domain.TeammateMatchRow
		if err := rows.Scan(
			&row.MatchID,
			&row.StartTime,
			&row.MapUI,
			&row.PairName,
			&row.Outcome,
			&row.Kills,
			&row.Deaths,
			&row.Assists,
			&row.Ratio,
			&row.TimePlayedSecs,
			&row.TeamMMR,
			&row.Accuracy,
		); err != nil {
			return nil, fmt.Errorf("LoadTeammateMatches scan: %w", err)
		}
		result = append(result, row)
	}
	return result, rows.Err()
}

// LoadImpactEvents charge les événements highlight pour une liste de match_ids (Q32 dynamique).
// matchIDs est la liste des identifiants — si vide, retourne nil directement.
func (r *SquadRepo) LoadImpactEvents(ctx context.Context, matchIDs []string) ([]domain.ImpactEventRow, error) {
	if len(matchIDs) == 0 {
		return nil, nil
	}
	ctx, cancel := context.WithTimeout(ctx, 15*time.Second)
	defer cancel()

	// Construire la clause IN dynamiquement.
	placeholders := make([]string, len(matchIDs))
	args := make([]interface{}, len(matchIDs))
	for i, id := range matchIDs {
		placeholders[i] = "?"
		args[i] = id
	}
	query := fmt.Sprintf(Q32SquadImpactEventsTemplate, strings.Join(placeholders, ","))

	rows, err := r.pdb.Player.Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("LoadImpactEvents: %w", err)
	}
	defer rows.Close()

	var result []domain.ImpactEventRow
	for rows.Next() {
		var row domain.ImpactEventRow
		if err := rows.Scan(
			&row.MatchID,
			&row.XUID,
			&row.Gamertag,
			&row.EventType,
			&row.TimeMS,
		); err != nil {
			return nil, fmt.Errorf("LoadImpactEvents scan: %w", err)
		}
		result = append(result, row)
	}
	return result, rows.Err()
}

// LoadSynthesisHeatmap charge les données heatmap map×mode (Q33).
func (r *SquadRepo) LoadSynthesisHeatmap(ctx context.Context, xuid string) ([]domain.SynthesisHeatmapRow, error) {
	ctx, cancel := context.WithTimeout(ctx, 15*time.Second)
	defer cancel()

	rows, err := r.pdb.Player.Query(ctx, Q33SynthesisHeatmap, xuid)
	if err != nil {
		return nil, fmt.Errorf("LoadSynthesisHeatmap: %w", err)
	}
	defer rows.Close()

	var result []domain.SynthesisHeatmapRow
	for rows.Next() {
		var row domain.SynthesisHeatmapRow
		if err := rows.Scan(
			&row.MapName,
			&row.ModeName,
			&row.MatchCount,
			&row.Wins,
		); err != nil {
			return nil, fmt.Errorf("LoadSynthesisHeatmap scan: %w", err)
		}
		result = append(result, row)
	}
	return result, rows.Err()
}

// LoadSynthesisMatches charge les matchs du joueur pour le calcul top_weeks (Q33b).
func (r *SquadRepo) LoadSynthesisMatches(ctx context.Context, xuid string) ([]domain.SynthesisMatchRow, error) {
	ctx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()

	rows, err := r.pdb.Player.Query(ctx, Q33bSynthesisMatches, xuid)
	if err != nil {
		return nil, fmt.Errorf("LoadSynthesisMatches: %w", err)
	}
	defer rows.Close()

	var result []domain.SynthesisMatchRow
	for rows.Next() {
		var row domain.SynthesisMatchRow
		if err := rows.Scan(
			&row.MatchID,
			&row.StartTime,
			&row.Outcome,
			&row.Kills,
			&row.Deaths,
			&row.KDA,
			&row.IsWithFriends,
			&row.Accuracy,
			&row.TimePlayedSecs,
			&row.PerformanceScore,
		); err != nil {
			return nil, fmt.Errorf("LoadSynthesisMatches scan: %w", err)
		}
		result = append(result, row)
	}
	return result, rows.Err()
}

// Ensure SquadRepo implements port.SquadRepository at compile time.
// (Vérification implicite via injection dans le service.)
