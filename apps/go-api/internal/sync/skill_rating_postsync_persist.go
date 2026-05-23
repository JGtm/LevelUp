// Package sync — skill_rating_postsync_persist.go : Phase 4.3 du refactor
// Collect→Persist — chemin INSERT-only pour upsertLUSRRatings.
//
// **Activation** : env var LEVELUP_POSTSYNC_INSERT_ONLY=1.
//
// Logique identique au chemin legacy (compute delta, tier, etc.) mais au
// lieu de UPDATE-then-INSERT row-by-row, accumule les LUSRRatingInsert
// en RAM puis appelle PostSyncLUSRPersister.Upsert qui fait DELETE WHERE
// match_id IN (...) AND rating_type='LUSR' + INSERT batch en 1 TX atomique.
//
// Conserve le filtre existingCSR (skip silencieusement les matchs CSR
// pour ne pas écraser un CSR avec un LUSR) et existingLUSR (anti-overwrite
// si non-force).

package sync

import (
	"context"
	"database/sql"
	"log/slog"
	"math"
	"time"

	"levelup/go-api/internal/persist"
)

// upsertLUSRRatingsBatch est la variante INSERT-only de upsertLUSRRatings.
// Sémantique identique : skip CSR, skip LUSR existant, calcul delta clipped,
// tier resolution. Différence : batch en RAM puis 1 DELETE+INSERT TX au lieu
// de N UPDATE-then-INSERT.
//
// Le breakdown LUSR (lusr_component_history) reste écrit row-by-row via
// writeLUSRComponentHistory (best-effort, pas critique côté ART).
func upsertLUSRRatingsBatch(
	ctx context.Context,
	playerDB *sql.DB,
	results []lusrResult,
	existingCSR, existingLUSR map[string]bool,
	seedRatings map[string]float64,
) (int, error) {
	now := time.Now().UTC()
	prevRating := make(map[string]float64)
	for pg, r := range seedRatings {
		prevRating[pg] = r
	}

	var (
		skippedByCSR int
	)
	rows := make([]persist.LUSRRatingInsert, 0, len(results))
	componentsPending := make([]lusrResult, 0, len(results))

	for _, r := range results {
		if existingCSR[r.MatchID] {
			skippedByCSR++
			continue
		}
		if existingLUSR[r.MatchID] {
			// Cohérent avec le legacy : si on n'est PAS en mode force, on
			// ne réécrit pas un LUSR existant. Le caller (batchComputeLUSR)
			// fournit existingLUSR vide en mode force.
			continue
		}

		ratingValue := r.RatingValue
		var delta *float64
		if prev, ok := prevRating[r.PlaylistGroup]; ok {
			rawDelta := ratingValue - prev
			if math.Abs(rawDelta) > LUSRMaxDelta {
				if rawDelta > 0 {
					rawDelta = LUSRMaxDelta
				} else {
					rawDelta = -LUSRMaxDelta
				}
				ratingValue = prev + rawDelta
			}
			delta = &rawDelta
		}
		prevRating[r.PlaylistGroup] = ratingValue

		tier, sub := GetTierForRating(ratingValue)
		var tierName, tierFR, tierLabel *string
		var subPtr *int
		if tier != nil {
			tierName = &tier.Name
			tierFR = &tier.NameFR
			label := FormatTierLabel(ratingValue)
			tierLabel = &label
			subVal := sub
			subPtr = &subVal
		}

		rows = append(rows, persist.LUSRRatingInsert{
			MatchID:         r.MatchID,
			RatingValue:     ratingValue,
			RatingDeviation: r.RatingDeviation,
			Tier:            tierName,
			TierFR:          tierFR,
			SubTier:         subPtr,
			TierLabel:       tierLabel,
			RatingDelta:     delta,
			PlaylistGroup:   r.PlaylistGroup,
		})
		componentsPending = append(componentsPending, r)
	}

	if len(rows) == 0 {
		if skippedByCSR > 0 {
			slog.InfoContext(ctx, "upsertLUSRRatingsBatch: aucun LUSR à persister",
				"skipped_by_csr_filter", skippedByCSR,
				"total_candidates", len(results))
		}
		return 0, nil
	}

	// Persist atomic batch via PostSyncLUSRPersister.
	persister := persist.NewPostSyncLUSRPersister(playerDB)
	if err := persister.Upsert(ctx, rows); err != nil {
		slog.ErrorContext(ctx, "upsertLUSRRatingsBatch: PostSyncLUSRPersister.Upsert échoué",
			"batch_size", len(rows), "err", err)
		return 0, err
	}

	// Breakdown LUSR (lusr_component_history) — best-effort row-by-row.
	// Cette table n'est PAS dans match_skill_rank donc pas touchée par
	// l'ART de cette PK. Écriture séparée acceptable.
	for _, r := range componentsPending {
		if len(r.Components) > 0 {
			if err := writeLUSRComponentHistory(ctx, playerDB, r.MatchID, r.Components, now); err != nil {
				slog.Warn("upsertLUSRRatingsBatch: lusr_component_history write failed",
					"match_id", r.MatchID, "err", err)
			}
		}
	}

	slog.InfoContext(ctx, "upsertLUSRRatingsBatch: batch terminé (INSERT-only path)",
		"updated", len(rows),
		"skipped_by_csr_filter", skippedByCSR,
		"total_candidates", len(results))
	return len(rows), nil
}
