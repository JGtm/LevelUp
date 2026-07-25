// Package analysis — citations_engine.go : moteur de calcul des citations par match.
//
// Port Go de src/analysis/citations/engine.py.
// 0 accès DB — entrées domain.*, sortie []domain.CitationMatchDelta.
//
// Mapping types supportés :
//   - "medal"      : 1 ou N medal_ids → sum des counts
//   - "stat"       : colonne match_participants → int(valeur)
//   - "pve_stat"   : idem depuis pve_match_stats (graceful degradation si absent)
//   - "objective_stat": colonne match_objective_stats (CTF/Zones/Oddball) → int(valeur)
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
//
// Sémantique progression (R1-R8) :
//   - Leaf : delta = min(raw, max(tier_targets) − cumulPre). Zéro si déjà masterisée (R3).
//   - Composite : +1 par enfant qui traverse son palier final durant ce match (R4).
//   - Méta (composite de composites) : même règle en cascade, passes itératives (R6).
//   - Sans tier_targets sur une leaf : delta = raw, pas de cap, pas de transition composite (R7).
//   - Sans tier_targets sur un composite : max = len(children) (R7).
func ComputeFullMatchCitations(
	in CitationProgressInput,
	mappings []domain.CitationFullMapping,
) []domain.CitationMatchDelta {
	cumulPre := in.CumulPre
	if cumulPre == nil {
		cumulPre = map[string]int{}
	}

	tierMax := buildTierMaxIndex(mappings)

	// Phase A — leaves : dispatch + cap (R1-R3).
	leafDeltas := make(map[string]int, len(mappings))
	for _, m := range mappings {
		if m.MappingType == domain.CitationMappingTypeComposite {
			continue
		}
		raw := dispatchFull(m, in.Ctx)
		if raw <= 0 {
			continue
		}
		if delta := capLeafDelta(raw, cumulPre[m.NameNorm], tierMax[m.NameNorm]); delta > 0 {
			leafDeltas[m.NameNorm] = delta
		}
	}

	// Phase B — état post-match des leaves.
	cumulPost := make(map[string]int, len(cumulPre)+len(leafDeltas))
	for k, v := range cumulPre {
		cumulPost[k] = v
	}
	for k, v := range leafDeltas {
		cumulPost[k] += v
	}

	// Phase C — composites/métas : transitions de palier (R4-R7).
	compositeDeltas := ComputeCompositeTransitions(cumulPre, cumulPost, tierMax, mappings)

	all := make([]domain.CitationMatchDelta, 0, len(leafDeltas)+len(compositeDeltas))
	for n, v := range leafDeltas {
		all = append(all, domain.CitationMatchDelta{NameNorm: n, Value: v})
	}
	for n, v := range compositeDeltas {
		all = append(all, domain.CitationMatchDelta{NameNorm: n, Value: v})
	}
	return all
}

// capLeafDelta applique R1-R3 : cap du delta brut au cap-room restant avant le palier final.
// maxTier == 0 → pas de tier_targets → pas de cap, delta = raw.
func capLeafDelta(raw, pre, maxTier int) int {
	if maxTier == 0 {
		return raw
	}
	capRoom := maxTier - pre
	if capRoom <= 0 {
		return 0
	}
	if raw > capRoom {
		return capRoom
	}
	return raw
}

// buildTierMaxIndex construit l'index citation_name_norm → max(tier_targets).
// Retourne 0 pour les citations sans tier_targets.
func buildTierMaxIndex(mappings []domain.CitationFullMapping) map[string]int {
	idx := make(map[string]int, len(mappings))
	for _, m := range mappings {
		idx[m.NameNorm] = ParseTierMax(m.TierTargets)
	}
	return idx
}

// ParseTierMax retourne la valeur maximale d'un CSV tier_targets.
// Retourne 0 si nil ou vide.
func ParseTierMax(tierTargetsCSV *string) int {
	if tierTargetsCSV == nil || *tierTargetsCSV == "" {
		return 0
	}
	max := 0
	for _, part := range strings.Split(*tierTargetsCSV, ",") {
		if v, err := strconv.Atoi(strings.TrimSpace(part)); err == nil && v > max {
			max = v
		}
	}
	return max
}

func dispatchFull(m domain.CitationFullMapping, ctx domain.CitationContext) int {
	switch m.MappingType {
	case domain.CitationMappingTypeMedal:
		return computeMedalValue(m, ctx.Medals)
	case domain.CitationMappingTypeStat, domain.CitationMappingTypePveStat,
		domain.CitationMappingTypeWeaponStat, domain.CitationMappingTypeObjectiveStat:
		if m.StatName == nil {
			return 0
		}
		// objective_stat : StatName = colonne match_objective_stats (zone_captures,
		// flag_returns, ...) injectée dans ctx.Stats par sync.loadObjectiveStats. Les
		// colonnes *_seconds (DOUBLE) sont tronquées à l'entier (paliers en secondes).
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
