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

// customDispatcher est le hook par lequel les règles citations Halo-only
// (ou autre titre) s'enregistrent. P5.4 : évite un import cycle
// analysis → halo_infinite → platform/duckdb → analysis. Le package
// halo_infinite enregistre sa fonction via init() au chargement.
var customDispatcher func(fnName string, ctx domain.CitationContext) int

// RegisterCustomDispatcher est appelée par le package title-spécifique
// (halo_infinite) pour fournir l'implémentation des règles custom citations.
// Doit être appelée avant ComputeFullMatchCitations (typiquement via init()).
func RegisterCustomDispatcher(fn func(string, domain.CitationContext) int) {
	customDispatcher = fn
}

// CitationProgressInput regroupe les données nécessaires au calcul de progression
// d'un match : contexte stats/medals/awards du match + état cumulatif avant ce match.
// CumulPre[citation_name_norm] = SUM(value) dans match_citations pour tous les matchs
// antérieurs au match courant. Nil ou vide = aucun historique (premier match).
type CitationProgressInput struct {
	Ctx      domain.CitationContext
	CumulPre map[string]int
}

// ComputeFullMatchCitations calcule les deltas de citations avec le moteur complet.
// Gère tous les mapping_types dont composite (post-traitement).
// in.Ctx      : données du match (medals, stats, awards, events, playlist, …).
// in.CumulPre : cumul avant ce match (nil accepté — traité comme vide).
// mappings    : règles chargées depuis citation_mappings.
func ComputeFullMatchCitations(
	in CitationProgressInput,
	mappings []domain.CitationFullMapping,
) []domain.CitationMatchDelta {
	totals := make(map[string]int, len(mappings))
	for _, m := range mappings {
		if m.MappingType == domain.CitationMappingTypeComposite {
			continue // calculé en post-traitement après le dispatch principal
		}
		val := dispatchFull(m, in.Ctx)
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
	case domain.CitationMappingTypeMedal:
		return computeMedalValue(m, ctx.Medals)
	case domain.CitationMappingTypeStat, domain.CitationMappingTypePveStat, domain.CitationMappingTypeWeaponStat:
		if m.StatName == nil {
			return 0
		}
		return int(ctx.Stats[*m.StatName])
	case domain.CitationMappingTypeAward:
		if m.AwardName == nil {
			return 0
		}
		return ctx.Awards[*m.AwardName]
	case domain.CitationMappingTypeCustom:
		if m.CustomFunction == nil {
			return 0
		}
		// P5.4 : custom dispatcher fourni par le titre via
		// RegisterCustomDispatcher (évite import cycle).
		if customDispatcher == nil {
			return 0
		}
		return customDispatcher(*m.CustomFunction, ctx)
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
