package assets

import "fmt"

// Ref est la clé canonique qui identifie un asset de façon unique.
// Elle sert de clé singleflight, de clé de cache FS/DuckDB et de logging.
type Ref struct {
	// Kind identifie le type d'asset.
	Kind Kind
	// TitleID est l'identifiant du titre Halo (ex: "halo_infinite", "HaloInfinite").
	TitleID string
	// ID est l'identifiant spécifique à ce Kind :
	//   medal-image    → strconv de int64 medal_id
	//   map-image      → map_id
	//   challenge-badge → slug dérivé du challenge path
	//   bp-track-image / bp-background → reward_track_path
	//   medal-meta     → "metadata" (singleton)
	//   challenge-def  → challenge_path
	//   track-def      → reward_track_path
	//   asset-translation → asset_id
	ID string
	// Variant est un qualificatif optionnel (ex: "spritesheet" pour medal-image,
	// "background" pour bp, asset_type pour asset-translation).
	Variant string
	// Lang est le code BCP-47 pour KindAssetTranslation (ex: "fr-FR", "en-US").
	// Vide pour les kinds non-localisés.
	Lang string
}

// String retourne la clé canonique unique de la référence.
// Format : "{kind}/{titleID}/{id}[/{variant}][/{lang}]"
// Utilisée comme clé singleflight et dans les logs structurés.
func (r Ref) String() string {
	s := fmt.Sprintf("%s/%s/%s", r.Kind, r.TitleID, r.ID)
	if r.Variant != "" {
		s += "/" + r.Variant
	}
	if r.Lang != "" {
		s += "/" + r.Lang
	}
	return s
}

// LogAttrs retourne les attributs slog pour une Ref.
func (r Ref) LogAttrs() []any {
	attrs := []any{"kind", string(r.Kind), "title", r.TitleID, "id", r.ID}
	if r.Variant != "" {
		attrs = append(attrs, "variant", r.Variant)
	}
	if r.Lang != "" {
		attrs = append(attrs, "lang", r.Lang)
	}
	return attrs
}

// CacheKey retourne la clé utilisée pour le stockage FS/DuckDB.
// Identique à String() mais peut être surchargée par le caller si besoin.
func (r Ref) CacheKey() string { return r.String() }
