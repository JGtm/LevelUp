// Package service — commendation_totals_service.go : orchestration de la page Totaux
// des commendations NATIVES (Halo 5, AXE B). Groupe les totaux à vie (dernier progress
// absolu par commendation, fournis enrichis par l'adapter) par catégorie.
package service

import (
	"context"
	"errors"
	"sort"

	"levelup/go-api/internal/analysis"
	"levelup/go-api/internal/domain"
	"levelup/go-api/internal/games"
	"levelup/go-api/internal/games/canonical"
)

// CommendationTotalsLoader = la surface adapter consommée (LoadCommendationTotals).
// Satisfaite par *halo_5.DataAdapter (type-assertion côté factory) ; les titres sans
// commendations natives ne l'implémentent pas → loader nil → réponse vide.
type CommendationTotalsLoader interface {
	LoadCommendationTotals(ctx context.Context) ([]canonical.CommendationTotal, error)
}

// CommendationTotalsService groupe les totaux à vie des commendations natives par
// catégorie. loader nil → réponse vide (titre sans capability).
type CommendationTotalsService struct {
	loader CommendationTotalsLoader
}

// NewCommendationTotalsService crée le service. loader nil est toléré (réponse vide).
func NewCommendationTotalsService(loader CommendationTotalsLoader) *CommendationTotalsService {
	return &CommendationTotalsService{loader: loader}
}

// GetTotals retourne les totaux groupés par catégorie. Dégradation gracieuse : loader
// nil ou ErrCapabilityNotSupported → réponse VIDE (pas une erreur), comme un titre
// sans commendations natives.
func (s *CommendationTotalsService) GetTotals(ctx context.Context) (*domain.NativeCommendationsTotalsResponse, error) {
	empty := &domain.NativeCommendationsTotalsResponse{
		Categories: []domain.NativeCommendationCategoryGroup{},
		TotalCount: 0,
	}
	if s == nil || s.loader == nil {
		return empty, nil
	}
	totals, err := s.loader.LoadCommendationTotals(ctx)
	if err != nil {
		if errors.Is(err, games.ErrCapabilityNotSupported) {
			return empty, nil
		}
		return nil, err
	}
	if len(totals) == 0 {
		return empty, nil
	}
	return groupNativeCommendations(totals), nil
}

// groupNativeCommendations groupe par catégorie (items triés total DESC ; catégories
// triées alphabétiquement). TotalCount = nombre de commendations distinctes.
func groupNativeCommendations(totals []canonical.CommendationTotal) *domain.NativeCommendationsTotalsResponse {
	byCat := map[string][]domain.NativeCommendationTotal{}
	for _, t := range totals {
		cat := t.Category
		if cat == "" {
			cat = "AUTRE"
		}
		// Progression À VIE : delta=0 (vitrine, pas un gain de match). On NE filtre
		// PAS les maîtrisées (à l'inverse des snippets de match) — une commendation
		// maîtrisée doit s'afficher pleine/dorée. tier_targets vide → tier_count=0 →
		// front masque l'anneau (dégradation propre).
		tiers := analysis.ParseTierTargets(t.TierTargets)
		tp := analysis.ComputeTierProgression(t.Total, 0, tiers)
		byCat[cat] = append(byCat[cat], domain.NativeCommendationTotal{
			ID: t.ID, Name: t.Name, Category: cat, Total: t.Total, IconURL: t.IconURL,
			ProgressPct:    tp.ProgressPct,
			TierIndex:      tp.TierIndex,
			TierCount:      tp.TierCount,
			NextTierTarget: tp.NextTierTarget,
			IsMastered:     tp.TierCount > 0 && tp.TierIndex >= tp.TierCount,
		})
	}
	cats := make([]string, 0, len(byCat))
	for c := range byCat {
		cats = append(cats, c)
	}
	sort.Strings(cats)
	groups := make([]domain.NativeCommendationCategoryGroup, 0, len(cats))
	for _, c := range cats {
		items := byCat[c]
		sort.SliceStable(items, func(i, j int) bool { return items[i].Total > items[j].Total })
		groups = append(groups, domain.NativeCommendationCategoryGroup{Category: c, Items: items})
	}
	return &domain.NativeCommendationsTotalsResponse{Categories: groups, TotalCount: len(totals)}
}
