// Package handlers — medals.go : handler HTTP de la page Médailles.
//
//	POST /api/v1/players/{player_slug}/pages/medals → MedalsPageResponse
//
// Migré Huma (comme citations.go) : Mount crée humacore.NewAPI(r) sur le sous-routeur
// {player_slug} (middleware ownership/title hérités) et enregistre le POST. Le corps
// est OPTIONNEL (filtre catégorie facultatif), lu via RawBody pour préserver le
// contrat 400 {invalid_body} sur JSON invalide (un Body typé renverrait un 422 Huma).
package handlers

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"

	"github.com/danielgtaylor/huma/v2"
	"github.com/go-chi/chi/v5"

	"levelup/go-api/internal/api/humacore"
	"levelup/go-api/internal/domain"
	"levelup/go-api/internal/port"
)

// MedalsHandler gère l'endpoint de la page Médailles.
type MedalsHandler struct {
	newSvc ContextFactory[port.MedalsService]
}

// NewMedalsHandler crée un MedalsHandler.
func NewMedalsHandler(newSvc ContextFactory[port.MedalsService]) *MedalsHandler {
	return &MedalsHandler{newSvc: newSvc}
}

// Mount enregistre la route via Huma sur le sous-routeur chi (préfixe
// /players/{player_slug} + middleware ownership/title hérités).
func (h *MedalsHandler) Mount(r chi.Router, opts ...humacore.MountOption) {
	api := humacore.NewAPI(r, opts...)
	huma.Post(api, "/pages/medals", h.handleGetMedals)
	humacore.MarkRequestBodyOptional(api, http.MethodPost, "/pages/medals")
}

// medalsInput : {player_slug} parent + corps brut décodé maison (optionnel).
type medalsInput struct {
	PlayerSlug string `path:"player_slug"`
	RawBody    []byte
}

type medalsPageOutput struct{ Body *domain.MedalsPageResponse }

// handleGetMedals retourne la page Médailles (catalogue complet + compteur joueur).
// POST /api/v1/players/{player_slug}/pages/medals
// Body (optionnel) : { "category": "..." }
func (h *MedalsHandler) handleGetMedals(ctx context.Context, in *medalsInput) (*medalsPageOutput, error) {
	svc, xuid, _, err := h.newSvc(ctx, in.PlayerSlug)
	if err != nil {
		return nil, humacore.NewError(http.StatusNotFound, "player_not_found", err.Error())
	}

	var req domain.MedalsPageRequest
	if len(in.RawBody) > 0 {
		if err := json.NewDecoder(bytes.NewReader(in.RawBody)).Decode(&req); err != nil {
			return nil, humacore.NewError(http.StatusBadRequest, "invalid_body", err.Error())
		}
	}

	page, err := svc.GetMedalsPage(ctx, xuid)
	if err != nil {
		return nil, humacore.NewError(http.StatusInternalServerError, "medals_page_error", err.Error())
	}

	if req.Category != "" {
		page = filterMedalsByCategory(page, req.Category)
	}
	return &medalsPageOutput{Body: page}, nil
}

// filterMedalsByCategory restreint la réponse à une clé de catégorie (Medals plats +
// Categories), en recalculant les totaux. Slices toujours initialisées (jamais nil).
func filterMedalsByCategory(page *domain.MedalsPageResponse, category string) *domain.MedalsPageResponse {
	medals := make([]domain.MedalSummaryItem, 0, len(page.Medals))
	for _, m := range page.Medals {
		if m.Category == category {
			medals = append(medals, m)
		}
	}
	cats := make([]domain.MedalCategoryGroup, 0, 1)
	out := &domain.MedalsPageResponse{Medals: medals, Categories: cats}
	for _, g := range page.Categories {
		if g.Category != category {
			continue
		}
		cats = append(cats, g)
		out.EarnedTotal += g.Earned
		out.CatalogTotal += g.Total
		out.TotalCount += g.TotalCount
	}
	out.Categories = cats
	return out
}
