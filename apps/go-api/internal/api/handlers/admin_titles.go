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
package handlers

import (
	"fmt"
	"log/slog"
	"net/http"
	"sort"
	"strings"

	"github.com/go-chi/chi/v5"

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

// adminTitleSummary = descripteur du titre + indicateur de mappings TOML chargés.
type adminTitleSummary struct {
	*titlePkg.TitleDescriptor
	HasMappings bool `json:"has_mappings"` // capabilities.toml chargé pour ce titre
}

type adminTitlesListResponse struct {
	Titles []adminTitleSummary `json:"titles"`
	Count  int                 `json:"count"`
}

// List répond GET /admin/titles : tous les titres enregistrés, triés par slug.
func (h *AdminTitlesHandler) List(w http.ResponseWriter, r *http.Request) {
	descriptors := h.titles.All()
	out := make([]adminTitleSummary, 0, len(descriptors))
	for _, d := range descriptors {
		_, hasMappings := h.caps.GetCapabilities(d.Slug)
		out = append(out, adminTitleSummary{TitleDescriptor: d, HasMappings: hasMappings})
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Slug < out[j].Slug })

	writeJSON(w, http.StatusOK, adminTitlesListResponse{Titles: out, Count: len(out)})
	h.logger.DebugContext(r.Context(), "admin_titles_list", "count", len(out))
}

// adminTitleDetail = descripteur + capabilities déclarées (TOML) + feature-matrix calculée.
type adminTitleDetail struct {
	*titlePkg.TitleDescriptor
	HasMappings          bool              `json:"has_mappings"`
	SchemaVersion        int               `json:"schema_version,omitempty"`
	DeclaredCapabilities map[string]string `json:"declared_capabilities,omitempty"` // capabilities.toml : key → supported|degraded|not_exposed
	FeatureMatrix        map[string]string `json:"feature_matrix,omitempty"`        // cascade calculée : key → available|degraded|unavailable
}

// Detail répond GET /admin/titles/{slug}.
func (h *AdminTitlesHandler) Detail(w http.ResponseWriter, r *http.Request) {
	slug := chi.URLParam(r, "slug")
	if slug == "" {
		writeError(r.Context(), w, http.StatusBadRequest, "missing_slug", "title slug requis")
		return
	}
	desc := h.titles.Get(slug)
	if desc == nil {
		writeError(r.Context(), w, http.StatusNotFound, "title_not_found", "titre inconnu : "+slug)
		return
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
			h.logger.WarnContext(r.Context(), "admin_titles_feature_matrix_degraded", "title", slug, "err", err)
		} else {
			matrix := games.ComputeFeatureMatrix(cm)
			fm := make(map[string]string, len(matrix))
			for k, v := range matrix {
				fm[string(k)] = string(v)
			}
			detail.FeatureMatrix = fm
		}
	}

	writeJSON(w, http.StatusOK, detail)
	h.logger.DebugContext(r.Context(), "admin_titles_detail", "title", slug, "has_mappings", detail.HasMappings)
}

// TOMLDraft répond GET /admin/titles/{slug}/toml-draft : un brouillon
// capabilities.toml collable pour le titre, généré depuis les capabilities
// déclarées (si présentes) ou un template des CapabilityKey canoniques.
//
// Décision D10 : AUCUNE écriture serveur — la string est renvoyée telle quelle
// (text/plain) ; le front copie via navigator.clipboard. Ne jamais écrire sur
// disque ici (garde-fou lint : aucune écriture fichier dans ce handler, cf.
// admin_titles_test.go).
func (h *AdminTitlesHandler) TOMLDraft(w http.ResponseWriter, r *http.Request) {
	slug := chi.URLParam(r, "slug")
	if slug == "" {
		writeError(r.Context(), w, http.StatusBadRequest, "missing_slug", "title slug requis")
		return
	}
	if h.titles.Get(slug) == nil {
		writeError(r.Context(), w, http.StatusNotFound, "title_not_found", "titre inconnu : "+slug)
		return
	}

	set, ok := h.caps.GetCapabilities(slug)
	if !ok {
		set = nil // dégradation : titre sans capabilities.toml → template par défaut
	}
	draft := buildCapabilitiesDraft(slug, set)

	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte(draft))
	h.logger.InfoContext(r.Context(), "admin_titles_toml_draft", "title", slug)
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
