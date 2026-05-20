// Package handlers — AchievementsHandler : GET .../pages/achievements.
package handlers

import (
	"log/slog"
	"net/http"

	"github.com/go-chi/chi/v5"

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

// GetAchievementsPage retourne la liste fusionnée définitions+progression du joueur.
func (h *AchievementsHandler) GetAchievementsPage(w http.ResponseWriter, r *http.Request) {
	slug := chi.URLParam(r, "player_slug")
	ctx := r.Context()

	svc, err := h.newSvc(ctx, slug)
	if err != nil {
		writeError(r.Context(), w, http.StatusNotFound, "player_not_found", err.Error())
		return
	}

	resp, err := svc.GetAchievementsPage(ctx)
	if err != nil {
		slog.ErrorContext(ctx, "achievements handler failed",
			"err", err, "player", slug, "endpoint", "GET /pages/achievements")
		writeError(r.Context(), w, http.StatusInternalServerError, "achievements_error", err.Error())
		return
	}
	writeJSON(w, http.StatusOK, resp)
}
