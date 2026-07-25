// Package handlers — notifications.go : endpoints du système de notifications in-app.
//
// Tous les endpoints sont préfixés /api/v1/players/{player_slug}/notifications.
// MIGRÉ vers Huma (Phase 3b) : Mount crée humacore.NewAPI(r) sur le sous-routeur
// (hérite ownership/title + lit {player_slug} parent) et enregistre via huma.*.
// Logique métier inchangée (service notifications), seul le wrapping HTTP change.
package handlers

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"strconv"

	"github.com/danielgtaylor/huma/v2"
	"github.com/go-chi/chi/v5"

	"levelup/go-api/internal/api/humacore"
	"levelup/go-api/internal/notifications"
	"levelup/go-api/internal/platform/dblease"
)

// notifWriteErr centralise le mapping d'erreurs des handlers write : ErrDBLocked
// → 503 + Retry-After:5 (via huma.ErrorWithHeaders), ErrNotFound → 404, autres
// → 500. Contrat identique à l'ancien writeNotifWriteErr.
func notifWriteErr(ctx context.Context, op string, err error) error {
	switch {
	case errors.Is(err, dblease.ErrDBLocked):
		return huma.ErrorWithHeaders(
			humacore.NewError(http.StatusServiceUnavailable, "db_busy", "database is currently busy, please retry"),
			http.Header{"Retry-After": []string{"5"}},
		)
	case errors.Is(err, notifications.ErrNotFound):
		return humacore.NewError(http.StatusNotFound, "not_found", "notification introuvable")
	default:
		slog.WarnContext(ctx, "notifications: "+op, "err", err)
		return humacore.NewError(http.StatusInternalServerError, op+"_error", err.Error())
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

// Mount enregistre tous les endpoints via Huma sur le sous-routeur chi
// (préfixe /players/{player_slug} + middleware ownership/title hérités).
func (h *NotificationsHandler) Mount(r chi.Router, opts ...humacore.MountOption) {
	api := humacore.NewAPI(r, opts...)
	huma.Get(api, "/notifications", h.handleList, humacore.Op("listNotifications", "Liste des notifications du joueur", "notifications"))
	huma.Get(api, "/notifications/unread-count", h.handleUnreadCount, humacore.Op("getNotificationUnreadCount", "Compteur de notifications non lues", "notifications"))
	huma.Post(api, "/notifications/mark-read", h.handleMarkRead, humacore.Op("postMarkRead", "Marque une liste de notifications comme lues", "notifications"))
	huma.Post(api, "/notifications/mark-all-read", h.handleMarkAllRead, humacore.Op("postMarkAllRead", "Marque toutes les notifications comme lues", "notifications"))
	huma.Patch(api, "/notifications/{id}/unread", h.handleMarkUnread, humacore.Op("patchNotificationUnread", "Repasse une notification en non-lue", "notifications"))
	huma.Delete(api, "/notifications/{id}", h.handleDelete, humacore.Op("deleteNotification", "Supprime une notification", "notifications"))
	huma.Get(api, "/notifications/preferences", h.handleGetPreferences, humacore.Op("getNotificationPreferences", "Préférences notifications du joueur", "notifications"))
	huma.Patch(api, "/notifications/preferences", h.handleUpdatePreferences, humacore.Op("patchNotificationPreferences", "Met à jour les préférences", "notifications"))
	huma.Post(api, "/notifications/test", h.handlePostTest, humacore.Op("postNotificationTest", "Émet une notification de test (debug)", "notifications"))
}

// ─── Inputs/Outputs Huma ─────────────────────────────────────────────────────

type notifPlayerInput struct {
	PlayerSlug string `path:"player_slug"`
}

// notifIDInput : {id} pris en STRING (pas int64) pour reproduire le contrat
// d'erreur d'origine — un id non numérique renvoie 400 invalid_id (parse maison),
// PAS le 422 de validation Huma qu'un `int64` produirait.
type notifIDInput struct {
	PlayerSlug string `path:"player_slug"`
	ID         string `path:"id"`
}

// parseNotifID reproduit l'ancien parseIDParam : 400 invalid_id si non numérique
// ou <= 0.
func parseNotifID(raw string) (int64, error) {
	id, err := strconv.ParseInt(raw, 10, 64)
	if err != nil {
		return 0, humacore.NewError(http.StatusBadRequest, "invalid_id", err.Error())
	}
	if id <= 0 {
		return 0, humacore.NewError(http.StatusBadRequest, "invalid_id", "id must be positive")
	}
	return id, nil
}

type notifListInput struct {
	PlayerSlug string `path:"player_slug"`
	UnreadOnly bool   `query:"unread_only"`
	Category   string `query:"category"`
	Limit      int    `query:"limit"`
	BeforeID   int64  `query:"before_id"`
}

type notifMarkReadInput struct {
	PlayerSlug string `path:"player_slug"`
	Body       struct {
		IDs []int64 `json:"ids"`
	}
}

// notifMarkAllReadInput : body OPTIONNEL (pointeur) — {category}? facultatif.
type notifMarkAllReadInput struct {
	PlayerSlug string `path:"player_slug"`
	Body       *struct {
		Category string `json:"category"`
	}
}

type notifUpdatePrefsInput struct {
	PlayerSlug string `path:"player_slug"`
	Body       struct {
		Items []notifications.Preference `json:"items"`
	}
}

type notifListOutput struct{ Body notifications.ListResult }
type notifUnreadCountOutput struct{ Body notifications.UnreadCount }
type notifMarkResultOutput struct{ Body notifications.MarkResult }
type notifPrefsOutput struct {
	Body struct {
		Items []notifications.Preference `json:"items"`
	}
}

// notifNoContent : réponse 204 sans corps (Status override la valeur par défaut).
type notifNoContent struct {
	Status int
}

// ─── Endpoints ───────────────────────────────────────────────────────────────

// handleList : GET /notifications?unread_only=&category=&limit=&before_id=
func (h *NotificationsHandler) handleList(ctx context.Context, in *notifListInput) (*notifListOutput, error) {
	svc, err := h.resolve(ctx, in.PlayerSlug)
	if err != nil {
		return nil, err
	}
	f := notifications.ListFilter{
		UnreadOnly: in.UnreadOnly,
		Category:   notifications.Category(in.Category),
		Limit:      in.Limit,
		BeforeID:   in.BeforeID,
	}
	res, err := svc.List(ctx, f)
	if err != nil {
		slog.WarnContext(ctx, "notifications: list", "err", err)
		return nil, humacore.NewError(http.StatusInternalServerError, "list_error", err.Error())
	}
	if res.Items == nil {
		res.Items = []notifications.Notification{}
	}
	return &notifListOutput{Body: res}, nil
}

// handleUnreadCount : GET /notifications/unread-count
func (h *NotificationsHandler) handleUnreadCount(ctx context.Context, in *notifPlayerInput) (*notifUnreadCountOutput, error) {
	svc, err := h.resolve(ctx, in.PlayerSlug)
	if err != nil {
		return nil, err
	}
	c, err := svc.UnreadCount(ctx)
	if err != nil {
		slog.WarnContext(ctx, "notifications: unread-count", "err", err)
		return nil, humacore.NewError(http.StatusInternalServerError, "unread_count_error", err.Error())
	}
	if c.ByCategory == nil {
		c.ByCategory = map[string]int{}
	}
	return &notifUnreadCountOutput{Body: c}, nil
}

// handleMarkRead : POST /notifications/mark-read  body: {"ids": [int64...]}
func (h *NotificationsHandler) handleMarkRead(ctx context.Context, in *notifMarkReadInput) (*notifMarkResultOutput, error) {
	svc, err := h.resolve(ctx, in.PlayerSlug)
	if err != nil {
		return nil, err
	}
	res, err := svc.MarkRead(ctx, in.Body.IDs)
	if err != nil {
		return nil, notifWriteErr(ctx, "mark_read", err)
	}
	return &notifMarkResultOutput{Body: res}, nil
}

// handleMarkAllRead : POST /notifications/mark-all-read  body: {"category"?: "..."}
func (h *NotificationsHandler) handleMarkAllRead(ctx context.Context, in *notifMarkAllReadInput) (*notifMarkResultOutput, error) {
	svc, err := h.resolve(ctx, in.PlayerSlug)
	if err != nil {
		return nil, err
	}
	category := ""
	if in.Body != nil {
		category = in.Body.Category
	}
	res, err := svc.MarkAllRead(ctx, notifications.Category(category))
	if err != nil {
		return nil, notifWriteErr(ctx, "mark_all_read", err)
	}
	return &notifMarkResultOutput{Body: res}, nil
}

// handleMarkUnread : PATCH /notifications/{id}/unread → 204
func (h *NotificationsHandler) handleMarkUnread(ctx context.Context, in *notifIDInput) (*notifNoContent, error) {
	svc, err := h.resolve(ctx, in.PlayerSlug)
	if err != nil {
		return nil, err
	}
	id, err := parseNotifID(in.ID)
	if err != nil {
		return nil, err
	}
	if err := svc.MarkUnread(ctx, id); err != nil {
		return nil, notifWriteErr(ctx, "mark_unread", err)
	}
	return &notifNoContent{Status: http.StatusNoContent}, nil
}

// handleDelete : DELETE /notifications/{id} → 204
func (h *NotificationsHandler) handleDelete(ctx context.Context, in *notifIDInput) (*notifNoContent, error) {
	svc, err := h.resolve(ctx, in.PlayerSlug)
	if err != nil {
		return nil, err
	}
	id, err := parseNotifID(in.ID)
	if err != nil {
		return nil, err
	}
	if err := svc.Delete(ctx, id); err != nil {
		return nil, notifWriteErr(ctx, "delete", err)
	}
	return &notifNoContent{Status: http.StatusNoContent}, nil
}

// handleGetPreferences : GET /notifications/preferences
func (h *NotificationsHandler) handleGetPreferences(ctx context.Context, in *notifPlayerInput) (*notifPrefsOutput, error) {
	svc, err := h.resolve(ctx, in.PlayerSlug)
	if err != nil {
		return nil, err
	}
	prefs, err := svc.GetPreferences(ctx)
	if err != nil {
		slog.WarnContext(ctx, "notifications: get-preferences", "err", err)
		return nil, humacore.NewError(http.StatusInternalServerError, "prefs_error", err.Error())
	}
	if prefs == nil {
		prefs = []notifications.Preference{}
	}
	out := &notifPrefsOutput{}
	out.Body.Items = prefs
	return out, nil
}

// handleUpdatePreferences : PATCH /notifications/preferences  body: {"items":[...]}
func (h *NotificationsHandler) handleUpdatePreferences(ctx context.Context, in *notifUpdatePrefsInput) (*notifPrefsOutput, error) {
	svc, err := h.resolve(ctx, in.PlayerSlug)
	if err != nil {
		return nil, err
	}
	updated, err := svc.UpdatePreferences(ctx, in.Body.Items)
	if err != nil {
		return nil, notifWriteErr(ctx, "prefs_update", err)
	}
	if updated == nil {
		updated = []notifications.Preference{}
	}
	out := &notifPrefsOutput{}
	out.Body.Items = updated
	return out, nil
}

// handlePostTest : POST /notifications/test → 204
// Émet une notification de test (catégorie app_release, severity info).
func (h *NotificationsHandler) handlePostTest(ctx context.Context, in *notifPlayerInput) (*notifNoContent, error) {
	svc, err := h.resolve(ctx, in.PlayerSlug)
	if err != nil {
		return nil, err
	}
	err = svc.Emit(ctx, notifications.EmitInput{
		Category:    notifications.CategoryAppRelease,
		Severity:    notifications.SeverityInfo,
		TitleKey:    "notif.test.title",
		BodyKey:     "notif.test.body",
		Params:      map[string]any{"slug": in.PlayerSlug},
		TargetRoute: "/help",
		Source:      "test_button",
	})
	if err != nil {
		slog.WarnContext(ctx, "notifications: test-emit", "slug", in.PlayerSlug, "err", err)
		return nil, humacore.NewError(http.StatusInternalServerError, "test_emit_error", err.Error())
	}
	slog.InfoContext(ctx, "notifications: test sent", "slug", in.PlayerSlug)
	return &notifNoContent{Status: http.StatusNoContent}, nil
}

// resolve récupère le service pour le slug courant ou renvoie une erreur Huma 404.
func (h *NotificationsHandler) resolve(ctx context.Context, slug string) (*notifications.Service, error) {
	svc, err := h.newSvc(ctx, slug)
	if err != nil {
		return nil, humacore.NewError(http.StatusNotFound, "player_not_found", err.Error())
	}
	return svc, nil
}

func atoi(s string) int {
	if s == "" {
		return 0
	}
	n, _ := strconv.Atoi(s)
	return n
}
