// Package handlers — session_page.go : handler HTTP pour la page détail de session.
package handlers

import (
	"encoding/json"
	"log/slog"
	"net/http"

	"github.com/go-chi/chi/v5"

	"levelup/go-api/internal/domain"
	"levelup/go-api/internal/port"
)

// SessionPageHandler gère l'endpoint POST /pages/sessions/detail.
type SessionPageHandler struct {
	newSvc ServiceFactory[port.SessionPageService]
}

// NewSessionPageHandler crée un SessionPageHandler.
func NewSessionPageHandler(newSvc ServiceFactory[port.SessionPageService]) *SessionPageHandler {
	return &SessionPageHandler{newSvc: newSvc}
}

// GetPage retourne la page de détail d'une session.
func (h *SessionPageHandler) GetPage(w http.ResponseWriter, r *http.Request) {
	slug := chi.URLParam(r, "player_slug")
	svc, err := h.newSvc(r.Context(), slug)
	if err != nil {
		slog.WarnContext(r.Context(), "session page: joueur introuvable", "player_slug", slug, "err", err)
		writeError(w, http.StatusNotFound, "player_not_found", err.Error())
		return
	}

	var req domain.SessionPageRequest
	if r.ContentLength > 0 {
		if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, 1<<20)).Decode(&req); err != nil {
			slog.WarnContext(r.Context(), "session page: corps invalide", "player_slug", slug, "err", err)
			writeError(w, http.StatusBadRequest, "invalid_json", "corps JSON invalide")
			return
		}
	}
	if err := req.Validate(); err != nil {
		slog.WarnContext(r.Context(), "session page: requête invalide", "player_slug", slug, "err", err)
		writeError(w, http.StatusBadRequest, "invalid_body", err.Error())
		return
	}

	slog.DebugContext(r.Context(), "session page: calcul",
		"player_slug", slug,
		"session_label", derefReqString(req.SessionLabel),
		"compare_session_label", derefReqString(req.CompareSessionLabel),
		"enable_compare", req.EnableCompare,
	)

	page, err := svc.GetPage(r.Context(), req)
	if err != nil {
		slog.ErrorContext(r.Context(), "session page: erreur service", "player_slug", slug, "err", err)
		writeError(w, http.StatusInternalServerError, "session_page_error", err.Error())
		return
	}

	slog.InfoContext(r.Context(), "session page: générée",
		"player_slug", slug,
		"session_label", derefReqString(req.SessionLabel),
		"resolved_session", derefSessionLabel(page.CurrentSession),
		"available_sessions", len(page.AvailableSessions),
		"match_count", len(page.Matches),
		"compare_enabled", page.CompareEnabled,
		"compare_session", derefSessionLabel(page.CompareSession),
		"compare_match_count", len(page.CompareMatches),
		"previous_session_label", derefReqString(page.PreviousSessionLabel),
		"next_session_label", derefReqString(page.NextSessionLabel),
	)
	writeJSON(w, http.StatusOK, page)
}

func derefReqString(value *string) string {
	if value == nil {
		return ""
	}
	return *value
}

func derefSessionLabel(entry *domain.SessionCompareEntry) string {
	if entry == nil {
		return ""
	}
	return entry.SessionLabel
}
