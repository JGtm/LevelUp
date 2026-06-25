// Package analysis — citations.go : algorithmes purs pour les pages Citations et Commendations.
//
// Miroir Go simplifié de :
//
//	src/analysis/citations/engine.py → MergeCitationTotals, MergeMedalSummary
//	src/data/citation_definitions.py → structure CitationMappingRow
//
// Règle architecture : 0 accès DB — entrée domain.*, sortie domain.*.
package analysis

import (
	"net/url"
	"sort"
	"strings"

	"levelup/go-api/internal/domain"
)

// OverrideCompositeTotals remplace les totaux des citations composites par le nombre
// d'enfants globalement masterisés (comme Python le fait au moment de l'affichage).
//
// Q35 retourne la somme brute des valeurs per-match pour les composites, ce qui est
// sans signification (ex. 2100 pour human_weapons_mastery). Cette fonction calcule
// le vrai total : "combien d'enfants ont atteint leur seuil max en cumulatif ?".
// Gère les composites imbriqués (ex: all_weapons → human_weapons → br75) via multi-pass.
func OverrideCompositeTotals(
	totals []domain.CitationTotalRow,
	mappings []domain.CitationMappingRow,
) []domain.CitationTotalRow {
	// Totaux cumulatifs (Q35) : base de calcul pour les enfants feuilles.
	aggMap := make(map[string]int, len(totals))
	for _, t := range totals {
		aggMap[t.NameNorm] = t.Total
	}

	// Index tier_targets pour le check de masterisation des enfants.
	// enabledNorms : citations activées seulement (Q34 filtre enabled IS NOT FALSE).
	// Utilisé pour ignorer les enfants désactivés (ex: brute_slayer, skimmer_slayer)
	// — identique au `if child not in mappings: continue` de Python.
	tierMap := make(map[string]*string, len(mappings))
	enabledNorms := make(map[string]struct{}, len(mappings))
	for _, m := range mappings {
		tierMap[m.NameNorm] = m.TierTargets
		enabledNorms[m.NameNorm] = struct{}{}
	}

	// Extraire les composites avec leurs enfants.
	type compositeEntry struct {
		norm     string
		children []string
	}
	var composites []compositeEntry
	compositeNorms := make(map[string]struct{})
	for _, m := range mappings {
		if m.MappingType != domain.CitationMappingTypeComposite || m.CompositeChildren == nil || *m.CompositeChildren == "" {
			continue
		}
		children, err := parseCompositeChildrenJSON(*m.CompositeChildren)
		if err != nil || len(children) == 0 {
			continue
		}
		composites = append(composites, compositeEntry{norm: m.NameNorm, children: children})
		compositeNorms[m.NameNorm] = struct{}{}
	}
	if len(composites) == 0 {
		return totals
	}

	// enabledChildCount[norm] = nb d'enfants activés pour chaque composite.
	// Utilisé pour détecter si un enfant composite est TERMINÉ (count == total).
	enabledChildCount := make(map[string]int, len(composites))
	for _, comp := range composites {
		n := 0
		for _, child := range comp.children {
			if _, ok := enabledNorms[child]; ok {
				n++
			}
		}
		enabledChildCount[comp.norm] = n
	}

	// Multi-pass : gère les composites imbriqués.
	// Règle de masterisation :
	//   - Enfant feuille : val >= max(tier_targets)
	//   - Enfant composite : count == enabledChildCount[child] (terminé à 100%)
	//     → un composite enfant entamé mais non terminé ne compte pas.
	for range 5 {
		changed := false
		for _, comp := range composites {
			count := 0
			for _, child := range comp.children {
				if _, ok := enabledNorms[child]; !ok {
					continue // désactivé → ignorer
				}
				if _, isCompositeChild := compositeNorms[child]; isCompositeChild {
					// Enfant composite : masterisé seulement s'il est entièrement terminé.
					total := enabledChildCount[child]
					if total > 0 && aggMap[child] >= total {
						count++
					}
				} else {
					// Enfant feuille (cumul) : masterisé si val >= max(tier_targets).
					// Sans tier_targets → masterisé dès val > 0 (affichage cumulatif).
					maxT := ParseTierMax(tierMap[child])
					if (maxT == 0 && aggMap[child] > 0) || (maxT > 0 && aggMap[child] >= maxT) {
						count++
					}
				}
			}
			if aggMap[comp.norm] != count {
				aggMap[comp.norm] = count
				changed = true
			}
		}
		if !changed {
			break
		}
	}

	// Reconstruire la slice : remplacer les valeurs composites, conserver les autres.
	// Tous les composites sont inclus (même count=0) pour afficher la progression 0/N.
	result := make([]domain.CitationTotalRow, 0, len(totals)+len(composites))
	seen := make(map[string]bool, len(totals))
	for _, t := range totals {
		if _, isComp := compositeNorms[t.NameNorm]; isComp {
			result = append(result, domain.CitationTotalRow{NameNorm: t.NameNorm, Total: aggMap[t.NameNorm]})
		} else {
			result = append(result, t)
		}
		seen[t.NameNorm] = true
	}
	// Ajouter les composites absents de Q35 (même non masterisés) pour affichage 0/N.
	for _, c := range composites {
		if !seen[c.norm] {
			result = append(result, domain.CitationTotalRow{NameNorm: c.norm, Total: aggMap[c.norm]})
		}
	}
	return result
}

