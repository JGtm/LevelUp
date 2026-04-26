package mappings

import (
	"errors"
	"fmt"
	"os"
	"strings"

	"github.com/pelletier/go-toml/v2"
)

// assetsTOML est la projection brute du fichier assets.toml avant validation.
type assetsTOML struct {
	Meta   metaSection                                 `toml:"meta"`
	Assets map[string]map[string]assetEntryTOML        `toml:"assets"` // kind → id → entry
	_      struct{}                                    `toml:"-"`
}

type assetEntryTOML struct {
	Labels       map[string]string `toml:"labels"`
	ColorToken   string            `toml:"color_token"`
	Icon         string            `toml:"icon"`
	DisplayOrder int               `toml:"display_order"`
}

// LoadAssetsFromFile lit et valide un assets.toml à un chemin donné.
func LoadAssetsFromFile(path string) (*AssetMappingSet, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read %s: %w", path, err)
	}
	return LoadAssetsFromBytes(path, raw)
}

// LoadAssetsFromBytes parse et valide un payload TOML déjà chargé en mémoire.
func LoadAssetsFromBytes(path string, raw []byte) (*AssetMappingSet, error) {
	var doc assetsTOML
	if err := toml.Unmarshal(raw, &doc); err != nil {
		return nil, fmt.Errorf("parse %s: %w", path, err)
	}

	var errs []error

	if doc.Meta.TitleSlug == "" {
		errs = append(errs, fmt.Errorf("[meta].title_slug manquant"))
	}
	if doc.Meta.SchemaVersion <= 0 {
		errs = append(errs, fmt.Errorf("[meta].schema_version doit être > 0 (reçu %d)", doc.Meta.SchemaVersion))
	}

	byKindID := make(map[string]map[string]AssetMapping)
	// Collision detection : pour chaque kind, deux assets ne doivent pas
	// partager le même display_order.
	orderCollisions := make(map[string]map[int]string)

	for kind, entries := range doc.Assets {
		if strings.TrimSpace(kind) == "" {
			errs = append(errs, fmt.Errorf("kind d'asset vide"))
			continue
		}
		if len(entries) == 0 {
			errs = append(errs, fmt.Errorf("[assets.%s] sans entrée", kind))
			continue
		}
		bucket := make(map[string]AssetMapping, len(entries))
		seen := make(map[int]string)
		for id, entry := range entries {
			if strings.TrimSpace(id) == "" {
				errs = append(errs, fmt.Errorf("[assets.%s.<empty>]", kind))
				continue
			}
			entryErrs := validateAsset(kind, id, entry)
			if len(entryErrs) > 0 {
				for _, e := range entryErrs {
					errs = append(errs, fmt.Errorf("[assets.%s.%s]: %w", kind, id, e))
				}
				continue
			}
			if other, ok := seen[entry.DisplayOrder]; ok {
				errs = append(errs,
					fmt.Errorf("[assets.%s] display_order=%d en collision : %s et %s",
						kind, entry.DisplayOrder, other, id))
			} else {
				seen[entry.DisplayOrder] = id
			}
			bucket[id] = AssetMapping{
				Kind:         kind,
				ID:           id,
				Labels:       copyStringMap(entry.Labels),
				ColorToken:   entry.ColorToken,
				Icon:         entry.Icon,
				DisplayOrder: entry.DisplayOrder,
			}
		}
		byKindID[kind] = bucket
		orderCollisions[kind] = seen
	}

	if len(errs) > 0 {
		return nil, fmt.Errorf("validation %s: %w", path, errors.Join(errs...))
	}

	return NewAssetMappingSet(doc.Meta.TitleSlug, doc.Meta.SchemaVersion, byKindID), nil
}

func validateAsset(kind, id string, e assetEntryTOML) []error {
	var errs []error
	if _, ok := e.Labels[LocaleEN]; !ok || strings.TrimSpace(e.Labels[LocaleEN]) == "" {
		errs = append(errs, fmt.Errorf("label EN manquant"))
	}
	if _, ok := e.Labels[LocaleFR]; !ok || strings.TrimSpace(e.Labels[LocaleFR]) == "" {
		errs = append(errs, fmt.Errorf("label FR manquant"))
	}
	if e.DisplayOrder < 0 {
		errs = append(errs, fmt.Errorf("display_order doit être >= 0 (reçu %d)", e.DisplayOrder))
	}
	_ = kind
	_ = id
	return errs
}

func copyStringMap(m map[string]string) map[string]string {
	out := make(map[string]string, len(m))
	for k, v := range m {
		out[k] = v
	}
	return out
}
