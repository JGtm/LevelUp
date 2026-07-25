// Package handlers — AchievementsHandler : GET .../pages/achievements.
//
// MIGRÉ vers Huma (Phase 3b) : Mount crée humacore.NewAPI(r) sur le sous-routeur
// chi (préfixe /players/{player_slug} + middleware ownership/title hérités, lit
// {player_slug} parent) et enregistre la route via huma.Get. Logique métier
// inchangée (service achievements), seul le wrapping HTTP change.
package handlers

import (
	"context"
	"log/slog"
	"net/http"

	"github.com/danielgtaylor/huma/v2"
	"github.com/go-chi/chi/v5"

	"levelup/go-api/internal/api/humacore"
	"levelup/go-api/internal/domain"
	"levelup/go-api/internal/port"
)

// AchievementsHandler gère l'endpoint de la page Achievements Xbox.
type AchievementsHandler struct {
	newSvc ServiceFactory[port.AchievementsService]
}

// NewAchievementsHandler crée un AchievementsHandler avec factory injectée.
func NewAchievementsHandler(newSvc ServiceFactory[port.AchievementsService]) *AchievementsHandler {
	return &AchievementsHandler{newSvc: newSvc}
}

// Mount enregistre la route via Huma sur le sous-routeur chi (préfixe
// /players/{player_slug} + middleware ownership/title hérités).
func (h *AchievementsHandler) Mount(r chi.Router, opts ...humacore.MountOption) {
	api := humacore.NewAPI(r, opts...)
	huma.Get(api, "/pages/achievements", h.GetAchievementsPage)
}

// ─── Inputs/Outputs Huma ─────────────────────────────────────────────────────

// achievementsInput : path param parent {player_slug}.
type achievementsInput struct {
	PlayerSlug string `path:"player_slug"`
}

type achievementsOutput struct {
	Body domain.AchievementsPageResponse
}

// ─── Endpoint ────────────────────────────────────────────────────────────────

// GetAchievementsPage retourne la liste fusionnée définitions+progression du joueur.
func (h *AchievementsHandler) GetAchievementsPage(ctx context.Context, in *achievementsInput) (*achievementsOutput, error) {
	svc, err := h.newSvc(ctx, in.PlayerSlug)
	if err != nil {
		return nil, humacore.NewError(http.StatusNotFound, "player_not_found", err.Error())
	}

	resp, err := svc.GetAchievementsPage(ctx)
	if err != nil {
		slog.ErrorContext(ctx, "achievements handler failed",
			"err", err, "player", in.PlayerSlug, "endpoint", "GET /pages/achievements")
		return nil, humacore.NewError(http.StatusInternalServerError, "achievements_error", err.Error())
	}
	return &achievementsOutput{Body: resp}, nil
}
