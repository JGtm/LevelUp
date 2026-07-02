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
	"levelup/go-api/internal/ctxkeys"
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
	sharedDB, releaseShared, err := e.acquireSharedWriter(ctxkeys.WithDBWriterLabel(ctx, "backfill_citations"))
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
		// Recréer la table plutôt que N DELETE individuels : évite le bug ART DuckDB
		// qui survient quand des lignes corrompues (value IS NULL) bloquent la suppression
		// par index. DROP + CREATE est équivalent à un TRUNCATE propre.
		if err := recreateCitationsTable(ctx, playerHandle.SQLDb()); err != nil {
			return 0, fmt.Errorf("RunBackfillCitations recreate force: %w", err)
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
		q = `SELECT match_id FROM player_match_enrichment_latest ORDER BY match_id`
	} else {
		q = `
SELECT pme.match_id
FROM player_match_enrichment_latest pme
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

// recreateCitationsTable supprime et recrée match_citations à vide.
// Utilisé pour force=true à la place de N DELETE individuels, car le bug ART DuckDB
// peut bloquer la suppression via index quand des lignes corrompues (value IS NULL)
// sont présentes. DROP + CREATE évite entièrement le chemin de l'index ART.
func recreateCitationsTable(ctx context.Context, db *sql.DB) error {
	stmts := []string{
		`DROP TABLE IF EXISTS match_citations`,
		`CREATE TABLE match_citations (
			match_id            VARCHAR NOT NULL,
			citation_name_norm  VARCHAR NOT NULL,
			value               INTEGER DEFAULT 1,
			PRIMARY KEY (match_id, citation_name_norm)
		)`,
	}
	for _, s := range stmts {
		if _, err := db.ExecContext(ctx, s); err != nil {
			return err
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

	// Phase 2 PLAN_FIX_SYNC_RELIABILITY_2026-05-24 (audit residuel 2026-05-25) :
	// sharedDB via cache duckdbpkg.OpenReadOnly pour DSN aligne.
	sharedHandle, err := duckdbpkg.OpenReadOnly(e.sharedDBPath)
	if err != nil {
		return 0, fmt.Errorf("RunBackfillCompositeOnlyCitations open shared: %w", err)
	}
	defer sharedHandle.Close()
	sharedDB := sharedHandle.SQLDb()

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

		// Append-only #23046 (Phase 2) : plus de DELETE composites + write partiel.
		// On écrit une génération COMPLÈTE (feuilles préservées à l'identique +
		// composites recalculés) ; writeCitations alloue une nouvelle génération qui
		// supersède l'ancienne via match_citations_latest. Si l'ensemble est vide, la
		// sentinelle '_processed' (writeCitations) marque le match.
		var deltas []domain.CitationMatchDelta
		for norm, v := range leafDeltas {
			deltas = append(deltas, domain.CitationMatchDelta{NameNorm: norm, Value: v})
		}
		for norm, v := range compositeDeltas {
			deltas = append(deltas, domain.CitationMatchDelta{NameNorm: norm, Value: v})
		}
		if err := writeCitations(ctx, playerHandle.SQLDb(), matchID, deltas); err != nil {
			return written, fmt.Errorf("composite-only write %s: %w", matchID, err)
		}
		written++

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
FROM match_citations_latest
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
	// Emprunte la connexion metadata déjà ouverte dans le cache process (RW par
	// le pool serveur). OpenReadOnly échoue si une connexion RW existe déjà sur
	// le même fichier ("Can't open a connection with a different configuration").
	// LookupCachedDB retourne RW en priorité, RO sinon — sans incrémenter le
	// refCount, donc pas de Close() ici.
	var metaDB *sql.DB
	if e.metaDB != nil {
		metaDB = e.metaDB
	} else if cached, ok := duckdbpkg.LookupCachedDB(e.metadataDBPath); ok {
		metaDB = cached.SQLDb()
	} else {
		slog.WarnContext(ctx, "citations post-sync: metadata DB non disponible — skip", "player", e.gamertag)
		return 0, nil
	}

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
