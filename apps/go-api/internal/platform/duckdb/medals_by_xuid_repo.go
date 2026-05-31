// Package duckdb — medals_by_xuid_repo.go : implementation DuckDB du loader
// medals par (xuid, match_id) (port.MedalsByXUIDRepository).
//
// Source : table medals_earned du catalogue shared_matches_v2. La lecture passe
// par SharedReadDB().Get() qui retourne (ADR 0016) une connexion DIRECTE — les
// tables sont a la racine, sans alias `shared`. La query reference donc
// medals_earned en bare (PAS `shared.medals_earned`, qui ne resout que sur la
// topologie de test legacy et renvoyait silencieusement ErrCapabilityNotSupported
// en prod via isTableNotFoundErr). La colonne stockee est medal_name_id (BIGINT
// dans le schema actuel — cf. internal/migration/steps_shared.go), exposee
// au port comme MedalID pour rester aligne avec le contract.
//
// Capability gating : si medals_earned est absente (titre sans cette capability),
// DuckDB remonte une erreur "Table with name ... does not exist" — interceptee
// via isTableNotFoundErr et convertie en games.ErrCapabilityNotSupported. Plus
// pérenne qu'une introspection information_schema (les CATALOG/SCHEMA varient
// entre prod RO direct, sharedprovider, et tests in-memory).
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

	q, args := buildMedalsByXUIDQuery(filters)
	db, release, err := r.pdb.SharedReadDB().Get(ctx)
	if err != nil {
		return nil, fmt.Errorf("MedalsByXUIDRepo.LoadMedalsForMatchesByXUID: shared reader: %w", err)
	}
	defer release()

	dbRows, err := db.QueryContext(ctx, q, args...)
	if err != nil {
		if isTableNotFoundErr(err) {
			slog.DebugContext(ctx, "MedalsByXUIDRepo: shared.medals_earned missing",
				"slug", slug,
				"match_count", len(filters.MatchIDs),
				"xuid_count", len(filters.XUIDs))
			return nil, games.ErrCapabilityNotSupported
		}
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
    me.xuid,
    me.match_id,
    me.medal_name_id::BIGINT AS medal_id,
    me.count
FROM medals_earned me
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
