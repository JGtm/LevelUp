// Package handlers — notifications.go : endpoints du système de notifications in-app.
//
// Tous les endpoints sont préfixés /api/v1/players/{player_slug}/notifications.
// Voir plan §1.3 (j-ai-chang-d-avis-et-dapper-conway.md).
package handlers

import (
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"

	"levelup/go-api/internal/notifications"
	"levelup/go-api/internal/platform/dblease"
)

// writeNotifWriteErr centralise le mapping d'erreurs des handlers notifications
// write : ErrDBLocked → 503 + Retry-After, ErrNotFound → 404, autres → 500.
// Évite la duplication du switch dans MarkRead / MarkUnread / Delete / etc.
func writeNotifWriteErr(w http.ResponseWriter, ctx context.Context, op string, err error) {
	switch {
	case errors.Is(err, dblease.ErrDBLocked):
		w.Header().Set("Retry-After", "5")
		writeError(w, http.StatusServiceUnavailable, "db_busy",
			"database is currently busy, please retry")
	case errors.Is(err, notifications.ErrNotFound):
		writeError(w, http.StatusNotFound, "not_found", "notification introuvable")
	default:
		slog.WarnContext(ctx, "notifications: "+op, "err", err)
		writeError(w, http.StatusInternalServerError, op+"_error", err.Error())
	}
}

// NotificationsServiceFactory construit un *notifications.Service à partir
// du player_slug courant (résolution PlayerDB déléguée à la registry).
type NotificationsServiceFactory func(ctx context.Context, slug string) (*notifications.Service, error)

// NotificationsHandler gère les endpoints /notifications.
type NotificationsHandler struct {
	newSvc NotificationsServiceFactory
}

// NewNotificationsHandler crée un NotificationsHandler.
func NewNotificationsHandler(newSvc NotificationsServiceFactory) *NotificationsHandler {
	return &NotificationsHandler{newSvc: newSvc}
}

// Mount enregistre tous les endpoints sur un router chi sous-monté.
// Appelé depuis server.go dans la subroute /players/{player_slug}.
func (h *NotificationsHandler) Mount(r chi.Router) {
	r.Get("/notifications", h.List)
	r.Get("/notifications/unread-count", h.UnreadCount)
	r.Post("/notifications/mark-read", h.MarkRead)
	r.Post("/notifications/mark-all-read", h.MarkAllRead)
	r.Patch("/notifications/{id}/unread", h.MarkUnread)
	r.Delete("/notifications/{id}", h.Delete)
	r.Get("/notifications/preferences", h.GetPreferences)
	r.Patch("/notifications/preferences", h.UpdatePreferences)
	r.Post("/notifications/test", h.PostTest)
}

// List : GET /notifications?unread_only=&category=&limit=&before_id=
func (h *NotificationsHandler) List(w http.ResponseWriter, r *http.Request) {
	svc, ok := h.resolve(w, r)
	if !ok {
		return
	}
	q := r.URL.Query()
	f := notifications.ListFilter{
		UnreadOnly: q.Get("unread_only") == "true",
		Category:   notifications.Category(q.Get("category")),
		Limit:      atoi(q.Get("limit")),
		BeforeID:   atoi64(q.Get("before_id")),
	}
	res, err := svc.List(r.Context(), f)
	if err != nil {
		slog.WarnContext(r.Context(), "notifications: list", "err", err)
		writeError(w, http.StatusInternalServerError, "list_error", err.Error())
		return
	}
	if res.Items == nil {
		res.Items = []notifications.Notification{}
	}
	writeJSON(w, http.StatusOK, res)
}

// UnreadCount : GET /notifications/unread-count
func (h *NotificationsHandler) UnreadCount(w http.ResponseWriter, r *http.Request) {
	svc, ok := h.resolve(w, r)
	if !ok {
		return
	}
	c, err := svc.UnreadCount(r.Context())
	if err != nil {
		slog.WarnContext(r.Context(), "notifications: unread-count", "err", err)
		writeError(w, http.StatusInternalServerError, "unread_count_error", err.Error())
		return
	}
	if c.ByCategory == nil {
		c.ByCategory = map[string]int{}
	}
	writeJSON(w, http.StatusOK, c)
}

