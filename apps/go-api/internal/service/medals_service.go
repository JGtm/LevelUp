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

// ghostMedalIDsBySlug : registre slug → allowlist des medal_name_id à masquer côté
// page Médailles (décision V72-33, 2026-07-25 — cf. halo_5.GhostMedalIDs pour le
// détail du constat). Même modèle que medalCategoryResolvers : peuplé UNE fois au
// boot par service.RegisterGhostMedalIDs, lu en read-only ensuite. Un titre non
// enregistré ne masque rien (comportement inchangé, aucune médaille perdue).
var ghostMedalIDsBySlug = map[string]map[int64]bool{}

// RegisterGhostMedalIDs enregistre l'allowlist de medal_name_id à masquer pour un
// titre (boot). slug vide ou ids vide/nil = no-op.
func RegisterGhostMedalIDs(slug string, ids map[int64]bool) {
	if slug == "" || len(ids) == 0 {
		return
	}
	ghostMedalIDsBySlug[slug] = ids
}

// filterGhostMedals retire de items les médailles masquées pour ce titre (V72-33) :
// obtenues en match réel mais sans nom ni icône exploitables (absentes du catalogue
// officiel ET de la référence wiki communautaire). Filtre EXPLICITE par ID (pas de
// heuristique nom/icône vide générique) : le fallback "#<id>" de MergeMedalCatalog
// sert précisément à ne jamais perdre une médaille gagnée dont le catalogue n'a pas
// encore été (re)synchronisé — un filtre générique masquerait aussi ce cas légitime.
// Aucun ID masqué pour ce titre → items retourné inchangé (no-op, aucune allocation).
func filterGhostMedals(items []domain.MedalSummaryItem, ghosts map[int64]bool) []domain.MedalSummaryItem {
	if len(ghosts) == 0 {
		return items
	}
	out := make([]domain.MedalSummaryItem, 0, len(items))
	for _, it := range items {
		if ghosts[it.MedalID] {
			continue
		}
		out = append(out, it)
	}
	return out
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
	items = filterGhostMedals(items, ghostMedalIDsBySlug[titleSlug])
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
