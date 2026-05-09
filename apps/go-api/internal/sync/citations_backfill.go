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

	"levelup/go-api/internal/analysis"
	"levelup/go-api/internal/domain"
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

// RunBackfillCompositeOnlyCitations recalcule les citations composites en s'appuyant
// uniquement sur les valeurs déjà présentes dans match_citations (pas de lecture shared).
// Toujours non-destructif : INSERT ... ON CONFLICT DO NOTHING.
//
// Logique per-match : un enfant "contribue" dès que val > 0 (les tier_targets sont des
// seuils cumulatifs affichage, pas des filtres per-match). Gère les composites imbriqués
// via passes répétées (ex: all_weapons_mastery → human_weapons_mastery → br75_mastery).
func (e *SyncEngine) RunBackfillCompositeOnlyCitations(ctx context.Context) (int, error) {
	writer, err := dblease.AcquireWriterCtx(ctx, nil, e.playerDBPath, dblease.KindPlayer)
	if err != nil {
		return 0, fmt.Errorf("RunBackfillCompositeOnlyCitations lease: %w", err)
	}
	defer writer.Release()

	playerHandle, err := OpenPlayerDB(e.playerDBPath)
	if err != nil {
		return 0, fmt.Errorf("RunBackfillCompositeOnlyCitations OpenPlayerDB: %w", err)
	}
	defer playerHandle.Close()

	metaDB, err := sql.Open("duckdb", e.metadataDBPath+"?access_mode=READ_ONLY")
	if err != nil {
		return 0, fmt.Errorf("RunBackfillCompositeOnlyCitations open metadata: %w", err)
	}
	defer metaDB.Close()
	metaDB.SetMaxOpenConns(1)

	mappings, err := loadFullCitationMappings(ctx, metaDB)
	if err != nil {
		return 0, fmt.Errorf("RunBackfillCompositeOnlyCitations mappings: %w", err)
	}
	if len(mappings) == 0 {
		slog.InfoContext(ctx, "composite-only: aucun mapping — skip", "player", e.gamertag)
		return 0, nil
	}

	compositeNames := buildCompositeNameSet(mappings)

	// Charge toutes les valeurs non-composites de match_citations (leaf citations).
	// Les composites existants sont exclus pour éviter d'utiliser des données obsolètes.
	nonCompositesPerMatch, err := loadNonCompositeCitationsByMatch(ctx, playerHandle.SQLDb(), compositeNames)
	if err != nil {
		return 0, fmt.Errorf("RunBackfillCompositeOnlyCitations load data: %w", err)
	}
	if len(nonCompositesPerMatch) == 0 {
		slog.InfoContext(ctx, "composite-only: aucune donnée dans match_citations", "player", e.gamertag)
		return 0, nil
	}

	written := 0
	for matchID, totals := range nonCompositesPerMatch {
		// Multi-pass : gère composites imbriqués (all_weapons → human_weapons → br75).
		analysis.ApplyCompositeCitationsPerMatch(totals, mappings)
		var deltas []domain.CitationMatchDelta
		for _, m := range mappings {
			if m.MappingType != "composite" {
				continue
			}
			if v := totals[m.NameNorm]; v > 0 {
				deltas = append(deltas, domain.CitationMatchDelta{NameNorm: m.NameNorm, Value: v})
			}
		}
		if len(deltas) == 0 {
			continue
		}
		if err := writeCitations(ctx, playerHandle.SQLDb(), matchID, deltas); err != nil {
			return written, fmt.Errorf("composite-only write %s: %w", matchID, err)
		}
		written++
	}

	slog.InfoContext(ctx, "composite-only: terminé",
		"player", e.gamertag, "matches_updated", written)
	return written, nil
}

// buildCompositeNameSet retourne l'ensemble des citation_name_norm de type composite.
func buildCompositeNameSet(mappings []domain.CitationFullMapping) map[string]struct{} {
	s := make(map[string]struct{})
	for _, m := range mappings {
		if m.MappingType == "composite" {
			s[m.NameNorm] = struct{}{}
		}
	}
	return s
}

// loadNonCompositeCitationsByMatch charge toutes les valeurs non-composites (val > 0)
// depuis match_citations. Les composites existants sont exclus afin de repartir des
// données feuilles pour le recalcul.
// Retourne map[match_id]map[citation_name_norm]value.
func loadNonCompositeCitationsByMatch(ctx context.Context, db *sql.DB, compositeNames map[string]struct{}) (map[string]map[string]int, error) {
	rows, err := db.QueryContext(ctx, `
SELECT match_id, citation_name_norm, value
FROM match_citations
WHERE value > 0`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	result := make(map[string]map[string]int)
	for rows.Next() {
		var matchID, nameNorm string
		var value int
		if err := rows.Scan(&matchID, &nameNorm, &value); err != nil {
			return nil, err
		}
		if _, isComposite := compositeNames[nameNorm]; isComposite {
			continue // recalculé depuis les feuilles
		}
		if result[matchID] == nil {
			result[matchID] = make(map[string]int)
		}
		result[matchID][nameNorm] = value
	}
	return result, rows.Err()
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
