// Package handlers — session.go : handler HTTP pour la gestion du contexte session.
//
// Endpoints :
//   POST /session/context  → SessionContextResponse
package handlers

import (
	"encoding/json"
	"net/http"

	"levelup/go-api/internal/api/middleware"
	"levelup/go-api/internal/domain"
	"levelup/go-api/internal/platform/session"
)

// SessionHandler gère les endpoints de session.
type SessionHandler struct {
	store *session.Store
}

// NewSessionHandler crée un SessionHandler.
func NewSessionHandler(store *session.Store) *SessionHandler {
	return &SessionHandler{store: store}
}

// PostContext met à jour le contexte de session (player_slug, locale).
// POST /session/context
func (h *SessionHandler) PostContext(w http.ResponseWriter, r *http.Request) {
	sess := middleware.GetSession(r.Context())
	if sess == nil {
		writeError(w, http.StatusInternalServerError, "no_session", "session non initialisée")
		return
	}

	var req domain.SessionContextRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_body", "corps JSON invalide")
		return
	}

	if req.PlayerSlug != nil {
		sess.CurrentPlayerSlug = req.PlayerSlug
	}
	if req.Locale != nil {
		sess.Locale = *req.Locale
	}

	if err := h.store.Save(sess); err != nil {
		writeError(w, http.StatusInternalServerError, "session_save_error", err.Error())
		return
	}

	writeJSON(w, http.StatusOK, domain.SessionContextResponse{
		CurrentPlayerSlug: sess.CurrentPlayerSlug,
		Locale:            sess.Locale,
		HintsVisible:      sess.HintsVisible,
		AuthReady:         sess.AuthReady,
	})
}
