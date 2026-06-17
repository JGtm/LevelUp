// Package handlers — SessionCompareHandler : POST /pages/session-compare.
//
// Sprint 33 — contrat API Lot 5.
// MIGRÉ vers Huma (Phase 3b) : Mount crée humacore.NewAPI(r) sur le sous-routeur
// (préfixe /players/{player_slug} + middleware ownership/title hérités, lit
// {player_slug} parent) et enregistre le POST via huma.Post. Logique métier
// inchangée (SessionCompareService), seul le wrapping HTTP change.
//
// Le corps est lu via RawBody (pas de Body typé) pour reproduire EXACTEMENT le
// contrat de décodage d'origine : un JSON invalide renvoie 400 {invalid_json}
// (parse maison) et non le 422 de validation Huma qu'un Body typé produirait.
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

// SessionCompareHandler gère POST /pages/session-compare.
type SessionCompareHandler struct {
	newSvc ServiceFactory[port.SessionCompareService]
}

// NewSessionCompareHandler crée un SessionCompareHandler.
func NewSessionCompareHandler(newSvc ServiceFactory[port.SessionCompareService]) *SessionCompareHandler {
	return &SessionCompareHandler{newSvc: newSvc}
}

// Mount enregistre la route via Huma sur le sous-routeur chi (préfixe
// /players/{player_slug} + middleware ownership/title hérités).
func (h *SessionCompareHandler) Mount(r chi.Router) {
	api := humacore.NewAPI(r)
	huma.Post(api, "/pages/session-compare", h.Compare)
}

// ─── Inputs/Outputs Huma ─────────────────────────────────────────────────────

// sessionCompareInput : {player_slug} parent + corps brut décodé maison.
// RawBody (pas Body typé) → préserve le contrat 400 {invalid_json} sur JSON
// invalide (un Body typé renverrait le 422 de validation Huma).
type sessionCompareInput struct {
	PlayerSlug string `path:"player_slug"`
	RawBody    []byte
}

type sessionCompareOutput struct {
	Body domain.SessionCompareResponse
}

// ─── Endpoint ────────────────────────────────────────────────────────────────

// Compare traite POST /api/v1/players/{player_slug}/pages/session-compare.
func (h *SessionCompareHandler) Compare(ctx context.Context, in *sessionCompareInput) (*sessionCompareOutput, error) {
	svc, err := h.newSvc(ctx, in.PlayerSlug)
	if err != nil {
		return nil, humacore.NewError(http.StatusNotFound, "player_not_found", err.Error())
	}

	var req domain.SessionCompareRequest
	if err := json.NewDecoder(bytes.NewReader(in.RawBody)).Decode(&req); err != nil {
		return nil, humacore.NewError(http.StatusBadRequest, "invalid_json", "corps JSON invalide")
	}

	resp, svcErr := svc.Compare(ctx, req)
	if svcErr != nil {
		return nil, humacore.NewError(http.StatusInternalServerError, "session_compare_error", svcErr.Error())
	}

	return &sessionCompareOutput{Body: resp}, nil
}
