// Package handlers — match_favorite.go : handler HTTP pour les favoris de matchs.
//
// Endpoint :
//
//	PATCH /api/v1/players/{player_slug}/matches/{match_id}/favorite → MatchFavoriteResponse
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

// MatchFavoriteHandler gère les bascules de favoris de matchs.
type MatchFavoriteHandler struct {
	newSvc ServiceFactory[port.SocialService]
}

// NewMatchFavoriteHandler crée un MatchFavoriteHandler.
func NewMatchFavoriteHandler(newSvc ServiceFactory[port.SocialService]) *MatchFavoriteHandler {
	return &MatchFavoriteHandler{newSvc: newSvc}
}

// PatchMatchFavorite bascule l'état favori d'un match pour un joueur.
// PATCH /api/v1/players/{player_slug}/matches/{match_id}/favorite
func (h *MatchFavoriteHandler) PatchMatchFavorite(w http.ResponseWriter, r *http.Request) {
	slug := chi.URLParam(r, "player_slug")
	matchID := chi.URLParam(r, "match_id")

	if matchID == "" {
		writeError(r.Context(), w, http.StatusBadRequest, "missing_match_id", "match_id requis")
		return
	}

	svc, err := h.newSvc(r.Context(), slug)
	if err != nil {
		writeError(r.Context(), w, http.StatusNotFound, "player_not_found", err.Error())
		return
	}

	var req domain.MatchFavoriteRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(r.Context(), w, http.StatusBadRequest, "invalid_body", "corps JSON invalide")
		return
	}
	req.PlayerSlug = slug
	req.MatchID = matchID

	if err := svc.ToggleMatchFavorite(r.Context(), req); err != nil {
		if errors.Is(err, dblease.ErrDBLocked) {
			w.Header().Set("Retry-After", "5")
			writeError(r.Context(), w, http.StatusServiceUnavailable, "db_busy",
				"database is currently busy, please retry")
			return
		}
		slog.ErrorContext(r.Context(), "match_favorite: erreur bascule",
			"err", err, "match_id", matchID, "player", slug)
		writeError(r.Context(), w, http.StatusInternalServerError, "favorite_error", err.Error())
		return
	}

	slog.DebugContext(r.Context(), "match_favorite: bascule ok",
		"match_id", matchID, "player", slug, "favorited", req.Favorited)

	writeJSON(w, http.StatusOK, domain.MatchFavoriteResponse(req))
}
