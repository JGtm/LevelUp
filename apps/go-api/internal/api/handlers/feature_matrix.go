// Package handlers — handler GET /api/v1/titles/{slug}/feature-matrix (Phase 1.7b).
//
// Charge la CapabilityMap du titre (CapabilitiesRegistry) puis DÉLÈGUE le calcul
// de cascade à la fonction pure games.ComputeFeatureMatrix ; le handler ne fait
// que charger, sérialiser le domain.feature.Matrix et gérer le cache HTTP. Répond
// « quelles surfaces produit sont exposables pour ce titre » (3 états) — consommé
// par le frontend pour le feature-gating (<FeatureGate>, Phase 5). Title-agnostic :
// aucune branche sur le slug, tout découle de la CapabilityMap.
//
// Gated par MULTI_TITLE_API_ENABLED (même flag que field-mappings/capabilities).
package handlers

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"

	"github.com/go-chi/chi/v5"

	"levelup/go-api/internal/games"
)

// FeatureMatrixHandler gère GET /api/v1/titles/{slug}/feature-matrix.
// Réutilise CapabilitiesRegistry (cf. capabilities.go) comme source des caps.
type FeatureMatrixHandler struct {
	registry CapabilitiesRegistry
	logger   *slog.Logger
}

// NewFeatureMatrixHandler crée un handler en injectant le registry des caps.
func NewFeatureMatrixHandler(reg CapabilitiesRegistry, logger *slog.Logger) *FeatureMatrixHandler {
	if logger == nil {
		logger = slog.Default()
	}
	return &FeatureMatrixHandler{registry: reg, logger: logger}
}

type featureMatrixResponse struct {
	TitleSlug     string            `json:"title_slug"`
	SchemaVersion int               `json:"schema_version"` // version du capabilities.toml source (cohérence endpoints frères)
	Features      map[string]string `json:"features"`       // featureKey → statut (available|degraded|unavailable)
}

// ServeHTTP gère la requête.
func (h *FeatureMatrixHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
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

	caps, err := games.CapabilityMapFromMappings(set)
	if err != nil {
		// TOML déclare une capability hors vocabulaire produit — erreur de config.
		h.logger.ErrorContext(r.Context(), "feature_matrix_caps_invalid", "title_slug", slug, "err", err)
		writeError(r.Context(), w, http.StatusInternalServerError, "capabilities_invalid", err.Error())
		return
	}

	matrix := games.ComputeFeatureMatrix(caps)
	features := make(map[string]string, len(matrix))
	for k, v := range matrix {
		features[string(k)] = string(v)
	}

	resp := featureMatrixResponse{TitleSlug: set.TitleSlug(), SchemaVersion: set.SchemaVersion(), Features: features}
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

	h.logger.Debug("feature_matrix_served", "title_slug", slug, "features", len(resp.Features))
}
