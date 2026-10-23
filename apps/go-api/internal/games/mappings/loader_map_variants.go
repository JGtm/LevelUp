package mappings

import (
	"fmt"
	"os"
	"strings"

	"github.com/pelletier/go-toml/v2"
)

// MapVariantSet porte les suffixes de nom qui font d'une carte la variante d'une autre
// (config/titles/{slug}/mappings/map_variants.toml). La règle de rattachement est dans
// analysis.BaseMapName ; ce type ne fait que porter les données du titre.
type MapVariantSet struct {
	titleSlug     string
	schemaVersion int
	suffixes      []string
}

// mapVariantsTOML : projection brute de map_variants.toml.
type mapVariantsTOML struct {
	Meta     metaSection `toml:"meta"`
	Variants struct {
		Suffixes []string `toml:"suffixes"`
	} `toml:"variants"`
}

// Suffixes rend une copie des suffixes déclarés. nil-safe : un set nil n'en déclare aucun.
func (s *MapVariantSet) Suffixes() []string {
	if s == nil {
		return nil
	}
	return append([]string(nil), s.suffixes...)
}

// SchemaVersion retourne la version de schéma déclarée.
func (s *MapVariantSet) SchemaVersion() int {
	if s == nil {
		return 0
	}
	return s.schemaVersion
}

// LoadMapVariantsFromFile lit et valide un map_variants.toml à un chemin donné.
func LoadMapVariantsFromFile(path string) (*MapVariantSet, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read %s: %w", path, err)
	}
	return LoadMapVariantsFromBytes(path, raw)
}

// LoadMapVariantsFromBytes parse et valide un payload TOML déjà en mémoire.
func LoadMapVariantsFromBytes(path string, raw []byte) (*MapVariantSet, error) {
	var doc mapVariantsTOML
	if err := toml.Unmarshal(raw, &doc); err != nil {
		return nil, fmt.Errorf("parse %s: %w", path, err)
	}
	if doc.Meta.TitleSlug == "" {
		return nil, fmt.Errorf("%s: [meta].title_slug manquant", path)
	}
	if doc.Meta.SchemaVersion <= 0 {
		return nil, fmt.Errorf("%s: [meta].schema_version doit être > 0 (reçu %d)", path, doc.Meta.SchemaVersion)
	}
	suffixes := make([]string, 0, len(doc.Variants.Suffixes))
	for _, s := range doc.Variants.Suffixes {
		s = strings.TrimSpace(s)
		if s == "" {
			return nil, fmt.Errorf("%s: [variants].suffixes contient un suffixe vide", path)
		}
		suffixes = append(suffixes, s)
	}
	return &MapVariantSet{
		titleSlug:     doc.Meta.TitleSlug,
		schemaVersion: doc.Meta.SchemaVersion,
		suffixes:      suffixes,
	}, nil
}
