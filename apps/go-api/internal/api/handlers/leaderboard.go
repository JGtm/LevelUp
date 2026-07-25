// Package handlers — leaderboard.go : handler HTTP pour le classement CSR.
//
// Endpoints :
//
//	GET /api/v1/players/{player_slug}/pages/leaderboard         → LeaderboardResponse
//	GET /api/v1/players/{player_slug}/pages/leaderboard/catalog → LeaderboardCatalog
//
// MIGRÉ vers Huma (Phase 3b) : Mount crée humacore.NewAPI(r) sur le sous-routeur
// (préfixe /players/{player_slug} + middleware ownership/title hérités, lit
// {player_slug} parent) et enregistre les 2 GET via huma.Get. Logique métier
// inchangée (LeaderboardService), seul le wrapping HTTP change.
//
// Sprint 54 E.
package handlers

import (
	"context"
	"net/http"
	"strconv"
	"strings"

	"github.com/danielgtaylor/huma/v2"
	"github.com/go-chi/chi/v5"

	"levelup/go-api/internal/api/humacore"
	"levelup/go-api/internal/domain"
	"levelup/go-api/internal/port"
)

// LeaderboardHandler gère l'endpoint du classement CSR.
type LeaderboardHandler struct {
	newSvc ContextFactory[port.LeaderboardService]
}

// NewLeaderboardHandler crée un LeaderboardHandler.
func NewLeaderboardHandler(newSvc ContextFactory[port.LeaderboardService]) *LeaderboardHandler {
	return &LeaderboardHandler{newSvc: newSvc}
}

// Mount enregistre les 2 routes via Huma sur le sous-routeur chi (préfixe
// /players/{player_slug} + middleware ownership/title hérités).
func (h *LeaderboardHandler) Mount(r chi.Router, opts ...humacore.MountOption) {
	api := humacore.NewAPI(r, opts...)
	huma.Get(api, "/pages/leaderboard", h.handleGetPage, humacore.Op("getLeaderboardPage", "Résumé CSR local du joueur courant", "leaderboard"))
	huma.Get(api, "/pages/leaderboard/catalog", h.handleGetCatalog, humacore.Op("getLeaderboardCatalog", "Saisons + playlists disponibles pour les sélecteurs du classement CSR mondial", "leaderboard"))
}

// ─── Inputs/Outputs Huma ─────────────────────────────────────────────────────

// leaderboardPageInput : {player_slug} parent + query params de filtrage.
// Limit pris en STRING (pas int) pour reproduire le contrat d'origine — un limit
// non numérique est toléré (→ 0), PAS le 422 de validation Huma qu'un `int`
// produirait.
type leaderboardPageInput struct {
	PlayerSlug string `path:"player_slug"`
	Category   string `query:"category"`
	Season     string `query:"season"`
	Playlist   string `query:"playlist"`
	TitleSlug  string `query:"title_slug"`
	Limit      string `query:"limit"`
}

// leaderboardCatalogInput : path param parent {player_slug} (pas de query).
type leaderboardCatalogInput struct {
	PlayerSlug string `path:"player_slug"`
}

type leaderboardPageOutput struct{ Body domain.LeaderboardResponse }
type leaderboardCatalogOutput struct{ Body domain.LeaderboardCatalog }

// ─── Endpoints ───────────────────────────────────────────────────────────────

// handleGetPage retourne le classement CSR.
// GET /api/v1/players/{player_slug}/pages/leaderboard?season=...&playlist=...
func (h *LeaderboardHandler) handleGetPage(ctx context.Context, in *leaderboardPageInput) (*leaderboardPageOutput, error) {
	svc, _, _, err := h.newSvc(ctx, in.PlayerSlug)
	if err != nil {
		return nil, humacore.NewError(http.StatusNotFound, "player_not_found", err.Error())
	}

	limit := 0
	if v := strings.TrimSpace(in.Limit); v != "" {
		if n, err := strconv.Atoi(v); err == nil {
			limit = n
		}
	}
	req := domain.LeaderboardRequest{
		Category:  in.Category,
		Season:    in.Season,
		Playlist:  in.Playlist,
		TitleSlug: in.TitleSlug,
		Limit:     limit,
	}

	resp, err := svc.GetPage(ctx, req)
	if err != nil {
		return nil, humacore.NewError(http.StatusInternalServerError, "leaderboard_error", err.Error())
	}

	return &leaderboardPageOutput{Body: resp}, nil
}

// handleGetCatalog retourne les saisons + playlists disponibles (sélecteurs).
// GET /api/v1/players/{player_slug}/pages/leaderboard/catalog
func (h *LeaderboardHandler) handleGetCatalog(ctx context.Context, in *leaderboardCatalogInput) (*leaderboardCatalogOutput, error) {
	svc, _, _, err := h.newSvc(ctx, in.PlayerSlug)
	if err != nil {
		return nil, humacore.NewError(http.StatusNotFound, "player_not_found", err.Error())
	}
	catalog, err := svc.GetCatalog(ctx)
	if err != nil {
		return nil, humacore.NewError(http.StatusInternalServerError, "leaderboard_error", err.Error())
	}
	return &leaderboardCatalogOutput{Body: catalog}, nil
}
