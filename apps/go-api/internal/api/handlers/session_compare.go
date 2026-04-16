// Package handlers — SessionCompareHandler : POST /pages/session-compare.
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

// SessionCompareHandler gère POST /pages/session-compare.
type SessionCompareHandler struct {
	cfg *config.AppConfig
}

// NewSessionCompareHandler crée un SessionCompareHandler.
func NewSessionCompareHandler(cfg *config.AppConfig) *SessionCompareHandler {
	return &SessionCompareHandler{cfg: cfg}
}

// Compare traite POST /api/v1/players/{player_slug}/pages/session-compare.
func (h *SessionCompareHandler) Compare(w http.ResponseWriter, r *http.Request) {
	pdb, err := h.resolvePlayer(r)
	if err != nil {
		writeError(w, http.StatusNotFound, "player_not_found", err.Error())
		return
	}

	var req domain.SessionCompareRequest
	if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, 1<<20)).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_json", "corps JSON invalide")
		return
	}

	statsRepo := duckdb.NewStatsRepo(pdb)
	sessionsRepo := duckdb.NewSessionsRepo(pdb)
	svc := service.NewSessionCompareService(sessionsRepo, statsRepo)

	resp, svcErr := svc.Compare(r.Context(), req)
	if svcErr != nil {
		writeError(w, http.StatusInternalServerError, "session_compare_error", svcErr.Error())
		return
	}

	writeJSON(w, http.StatusOK, resp)
}

func (h *SessionCompareHandler) resolvePlayer(r *http.Request) (*duckdb.PlayerDB, error) {
	slug := chi.URLParam(r, "player_slug")
	return config.ResolvePlayer(r.Context(), h.cfg, slug)
}
