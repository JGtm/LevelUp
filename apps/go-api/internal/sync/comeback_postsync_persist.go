// Package sync — comeback_postsync_persist.go : Phase 4.4 du refactor
// Collect→Persist — chemin batch pour BackfillDominanceFlags.
//
// **Activation** : env var LEVELUP_POSTSYNC_INSERT_ONLY=1.
//
// Pattern : accumule les (match_id, dominance_flag) calculés en RAM puis
// 1 single UPDATE multi-row via PostSyncEnrichmentPersister.BatchUpdateColumn
// au lieu de N UPDATE row-by-row.

package sync

import (
	"context"
	"database/sql"
	"log/slog"

	"levelup/go-api/internal/persist"
)

// backfillDominanceFlagsBatch est la variante batch de BackfillDominanceFlags.
// Sémantique identique mais 1 single UPDATE par cycle au lieu de N.
//
// La compute (computeMatchDominanceFlag) reste row-by-row car elle lit
// shared.match_participants + medals + events — beaucoup d'IO read difficile
// à batcher proprement. Le gain ART vient du write batch.
func backfillDominanceFlagsBatch(
	ctx context.Context,
	sharedDB, playerDB *sql.DB,
	xuid string,
	matchIDs []string,
) error {
	rows := make([]persist.EnrichmentColumnRow, 0, len(matchIDs))
	for _, matchID := range matchIDs {
		flag, err := computeMatchDominanceFlag(ctx, sharedDB, xuid, matchID)
		if err != nil {
			slog.WarnContext(ctx, "backfillDominanceFlagsBatch: compute",
				"match_id", matchID, "err", err)
			continue
		}
		rows = append(rows, persist.EnrichmentColumnRow{MatchID: matchID, Value: flag})
	}

	if len(rows) == 0 {
		return nil
	}

	// Seed placeholder rows (INSERT OR IGNORE) pour aligner avec la sémantique
	// legacy UPSERT — sinon le UPDATE batch ne touche que les rows existantes
	// et skip silencieusement les match_id manquants (cas tests + cas CLI
	// backfill --force sur DB neuve). En prod le path submitMatchAsBatch
	// pré-seed déjà ces rows, donc INSERT OR IGNORE = no-op.
	for _, r := range rows {
		if _, err := playerDB.ExecContext(ctx,
			`INSERT OR IGNORE INTO player_match_enrichment (match_id) VALUES (?)`,
			r.MatchID,
		); err != nil {
			slog.WarnContext(ctx, "backfillDominanceFlagsBatch: seed placeholder échoué",
				"match_id", r.MatchID, "err", err)
		}
	}

	persister := persist.NewPostSyncEnrichmentPersister(playerDB)
	err := persister.BatchUpdateColumn(ctx, persist.EnrichmentColumnUpdate{
		Column: "dominance_flag",
		Rows:   rows,
	})
	if err != nil {
		slog.ErrorContext(ctx, "backfillDominanceFlagsBatch: BatchUpdateColumn échoué",
			"batch_size", len(rows), "err", err)
		return err
	}

	slog.InfoContext(ctx, "backfillDominanceFlagsBatch: batch terminé (INSERT-only path)",
		"updated", len(rows), "total_candidates", len(matchIDs))
	return nil
}
