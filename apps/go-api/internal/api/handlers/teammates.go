// Package handlers — TeammatesHandler : POST /pages/teammates.
//
// Sprint 33 — contrat API Lot 4.
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

// TeammatesHandler gère POST /pages/teammates.
type TeammatesHandler struct {
	cfg *config.AppConfig
}

// NewTeammatesHandler crée un TeammatesHandler.
func NewTeammatesHandler(cfg *config.AppConfig) *TeammatesHandler {
	return &TeammatesHandler{cfg: cfg}
}

// GetPage traite POST /api/v1/players/{player_slug}/pages/teammates.
func (h *TeammatesHandler) GetPage(w http.ResponseWriter, r *http.Request) {
	pdb, err := h.resolvePlayer(r)
	if err != nil {
		writeError(w, http.StatusNotFound, "player_not_found", err.Error())
		return
	}

	var req domain.TeammatesQueryRequest
	if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, 1<<20)).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_json", "corps JSON invalide")
		return
	}

	repo := duckdb.NewSquadRepo(pdb)
	svc := service.NewTeammatesService(repo)

	resp, svcErr := svc.GetPage(r.Context(), pdb.XUID, req)
	if svcErr != nil {
		writeError(w, http.StatusInternalServerError, "teammates_error", svcErr.Error())
		return
	}

	writeJSON(w, http.StatusOK, resp)
}

func (h *TeammatesHandler) resolvePlayer(r *http.Request) (*duckdb.PlayerDB, error) {
	slug := chi.URLParam(r, "player_slug")
	return config.ResolvePlayer(r.Context(), h.cfg, slug)
}