// MergeCitationTotals fusionne les totaux agrégés avec les métadonnées de citation
// pour produire la liste enrichie affichée dans la page Citations.
//
// totalRows : résultat de Q35 après OverrideCompositeTotals (composites corrigés).
// mappings  : résultat de Q34 (metadata citation_mappings, incl. composite_children).
// Toutes les citations du catalogue (enabled) sont affichées, même celles non commencées (total=0).
// Les citations inactives (skimmer, brutes) sont déjà exclues en SQL via WHERE enabled IS NOT FALSE.
func MergeCitationTotals(
	totalRows []domain.CitationTotalRow,
	mappings []domain.CitationMappingRow,
) []domain.CitationItem {
	// Index des totaux par nom pour lookup O(1).
	totals := make(map[string]int, len(totalRows))
	for _, t := range totalRows {
		totals[t.NameNorm] = t.Total
	}

	// byNorm sert au composite children lookup (enfants activés).
	byNorm := make(map[string]domain.CitationMappingRow, len(mappings))
	for _, m := range mappings {
		byNorm[m.NameNorm] = m
	}

	// Itérer sur le catalogue (mappings) — source de vérité pour les citations à afficher.
	items := make([]domain.CitationItem, 0, len(mappings))
	for _, m := range mappings {
		total := totals[m.NameNorm] // 0 si jamais commencée

		var imgURL *string
		if m.ImagePath != nil && *m.ImagePath != "" {
			parts := strings.Split(*m.ImagePath, "/")
			encoded := make([]string, len(parts))
			for i, seg := range parts {
				encoded[i] = url.PathEscape(seg)
			}
			p := "/" + strings.Join(encoded, "/")
			imgURL = &p
		}
		item := domain.CitationItem{
			NameNorm:    m.NameNorm,
			NameDisplay: m.NameDisplay,
			Category:    m.Category,
			Total:       total,
			ImageURL:    imgURL,
			Description: m.Description,
		}

		if m.MappingType == domain.CitationMappingTypeComposite && m.CompositeChildren != nil && *m.CompositeChildren != "" {
			// Composite : Total = nb d'enfants masterisés (après OverrideCompositeTotals).
			// TierCount = nb d'enfants ACTIVÉS (présents dans byNorm) → même dénominateur que Python
			// (les enfants disabled comme brute_slayer/skimmer_slayer sont exclus du total).
			if children, err := parseCompositeChildrenJSON(*m.CompositeChildren); err == nil && len(children) > 0 {
				n := 0
				for _, child := range children {
					if _, ok := byNorm[child]; ok {
						n++
					}
				}
				if n == 0 {
					n = len(children) // fallback si byNorm vide (ne devrait pas arriver)
				}
				item.TierCount = n
				item.EarnedTiers = total
				if total >= n {
					item.MasteryPct = 100.0
				} else {
					item.MasteryPct = float64(total) / float64(n) * 100.0
					item.NextTierTarget = n // objectif : tous les enfants activés
				}
			}
		} else if m.TierTargets != nil && *m.TierTargets != "" {
			tiers := ParseTierTargets(*m.TierTargets)
			item.TierCount = len(tiers)
			for _, tier := range tiers {
				if total >= tier {
					item.EarnedTiers++
				}
			}
			if len(tiers) > 0 {
				lastTier := tiers[len(tiers)-1]
				if total >= lastTier {
					item.MasteryPct = 100.0
				} else {
					item.MasteryPct = computeTierProgress(total, tiers)
					for _, tier := range tiers {
						if total < tier {
							item.NextTierTarget = tier
							break
						}
					}
				}
			}
		}
		items = append(items, item)
	}

	// Rang d'affichage : 0 = méta-composite, 1 = composite, 2 = citation normale.
	// Méta-composite : composite dont au moins un enfant est lui-même un composite.
	citationRank := func(norm string) int {
		m, ok := byNorm[norm]
		if !ok || m.MappingType != domain.CitationMappingTypeComposite {
			return 2
		}
		if m.CompositeChildren != nil && *m.CompositeChildren != "" {
			if children, err := parseCompositeChildrenJSON(*m.CompositeChildren); err == nil {
				for _, child := range children {
					if cm, ok := byNorm[child]; ok && cm.MappingType == domain.CitationMappingTypeComposite {
						return 0
					}
				}
			}
		}
		return 1
	}

	sort.Slice(items, func(i, j int) bool {
		if items[i].Category != items[j].Category {
			return items[i].Category < items[j].Category
		}
		ri, rj := citationRank(items[i].NameNorm), citationRank(items[j].NameNorm)
		if ri != rj {
			return ri < rj
		}
		return items[i].Total > items[j].Total
	})
	return items
}

