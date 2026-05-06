// Package sync — citations_backfill.go : orchestration RunBackfillCitations.
//
// Pendant à RunBackfillComebackBadges et RunBackfillEngagementScores : ouvre
// les leases player + shared, attache metadata.duckdb en lecture seule, puis
// délègue à BackfillMatchCitations (citations.go).
//
// Sélection des match_ids :
//   - force=true   : tous les matchs du joueur (player_match_enrichment)
//   - force=false  : matchs sans entrée dans match_citations (idempotent)
package sync

import (
	"context"
	"database/sql"
	"fmt"
	"log/slog"

	_ "github.com/duckdb/duckdb-go/v2"

	"levelup/go-api/internal/platform/dblease"
)

// RunBackfillCitations calcule et persiste les citations dans match_citations
// pour les matchs du joueur. Retourne le nombre de match_ids traités.
//
// Si force=true, supprime d'abord les citations existantes pour ces matchs ;
// sinon ne traite que les matchs qui n'ont aucune entrée dans match_citations.
func (e *SyncEngine) RunBackfillCitations(ctx context.Context, force bool) (int, error) {
	writerPlayer, err := dblease.AcquireWriterCtx(ctx, nil, e.playerDBPath, dblease.KindPlayer)
	if err != nil {
		return 0, fmt.Errorf("RunBackfillCitations lease player: %w", err)
	}
	defer writerPlayer.Release()

	writerShared, err := dblease.AcquireWriterCtx(ctx, nil, e.sharedDBPath, dblease.KindSharedMatches)
	if err != nil {
		return 0, fmt.Errorf("RunBackfillCitations lease shared: %w", err)
	}
	defer writerShared.Release()

	playerHandle, err := OpenPlayerDB(e.playerDBPath)
	if err != nil {
		return 0, fmt.Errorf("RunBackfillCitations OpenPlayerDB: %w", err)
	}
	defer playerHandle.Close()

	sharedHandle, err := OpenSharedDB(e.sharedDBPath)
	if err != nil {
		return 0, fmt.Errorf("RunBackfillCitations OpenSharedDB: %w", err)
	}
	defer sharedHandle.Close()

	metaDB, err := sql.Open("duckdb", e.metadataDBPath+"?access_mode=READ_ONLY")
	if err != nil {
		return 0, fmt.Errorf("RunBackfillCitations open metadata: %w", err)
	}
	defer metaDB.Close()
	metaDB.SetMaxOpenConns(1)

	matchIDs, err := selectMatchesForCitations(ctx, playerHandle.SQLDb(), force)
	if err != nil {
		return 0, fmt.Errorf("RunBackfillCitations select: %w", err)
	}
	if len(matchIDs) == 0 {
		slog.InfoContext(ctx, "citations: aucun match a traiter",
			"player", e.gamertag, "force", force)
		return 0, nil
	}

	if force {
		if err := deleteCitationsForMatches(ctx, playerHandle.SQLDb(), matchIDs); err != nil {
			return 0, fmt.Errorf("RunBackfillCitations delete force: %w", err)
		}
	}

	slog.InfoContext(ctx, "citations: backfill en cours",
		"player", e.gamertag, "match_count", len(matchIDs), "force", force)

	if err := BackfillMatchCitations(
		ctx, metaDB, sharedHandle.SQLDb(), playerHandle.SQLDb(),
		e.xuid, matchIDs,
	); err != nil {
		return 0, fmt.Errorf("RunBackfillCitations backfill: %w", err)
	}
	return len(matchIDs), nil
}

// selectMatchesForCitations retourne les match_ids candidats au backfill.
//
// force=true  : tous les matchs joués (player_match_enrichment).
// force=false : matchs absents de match_citations (LEFT JOIN IS NULL).
func selectMatchesForCitations(ctx context.Context, playerDB *sql.DB, force bool) ([]string, error) {
	var q string
	if force {
		q = `SELECT match_id FROM player_match_enrichment ORDER BY match_id`
	} else {
		q = `
SELECT pme.match_id
FROM player_match_enrichment pme
LEFT JOIN (SELECT DISTINCT match_id FROM match_citations) mc
  ON mc.match_id = pme.match_id
WHERE mc.match_id IS NULL
ORDER BY pme.match_id`
	}
	rows, err := playerDB.QueryContext(ctx, q)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var ids []string
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			return nil, err
		}
		ids = append(ids, id)
	}
	return ids, rows.Err()
}

// deleteCitationsForMatches purge match_citations pour la liste de matchs
// avant un recalcul forcé. Utilisé uniquement quand force=true.
func deleteCitationsForMatches(ctx context.Context, playerDB *sql.DB, matchIDs []string) error {
	if len(matchIDs) == 0 {
		return nil
	}
	const q = `DELETE FROM match_citations WHERE match_id = ?`
	for _, id := range matchIDs {
		if _, err := playerDB.ExecContext(ctx, q, id); err != nil {
			return fmt.Errorf("delete %s: %w", id, err)
		}
	}
	return nil
}

// runPostSyncCitations branche les citations dans le pipeline post-sync.
// Réutilise les DBs déjà ouvertes par runPostSyncPipeline (player + shared)
// au lieu d'acquérir de nouveaux leases. Best-effort : retourne (0, nil) si
// metadata.duckdb absent ou citation_mappings vide.
func (e *SyncEngine) runPostSyncCitations(ctx context.Context, playerDB, sharedDB *sql.DB) (int, error) {
	metaDB, err := sql.Open("duckdb", e.metadataDBPath+"?access_mode=READ_ONLY")
	if err != nil {
		return 0, fmt.Errorf("open metadata: %w", err)
	}
	defer metaDB.Close()
	metaDB.SetMaxOpenConns(1)

	matchIDs, err := selectMatchesForCitations(ctx, playerDB, false)
	if err != nil {
		return 0, fmt.Errorf("select: %w", err)
	}
	if len(matchIDs) == 0 {
		return 0, nil
	}
	if err := BackfillMatchCitations(ctx, metaDB, sharedDB, playerDB, e.xuid, matchIDs); err != nil {
		return 0, fmt.Errorf("backfill: %w", err)
	}
	return len(matchIDs), nil
}
