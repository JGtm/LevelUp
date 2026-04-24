// Package handlers — home.go : handlers HTTP pour la page d'accueil Mission Control.
//
// Endpoints :
//
//	GET /api/v1/players/{player_slug}/pages/home     → HomePageResponse
//	GET /api/v1/players/{player_slug}/battlepass     → BattlePassResponse
//	GET /api/v1/players/{player_slug}/challenges     → ChallengesResponse
package handlers

import (
	"context"
	"log/slog"
	"net/http"
	"strings"

	"github.com/go-chi/chi/v5"

	settings_platform "levelup/go-api/internal/platform/settings"
	"levelup/go-api/internal/port"
)

// HomeAuthFactory est une factory qui retourne un HomeService + contexte enrichi avec HaloTokens.
type HomeAuthFactory func(ctx context.Context, slug string) (svc port.HomeService, enrichedCtx context.Context, xuid, gamertag string, err error)

// HomeHandler gère les endpoints de la page d'accueil Mission Control.
type HomeHandler struct {
	newSvc        HomeAuthFactory
	settingsStore *settings_platform.Store
}

// NewHomeHandler crée un HomeHandler.
func NewHomeHandler(newSvc HomeAuthFactory, settingsStore *settings_platform.Store) *HomeHandler {
	return &HomeHandler{newSvc: newSvc, settingsStore: settingsStore}
}

func (h *HomeHandler) locale() string {
	if h.settingsStore == nil {
		return "fr"
	}
	settings, err := h.settingsStore.Load()
	if err != nil || settings == nil {
		return "fr"
	}
	if strings.HasPrefix(strings.ToLower(strings.TrimSpace(settings.Lang)), "en") {
		return "en"
	}
	return "fr"
}

// GetHomePage retourne la page d'accueil agrégée.
// GET /api/v1/players/{player_slug}/pages/home
func (h *HomeHandler) GetHomePage(w http.ResponseWriter, r *http.Request) {
	slug := chi.URLParam(r, "player_slug")
	svc, ctx, _, gamertag, err := h.newSvc(r.Context(), slug)
	if err != nil {
		slog.ErrorContext(r.Context(), "home: newSvc error", "slug", slug, "err", err)
		writeError(w, http.StatusNotFound, "player_not_found", "joueur introuvable")
		return
	}

	page, err := svc.GetHomePage(ctx, gamertag, h.locale())
	if err != nil {
		slog.ErrorContext(ctx, "home: GetHomePage error", "err", err, "gamertag", gamertag)
		writeError(w, http.StatusInternalServerError, "home_page_error", "erreur chargement page d'accueil")
		return
	}

	writeJSON(w, http.StatusOK, page)
}

// GetBattlePass retourne les informations Battle Pass (best-effort).
// GET /api/v1/players/{player_slug}/battlepass
func (h *HomeHandler) GetBattlePass(w http.ResponseWriter, r *http.Request) {
	slug := chi.URLParam(r, "player_slug")
	svc, ctx, _, _, err := h.newSvc(r.Context(), slug)
	if err != nil {
		writeError(w, http.StatusNotFound, "player_not_found", "joueur introuvable")
		return
	}

	resp := svc.GetBattlePass(ctx)
	writeJSON(w, http.StatusOK, resp)
}

// GetChallenges retourne les défis actifs (best-effort).
// GET /api/v1/players/{player_slug}/challenges
func (h *HomeHandler) GetChallenges(w http.ResponseWriter, r *http.Request) {
	slug := chi.URLParam(r, "player_slug")
	svc, ctx, _, _, err := h.newSvc(r.Context(), slug)
	if err != nil {
		writeError(w, http.StatusNotFound, "player_not_found", "joueur introuvable")
		return
	}

	resp := svc.GetChallenges(ctx)
	writeJSON(w, http.StatusOK, resp)
}
