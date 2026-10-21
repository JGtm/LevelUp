// Package duckdb — SessionsRepo : chargement des données pour le calcul des sessions.
package duckdb

import (
	"context"
	"database/sql"
	"errors"
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

// LookupFriendXUIDs résout des gamertags d'amis en xuids depuis xuid_aliases (SharedReader),
// comme la synchro le fait avant de découper les sessions (sync.LookupFriendXUIDs). Un
// gamertag absent de xuid_aliases n'apparaît pas dans le résultat.
func (r *SessionsRepo) LookupFriendXUIDs(ctx context.Context, gamertags []string) ([]string, error) {
	if len(gamertags) == 0 {
		return nil, nil
	}
	sharedDB, release, err := r.pdb.SharedReadDB().Get(ctx)
	if err != nil {
		return nil, fmt.Errorf("SessionsRepo.LookupFriendXUIDs: shared reader: %w", err)
	}
	defer release()
	var xuids []string
	for _, gt := range gamertags {
		var xuid string
		err := sharedDB.QueryRowContext(ctx,
			"SELECT xuid FROM xuid_aliases WHERE LOWER(gamertag) = LOWER(?)", gt).Scan(&xuid)
		switch {
		case errors.Is(err, sql.ErrNoRows):
			continue
		case err != nil:
			return nil, fmt.Errorf("SessionsRepo.LookupFriendXUIDs %q: %w", gt, err)
		case xuid != "":
			xuids = append(xuids, xuid)
		}
	}
	return xuids, nil
}