// MarkRead : POST /notifications/mark-read  body: {"ids": [int64...]}
func (h *NotificationsHandler) MarkRead(w http.ResponseWriter, r *http.Request) {
	svc, ok := h.resolve(w, r)
	if !ok {
		return
	}
	var req struct {
		IDs []int64 `json:"ids"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_body", err.Error())
		return
	}
	res, err := svc.MarkRead(r.Context(), req.IDs)
	if err != nil {
		writeNotifWriteErr(w, r.Context(), "mark_read", err)
		return
	}
	writeJSON(w, http.StatusOK, res)
}

// MarkAllRead : POST /notifications/mark-all-read  body: {"category"?: "..."}
func (h *NotificationsHandler) MarkAllRead(w http.ResponseWriter, r *http.Request) {
	svc, ok := h.resolve(w, r)
	if !ok {
		return
	}
	var req struct {
		Category string `json:"category"`
	}
	// Body optionnel
	if r.ContentLength > 0 {
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeError(w, http.StatusBadRequest, "invalid_body", err.Error())
			return
		}
	}
	res, err := svc.MarkAllRead(r.Context(), notifications.Category(req.Category))
	if err != nil {
		writeNotifWriteErr(w, r.Context(), "mark_all_read", err)
		return
	}
	writeJSON(w, http.StatusOK, res)
}

// MarkUnread : PATCH /notifications/{id}/unread
func (h *NotificationsHandler) MarkUnread(w http.ResponseWriter, r *http.Request) {
	svc, ok := h.resolve(w, r)
	if !ok {
		return
	}
	id, err := parseIDParam(r)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid_id", err.Error())
		return
	}
	if err := svc.MarkUnread(r.Context(), id); err != nil {
		writeNotifWriteErr(w, r.Context(), "mark_unread", err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// Delete : DELETE /notifications/{id}
func (h *NotificationsHandler) Delete(w http.ResponseWriter, r *http.Request) {
	svc, ok := h.resolve(w, r)
	if !ok {
		return
	}
	id, err := parseIDParam(r)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid_id", err.Error())
		return
	}
	if err := svc.Delete(r.Context(), id); err != nil {
		writeNotifWriteErr(w, r.Context(), "delete", err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// GetPreferences : GET /notifications/preferences
func (h *NotificationsHandler) GetPreferences(w http.ResponseWriter, r *http.Request) {
	svc, ok := h.resolve(w, r)
	if !ok {
		return
	}
	prefs, err := svc.GetPreferences(r.Context())
	if err != nil {
		slog.WarnContext(r.Context(), "notifications: get-preferences", "err", err)
		writeError(w, http.StatusInternalServerError, "prefs_error", err.Error())
		return
	}
	if prefs == nil {
		prefs = []notifications.Preference{}
	}
	writeJSON(w, http.StatusOK, map[string]any{"items": prefs})
}

// UpdatePreferences : PATCH /notifications/preferences  body: {"items":[{category,enabled,delivery}]}
func (h *NotificationsHandler) UpdatePreferences(w http.ResponseWriter, r *http.Request) {
	svc, ok := h.resolve(w, r)
	if !ok {
		return
	}
	var req struct {
		Items []notifications.Preference `json:"items"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_body", err.Error())
		return
	}
	updated, err := svc.UpdatePreferences(r.Context(), req.Items)
	if err != nil {
		writeNotifWriteErr(w, r.Context(), "prefs_update", err)
		return
	}
	if updated == nil {
		updated = []notifications.Preference{}
	}
	writeJSON(w, http.StatusOK, map[string]any{"items": updated})
}

// PostTest : POST /notifications/test
// Émet une notification de test (catégorie app_release, severity info) pour le
// joueur courant. Utile pour valider le pipeline UI (toast + dropdown + a11y)
// depuis le bouton "Envoyer une notification de test" du Settings tab.
func (h *NotificationsHandler) PostTest(w http.ResponseWriter, r *http.Request) {
	svc, ok := h.resolve(w, r)
	if !ok {
		return
	}
	slug := chi.URLParam(r, "player_slug")
	err := svc.Emit(r.Context(), notifications.EmitInput{
		Category:    notifications.CategoryAppRelease,
		Severity:    notifications.SeverityInfo,
		TitleKey:    "notif.test.title",
		BodyKey:     "notif.test.body",
		Params:      map[string]any{"slug": slug},
		TargetRoute: "/help",
		Source:      "test_button",
	})
	if err != nil {
		slog.WarnContext(r.Context(), "notifications: test-emit", "slug", slug, "err", err)
		writeError(w, http.StatusInternalServerError, "test_emit_error", err.Error())
		return
	}
	slog.InfoContext(r.Context(), "notifications: test sent", "slug", slug)
	w.WriteHeader(http.StatusNoContent)
}

// resolve récupère le service pour le slug courant ou écrit 404.
func (h *NotificationsHandler) resolve(w http.ResponseWriter, r *http.Request) (*notifications.Service, bool) {
	slug := chi.URLParam(r, "player_slug")
	svc, err := h.newSvc(r.Context(), slug)
	if err != nil {
		writeError(w, http.StatusNotFound, "player_not_found", err.Error())
		return nil, false
	}
	return svc, true
}

func parseIDParam(r *http.Request) (int64, error) {
	idStr := chi.URLParam(r, "id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		return 0, err
	}
	if id <= 0 {
		return 0, errors.New("id must be positive")
	}
	return id, nil
}

func atoi(s string) int {
	if s == "" {
		return 0
	}
	n, _ := strconv.Atoi(s)
	return n
}

func atoi64(s string) int64 {
	if s == "" {
		return 0
	}
	n, _ := strconv.ParseInt(s, 10, 64)
	return n
}
