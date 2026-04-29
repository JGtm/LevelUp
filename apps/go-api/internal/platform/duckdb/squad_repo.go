// Package duckdb â€” squad_repo.go : accÃ¨s DB pour la page Escouade et SynthÃ¨se.
package duckdb

import (
	"context"
	"fmt"
	"strings"
	"time"

	"levelup/go-api/internal/domain"
	"levelup/go-api/internal/legacymatch"
)

// SquadRepo implÃ©mente port.SquadRepository.
type SquadRepo struct {
	pdb *PlayerDB
}

// NewSquadRepo crÃ©e un SquadRepo pour un joueur.
func NewSquadRepo(pdb *PlayerDB) *SquadRepo {
	return &SquadRepo{pdb: pdb}
}

// LoadTopTeammates charge les meilleurs coÃ©quipiers du joueur (Q29, top 50).
func (r *SquadRepo) LoadTopTeammates(ctx context.Context, xuid string) ([]domain.TopTeammateRow, error) {
	ctx, cancel := context.WithTimeout(ctx, 15*time.Second)
	defer cancel()

	rows, err := r.pdb.ReadDB().Query(ctx, Q29TopTeammates, xuid, xuid)
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

// LookupXUIDByGamertag rÃ©sout un gamertag (ILIKE, case-insensitive) vers son
// XUID via shared.xuid_aliases. Sert de fallback pour les coÃ©quipiers sÃ©lectionnÃ©s
// qui sortent du top 50 LoadTopTeammates (saisie libre dans la combobox).
//
// Si plusieurs aliases correspondent au mÃªme gamertag (changement de pseudo
// historique), on retourne le plus rÃ©cent. Si aucun alias, retourne ("", false, nil).
func (r *SquadRepo) LookupXUIDByGamertag(ctx context.Context, gamertag string) (string, bool, error) {
	gamertag = strings.TrimSpace(gamertag)
	if gamertag == "" {
		return "", false, nil
	}
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	// shared.xuid_aliases : (xuid, gamertag, last_seen_at)
	const q = `
SELECT xuid
FROM shared.xuid_aliases
WHERE gamertag ILIKE ?
ORDER BY last_seen_at DESC NULLS LAST
LIMIT 1`

	rows, err := r.pdb.ReadDB().Query(ctx, q, gamertag)
	if err != nil {
		return "", false, fmt.Errorf("LookupXUIDByGamertag(%q): %w", gamertag, err)
	}
	defer rows.Close()
	if !rows.Next() {
		return "", false, rows.Err()
	}
	var xuid string
	if err := rows.Scan(&xuid); err != nil {
		return "", false, fmt.Errorf("LookupXUIDByGamertag scan: %w", err)
	}
	return xuid, xuid != "", nil
}

// LoadSquadMatches charge les matchs communs joueur+coÃ©quipier (Q30).
func (r *SquadRepo) LoadSquadMatches(ctx context.Context, playerXUID, teammateXUID string) ([]domain.SquadMatchRow, error) {
	ctx, cancel := context.WithTimeout(ctx, 15*time.Second)
	defer cancel()

	rows, err := r.pdb.ReadDB().Query(ctx, Q30SquadMatches, teammateXUID, playerXUID)
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
			&row.HeadshotKills,
			&row.PerfectKills,
		); err != nil {
			return nil, fmt.Errorf("LoadSquadMatches scan: %w", err)
		}
		result = append(result, row)
	}
	return result, rows.Err()
}

// LoadTeammateMatches charge les stats du coÃ©quipier sur les matchs communs (Q31).
func (r *SquadRepo) LoadTeammateMatches(ctx context.Context, playerXUID, teammateXUID string) ([]domain.TeammateMatchRow, error) {
	ctx, cancel := context.WithTimeout(ctx, 15*time.Second)
	defer cancel()

	rows, err := r.pdb.ReadDB().Query(ctx, Q31TeammateMatches, playerXUID, teammateXUID)
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
			&row.MyTeamScore,
			&row.EnemyTeamScore,
		); err != nil {
			return nil, fmt.Errorf("LoadTeammateMatches scan: %w", err)
		}
		result = append(result, row)
	}
	return result, rows.Err()
}

// LoadImpactEvents charge les Ã©vÃ©nements highlight pour une liste de match_ids (Q32 dynamique).
// matchIDs est la liste des identifiants â€” si vide, retourne nil directement.
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

	rows, err := r.pdb.ReadDB().Query(ctx, query, args...)
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

// LoadSynthesisHeatmap charge les donnÃ©es heatmap mapÃ—mode (Q33).
func (r *SquadRepo) LoadSynthesisHeatmap(ctx context.Context, xuid string) ([]domain.SynthesisHeatmapRow, error) {
	ctx, cancel := context.WithTimeout(ctx, 15*time.Second)
	defer cancel()

	rows, err := r.pdb.ReadDB().Query(ctx, Q33SynthesisHeatmap, xuid)
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
func (r *SquadRepo) LoadSynthesisMatches(ctx context.Context, xuid string) ([]legacymatch.SynthesisMatchRow, error) {
	ctx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()

	rows, err := r.pdb.ReadDB().Query(ctx, Q33bSynthesisMatches, xuid)
	if err != nil {
		return nil, fmt.Errorf("LoadSynthesisMatches: %w", err)
	}
	defer rows.Close()

	var result []legacymatch.SynthesisMatchRow
	for rows.Next() {
		var row legacymatch.SynthesisMatchRow
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
			&row.SessionLabel,
			&row.IsRanked,
			&row.IsFirefight,
			&row.PlaylistName,
		); err != nil {
			return nil, fmt.Errorf("LoadSynthesisMatches scan: %w", err)
		}
		result = append(result, row)
	}
	return result, rows.Err()
}

// Ensure SquadRepo implements port.SquadRepository at compile time.
// (VÃ©rification implicite via injection dans le service.)
