// Package handlers — handler GET /api/v1/titles/{slug}/field-mappings.
//
// Phase A du plan multi-titres : endpoint d'exposition des libellés et
// présentation des FieldKey canoniques par titre. Lit les TOML chargés au
// boot par le FieldMappingRegistry.
//
// MIGRÉ vers Huma (Phase 3b) : Mount crée humacore.NewAPI(r) et enregistre
// l'unique GET via huma.Get. Le corps JSON est marshalé par le handler lui-même
// puis émis tel quel via un champ Body []byte (passthrough raw de Huma) — ce qui
// préserve byte-pour-byte le corps d'origine (json.Marshal HTML-escapé, sans
// trailing newline) sur lequel l'ETag est calculé, ainsi que les chemins 304
// (If-None-Match), Cache-Control et ETag.
package handlers

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"sync"
	"time"

	"github.com/danielgtaylor/huma/v2"
	"github.com/go-chi/chi/v5"

	"levelup/go-api/internal/api/humacore"
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
	ID    string
	Label string
	// LabelEN = libellé EN (GH3-1). Vide → Label sert dans les deux locales.
	LabelEN      string
	Start        time.Time
	End          *time.Time
	DisplayOrder int
	Extra        map[string]string
}

// FieldMappingsHandler gère GET /api/v1/titles/{slug}/field-mappings.
//
// Le handler est enregistré INCONDITIONNELLEMENT depuis le 2026-08-02 (v7.3 lot 2,
// item 3.3) : le gate de rollout MULTI_TITLE_API_ENABLED a été retiré, et les
// libellés servis ici sont la source unique du front (plus aucun dictionnaire de
// repli côté TS). Garde-rail : internal/api/multi_title_smoke_test.go.
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

// Mount enregistre l'unique GET via Huma. NewAPI(r) peut recevoir le routeur
// racine OU un sous-routeur ; le chemin relatif exact est repris tel quel.
func (h *FieldMappingsHandler) Mount(r chi.Router, opts ...humacore.MountOption) {
	api := humacore.NewAPI(r, opts...)
	huma.Get(api, "/titles/{slug}/field-mappings", h.handleGet, humacore.Op("getTitleFieldMappings", "Field mappings TOML d'un titre", "titles"))
}

// fieldMappingsInput : {slug} path + ?locale= optionnel + If-None-Match (ETag
// conditionnel). Le slug est validé côté handler pour reproduire le 400
// missing_slug d'origine (chi route un slug vide rarement, mais le contrat le
// couvrait explicitement).
type fieldMappingsInput struct {
	Slug        string `path:"slug"`
	Locale      string `query:"locale"`
	IfNoneMatch string `header:"If-None-Match"`
}

// fieldMappingsOutput émet le corps marshalé par le handler tel quel (Body
// []byte → passthrough raw Huma, pas de re-marshal ni de trailing newline). Les
// trois en-têtes (chaînes vides non émises côté Huma) restent vides sur le
// chemin 304 pour reproduire l'absence d'en-têtes de l'ancien handler.
type fieldMappingsOutput struct {
	Status       int
	ContentType  string `header:"Content-Type"`
	CacheControl string `header:"Cache-Control"`
	ETag         string `header:"ETag"`
	Body         []byte
}

// handleGet gère la requête (logique métier inchangée : marshal maison, ETag,
// 304 conditionnel).
func (h *FieldMappingsHandler) handleGet(ctx context.Context, in *fieldMappingsInput) (*fieldMappingsOutput, error) {
	slug := in.Slug
	if slug == "" {
		return nil, humacore.NewError(http.StatusBadRequest, "missing_slug", "title slug requis")
	}

	// Locale inconnue → la fallback EN est gérée par FieldMapping.Label.
	// On conserve la valeur d'origine dans la réponse pour que le frontend
	// sache ce qu'il a demandé.
	locale := in.Locale
	if locale == "" {
		locale = mappings.LocaleFR
	}

	set, ok := h.registry.Get(slug)
	if !ok {
		return nil, humacore.NewError(http.StatusNotFound, "title_not_found",
			fmt.Sprintf("title %q n'a pas de field mappings chargés", slug))
	}

	resp := fieldMappingsResponse{
		TitleSlug:     set.TitleSlug(),
		SchemaVersion: set.SchemaVersion(),
		Locale:        locale,
		Fields:        h.buildFieldsDTO(set, locale),
		Assets:        h.buildAssetsDTO(ctx, slug, locale),
		Outcomes:      h.buildOutcomesDTO(slug, locale),
	}

	body, err := json.Marshal(resp)
	if err != nil {
		return nil, humacore.NewError(http.StatusInternalServerError, "marshal_failed", err.Error())
	}

	etag := h.etagFor(slug, locale, set.SchemaVersion(), body)
	if in.IfNoneMatch != "" && in.IfNoneMatch == etag {
		return &fieldMappingsOutput{Status: http.StatusNotModified}, nil
	}

	h.logger.Debug("field_mappings_served",
		"title_slug", slug,
		"locale", locale,
		"fields_count", len(resp.Fields),
	)

	return &fieldMappingsOutput{
		Status:       http.StatusOK,
		ContentType:  "application/json",
		CacheControl: "public, max-age=300",
		ETag:         etag,
		Body:         body,
	}, nil
}

