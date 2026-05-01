// Package handlers — catalog.go : Phase H du plan PLAN_PLAYLISTS_CATALOG.md.
//
// Endpoints REST title-aware exposant le catalogue Playlists/Pairs/Maps :
//
//	GET /api/v1/titles/{slug}/catalog/playlists?xuid={xuid}&only_played={bool}
//	GET /api/v1/titles/{slug}/catalog/pairs?playlist_asset_id={uuid}
//	GET /api/v1/titles/{slug}/catalog/maps?xuid={xuid}&only_played={bool}
//
// Gated par MULTI_TITLE_API_ENABLED (cf. field_mappings.go).
package handlers

import (
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"

	"levelup/go-api/internal/port"
)

// CatalogHandler regroupe les 3 endpoints catalogue.
type CatalogHandler struct {
	repo port.CatalogRepo
}

// NewCatalogHandler construit le handler en injectant le repo.
func NewCatalogHandler(repo port.CatalogRepo) *CatalogHandler {
	return &CatalogHandler{repo: repo}
}

// PlaylistsHandler sert GET /api/v1/titles/{slug}/catalog/playlists.
func (h *CatalogHandler) PlaylistsHandler(w http.ResponseWriter, r *http.Request) {
	slug := chi.URLParam(r, "slug")
	if slug == "" {
		writeError(w, http.StatusBadRequest, "missing_slug", "title slug requis")
		return
	}
	xuid := r.URL.Query().Get("xuid")
	onlyPlayed, _ := strconv.ParseBool(r.URL.Query().Get("only_played"))

	playlists, err := h.repo.PlaylistsByTitle(r.Context(), slug, xuid, onlyPlayed)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "catalog_query_failed", err.Error())
		return
	}
	writeJSONCatalog(w, map[string]any{
		"title_slug": slug,
		"playlists":  playlists,
	})
}

// PairsHandler sert GET /api/v1/titles/{slug}/catalog/pairs.
func (h *CatalogHandler) PairsHandler(w http.ResponseWriter, r *http.Request) {
	slug := chi.URLParam(r, "slug")
	if slug == "" {
		writeError(w, http.StatusBadRequest, "missing_slug", "title slug requis")
		return
	}
	playlistID := r.URL.Query().Get("playlist_asset_id") // optionnel

	pairs, err := h.repo.PairsByPlaylist(r.Context(), slug, playlistID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "catalog_query_failed", err.Error())
		return
	}
	writeJSONCatalog(w, map[string]any{
		"title_slug":        slug,
		"playlist_asset_id": playlistID,
		"pairs":             pairs,
	})
}

// MapsHandler sert GET /api/v1/titles/{slug}/catalog/maps.
func (h *CatalogHandler) MapsHandler(w http.ResponseWriter, r *http.Request) {
	slug := chi.URLParam(r, "slug")
	if slug == "" {
		writeError(w, http.StatusBadRequest, "missing_slug", "title slug requis")
		return
	}
	xuid := r.URL.Query().Get("xuid")
	onlyPlayed, _ := strconv.ParseBool(r.URL.Query().Get("only_played"))

	maps, err := h.repo.MapsByTitle(r.Context(), slug, xuid, onlyPlayed)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "catalog_query_failed", err.Error())
		return
	}
	writeJSONCatalog(w, map[string]any{
		"title_slug": slug,
		"maps":       maps,
	})
}

func writeJSONCatalog(w http.ResponseWriter, body any) {
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Cache-Control", "public, max-age=300")
	if err := json.NewEncoder(w).Encode(body); err != nil {
		writeError(w, http.StatusInternalServerError, "marshal_failed", err.Error())
	}
}
