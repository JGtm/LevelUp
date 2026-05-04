// Package handlers — handler GET /api/v1/titles/{slug}/field-mappings.
//
// Phase A du plan multi-titres : endpoint d'exposition des libellés et
// présentation des FieldKey canoniques par titre. Lit les TOML chargés au
// boot par le FieldMappingRegistry.
package handlers

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/go-chi/chi/v5"

	"levelup/go-api/internal/games/mappings"
)

// FieldMappingsRegistry expose les FieldMappingSet/AssetMappingSet/
// OutcomeMappingSet chargés au boot.
type FieldMappingsRegistry interface {
	Get(titleSlug string) (*mappings.FieldMappingSet, bool)
	GetAssets(titleSlug string) (*mappings.AssetMappingSet, bool)
	GetOutcomes(titleSlug string) (*mappings.OutcomeMappingSet, bool)
}

// SeasonsCatalogResolver résout le catalog unifié des saisons (TOML + DB +
// lazy fetch) pour un titre. Implémenté par service.SeasonsCatalog.
//
// L'interface est définie ici pour découpler le handler de l'implémentation
// concrète (tests : injecter un fake ; runtime : injecter le catalog du
// service.SeasonsCatalog wiré dans server.go).
type SeasonsCatalogResolver interface {
	Load(ctx context.Context, titleID string) []SeasonCatalogEntry
}

// SeasonCatalogEntry est la projection minimale d'une saison utilisée par le
// handler pour enrichir le DTO assets. Aligné sur service.SeasonCatalogEntry
// mais redéfini ici pour éviter l'import circulaire (les types domain/canonical
// ne suffisent pas car on a besoin du Label localisé).
type SeasonCatalogEntry struct {
	ID           string
	Label        string
	Start        time.Time
	End          *time.Time
	DisplayOrder int
	Extra        map[string]string
}

// FieldMappingsHandler gère GET /api/v1/titles/{slug}/field-mappings.
//
// Le handler n'est enregistré que si MULTI_TITLE_API_ENABLED=true au boot.
// En Phase A : flag off par défaut → endpoint absent du routeur.
type FieldMappingsHandler struct {
	registry FieldMappingsRegistry
	seasons  SeasonsCatalogResolver // optionnel : nil → kind season exposé tel quel depuis TOML
	logger   *slog.Logger

	mu        sync.RWMutex
	etagByKey map[string]string
}

// NewFieldMappingsHandler crée un handler en injectant le registry.
func NewFieldMappingsHandler(reg FieldMappingsRegistry, logger *slog.Logger) *FieldMappingsHandler {
	if logger == nil {
		logger = slog.Default()
	}
	return &FieldMappingsHandler{
		registry:  reg,
		logger:    logger,
		etagByKey: make(map[string]string),
	}
}

// WithSeasonsCatalog branche le résolveur unifié (TOML + DB + lazy fetch).
// Quand câblé, le DTO assets.season retourné par le handler contient l'union
// TOML + DB : les saisons découvertes en DB mais absentes du TOML apparaissent
// avec leur libellé Waypoint brut (en attente de traduction FR via update du
// TOML). Symétrie avec FiltersService qui consomme le même catalog pour les
// SeasonCounts.
func (h *FieldMappingsHandler) WithSeasonsCatalog(resolver SeasonsCatalogResolver) *FieldMappingsHandler {
	h.seasons = resolver
	return h
}

// MultiTitleAPIEnabled retourne true si la feature flag MULTI_TITLE_API_ENABLED
// est activée. Par défaut false en Phase A.
func MultiTitleAPIEnabled() bool {
	v := strings.ToLower(strings.TrimSpace(os.Getenv("MULTI_TITLE_API_ENABLED")))
	return v == "1" || v == "true" || v == "yes"
}

type fieldMappingDTO struct {
	Label        string `json:"label"`
	Description  string `json:"description,omitempty"`
	StorageUnit  string `json:"storage_unit"`
	DisplayUnit  string `json:"display_unit"`
	Format       string `json:"format"`
	DisplayOrder int    `json:"display_order"`
	Group        string `json:"group"`
	Icon         string `json:"icon,omitempty"`
}

type assetMappingDTO struct {
	Label        string            `json:"label"`
	ColorToken   string            `json:"color_token,omitempty"`
	Icon         string            `json:"icon,omitempty"`
	DisplayOrder int               `json:"display_order"`
	StartDate    *time.Time        `json:"start_date,omitempty"`
	EndDate      *time.Time        `json:"end_date,omitempty"`
	Extra        map[string]string `json:"extra,omitempty"`
}

type outcomeMappingDTO struct {
	Label      string `json:"label"`
	ColorToken string `json:"color_token"`
}

type fieldMappingsResponse struct {
	TitleSlug     string                                `json:"title_slug"`
	SchemaVersion int                                   `json:"schema_version"`
	Locale        string                                `json:"locale"`
	Fields        map[string]fieldMappingDTO            `json:"fields"`
	Assets        map[string]map[string]assetMappingDTO `json:"assets,omitempty"`
	Outcomes      map[string]outcomeMappingDTO          `json:"outcomes,omitempty"`
}

