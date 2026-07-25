// Package handlers — handler GET /api/v1/titles/{slug}/feature-matrix (Phase 1.7b).
//
// Charge la CapabilityMap du titre (CapabilitiesRegistry) puis DÉLÈGUE le calcul
// de cascade à la fonction pure games.ComputeFeatureMatrix ; le handler ne fait
// que charger, sérialiser le domain.feature.Matrix et gérer le cache HTTP. Répond
// « quelles surfaces produit sont exposables pour ce titre » (3 états) — consommé
// par le frontend pour le feature-gating (<FeatureGate>, Phase 5). Title-agnostic :
// aucune branche sur le slug, tout découle de la CapabilityMap.
//
// MIGRÉ vers Huma (Phase 3b) : Mount crée humacore.NewAPI(r) et enregistre le GET
// via huma.Get. Logique métier inchangée ; le wrapping HTTP (path param, header
// conditionnel If-None-Match → 304, headers de cache, mapping d'erreurs) passe par
// les Input/Output Huma. Le corps est sérialisé en RawBody []byte (json.Marshal
// maison) pour préserver les octets EXACTS et l'ETag dérivé du corps (pas la
// sérialisation JSONFormat de humacore, qui ajouterait un "\n" final).
//
// Gated par MULTI_TITLE_API_ENABLED (même flag que field-mappings/capabilities).
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

// Mount enregistre la route via Huma sur le routeur chi `r` (qui porte déjà le
// préfixe /api/v1). Le chemin relatif est repris à l'identique de l'ancien
// r.Get(".../feature-matrix", ...).
func (h *FeatureMatrixHandler) Mount(r chi.Router, opts ...humacore.MountOption) {
	api := humacore.NewAPI(r, opts...)
	huma.Get(api, "/titles/{slug}/feature-matrix", h.handleGet)
}

type featureMatrixResponse struct {
	TitleSlug     string            `json:"title_slug"`
	SchemaVersion int               `json:"schema_version"` // version du capabilities.toml source (cohérence endpoints frères)
	Features      map[string]string `json:"features"`       // featureKey → statut (available|degraded|unavailable)
}

// featureMatrixInput : {slug} path + If-None-Match conditionnel (cache HTTP 304).
type featureMatrixInput struct {
	Slug        string `path:"slug"`
	IfNoneMatch string `header:"If-None-Match"`
}

// featureMatrixOutput reproduit le contrat HTTP d'origine :
//   - 200 : Body (octets json.Marshal bruts) + Content-Type/Cache-Control/ETag ;
//   - 304 : Status=304, tous les champs header/body vides → aucun header ni corps
//     écrit (writeHeader saute les strings vides, Body nil n'écrit rien).
//
// Body est []byte : Huma écrit les octets verbatim (pas de JSONFormat ni "\n"
// final), donc le corps reste byte-identique au json.Marshal qui dérive l'ETag.
type featureMatrixOutput struct {
	Status       int
	ContentType  string `header:"Content-Type"`
	CacheControl string `header:"Cache-Control"`
	ETag         string `header:"ETag"`
	Body         []byte
}

// handleGet gère la requête (logique métier inchangée vs l'ancien ServeHTTP).
func (h *FeatureMatrixHandler) handleGet(ctx context.Context, in *featureMatrixInput) (*featureMatrixOutput, error) {
	slug := in.Slug
	if slug == "" {
		return nil, humacore.NewError(http.StatusBadRequest, "missing_slug", "title slug requis")
	}

	set, ok := h.registry.GetCapabilities(slug)
	if !ok {
		return nil, humacore.NewError(http.StatusNotFound, "title_not_found",
			fmt.Sprintf("title %q n'a pas de capabilities chargées", slug))
	}

	caps, err := games.CapabilityMapFromMappings(set)
	if err != nil {
		// TOML déclare une capability hors vocabulaire produit — erreur de config.
		h.logger.ErrorContext(ctx, "feature_matrix_caps_invalid", "title_slug", slug, "err", err)
		return nil, humacore.NewError(http.StatusInternalServerError, "capabilities_invalid", err.Error())
	}

	matrix := games.ComputeFeatureMatrix(caps)
	features := make(map[string]string, len(matrix))
	for k, v := range matrix {
		features[string(k)] = string(v)
	}

	resp := featureMatrixResponse{TitleSlug: set.TitleSlug(), SchemaVersion: set.SchemaVersion(), Features: features}
	body, err := json.Marshal(resp)
	if err != nil {
		return nil, humacore.NewError(http.StatusInternalServerError, "marshal_failed", err.Error())
	}

	sum := sha256.Sum256(body)
	etag := `"` + hex.EncodeToString(sum[:8]) + `"`
	if in.IfNoneMatch != "" && in.IfNoneMatch == etag {
		return &featureMatrixOutput{Status: http.StatusNotModified}, nil
	}

	h.logger.Debug("feature_matrix_served", "title_slug", slug, "features", len(resp.Features))

	return &featureMatrixOutput{
		Status:       http.StatusOK,
		ContentType:  "application/json",
		CacheControl: "public, max-age=300",
		ETag:         etag,
		Body:         body,
	}, nil
}
