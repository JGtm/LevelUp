// Package duckdb — engagement_response_bins_repo.go : persistence DuckDB des
// bins de reponse d'engagement (modele lobby-anchored v2, cf.
// .ai/V7/PLAN_ENGAGEMENT_REFONTE_LOBBY_2026-07.md).
//
// Table engagement_response_bins (player DB) : coef de reponse par (xuid,
// mode_category, intensity_bin). Ecritures basse frequence, SELECT-then-UPDATE
// -or-INSERT sous lease KindPlayer (ART-safe, meme pattern que
// engagement_coefficients). Extrait de engagement_score_repo.go (limite 500L).
package duckdb

import (
	"context"
	"errors"
	"fmt"
	"time"

	"levelup/go-api/internal/domain"
	"levelup/go-api/internal/platform/dblease"
	"levelup/go-api/internal/port"
)

// LoadResponseBins charge les bins de reponse (terciles d'intensite) du joueur
// pour une categorie de mode. Retourne (nil, nil) si aucun bin persiste.
func (r *EngagementScoreRepo) LoadResponseBins(
	ctx context.Context,
	xuid, modeCategory string,
) (*domain.EngagementResponseBins, error) {
	if xuid == "" || modeCategory == "" {
		return nil, errors.New("EngagementScoreRepo.LoadResponseBins: xuid and modeCategory required")
	}

	ctx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	if !r.responseBinsTableExists(ctx) {
		return nil, port.ErrEngagementUnavailable
	}

	const q = `
		SELECT intensity_bin, lower_bound, upper_bound, coef_lobby, n_matches
		FROM engagement_response_bins
		WHERE xuid = ? AND mode_category = ?
		ORDER BY lower_bound
	`
	rows, err := r.pdb.ReadDB().QueryRecovered(ctx, q, xuid, modeCategory)
	if err != nil {
		return nil, fmt.Errorf("EngagementScoreRepo.LoadResponseBins: query: %w", err)
	}
	defer rows.Close()

	bins := make([]domain.EngagementIntensityBin, 0, 3)
	for rows.Next() {
		var b domain.EngagementIntensityBin
		if err := rows.Scan(&b.Bin, &b.LowerBound, &b.UpperBound, &b.CoefLobby, &b.NMatches); err != nil {
			return nil, fmt.Errorf("EngagementScoreRepo.LoadResponseBins: scan: %w", err)
		}
		bins = append(bins, b)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("EngagementScoreRepo.LoadResponseBins: rows: %w", err)
	}
	if len(bins) == 0 {
		return nil, nil
	}
	return &domain.EngagementResponseBins{XUID: xuid, ModeCategory: modeCategory, Bins: bins}, nil
}

// SaveResponseBins persiste / met a jour les bins de reponse (SELECT-then-UPDATE
// -or-INSERT par bin, sous lease KindPlayer). ART-safe : pas d'ON CONFLICT, pas
// d'index secondaire mute (meme pattern que SaveEngagementCoefficient).
func (r *EngagementScoreRepo) SaveResponseBins(
	ctx context.Context,
	bins domain.EngagementResponseBins,
) error {
	if bins.XUID == "" || bins.ModeCategory == "" {
		return errors.New("EngagementScoreRepo.SaveResponseBins: XUID and ModeCategory required")
	}

	ctx, cancel := context.WithTimeout(ctx, 15*time.Second)
	defer cancel()

	if !r.responseBinsTableExists(ctx) {
		return port.ErrEngagementUnavailable
	}

	w, err := r.pdb.AcquirePlayerWriterTimeout(dblease.PlayerLeaseTimeout)
	if err != nil {
		return fmt.Errorf("EngagementScoreRepo.SaveResponseBins: lease: %w", err)
	}
	defer w.Release()

	now := time.Now().UTC()
	for _, b := range bins.Bins {
		if err := r.pdb.Player.UpsertNoConflict(ctx,
			`SELECT 1 FROM engagement_response_bins WHERE xuid = ? AND mode_category = ? AND intensity_bin = ?`,
			[]any{bins.XUID, bins.ModeCategory, b.Bin},
			`UPDATE engagement_response_bins SET lower_bound = ?, upper_bound = ?, coef_lobby = ?, n_matches = ?, last_updated = ?
			 WHERE xuid = ? AND mode_category = ? AND intensity_bin = ?`,
			[]any{b.LowerBound, b.UpperBound, b.CoefLobby, b.NMatches, now, bins.XUID, bins.ModeCategory, b.Bin},
			`INSERT INTO engagement_response_bins (xuid, mode_category, intensity_bin, lower_bound, upper_bound, coef_lobby, n_matches, last_updated)
			 VALUES (?, ?, ?, ?, ?, ?, ?, ?)`,
			[]any{bins.XUID, bins.ModeCategory, b.Bin, b.LowerBound, b.UpperBound, b.CoefLobby, b.NMatches, now},
		); err != nil {
			return fmt.Errorf("EngagementScoreRepo.SaveResponseBins: bin %s: %w", b.Bin, err)
		}
	}
	return nil
}

// responseBinsTableExists verifie l'existence de engagement_response_bins.
// Le resultat est memoise le temps de vie du repo (E1) : le scan
// information_schema ne tourne qu'une fois par handle, pas une fois par appel.
func (r *EngagementScoreRepo) responseBinsTableExists(ctx context.Context) bool {
	if r.responseBinsExists != nil {
		return *r.responseBinsExists
	}
	exists := r.queryResponseBinsTableExists(ctx)
	r.responseBinsExists = &exists
	return exists
}

// queryResponseBinsTableExists scanne information_schema (sans cache).
func (r *EngagementScoreRepo) queryResponseBinsTableExists(ctx context.Context) bool {
	if r.pdb == nil || r.pdb.ReadDB() == nil {
		return false
	}
	var count int
	rows, err := r.pdb.ReadDB().QueryRowRecovered(ctx, `
		SELECT COUNT(*) FROM information_schema.tables
		WHERE table_name = 'engagement_response_bins'
	`)
	if err != nil {
		return false
	}
	defer rows.Close()
	if err := rows.Scan(&count); err != nil {
		return false
	}
	return count > 0
}
