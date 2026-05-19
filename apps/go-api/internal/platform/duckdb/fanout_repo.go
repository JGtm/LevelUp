// Package duckdb — fanout_repo.go : implémentation de port.FanoutRepository.
package duckdb

import (
	"context"
	"log/slog"
)

// FanoutRepo implémente port.FanoutRepository sur un PlayerDB ouvert.
type FanoutRepo struct {
	pdb *PlayerDB
}

// NewFanoutRepo crée un FanoutRepo à partir d'un PlayerDB.
func NewFanoutRepo(pdb *PlayerDB) *FanoutRepo {
	return &FanoutRepo{pdb: pdb}
}

// CountCommonMatchesForXUID compte les matchs de matchIDs
// où targetXUID était participant dans shared.match_participants.
func (r *FanoutRepo) CountCommonMatchesForXUID(
	ctx context.Context,
	targetXUID string,
	matchIDs []string,
) (int, error) {
	if len(matchIDs) == 0 {
		return 0, nil
	}
	query := `
		SELECT COUNT(DISTINCT match_id)
		FROM shared.match_participants
		WHERE xuid = ?
		AND match_id IN (SELECT UNNEST(?::VARCHAR[]))
	`
	db, release, err := r.pdb.SharedReadDB().Get(ctx)
	if err != nil {
		return 0, err
	}
	defer release()

	var count int
	err = db.QueryRowContext(ctx, query, targetXUID, matchIDs).Scan(&count)
	return count, err
}

// LoadExistingEnrichments retourne le set des match_ids déjà présents
// dans player_match_enrichment.
func (r *FanoutRepo) LoadExistingEnrichments(
	ctx context.Context,
	matchIDs []string,
) (map[string]bool, error) {
	result := make(map[string]bool, len(matchIDs))
	if len(matchIDs) == 0 {
		return result, nil
	}
	query := `
		SELECT match_id
		FROM player_match_enrichment
		WHERE match_id IN (SELECT UNNEST(?::VARCHAR[]))
	`
	rows, err := r.pdb.Player.Query(ctx, query, matchIDs)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		var mid string
		if err := rows.Scan(&mid); err != nil {
			return nil, err
		}
		result[mid] = true
	}
	return result, rows.Err()
}

// InsertStubEnrichments insère des enregistrements stub dans player_match_enrichment
// pour les matchs manquants. Le performance_score sera recalculé plus tard.
func (r *FanoutRepo) InsertStubEnrichments(
	ctx context.Context,
	xuid string,
	matchIDs []string,
) (int, error) { //nolint:unparam // error toujours nil actuellement, interface-compatible
	_ = xuid // disponible pour future extension

	inserted := 0
	for _, mid := range matchIDs {
		_, err := r.pdb.Player.Exec(ctx,
			`INSERT OR IGNORE INTO player_match_enrichment (match_id) VALUES (?)`,
			mid,
		)
		if err != nil {
			slog.Warn("fanout: insert stub échoué", "match_id", mid, "err", err)
			continue
		}
		inserted++
	}
	return inserted, nil
}
