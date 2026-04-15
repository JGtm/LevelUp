// Package handlers — MatchViewHandler : GET .../matches/{match_id}.
package handlers

import (
	"net/http"

	"github.com/go-chi/chi/v5"

	"levelup/go-api/internal/config"
	"levelup/go-api/internal/platform/duckdb"
	"levelup/go-api/internal/service"
)

// MatchViewHandler gère GET /players/{player_slug}/matches/{match_id}.
type MatchViewHandler struct {
	cfg *config.AppConfig
}

// NewMatchViewHandler crée un MatchViewHandler.
func NewMatchViewHandler(cfg *config.AppConfig) *MatchViewHandler {
	return &MatchViewHandler{cfg: cfg}
}

// GetMatchView retourne la vue détaillée d'un match pour un joueur.
// GET /api/v1/players/{player_slug}/matches/{match_id}
func (h *MatchViewHandler) GetMatchView(w http.ResponseWriter, r *http.Request) {
	pdb, err := h.resolvePlayer(r)
	if err != nil {
		writeError(w, http.StatusNotFound, "player_not_found", err.Error())
		return
	}

	matchID := chi.URLParam(r, "match_id")
	if matchID == "" {
		writeError(w, http.StatusBadRequest, "missing_match_id", "match_id est requis")
		return
	}

	repo := duckdb.NewMatchViewRepo(pdb, pdb.XUID)
	svc := service.NewMatchViewService(repo, pdb.XUID)

	resp, err := svc.GetMatchView(r.Context(), matchID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "match_view_error", err.Error())
		return
	}
	writeJSON(w, http.StatusOK, resp)
}

func (h *MatchViewHandler) resolvePlayer(r *http.Request) (*duckdb.PlayerDB, error) {
	slug := chi.URLParam(r, "player_slug")
	return config.ResolvePlayer(r.Context(), h.cfg, slug)
}
