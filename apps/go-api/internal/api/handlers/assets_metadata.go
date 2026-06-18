// Package handlers — assets_metadata.go : endpoints listing maps & armes pour l'Asset Drawer.
//
// MIGRÉ vers Huma (Phase 3b) : Mount crée humacore.NewAPI(r) sur le routeur fourni
// et enregistre les 2 GET via huma.Get. Routes TOP-LEVEL (sous /api/v1) — {title_id}
// est le seul path param (pas de player_slug). Logique métier inchangée
// (AssetService + gate hasCapability), seul le wrapping HTTP change.
package handlers

import (
	"context"
	"log/slog"
	"net/http"

	"github.com/danielgtaylor/huma/v2"
	"github.com/go-chi/chi/v5"

	"levelup/go-api/internal/api/humacore"
	titlePkg "levelup/go-api/internal/domain/title"
	"levelup/go-api/internal/games/canonical"
	"levelup/go-api/internal/port"
)

// AssetMetadataHandler liste les métadonnées d'assets (maps, armes) pour l'Asset Drawer.
type AssetMetadataHandler struct {
	svc           port.AssetService
	hasCapability func(titleSlug string, cap titlePkg.Capability) bool
}

// NewAssetMetadataHandler crée un AssetMetadataHandler.
// hasCapability gate les endpoints — retourne false si le titre ne supporte pas asset.images.
func NewAssetMetadataHandler(
	svc port.AssetService,
	hasCapability func(titleSlug string, cap titlePkg.Capability) bool,
) *AssetMetadataHandler {
	return &AssetMetadataHandler{svc: svc, hasCapability: hasCapability}
}

// Mount enregistre les 2 routes via Huma sur le routeur chi fourni
// (préfixe hérité du point de montage, ex. /api/v1).
func (h *AssetMetadataHandler) Mount(r chi.Router) {
	api := humacore.NewAPI(r)
	huma.Get(api, "/assets/{title_id}/maps", h.ListMaps)
	huma.Get(api, "/assets/{title_id}/weapons", h.ListWeapons)
}

// ─── Inputs/Outputs Huma ─────────────────────────────────────────────────────

// assetMetaInput : {title_id} path param + ?q= optionnel (filtre de recherche).
type assetMetaInput struct {
	TitleID string `path:"title_id"`
	Search  string `query:"q"`
}

// assetMetaOutput : liste d'AssetMeta sérialisée à plat (même corps que writeJSON).
type assetMetaOutput struct {
	Body []canonical.AssetMeta
}

// ─── Endpoints ───────────────────────────────────────────────────────────────

// ListMaps liste les maps d'un titre pour l'Asset Drawer.
// GET /api/v1/assets/{title_id}/maps?q=
func (h *AssetMetadataHandler) ListMaps(ctx context.Context, in *assetMetaInput) (*assetMetaOutput, error) {
	if !h.hasCapability(in.TitleID, titlePkg.CapAssetImages) {
		return nil, humacore.NewError(http.StatusNotFound, "asset_capability_unavailable",
			"capability asset.images non supportée pour ce titre")
	}

	items, err := h.svc.ListMaps(ctx, in.TitleID, in.Search)
	if err != nil {
		slog.ErrorContext(ctx, "ListMaps failed", "err", err, "title", in.TitleID)
		return nil, humacore.NewError(http.StatusInternalServerError, "internal_error", "erreur interne")
	}

	slog.DebugContext(ctx, "ListMaps", "title", in.TitleID, "q", in.Search, "n", len(items))
	if items == nil {
		items = []canonical.AssetMeta{}
	}
	return &assetMetaOutput{Body: items}, nil
}

// ListWeapons liste les armes d'un titre pour l'Asset Drawer.
// GET /api/v1/assets/{title_id}/weapons?q=
func (h *AssetMetadataHandler) ListWeapons(ctx context.Context, in *assetMetaInput) (*assetMetaOutput, error) {
	if !h.hasCapability(in.TitleID, titlePkg.CapAssetImages) {
		return nil, humacore.NewError(http.StatusNotFound, "asset_capability_unavailable",
			"capability asset.images non supportée pour ce titre")
	}

	items, err := h.svc.ListWeapons(ctx, in.TitleID, in.Search)
	if err != nil {
		slog.ErrorContext(ctx, "ListWeapons failed", "err", err, "title", in.TitleID)
		return nil, humacore.NewError(http.StatusInternalServerError, "internal_error", "erreur interne")
	}

	slog.DebugContext(ctx, "ListWeapons", "title", in.TitleID, "q", in.Search, "n", len(items))
	if items == nil {
		items = []canonical.AssetMeta{}
	}
	return &assetMetaOutput{Body: items}, nil
}
