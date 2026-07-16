// Package duckdb — SessionsRepo : chargement des données pour le calcul des sessions.
package duckdb

import (
	"context"
	"fmt"
	"time"

	"levelup/go-api/internal/domain"
)

// SessionsRepo charge les données brutes pour le calcul des sessions.
type SessionsRepo struct {
	pdb *PlayerDB
}

// NewSessionsRepo crée un SessionsRepo depuis un PlayerDB.
func NewSessionsRepo(pdb *PlayerDB) *SessionsRepo {
	return &SessionsRepo{pdb: pdb}
}

// LoadSessionMatches charge les matchs d'un joueur pour le calcul des sessions.
// Exécutée sur SharedReader (ADR 0016, shared-only).
func (r *SessionsRepo) LoadSessionMatches(ctx context.Context) ([]domain.SessionMatchRow, error) {
	ctx, cancel := context.WithTimeout(ctx, 60*time.Second)
	defer cancel()

	sharedDB, release, err := r.pdb.SharedReadDB().Get(ctx)
	if err != nil {
		return nil, fmt.Errorf("SessionsRepo.LoadSessionMatches: shared reader: %w", err)
	}
	defer release()
	q := resolveCampaignExclusion(Q22SessionMatches, r.pdb.TitleSlug, "r")
	rows, err := sharedDB.QueryContext(ctx, q, r.pdb.XUID, r.pdb.XUID)
	if err != nil {
		return nil, fmt.Errorf("SessionsRepo.LoadSessionMatches: %w", err)
	}
	defer rows.Close()

	var results []domain.SessionMatchRow
	for rows.Next() {
		var m domain.SessionMatchRow
		var endTime *time.Time
		if err := rows.Scan(
			&m.MatchID,
			&m.StartTime,
			&m.TeammatesSig,
			&m.IsRanked,
			&m.TimePlayedSecs,
			&endTime,
		); err != nil {
			return nil, fmt.Errorf("SessionsRepo.LoadSessionMatches scan: %w", err)
		}
		m.EndTime = endTime
		results = append(results, m)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("SessionsRepo.LoadSessionMatches rows: %w", err)
	}
	return results, nil
}
