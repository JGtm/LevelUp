// Package service — explorer_target_seasons.go : bucketing "matchs par saison"
// du joueur cible à partir des start_time de ses matchs (shared DB) + les
// plages temporelles de saison (SeasonsCatalog). Joueur local = historique
// complet ; adversaire = matchs observés (communs).
package service

import (
	"time"

	"levelup/go-api/internal/domain"
)

// seasonShortLabel retourne le libellé compact d'une saison (extra.short_label
// si présent, ex. "S13", sinon le Label complet).
func seasonShortLabel(s *SeasonCatalogEntry) string {
	if s.Extra != nil {
		if sl := s.Extra["short_label"]; sl != "" {
			return sl
		}
	}
	return s.Label
}

// buildMatchesPerSeason compte les matchs par saison en rangeant chaque
// start_time dans la première saison dont l'intervalle [Start, End) le contient.
// Les saisons sont parcourues dans l'ordre fourni (DisplayOrder). Seules les
// saisons avec au moins 1 match sont retournées (ordre préservé). nil si vide.
func buildMatchesPerSeason(starts []time.Time, seasons []SeasonCatalogEntry) []domain.SeasonMatchCount {
	if len(starts) == 0 || len(seasons) == 0 {
		return nil
	}
	counts := make(map[string]int, len(seasons))
	for _, t := range starts {
		for i := range seasons {
			s := &seasons[i]
			if !t.Before(s.Start) && (s.End == nil || t.Before(*s.End)) {
				counts[s.ID]++
				break
			}
		}
	}
	out := make([]domain.SeasonMatchCount, 0, len(seasons))
	for i := range seasons {
		s := &seasons[i]
		if n := counts[s.ID]; n > 0 {
			out = append(out, domain.SeasonMatchCount{
				SeasonID:   s.ID,
				SeasonName: seasonShortLabel(s),
				Matches:    n,
			})
		}
	}
	if len(out) == 0 {
		return nil
	}
	return out
}
