// Package handlers — citations.go : handlers HTTP pour les pages Citations et Commendations.
//
// Endpoints :
//
//	POST /api/v1/players/{player_slug}/pages/citations        → CitationsPageResponse
//	POST /api/v1/players/{player_slug}/pages/commendations    → CommendationsPageResponse
//
// MIGRÉ vers Huma (Phase 3b) : Mount crée humacore.NewAPI(r) sur le sous-routeur
// (préfixe /players/{player_slug} + middleware ownership/title hérités, lit
// {player_slug} parent) et enregistre les 2 POST via huma.Post. Logique métier
// inchangée (CitationsService), seul le wrapping HTTP change.
//
// Le corps est lu via RawBody (pas de Body typé) pour reproduire EXACTEMENT le
// contrat de décodage d'origine : un JSON invalide renvoie 400 {invalid_body}
// (parse maison, uniquement si un corps est présent) et non le 422 de validation
// Huma qu'un Body typé produirait.
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

// CitationsHandler gère les endpoints des pages Citations et Commendations.
type CitationsHandler struct {
	newSvc ContextFactory[port.CitationsService]
}

// NewCitationsHandler crée un CitationsHandler.
func NewCitationsHandler(newSvc ContextFactory[port.CitationsService]) *CitationsHandler {
	return &CitationsHandler{newSvc: newSvc}
}

// Mount enregistre les 2 routes via Huma sur le sous-routeur chi (préfixe
// /players/{player_slug} + middleware ownership/title hérités).
func (h *CitationsHandler) Mount(r chi.Router, opts ...humacore.MountOption) {
	api := humacore.NewAPI(r, opts...)
	huma.Post(api, "/pages/citations", h.handleGetCitations, humacore.Op("postCitations", "Citations (filtres en body)", "career"))
	huma.Post(api, "/pages/commendations", h.handleGetCommendations, humacore.Op("postCommendations", "Commendations et médailles par catégorie", "career"))
	// Body OPTIONNEL (décodé seulement si présent) : RawBody est requis par défaut
	// côté Huma → on le rend optionnel pour préserver le 200 sur corps absent.
	humacore.MarkRequestBodyOptional(api, http.MethodPost, "/pages/citations")
	humacore.MarkRequestBodyOptional(api, http.MethodPost, "/pages/commendations")
}

// ─── Inputs/Outputs Huma ─────────────────────────────────────────────────────

// citationsInput : {player_slug} parent + corps brut décodé maison.
// RawBody (pas Body typé) → préserve le contrat 400 {invalid_body} sur JSON
// invalide (un Body typé renverrait le 422 de validation Huma). Corps optionnel :
// décodé uniquement si présent.
type citationsInput struct {
	PlayerSlug string `path:"player_slug"`
	RawBody    []byte
}

type citationsPageOutput struct{ Body *domain.CitationsPageResponse }
type commendationsPageOutput struct {
	Body *domain.CommendationsPageResponse
}

// ─── Endpoints ───────────────────────────────────────────────────────────────

// handleGetCitations retourne la page Citations (accomplissements personnels).
// POST /api/v1/players/{player_slug}/pages/citations
// Body (optionnel) : { "category": "..." }
func (h *CitationsHandler) handleGetCitations(ctx context.Context, in *citationsInput) (*citationsPageOutput, error) {
	svc, _, _, err := h.newSvc(ctx, in.PlayerSlug)
	if err != nil {
		return nil, humacore.NewError(http.StatusNotFound, "player_not_found", err.Error())
	}

	var req domain.CitationsPageRequest
	// Body optionnel : décoder uniquement si présent.
	if len(in.RawBody) > 0 {
		if err := json.NewDecoder(bytes.NewReader(in.RawBody)).Decode(&req); err != nil {
			return nil, humacore.NewError(http.StatusBadRequest, "invalid_body", err.Error())
		}
	}

	page, err := svc.GetCitationsPage(ctx)
	if err != nil {
		return nil, humacore.NewError(http.StatusInternalServerError, "citations_page_error", err.Error())
	}

	// Filtrage par catégorie si demandé.
	if req.Category != "" {
		page = filterCitationsByCategory(page, req.Category)
	}

	return &citationsPageOutput{Body: page}, nil
}

// handleGetCommendations retourne la page Commendations (médailles gagnées).
// POST /api/v1/players/{player_slug}/pages/commendations
// Body (optionnel) : { "category": "..." }
func (h *CitationsHandler) handleGetCommendations(ctx context.Context, in *citationsInput) (*commendationsPageOutput, error) {
	svc, xuid, _, err := h.newSvc(ctx, in.PlayerSlug)
	if err != nil {
		return nil, humacore.NewError(http.StatusNotFound, "player_not_found", err.Error())
	}

	var req domain.CommendationsPageRequest
	if len(in.RawBody) > 0 {
		if err := json.NewDecoder(bytes.NewReader(in.RawBody)).Decode(&req); err != nil {
			return nil, humacore.NewError(http.StatusBadRequest, "invalid_body", err.Error())
		}
	}

	page, err := svc.GetCommendationsPage(ctx, xuid)
	if err != nil {
		return nil, humacore.NewError(http.StatusInternalServerError, "commendations_page_error", err.Error())
	}

	// Filtrage par catégorie si demandé.
	if req.Category != "" {
		page = filterCommendationsByCategory(page, req.Category)
	}

	return &commendationsPageOutput{Body: page}, nil
}

// ---------------------------------------------------------------------------
// Helpers de filtrage
// ---------------------------------------------------------------------------

// filterCitationsByCategory filtre les items Citations sur une catégorie.
func filterCitationsByCategory(page *domain.CitationsPageResponse, category string) *domain.CitationsPageResponse {
	filtered := make([]domain.CitationItem, 0, len(page.Citations))
	for _, c := range page.Citations {
		if c.Category == category {
			filtered = append(filtered, c)
		}
	}
	return &domain.CitationsPageResponse{
		Citations:  filtered,
		Categories: page.Categories,
		TotalCount: len(filtered),
	}
}

// filterCommendationsByCategory filtre les Commendations sur une catégorie.
func filterCommendationsByCategory(page *domain.CommendationsPageResponse, category string) *domain.CommendationsPageResponse {
	filtered := make([]domain.CommendationCategory, 0)
	total := 0
	for _, cat := range page.Categories {
		if cat.Category == category {
			filtered = append(filtered, cat)
			total += cat.Total
		}
	}
	return &domain.CommendationsPageResponse{
		Categories: filtered,
		TotalCount: total,
	}
}
