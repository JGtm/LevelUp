// Package analysis — medals.go : algorithmes purs de la page Médailles.
//
// Règle architecture : 0 accès DB, 0 HTTP — entrée domain.*, sortie domain.*.
// La résolution de catégorie est injectée (MedalCategoryResolveFunc) pour garder
// ces fonctions testables sans dépendance à un titre : le baseline (regroupement
// natif par medal_type) sert de défaut title-agnostic.
package analysis

import (
	"sort"
	"strconv"
	"strings"

	"levelup/go-api/internal/domain"
)

// SuperSectionOther est la super-section fallback (title-agnostic). Toute médaille
// non enrichie y atterrit, et cette super-section est toujours affichée en dernier.
const SuperSectionOther = "other"

// MedalCategoryResolveFunc résout (catégorie, super-section, tri) d'une médaille.
// Signature identique à port.MedalCategoryResolver.Resolve pour qu'une méthode de
// resolver s'y assigne directement.
type MedalCategoryResolveFunc func(medalID int64, medalType string, difficultyIndex int) (categoryKey, superSectionKey string, sort int)

// BaselineMedalCategory est le regroupement NATIF de tout titre : la catégorie est
// le medal_type normalisé, la super-section est "other", le tri est l'index de
// rareté. medalType vide → catégorie "other" (aucune médaille perdue).
func BaselineMedalCategory(medalType string, difficultyIndex int) (categoryKey, superSectionKey string, sort int) {
	key := NormalizeMedalKey(medalType)
	if key == "" {
		key = SuperSectionOther
	}
	return key, SuperSectionOther, difficultyIndex
}

// BaselineMedalCategoryResolver est le resolver par défaut (implémente
// port.MedalCategoryResolver de façon structurelle, sans importer port).
type BaselineMedalCategoryResolver struct{}

// Resolve applique le regroupement baseline (medal_type + super-section "other").
func (BaselineMedalCategoryResolver) Resolve(_ int64, medalType string, difficultyIndex int) (string, string, int) {
	return BaselineMedalCategory(medalType, difficultyIndex)
}

// NormalizeMedalKey transforme un libellé en clé stable snake_case
// ("King of the Hill" → "king_of_the_hill", "Game End" → "game_end",
// "Heroic" → "heroic"). Vide → "".
func NormalizeMedalKey(s string) string {
	s = strings.ToLower(strings.TrimSpace(s))
	if s == "" {
		return ""
	}
	var b strings.Builder
	prevSep := false
	for _, r := range s {
		if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') {
			b.WriteRune(r)
			prevSep = false
			continue
		}
		if !prevSep && b.Len() > 0 {
			b.WriteByte('_')
			prevSep = true
		}
	}
	return strings.TrimRight(b.String(), "_")
}

// MergeMedalCatalog itère le CATALOGUE (toutes les médailles du titre) et attache à
// chacune son compteur obtenu (0 si jamais), sa catégorie/super-section/tri résolus.
// Une médaille présente dans earned mais absente du catalogue produit un item
// fallback ("#<id>", catégorie résolue, super-section "other" par baseline) — aucune
// médaille obtenue n'est perdue. La liste retournée est triée de façon déterministe
// (super-section, catégorie, tri, id) ; les images sont ajoutées par le service.
func MergeMedalCatalog(
	catalog []domain.MedalCatalogRow,
	earned []domain.MedalEarnedRow,
	resolve MedalCategoryResolveFunc,
) []domain.MedalSummaryItem {
	if resolve == nil {
		resolve = BaselineMedalCategoryResolver{}.Resolve
	}
	earnedByID := make(map[int64]int, len(earned))
	for _, e := range earned {
		earnedByID[e.MedalID] += e.TotalCount
	}

	items := make([]domain.MedalSummaryItem, 0, len(catalog)+4)
	seen := make(map[int64]bool, len(catalog))
	for _, row := range catalog {
		seen[row.MedalID] = true
		cat, super, srt := resolve(row.MedalID, row.MedalType, row.DifficultyIndex)
		items = append(items, domain.MedalSummaryItem{
			MedalID:        row.MedalID,
			Name:           row.Label,
			Description:    row.Description,
			Difficulty:     row.Difficulty,
			DifficultyKey:  NormalizeMedalKey(row.Difficulty),
			DifficultyRank: row.DifficultyIndex,
			Category:       cat,
			SuperSection:   super,
			PersonalScore:  row.PersonalScore,
			Count:          earnedByID[row.MedalID],
			Sort:           srt,
		})
	}

	// Médailles obtenues absentes du catalogue → fallback (jamais perdues).
	for _, e := range earned {
		if seen[e.MedalID] || e.TotalCount <= 0 {
			continue
		}
		seen[e.MedalID] = true
		cat, super, srt := resolve(e.MedalID, "", 0)
		items = append(items, domain.MedalSummaryItem{
			MedalID:      e.MedalID,
			Name:         "#" + strconv.FormatInt(e.MedalID, 10),
			Category:     cat,
			SuperSection: super,
			Count:        e.TotalCount,
			Sort:         srt,
		})
	}

	sortMedalItems(items)
	return items
}

// GroupMedalsByCategory regroupe les items par catégorie (une catégorie → une
// super-section). Earned = médailles distinctes obtenues (Count>0), Total = nb de
// médailles, TotalCount = somme des compteurs. Groupes ordonnés comme les items
// (super-section puis catégorie), items ordonnés par tri puis id.
func GroupMedalsByCategory(items []domain.MedalSummaryItem) []domain.MedalCategoryGroup {
	groups := make([]domain.MedalCategoryGroup, 0)
	idx := make(map[string]int, len(items))
	for _, it := range items {
		gi, ok := idx[it.Category]
		if !ok {
			gi = len(groups)
			idx[it.Category] = gi
			groups = append(groups, domain.MedalCategoryGroup{
				SuperSection: it.SuperSection,
				Category:     it.Category,
				Items:        make([]domain.MedalSummaryItem, 0, 4),
			})
		}
		g := &groups[gi]
		g.Items = append(g.Items, it)
		g.Total++
		g.TotalCount += it.Count
		if it.Count > 0 {
			g.Earned++
		}
	}
	return groups
}

// sortMedalItems ordonne : super-section (other en dernier), catégorie, tri, id.
func sortMedalItems(items []domain.MedalSummaryItem) {
	sort.SliceStable(items, func(i, j int) bool {
		a, b := items[i], items[j]
		if ra, rb := superSectionRank(a.SuperSection), superSectionRank(b.SuperSection); ra != rb {
			return ra < rb
		}
		if a.SuperSection != b.SuperSection {
			return a.SuperSection < b.SuperSection
		}
		if a.Category != b.Category {
			return a.Category < b.Category
		}
		if a.Sort != b.Sort {
			return a.Sort < b.Sort
		}
		return a.MedalID < b.MedalID
	})
}

// superSectionRank force la super-section "other" en dernier ; les autres sont
// départagées alphabétiquement (title-agnostic — pas de liste de clés en dur).
func superSectionRank(s string) int {
	if s == SuperSectionOther {
		return 1
	}
	return 0
}
