// Package sync — citations_checks.go : invariants post-compute pour match_citations.
//
// Invariants vérifiés (V1-V4) :
//   V1 : aucune valeur ≤ 0 dans match_citations (delta zéro ne doit pas être écrit).
//   V2 : cumul feuille ≤ max(tier_targets) pour les citations avec tier_targets.
//   V3 : cumul composite ≤ effectiveMax(tier_targets, len(children)).
//   V4 : valeur per-match d'un composite ≤ len(children).
package sync

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"log/slog"
	"strings"

	"levelup/go-api/internal/analysis"
	"levelup/go-api/internal/domain"
)

// CitationCheckViolation décrit une violation d'invariant post-compute.
type CitationCheckViolation struct {
	Rule    string
	Details string
}

// RunCitationPostComputeChecks vérifie les invariants V1-V4 après un recompute.
// Retourne la liste des violations (vide = tout est correct).
func (e *SyncEngine) RunCitationPostComputeChecks(ctx context.Context) ([]CitationCheckViolation, error) {
	playerHandle, err := OpenPlayerDB(e.playerDBPath)
	if err != nil {
		return nil, fmt.Errorf("RunCitationPostComputeChecks open player: %w", err)
	}
	defer playerHandle.Close()

	metaDB, err := sql.Open("duckdb", e.metadataDBPath+"?access_mode=READ_ONLY")
	if err != nil {
		return nil, fmt.Errorf("RunCitationPostComputeChecks open metadata: %w", err)
	}
	defer metaDB.Close()
	metaDB.SetMaxOpenConns(1)

	mappings, err := loadFullCitationMappings(ctx, metaDB)
	if err != nil {
		return nil, fmt.Errorf("RunCitationPostComputeChecks mappings: %w", err)
	}

	var violations []CitationCheckViolation

	// V1 : aucune valeur ≤ 0.
	if v := checkV1NoZeroValues(ctx, playerHandle.SQLDb()); v != nil {
		violations = append(violations, *v)
	}

	// V2 + V3 + V4 nécessitent les mappings.
	childCountByNorm := buildCompositeChildCount(mappings)
	tierMax := make(map[string]int, len(mappings))
	for _, m := range mappings {
		tierMax[m.NameNorm] = analysis.ParseTierMax(m.TierTargets)
	}

	cumuls, err := loadAllCumulTotals(ctx, playerHandle.SQLDb())
	if err != nil {
		return nil, fmt.Errorf("RunCitationPostComputeChecks load cumuls: %w", err)
	}
	perMatch, err := loadAllPerMatchValues(ctx, playerHandle.SQLDb())
	if err != nil {
		return nil, fmt.Errorf("RunCitationPostComputeChecks load per-match: %w", err)
	}

	violations = append(violations, checkV2LeafCumul(mappings, tierMax, cumuls)...)
	violations = append(violations, checkV3CompositeCumul(mappings, tierMax, childCountByNorm, cumuls)...)
	violations = append(violations, checkV4CompositePerMatch(mappings, childCountByNorm, perMatch)...)

	if len(violations) == 0 {
		slog.InfoContext(ctx, "citations: invariants V1-V4 OK", "player", e.gamertag)
	} else {
		slog.WarnContext(ctx, "citations: invariants violés",
			"player", e.gamertag, "count", len(violations))
	}
	return violations, nil
}

// checkV1NoZeroValues : aucune valeur ≤ 0 dans match_citations.
func checkV1NoZeroValues(ctx context.Context, db *sql.DB) *CitationCheckViolation {
	var count int
	if err := db.QueryRowContext(ctx,
		`SELECT COUNT(*) FROM match_citations WHERE value <= 0`,
	).Scan(&count); err != nil || count == 0 {
		return nil
	}
	return &CitationCheckViolation{
		Rule:    "V1",
		Details: fmt.Sprintf("%d lignes avec value ≤ 0 dans match_citations", count),
	}
}

