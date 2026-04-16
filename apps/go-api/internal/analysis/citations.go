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
	"sort"
	"strings"

	"levelup/go-api/internal/domain"
)

// MergeCitationTotals fusionne les totaux agrégés avec les métadonnées de citation
// pour produire la liste enrichie affichée dans la page Citations.
//
// totalRows : résultat de Q35 (name_norm → total).
// mappings  : résultat de Q34 (metadata citation_mappings).
// Seules les citations dont le total > 0 ET qui ont un mapping sont retournées.
func MergeCitationTotals(
	totalRows []domain.CitationTotalRow,
	mappings []domain.CitationMappingRow,
) []domain.CitationItem {
	// Construire un index par name_norm.
	byNorm := make(map[string]domain.CitationMappingRow, len(mappings))
	for _, m := range mappings {
		byNorm[m.NameNorm] = m
	}

	items := make([]domain.CitationItem, 0, len(totalRows))
	for _, t := range totalRows {
		if t.Total <= 0 {
			continue
		}
		m, ok := byNorm[t.NameNorm]
		if !ok {
			// Citation sans mapping connu — inclure quand même avec catégorie "misc".
			items = append(items, domain.CitationItem{
				NameNorm:    t.NameNorm,
				NameDisplay: t.NameNorm,
				Category:    "misc",
				Total:       t.Total,
			})
			continue
		}
		items = append(items, domain.CitationItem{
			NameNorm:    t.NameNorm,
			NameDisplay: m.NameDisplay,
			Category:    m.Category,
			Total:       t.Total,
			ImagePath:   m.ImagePath,
			Description: m.Description,
		})
	}

	// Trier par catégorie puis total décroissant.
	sort.Slice(items, func(i, j int) bool {
		if items[i].Category != items[j].Category {
			return items[i].Category < items[j].Category
		}
		return items[i].Total > items[j].Total
	})
	return items
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
				Category:  "misc",
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
