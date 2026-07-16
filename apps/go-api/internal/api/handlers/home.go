// Package handlers — home.go : handlers HTTP pour la page d'accueil Mission Control.
//
// MIGRÉ vers Huma (Phase 3b) : Mount crée humacore.NewAPI(r) sur le sous-routeur
// /players/{player_slug} (ownership/title hérités) et enregistre l'unique GET. Les
// en-têtes de cache (anciens middlewares CacheMaxAge/NoStore) sont posés dans les
// Output. /pages/home conserve son ETag/304 byte-exact via un Body []byte
// passthrough (writeJSONCached n'ajoute pas de trailing newline, contrairement à
// writeJSON) — même pattern que field_mappings.go.
//
// Endpoints :
//
//	GET /api/v1/players/{player_slug}/pages/home     → HomePageResponse (ETag/304, max-age 30)
package handlers

import (
	"context"
	"crypto/sha256"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"strings"

	"github.com/danielgtaylor/huma/v2"
	"github.com/go-chi/chi/v5"

	"levelup/go-api/internal/api/humacore"
	duckdbpkg "levelup/go-api/internal/platform/duckdb"
	"levelup/go-api/internal/platform/duckdb/sharedprovider"
	settings_platform "levelup/go-api/internal/platform/settings"
	"levelup/go-api/internal/port"
)

// homeCacheControl reproduit l'en-tête de l'ancien middleware CacheMaxAge(30).
const homeCacheControl = "public, max-age=30, stale-while-revalidate=15"

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

// Mount enregistre l'unique GET via Huma sur le sous-routeur chi (préfixe
// /players/{player_slug} + middleware ownership/title hérités).
func (h *HomeHandler) Mount(r chi.Router) {
	api := humacore.NewAPI(r)
	huma.Get(api, "/pages/home", h.handleGetHomePage)
}

// ─── Inputs/Outputs Huma ─────────────────────────────────────────────────────

// homePageInput : {player_slug} + X-LevelUp-Locale (résolution locale) + If-None-Match (ETag).
type homePageInput struct {
	PlayerSlug  string `path:"player_slug"`
	Locale      string `header:"X-LevelUp-Locale"`
	IfNoneMatch string `header:"If-None-Match"`
}

// homePageOutput émet le corps marshalé maison tel quel (Body []byte passthrough,
// SANS trailing newline — byte-exact vs writeJSONCached). Status dynamique (200/304).
type homePageOutput struct {
	Status       int
	ContentType  string `header:"Content-Type"`
	CacheControl string `header:"Cache-Control"`
	ETag         string `header:"ETag"`
	Body         []byte
}

// resolveLocaleFromHeader détermine la locale à utiliser pour cette requête.
// Priorité : header X-LevelUp-Locale (envoyé par le frontend) → settings store
// (app_settings.json:lang) → "fr" par défaut.
//
// Le header permet au frontend de basculer la locale en runtime sans dépendre
// d'un re-bootstrap après modification de app_settings.json.
func (h *HomeHandler) resolveLocaleFromHeader(localeHeader string) string {
	if v := strings.ToLower(strings.TrimSpace(localeHeader)); v != "" {
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

// handleGetHomePage retourne la page d'accueil agrégée (migré Huma).
// GET /api/v1/players/{player_slug}/pages/home
func (h *HomeHandler) handleGetHomePage(ctx context.Context, in *homePageInput) (*homePageOutput, error) {
	svc, sctx, _, gamertag, err := h.newSvc(ctx, in.PlayerSlug)
	if err != nil {
		slog.ErrorContext(ctx, "home: newSvc error", "slug", in.PlayerSlug, "err", err)
		return nil, humacore.NewError(http.StatusNotFound, "player_not_found", "joueur introuvable")
	}

	page, err := svc.GetHomePage(sctx, gamertag, h.resolveLocaleFromHeader(in.Locale))
	if err != nil {
		slog.ErrorContext(sctx, "home: GetHomePage error", "err", err, "gamertag", gamertag)
		// Phase 5 ART : distinguer FATAL DB (recovery en cours, retry possible)
		// d'une erreur métier permanente. Pour le scénario du crash home
		// 2026-05-24 20:41:04 (player DB invalidée par crash ART sur autre table),
		// le caller peut re-tenter quelques secondes plus tard une fois le
		// Reopen() effectué côté provider.
		if isHandleClosedOrInvalidated(err) {
			return nil, huma.ErrorWithHeaders(
				humacore.NewError(http.StatusServiceUnavailable, "home_page_db_recovering",
					"page d'accueil temporairement indisponible — connexion DB en cours de récupération"),
				http.Header{"Retry-After": []string{"5"}},
			)
		}
		// Contention de swap : le shared reader n'a pas pu être obtenu dans le budget
		// user-facing (un sync tient le writer RW). 503 Retry-After court plutôt qu'un
		// 500 opaque — la lecture est idempotente et repassera dès le retour en RO.
		if isSharedSwapContention(err) {
			return nil, huma.ErrorWithHeaders(
				humacore.NewError(http.StatusServiceUnavailable, "home_page_db_busy",
					"page d'accueil temporairement indisponible — base occupée par une synchronisation"),
				http.Header{"Retry-After": []string{"2"}},
			)
		}
		return nil, humacore.NewError(http.StatusInternalServerError, "home_page_error", "erreur chargement page d'accueil")
	}

	// Sérialisation byte-exacte de writeJSONCached : sanitize → json.Marshal → ETag
	// sha256[:8], SANS trailing newline (Body []byte passthrough, pas JSONFormat).
	sanitized, nanPaths := humacore.SanitizeFloatsForJSON(page)
	if len(nanPaths) > 0 {
		// Parité avec writeJSONCached : tracer la neutralisation NaN/Inf (signal serveur).
		slog.WarnContext(sctx, "home: NaN/Inf neutralized", "paths", nanPaths, "count", len(nanPaths))
	}
	body, merr := json.Marshal(sanitized)
	if merr != nil {
		return nil, humacore.NewError(http.StatusInternalServerError, "encode_error", "erreur de sérialisation")
	}
	sum := sha256.Sum256(body)
	etag := fmt.Sprintf(`"%x"`, sum[:8])
	if in.IfNoneMatch != "" && in.IfNoneMatch == etag {
		return &homePageOutput{Status: http.StatusNotModified, CacheControl: homeCacheControl, ETag: etag}, nil
	}
	return &homePageOutput{
		Status:       http.StatusOK,
		ContentType:  "application/json",
		CacheControl: homeCacheControl,
		ETag:         etag,
		Body:         body,
	}, nil
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

// isSharedSwapContention reconnaît les erreurs du SharedDBProvider signalant que
// la base partagée est momentanément indisponible car un sync tient le writer RW
// (swap en cours ou provider en récupération). Elles justifient un 503 + Retry-After
// court : la lecture est idempotente et repassera dès le retour en steady state RO.
func isSharedSwapContention(err error) bool {
	return errors.Is(err, sharedprovider.ErrSwapTimeout) ||
		errors.Is(err, sharedprovider.ErrSwapFailed) ||
		errors.Is(err, sharedprovider.ErrProviderClosed)
}
