// Package handlers — StatsHandler : endpoint des séries temporelles et stats analytiques.
//
// MIGRÉ vers Huma (Phase 3b) : Mount crée humacore.NewAPI(r) sur le sous-routeur
// (préfixe /players/{player_slug} + middleware ownership/title hérités, lit
// {player_slug} parent) et enregistre le POST via huma.Post. Logique métier
// inchangée (StatsService), seul le wrapping HTTP change.
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

// StatsHandler gère les requêtes de stats/séries analytiques.
type StatsHandler struct {
	newSvc ServiceFactory[port.StatsService]
}

// NewStatsHandler crée un StatsHandler.
func NewStatsHandler(newSvc ServiceFactory[port.StatsService]) *StatsHandler {
	return &StatsHandler{newSvc: newSvc}
}

// Mount enregistre la route via Huma sur le sous-routeur chi (préfixe
// /players/{player_slug} + middleware ownership/title hérités).
func (h *StatsHandler) Mount(r chi.Router, opts ...humacore.MountOption) {
	api := humacore.NewAPI(r, opts...)
	huma.Post(api, "/pages/stats/query", h.handleGetPage, humacore.Op("postStatsQuery", "Séries temporelles (legacy — voir aussi POST /pages/timeseries)", "timeseries"))
}

// ─── Inputs/Outputs Huma ─────────────────────────────────────────────────────

// statsQueryInput : {player_slug} parent + corps brut décodé maison.
// RawBody (pas Body typé) → préserve le contrat 400 {invalid_json} sur JSON
// invalide (un Body typé renverrait le 422 de validation Huma).
type statsQueryInput struct {
	PlayerSlug string `path:"player_slug"`
	RawBody    []byte
}

type statsPageOutput struct{ Body domain.StatsPageResponse }

// ─── Endpoint ────────────────────────────────────────────────────────────────

// handleGetPage traite POST /api/v1/players/{player_slug}/pages/stats/query.
//
// Corps JSON : { "tab": "win_loss|accuracy|objective|form|lusr|all", "mode": "period|sessions" }
func (h *StatsHandler) handleGetPage(ctx context.Context, in *statsQueryInput) (*statsPageOutput, error) {
	svc, err := h.newSvc(ctx, in.PlayerSlug)
	if err != nil {
		return nil, humacore.NewError(http.StatusBadRequest, "player_not_found", "joueur introuvable")
	}

	var req domain.StatsQueryRequest
	if err := json.NewDecoder(bytes.NewReader(in.RawBody)).Decode(&req); err != nil {
		return nil, humacore.NewError(http.StatusBadRequest, "invalid_json", "corps JSON invalide")
	}
	if req.Tab == "" {
		req.Tab = "win_loss"
	}

	resp, svcErr := svc.GetPage(ctx, req)
	if svcErr != nil {
		return nil, humacore.NewError(http.StatusInternalServerError, "stats_error", "erreur chargement stats")
	}

	return &statsPageOutput{Body: resp}, nil
}
