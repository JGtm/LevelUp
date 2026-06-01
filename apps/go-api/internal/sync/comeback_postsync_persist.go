// Package sync — comeback_postsync_persist.go : Phase 4.4 du refactor
// Collect→Persist — chemin batch pour BackfillDominanceFlags. C'est désormais
// l'unique chemin (l'ancien flag LEVELUP_POSTSYNC_INSERT_ONLY a été supprimé
// en Phase 3, cf. comeback.go).
//
// Pattern : accumule les (match_id, dominance_flag) calculés en RAM puis les
// écrit via PostSyncEnrichmentPersister.BatchUpdateColumn — N UPDATE
// row-by-row dans 1 transaction. PAS un UPDATE multi-row : la forme bulk
// `UPDATE … FROM (VALUES …)` est précisément le déclencheur ART proscrit
// (1 statement touchant N entrées de l'index). La sûreté vient du row-by-row
// (1 entrée ART/statement) + single-writer (dblease KindPlayer), pas d'un
// hypothétique "single UPDATE".

package sync

import (
	"context"
	"database/sql"
	"log/slog"

	"levelup/go-api/internal/persist"
)

// backfillDominanceFlagsBatch est la variante batch de BackfillDominanceFlags.
// Sémantique identique ; les writes sont regroupés dans 1 transaction via
// BatchUpdateColumn (N UPDATE row-by-row), au lieu de N transactions séparées.
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
