// Package handlers — admin_lusr_gaps.go : endpoints du panneau monitoring « Notes
// LUSR — trous & garde-fou ».
//
//   - GET  /admin/monitoring/lusr-gaps?title=          : rapport trous + garde-fou
//   - POST /admin/monitoring/lusr-gaps/{player}/recompute : replay LUSR d'un joueur
//
// Runners injectés (ServiceRegistry) pour éviter le cycle d'import. RequireAuth +
// RequireAdmin hérités du groupe /admin ; NoStore appliqué au montage.
package handlers

import (
	"context"
	"log/slog"
	"net/http"

	"github.com/danielgtaylor/huma/v2"
	"github.com/go-chi/chi/v5"

	"levelup/go-api/internal/api/humacore"
	"levelup/go-api/internal/domain"
)

// LUSRGapsRunner retourne le rapport de trous LUSR d'un titre (implémenté par
// ServiceRegistry.LUSRGapsReport).
type LUSRGapsRunner func(ctx context.Context, titleSlug string) (domain.AdminLUSRGaps, error)

// LUSRRecomputeRunner déclenche le replay LUSR d'un joueur (implémenté par
// ServiceRegistry.RecomputeLUSRGapsForPlayer). playerRef = slug ou gamertag.
type LUSRRecomputeRunner func(ctx context.Context, titleSlug, playerRef string) (domain.AdminLUSRRecomputeResponse, error)

// AdminLUSRGapsHandler sert le rapport + l'action replay.
type AdminLUSRGapsHandler struct {
	gaps      LUSRGapsRunner
	recompute LUSRRecomputeRunner // nil → action 503
}

// NewAdminLUSRGapsHandler construit le handler. recompute peut être nil
// (dégradation propre : rapport lisible, action indisponible).
func NewAdminLUSRGapsHandler(gaps LUSRGapsRunner, recompute LUSRRecomputeRunner) *AdminLUSRGapsHandler {
	return &AdminLUSRGapsHandler{gaps: gaps, recompute: recompute}
}

// Mount enregistre les routes Huma sur le sous-routeur /admin.
func (h *AdminLUSRGapsHandler) Mount(r chi.Router, opts ...humacore.MountOption) {
	api := humacore.NewAPI(r, opts...)
	huma.Get(api, "/monitoring/lusr-gaps", h.handleGet)
	huma.Post(api, "/monitoring/lusr-gaps/{player}/recompute", h.handleRecompute)
}

type adminLUSRGapsOutput struct {
	Body domain.AdminLUSRGaps
}

// handleGet — GET /admin/monitoring/lusr-gaps?title={slug}.
func (h *AdminLUSRGapsHandler) handleGet(ctx context.Context, in *titleInput) (*adminLUSRGapsOutput, error) {
	titleSlug := titleOrDefaultSlug(in.Title)
	resp, err := h.gaps(ctx, titleSlug)
	if err != nil {
		slog.ErrorContext(ctx, "admin_monitoring: lusr gaps failed", "title", titleSlug, "err", err)
		return nil, humacore.NewError(http.StatusInternalServerError, "lusr_gaps_error",
			"Impossible de calculer les trous LUSR.")
	}
	return &adminLUSRGapsOutput{Body: resp}, nil
}

// lusrRecomputeInput : {player} + ?title= optionnel.
type lusrRecomputeInput struct {
	Player string `path:"player"`
	Title  string `query:"title"`
}
type adminLUSRRecomputeOutput struct {
	Body domain.AdminLUSRRecomputeResponse
}

// handleRecompute — POST /admin/monitoring/lusr-gaps/{player}/recompute.
// Replay LUSR chronologique complet (comble les trous d'intérieur).
func (h *AdminLUSRGapsHandler) handleRecompute(ctx context.Context, in *lusrRecomputeInput) (*adminLUSRRecomputeOutput, error) {
	if h.recompute == nil {
		return nil, humacore.NewError(http.StatusServiceUnavailable, "lusr_recompute_unavailable",
			"Le replay LUSR n'est pas disponible (moteur non câblé).")
	}
	titleSlug := titleOrDefaultSlug(in.Title)
	resp, err := h.recompute(ctx, titleSlug, in.Player)
	if err != nil {
		slog.ErrorContext(ctx, "admin_monitoring: lusr recompute failed",
			"title", titleSlug, "player", in.Player, "err", err)
		return nil, humacore.NewError(http.StatusInternalServerError, "lusr_recompute_error",
			"Le replay LUSR a échoué.")
	}
	return &adminLUSRRecomputeOutput{Body: resp}, nil
}