// BuildCitationsByCategory regroupe les citations par catégorie en préservant l'ordre
// de tri de items (catégorie alphabétique, puis total décroissant).
func BuildCitationsByCategory(items []domain.CitationItem) []domain.CitationCategoryGroup {
	if len(items) == 0 {
		return nil
	}
	var catOrder []string
	seen := make(map[string]bool)
	for _, item := range items {
		if !seen[item.Category] {
			seen[item.Category] = true
			catOrder = append(catOrder, item.Category)
		}
	}
	byCat := make(map[string]*domain.CitationCategoryGroup, len(catOrder))
	for _, cat := range catOrder {
		byCat[cat] = &domain.CitationCategoryGroup{Category: cat}
	}
	for _, item := range items {
		g := byCat[item.Category]
		g.Items = append(g.Items, item)
		g.Total += item.Total
		if item.MasteryPct >= 100.0 {
			g.Completed++
		}
	}
	result := make([]domain.CitationCategoryGroup, 0, len(catOrder))
	for _, cat := range catOrder {
		result = append(result, *byCat[cat])
	}
	return result
}

// ExtractCategories retourne la liste dédupliquée et triée des catégories.
func ExtractCategories(items []domain.CitationItem) []string {
	seen := make(map[string]bool)
	var cats []string
	for _, item := range items {
		if !seen[item.Category] {
			seen[item.Category] = true
			cats = append(cats, item.Category)
		}
	}
	sort.Strings(cats)
	return cats
}

// MergeMedalSummary fusionne les totaux de médailles avec les mappings citation
// pour produire la liste enrichie des commendations (médailles).
//
// medalRows    : résultat de Q36a (medal_id → total_count).
// citMappings  : résultat de Q36b (medal_id → display info depuis metadata).
func MergeMedalSummary(
	medalRows []domain.MedalEarnedRow,
	citMappings []domain.MedalCitationRow,
) []domain.CommendationItem {
	// Index par medal_id.
	byMedalID := make(map[int64]domain.MedalCitationRow, len(citMappings))
	for _, m := range citMappings {
		byMedalID[m.MedalID] = m
	}

	items := make([]domain.CommendationItem, 0, len(medalRows))
	for _, me := range medalRows {
		if me.TotalCount <= 0 {
			continue
		}
		m, ok := byMedalID[me.MedalID]
		if !ok {
			items = append(items, domain.CommendationItem{
				MedalID:   me.MedalID,
				MedalName: "Unknown",
				Count:     me.TotalCount,
				Category:  domain.CitationCategoryMisc,
			})
			continue
		}
		items = append(items, domain.CommendationItem{
			MedalID:   me.MedalID,
			MedalName: m.NameDisplay,
			Count:     me.TotalCount,
			Category:  m.Category,
			ImagePath: m.ImagePath,
		})
	}

	// Trier par catégorie puis count décroissant.
	sort.Slice(items, func(i, j int) bool {
		if items[i].Category != items[j].Category {
			return items[i].Category < items[j].Category
		}
		return items[i].Count > items[j].Count
	})
	return items
}

// GroupCommendationsByCategory regroupe les commendations par catégorie.
func GroupCommendationsByCategory(items []domain.CommendationItem) []domain.CommendationCategory {
	if len(items) == 0 {
		return nil
	}

	// Maintenir l'ordre des catégories tel que triées dans items.
	var catOrder []string
	seen := make(map[string]bool)
	for _, item := range items {
		cat := strings.ToLower(item.Category)
		if !seen[cat] {
			seen[cat] = true
			catOrder = append(catOrder, item.Category)
		}
	}

	byCategory := make(map[string]*domain.CommendationCategory, len(catOrder))
	for _, cat := range catOrder {
		key := strings.ToLower(cat)
		byCategory[key] = &domain.CommendationCategory{Category: cat}
	}

	for _, item := range items {
		key := strings.ToLower(item.Category)
		cat := byCategory[key]
		cat.Items = append(cat.Items, item)
		cat.Total += item.Count
	}

	result := make([]domain.CommendationCategory, 0, len(catOrder))
	for _, cat := range catOrder {
		key := strings.ToLower(cat)
		result = append(result, *byCategory[key])
	}
	return result
}
