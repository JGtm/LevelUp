// Package analysis — citations_composite.go : calcul des citations composite par match.
//
// Port Go de src/analysis/citations/composite.py (_apply_composite_citations).
// Les citations composite ne sont PAS calculées ligne-par-ligne dans le moteur principal ;
// elles sont calculées en post-traitement à partir des totaux du même match.
//
// Valeur = nombre d'enfants « masterisés » dans ce match :
//   - masterisé si child_val >= max(tier_targets) si tier_targets non vide
//   - masterisé si child_val > 0 sinon
package analysis

import (
	"encoding/json"
	"strconv"
	"strings"

	"levelup/go-api/internal/domain"
)

// ApplyCompositeCitations est la version exportée de computeCompositeCitations.
// Utilisée par le moteur principal (per-match, avec vérification des tiers).
func ApplyCompositeCitations(totals map[string]int, mappings []domain.CitationFullMapping) {
	computeCompositeCitations(totals, mappings)
}

// ApplyCompositeCitationsPerMatch calcule les composites per-match en mode backfill.
// Différence avec computeCompositeCitations : un enfant est "actif dans ce match"
// dès que val > 0 — les tier_targets sont des seuils cumulatifs, pas per-match.
// Gère les composites imbriqués via passes répétées jusqu'à stabilisation.
func ApplyCompositeCitationsPerMatch(totals map[string]int, mappings []domain.CitationFullMapping) {
	for range 5 {
		if !applyCompositesPass(totals, mappings) {
			break
		}
	}
}

// applyCompositesPass effectue une passe sur les composites.
// Retourne true si au moins une valeur a changé (indique qu'une autre passe est utile).
func applyCompositesPass(totals map[string]int, mappings []domain.CitationFullMapping) bool {
	changed := false
	for _, m := range mappings {
		if m.MappingType != "composite" || m.CompositeChildren == nil || *m.CompositeChildren == "" {
			continue
		}
		children, err := parseCompositeChildrenJSON(*m.CompositeChildren)
		if err != nil || len(children) == 0 {
			continue
		}
		count := 0
		for _, child := range children {
			if totals[child] > 0 {
				count++
			}
		}
		if count > 0 && totals[m.NameNorm] != count {
			totals[m.NameNorm] = count
			changed = true
		}
	}
	return changed
}

// computeCompositeCitations enrichit totals avec les valeurs des citations composite.
// Doit être appelée après le dispatch principal (tous les autres mapping_types calculés).
// Les composites utilisent les valeurs déjà dans totals pour déterminer combien
// d'enfants sont masterisés dans ce match.
func computeCompositeCitations(totals map[string]int, mappings []domain.CitationFullMapping) {
	// Index name_norm → TierTargets (CSV) pour résoudre les seuils des enfants.
	childTiers := make(map[string]*string, len(mappings))
	for i := range mappings {
		childTiers[mappings[i].NameNorm] = mappings[i].TierTargets
	}

	for _, m := range mappings {
		if m.MappingType != "composite" || m.CompositeChildren == nil || *m.CompositeChildren == "" {
			continue
		}
		children, err := parseCompositeChildrenJSON(*m.CompositeChildren)
		if err != nil || len(children) == 0 {
			continue
		}
		count := 0
		for _, child := range children {
			if compositeChildMasterised(totals[child], childTiers[child]) {
				count++
			}
		}
		if count > 0 {
			totals[m.NameNorm] += count
		}
	}
}

// parseCompositeChildrenJSON décode la liste JSON des enfants d'une citation composite.
// Ex: `["wins_ctf","wins_slayer","wins_strongholds"]` → slice de strings.
func parseCompositeChildrenJSON(s string) ([]string, error) {
	s = strings.TrimSpace(s)
	if s == "" {
		return nil, nil
	}
	var children []string
	if err := json.Unmarshal([]byte(s), &children); err != nil {
		return nil, err
	}
	return children, nil
}

// compositeChildMasterised retourne true si val atteint le seuil max de tierTargets.
// Si tierTargets est nil ou vide, masterisé dès que val > 0.
// Réutilise la même logique que computeTierProgress pour la cohérence.
func compositeChildMasterised(val int, tierTargetsCSV *string) bool {
	if val <= 0 {
		return false
	}
	if tierTargetsCSV == nil || *tierTargetsCSV == "" {
		return true
	}
	maxTier := 0
	for _, part := range strings.Split(*tierTargetsCSV, ",") {
		v, err := strconv.Atoi(strings.TrimSpace(part))
		if err == nil && v > maxTier {
			maxTier = v
		}
	}
	if maxTier == 0 {
		return true
	}
	return val >= maxTier
}
