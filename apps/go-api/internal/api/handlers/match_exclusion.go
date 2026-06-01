// Package handlers — match_exclusion.go : gestion des matchs non pertinents.
//
// Endpoint :
//
//	PATCH /api/v1/players/{player_slug}/matches/{match_id}/exclusion
//
// NOTE : GET /match-exclusions a été supprimé en revue 2026-04-29 P0.2 Q6
// (orphelin côté front). À réintroduire si une vue admin de listing devient
// nécessaire.
package handlers

import (
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"

	"github.com/go-chi/chi/v5"

	"levelup/go-api/internal/domain"
	"levelup/go-api/internal/platform/dblease"
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
		writeError(r.Context(), w, http.StatusNotFound, "player_not_found", err.Error())
		return
	}

	var req domain.SetMatchExclusionRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(r.Context(), w, http.StatusBadRequest, "invalid_body", err.Error())
		return
	}

	if err := svc.SetExclusion(r.Context(), matchID, req.Excluded); err != nil {
		switch {
		case errors.Is(err, domain.ErrMatchNotFound):
			writeError(r.Context(), w, http.StatusNotFound, "match_not_found",
				"Match introuvable dans le registre")
			return
		case errors.Is(err, domain.ErrRankedMatchNotExcludable):
			writeError(r.Context(), w, http.StatusUnprocessableEntity, "ranked_not_excludable",
				"Les matchs classés ne peuvent pas être exclus")
			return
		case errors.Is(err, dblease.ErrDBLocked):
			w.Header().Set("Retry-After", "5")
			writeError(r.Context(), w, http.StatusServiceUnavailable, "db_busy",
				"database is currently busy, please retry")
			return
		}
		slog.WarnContext(r.Context(), "match exclusion: db error",
			"match_id", matchID,
			"err", err,
		)
		writeError(r.Context(), w, http.StatusInternalServerError, "exclusion_error", err.Error())
		return
	}

	slog.InfoContext(r.Context(), "match exclusion updated",
		"player_slug", slug,
		"match_id", matchID,
		"excluded", req.Excluded,
	)
	w.WriteHeader(http.StatusNoContent)
}
