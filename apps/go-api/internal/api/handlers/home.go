// Package handlers — home.go : handlers HTTP pour la page d'accueil Mission Control.
//
// Endpoints :
//
//	GET /api/v1/players/{player_slug}/pages/home     → HomePageResponse
//	GET /api/v1/players/{player_slug}/battlepass     → BattlePassResponse
//	GET /api/v1/players/{player_slug}/challenges     → ChallengesResponse
package handlers

import (
	"net/http"

	"github.com/go-chi/chi/v5"

	"levelup/go-api/internal/config"
	"levelup/go-api/internal/platform/duckdb"
	"levelup/go-api/internal/service"
)

// HomeHandler gère les endpoints de la page d'accueil Mission Control.
type HomeHandler struct {
	cfg *config.AppConfig
}

// NewHomeHandler crée un HomeHandler.
func NewHomeHandler(cfg *config.AppConfig) *HomeHandler {
	return &HomeHandler{cfg: cfg}
}

// GetHomePage retourne la page d'accueil agrégée.
// GET /api/v1/players/{player_slug}/pages/home
func (h *HomeHandler) GetHomePage(w http.ResponseWriter, r *http.Request) {
	pdb, err := h.resolvePlayer(r)
	if err != nil {
		writeError(w, http.StatusNotFound, "player_not_found", "joueur introuvable")
		return
	}

	repo := duckdb.NewHomeRepo(pdb)
	svc := service.NewHomeService(repo)

	page, err := svc.GetHomePage(r.Context(), pdb.Gamertag)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "home_page_error", "erreur chargement page d'accueil")
		return
	}

	writeJSON(w, http.StatusOK, page)
}

// GetBattlePass retourne les informations Battle Pass (best-effort).
// GET /api/v1/players/{player_slug}/battlepass
func (h *HomeHandler) GetBattlePass(w http.ResponseWriter, r *http.Request) {
	pdb, err := h.resolvePlayer(r)
	if err != nil {
		writeError(w, http.StatusNotFound, "player_not_found", "joueur introuvable")
		return
	}

	repo := duckdb.NewHomeRepo(pdb)
	svc := service.NewHomeService(repo)

	resp := svc.GetBattlePass(r.Context())
	writeJSON(w, http.StatusOK, resp)
}

// GetChallenges retourne les défis actifs (best-effort).
// GET /api/v1/players/{player_slug}/challenges
func (h *HomeHandler) GetChallenges(w http.ResponseWriter, r *http.Request) {
	pdb, err := h.resolvePlayer(r)
	if err != nil {
		writeError(w, http.StatusNotFound, "player_not_found", "joueur introuvable")
		return
	}

	repo := duckdb.NewHomeRepo(pdb)
	svc := service.NewHomeService(repo)

	resp := svc.GetChallenges(r.Context())
	writeJSON(w, http.StatusOK, resp)
}

// resolvePlayer traduit le slug URL en PlayerDB.
func (h *HomeHandler) resolvePlayer(r *http.Request) (*duckdb.PlayerDB, error) {
	slug := chi.URLParam(r, "player_slug")
	return config.ResolvePlayer(r.Context(), h.cfg, slug)
}
