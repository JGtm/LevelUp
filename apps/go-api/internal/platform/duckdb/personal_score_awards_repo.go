// Package duckdb — personal_score_awards_repo.go : implementation DuckDB du
// loader aggregat des awards par (xuid, match_id, award_name)
// (port.PersonalScoreAwardsRepository).
//
// Source : table `personal_score_awards` côté player DB (peuplée par
// le sync via Personal Scores API). Schéma : (id, match_id, xuid,
// award_name, award_count). Cf. internal/sync/citations.go ligne ~310 pour
// le format exact.
//
// Capability gating : vérifie l'existence de personal_score_awards via
// information_schema.tables. Si absente -> games.ErrCapabilityNotSupported.
//
// Usage MV4.B : consommé par MatchView/Squad pour le radar 6 axes via
// narrative.ComputeParticipationProfile (mapping award_name -> axis fait
// par le service amont).
package duckdb

import (
	"context"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"levelup/go-api/internal/games"
	"levelup/go-api/internal/port"
)

// PersonalScoreAwardsRepo implemente port.PersonalScoreAwardsRepository.
type PersonalScoreAwardsRepo struct {
	pdb *PlayerDB
}

// NewPersonalScoreAwardsRepo cree un repo lié à un PlayerDB.
func NewPersonalScoreAwardsRepo(pdb *PlayerDB) *PersonalScoreAwardsRepo {
	return &PersonalScoreAwardsRepo{pdb: pdb}
}

// LoadPersonalScoreAwards charge les awards aggregés par
// (xuid, match_id, award_name) pour les matchs et joueurs spécifiés.
//
// Filtres MatchIDs et XUIDs requis (Validate strict).
func (r *PersonalScoreAwardsRepo) LoadPersonalScoreAwards(
	ctx context.Context,
	slug string,
	filters port.PersonalScoreAwardsFilters,
) ([]port.PersonalScoreAwardRow, error) {
	if err := filters.Validate(); err != nil {
		return nil, fmt.Errorf("PersonalScoreAwardsRepo.LoadPersonalScoreAwards: %w", err)
	}

	ctx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()

	if !r.awardsTableExists(ctx) {
		slog.DebugContext(ctx, "PersonalScoreAwardsRepo: personal_score_awards missing",
			"slug", slug,
			"match_count", len(filters.MatchIDs),
			"xuid_count", len(filters.XUIDs))
		return nil, games.ErrCapabilityNotSupported
	}

	q, args := buildAwardsQuery(filters)
	dbRows, err := r.pdb.ReadDB().QueryRecovered(ctx, q, args...)
	if err != nil {
		slog.ErrorContext(ctx, "PersonalScoreAwardsRepo: query failed",
			"slug", slug,
			"match_count", len(filters.MatchIDs),
			"err", err)
		return nil, fmt.Errorf("PersonalScoreAwardsRepo.LoadPersonalScoreAwards: query: %w", err)
	}
	defer dbRows.Close()

	var out []port.PersonalScoreAwardRow
	for dbRows.Next() {
		var row port.PersonalScoreAwardRow
		if err := dbRows.Scan(&row.XUID, &row.MatchID, &row.AwardName, &row.Total); err != nil {
			return nil, fmt.Errorf("PersonalScoreAwardsRepo.LoadPersonalScoreAwards: scan: %w", err)
		}
		out = append(out, row)
	}
	if err := dbRows.Err(); err != nil {
		return nil, fmt.Errorf("PersonalScoreAwardsRepo.LoadPersonalScoreAwards: rows: %w", err)
	}
	return out, nil
}

// buildAwardsQuery compose le SELECT avec GROUP BY pour agréger les
// award_count sur le triplet (xuid, match_id, award_name).
func buildAwardsQuery(f port.PersonalScoreAwardsFilters) (string, []any) {
	args := make([]any, 0, len(f.MatchIDs)+len(f.XUIDs))

	matchPH := Placeholders(len(f.MatchIDs))
	for _, id := range f.MatchIDs {
		args = append(args, id)
	}
	xuidPH := Placeholders(len(f.XUIDs))
	for _, x := range f.XUIDs {
		args = append(args, x)
	}

	var sb strings.Builder
	sb.WriteString(`
SELECT
    psa.xuid,
    psa.match_id,
    psa.award_name,
    COALESCE(SUM(psa.award_count), 0)::INTEGER AS total
FROM personal_score_awards_latest psa
WHERE psa.match_id IN (`)
	sb.WriteString(matchPH)
	sb.WriteString(`)
  AND psa.xuid IN (`)
	sb.WriteString(xuidPH)
	sb.WriteString(`)
GROUP BY psa.xuid, psa.match_id, psa.award_name
ORDER BY psa.xuid, psa.match_id, psa.award_name`)

	return sb.String(), args
}

// awardsTableExists vérifie la présence de personal_score_awards.
func (r *PersonalScoreAwardsRepo) awardsTableExists(ctx context.Context) bool {
	if r.pdb == nil || r.pdb.ReadDB() == nil {
		return false
	}
	var count int
	rows, err := r.pdb.ReadDB().QueryRowRecovered(ctx, `
		SELECT COUNT(*)
		FROM information_schema.tables
		WHERE table_name = 'personal_score_awards'
	`)
	if err != nil {
		return false
	}
	defer rows.Close()
	if err := rows.Scan(&count); err != nil {
		return false
	}
	return count > 0
}
