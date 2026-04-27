// Package analysis — citations_engine.go : moteur de calcul des citations par match.
//
// Port Go de src/analysis/citations/engine.py.
// 0 accès DB — entrées domain.*, sortie []domain.CitationMatchDelta.
//
// Mapping types supportés :
//   - "medal"      : 1 ou N medal_ids → sum des counts
//   - "stat"       : colonne match_participants → int(valeur)
//   - "pve_stat"   : idem depuis pve_match_stats (graceful degradation si absent)
//   - "weapon_stat": "weapon_kills:<name>" dans Stats
//   - "award"      : personal_score_awards (award_name → total)
//   - "custom"     : dispatch via citations_custom.go
//   - "composite"  : non calculé par-match (agrégation uniquement)
package analysis

import (
	"strconv"
	"strings"

	"levelup/go-api/internal/domain"
)

// ComputeFullMatchCitations calcule les deltas de citations avec le moteur complet.
// Gère tous les mapping_types dont composite (post-traitement).
// ctx  : données du match (medals, stats, awards, events, playlist, …).
// mappings : règles chargées depuis citation_mappings (Q40).
func ComputeFullMatchCitations(
	ctx domain.CitationContext,
	mappings []domain.CitationFullMapping,
) []domain.CitationMatchDelta {
	totals := make(map[string]int, len(mappings))
	for _, m := range mappings {
		if m.MappingType == "composite" {
			continue // calculé en post-traitement après le dispatch principal
		}
		val := dispatchFull(m, ctx)
		if val > 0 {
			totals[m.NameNorm] += val
		}
	}

	// Post-traitement : citations composite (valeur = nb d'enfants masterisés dans ce match).
	computeCompositeCitations(totals, mappings)

	deltas := make([]domain.CitationMatchDelta, 0, len(totals))
	for norm, val := range totals {
		if val > 0 {
			deltas = append(deltas, domain.CitationMatchDelta{NameNorm: norm, Value: val})
		}
	}
	return deltas
}

func dispatchFull(m domain.CitationFullMapping, ctx domain.CitationContext) int {
	switch m.MappingType {
	case "medal":
		return computeMedalValue(m, ctx.Medals)
	case "stat", "pve_stat", "weapon_stat":
		if m.StatName == nil {
			return 0
		}
		return int(ctx.Stats[*m.StatName])
	case "award":
		if m.AwardName == nil {
			return 0
		}
		return ctx.Awards[*m.AwardName]
	case "custom":
		if m.CustomFunction == nil {
			return 0
		}
		return dispatchCustom(*m.CustomFunction, ctx)
	}
	return 0
}

func computeMedalValue(m domain.CitationFullMapping, medals map[int64]int) int {
	if m.MedalIDs != nil && *m.MedalIDs != "" {
		total := 0
		for _, part := range strings.Split(*m.MedalIDs, ",") {
			if id, err := strconv.ParseInt(strings.TrimSpace(part), 10, 64); err == nil {
				total += medals[id]
			}
		}
		return total
	}
	if m.MedalID != nil {
		return medals[*m.MedalID]
	}
	return 0
}

// ComputeMatchCitations calcule les deltas de citations pour un match.
//
// medals   : médailles gagnées par le joueur dans ce match (medal_id → count).
// mappings : règles chargées depuis citation_mappings (Q39).
//
// Retourne uniquement les citations dont la valeur est > 0.
func ComputeMatchCitations(
	medals []domain.MedalRaw,
	mappings []domain.CitationMedalMapping,
) []domain.CitationMatchDelta {
	// Index medal_id → count pour O(1) lookup.
	medalCount := make(map[int64]int, len(medals))
	for _, m := range medals {
		medalCount[m.MedalID] += m.Count
	}

	// Agréger par citation_name_norm (composite = plusieurs medal_ids).
	totals := make(map[string]int, len(mappings))
	for _, m := range mappings {
		if m.MedalID == 0 {
			continue
		}
		if c, ok := medalCount[m.MedalID]; ok && c > 0 {
			totals[m.NameNorm] += c
		}
	}

	deltas := make([]domain.CitationMatchDelta, 0, len(totals))
	for norm, val := range totals {
		if val > 0 {
			deltas = append(deltas, domain.CitationMatchDelta{NameNorm: norm, Value: val})
		}
	}
	return deltas
}
