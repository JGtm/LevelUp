// Package handlers — CareerHandler : GET .../pages/career[/top-matches|/encounters].
package handlers

import (
	"net/http"

	"github.com/go-chi/chi/v5"

	"levelup/go-api/internal/config"
	"levelup/go-api/internal/platform/duckdb"
	"levelup/go-api/internal/service"
)

// CareerHandler gère les 3 endpoints de la page Carrière.
type CareerHandler struct {
	cfg *config.AppConfig
}

// NewCareerHandler crée un CareerHandler.
func NewCareerHandler(cfg *config.AppConfig) *CareerHandler {
	return &CareerHandler{cfg: cfg}
}

// GetCareer retourne la réponse complète de la page Carrière.
func (h *CareerHandler) GetCareer(w http.ResponseWriter, r *http.Request) {
	pdb, err := h.resolvePlayer(r)
	if err != nil {
		writeError(w, http.StatusNotFound, "player_not_found", err.Error())
		return
	}
	svc := service.NewCareerService(duckdb.NewCareerRepo(pdb))
	resp, err := svc.GetCareerPage(r.Context())
	if err != nil {
		writeError(w, http.StatusInternalServerError, "career_error", err.Error())
		return
	}
	writeJSON(w, http.StatusOK, resp)
}

// GetTopMatches retourne les top/pires matchs du joueur.
func (h *CareerHandler) GetTopMatches(w http.ResponseWriter, r *http.Request) {
	pdb, err := h.resolvePlayer(r)
	if err != nil {
		writeError(w, http.StatusNotFound, "player_not_found", err.Error())
		return
	}
	svc := service.NewCareerService(duckdb.NewCareerRepo(pdb))
	resp, err := svc.GetTopMatches(r.Context())
	if err != nil {
		writeError(w, http.StatusInternalServerError, "top_matches_error", err.Error())
		return
	}
	writeJSON(w, http.StatusOK, resp)
}

// GetEncounters retourne les joueurs les plus fréquemment croisés.
func (h *CareerHandler) GetEncounters(w http.ResponseWriter, r *http.Request) {
	pdb, err := h.resolvePlayer(r)
	if err != nil {
		writeError(w, http.StatusNotFound, "player_not_found", err.Error())
		return
	}
	svc := service.NewCareerService(duckdb.NewCareerRepo(pdb))
	resp, err := svc.GetEncounters(r.Context())
	if err != nil {
		writeError(w, http.StatusInternalServerError, "encounters_error", err.Error())
		return
	}
	writeJSON(w, http.StatusOK, resp)
}

func (h *CareerHandler) resolvePlayer(r *http.Request) (*duckdb.PlayerDB, error) {
	slug := chi.URLParam(r, "player_slug")
	return config.ResolvePlayer(r.Context(), h.cfg, slug)
}
