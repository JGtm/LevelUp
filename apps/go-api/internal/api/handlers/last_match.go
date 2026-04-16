// Package handlers — LastMatchHandler : POST /pages/last-match/resolve.
//
// Sprint 33 — contrat API Lot 5.
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

// LastMatchHandler gère POST /pages/last-match/resolve.
type LastMatchHandler struct {
	cfg *config.AppConfig
}

// NewLastMatchHandler crée un LastMatchHandler.
func NewLastMatchHandler(cfg *config.AppConfig) *LastMatchHandler {
	return &LastMatchHandler{cfg: cfg}
}

// Resolve traite POST /api/v1/players/{player_slug}/pages/last-match/resolve.
func (h *LastMatchHandler) Resolve(w http.ResponseWriter, r *http.Request) {
	pdb, err := h.resolvePlayer(r)
	if err != nil {
		writeError(w, http.StatusNotFound, "player_not_found", err.Error())
		return
	}

	var req domain.LastMatchResolveRequest
	if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, 1<<20)).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_json", "corps JSON invalide")
		return
	}

	repo := duckdb.NewStatsRepo(pdb)
	svc := service.NewLastMatchService(repo)

	resp, svcErr := svc.Resolve(r.Context(), req)
	if svcErr != nil {
		writeError(w, http.StatusInternalServerError, "last_match_error", svcErr.Error())
		return
	}

	writeJSON(w, http.StatusOK, resp)
}

func (h *LastMatchHandler) resolvePlayer(r *http.Request) (*duckdb.PlayerDB, error) {
	slug := chi.URLParam(r, "player_slug")
	return config.ResolvePlayer(r.Context(), h.cfg, slug)
}
