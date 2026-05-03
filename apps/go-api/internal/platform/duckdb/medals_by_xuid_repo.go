// Package duckdb — medals_by_xuid_repo.go : implementation DuckDB du loader
// medals par (xuid, match_id) (port.MedalsByXUIDRepository).
//
// Source : shared.medals_earned. La colonne stockee est medal_name_id (BIGINT
// dans le schema actuel — cf. internal/migration/steps_shared.go), exposee
// au port comme MedalID pour rester aligne avec le contract.
//
// Capability gating : verifie l'existence de shared.medals_earned via
// information_schema.tables. Si absente -> games.ErrCapabilityNotSupported.
//
// Labels medailles : non resolus dans le repo. Le service appelant peut
// charger les libelles via CitationsRepo.LoadMedalCitationMappings (ou
// equivalent) puis fusionner les rows avec leur Label. Decision : la jointure
// metadata.citation_mappings n'est pas faisable directement en SQL (DB
// separee, pas d'ATTACH dans ce repo) et la liste des medailles peut etre
// volumineuse (top-N par squad) — laisser le service decider du cache de
// labels evite un round-trip metadata par appel.
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

// MedalsByXUIDRepo implemente port.MedalsByXUIDRepository.
type MedalsByXUIDRepo struct {
	pdb *PlayerDB
}

// NewMedalsByXUIDRepo cree un MedalsByXUIDRepo lie a un PlayerDB.
func NewMedalsByXUIDRepo(pdb *PlayerDB) *MedalsByXUIDRepo {
	return &MedalsByXUIDRepo{pdb: pdb}
}

// LoadMedalsForMatchesByXUID charge les medailles pour des (xuid, match_id)
// fermes. Les filtres MatchIDs et XUIDs sont requis (Validate strict).
func (r *MedalsByXUIDRepo) LoadMedalsForMatchesByXUID(
	ctx context.Context,
	slug string,
	filters port.MedalsByXUIDFilters,
) ([]port.MedalRow, error) {
	if err := filters.Validate(); err != nil {
		return nil, fmt.Errorf("MedalsByXUIDRepo.LoadMedalsForMatchesByXUID: %w", err)
	}

	ctx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()

	if !r.medalsEarnedTableExists(ctx) {
		slog.DebugContext(ctx, "MedalsByXUIDRepo: shared.medals_earned missing",
			"slug", slug,
			"match_count", len(filters.MatchIDs),
			"xuid_count", len(filters.XUIDs))
		return nil, games.ErrCapabilityNotSupported
	}

	q, args := buildMedalsByXUIDQuery(filters)
	dbRows, err := r.pdb.ReadDB().Query(ctx, q, args...)
	if err != nil {
		slog.ErrorContext(ctx, "MedalsByXUIDRepo: query failed",
			"slug", slug,
			"match_count", len(filters.MatchIDs),
			"err", err)
		return nil, fmt.Errorf("MedalsByXUIDRepo.LoadMedalsForMatchesByXUID: query: %w", err)
	}
	defer dbRows.Close()

	var out []port.MedalRow
	for dbRows.Next() {
		var row port.MedalRow
		if err := dbRows.Scan(&row.XUID, &row.MatchID, &row.MedalID, &row.Count); err != nil {
			return nil, fmt.Errorf("MedalsByXUIDRepo.LoadMedalsForMatchesByXUID: scan: %w", err)
		}
		out = append(out, row)
	}
	if err := dbRows.Err(); err != nil {
		return nil, fmt.Errorf("MedalsByXUIDRepo.LoadMedalsForMatchesByXUID: rows: %w", err)
	}
	return out, nil
}

// buildMedalsByXUIDQuery compose le SELECT avec IN-list parametrees pour
// MatchIDs et XUIDs. Tri stable (xuid, match_id, count desc) pour faciliter
// le top-N cote service.
func buildMedalsByXUIDQuery(f port.MedalsByXUIDFilters) (string, []any) {
	args := make([]any, 0, len(f.MatchIDs)+len(f.XUIDs))

	matchPH := placeholders(len(f.MatchIDs))
	for _, id := range f.MatchIDs {
		args = append(args, id)
	}
	xuidPH := placeholders(len(f.XUIDs))
	for _, x := range f.XUIDs {
		args = append(args, x)
	}

	var sb strings.Builder
	sb.WriteString(`
SELECT
    me.xuid,
    me.match_id,
    me.medal_name_id::BIGINT AS medal_id,
    me.count
FROM shared.medals_earned me
WHERE me.match_id IN (`)
	sb.WriteString(matchPH)
	sb.WriteString(`)
  AND me.xuid IN (`)
	sb.WriteString(xuidPH)
	sb.WriteString(`)
ORDER BY me.xuid, me.match_id, me.count DESC`)

	if f.Limit > 0 {
		sb.WriteString(`
LIMIT ?`)
		args = append(args, f.Limit)
	}

	return sb.String(), args
}

// medalsEarnedTableExists verifie la presence de shared.medals_earned.
func (r *MedalsByXUIDRepo) medalsEarnedTableExists(ctx context.Context) bool {
	if r.pdb == nil || r.pdb.ReadDB() == nil {
		return false
	}
	var count int
	err := r.pdb.ReadDB().QueryRow(ctx, `
		SELECT COUNT(*)
		FROM information_schema.tables
		WHERE table_catalog = 'shared'
		  AND table_name = 'medals_earned'
	`).Scan(&count)
	if err != nil {
		return false
	}
	return count > 0
}
