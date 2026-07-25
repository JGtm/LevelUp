// Package handlers — admin_titles.go : gestion des titres côté admin (PMT-14 volet A).
//
// GET /api/v1/admin/titles        — liste des titres enregistrés (slug, nom, Status
//
//	lifecycle, capabilities, défaut). Première surface produit qui LIT
//	TitleDescriptor.Status (MT-22 : champ jusqu'ici défini mais jamais exposé).
//
// GET /api/v1/admin/titles/{slug} — détail d'un titre : descripteur + capabilities
//
//	déclarées (capabilities.toml) + feature-matrix calculée. RÉUTILISE la logique
//	1.7a/b (games.ComputeFeatureMatrix) — aucun recalcul divergent.
//
// Read-only. Monté sous le groupe /admin de server.go (RequireAuth + RequireAdmin).
// Title-agnostic : aucune branche sur un slug littéral, tout découle du registre.
//
// MIGRÉ vers Huma (Phase 3b) : Mount crée humacore.NewAPI(r) sur le sous-routeur
// /admin (middleware RequireAuth+RequireAdmin+NoStore hérités) et enregistre les
// 3 GET via huma.Get. Logique métier inchangée (registre titres + capabilities),
// seul le wrapping HTTP change. Le draft TOML reste émis en text/plain via un
// champ Body []byte (passthrough raw Huma) byte-pour-byte identique à l'origine.
package handlers

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"sort"
	"strings"

	"github.com/danielgtaylor/huma/v2"
	"github.com/go-chi/chi/v5"

	"levelup/go-api/internal/api/humacore"
	titlePkg "levelup/go-api/internal/domain/title"
	"levelup/go-api/internal/games"
	"levelup/go-api/internal/games/mappings"
)

// TitleLister expose le registre des titres en lecture seule (sous-ensemble de
// *title.Registry). Interface étroite pour la testabilité (ISP).
type TitleLister interface {
	All() []*titlePkg.TitleDescriptor
	Get(slug string) *titlePkg.TitleDescriptor
}

// AdminTitlesHandler sert la gestion des titres côté admin (read-only).
type AdminTitlesHandler struct {
	titles TitleLister
	caps   CapabilitiesRegistry // réutilisé des endpoints 1.7a/b (cf. capabilities.go)
	logger *slog.Logger
}

// NewAdminTitlesHandler crée le handler en injectant le registre des titres et le
// registre des capabilities (même source que /titles/{slug}/capabilities).
func NewAdminTitlesHandler(titles TitleLister, caps CapabilitiesRegistry, logger *slog.Logger) *AdminTitlesHandler {
	if logger == nil {
		logger = slog.Default()
	}
	return &AdminTitlesHandler{titles: titles, caps: caps, logger: logger}
}

// Mount enregistre les 3 routes via Huma sur le routeur chi `r` (le sous-routeur
// /admin de server.go : middleware RequireAuth+RequireAdmin+NoStore hérités). Le
// chemin relatif EXACT est repris tel quel (/titles, /titles/{slug},
// /titles/{slug}/toml-draft).
func (h *AdminTitlesHandler) Mount(r chi.Router, opts ...humacore.MountOption) {
	api := humacore.NewAPI(r, opts...)
	huma.Get(api, "/titles", h.handleList, humacore.Op("getAdminTitles", "Liste des titres enregistrés (slug, nom, Status lifecycle, capabilities, has_mappings) (auth admin requis)", "admin"))
	huma.Get(api, "/titles/{slug}", h.handleDetail, humacore.Op("getAdminTitleDetail", "Détail d'un titre : descripteur + capabilities déclarées (TOML) + feature-matrix calculée (auth admin requis)", "admin"))
	huma.Get(api, "/titles/{slug}/toml-draft", h.handleTOMLDraft, humacore.Op(
		"getAdminTitleTOMLDraft",
		"Brouillon capabilities.toml collable pour un titre — ZÉRO écriture serveur (D10, auth admin requis)",
		"admin"))
}

// adminTitleSlugInput : path param {slug} commun à Detail et TOMLDraft.
type adminTitleSlugInput struct {
	Slug string `path:"slug"`
}

type adminTitlesListOutput struct{ Body adminTitlesListResponse }
type adminTitleDetailOutput struct{ Body adminTitleDetail }

// adminTOMLDraftOutput émet le draft tel quel (Body []byte → passthrough raw
// Huma, pas de re-marshal JSON) avec son Content-Type text/plain d'origine.
type adminTOMLDraftOutput struct {
	ContentType string `header:"Content-Type"`
	Body        []byte
}

// adminTitleSummary = descripteur du titre + indicateur de mappings TOML chargés.
type adminTitleSummary struct {
	*titlePkg.TitleDescriptor
	HasMappings bool `json:"has_mappings"` // capabilities.toml chargé pour ce titre
}

type adminTitlesListResponse struct {
	Titles []adminTitleSummary `json:"titles"`
	Count  int                 `json:"count"`
}

// handleList répond GET /admin/titles : tous les titres enregistrés, triés par slug.
func (h *AdminTitlesHandler) handleList(ctx context.Context, _ *struct{}) (*adminTitlesListOutput, error) {
	descriptors := h.titles.All()
	out := make([]adminTitleSummary, 0, len(descriptors))
	for _, d := range descriptors {
		_, hasMappings := h.caps.GetCapabilities(d.Slug)
		out = append(out, adminTitleSummary{TitleDescriptor: d, HasMappings: hasMappings})
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Slug < out[j].Slug })

	h.logger.DebugContext(ctx, "admin_titles_list", "count", len(out))
	return &adminTitlesListOutput{Body: adminTitlesListResponse{Titles: out, Count: len(out)}}, nil
}

