// Package handlers — TimeseriesHandler : POST /pages/timeseries.
//
// Sprint 33 — contrat API Lot 4.
// MIGRÉ vers Huma (Phase 3b) : Mount crée humacore.NewAPI(r) sur le sous-routeur
// (préfixe /players/{player_slug} + middleware ownership/title hérités, lit
// {player_slug} parent) et enregistre le POST via huma.Post. Logique métier
// inchangée (TimeseriesService), seul le wrapping HTTP change.
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

// TimeseriesHandler gère POST /pages/timeseries.
type TimeseriesHandler struct {
	newSvc ServiceFactory[port.TimeseriesService]
}

// NewTimeseriesHandler crée un TimeseriesHandler.
func NewTimeseriesHandler(newSvc ServiceFactory[port.TimeseriesService]) *TimeseriesHandler {
	return &TimeseriesHandler{newSvc: newSvc}
}

// Mount enregistre la route via Huma sur le sous-routeur chi (préfixe
// /players/{player_slug} + middleware ownership/title hérités).
func (h *TimeseriesHandler) Mount(r chi.Router, opts ...humacore.MountOption) {
	api := humacore.NewAPI(r, opts...)
	huma.Post(api, "/pages/timeseries", h.GetPage, humacore.Op("postTimeseriesPage", "Séries temporelles (filtres en body)", "timeseries"))
}

// ─── Inputs/Outputs Huma ─────────────────────────────────────────────────────

// timeseriesQueryInput : {player_slug} parent + corps brut décodé maison.
// RawBody (pas Body typé) → préserve le contrat 400 {invalid_json} sur JSON
// invalide (un Body typé renverrait le 422 de validation Huma).
type timeseriesQueryInput struct {
	PlayerSlug string `path:"player_slug"`
	RawBody    []byte
}

type timeseriesPageOutput struct{ Body domain.TimeseriesPageResponse }

// ─── Endpoint ────────────────────────────────────────────────────────────────

// GetPage traite POST /api/v1/players/{player_slug}/pages/timeseries.
func (h *TimeseriesHandler) GetPage(ctx context.Context, in *timeseriesQueryInput) (*timeseriesPageOutput, error) {
	svc, err := h.newSvc(ctx, in.PlayerSlug)
	if err != nil {
		return nil, humacore.NewError(http.StatusNotFound, "player_not_found", err.Error())
	}

	var req domain.TimeseriesQueryRequest
	if err := json.NewDecoder(bytes.NewReader(in.RawBody)).Decode(&req); err != nil {
		return nil, humacore.NewError(http.StatusBadRequest, "invalid_json", "corps JSON invalide")
	}

	resp, svcErr := svc.GetPage(ctx, req)
	if svcErr != nil {
		slog.ErrorContext(ctx, "timeseries: erreur service", "player", in.PlayerSlug, "err", svcErr)
		return nil, humacore.NewError(http.StatusInternalServerError, "timeseries_error", svcErr.Error())
	}

	return &timeseriesPageOutput{Body: resp}, nil
}
