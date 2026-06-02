package mappings

import "sort"

// Statuts de capability admis dans capabilities.toml (Phase 1.7a).
//
// Le package mappings reste title-agnostic : il valide le STATUT (vocabulaire
// stable) mais PAS les clés de capability (vocabulaire produit, propriété du
// package games qui convertit le set en games.CapabilityMap).
const (
	CapStatusSupported  = "supported"
	CapStatusDegraded   = "degraded"
	CapStatusNotExposed = "not_exposed"
)

// ValidCapStatus indique si un statut de capability est admis.
func ValidCapStatus(status string) bool {
	switch status {
	case CapStatusSupported, CapStatusDegraded, CapStatusNotExposed:
		return true
	}
	return false
}

// CapabilityMappingSet est l'ensemble des capabilities déclarées par un titre
// (projection de capabilities.toml). Clé = CapabilityKey canonique (string),
// valeur = statut. Title-agnostic : ne connaît pas le vocabulaire des clés.
type CapabilityMappingSet struct {
	titleSlug     string
	schemaVersion int
	byKey         map[string]string // capabilityKey → statut
}

// NewCapabilityMappingSet construit un CapabilityMappingSet (loader + tests).
func NewCapabilityMappingSet(titleSlug string, schemaVersion int, byKey map[string]string) *CapabilityMappingSet {
	if byKey == nil {
		byKey = make(map[string]string)
	}
	return &CapabilityMappingSet{
		titleSlug:     titleSlug,
		schemaVersion: schemaVersion,
		byKey:         byKey,
	}
}

// TitleSlug retourne le slug du titre porteur.
func (s *CapabilityMappingSet) TitleSlug() string { return s.titleSlug }

// SchemaVersion retourne la version du schéma TOML.
func (s *CapabilityMappingSet) SchemaVersion() int { return s.schemaVersion }

// Status retourne le statut d'une capability si déclarée.
func (s *CapabilityMappingSet) Status(key string) (string, bool) {
	if s == nil {
		return "", false
	}
	v, ok := s.byKey[key]
	return v, ok
}

// All retourne le mapping clé→statut (copie défensive, ordre stable non garanti
// sur la map mais Keys() fournit l'ordre trié).
func (s *CapabilityMappingSet) All() map[string]string {
	if s == nil {
		return nil
	}
	out := make(map[string]string, len(s.byKey))
	for k, v := range s.byKey {
		out[k] = v
	}
	return out
}

// Keys retourne les clés de capability triées (déterminisme tests/diff).
func (s *CapabilityMappingSet) Keys() []string {
	if s == nil {
		return nil
	}
	out := make([]string, 0, len(s.byKey))
	for k := range s.byKey {
		out = append(out, k)
	}
	sort.Strings(out)
	return out
}
