// Package service — explorer_target_medals.go : construction du "top médailles"
// de l'encart Profil joueur cible à partir des médailles lifetime du service
// record Waypoint + métadonnées locales (label/description) + image statique.
package service

import (
	"context"
	"log/slog"
	"sort"

	"levelup/go-api/internal/assets/static"
	"levelup/go-api/internal/ctxkeys"
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
		item := domain.MedalDigestItem{
			MedalID:       m.NameID,
			Label:         def.Label,
			Description:   def.Description,
			TotalCount:    m.Count,
			Category:      def.MedalType,
			Difficulty:    def.Difficulty,
			PersonalScore: def.PersonalScore,
		}
		if titleSlug != "" {
			png, sp := static.MedalImage(titleSlug, m.NameID)
			if sp != nil {
				item.SpriteSheet, item.SpriteLeft, item.SpriteTop, item.SpriteWidth, item.SpriteHeight =
					sp.SheetURL, sp.Left, sp.Top, sp.Width, sp.Height
			} else {
				item.ImageURL = png
			}
		}
		items = append(items, item)
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

// computeTargetTopMedalsLocal agrège le top médailles de la cible sur EXACTEMENT
// les matchs du profil de combat LOCAL (shared.medals_earned, SUM(count) par
// identifiant), puis les enrichit via buildTargetTopMedals — MÊMES libellés,
// images et cap que la liste lifetime, aucune duplication de mapping.
//
// Alimente le bloc « Top médailles » quand le toggle du profil de combat est sur
// « Local ». Best-effort : toute défaillance est loguée puis dégradée en nil (le
// front masque alors le bloc), jamais fatale pour l'encart.
func (s *ExplorerService) computeTargetTopMedalsLocal(
	ctx context.Context, targetXUID string, localMatches []domain.ExplorerTargetRecentMatch,
) []domain.MedalDigestItem {
	if len(localMatches) == 0 {
		return nil
	}
	matchIDs := make([]string, 0, len(localMatches))
	for _, m := range localMatches {
		if m.MatchID != "" {
			matchIDs = append(matchIDs, m.MatchID)
		}
	}
	if len(matchIDs) == 0 {
		return nil
	}
	counts, err := s.repo.GetTopMedalsForMatches(ctx, targetXUID, matchIDs, explorerTopMedalsCap)
	if err != nil {
		slog.WarnContext(ctx, "explorer_target_top_medals_local_failed",
			"xuid", targetXUID, "matches", len(matchIDs), "err", err)
		return nil
	}
	items := buildTargetTopMedals(ctx, s.deps.MedalDefs, counts, s.deps.TitleSlug, ctxkeys.Locale(ctx))
	slog.DebugContext(ctx, "explorer_target_top_medals",
		"xuid", targetXUID, "matches", len(matchIDs), "medals", len(items), "source", "local")
	return items
}
