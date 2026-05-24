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

	duckdbpkg "levelup/go-api/internal/platform/duckdb"
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

// resolveLocale détermine la locale à utiliser pour cette requête.
// Priorité : header X-LevelUp-Locale (envoyé par le frontend) → settings store
// (app_settings.json:lang) → "fr" par défaut.
//
// Le header permet au frontend de basculer la locale en runtime sans dépendre
// d'un re-bootstrap après modification de app_settings.json.
func (h *HomeHandler) resolveLocale(r *http.Request) string {
	if v := strings.ToLower(strings.TrimSpace(r.Header.Get("X-LevelUp-Locale"))); v != "" {
		if strings.HasPrefix(v, "en") {
			return "en"
		}
		if strings.HasPrefix(v, "fr") {
			return "fr"
		}
	}
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
		writeError(r.Context(), w, http.StatusNotFound, "player_not_found", "joueur introuvable")
		return
	}

	page, err := svc.GetHomePage(ctx, gamertag, h.resolveLocale(r))
	if err != nil {
		slog.ErrorContext(ctx, "home: GetHomePage error", "err", err, "gamertag", gamertag)
		// Phase 5 ART : distinguer FATAL DB (recovery en cours, retry possible)
		// d'une erreur métier permanente. Pour le scénario du crash home
		// 2026-05-24 20:41:04 (player DB invalidée par crash ART sur autre table),
		// le caller peut re-tenter quelques secondes plus tard une fois le
		// Reopen() effectué côté provider.
		if isHandleClosedOrInvalidated(err) {
			w.Header().Set("Retry-After", "5")
			writeError(r.Context(), w, http.StatusServiceUnavailable,
				"home_page_db_recovering",
				"page d'accueil temporairement indisponible — connexion DB en cours de récupération")
			return
		}
		writeError(r.Context(), w, http.StatusInternalServerError, "home_page_error", "erreur chargement page d'accueil")
		return
	}

	writeJSONCached(w, r, http.StatusOK, page)
}

// isHandleClosedOrInvalidated reconnaît les erreurs DuckDB qui justifient
// un 503 + Retry-After au lieu d'un 500. Couvre :
//   - `sql: database is closed` (handle fermée mais reopen possible)
//   - `database has been invalidated...` (FATAL DuckDB, cf. IsInvalidatedError)
//
// Le caller doit ré-essayer dans quelques secondes une fois le Reopen() effectué.
func isHandleClosedOrInvalidated(err error) bool {
	if err == nil {
		return false
	}
	if duckdbpkg.IsInvalidatedError(err) {
		return true
	}
	return strings.Contains(err.Error(), "database is closed")
}

// GetBattlePass retourne les informations Battle Pass (best-effort).
// GET /api/v1/players/{player_slug}/battlepass
func (h *HomeHandler) GetBattlePass(w http.ResponseWriter, r *http.Request) {
	slug := chi.URLParam(r, "player_slug")
	svc, ctx, _, _, err := h.newSvc(r.Context(), slug)
	if err != nil {
		writeError(r.Context(), w, http.StatusNotFound, "player_not_found", "joueur introuvable")
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
		writeError(r.Context(), w, http.StatusNotFound, "player_not_found", "joueur introuvable")
		return
	}

	resp := svc.GetChallenges(ctx)
	writeJSON(w, http.StatusOK, resp)
}