// adminTitleDetail = descripteur + capabilities déclarées (TOML) + feature-matrix calculée.
type adminTitleDetail struct {
	*titlePkg.TitleDescriptor
	HasMappings          bool              `json:"has_mappings"`
	SchemaVersion        int               `json:"schema_version,omitempty"`
	DeclaredCapabilities map[string]string `json:"declared_capabilities,omitempty"` // capabilities.toml : key → supported|degraded|not_exposed
	FeatureMatrix        map[string]string `json:"feature_matrix,omitempty"`        // cascade calculée : key → available|degraded|unavailable
}

// handleDetail répond GET /admin/titles/{slug}.
func (h *AdminTitlesHandler) handleDetail(ctx context.Context, in *adminTitleSlugInput) (*adminTitleDetailOutput, error) {
	slug := in.Slug
	if slug == "" {
		return nil, humacore.NewError(http.StatusBadRequest, "missing_slug", "title slug requis")
	}
	desc := h.titles.Get(slug)
	if desc == nil {
		return nil, humacore.NewError(http.StatusNotFound, "title_not_found", "titre inconnu : "+slug)
	}

	detail := adminTitleDetail{TitleDescriptor: desc}
	if set, ok := h.caps.GetCapabilities(slug); ok {
		detail.HasMappings = true
		detail.SchemaVersion = set.SchemaVersion()
		detail.DeclaredCapabilities = set.All()
		if cm, err := games.CapabilityMapFromMappings(set); err != nil {
			// capabilities.toml déclare une capability hors vocabulaire produit →
			// feature-matrix dégradée (omise), pas d'erreur 500 : l'admin voit
			// quand même le titre + ses capabilities déclarées brutes.
			h.logger.WarnContext(ctx, "admin_titles_feature_matrix_degraded", "title", slug, "err", err)
		} else {
			matrix := games.ComputeFeatureMatrix(cm)
			fm := make(map[string]string, len(matrix))
			for k, v := range matrix {
				fm[string(k)] = string(v)
			}
			detail.FeatureMatrix = fm
		}
	}

	h.logger.DebugContext(ctx, "admin_titles_detail", "title", slug, "has_mappings", detail.HasMappings)
	return &adminTitleDetailOutput{Body: detail}, nil
}

// TOMLDraft répond GET /admin/titles/{slug}/toml-draft : un brouillon
// capabilities.toml collable pour le titre, généré depuis les capabilities
// déclarées (si présentes) ou un template des CapabilityKey canoniques.
//
// Décision D10 : AUCUNE écriture serveur — la string est renvoyée telle quelle
// (text/plain) ; le front copie via navigator.clipboard. Ne jamais écrire sur
// disque ici (garde-fou lint : aucune écriture fichier dans ce handler, cf.
// admin_titles_test.go).
func (h *AdminTitlesHandler) handleTOMLDraft(ctx context.Context, in *adminTitleSlugInput) (*adminTOMLDraftOutput, error) {
	slug := in.Slug
	if slug == "" {
		return nil, humacore.NewError(http.StatusBadRequest, "missing_slug", "title slug requis")
	}
	if h.titles.Get(slug) == nil {
		return nil, humacore.NewError(http.StatusNotFound, "title_not_found", "titre inconnu : "+slug)
	}

	set, ok := h.caps.GetCapabilities(slug)
	if !ok {
		set = nil // dégradation : titre sans capabilities.toml → template par défaut
	}
	draft := buildCapabilitiesDraft(slug, set)

	h.logger.InfoContext(ctx, "admin_titles_toml_draft", "title", slug)
	return &adminTOMLDraftOutput{
		ContentType: "text/plain; charset=utf-8",
		Body:        []byte(draft),
	}, nil
}

// buildCapabilitiesDraft génère un capabilities.toml brouillon pour `slug`.
// Si `set` est non nil, ses valeurs déclarées sont reprises ; sinon chaque
// CapabilityKey canonique est mise à "not_exposed" (l'admin ajuste).
func buildCapabilitiesDraft(slug string, set *mappings.CapabilityMappingSet) string {
	declared := map[string]string{}
	schema := 1
	if set != nil {
		declared = set.All()
		if sv := set.SchemaVersion(); sv > 0 {
			schema = sv
		}
	}

	var b strings.Builder
	fmt.Fprintf(&b, "# Draft capabilities.toml pour le titre %q.\n", slug)
	b.WriteString("# Genere par GET /admin/titles/{slug}/toml-draft — AUCUNE ecriture serveur (D10).\n")
	fmt.Fprintf(&b, "# Copiez ce bloc dans config/titles/%s/mappings/capabilities.toml puis ajustez.\n", slug)
	b.WriteString("# Valeurs admises : \"supported\" | \"degraded\" | \"not_exposed\".\n\n")
	b.WriteString("[meta]\n")
	fmt.Fprintf(&b, "title_slug     = %q\n", slug)
	fmt.Fprintf(&b, "schema_version = %d\n\n", schema)
	b.WriteString("[capabilities]\n")
	for _, k := range games.AllCapabilityKeys() {
		val := declared[string(k)]
		if val == "" {
			val = "not_exposed"
		}
		fmt.Fprintf(&b, "%q = %q\n", string(k), val)
	}
	b.WriteString("\n# [data] : sources de donnees par capability (optionnel, ajoutez selon le titre).\n")
	return b.String()
}
