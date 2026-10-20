// Package handlers — field_mappings_cache.go : cache du DTO field-mappings et
// clé de version (plan perf 2026-09-23, lot L5b, décision D5b.2).
//
// Version d'un titre = empreinte du CONTENU de ses trois TOML (champs, assets,
// issues) + empreinte de son catalogue de saisons. Un DTO caché n'est servi que
// s'il est à la version courante : un TOML ou une saison qui change produit un
// nouveau corps, donc un nouvel ETag (hash du corps, cf. buildDTO).
package handlers

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"sort"
	"strconv"
	"strings"
	"time"

	"levelup/go-api/internal/games/mappings"
)

// fieldMappingsCacheEntry est un DTO construit, valable tant que sa version est la
// version courante du titre.
type fieldMappingsCacheEntry struct {
	version     string
	body        []byte
	etag        string
	fieldsCount int
}

// titleMappingSets regroupe les trois jeux TOML d'un titre. Les sets sont
// immuables après chargement : leur identité suffit à mémoriser leur empreinte.
type titleMappingSets struct {
	fields   *mappings.FieldMappingSet
	assets   *mappings.AssetMappingSet
	outcomes *mappings.OutcomeMappingSet
}

// cachedDTO rend l'entrée cachée si elle est à la version courante.
func (h *FieldMappingsHandler) cachedDTO(key, version string) (fieldMappingsCacheEntry, bool) {
	h.mu.RLock()
	defer h.mu.RUnlock()
	entry, ok := h.dtoByKey[key]
	if !ok || entry.version != version {
		return fieldMappingsCacheEntry{}, false
	}
	return entry, true
}

func (h *FieldMappingsHandler) storeDTO(key string, entry fieldMappingsCacheEntry) {
	h.mu.Lock()
	h.dtoByKey[key] = entry
	h.mu.Unlock()
}

// tomlFingerprint rend l'empreinte du CONTENU des trois jeux TOML du titre
// (champs, assets, issues, toutes locales). Calculée une fois par jeu de sets
// chargés, puis mémorisée : les sets sont immuables, un rechargement en produit
// de nouveaux (nouvelle identité, nouvelle empreinte).
func (h *FieldMappingsHandler) tomlFingerprint(ctx context.Context, sets titleMappingSets) string {
	h.mu.RLock()
	fp, ok := h.tomlPrints[sets]
	h.mu.RUnlock()
	if ok {
		return fp
	}
	fp, err := computeTOMLFingerprint(sets)
	if err != nil {
		// Repli sûr : une empreinte par identité des sets, qui change avec eux.
		h.logger.ErrorContext(ctx, "field_mappings: empreinte TOML impossible — repli sur les pointeurs des sets",
			"title_slug", sets.fields.TitleSlug(), "err", err)
		fp = fmt.Sprintf("ptr:%p:%p:%p", sets.fields, sets.assets, sets.outcomes)
	}
	h.mu.Lock()
	h.tomlPrints[sets] = fp
	h.mu.Unlock()
	return fp
}

// computeTOMLFingerprint hache une projection déterministe des trois jeux (les
// listes sont triées par les sets, les maps par encoding/json).
func computeTOMLFingerprint(sets titleMappingSets) (string, error) {
	projection := struct {
		FieldsSchema   int
		Fields         []mappings.FieldMapping
		AssetsSchema   int
		Assets         map[string][]mappings.AssetMapping
		OutcomesSchema int
		Outcomes       []mappings.OutcomeMapping
	}{
		FieldsSchema: sets.fields.SchemaVersion(),
		Fields:       sets.fields.All(),
		Assets:       map[string][]mappings.AssetMapping{},
		Outcomes:     sets.outcomes.All(),
	}
	if sets.assets != nil {
		projection.AssetsSchema = sets.assets.SchemaVersion()
		for _, kind := range sets.assets.Kinds() {
			projection.Assets[kind] = sets.assets.AllOfKind(kind)
		}
	}
	if sets.outcomes != nil {
		projection.OutcomesSchema = sets.outcomes.SchemaVersion()
	}
	raw, err := json.Marshal(projection)
	if err != nil {
		return "", err
	}
	sum := sha256.Sum256(raw)
	return hex.EncodeToString(sum[:12]), nil
}

// seasonsCatalogVersion rend l'empreinte du catalogue de saisons : toute saison
// ajoutée, retirée ou modifiée (dates, libellés, ordre, extras) la change.
func seasonsCatalogVersion(catalog []SeasonCatalogEntry) string {
	var b strings.Builder
	for _, e := range catalog {
		b.WriteString(e.ID + "\x1f" + e.Label + "\x1f" + e.LabelEN + "\x1f" + e.Start.UTC().Format(time.RFC3339Nano) + "\x1f")
		if e.End != nil {
			b.WriteString(e.End.UTC().Format(time.RFC3339Nano))
		}
		b.WriteString("\x1f" + strconv.Itoa(e.DisplayOrder))
		keys := make([]string, 0, len(e.Extra))
		for k := range e.Extra {
			keys = append(keys, k)
		}
		sort.Strings(keys)
		for _, k := range keys {
			b.WriteString("\x1f" + k + "=" + e.Extra[k])
		}
		b.WriteString("\x1e")
	}
	sum := sha256.Sum256([]byte(b.String()))
	return hex.EncodeToString(sum[:12])
}
