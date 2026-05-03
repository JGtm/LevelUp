package mappings

import (
	"errors"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/pelletier/go-toml/v2"
)

// assetsTOML est la projection brute du fichier assets.toml avant validation.
type assetsTOML struct {
	Meta   metaSection                          `toml:"meta"`
	Assets map[string]map[string]assetEntryTOML `toml:"assets"` // kind → id → entry
	_      struct{}                             `toml:"-"`
}

type assetEntryTOML struct {
	Labels       map[string]string `toml:"labels"`
	ColorToken   string            `toml:"color_token"`
	Icon         string            `toml:"icon"`
	DisplayOrder int               `toml:"display_order"`

	// Champs optionnels pour les kinds time-bounded (ex: "season").
	// Format strict RFC 3339 (ex: "2021-12-08T00:00:00Z").
	StartDate string            `toml:"start_date,omitempty"`
	EndDate   string            `toml:"end_date,omitempty"`
	Extra     map[string]string `toml:"extra,omitempty"`
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
			startDate, endDate, dateErr := parseAssetWindow(entry.StartDate, entry.EndDate)
			if dateErr != nil {
				errs = append(errs, fmt.Errorf("[assets.%s.%s]: %w", kind, id, dateErr))
				continue
			}
			bucket[id] = AssetMapping{
				Kind:         kind,
				ID:           id,
				Labels:       copyStringMap(entry.Labels),
				ColorToken:   entry.ColorToken,
				Icon:         entry.Icon,
				DisplayOrder: entry.DisplayOrder,
				StartDate:    startDate,
				EndDate:      endDate,
				Extra:        copyStringMap(entry.Extra),
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
	if len(m) == 0 {
		return nil
	}
	out := make(map[string]string, len(m))
	for k, v := range m {
		out[k] = v
	}
	return out
}

// parseAssetWindow parse les champs start_date / end_date d'une entrée TOML.
//
// Format attendu : RFC 3339 strict (ex: "2021-12-08T00:00:00Z").
// Retourne (nil, nil, nil) si les deux champs sont vides (kind non-temporel).
// Retourne une erreur si une date est mal formée OU si end < start.
func parseAssetWindow(startRaw, endRaw string) (*time.Time, *time.Time, error) {
	startRaw = strings.TrimSpace(startRaw)
	endRaw = strings.TrimSpace(endRaw)
	if startRaw == "" && endRaw == "" {
		return nil, nil, nil
	}

	var startPtr, endPtr *time.Time
	if startRaw != "" {
		t, err := time.Parse(time.RFC3339, startRaw)
		if err != nil {
			return nil, nil, fmt.Errorf("start_date invalide %q : RFC 3339 attendu (ex: 2021-12-08T00:00:00Z)", startRaw)
		}
		startPtr = &t
	}
	if endRaw != "" {
		t, err := time.Parse(time.RFC3339, endRaw)
		if err != nil {
			return nil, nil, fmt.Errorf("end_date invalide %q : RFC 3339 attendu (ex: 2022-05-03T00:00:00Z)", endRaw)
		}
		endPtr = &t
	}
	if startPtr != nil && endPtr != nil && endPtr.Before(*startPtr) {
		return nil, nil, fmt.Errorf("end_date %q est avant start_date %q", endRaw, startRaw)
	}
	return startPtr, endPtr, nil
}
