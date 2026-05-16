// Package duckdb — MatchExclusionRepo : lecture/écriture du flag is_excluded.
package duckdb

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"levelup/go-api/internal/domain"
)

// MatchExclusionRepo implémente port.MatchExclusionRepository.
type MatchExclusionRepo struct {
	pdb *PlayerDB
}

// NewMatchExclusionRepo crée un MatchExclusionRepo depuis un PlayerDB.
func NewMatchExclusionRepo(pdb *PlayerDB) *MatchExclusionRepo {
	return &MatchExclusionRepo{pdb: pdb}
}

// SetExclusion positionne is_excluded pour un match donné (UPSERT).
// Ouvre la DB en lecture-écriture le temps de l'opération.
func (r *MatchExclusionRepo) SetExclusion(ctx context.Context, matchID string, excluded bool) error {
	ctx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	rwDB, err := OpenReadWrite(r.pdb.Player.Path())
	if err != nil {
		return fmt.Errorf("MatchExclusionRepo.SetExclusion: open rw: %w", err)
	}
	defer rwDB.Close()

	_, err = rwDB.Exec(ctx, `
		INSERT INTO player_match_enrichment (match_id, is_excluded, updated_at)
		VALUES (?, ?, NOW())
		ON CONFLICT (match_id) DO UPDATE SET
			is_excluded = EXCLUDED.is_excluded,
			updated_at  = NOW()
	`, matchID, excluded)
	if err != nil {
		return fmt.Errorf("MatchExclusionRepo.SetExclusion exec: %w", err)
	}
	return nil
}

// GetMatchRegistryInfo lit shared.match_registry pour un match donné.
// Renvoie domain.ErrMatchNotFound si le match n'existe pas.
func (r *MatchExclusionRepo) GetMatchRegistryInfo(ctx context.Context, matchID string) (domain.MatchRegistryInfo, error) {
	ctx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	var (
		info        domain.MatchRegistryInfo
		startTime   time.Time
		isRanked    sql.NullBool
		isFirefight sql.NullBool
		pairName    sql.NullString
	)
	err := r.pdb.ReadDB().QueryRow(ctx, `
		SELECT
			match_id,
			start_time,
			COALESCE(is_ranked, FALSE)    AS is_ranked,
			COALESCE(is_firefight, FALSE) AS is_firefight,
			COALESCE(pair_name, '')       AS pair_name
		FROM shared.match_registry
		WHERE match_id = ?
		LIMIT 1
	`, matchID).Scan(&info.MatchID, &startTime, &isRanked, &isFirefight, &pairName)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return domain.MatchRegistryInfo{}, domain.ErrMatchNotFound
		}
		return domain.MatchRegistryInfo{}, fmt.Errorf("MatchExclusionRepo.GetMatchRegistryInfo: %w", err)
	}
	info.StartTime = startTime
	if isRanked.Valid {
		info.IsRanked = isRanked.Bool
	}
	if isFirefight.Valid {
		info.IsFirefight = isFirefight.Bool
	}
	if pairName.Valid {
		info.PairName = pairName.String
	}
	return info, nil
}

// ListExcluded retourne les matchs marqués is_excluded = TRUE avec métadonnées.
func (r *MatchExclusionRepo) ListExcluded(ctx context.Context) ([]domain.ExcludedMatch, error) {
	ctx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()

	rows, err := r.pdb.ReadDB().Query(ctx, `
		SELECT
			pme.match_id,
			COALESCE(
				r.start_time_utc,
				r.start_time AT TIME ZONE 'UTC',
				pme.updated_at AT TIME ZONE 'UTC'
			)                                        AS start_time,
			COALESCE(r.map_name,   '')               AS map_name,
			COALESCE(r.pair_name,  '')               AS mode_name
		FROM player_match_enrichment pme
		LEFT JOIN shared.match_registry r ON pme.match_id = r.match_id
		WHERE pme.is_excluded = TRUE
		ORDER BY start_time DESC
	`)
	if err != nil {
		return nil, fmt.Errorf("MatchExclusionRepo.ListExcluded: %w", err)
	}
	defer rows.Close()

	var results []domain.ExcludedMatch
	for rows.Next() {
		var m domain.ExcludedMatch
		if err := rows.Scan(&m.MatchID, &m.StartTime, &m.MapName, &m.ModeName); err != nil {
			return nil, fmt.Errorf("MatchExclusionRepo.ListExcluded scan: %w", err)
		}
		results = append(results, m)
	}
	return results, rows.Err()
}
