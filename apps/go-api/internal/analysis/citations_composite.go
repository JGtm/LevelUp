// Package analysis — citations_composite.go : transitions de palier pour les citations
// composite et méta (composite de composites).
//
// Sémantique R4-R7 :
//
//	R4 : composite +1 par enfant qui traverse son palier final dans le match courant.
//	     Condition : cumulPre[child] < max(tier_targets[child]) AND cumulPost[child] >= max.
//	R6 : passes itératives pour les métas (composite de composites).
//	R7 : sans tier_targets sur un composite, max = len(children).
//	     Sans tier_targets sur une leaf enfant, max = 0 → jamais de transition.
package analysis

import (
	"encoding/json"
	"strings"

	"levelup/go-api/internal/domain"
)

// ComputeCompositeTransitions calcule les deltas des citations composite et méta.
//
// Chaque composite est traité exactement une fois, seulement quand tous ses enfants
// composites ont déjà été résolus (ordre topologique implicite via passes itératives).
// Les leaves sont pré-calculées une seule fois dans transitioned — elles ne sont
// jamais re-comptées, évitant le double comptage entre passes.
func ComputeCompositeTransitions(
	cumulPre, cumulPost map[string]int,
	tierMaxByNorm map[string]int,
	mappings []domain.CitationFullMapping,
) map[string]int {
	childCountByNorm := buildChildrenCountIndex(mappings)
	result := make(map[string]int)

	// transitioned[norm] = true si le nœud a traversé son palier final dans ce match.
	// Pour les composites : marqué après calcul si newTotal >= maxComp.
	transitioned := make(map[string]bool, len(mappings))
	for _, m := range mappings {
		if m.MappingType == domain.CitationMappingTypeComposite {
			continue
		}
		max := tierMaxByNorm[m.NameNorm]
		if max > 0 && cumulPre[m.NameNorm] < max && cumulPost[m.NameNorm] >= max {
			transitioned[m.NameNorm] = true
		}
	}

	// processed[norm] = true quand le delta du composite est final (traité une seule fois).
	processed := make(map[string]bool, len(mappings))
	for _, m := range mappings {
		if m.MappingType != domain.CitationMappingTypeComposite {
			processed[m.NameNorm] = true // les leaves sont "processed" dès le départ
		}
	}

	// Passes jusqu'à stabilisation (max = profondeur de la hiérarchie composite).
	// Chaque passe traite les composites dont tous les enfants composites sont déjà résolus.
	for range 5 {
		changed := false
		for _, m := range mappings {
			if m.MappingType != domain.CitationMappingTypeComposite ||
				m.CompositeChildren == nil || *m.CompositeChildren == "" ||
				processed[m.NameNorm] {
				continue
			}
			children, err := parseCompositeChildrenJSON(*m.CompositeChildren)
			if err != nil || len(children) == 0 {
				processed[m.NameNorm] = true
				continue
			}

			// Attendre que tous les enfants composites soient résolus (ordre topo).
			allReady := true
			for _, child := range children {
				if childCountByNorm[child] > 0 && !processed[child] {
					allReady = false
					break
				}
			}
			if !allReady {
				continue
			}

			newlyMastered := 0
			for _, child := range children {
				maxChild := effectiveMax(tierMaxByNorm[child], childCountByNorm[child])
				if maxChild > 0 && transitioned[child] {
					newlyMastered++ // R4
				}
			}

			processed[m.NameNorm] = true
			changed = true

			if newlyMastered == 0 {
				continue
			}

			maxComp := effectiveMax(tierMaxByNorm[m.NameNorm], len(children))
			capRoom := maxComp - cumulPre[m.NameNorm]
			if capRoom <= 0 {
				transitioned[m.NameNorm] = true // déjà au max → peut déclencher méta parent
				continue
			}
			delta := newlyMastered
			if delta > capRoom {
				delta = capRoom
			}
			result[m.NameNorm] = delta
			cumulPost[m.NameNorm] += delta
			if cumulPre[m.NameNorm]+delta >= maxComp {
				transitioned[m.NameNorm] = true // R4 → peut déclencher méta parent
			}
		}
		if !changed {
			break
		}
	}
	return result
}

// effectiveMax retourne max si > 0, sinon fallback.
func effectiveMax(max, fallback int) int {
	if max > 0 {
		return max
	}
	return fallback
}

// buildChildrenCountIndex construit l'index name_norm → nb d'enfants pour les composites.
func buildChildrenCountIndex(mappings []domain.CitationFullMapping) map[string]int {
	idx := make(map[string]int)
	for _, m := range mappings {
		if m.MappingType != domain.CitationMappingTypeComposite || m.CompositeChildren == nil {
			continue
		}
		children, _ := parseCompositeChildrenJSON(*m.CompositeChildren)
		idx[m.NameNorm] = len(children)
	}
	return idx
}

// ApplyCompositeCitationsPerMatch est un outil de rescue pour recalculer les composites
// à partir de valeurs existantes dans match_citations (mode backfill d'urgence).
// Le sync autonome (BackfillMatchCitations) gère la sémantique correcte automatiquement.
//
// Sera refactorisé pour utiliser le moteur unifié (commit 4).
func ApplyCompositeCitationsPerMatch(totals map[string]int, mappings []domain.CitationFullMapping) {
	for range 5 {
		if !applyCompositesPass(totals, mappings) {
			break
		}
	}
}

func applyCompositesPass(totals map[string]int, mappings []domain.CitationFullMapping) bool {
	changed := false
	for _, m := range mappings {
		if m.MappingType != domain.CitationMappingTypeComposite ||
			m.CompositeChildren == nil || *m.CompositeChildren == "" {
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

// parseCompositeChildrenJSON décode la liste JSON des enfants d'une citation composite.
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