// checkV2LeafCumul : cumul feuille ≤ max(tier_targets).
func checkV2LeafCumul(
	mappings []domain.CitationFullMapping,
	tierMax map[string]int,
	cumuls map[string]int,
) []CitationCheckViolation {
	var out []CitationCheckViolation
	for _, m := range mappings {
		if m.MappingType == domain.CitationMappingTypeComposite {
			continue
		}
		max := tierMax[m.NameNorm]
		if max <= 0 {
			continue // pas de cap → pas de vérification
		}
		if total := cumuls[m.NameNorm]; total > max {
			out = append(out, CitationCheckViolation{
				Rule: "V2",
				Details: fmt.Sprintf("leaf %s: cumul=%d > max_tier=%d",
					m.NameNorm, total, max),
			})
		}
	}
	return out
}

// checkV3CompositeCumul : cumul composite ≤ effectiveMax.
func checkV3CompositeCumul(
	mappings []domain.CitationFullMapping,
	tierMax, childCountByNorm map[string]int,
	cumuls map[string]int,
) []CitationCheckViolation {
	var out []CitationCheckViolation
	for _, m := range mappings {
		if m.MappingType != domain.CitationMappingTypeComposite {
			continue
		}
		childCount := childCountByNorm[m.NameNorm]
		effectiveMax := analysis.ParseTierMax(m.TierTargets)
		if effectiveMax <= 0 {
			effectiveMax = childCount
		}
		if effectiveMax <= 0 {
			continue
		}
		if total := cumuls[m.NameNorm]; total > effectiveMax {
			out = append(out, CitationCheckViolation{
				Rule: "V3",
				Details: fmt.Sprintf("composite %s: cumul=%d > effectiveMax=%d",
					m.NameNorm, total, effectiveMax),
			})
		}
	}
	return out
}

// checkV4CompositePerMatch : valeur per-match composite ≤ len(children).
func checkV4CompositePerMatch(
	mappings []domain.CitationFullMapping,
	childCountByNorm map[string]int,
	perMatch map[string]map[string]int,
) []CitationCheckViolation {
	compositeSet := make(map[string]struct{}, len(mappings))
	for _, m := range mappings {
		if m.MappingType == domain.CitationMappingTypeComposite {
			compositeSet[m.NameNorm] = struct{}{}
		}
	}
	var out []CitationCheckViolation
	for matchID, byNorm := range perMatch {
		for norm, val := range byNorm {
			if _, isComp := compositeSet[norm]; !isComp {
				continue
			}
			max := childCountByNorm[norm]
			if max > 0 && val > max {
				out = append(out, CitationCheckViolation{
					Rule: "V4",
					Details: fmt.Sprintf("composite %s match %s: value=%d > len(children)=%d",
						norm, matchID, val, max),
				})
			}
		}
	}
	return out
}

// loadAllCumulTotals charge SUM(value) par citation_name_norm depuis match_citations.
func loadAllCumulTotals(ctx context.Context, db *sql.DB) (map[string]int, error) {
	rows, err := db.QueryContext(ctx,
		`SELECT citation_name_norm, SUM(value) FROM match_citations GROUP BY citation_name_norm`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	result := make(map[string]int)
	for rows.Next() {
		var name string
		var total int
		if err := rows.Scan(&name, &total); err != nil {
			return nil, err
		}
		result[name] = total
	}
	return result, rows.Err()
}

// loadAllPerMatchValues charge toutes les (match_id, citation_name_norm, value) depuis match_citations.
func loadAllPerMatchValues(ctx context.Context, db *sql.DB) (map[string]map[string]int, error) {
	rows, err := db.QueryContext(ctx,
		`SELECT match_id, citation_name_norm, value FROM match_citations`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	result := make(map[string]map[string]int)
	for rows.Next() {
		var matchID, name string
		var val int
		if err := rows.Scan(&matchID, &name, &val); err != nil {
			return nil, err
		}
		if result[matchID] == nil {
			result[matchID] = make(map[string]int)
		}
		result[matchID][name] = val
	}
	return result, rows.Err()
}

// buildCompositeChildCount retourne name_norm → nb d'enfants pour les composites.
func buildCompositeChildCount(mappings []domain.CitationFullMapping) map[string]int {
	idx := make(map[string]int)
	for _, m := range mappings {
		if m.MappingType != domain.CitationMappingTypeComposite || m.CompositeChildren == nil {
			continue
		}
		s := strings.TrimSpace(*m.CompositeChildren)
		if s == "" {
			continue
		}
		var children []string
		if err := json.Unmarshal([]byte(s), &children); err == nil {
			idx[m.NameNorm] = len(children)
		}
	}
	return idx
}
