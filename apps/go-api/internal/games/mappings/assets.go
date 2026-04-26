package mappings

import "sort"

// AssetMapping est la projection d'une section [assets.{kind}.{id}] du TOML.
//
// Les "assets" couvrent les regroupements UX et les enums sémantiques portés par
// le titre : modes, playlists, tiers de médaille, tiers de défi, cadences,
// statuts de défi. Pas les médailles elles-mêmes (cf. plan §6.7 : restent en DB).
type AssetMapping struct {
	Kind         string            // "mode" | "playlist" | "medal_tier" | "challenge_tier" | "cadence" | "challenge_status"
	ID           string            // identifiant stable au sein du Kind (ex : "ranked", "heroic", "daily")
	Labels       map[string]string // locale → libellé
	ColorToken   string            // token de design system (optionnel — vide si non applicable)
	Icon         string            // chemin SVG/PNG (optionnel)
	DisplayOrder int               // ordre stable dans le Kind
}

// Label retourne le libellé pour la locale demandée + fallback.
//
// Chaîne de fallback : locale → en → ID brut.
func (a AssetMapping) Label(locale string) (label string, usedFallback bool) {
	if v, ok := a.Labels[locale]; ok && v != "" {
		return v, false
	}
	if v, ok := a.Labels[LocaleEN]; ok && v != "" {
		return v, true
	}
	return a.ID, true
}

// AssetMappingSet est l'ensemble des AssetMapping d'un titre, construit par
// le loader et exposé en lecture seule.
type AssetMappingSet struct {
	titleSlug     string
	schemaVersion int
	byKindID      map[string]map[string]AssetMapping // kind → id → mapping
}

// NewAssetMappingSet construit un AssetMappingSet (utilisé par le loader et les
// tests). Les arguments sont consommés tels quels : pas de copie défensive.
func NewAssetMappingSet(titleSlug string, schemaVersion int, byKindID map[string]map[string]AssetMapping) *AssetMappingSet {
	if byKindID == nil {
		byKindID = make(map[string]map[string]AssetMapping)
	}
	return &AssetMappingSet{
		titleSlug:     titleSlug,
		schemaVersion: schemaVersion,
		byKindID:      byKindID,
	}
}

// TitleSlug retourne le slug du titre porteur.
func (s *AssetMappingSet) TitleSlug() string { return s.titleSlug }

// SchemaVersion retourne la version du schéma TOML.
func (s *AssetMappingSet) SchemaVersion() int { return s.schemaVersion }

// Get retourne le mapping d'un asset (kind, id) s'il existe.
func (s *AssetMappingSet) Get(kind, id string) (AssetMapping, bool) {
	if s == nil {
		return AssetMapping{}, false
	}
	bucket, ok := s.byKindID[kind]
	if !ok {
		return AssetMapping{}, false
	}
	a, ok := bucket[id]
	return a, ok
}

// AllOfKind retourne tous les assets d'un kind, triés par display_order puis id.
func (s *AssetMappingSet) AllOfKind(kind string) []AssetMapping {
	if s == nil {
		return nil
	}
	bucket, ok := s.byKindID[kind]
	if !ok {
		return nil
	}
	out := make([]AssetMapping, 0, len(bucket))
	for _, a := range bucket {
		out = append(out, a)
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].DisplayOrder != out[j].DisplayOrder {
			return out[i].DisplayOrder < out[j].DisplayOrder
		}
		return out[i].ID < out[j].ID
	})
	return out
}

// Kinds retourne la liste triée des kinds présents dans cet ensemble.
func (s *AssetMappingSet) Kinds() []string {
	if s == nil {
		return nil
	}
	out := make([]string, 0, len(s.byKindID))
	for k := range s.byKindID {
		out = append(out, k)
	}
	sort.Strings(out)
	return out
}
