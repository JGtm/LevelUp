package mappings

import (
	"sort"

	"levelup/go-api/internal/games/canonical"
	"levelup/go-api/internal/i18n"
)

// FieldMapping est la projection d'une section [fields.X] du TOML.
type FieldMapping struct {
	Key          canonical.FieldKey
	Labels       i18n.Text
	Descriptions i18n.Text
	StorageUnit  Unit
	DisplayUnit  Unit
	Format       Format
	DisplayOrder int
	Group        string
	Icon         string
}

// Label retourne le libellé pour la locale demandée et un bool indiquant si
// un fallback a été utilisé (pour le logging field_lookup_missing rate-limité).
//
// Chaîne de fallback : locale demandée → i18n.Fallback → key as string.
func (m FieldMapping) Label(locale string) (label string, usedFallback bool) {
	v, fell := m.Labels.Resolve(i18n.Locale(locale))
	if v == "" {
		return string(m.Key), true
	}
	return v, fell
}

// Description retourne la description pour la locale demandée + fallback.
func (m FieldMapping) Description(locale string) (string, bool) {
	v, fell := m.Descriptions.Resolve(i18n.Locale(locale))
	if v == "" {
		return "", true
	}
	return v, fell
}

// FieldMappingSet est l'ensemble des FieldMapping d'un titre, construit par
// le loader et exposé en lecture seule.
type FieldMappingSet struct {
	titleSlug     string
	schemaVersion int
	byKey         map[canonical.FieldKey]FieldMapping
}

// TitleSlug retourne le slug du titre porteur de cet ensemble.
func (s *FieldMappingSet) TitleSlug() string { return s.titleSlug }

// SchemaVersion retourne la version du schéma TOML.
func (s *FieldMappingSet) SchemaVersion() int { return s.schemaVersion }

// Get retourne le mapping d'un FieldKey s'il existe.
func (s *FieldMappingSet) Get(key canonical.FieldKey) (FieldMapping, bool) {
	m, ok := s.byKey[key]
	return m, ok
}

// All retourne tous les mappings dans l'ordre stable des FieldKey canoniques.
func (s *FieldMappingSet) All() []FieldMapping {
	out := make([]FieldMapping, 0, len(s.byKey))
	for _, k := range canonical.AllFieldKeys() {
		if m, ok := s.byKey[k]; ok {
			out = append(out, m)
		}
	}
	return out
}

// OrderedByGroup retourne les mappings groupés par 'group' et triés par
// display_order au sein de chaque groupe.
func (s *FieldMappingSet) OrderedByGroup() map[string][]FieldMapping {
	groups := make(map[string][]FieldMapping)
	for _, m := range s.byKey {
		groups[m.Group] = append(groups[m.Group], m)
	}
	for g, list := range groups {
		sort.Slice(list, func(i, j int) bool {
			return list[i].DisplayOrder < list[j].DisplayOrder
		})
		groups[g] = list
	}
	return groups
}

// Keys retourne les FieldKey présents dans cet ensemble (set d'appartenance).
func (s *FieldMappingSet) Keys() []canonical.FieldKey {
	out := make([]canonical.FieldKey, 0, len(s.byKey))
	for k := range s.byKey {
		out = append(out, k)
	}
	sort.Slice(out, func(i, j int) bool { return out[i] < out[j] })
	return out
}
