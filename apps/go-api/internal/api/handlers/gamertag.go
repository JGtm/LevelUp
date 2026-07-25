// Package handlers — GamertagHandler : GET /api/v1/directory/gamertags/search.
//
// Route MIGRÉE vers Huma (Phase 3b, registerGamertagHuma dans le package api,
// shape query-param). La logique métier (recherche + normalisation) reste ici
// via Query ; le wrapping HTTP (query param + mapping 503/500) vit dans
// api/huma_routes.go.
package handlers

import (
	"context"
	"errors"
	"strings"

	"levelup/go-api/internal/ctxkeys"
	"levelup/go-api/internal/domain"
	"levelup/go-api/internal/port"
)

// ErrGamertagSearchUnavailable : le service de recherche n'est pas câblé (shared
// DB absente) → 503 côté HTTP.
var ErrGamertagSearchUnavailable = errors.New("gamertag search unavailable")

// GamertagHandler gère GET /api/v1/directory/gamertags/search?q=.
type GamertagHandler struct {
	svc port.GamertagSearchService
}

// NewGamertagHandler crée un GamertagHandler.
func NewGamertagHandler(svc port.GamertagSearchService) *GamertagHandler {
	return &GamertagHandler{svc: svc}
}

// Query cherche les gamertags correspondant à q (trim interne). Items jamais nil.
// Retourne ErrGamertagSearchUnavailable si le service n'est pas câblé (503) ;
// une query vide court-circuite avec une réponse vide (200, pas d'appel service).
//
// live arme le repli LIVE (résolution Xbox d'un joueur jamais croisé) : défaut false
// (recherche locale rapide, pour le typeahead) ; true uniquement sur intention explicite
// de l'utilisateur (« Rechercher sur Xbox »). Posé dans le contexte pour le décorateur
// LiveFallbackGamertagSearch (challenge V72-24). Les autres appelants de l'endpoint
// (setup/admin) ne passent pas live → comportement local seul par défaut.
func (h *GamertagHandler) Query(ctx context.Context, q string, live bool) (domain.GamertagSearchResponse, error) {
	if h.svc == nil {
		return domain.GamertagSearchResponse{}, ErrGamertagSearchUnavailable
	}
	q = strings.TrimSpace(q)
	if q == "" {
		return domain.GamertagSearchResponse{Query: q, Items: []domain.GamertagSearchResult{}}, nil
	}
	items, err := h.svc.Search(ctxkeys.WithGamertagLiveSearch(ctx, live), q)
	if err != nil {
		return domain.GamertagSearchResponse{}, err
	}
	if items == nil {
		items = []domain.GamertagSearchResult{}
	}
	return domain.GamertagSearchResponse{Query: q, Items: items}, nil
}
