package mappings

import (
	"fmt"
	"os"
	"strings"

	"github.com/pelletier/go-toml/v2"
)

// PlaylistLabelSet porte les overrides d'affichage du libellé de playlist d'un
// titre (data-driven, config/titles/{slug}/mappings/playlist_labels.toml).
//
// Motivation (Halo 5) : l'API sert des noms « nom + catégorie » accolés (ex.
// « Super Fiesta Fête ») ; le strip de préfixe Infinite (NormalizePlaylistLabel)
// mangeait le mauvais mot (« Super Fiesta » est un préfixe de catégorie → il ne
// restait que « Fête »). On mappe donc explicitement le nom BRUT vers le nom
// court d'affichage. Zéro heuristique : seules les entrées listées sont
// transformées (aucun risque de raccourcir une playlist inconnue).
type PlaylistLabelSet struct {
	titleSlug     string
	schemaVersion int
	// overrides : nom brut résolu (asset_translations) → libellé court affiché.
	overrides map[string]string
}

// playlistLabelsTOML — projection brute de playlist_labels.toml.
type playlistLabelsTOML struct {
	Meta      metaSection       `toml:"meta"`
	Overrides map[string]string `toml:"overrides"`
}

// Display retourne le libellé d'affichage pour un nom brut : l'override s'il
// existe, sinon le nom inchangé. Un set nil est un no-op (retourne raw).
func (s *PlaylistLabelSet) Display(raw string) string {
	if s == nil {
		return raw
	}
	if v, ok := s.overrides[raw]; ok {
		return v
	}
	return raw
}

// OverridesMap retourne une copie de la table d'overrides (nom brut → libellé).
// nil-safe (retourne un map vide). Utilisé par le wiring pour injecter la table
// dans les repos sans exposer le type interne.
func (s *PlaylistLabelSet) OverridesMap() map[string]string {
	out := make(map[string]string)
	if s == nil {
		return out
	}
	for k, v := range s.overrides {
		out[k] = v
	}
	return out
}

// SchemaVersion retourne la version de schéma déclarée.
func (s *PlaylistLabelSet) SchemaVersion() int {
	if s == nil {
		return 0
	}
	return s.schemaVersion
}

// LoadPlaylistLabelsFromFile lit et valide un playlist_labels.toml à un chemin donné.
func LoadPlaylistLabelsFromFile(path string) (*PlaylistLabelSet, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read %s: %w", path, err)
	}
	return LoadPlaylistLabelsFromBytes(path, raw)
}

// LoadPlaylistLabelsFromBytes parse et valide un payload TOML déjà en mémoire.
func LoadPlaylistLabelsFromBytes(path string, raw []byte) (*PlaylistLabelSet, error) {
	var doc playlistLabelsTOML
	if err := toml.Unmarshal(raw, &doc); err != nil {
		return nil, fmt.Errorf("parse %s: %w", path, err)
	}
	if doc.Meta.TitleSlug == "" {
		return nil, fmt.Errorf("%s: [meta].title_slug manquant", path)
	}
	if doc.Meta.SchemaVersion <= 0 {
		return nil, fmt.Errorf("%s: [meta].schema_version doit être > 0 (reçu %d)", path, doc.Meta.SchemaVersion)
	}
	overrides := make(map[string]string, len(doc.Overrides))
	for rawName, display := range doc.Overrides {
		key := strings.TrimSpace(rawName)
		val := strings.TrimSpace(display)
		if key == "" || val == "" {
			return nil, fmt.Errorf("%s: override vide (clé=%q valeur=%q)", path, rawName, display)
		}
		overrides[key] = val
	}
	return &PlaylistLabelSet{
		titleSlug:     doc.Meta.TitleSlug,
		schemaVersion: doc.Meta.SchemaVersion,
		overrides:     overrides,
	}, nil
}
