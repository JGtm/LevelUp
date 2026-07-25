// Package handlers — TeammatesHandler : POST /pages/teammates.
//
// Sprint 33 — contrat API Lot 4.
// MIGRÉ vers Huma (Phase 3b) : Mount crée humacore.NewAPI(r) sur le sous-routeur
// (préfixe /players/{player_slug} + middleware ownership/title hérités, lit
// {player_slug} parent) et enregistre le POST via huma.Post. Logique métier
// inchangée (TeammatesService), seul le wrapping HTTP change.
//
// Le corps est lu via RawBody (pas de Body typé) pour reproduire EXACTEMENT le
// contrat de décodage d'origine : un JSON invalide renvoie 400 {invalid_json}
// (parse maison) et non le 422 de validation Huma qu'un Body typé produirait.
package handlers

import (
	"bytes"
	"context"
	"encoding/json"
	"log/slog"
	"net/http"

	"github.com/danielgtaylor/huma/v2"
	"github.com/go-chi/chi/v5"

	"levelup/go-api/internal/api/humacore"
	"levelup/go-api/internal/domain"
	"levelup/go-api/internal/port"
)

// TeammatesHandler gère POST /pages/teammates.
type TeammatesHandler struct {
	newSvc ContextFactory[port.TeammatesService]
}

// NewTeammatesHandler crée un TeammatesHandler.
func NewTeammatesHandler(newSvc ContextFactory[port.TeammatesService]) *TeammatesHandler {
	return &TeammatesHandler{newSvc: newSvc}
}

// Mount enregistre la route via Huma sur le sous-routeur chi (préfixe
// /players/{player_slug} + middleware ownership/title hérités).
func (h *TeammatesHandler) Mount(r chi.Router, opts ...humacore.MountOption) {
	api := humacore.NewAPI(r, opts...)
	huma.Post(api, "/pages/teammates", h.handleGetPage, humacore.Op("postTeammatesPage", "Analyse coéquipiers (filtres en body)", "teammates"))
}

// ─── Inputs/Outputs Huma ─────────────────────────────────────────────────────

// teammatesQueryInput : {player_slug} parent + corps brut décodé maison.
// RawBody (pas Body typé) → préserve le contrat 400 {invalid_json} sur JSON
// invalide (un Body typé renverrait le 422 de validation Huma).
type teammatesQueryInput struct {
	PlayerSlug string `path:"player_slug"`
	RawBody    []byte
}

type teammatesPageOutput struct{ Body domain.TeammatesPageResponse }

// ─── Endpoint ────────────────────────────────────────────────────────────────

// handleGetPage traite POST /api/v1/players/{player_slug}/pages/teammates.
func (h *TeammatesHandler) handleGetPage(ctx context.Context, in *teammatesQueryInput) (*teammatesPageOutput, error) {
	svc, xuid, _, err := h.newSvc(ctx, in.PlayerSlug)
	if err != nil {
		return nil, humacore.NewError(http.StatusNotFound, "player_not_found", err.Error())
	}

	var req domain.TeammatesQueryRequest
	if err := json.NewDecoder(bytes.NewReader(in.RawBody)).Decode(&req); err != nil {
		return nil, humacore.NewError(http.StatusBadRequest, "invalid_json", "corps JSON invalide")
	}

	resp, svcErr := svc.GetPage(ctx, xuid, req)
	if svcErr != nil {
		slog.ErrorContext(ctx, "teammates: erreur service", "player", in.PlayerSlug, "err", svcErr)
		return nil, humacore.NewError(http.StatusInternalServerError, "teammates_error", svcErr.Error())
	}

	return &teammatesPageOutput{Body: resp}, nil
}
