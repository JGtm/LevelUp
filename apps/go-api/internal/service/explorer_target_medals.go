// Package service — explorer_target_medals.go : construction du "top médailles"
// de l'encart Profil joueur cible à partir des médailles lifetime du service
// record Waypoint + métadonnées locales (label/description) + image statique.
package service

import (
	"context"
	"fmt"
	"sort"

	"levelup/go-api/internal/domain"
	"levelup/go-api/internal/port"
)

// explorerTopMedalsCap borne le nombre de médailles renvoyées au front (top 5
// affiché + expander jusqu'à 20 côté UI).
const explorerTopMedalsCap = 20

// buildTargetTopMedals mappe les médailles lifetime (NameID+Count) en
// MedalDigestItem triés par count décroissant (cap explorerTopMedalsCap),
// enrichis du label/description (locale) et de l'URL image statique
// `/static/medals/{titleSlug}/{id}.png`. Best-effort : si les définitions ne se
// résolvent pas, on garde compteur + image (label/desc vides). Retourne nil si
// pas de médailles ou pas de repo.
func buildTargetTopMedals(
	ctx context.Context,
	repo port.MedalDefinitionsRepository,
	medals []domain.RemoteMedalCount,
	titleSlug, locale string,
) []domain.MedalDigestItem {
	if repo == nil || len(medals) == 0 {
		return nil
	}

	ids := make([]int64, 0, len(medals))
	for _, m := range medals {
		ids = append(ids, m.NameID)
	}
	defs, err := repo.LookupByIDs(ctx, ids, locale)
	if err != nil {
		defs = nil // dégradation : on garde compteur + image sans label
	}

	items := make([]domain.MedalDigestItem, 0, len(medals))
	for _, m := range medals {
		def := defs[m.NameID]
		imageURL := ""
		if titleSlug != "" {
			imageURL = fmt.Sprintf("/static/medals/%s/%d.png", titleSlug, m.NameID)
		}
		items = append(items, domain.MedalDigestItem{
			MedalID:       m.NameID,
			Label:         def.Label,
			Description:   def.Description,
			ImageURL:      imageURL,
			TotalCount:    m.Count,
			Category:      def.MedalType,
			Difficulty:    def.Difficulty,
			PersonalScore: def.PersonalScore,
		})
	}

	sort.SliceStable(items, func(i, j int) bool {
		if items[i].TotalCount != items[j].TotalCount {
			return items[i].TotalCount > items[j].TotalCount
		}
		return items[i].MedalID < items[j].MedalID
	})

	if len(items) > explorerTopMedalsCap {
		items = items[:explorerTopMedalsCap]
	}
	return items
}
