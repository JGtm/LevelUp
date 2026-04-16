// Package handlers — SessionsHandler : endpoint de calcul des sessions.
package handlers

import (
	"fmt"
	"net/http"

	"levelup/go-api/internal/analysis"
	"levelup/go-api/internal/config"
	"levelup/go-api/internal/domain"
	"levelup/go-api/internal/platform/duckdb"
	"levelup/go-api/internal/service"

	"github.com/go-chi/chi/v5"
)

// SessionsHandler gère les requêtes de calcul des sessions.
type SessionsHandler struct {
	cfg *config.AppConfig
}

// NewSessionsHandler crée un SessionsHandler.
func NewSessionsHandler(cfg *config.AppConfig) *SessionsHandler {
	return &SessionsHandler{cfg: cfg}
}

// GetSessions traite GET /api/v1/players/{player_slug}/pages/sessions.
func (h *SessionsHandler) GetSessions(w http.ResponseWriter, r *http.Request) {
	pdb, err := h.resolvePlayer(r)
	if err != nil {
		writeError(w, http.StatusBadRequest, "player_not_found", "joueur introuvable")
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
	if splitRanked := r.URL.Query().Get("split_ranked"); splitRanked == "true" {
		opts.SplitOnRankedChange = true
	}

	repo := duckdb.NewSessionsRepo(pdb)
	svc := service.NewSessionsService(repo)

	resp, svcErr := svc.GetSessions(r.Context(), opts)
	if svcErr != nil {
		writeError(w, http.StatusInternalServerError, "sessions_error", "erreur calcul sessions")
		return
	}

	writeJSON(w, http.StatusOK, resp)
}

func (h *SessionsHandler) resolvePlayer(r *http.Request) (*duckdb.PlayerDB, error) {
	slug := chi.URLParam(r, "player_slug")
	return config.ResolvePlayer(r.Context(), h.cfg, slug)
}
