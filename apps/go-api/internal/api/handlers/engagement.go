// Package handlers — engagement.go : endpoints API EngagementScore (Phase 4 plan).
//
// Routes :
//
//   - GET /api/v1/players/{slug}/matches/{match_id}/engagement
//     Retourne le score d'engagement + la courbe pour un match (Mock 10 Match View).
//
//   - GET /api/v1/players/{slug}/engagement_profile
//     Retourne les coefficients d'engagement par categorie de mode.
package handlers

import (
	"errors"
	"net/http"

	"github.com/go-chi/chi/v5"

	"levelup/go-api/internal/port"
	"levelup/go-api/internal/service"
)

// EngagementHandler gere les endpoints engagement.
type EngagementHandler struct {
	newSvc ServiceFactory[*service.PlayerEngagementService]
}

// NewEngagementHandler cree un EngagementHandler.
func NewEngagementHandler(newSvc ServiceFactory[*service.PlayerEngagementService]) *EngagementHandler {
	return &EngagementHandler{newSvc: newSvc}
}

// GetMatchEngagement : GET /matches/{match_id}/engagement
//
// Reponse JSON :
//
//	{
//	  "engagement_score": 62.0,
//	  "residual_brut": 1.45,
//	  "engagement_curve": [{"time_ms": ..., "pace_joueur": ..., "pace_team": ..., "pace_attendu": ..., "pace_lobby": ...}],
//	  "match_intensity": 14.2,
//	  "confidence": "full",
//	  "n_history_matches": 47
//	}
//
// Codes d'erreur :
//   - 404 player_not_found / match_not_found
//   - 400 invalid_request (matchID manquant)
//   - 422 pve_not_supported (match PvE - non couvert v1)
//   - 503 engagement_unavailable (migration Phase 2 non appliquee)
//   - 500 engagement_error (autre)
func (h *EngagementHandler) GetMatchEngagement(w http.ResponseWriter, r *http.Request) {
	slug := chi.URLParam(r, "player_slug")
	svc, err := h.newSvc(r.Context(), slug)
	if err != nil {
		writeError(w, http.StatusNotFound, "player_not_found", err.Error())
		return
	}

	matchID := chi.URLParam(r, "match_id")
	if matchID == "" {
		writeError(w, http.StatusBadRequest, "invalid_request", "match_id est requis")
		return
	}

	result, err := svc.GetMatchEngagement(r.Context(), matchID)
	if err != nil {
		switch {
		case errors.Is(err, service.ErrEngagementMatchNotFound):
			writeError(w, http.StatusNotFound, "match_not_found", "match introuvable pour ce joueur : "+matchID)
		case errors.Is(err, service.ErrEngagementPvENotSupported):
			writeError(w, http.StatusUnprocessableEntity, "pve_not_supported", "engagement non couvert pour les matchs PvE en v1")
		case errors.Is(err, port.ErrEngagementUnavailable):
			writeError(w, http.StatusServiceUnavailable, "engagement_unavailable", "migration EngagementScore non appliquee")
		default:
			writeError(w, http.StatusInternalServerError, "engagement_error", err.Error())
		}
		return
	}

	writeJSON(w, http.StatusOK, result)
}

// GetEngagementProfile : GET /engagement_profile
//
// Reponse JSON : tableau de coefficients par categorie de mode.
//
//	[
//	  {"xuid": "...", "mode_category": "PvP_ranked", "coef_team_share": 1.12, "coef_lobby_share": 1.05, "n_matches": 187, "last_updated": "..."},
//	  ...
//	]
//
// Reponse vide si aucun coefficient n'a encore ete calcule (cold start).
func (h *EngagementHandler) GetEngagementProfile(w http.ResponseWriter, r *http.Request) {
	slug := chi.URLParam(r, "player_slug")
	svc, err := h.newSvc(r.Context(), slug)
	if err != nil {
		writeError(w, http.StatusNotFound, "player_not_found", err.Error())
		return
	}

	coefs, err := svc.GetEngagementProfile(r.Context())
	if err != nil {
		writeError(w, http.StatusInternalServerError, "engagement_error", err.Error())
		return
	}
	writeJSON(w, http.StatusOK, coefs)
}
