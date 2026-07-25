// Package handlers — handler GET /api/v1/titles/{slug}/capabilities (Phase 1.7a).
//
// Expose les capabilities produit déclarées par un titre (capabilities.toml),
// chargées au boot dans le mappings.Registry. Répond la question « que supporte
// ce titre » (intention statique title-level) — distinct de ce qu'expose une
// instance d'adapter player-scoped au runtime (career override, etc.).
//
// MIGRÉ vers Huma (Phase 3b) : Mount crée humacore.NewAPI(r) sur le sous-routeur
// chi (préfixe /api/v1) et enregistre le GET via huma.Get. Logique métier et
// contrat HTTP inchangés (corps JSON byte-identique, ETag SHA-256, 304 sur
// If-None-Match, Cache-Control) — seul le wrapping HTTP change.
//
// Gated par MULTI_TITLE_API_ENABLED (même flag que field-mappings).
package handlers

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"

	"github.com/danielgtaylor/huma/v2"
	"github.com/go-chi/chi/v5"

	"levelup/go-api/internal/api/humacore"
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

// Mount enregistre la route via Huma sur le sous-routeur chi (préfixe /api/v1).
func (h *CapabilitiesHandler) Mount(r chi.Router, opts ...humacore.MountOption) {
	api := humacore.NewAPI(r, opts...)
	huma.Get(api, "/titles/{slug}/capabilities", h.handleGet)
}

// ─── Input/Output Huma ───────────────────────────────────────────────────────

// capabilitiesInput : {slug} path + If-None-Match pour la négociation de cache.
type capabilitiesInput struct {
	Slug        string `path:"slug"`
	IfNoneMatch string `header:"If-None-Match"`
}

// capabilitiesOutput : corps RAW []byte (json.Marshal direct, byte-identique à
// l'ancien ServeHTTP) + headers de cache. Status override le défaut 200 (utilisé
// pour le 304). Les headers à valeur vide ne sont PAS émis (cf. huma writeHeader)
// — d'où le 304 sans Content-Type/Cache-Control/ETag, comme l'ancien handler.
type capabilitiesOutput struct {
	Status       int
	ContentType  string `header:"Content-Type"`
	CacheControl string `header:"Cache-Control"`
	ETag         string `header:"ETag"`
	Body         []byte
}

type capabilitiesResponse struct {
	TitleSlug     string            `json:"title_slug"`
	SchemaVersion int               `json:"schema_version"`
	Capabilities  map[string]string `json:"capabilities"` // capabilityKey → statut (supported|degraded|not_exposed)
}

// handleGet gère la requête.
func (h *CapabilitiesHandler) handleGet(ctx context.Context, in *capabilitiesInput) (*capabilitiesOutput, error) {
	slug := in.Slug
	if slug == "" {
		return nil, humacore.NewError(http.StatusBadRequest, "missing_slug", "title slug requis")
	}

	set, ok := h.registry.GetCapabilities(slug)
	if !ok {
		return nil, humacore.NewError(http.StatusNotFound, "title_not_found",
			fmt.Sprintf("title %q n'a pas de capabilities chargées", slug))
	}

	resp := capabilitiesResponse{
		TitleSlug:     set.TitleSlug(),
		SchemaVersion: set.SchemaVersion(),
		Capabilities:  set.All(),
	}

	body, err := json.Marshal(resp)
	if err != nil {
		return nil, humacore.NewError(http.StatusInternalServerError, "marshal_failed", err.Error())
	}

	sum := sha256.Sum256(body)
	etag := `"` + hex.EncodeToString(sum[:8]) + `"`
	if in.IfNoneMatch != "" && in.IfNoneMatch == etag {
		return &capabilitiesOutput{Status: http.StatusNotModified}, nil
	}

	h.logger.Debug("capabilities_served", "title_slug", slug, "count", len(resp.Capabilities))

	return &capabilitiesOutput{
		Status:       http.StatusOK,
		ContentType:  "application/json",
		CacheControl: "public, max-age=300",
		ETag:         etag,
		Body:         body,
	}, nil
}
