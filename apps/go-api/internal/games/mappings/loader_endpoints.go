package mappings

import (
	"errors"
	"fmt"
	"os"
	"strings"

	"github.com/pelletier/go-toml/v2"
)

// endpointsTOML est la projection brute de constants.toml (section [endpoints]).
type endpointsTOML struct {
	Meta      metaSection       `toml:"meta"`
	Endpoints map[string]string `toml:"endpoints"`
}

// allowedEndpointKeys est la liste exhaustive des clés d'endpoint admises (MT-01).
// Toute autre clé dans [endpoints] est rejetée (un titre ne déclare pas un host
// hors vocabulaire d'ingestion canonique).
var allowedEndpointKeys = map[EndpointKey]struct{}{
	EndpointStats:        {},
	EndpointGameCMS:      {},
	EndpointEconomy:      {},
	EndpointSkill:        {},
	EndpointUGCFilm:      {},
	EndpointDiscoveryUGC: {},
	EndpointChallenges:   {},
	EndpointNameplate:    {},
}

// LoadEndpointsFromFile lit et valide la section [endpoints] d'un constants.toml.
func LoadEndpointsFromFile(path string) (*EndpointSet, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read %s: %w", path, err)
	}
	return LoadEndpointsFromBytes(path, raw)
}

// LoadEndpointsFromBytes parse et valide un payload TOML déjà chargé en mémoire.
// path n'est utilisé que pour les messages d'erreur.
func LoadEndpointsFromBytes(path string, raw []byte) (*EndpointSet, error) {
	var doc endpointsTOML
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
	if len(doc.Endpoints) == 0 {
		errs = append(errs, fmt.Errorf("section [endpoints] absente ou vide"))
	}

	byKey := make(map[EndpointKey]string, len(doc.Endpoints))
	for rawKey, host := range doc.Endpoints {
		key := EndpointKey(rawKey)
		if _, ok := allowedEndpointKeys[key]; !ok {
			errs = append(errs, fmt.Errorf("[endpoints.%s] clé inconnue (admises : stats, gamecms, economy, skill, ugc_film, discovery_ugc, challenges, nameplate)", rawKey))
			continue
		}
		trimmed := strings.TrimSpace(host)
		if trimmed == "" {
			errs = append(errs, fmt.Errorf("[endpoints.%s] host vide", rawKey))
			continue
		}
		if !strings.HasPrefix(trimmed, "https://") {
			errs = append(errs, fmt.Errorf("[endpoints.%s] host non-https: %q", rawKey, trimmed))
			continue
		}
		byKey[key] = trimmed
	}

	if len(errs) > 0 {
		return nil, fmt.Errorf("validation %s: %w", path, errors.Join(errs...))
	}

	return NewEndpointSet(doc.Meta.TitleSlug, doc.Meta.SchemaVersion, byKey), nil
}
