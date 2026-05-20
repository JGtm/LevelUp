// Package handlers — SessionsHandler : endpoint de calcul des sessions.
package handlers

import (
	"fmt"
	"net/http"

	"levelup/go-api/internal/analysis"
	"levelup/go-api/internal/domain"
	"levelup/go-api/internal/port"

	"github.com/go-chi/chi/v5"
)

// SessionsHandler gère les requêtes de calcul des sessions.
type SessionsHandler struct {
	newSvc ServiceFactory[port.SessionsService]
}

// NewSessionsHandler crée un SessionsHandler.
func NewSessionsHandler(newSvc ServiceFactory[port.SessionsService]) *SessionsHandler {
	return &SessionsHandler{newSvc: newSvc}
}

// GetSessions traite GET /api/v1/players/{player_slug}/pages/sessions.
func (h *SessionsHandler) GetSessions(w http.ResponseWriter, r *http.Request) {
	slug := chi.URLParam(r, "player_slug")
	svc, err := h.newSvc(r.Context(), slug)
	if err != nil {
		writeError(r.Context(), w, http.StatusBadRequest, "player_not_found", "joueur introuvable")
		return
	}

	// Options depuis query params (optionnels — valeurs par défaut raisonnables).
	opts := analysis.DefaultSessionOptions()
	if gapStr := r.URL.Query().Get("gap_minutes"); gapStr != "" {
		var gap int
		if _, parseErr := fmt.Sscanf(gapStr, "%d", &gap); parseErr == nil && gap > 0 {
			opts.GapMinutes = gap
		}
	}
	if modeStr := r.URL.Query().Get("mode"); modeStr != "" {
		opts.Mode = domain.SessionComputeMode(modeStr)
	}
	if splitRanked := r.URL.Query().Get("split_ranked"); splitRanked == jsonBoolTrueStr {
		opts.SplitOnRankedChange = true
	}

	resp, svcErr := svc.GetSessions(r.Context(), opts)
	if svcErr != nil {
		writeError(r.Context(), w, http.StatusInternalServerError, "sessions_error", "erreur calcul sessions")
		return
	}

	writeJSON(w, http.StatusOK, resp)
}
