// Package sync — backfill_personal_scores.go : pipeline rétroactif PSA.
//
// Pour chaque match dans la liste, fetch GetMatchStats Halo, extrait les
// PersonalScores du joueur courant (e.xuid), et écrit dans
// player.personal_score_awards.
//
// Le pipeline LIVE écrit déjà PSA via le batch Collect→Persist
// (buildBatchFromFetchedMatch → submitMatchAsBatch, cf. engine_batch_path.go).
// Cette fonction sert uniquement à rattraper les matchs synced
// avant que l'extraction ne soit câblée (PSA absent en avril/mai 2026 sur
// les player DBs).
package sync

import (
	"context"
	"fmt"
	"log/slog"

	"levelup/go-api/internal/platform/dblease"
)

// BackfillPersonalScoreAwardsForMatches traite une liste de matchs : fetch
// MatchStats, extrait PSA, insère dans la player DB. Idempotent (DELETE+INSERT
// dans InsertPersonalScoreAwards).
//
// Retourne (matchesProcessed, totalRowsInserted, error).
func (e *SyncEngine) BackfillPersonalScoreAwardsForMatches(
	ctx context.Context,
	matchIDs []string,
) (matches, rows int, err error) {
	if len(matchIDs) == 0 {
		return 0, 0, nil
	}
	if e.tokens == nil || e.tokens.SpartanToken == "" {
		return 0, 0, fmt.Errorf("BackfillPersonalScoreAwardsForMatches: tokens Halo absents")
	}

	writerPlayer, err := dblease.AcquireWriterCtx(ctx, nil, e.playerDBPath, dblease.KindPlayer)
	if err != nil {
		return 0, 0, fmt.Errorf("BackfillPersonalScoreAwardsForMatches lease player: %w", err)
	}
	defer writerPlayer.Release()

	playerHandle, err := OpenPlayerDB(e.playerDBPath)
	if err != nil {
		return 0, 0, fmt.Errorf("BackfillPersonalScoreAwardsForMatches OpenPlayerDB: %w", err)
	}
	defer playerHandle.Close()
	playerDB := playerHandle.SQLDb()

	client := NewHaloAPIClient(e.tokens.SpartanToken, e.tokens.ClearanceToken, 3)

	for _, matchID := range matchIDs {
		if ctx.Err() != nil {
			break
		}
		matchJSON, fetchErr := client.GetMatchStats(ctx, matchID)
		if fetchErr != nil {
			slog.WarnContext(ctx, "backfill_psa: GetMatchStats échoué",
				"match_id", matchID, "err", fetchErr,
			)
			continue
		}
		extracted := ExtractPersonalScoreAwards(matchJSON, matchID, e.xuid)
		if len(extracted) == 0 {
			// Aucun PSA pour ce joueur dans ce match (ex: DNF, ou stats privées).
			// On insère quand même un DELETE — pour idempotence — via
			// InsertPersonalScoreAwards avec rows vides.
			if err := InsertPersonalScoreAwards(ctx, playerDB, matchID, e.xuid, nil); err != nil {
				slog.WarnContext(ctx, "backfill_psa: clear échoué",
					"match_id", matchID, "err", err,
				)
			}
			matches++
			continue
		}
		if err := InsertPersonalScoreAwards(ctx, playerDB, matchID, e.xuid, extracted); err != nil {
			slog.WarnContext(ctx, "backfill_psa: InsertPersonalScoreAwards échoué",
				"match_id", matchID, "err", err,
			)
			continue
		}
		matches++
		rows += len(extracted)
	}
	return matches, rows, nil
}
