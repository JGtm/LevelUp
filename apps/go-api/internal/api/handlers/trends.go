// Package handlers — TrendsHandler : POST /pages/trends (page Tendances).
//
// Mount crée humacore.NewAPI(r) sur le sous-routeur (préfixe
// /players/{player_slug} + middleware ownership/title hérités). Le corps est lu
// via RawBody pour garder 400 {invalid_json} sur un JSON invalide (un Body typé
// renverrait le 422 de validation Huma) ; un corps vide vaut {}.
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

// TrendsHandler gère POST /pages/trends.
type TrendsHandler struct {
	newSvc      ServiceFactory[port.TrendsService]
	newSquadSvc ContextFactory[port.SquadTrendsService]
}

// NewTrendsHandler crée un TrendsHandler : newSvc sert la vue Solo, newSquadSvc la vue Escouade.
func NewTrendsHandler(
	newSvc ServiceFactory[port.TrendsService], newSquadSvc ContextFactory[port.SquadTrendsService],
) *TrendsHandler {
	return &TrendsHandler{newSvc: newSvc, newSquadSvc: newSquadSvc}
}

// Mount enregistre la route via Huma sur le sous-routeur chi. Le corps est optionnel
// (absent = {}) : MarkRequestBodyOptional, sinon Huma rend 400 sur un corps vide.
func (h *TrendsHandler) Mount(r chi.Router, opts ...humacore.MountOption) {
	api := humacore.NewAPI(r, opts...)
	huma.Post(api, "/pages/trends", h.GetPage, humacore.Op("postTrendsPage", "Tendances (vue et type de partie en body)", "trends"))
	humacore.MarkRequestBodyOptional(api, http.MethodPost, "/pages/trends")
}

// trendsQueryInput : {player_slug} parent + corps brut décodé maison.
type trendsQueryInput struct {
	PlayerSlug string `path:"player_slug"`
	RawBody    []byte
}

type trendsPageOutput struct{ Body domain.TrendsPageResponse }

// GetPage traite POST /api/v1/players/{player_slug}/pages/trends.
//
// Erreurs : JSON invalide -> 400 invalid_json ; requête refusée par Validate -> 400
// invalid_request ; joueur inconnu -> 404 player_not_found ; capacité absente du titre ->
// 503 capability_not_supported ; le reste par mapServiceError. La vue Escouade passe par
// la fabrique d'escouade, la vue Solo n'y touche pas.
func (h *TrendsHandler) GetPage(ctx context.Context, in *trendsQueryInput) (*trendsPageOutput, error) {
	var req domain.TrendsQueryRequest
	if len(bytes.TrimSpace(in.RawBody)) > 0 {
		if err := json.NewDecoder(bytes.NewReader(in.RawBody)).Decode(&req); err != nil {
			return nil, humacore.NewError(http.StatusBadRequest, "invalid_json", "corps JSON invalide")
		}
	}
	if err := req.Validate(); err != nil {
		return nil, humacore.NewError(http.StatusBadRequest, "invalid_request", err.Error())
	}

	var resp domain.TrendsPageResponse
	var svcErr error
	if req.View == domain.TrendsViewSquad {
		svc, xuid, _, err := h.newSquadSvc(ctx, in.PlayerSlug)
		if err != nil {
			return nil, humacore.NewError(http.StatusNotFound, "player_not_found", err.Error())
		}
		resp, svcErr = svc.GetSquadTrends(ctx, xuid, req)
	} else {
		svc, err := h.newSvc(ctx, in.PlayerSlug)
		if err != nil {
			return nil, humacore.NewError(http.StatusNotFound, "player_not_found", err.Error())
		}
		resp, svcErr = svc.GetPage(ctx, req)
	}
	if svcErr != nil {
		if mapped, ok := MapCapabilityError(ctx, svcErr, "trends.page"); ok {
			return nil, mapped
		}
		return nil, mapServiceError(ctx, svcErr, "trends_error")
	}
	return &trendsPageOutput{Body: resp}, nil
}
