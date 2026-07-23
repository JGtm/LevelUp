// Package service — medals_service.go : orchestration de la page Médailles.
//
// Compose le catalogue du titre (medal_definitions) avec les totaux obtenus par le
// joueur (medals_earned), catégorise chaque médaille via le MedalCategoryResolver du
// titre courant (baseline par défaut, enrichissement Halo Infinite si enregistré) et
// enrichit l'image (PNG Infinite / sprite Halo 5).
package service

import (
	"context"
	"log/slog"

	"levelup/go-api/internal/analysis"
	"levelup/go-api/internal/assets/static"
	"levelup/go-api/internal/ctxkeys"
	"levelup/go-api/internal/domain"
	"levelup/go-api/internal/port"
)

// Compile-time check : MedalsService implémente port.MedalsService.
var _ port.MedalsService = (*MedalsService)(nil)

// medalCategoryResolvers : registre slug → resolver, peuplé UNE fois au boot par le
// package de chaque titre (service.RegisterMedalCategoryResolver) et lu en read-only
// ensuite — même modèle que csrBadgeResolver / medalSpriteResolver (aucun accès
// concurrent au-delà du set initial). Un titre non enregistré utilise la baseline.
var medalCategoryResolvers = map[string]port.MedalCategoryResolver{}

// RegisterMedalCategoryResolver enregistre le resolver enrichi d'un titre (boot).
// slug vide ou resolver nil = no-op (le titre retombe alors sur la baseline).
func RegisterMedalCategoryResolver(slug string, r port.MedalCategoryResolver) {
	if slug == "" || r == nil {
		return
	}
	medalCategoryResolvers[slug] = r
}

// medalCategoryResolverFor retourne le resolver du titre, ou la baseline par défaut
// (regroupement natif par medal_type). Data-driven : jamais de comparaison de slug.
func medalCategoryResolverFor(slug string) port.MedalCategoryResolver {
	if r, ok := medalCategoryResolvers[slug]; ok && r != nil {
		return r
	}
	return analysis.BaselineMedalCategoryResolver{}
}

// MedalsService orchestre les données de la page Médailles.
type MedalsService struct {
	repo port.MedalsRepository
}

// NewMedalsService crée un MedalsService avec le repository injecté.
func NewMedalsService(repo port.MedalsRepository) *MedalsService {
	return &MedalsService{repo: repo}
}

// GetMedalsPage construit la réponse : catalogue complet du titre + compteur obtenu
// par le joueur (0 = jamais), regroupé par catégorie/super-section.
func (s *MedalsService) GetMedalsPage(ctx context.Context, playerXUID string) (*domain.MedalsPageResponse, error) {
	locale := ctxkeys.Locale(ctx)
	titleSlug := ctxkeys.TitleSlug(ctx)

	catalog, err := s.repo.ListAllMedals(ctx, locale)
	if err != nil {
		return nil, err
	}

	earned, err := s.repo.LoadMedalTotals(ctx, playerXUID)
	if err != nil {
		// Dégradation best-effort : catalogue affiché avec compteurs à 0. Loggé avant
		// dégradation (jamais d'erreur avalée en silence).
		slog.WarnContext(ctx, "medals: totaux indisponibles, catalogue seul", "err", err, "player", playerXUID)
		earned = nil
	}

	resolver := medalCategoryResolverFor(titleSlug)
	items := analysis.MergeMedalCatalog(catalog, earned, resolver.Resolve)
	for i := range items {
		enrichMedalImage(&items[i], titleSlug)
	}
	groups := analysis.GroupMedalsByCategory(items)

	resp := &domain.MedalsPageResponse{Medals: items, Categories: groups}
	for i := range groups {
		resp.EarnedTotal += groups[i].Earned
		resp.CatalogTotal += groups[i].Total
		resp.TotalCount += groups[i].TotalCount
	}

	// Init [] plutôt que nil : un slice nil sérialise en JSON `null` et crashe le
	// front (testutil.RequireNoNilSlicesWithoutOmitempty).
	if resp.Medals == nil {
		resp.Medals = []domain.MedalSummaryItem{}
	}
	if resp.Categories == nil {
		resp.Categories = []domain.MedalCategoryGroup{}
	}
	return resp, nil
}

// enrichMedalImage renseigne l'image de la médaille de façon title-aware : sprite
// (Halo 5) ou URL PNG (Halo Infinite). Miroir de buildTargetTopMedals.
func enrichMedalImage(item *domain.MedalSummaryItem, titleSlug string) {
	if titleSlug == "" {
		return
	}
	png, sp := static.MedalImage(titleSlug, item.MedalID)
	if sp != nil {
		item.SpriteSheet = sp.SheetURL
		item.SpriteLeft, item.SpriteTop = sp.Left, sp.Top
		item.SpriteWidth, item.SpriteHeight = sp.Width, sp.Height
		return
	}
	if png != "" {
		item.ImageURL = &png
	}
}