// buildFieldsDTO projette les FieldMapping du set en DTO localisés.
// Loggue chaque fallback de locale pour faciliter le diagnostic des trad
// manquantes.
func (h *FieldMappingsHandler) buildFieldsDTO(set *mappings.FieldMappingSet, locale string) map[string]fieldMappingDTO {
	out := make(map[string]fieldMappingDTO, len(set.All()))
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
		out[string(m.Key)] = fieldMappingDTO{
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
	return out
}

// buildAssetsDTO construit le bucket assets du DTO.
//
// Trois sources potentielles fusionnées :
//   - registry TOML (toujours, si chargé) → tous les kinds (mode, map, season…)
//   - SeasonsCatalogResolver (V2 saisons, optionnel) → remplace le bucket
//     "season" par l'union TOML+DB+lazy-fetch live
//
// Retourne nil si aucune source ne fournit d'assets (omitempty côté JSON).
func (h *FieldMappingsHandler) buildAssetsDTO(ctx context.Context, slug, locale string) map[string]map[string]assetMappingDTO {
	var out map[string]map[string]assetMappingDTO

	if assets, ok := h.registry.GetAssets(slug); ok && assets != nil {
		out = make(map[string]map[string]assetMappingDTO, len(assets.Kinds()))
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
	}

	// V2 saisons : si le SeasonsCatalogResolver est câblé, on remplace
	// purement le bucket "season" du DTO par le résultat du resolver — qui
	// fait l'union TOML + DB + éventuel lazy fetch live (avec persistance).
	// Cela permet à une nouvelle Operation Halo découverte en DB d'apparaître
	// automatiquement dans la SaisonPill côté frontend, sans intervention
	// manuelle sur le TOML. Les saisons DB-only (pas encore de FR) sont
	// affichées avec leur libellé Waypoint brut.
	if h.seasons != nil {
		catalog := h.seasons.Load(ctx, slug)
		if len(catalog) > 0 {
			if out == nil {
				out = make(map[string]map[string]assetMappingDTO, 1)
			}
			out["season"] = projectCatalogToBucket(catalog, locale)
		}
	}

	return out
}

// projectCatalogToBucket projette le catalog unifié saisons en bucket DTO,
// résolu dans la locale de requête (GH3-1). Sous EN, le libellé EN prime
// (LabelEN) ; sinon le libellé FR/canonique (Label). Les saisons DB-only sans
// traduction portent le même Name dans les deux locales (LabelEN == Label).
func projectCatalogToBucket(catalog []SeasonCatalogEntry, locale string) map[string]assetMappingDTO {
	bucket := make(map[string]assetMappingDTO, len(catalog))
	for _, e := range catalog {
		start := e.Start
		label := e.Label
		if locale == mappings.LocaleEN && e.LabelEN != "" {
			label = e.LabelEN
		}
		bucket[e.ID] = assetMappingDTO{
			Label:        label,
			DisplayOrder: e.DisplayOrder,
			StartDate:    &start,
			EndDate:      e.End,
			Extra:        e.Extra,
		}
	}
	return bucket
}

// buildOutcomesDTO projette les OutcomeMapping en DTO localisés.
// Retourne nil si le set d'outcomes n'est pas chargé pour ce titre.
func (h *FieldMappingsHandler) buildOutcomesDTO(slug, locale string) map[string]outcomeMappingDTO {
	outcomes, ok := h.registry.GetOutcomes(slug)
	if !ok || outcomes == nil {
		return nil
	}
	out := make(map[string]outcomeMappingDTO, len(outcomes.All()))
	for _, o := range outcomes.All() {
		label, _ := o.Label(locale)
		out[o.Key] = outcomeMappingDTO{
			Label:      label,
			ColorToken: o.ColorToken,
		}
	}
	return out
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
