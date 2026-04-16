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

	"levelup/go-api/internal/port"
)

// HomeHandler gère les endpoints de la page d'accueil Mission Control.
type HomeHandler struct {
	newSvc ContextFactory[port.HomeService]
}

// NewHomeHandler crée un HomeHandler.
func NewHomeHandler(newSvc ContextFactory[port.HomeService]) *HomeHandler {
	return &HomeHandler{newSvc: newSvc}
}

// GetHomePage retourne la page d'accueil agrégée.
// GET /api/v1/players/{player_slug}/pages/home
func (h *HomeHandler) GetHomePage(w http.ResponseWriter, r *http.Request) {
	slug := chi.URLParam(r, "player_slug")
	svc, _, gamertag, err := h.newSvc(r.Context(), slug)
	if err != nil {
		writeError(w, http.StatusNotFound, "player_not_found", "joueur introuvable")
		return
	}

	page, err := svc.GetHomePage(r.Context(), gamertag)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "home_page_error", "erreur chargement page d'accueil")
		return
	}

	writeJSON(w, http.StatusOK, page)
}

// GetBattlePass retourne les informations Battle Pass (best-effort).
// GET /api/v1/players/{player_slug}/battlepass
func (h *HomeHandler) GetBattlePass(w http.ResponseWriter, r *http.Request) {
	slug := chi.URLParam(r, "player_slug")
	svc, _, _, err := h.newSvc(r.Context(), slug)
	if err != nil {
		writeError(w, http.StatusNotFound, "player_not_found", "joueur introuvable")
		return
	}

	resp := svc.GetBattlePass(r.Context())
	writeJSON(w, http.StatusOK, resp)
}

// GetChallenges retourne les défis actifs (best-effort).
// GET /api/v1/players/{player_slug}/challenges
func (h *HomeHandler) GetChallenges(w http.ResponseWriter, r *http.Request) {
	slug := chi.URLParam(r, "player_slug")
	svc, _, _, err := h.newSvc(r.Context(), slug)
	if err != nil {
		writeError(w, http.StatusNotFound, "player_not_found", "joueur introuvable")
		return
	}

	resp := svc.GetChallenges(r.Context())
	writeJSON(w, http.StatusOK, resp)
}
