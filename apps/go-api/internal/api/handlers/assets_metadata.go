// Package handlers — assets_metadata.go : endpoints listing maps & armes pour l'Asset Drawer.
package handlers

import (
	"log/slog"
	"net/http"

	"github.com/go-chi/chi/v5"

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

// ListMaps liste les maps d'un titre pour l'Asset Drawer.
// GET /api/v1/assets/{title_id}/maps?q=
func (h *AssetMetadataHandler) ListMaps(w http.ResponseWriter, r *http.Request) {
	titleID := chi.URLParam(r, "title_id")
	if !h.hasCapability(titleID, titlePkg.CapAssetImages) {
		httpError(r.Context(), w, "capability asset.images non supportée pour ce titre", http.StatusNotFound)
		return
	}

	search := r.URL.Query().Get("q")
	items, err := h.svc.ListMaps(r.Context(), titleID, search)
	if err != nil {
		slog.ErrorContext(r.Context(), "ListMaps failed", "err", err, "title", titleID)
		httpError(r.Context(), w, "erreur interne", http.StatusInternalServerError)
		return
	}

	slog.DebugContext(r.Context(), "ListMaps", "title", titleID, "q", search, "n", len(items))
	if items == nil {
		items = []canonical.AssetMeta{}
	}
	writeJSON(w, http.StatusOK, items)
}

// ListWeapons liste les armes d'un titre pour l'Asset Drawer.
// GET /api/v1/assets/{title_id}/weapons?q=
func (h *AssetMetadataHandler) ListWeapons(w http.ResponseWriter, r *http.Request) {
	titleID := chi.URLParam(r, "title_id")
	if !h.hasCapability(titleID, titlePkg.CapAssetImages) {
		httpError(r.Context(), w, "capability asset.images non supportée pour ce titre", http.StatusNotFound)
		return
	}

	search := r.URL.Query().Get("q")
	items, err := h.svc.ListWeapons(r.Context(), titleID, search)
	if err != nil {
		slog.ErrorContext(r.Context(), "ListWeapons failed", "err", err, "title", titleID)
		httpError(r.Context(), w, "erreur interne", http.StatusInternalServerError)
		return
	}

	slog.DebugContext(r.Context(), "ListWeapons", "title", titleID, "q", search, "n", len(items))
	if items == nil {
		items = []canonical.AssetMeta{}
	}
	writeJSON(w, http.StatusOK, items)
}
