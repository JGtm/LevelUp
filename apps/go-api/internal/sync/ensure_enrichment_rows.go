// Package sync — ensure_enrichment_rows.go : crée les rows
// `player_match_enrichment` manquantes pour les matchs où le joueur a
// participé (présents dans `shared.match_participants`) mais pour lesquels
// aucun INSERT player n'a jamais eu lieu.
//
// CONTEXTE — incident 2026-05-27 :
//
//	Madina/Choco/XxDaemon ont joué leurs matchs ensemble avec JGtm. Le
//	watcher JGtm a sync les 8 matchs → INSERT dans shared.match_registry
//	+ shared.match_participants (rows pour les 4 joueurs). Mais pour
//	Madina/Choco/XxDaemon eux-mêmes, leur sync delta voit ces 8 match_ids
//	comme "déjà connus" (loadKnownMatchIDs source #2 = shared.match_participants
//	WHERE xuid=?) → arrête le delta → PlayerPersister.Persist jamais appelé
//	→ aucune row dans player_match_enrichment côté Madina/Choco/XxDaemon.
//
//	Le post-sync existant (batchComputePerformanceScores, upsertLUSRRatingsBatch,
//	writeSessionAssignmentsBatch, citations, dominance, etc.) est composé
//	exclusivement d'UPDATE — `affected:0` silencieux si la row n'existe pas.
//	Conséquence UI : page Match detail vide (citations, sessions, perf, weapon).
//
//	Le commentaire dans engine.go:622-631 affirmait que le post-sync
//	"UPSERT naturellement les rows enrichment manquantes" — c'est FAUX
//	(vérifié post_sync_enrichment_persister.go:83+158, deux UPDATE purs).
//
// FIX :
//
//	En début de runPostSyncPipeline, pour le joueur courant :
//	  INSERT INTO player_match_enrichment (match_id)
//	  SELECT DISTINCT mp.match_id
//	  FROM <shared.match_participants reader> mp
//	  WHERE mp.xuid = ?
//	    AND mp.match_id NOT IN (SELECT match_id FROM player_match_enrichment)
//
//	Idempotent : 0 row insérée si tous les enrichment existent déjà (cas
//	stationnaire JGtm). Crée les rows orphelines (cas Madina/Choco/XxDaemon).
//	Le post-sync UPDATE remplit ensuite naturellement les scores.
package sync

import (
	"context"
	"database/sql"
	"fmt"
	"log/slog"
)

// ensurePlayerEnrichmentRows crée une row `player_match_enrichment(match_id)`
// pour chaque match présent dans `shared.match_participants` pour ce xuid
// mais absent de player_match_enrichment.
//
// playerDB : conn RW vers la player DB (stats.duckdb du joueur).
// sharedDB : conn RO ou RW vers shared_matches_v2.duckdb. Si nil, la fonction
// est un no-op (rien à faire côté shared).
// xuid : xuid numérique du joueur courant (sans préfixe `xuid(...)`).
//
// Retourne le nombre de rows créées + erreur si la query DB échoue.
//
// La sémantique est volontairement narrow : on INSERT UNIQUEMENT le PK
// match_id, laissant tous les autres champs à NULL/DEFAULT. C'est le
// post-sync qui peuplera ensuite performance_score/session_id/etc. via
// ses UPDATE existants.
func ensurePlayerEnrichmentRows(
	ctx context.Context,
	playerDB *sql.DB,
	sharedDB *sql.DB,
	xuid string,
) (int, error) {
	if sharedDB == nil || xuid == "" {
		return 0, nil
	}

	// Lire les match_ids du joueur côté shared.
	rows, err := sharedDB.QueryContext(ctx, `
		SELECT DISTINCT match_id
		FROM match_participants
		WHERE xuid = ?
		  AND match_id IS NOT NULL
	`, xuid)
	if err != nil {
		return 0, fmt.Errorf("ensurePlayerEnrichmentRows: query shared.match_participants: %w", err)
	}
	defer rows.Close()

	var sharedMatchIDs []string
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			return 0, fmt.Errorf("ensurePlayerEnrichmentRows: scan shared: %w", err)
		}
		sharedMatchIDs = append(sharedMatchIDs, id)
	}
	if err := rows.Err(); err != nil {
		return 0, fmt.Errorf("ensurePlayerEnrichmentRows: rows.Err shared: %w", err)
	}

	if len(sharedMatchIDs) == 0 {
		return 0, nil
	}

	// Lire les match_ids déjà présents côté player.
	existing := make(map[string]struct{}, len(sharedMatchIDs))
	pmeRows, err := playerDB.QueryContext(ctx, `SELECT match_id FROM player_match_enrichment`)
	if err != nil {
		return 0, fmt.Errorf("ensurePlayerEnrichmentRows: query player_match_enrichment: %w", err)
	}
	for pmeRows.Next() {
		var id string
		if err := pmeRows.Scan(&id); err != nil {
			_ = pmeRows.Close()
			return 0, fmt.Errorf("ensurePlayerEnrichmentRows: scan player: %w", err)
		}
		existing[id] = struct{}{}
	}
	pmeErr := pmeRows.Err()
	_ = pmeRows.Close()
	if pmeErr != nil {
		return 0, fmt.Errorf("ensurePlayerEnrichmentRows: rows.Err player: %w", pmeErr)
	}

	// Calculer le delta (shared \ existing). Cap basé sur sharedMatchIDs (pas
	// le diff, qui peut être négatif si existing > sharedMatchIDs — cas joueur
	// avec gros historique player vs petite fenêtre shared).
	missing := make([]string, 0, len(sharedMatchIDs))
	for _, id := range sharedMatchIDs {
		if _, ok := existing[id]; !ok {
			missing = append(missing, id)
		}
	}
	if len(missing) == 0 {
		return 0, nil
	}

	// INSERT les rows manquantes en 1 transaction (atomicité + perf).
	tx, err := playerDB.BeginTx(ctx, nil)
	if err != nil {
		return 0, fmt.Errorf("ensurePlayerEnrichmentRows: BeginTx: %w", err)
	}
	defer func() { _ = tx.Rollback() }()

	const insertSQL = `INSERT INTO player_match_enrichment (match_id) VALUES (?)`
	stmt, err := tx.PrepareContext(ctx, insertSQL)
	if err != nil {
		return 0, fmt.Errorf("ensurePlayerEnrichmentRows: Prepare: %w", err)
	}
	defer stmt.Close()

	for _, id := range missing {
		if _, err := stmt.ExecContext(ctx, id); err != nil {
			// On continue en cas d'erreur ponctuelle (race avec un autre writer)
			// — Si l'INSERT échoue pour cause de PK conflict, c'est qu'un autre
			// path a créé la row entre temps. Log et continue.
			slog.WarnContext(ctx, "ensurePlayerEnrichmentRows: INSERT row failed",
				"match_id", id, "err", err)
			continue
		}
	}

	if err := tx.Commit(); err != nil {
		return 0, fmt.Errorf("ensurePlayerEnrichmentRows: Commit: %w", err)
	}

	return len(missing), nil
}
