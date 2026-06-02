package mappings

import (
	"errors"
	"fmt"
	"os"

	"github.com/pelletier/go-toml/v2"
)

// capabilitiesTOML est la projection brute de capabilities.toml avant validation.
type capabilitiesTOML struct {
	Meta         metaSection       `toml:"meta"`
	Capabilities map[string]string `toml:"capabilities"`
}

// LoadCapabilitiesFromFile lit et valide un capabilities.toml à un chemin donné.
func LoadCapabilitiesFromFile(path string) (*CapabilityMappingSet, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read %s: %w", path, err)
	}
	return LoadCapabilitiesFromBytes(path, raw)
}

// LoadCapabilitiesFromBytes parse et valide un payload TOML déjà en mémoire.
//
// Validation : meta présente + chaque statut ∈ {supported, degraded,
// not_exposed}. Les CLÉS de capability NE sont PAS validées ici (vocabulaire
// produit, propriété de games) — un titre peut déclarer une clé que games ne
// connaît pas ; la conversion games.CapabilityMapFromMappings la rejettera.
func LoadCapabilitiesFromBytes(path string, raw []byte) (*CapabilityMappingSet, error) {
	var doc capabilitiesTOML
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
	if len(doc.Capabilities) == 0 {
		errs = append(errs, fmt.Errorf("[capabilities] vide ou absent"))
	}

	byKey := make(map[string]string, len(doc.Capabilities))
	for key, status := range doc.Capabilities {
		if !ValidCapStatus(status) {
			errs = append(errs, fmt.Errorf("[capabilities.%q] statut inconnu %q (admis : supported, degraded, not_exposed)", key, status))
			continue
		}
		byKey[key] = status
	}

	if len(errs) > 0 {
		return nil, fmt.Errorf("validation %s: %w", path, errors.Join(errs...))
	}

	return NewCapabilityMappingSet(doc.Meta.TitleSlug, doc.Meta.SchemaVersion, byKey), nil
}
