// Package handlers — catalog.go : Phase H du plan PLAN_PLAYLISTS_CATALOG.md.
//
// Endpoints REST title-aware exposant le catalogue Playlists/Pairs/Maps :
//
//	GET /api/v1/titles/{slug}/catalog/playlists?xuid={xuid}&only_played={bool}
//	GET /api/v1/titles/{slug}/catalog/pairs?playlist_asset_id={uuid}
//	GET /api/v1/titles/{slug}/catalog/maps?xuid={xuid}&only_played={bool}
//
// Gated par MULTI_TITLE_API_ENABLED (cf. field_mappings.go).
//
// MIGRÉ vers Huma (Phase 3b) : Mount crée humacore.NewAPI(r) et enregistre les 3
// GET via huma.Get. Logique métier inchangée (CatalogRepo), seul le wrapping HTTP
// change. Le header Cache-Control: public, max-age=300 est préservé via un champ
// header sur les structs Output.
package handlers

import (
	"context"
	"log/slog"
	"net/http"
	"strconv"

	"github.com/danielgtaylor/huma/v2"
	"github.com/go-chi/chi/v5"

	"levelup/go-api/internal/api/humacore"
	"levelup/go-api/internal/domain"
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

// Mount enregistre les 3 routes catalogue via Huma sur le routeur chi `r`
// (préfixe exact /titles/{slug}/catalog/... relatif au point de montage).
func (h *CatalogHandler) Mount(r chi.Router, opts ...humacore.MountOption) {
	api := humacore.NewAPI(r, opts...)
	huma.Get(api, "/titles/{slug}/catalog/playlists", h.handlePlaylists)
	huma.Get(api, "/titles/{slug}/catalog/pairs", h.handlePairs)
	huma.Get(api, "/titles/{slug}/catalog/maps", h.handleMaps)
}

// ─── Inputs/Outputs Huma ─────────────────────────────────────────────────────

// catalogPlaylistsInput : {slug} + ?xuid=&only_played= (tous tolérés vides).
type catalogPlaylistsInput struct {
	Slug       string `path:"slug"`
	XUID       string `query:"xuid"`
	OnlyPlayed string `query:"only_played"`
}

// catalogPairsInput : {slug} + ?playlist_asset_id= optionnel.
type catalogPairsInput struct {
	Slug            string `path:"slug"`
	PlaylistAssetID string `query:"playlist_asset_id"`
}

// catalogMapsInput : {slug} + ?xuid=&only_played= (tous tolérés vides).
type catalogMapsInput struct {
	Slug       string `path:"slug"`
	XUID       string `query:"xuid"`
	OnlyPlayed string `query:"only_played"`
}

// Outputs : corps JSON byte-identiques aux anciens map[string]any (mêmes clés,
// même ordre des champs). Cache-Control préservé via le champ header.
type catalogPlaylistsOutput struct {
	CacheControl string `header:"Cache-Control"`
	Body         struct {
		TitleSlug string                   `json:"title_slug"`
		Playlists []domain.CatalogPlaylist `json:"playlists"`
	}
}

type catalogPairsOutput struct {
	CacheControl string `header:"Cache-Control"`
	Body         struct {
		TitleSlug       string               `json:"title_slug"`
		PlaylistAssetID string               `json:"playlist_asset_id"`
		Pairs           []domain.CatalogPair `json:"pairs"`
	}
}

type catalogMapsOutput struct {
	CacheControl string `header:"Cache-Control"`
	Body         struct {
		TitleSlug string              `json:"title_slug"`
		Maps      []domain.CatalogMap `json:"maps"`
	}
}

const catalogCacheControl = "public, max-age=300"

// ─── Endpoints ───────────────────────────────────────────────────────────────

// handlePlaylists sert GET /titles/{slug}/catalog/playlists.
func (h *CatalogHandler) handlePlaylists(ctx context.Context, in *catalogPlaylistsInput) (*catalogPlaylistsOutput, error) {
	if in.Slug == "" {
		return nil, humacore.NewError(http.StatusBadRequest, "missing_slug", "title slug requis")
	}
	onlyPlayed, perr := strconv.ParseBool(in.OnlyPlayed)
	if perr != nil && in.OnlyPlayed != "" {
		slog.WarnContext(ctx, "catalog: only_played invalide, défaut false",
			"slug", in.Slug, "only_played", in.OnlyPlayed, "err", perr)
	}

	playlists, err := h.repo.PlaylistsByTitle(ctx, in.Slug, in.XUID, onlyPlayed)
	if err != nil {
		return nil, humacore.NewError(http.StatusInternalServerError, "catalog_query_failed", err.Error())
	}
	out := &catalogPlaylistsOutput{CacheControl: catalogCacheControl}
	out.Body.TitleSlug = in.Slug
	out.Body.Playlists = playlists
	return out, nil
}

// handlePairs sert GET /titles/{slug}/catalog/pairs.
func (h *CatalogHandler) handlePairs(ctx context.Context, in *catalogPairsInput) (*catalogPairsOutput, error) {
	if in.Slug == "" {
		return nil, humacore.NewError(http.StatusBadRequest, "missing_slug", "title slug requis")
	}
	pairs, err := h.repo.PairsByPlaylist(ctx, in.Slug, in.PlaylistAssetID) // playlist_asset_id optionnel
	if err != nil {
		return nil, humacore.NewError(http.StatusInternalServerError, "catalog_query_failed", err.Error())
	}
	out := &catalogPairsOutput{CacheControl: catalogCacheControl}
	out.Body.TitleSlug = in.Slug
	out.Body.PlaylistAssetID = in.PlaylistAssetID
	out.Body.Pairs = pairs
	return out, nil
}

// handleMaps sert GET /titles/{slug}/catalog/maps.
func (h *CatalogHandler) handleMaps(ctx context.Context, in *catalogMapsInput) (*catalogMapsOutput, error) {
	if in.Slug == "" {
		return nil, humacore.NewError(http.StatusBadRequest, "missing_slug", "title slug requis")
	}
	onlyPlayed, perr := strconv.ParseBool(in.OnlyPlayed)
	if perr != nil && in.OnlyPlayed != "" {
		slog.WarnContext(ctx, "catalog: only_played invalide, défaut false",
			"slug", in.Slug, "only_played", in.OnlyPlayed, "err", perr)
	}

	maps, err := h.repo.MapsByTitle(ctx, in.Slug, in.XUID, onlyPlayed)
	if err != nil {
		return nil, humacore.NewError(http.StatusInternalServerError, "catalog_query_failed", err.Error())
	}
	out := &catalogMapsOutput{CacheControl: catalogCacheControl}
	out.Body.TitleSlug = in.Slug
	out.Body.Maps = maps
	return out, nil
}
