// Package handlers — MatchHistoryHandler : POST .../pages/match-history/query.
package handlers

import (
	"encoding/json"
	"net/http"

	"github.com/go-chi/chi/v5"

	"levelup/go-api/internal/config"
	"levelup/go-api/internal/domain"
	"levelup/go-api/internal/platform/duckdb"
	"levelup/go-api/internal/service"
)

// MatchHistoryHandler gère POST .../pages/match-history/query.
type MatchHistoryHandler struct {
	cfg *config.AppConfig
}

// NewMatchHistoryHandler crée un MatchHistoryHandler.
func NewMatchHistoryHandler(cfg *config.AppConfig) *MatchHistoryHandler {
	return &MatchHistoryHandler{cfg: cfg}
}

// Query retourne la page d'historique paginée et filtrée.
func (h *MatchHistoryHandler) Query(w http.ResponseWriter, r *http.Request) {
	slug := chi.URLParam(r, "player_slug")
	pdb, err := config.ResolvePlayer(r.Context(), h.cfg, slug)
	if err != nil {
		writeError(w, http.StatusNotFound, "player_not_found", err.Error())
		return
	}

	var req domain.MatchHistoryQueryRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_body", err.Error())
		return
	}

	repo := duckdb.NewMatchHistoryRepo(pdb)
	svc := service.NewMatchHistoryService(repo, pdb.Gamertag)
	resp, err := svc.GetPage(r.Context(), req)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "match_history_error", err.Error())
		return
	}
	writeJSON(w, http.StatusOK, resp)
}
