package mappings

import (
	"errors"
	"fmt"
	"os"
	"sort"

	"github.com/pelletier/go-toml/v2"

	"levelup/go-api/internal/games/canonical"
	"levelup/go-api/internal/i18n"
)

// fieldsTOML est la projection brute du fichier fields.toml avant validation.
type fieldsTOML struct {
	Meta   metaSection               `toml:"meta"`
	Fields map[string]fieldEntryTOML `toml:"fields"`
}

type metaSection struct {
	TitleSlug     string `toml:"title_slug"`
	SchemaVersion int    `toml:"schema_version"`
	// GamePrefix est le segment d'URL de jeu ("hi" pour Halo Infinite, "h5" pour
	// Halo 5) injecté dans les chemins d'API d'ingestion (/hi/players/..., etc.).
	// Optionnel : lu seulement par le loader endpoints (constants.toml) ; absent
	// des autres mappings TOML. Vide → le consommateur retombe sur le défaut "hi".
	GamePrefix string `toml:"game_prefix"`
}

type fieldEntryTOML struct {
	Labels       i18n.Text `toml:"labels"`
	Description  i18n.Text `toml:"description"`
	StorageUnit  string    `toml:"storage_unit"`
	DisplayUnit  string    `toml:"display_unit"`
	Format       string    `toml:"format"`
	DisplayOrder int       `toml:"display_order"`
	Group        string    `toml:"group"`
	Icon         string    `toml:"icon"`
}

// LoadFieldsFromFile lit et valide un fields.toml à un chemin donné.
// Retourne une erreur agrégée listant tous les problèmes détectés (le boot
// d'un titre est tout-ou-rien : on n'autorise pas un FieldMappingSet partiel).
func LoadFieldsFromFile(path string) (*FieldMappingSet, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read %s: %w", path, err)
	}
	return LoadFieldsFromBytes(path, raw)
}

// LoadFieldsFromBytes parse et valide un payload TOML déjà chargé en mémoire.
// path n'est utilisé que pour les messages d'erreur.
func LoadFieldsFromBytes(path string, raw []byte) (*FieldMappingSet, error) {
	var doc fieldsTOML
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

	if len(doc.Fields) == 0 {
		errs = append(errs, fmt.Errorf("aucune section [fields.X] trouvée"))
	}

	mappings := make(map[canonical.FieldKey]FieldMapping, len(doc.Fields))
	groupOrders := make(map[string]map[int]string) // group -> displayOrder -> fieldKey (collision detection)

	for name, entry := range doc.Fields {
		key := canonical.FieldKey(name)
		entryErrs := validateField(key, entry)
		if len(entryErrs) > 0 {
			for _, e := range entryErrs {
				errs = append(errs, fmt.Errorf("[fields.%s]: %w", name, e))
			}
			continue
		}

		mapping := FieldMapping{
			Key:          key,
			Labels:       entry.Labels.Clone(),
			Descriptions: entry.Description.Clone(),
			StorageUnit:  Unit(entry.StorageUnit),
			DisplayUnit:  Unit(entry.DisplayUnit),
			Format:       Format(entry.Format),
			DisplayOrder: entry.DisplayOrder,
			Group:        entry.Group,
			Icon:         entry.Icon,
		}

		// Conversion d'unités déclarée mais non implémentée → erreur.
		if mapping.StorageUnit != mapping.DisplayUnit {
			if _, ok := ConvertValue(1.0, mapping.StorageUnit, mapping.DisplayUnit); !ok {
				errs = append(errs, fmt.Errorf("[fields.%s]: conversion %s -> %s non supportée", name, mapping.StorageUnit, mapping.DisplayUnit))
				continue
			}
		}

		// Collision display_order au sein du même groupe.
		if _, ok := groupOrders[mapping.Group]; !ok {
			groupOrders[mapping.Group] = make(map[int]string)
		}
		if other, dup := groupOrders[mapping.Group][mapping.DisplayOrder]; dup {
			errs = append(errs, fmt.Errorf(
				"[fields.%s]: display_order %d collisionne avec [fields.%s] dans le groupe %q",
				name, mapping.DisplayOrder, other, mapping.Group,
			))
			continue
		}
		groupOrders[mapping.Group][mapping.DisplayOrder] = name

		mappings[key] = mapping
	}

	if len(errs) > 0 {
		// Tri stable pour rendre les messages d'erreur reproductibles.
		msgs := make([]string, len(errs))
		for i, e := range errs {
			msgs[i] = e.Error()
		}
		sort.Strings(msgs)
		return nil, fmt.Errorf("validation %s: %d erreur(s):\n  - %s", path, len(errs), joinSorted(msgs))
	}

	return &FieldMappingSet{
		titleSlug:     doc.Meta.TitleSlug,
		schemaVersion: doc.Meta.SchemaVersion,
		byKey:         mappings,
	}, nil
}

func validateField(key canonical.FieldKey, e fieldEntryTOML) []error {
	var errs []error

	if !canonical.IsKnownFieldKey(key) {
		errs = append(errs, fmt.Errorf("FieldKey %q absent du canonique central (canonical.AllFieldKeys)", key))
	}

	if err := e.Labels.Trimmed().Validate(); err != nil {
		errs = append(errs, fmt.Errorf("labels: %w", err))
	}

	// Description optionnelle, mais dès qu'elle existe elle couvre chaque langue required.
	if len(e.Description) > 0 {
		if err := e.Description.Trimmed().Validate(); err != nil {
			errs = append(errs, fmt.Errorf("description: %w", err))
		}
	}

	if !IsKnownUnit(Unit(e.StorageUnit)) {
		errs = append(errs, fmt.Errorf("storage_unit inconnue: %q", e.StorageUnit))
	}
	if !IsKnownUnit(Unit(e.DisplayUnit)) {
		errs = append(errs, fmt.Errorf("display_unit inconnue: %q", e.DisplayUnit))
	}
	if !IsKnownFormat(Format(e.Format)) {
		errs = append(errs, fmt.Errorf("format inconnu: %q", e.Format))
	}
	if e.Group == "" {
		errs = append(errs, errors.New("group manquant"))
	}
	if e.DisplayOrder <= 0 {
		errs = append(errs, fmt.Errorf("display_order doit être > 0 (reçu %d)", e.DisplayOrder))
	}

	return errs
}

func joinSorted(msgs []string) string {
	out := ""
	for i, m := range msgs {
		if i > 0 {
			out += "\n  - "
		}
		out += m
	}
	return out
}
