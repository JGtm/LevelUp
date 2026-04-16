// Package handlers — CareerHandler : GET .../pages/career[/top-matches|/encounters].
package handlers

import (
	"context"
	"net/http"

	"github.com/go-chi/chi/v5"

	"levelup/go-api/internal/port"
)

// ServiceFactory est un type générique pour les factory de service player-scoped.
// Chaque handler reçoit une factory qui résout le slug → service injecté.
type ServiceFactory[S any] func(ctx context.Context, slug string) (S, error)

// ContextFactory est une factory qui retourne un service + XUID + Gamertag.
// Utilisé par les handlers qui ont besoin du contexte joueur en plus du service.
type ContextFactory[S any] func(ctx context.Context, slug string) (svc S, xuid, gamertag string, err error)

// CareerHandler gère les 3 endpoints de la page Carrière.
type CareerHandler struct {
	newSvc ServiceFactory[port.CareerService]
}

// NewCareerHandler crée un CareerHandler avec une factory de service injectée.
func NewCareerHandler(newSvc ServiceFactory[port.CareerService]) *CareerHandler {
	return &CareerHandler{newSvc: newSvc}
}

// GetCareer retourne la réponse complète de la page Carrière.
func (h *CareerHandler) GetCareer(w http.ResponseWriter, r *http.Request) {
	slug := chi.URLParam(r, "player_slug")
	svc, err := h.newSvc(r.Context(), slug)
	if err != nil {
		writeError(w, http.StatusNotFound, "player_not_found", err.Error())
		return
	}
	resp, err := svc.GetCareerPage(r.Context())
	if err != nil {
		writeError(w, http.StatusInternalServerError, "career_error", err.Error())
		return
	}
	writeJSON(w, http.StatusOK, resp)
}

// GetTopMatches retourne les top/pires matchs du joueur.
func (h *CareerHandler) GetTopMatches(w http.ResponseWriter, r *http.Request) {
	slug := chi.URLParam(r, "player_slug")
	svc, err := h.newSvc(r.Context(), slug)
	if err != nil {
		writeError(w, http.StatusNotFound, "player_not_found", err.Error())
		return
	}
	resp, err := svc.GetTopMatches(r.Context())
	if err != nil {
		writeError(w, http.StatusInternalServerError, "top_matches_error", err.Error())
		return
	}
	writeJSON(w, http.StatusOK, resp)
}

// GetEncounters retourne les joueurs les plus fréquemment croisés.
func (h *CareerHandler) GetEncounters(w http.ResponseWriter, r *http.Request) {
	slug := chi.URLParam(r, "player_slug")
	svc, err := h.newSvc(r.Context(), slug)
	if err != nil {
		writeError(w, http.StatusNotFound, "player_not_found", err.Error())
		return
	}
	resp, err := svc.GetEncounters(r.Context())
	if err != nil {
		writeError(w, http.StatusInternalServerError, "encounters_error", err.Error())
		return
	}
	writeJSON(w, http.StatusOK, resp)
}
