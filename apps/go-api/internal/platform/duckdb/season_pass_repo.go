// Package duckdb — season_pass_repo.go : accès DB pour la page Season Pass.
package duckdb

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"time"

	"levelup/go-api/internal/domain"
)

// SeasonPassRepo implémente port.SeasonPassRepository.
type SeasonPassRepo struct {
	pdb *PlayerDB
}

// NewSeasonPassRepo crée un SeasonPassRepo pour un joueur.
func NewSeasonPassRepo(pdb *PlayerDB) *SeasonPassRepo {
	return &SeasonPassRepo{pdb: pdb}
}

// trackProgressMap mappe reward_track_path → {rank, partial} depuis le payload Waypoint.
type trackProgressMap map[string]struct {
	Rank    int
	Partial int
}

// activePayload désérialise le JSON Waypoint pour l'extraction de progression.
type activePayload struct {
	ActiveOperationRewardTrackPath string `json:"ActiveOperationRewardTrackPath"`
	OperationRewardTracks          []struct {
		RewardTrackPath string `json:"RewardTrackPath"`
		CurrentProgress struct {
			Rank            int `json:"Rank"`
			PartialProgress int `json:"PartialProgress"`
		} `json:"CurrentProgress"`
	} `json:"OperationRewardTracks"`
}

// loadActivePayload charge la progression du joueur depuis le payload Waypoint le plus récent.
// Retourne une map vide (sans erreur) si aucune entrée n'existe.
func (r *SeasonPassRepo) loadActivePayload(ctx context.Context) (trackProgressMap, string, error) {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	var rawJSON, activeTrackPath string
	err := r.pdb.Metadata.QueryRow(ctx, `
		SELECT raw_payload_json
		FROM battlepass_track_definitions
		WHERE is_current = TRUE
		ORDER BY last_seen_at DESC
		LIMIT 1`).Scan(&rawJSON)
	if err != nil {
		if err == sql.ErrNoRows {
			return trackProgressMap{}, activeTrackPath, nil
		}
		return nil, activeTrackPath, fmt.Errorf("season_pass_repo: active payload query: %w", err)
	}

	var payload activePayload
	if jsonErr := json.Unmarshal([]byte(rawJSON), &payload); jsonErr != nil {
		// Payload corrompu → retourner map vide sans bloquer
		return trackProgressMap{}, "", nil
	}

	activeTrackPath = payload.ActiveOperationRewardTrackPath
	m := make(trackProgressMap, len(payload.OperationRewardTracks))
	for _, t := range payload.OperationRewardTracks {
		m[t.RewardTrackPath] = struct {
			Rank    int
			Partial int
		}{Rank: t.CurrentProgress.Rank, Partial: t.CurrentProgress.PartialProgress}
	}
	return m, activeTrackPath, nil
}

// seasonPassTrackRow représente une ligne du JOIN tracks + translations.
type seasonPassTrackRow struct {
	rewardTrackPath    string
	xpPerRank          sql.NullInt64
	isCurrent          bool
	trackName          sql.NullString
}

// LoadSeasonPassTracks charge toutes les tracks connues avec traductions.
// La progression joueur est injectée depuis le payload Waypoint persisté.
func (r *SeasonPassRepo) LoadSeasonPassTracks(ctx context.Context, _, _ string) ([]domain.SeasonPassTrackSummary, error) {
	progressMap, activeTrackPath, err := r.loadActivePayload(ctx)
	if err != nil {
		return nil, err
	}

	ctx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	// Récupère la définition la plus récente par track + traduction FR (fallback EN).
	const query = `
		WITH latest AS (
			SELECT reward_track_path, content_hash, xp_per_rank, is_current, last_seen_at,
			       ROW_NUMBER() OVER (PARTITION BY reward_track_path ORDER BY last_seen_at DESC) AS rn
			FROM battlepass_track_definitions
		)
		SELECT d.reward_track_path, d.xp_per_rank, d.is_current,
		       COALESCE(t_fr.track_name, t_en.track_name) AS track_name
		FROM latest d
		LEFT JOIN battlepass_track_translations t_fr
		       ON t_fr.reward_track_path = d.reward_track_path
		      AND t_fr.content_hash = d.content_hash
		      AND t_fr.lang = 'fr'
		LEFT JOIN battlepass_track_translations t_en
		       ON t_en.reward_track_path = d.reward_track_path
		      AND t_en.content_hash = d.content_hash
		      AND t_en.lang = 'en'
		WHERE d.rn = 1
		ORDER BY d.is_current DESC, d.last_seen_at DESC`

	rows, err := r.pdb.Metadata.Query(ctx, query)
	if err != nil {
		if isTableNotFoundErr(err) {
			return nil, nil
		}
		return nil, fmt.Errorf("season_pass_repo: tracks query: %w", err)
	}
	defer rows.Close()

	var tracks []domain.SeasonPassTrackSummary
	for rows.Next() {
		var row seasonPassTrackRow
		if err := rows.Scan(
			&row.rewardTrackPath,
			&row.xpPerRank,
			&row.isCurrent,
			&row.trackName,
		); err != nil {
			return nil, fmt.Errorf("season_pass_repo: scan: %w", err)
		}

		prog := progressMap[row.rewardTrackPath]
		isActive := row.rewardTrackPath == activeTrackPath || row.isCurrent
		summary := buildTrackSummary(row, prog.Rank, prog.Partial, isActive)
		tracks = append(tracks, summary)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return tracks, nil
}

// buildTrackSummary construit un SeasonPassTrackSummary depuis les données brutes.
func buildTrackSummary(row seasonPassTrackRow, rank, partial int, isActive bool) domain.SeasonPassTrackSummary {
	name := row.rewardTrackPath
	if row.trackName.Valid && row.trackName.String != "" {
		name = row.trackName.String
	}

	status := computeSeasonPassStatus(isActive, rank)

	s := domain.SeasonPassTrackSummary{
		RewardTrackPath: row.rewardTrackPath,
		Name:            name,
		Status:          status,
		IsActive:        isActive,
		IsOwned:         rank > 0 || isActive,
		CurrentRank:     rank,
		PartialProgress: partial,
	}
	if row.xpPerRank.Valid {
		v := int(row.xpPerRank.Int64)
		s.XPPerRank = &v
	}
	return s
}

// computeSeasonPassStatus détermine le statut d'un track depuis les indicateurs connus.
func computeSeasonPassStatus(isActive bool, currentRank int) domain.SeasonPassStatus {
	if isActive {
		return domain.SeasonPassStatusActive
	}
	if currentRank > 0 {
		return domain.SeasonPassStatusInProgress
	}
	return domain.SeasonPassStatusNotStarted
}
