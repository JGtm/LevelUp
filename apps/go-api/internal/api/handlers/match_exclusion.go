// Package handlers — match_exclusion.go : gestion des matchs non pertinents.
//
// Endpoints :
//
//	PATCH /api/v1/players/{player_slug}/matches/{match_id}/exclusion
//	GET   /api/v1/players/{player_slug}/match-exclusions
package handlers

import (
	"encoding/json"
	"log/slog"
	"net/http"

	"github.com/go-chi/chi/v5"

	"levelup/go-api/internal/domain"
	"levelup/go-api/internal/port"
)

// MatchExclusionHandler gère les 2 endpoints d'exclusion de matchs.
type MatchExclusionHandler struct {
	newSvc ServiceFactory[port.MatchExclusionService]
}

// NewMatchExclusionHandler crée un MatchExclusionHandler.
func NewMatchExclusionHandler(newSvc ServiceFactory[port.MatchExclusionService]) *MatchExclusionHandler {
	return &MatchExclusionHandler{newSvc: newSvc}
}

// SetExclusion marque ou démarque un match comme non pertinent.
// PATCH /api/v1/players/{player_slug}/matches/{match_id}/exclusion
// Body: {"excluded": true}
// Réponse: 204 No Content
func (h *MatchExclusionHandler) SetExclusion(w http.ResponseWriter, r *http.Request) {
	slug := chi.URLParam(r, "player_slug")
	matchID := chi.URLParam(r, "match_id")

	svc, err := h.newSvc(r.Context(), slug)
	if err != nil {
		writeError(w, http.StatusNotFound, "player_not_found", err.Error())
		return
	}

	var req domain.SetMatchExclusionRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_body", err.Error())
		return
	}

	if err := svc.SetExclusion(r.Context(), matchID, req.Excluded); err != nil {
		slog.WarnContext(r.Context(), "match exclusion: db error",
			"match_id", matchID,
			"err", err,
		)
		writeError(w, http.StatusInternalServerError, "exclusion_error", err.Error())
		return
	}

	slog.InfoContext(r.Context(), "match exclusion updated",
		"player_slug", slug,
		"match_id", matchID,
		"excluded", req.Excluded,
	)
	w.WriteHeader(http.StatusNoContent)
}

// ListExclusions retourne les matchs marqués non pertinents.
// GET /api/v1/players/{player_slug}/match-exclusions
// Réponse: 200 [{"match_id": "...", "start_time": "...", ...}]
func (h *MatchExclusionHandler) ListExclusions(w http.ResponseWriter, r *http.Request) {
	slug := chi.URLParam(r, "player_slug")

	svc, err := h.newSvc(r.Context(), slug)
	if err != nil {
		writeError(w, http.StatusNotFound, "player_not_found", err.Error())
		return
	}

	matches, err := svc.ListExcluded(r.Context())
	if err != nil {
		slog.WarnContext(r.Context(), "match exclusion list: db error",
			"player_slug", slug,
			"err", err,
		)
		writeError(w, http.StatusInternalServerError, "list_exclusions_error", err.Error())
		return
	}

	if matches == nil {
		matches = []domain.ExcludedMatch{}
	}
	writeJSON(w, http.StatusOK, matches)
}
