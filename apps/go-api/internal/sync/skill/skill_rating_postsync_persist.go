// Package sync — skill_rating_postsync_persist.go : chemin batch pour
// upsertLUSRRatings, basé sur le AppendOnlyLUSRPersister (Phase 2.C du
// plan d'éradication ART, cf. .ai/PLAN_LUSR_ART_HOME_CRASH.md).
//
// **Sémantique append-only** : chaque batch est un INSERT pur (pas de
// DELETE, pas d'UPDATE). La table match_skill_rank stocke N versions
// par (match_id, rating_type) ; la vue match_skill_rank_latest expose
// la version la plus récente.
//
// Logique métier identique au chemin legacy (compute delta, tier, etc.).
// Conserve les filtres :
//   - existingCSR : skip les matchs ranked CSR pour ne pas écrire un LUSR
//     sur un match déjà rated par Halo (la vue latest le préserverait,
//     mais on évite le bruit physique).
//   - existingLUSR : skip les LUSR déjà persistés en mode non-force.

package skill

import (
	"context"
	"database/sql"
	"log/slog"
	"math"
	"time"

	"levelup/go-api/internal/persist"
)

// NB : la compaction des versions superseded de match_skill_rank (ancien
// compactMatchSkillRankSuperseded : DELETE id NOT IN MAX(id)…) a été SUPPRIMÉE —
// elle déclenchait le bug ART DuckDB amont #23046 (crash JGtm 2026-06-20) malgré
// mono-writer + PK BIGINT. La table reste append-only pur ; la vue
// match_skill_rank_latest (MAX(id)) reste correcte avec les versions superseded.

// upsertLUSRRatingsBatch est la variante INSERT-only de upsertLUSRRatings.
// Sémantique identique : skip CSR, skip LUSR existant, calcul delta clipped,
// tier resolution. Différence : batch en RAM puis INSERT pur append-only via
// AppendOnlyLUSRPersister (pas de DELETE, pas de ON CONFLICT) au lieu de N
// UPDATE-then-INSERT — la version courante est lue via la vue match_skill_rank_latest.
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
			// ne réécrit pas un LUSR existant. Le caller (BatchComputeLUSRWithMedals)
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

	// Persist atomic batch via AppendOnlyLUSRPersister (INSERT pur, jamais
	// de DELETE → bug ART impossible par construction).
	persister := persist.NewAppendOnlyLUSRPersister(playerDB)
	if err := persister.Persist(ctx, rows); err != nil {
		slog.ErrorContext(ctx, "upsertLUSRRatingsBatch: AppendOnlyLUSRPersister.Persist échoué",
			"batch_size", len(rows), "err", err)
		return 0, err
	}

	// Breakdown LUSR (lusr_component_history) — best-effort row-by-row.
	// Cette table n'est PAS dans match_skill_rank donc pas touchée par
	// l'ART de cette PK. Écriture séparée acceptable.
	for _, r := range componentsPending {
		if len(r.Components) > 0 {
			if err := writeLUSRComponentHistory(ctx, playerDB, r.MatchID, r.Components, now); err != nil {
				slog.WarnContext(ctx, "upsertLUSRRatingsBatch: lusr_component_history write failed",
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