// ServeHTTP gère la requête.
func (h *FieldMappingsHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	slug := chi.URLParam(r, "slug")
	if slug == "" {
		writeError(w, http.StatusBadRequest, "missing_slug", "title slug requis")
		return
	}

	// Locale inconnue → la fallback EN est gérée par FieldMapping.Label.
	// On conserve la valeur d'origine dans la réponse pour que le frontend
	// sache ce qu'il a demandé.
	locale := r.URL.Query().Get("locale")
	if locale == "" {
		locale = mappings.LocaleFR
	}

	set, ok := h.registry.Get(slug)
	if !ok {
		writeError(w, http.StatusNotFound, "title_not_found",
			fmt.Sprintf("title %q n'a pas de field mappings chargés", slug))
		return
	}

	resp := fieldMappingsResponse{
		TitleSlug:     set.TitleSlug(),
		SchemaVersion: set.SchemaVersion(),
		Locale:        locale,
		Fields:        make(map[string]fieldMappingDTO, len(set.All())),
	}

	for _, m := range set.All() {
		label, fellback := m.Label(locale)
		if fellback {
			h.logger.Debug("field_mappings_fallback",
				"title_slug", set.TitleSlug(),
				"field_key", m.Key,
				"locale", locale,
			)
		}
		desc, _ := m.Description(locale)
		resp.Fields[string(m.Key)] = fieldMappingDTO{
			Label:        label,
			Description:  desc,
			StorageUnit:  string(m.StorageUnit),
			DisplayUnit:  string(m.DisplayUnit),
			Format:       string(m.Format),
			DisplayOrder: m.DisplayOrder,
			Group:        m.Group,
			Icon:         m.Icon,
		}
	}

	// Phase 1 plan finition multi-titres : exposer les assets s'ils sont chargés.
	if assets, ok := h.registry.GetAssets(slug); ok && assets != nil {
		out := make(map[string]map[string]assetMappingDTO, len(assets.Kinds()))
		for _, kind := range assets.Kinds() {
			byID := make(map[string]assetMappingDTO)
			for _, a := range assets.AllOfKind(kind) {
				label, _ := a.Label(locale)
				byID[a.ID] = assetMappingDTO{
					Label:        label,
					ColorToken:   a.ColorToken,
					Icon:         a.Icon,
					DisplayOrder: a.DisplayOrder,
					StartDate:    a.StartDate,
					EndDate:      a.EndDate,
					Extra:        a.Extra,
				}
			}
			out[kind] = byID
		}
		resp.Assets = out
	}

	// V2 saisons : si le SeasonsCatalogResolver est câblé, on remplace
	// purement le bucket "season" du DTO par le résultat du resolver — qui
	// fait l'union TOML + DB + éventuel lazy fetch live (avec persistance).
	// Cela permet à une nouvelle Operation Halo découverte en DB d'apparaître
	// automatiquement dans la SaisonPill côté frontend, sans intervention
	// manuelle sur le TOML. Les saisons DB-only (pas encore de FR) sont
	// affichées avec leur libellé Waypoint brut.
	if h.seasons != nil {
		catalog := h.seasons.Load(r.Context(), slug)
		if len(catalog) > 0 {
			if resp.Assets == nil {
				resp.Assets = make(map[string]map[string]assetMappingDTO, 1)
			}
			seasonBucket := make(map[string]assetMappingDTO, len(catalog))
			for _, e := range catalog {
				start := e.Start
				seasonBucket[e.ID] = assetMappingDTO{
					Label:        e.Label,
					DisplayOrder: e.DisplayOrder,
					StartDate:    &start,
					EndDate:      e.End,
					Extra:        e.Extra,
				}
			}
			resp.Assets["season"] = seasonBucket
		}
	}

	// Phase 1 plan finition multi-titres : exposer les outcomes s'ils sont chargés.
	if outcomes, ok := h.registry.GetOutcomes(slug); ok && outcomes != nil {
		out := make(map[string]outcomeMappingDTO, len(outcomes.All()))
		for _, o := range outcomes.All() {
			label, _ := o.Label(locale)
			out[o.Key] = outcomeMappingDTO{
				Label:      label,
				ColorToken: o.ColorToken,
			}
		}
		resp.Outcomes = out
	}

	body, err := json.Marshal(resp)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "marshal_failed", err.Error())
		return
	}

	etag := h.etagFor(slug, locale, set.SchemaVersion(), body)
	if match := r.Header.Get("If-None-Match"); match != "" && match == etag {
		w.WriteHeader(http.StatusNotModified)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Cache-Control", "public, max-age=300")
	w.Header().Set("ETag", etag)
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write(body)

	h.logger.Debug("field_mappings_served",
		"title_slug", slug,
		"locale", locale,
		"fields_count", len(resp.Fields),
	)
}

func (h *FieldMappingsHandler) etagFor(slug, locale string, schemaVersion int, body []byte) string {
	cacheKey := fmt.Sprintf("%s|%s|%d", slug, locale, schemaVersion)

	h.mu.RLock()
	if v, ok := h.etagByKey[cacheKey]; ok {
		h.mu.RUnlock()
		return v
	}
	h.mu.RUnlock()

	sum := sha256.Sum256(body)
	etag := `"` + hex.EncodeToString(sum[:8]) + `"` // 8 bytes suffisent pour invalidation

	h.mu.Lock()
	h.etagByKey[cacheKey] = etag
	h.mu.Unlock()
	return etag
}
