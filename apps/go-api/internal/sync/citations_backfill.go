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
	duckdbpkg "levelup/go-api/internal/platform/duckdb"
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

	playerHandle, err := OpenPlayerDB(e.playerDBPath)
	if err != nil {
		return 0, fmt.Errorf("RunBackfillCitations OpenPlayerDB: %w", err)
	}
	defer playerHandle.Close()

	// Sprint B1 commit 13a : acquireSharedWriter centralise lease + open.
	sharedDB, releaseShared, err := e.acquireSharedWriter(ctx)
	if err != nil {
		return 0, fmt.Errorf("RunBackfillCitations: %w", err)
	}
	defer releaseShared()

	// Phase 2 du PLAN_FIX_SYNC_RELIABILITY_2026-05-24 : cache duckdbpkg (DSN aligne).
	metaHandle, err := duckdbpkg.OpenReadOnly(e.metadataDBPath)
	if err != nil {
		return 0, fmt.Errorf("RunBackfillCitations open metadata: %w", err)
	}
	defer metaHandle.Close()
	metaDB := metaHandle.SQLDb()

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
		ctx, metaDB, sharedDB, playerHandle.SQLDb(),
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
// uniquement sur les valeurs feuilles déjà présentes dans match_citations.
//
// Outil de secours (rescue) : ne relit pas les stats/médailles depuis shared.
// Sémantique correcte : un enfant composite déclenche le parent (+1) uniquement
// quand son cumulatif franchit son palier final dans ce match (même règle que le
// moteur principal ComputeCompositeTransitions, R4-R7).
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

	// Phase 2 : cache duckdbpkg (DSN aligne).
	metaHandle, err := duckdbpkg.OpenReadOnly(e.metadataDBPath)
	if err != nil {
		return 0, fmt.Errorf("RunBackfillCompositeOnlyCitations open metadata: %w", err)
	}
	defer metaHandle.Close()
	metaDB := metaHandle.SQLDb()

	sharedDB, err := sql.Open("duckdb", e.sharedDBPath+"?access_mode=READ_ONLY")
	if err != nil {
		return 0, fmt.Errorf("RunBackfillCompositeOnlyCitations open shared: %w", err)
	}
	defer sharedDB.Close()
	sharedDB.SetMaxOpenConns(1)

	mappings, err := loadFullCitationMappings(ctx, metaDB)
	if err != nil {
		return 0, fmt.Errorf("RunBackfillCompositeOnlyCitations mappings: %w", err)
	}
	if len(mappings) == 0 {
		slog.InfoContext(ctx, "composite-only: aucun mapping — skip", "player", e.gamertag)
		return 0, nil
	}

	compositeNames := buildCompositeNameSet(mappings)

	// tierMax par citation_name_norm (max(tier_targets), 0 si absent).
	tierMax := make(map[string]int, len(mappings))
	for _, m := range mappings {
		tierMax[m.NameNorm] = analysis.ParseTierMax(m.TierTargets)
	}

	// Charge toutes les citations feuilles depuis match_citations, hors composites.
	nonCompositesPerMatch, err := loadNonCompositeCitationsByMatch(ctx, playerHandle.SQLDb(), compositeNames)
	if err != nil {
		return 0, fmt.Errorf("RunBackfillCompositeOnlyCitations load data: %w", err)
	}
	if len(nonCompositesPerMatch) == 0 {
		slog.InfoContext(ctx, "composite-only: aucune donnée dans match_citations", "player", e.gamertag)
		return 0, nil
	}

	// Tri chrono des matchIDs pour que le cumulPre soit exact entre les passes.
	allMatchIDs := make([]string, 0, len(nonCompositesPerMatch))
	for id := range nonCompositesPerMatch {
		allMatchIDs = append(allMatchIDs, id)
	}
	sorted, err := sortMatchIDsChrono(ctx, sharedDB, allMatchIDs)
	if err != nil {
		slog.WarnContext(ctx, "composite-only: sort chrono failed, ordre non garanti", "err", err)
		sorted = allMatchIDs
	}

	slog.InfoContext(ctx, "composite-only: recalcul démarré",
		"player", e.gamertag, "matches", len(sorted), "composites", len(compositeNames))

	cumulPre := make(map[string]int)
	written := 0

	for _, matchID := range sorted {
		leafDeltas := nonCompositesPerMatch[matchID]

		// cumulPost feuilles = cumulPre + deltas ce match.
		cumulPost := make(map[string]int, len(cumulPre)+len(leafDeltas))
		for k, v := range cumulPre {
			cumulPost[k] = v
		}
		for k, v := range leafDeltas {
			cumulPost[k] += v
		}

		compositeDeltas := analysis.ComputeCompositeTransitions(cumulPre, cumulPost, tierMax, mappings)

		// Supprimer les anciens composites pour ce match, réécrire les nouveaux.
		if err := deleteCompositeCitationsForMatch(ctx, playerHandle.SQLDb(), matchID, compositeNames); err != nil {
			slog.WarnContext(ctx, "composite-only: delete", "match_id", matchID, "err", err)
		}

		var deltas []domain.CitationMatchDelta
		for norm, v := range compositeDeltas {
			deltas = append(deltas, domain.CitationMatchDelta{NameNorm: norm, Value: v})
		}
		if len(deltas) > 0 {
			if err := writeCitations(ctx, playerHandle.SQLDb(), matchID, deltas); err != nil {
				return written, fmt.Errorf("composite-only write %s: %w", matchID, err)
			}
			written++
		}

		// Mise à jour cumulPre pour le match suivant.
		for k, v := range leafDeltas {
			cumulPre[k] += v
		}
		for k, v := range compositeDeltas {
			cumulPre[k] += v
		}
	}

	slog.InfoContext(ctx, "composite-only: terminé",
		"player", e.gamertag, "matches_updated", written)
	return written, nil
}

// deleteCompositeCitationsForMatch supprime les citations composites d'un match
// avant réécriture (rescue tool — on ne touche pas les feuilles déjà correctes).
func deleteCompositeCitationsForMatch(ctx context.Context, db *sql.DB, matchID string, compositeNames map[string]struct{}) error {
	for norm := range compositeNames {
		if _, err := db.ExecContext(ctx,
			`DELETE FROM match_citations WHERE match_id = ? AND citation_name_norm = ?`,
			matchID, norm,
		); err != nil {
			return err
		}
	}
	return nil
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
	// Phase 2 du PLAN_FIX_SYNC_RELIABILITY_2026-05-24 : cache duckdbpkg (DSN aligne).
	// Site previously failing in production avec "Can't open a connection with
	// a different configuration" pendant le post-sync (citations_computed=0).
	metaHandle, err := duckdbpkg.OpenReadOnly(e.metadataDBPath)
	if err != nil {
		return 0, fmt.Errorf("open metadata: %w", err)
	}
	defer metaHandle.Close()
	metaDB := metaHandle.SQLDb()

	matchIDs, err := selectMatchesForCitations(ctx, playerDB, false)
	if err != nil {
		return 0, fmt.Errorf("select: %w", err)
	}
	if len(matchIDs) == 0 {
		slog.DebugContext(ctx, "citations post-sync: aucun nouveau match", "player", e.gamertag)
		return 0, nil
	}
	slog.InfoContext(ctx, "citations post-sync: nouveaux matchs détectés",
		"player", e.gamertag, "count", len(matchIDs))
	if err := BackfillMatchCitations(ctx, metaDB, sharedDB, playerDB, e.xuid, matchIDs); err != nil {
		return 0, fmt.Errorf("backfill: %w", err)
	}
	return len(matchIDs), nil
}
