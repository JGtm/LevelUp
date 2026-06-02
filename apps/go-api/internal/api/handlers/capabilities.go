// Package handlers — handler GET /api/v1/titles/{slug}/capabilities (Phase 1.7a).
//
// Expose les capabilities produit déclarées par un titre (capabilities.toml),
// chargées au boot dans le mappings.Registry. Répond la question « que supporte
// ce titre » (intention statique title-level) — distinct de ce qu'expose une
// instance d'adapter player-scoped au runtime (career override, etc.).
//
// Gated par MULTI_TITLE_API_ENABLED (même flag que field-mappings).
package handlers

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"

	"github.com/go-chi/chi/v5"

	"levelup/go-api/internal/games/mappings"
)

// CapabilitiesRegistry expose les CapabilityMappingSet chargés au boot.
// Implémenté par *mappings.Registry.
type CapabilitiesRegistry interface {
	GetCapabilities(titleSlug string) (*mappings.CapabilityMappingSet, bool)
}

// CapabilitiesHandler gère GET /api/v1/titles/{slug}/capabilities.
type CapabilitiesHandler struct {
	registry CapabilitiesRegistry
	logger   *slog.Logger
}

// NewCapabilitiesHandler crée un handler en injectant le registry.
func NewCapabilitiesHandler(reg CapabilitiesRegistry, logger *slog.Logger) *CapabilitiesHandler {
	if logger == nil {
		logger = slog.Default()
	}
	return &CapabilitiesHandler{registry: reg, logger: logger}
}

type capabilitiesResponse struct {
	TitleSlug     string            `json:"title_slug"`
	SchemaVersion int               `json:"schema_version"`
	Capabilities  map[string]string `json:"capabilities"` // capabilityKey → statut (supported|degraded|not_exposed)
}

// ServeHTTP gère la requête.
func (h *CapabilitiesHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	slug := chi.URLParam(r, "slug")
	if slug == "" {
		writeError(r.Context(), w, http.StatusBadRequest, "missing_slug", "title slug requis")
		return
	}

	set, ok := h.registry.GetCapabilities(slug)
	if !ok {
		writeError(r.Context(), w, http.StatusNotFound, "title_not_found",
			fmt.Sprintf("title %q n'a pas de capabilities chargées", slug))
		return
	}

	resp := capabilitiesResponse{
		TitleSlug:     set.TitleSlug(),
		SchemaVersion: set.SchemaVersion(),
		Capabilities:  set.All(),
	}

	body, err := json.Marshal(resp)
	if err != nil {
		writeError(r.Context(), w, http.StatusInternalServerError, "marshal_failed", err.Error())
		return
	}

	sum := sha256.Sum256(body)
	etag := `"` + hex.EncodeToString(sum[:8]) + `"`
	if match := r.Header.Get("If-None-Match"); match != "" && match == etag {
		w.WriteHeader(http.StatusNotModified)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Cache-Control", "public, max-age=300")
	w.Header().Set("ETag", etag)
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write(body)

	h.logger.Debug("capabilities_served", "title_slug", slug, "count", len(resp.Capabilities))
}
