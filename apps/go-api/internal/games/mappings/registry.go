package mappings

import (
	"errors"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"sync"
)

// Registry est le registre title-aware des mappings TOML chargés au boot.
//
// Construction : NewRegistry() puis LoadFromConfigDir(repoRoot, slugs...).
// Lecture : Get(slug) → FieldMappingSet, GetAssets(slug) → AssetMappingSet,
// GetOutcomes(slug) → OutcomeMappingSet.
// Une erreur de chargement pour un titre n'invalide pas les autres.
type Registry struct {
	mu           sync.RWMutex
	fields       map[string]*FieldMappingSet
	assets       map[string]*AssetMappingSet
	outcomes     map[string]*OutcomeMappingSet
	capabilities map[string]*CapabilityMappingSet
	endpoints    map[string]*EndpointSet
}

// NewRegistry crée un registre vide.
func NewRegistry() *Registry {
	return &Registry{
		fields:       make(map[string]*FieldMappingSet),
		assets:       make(map[string]*AssetMappingSet),
		outcomes:     make(map[string]*OutcomeMappingSet),
		capabilities: make(map[string]*CapabilityMappingSet),
		endpoints:    make(map[string]*EndpointSet),
	}
}

// LoadFromConfigDir charge tous les TOML mappings disponibles pour chaque slug
// fourni. fields.toml est obligatoire ; assets.toml et outcomes.toml sont
// optionnels (leur absence est silencieuse, leur présence est validée
// strictement).
//
// Retourne la liste agrégée des erreurs (une par titre/fichier) et nil si
// tout charge correctement.
func (r *Registry) LoadFromConfigDir(repoRoot string, slugs []string, logger *slog.Logger) []error {
	if logger == nil {
		logger = slog.Default()
	}
	var errs []error
	for _, slug := range slugs {
		titleDir := filepath.Join(repoRoot, "config", "titles", slug)
		mappingsDir := filepath.Join(titleDir, "mappings")

		// fields.toml — obligatoire
		fieldsPath := filepath.Join(mappingsDir, "fields.toml")
		fset, err := LoadFieldsFromFile(fieldsPath)
		if err != nil {
			logger.Error("mappings_validation_failed", "title_slug", slug, "path", fieldsPath, "err", err)
			errs = append(errs, fmt.Errorf("load fields %s: %w", slug, err))
			continue
		}
		r.mu.Lock()
		r.fields[slug] = fset
		r.mu.Unlock()
		logger.Info("mappings_loaded",
			"title_slug", slug,
			"kind", "fields",
			"fields_count", len(fset.All()),
			"schema_version", fset.SchemaVersion(),
		)

		// assets.toml — optionnel
		assetsPath := filepath.Join(mappingsDir, "assets.toml")
		if aset, loadErr := loadAssetsIfExists(assetsPath); loadErr != nil {
			logger.Error("mappings_validation_failed", "title_slug", slug, "path", assetsPath, "err", loadErr)
			errs = append(errs, fmt.Errorf("load assets %s: %w", slug, loadErr))
		} else if aset != nil {
			r.mu.Lock()
			r.assets[slug] = aset
			r.mu.Unlock()
			logger.Info("mappings_loaded",
				"title_slug", slug,
				"kind", "assets",
				"kinds_count", len(aset.Kinds()),
				"schema_version", aset.SchemaVersion(),
			)
		}

		// outcomes.toml — optionnel
		outcomesPath := filepath.Join(mappingsDir, "outcomes.toml")
		if oset, loadErr := loadOutcomesIfExists(outcomesPath); loadErr != nil {
			logger.Error("mappings_validation_failed", "title_slug", slug, "path", outcomesPath, "err", loadErr)
			errs = append(errs, fmt.Errorf("load outcomes %s: %w", slug, loadErr))
		} else if oset != nil {
			r.mu.Lock()
			r.outcomes[slug] = oset
			r.mu.Unlock()
			logger.Info("mappings_loaded",
				"title_slug", slug,
				"kind", "outcomes",
				"outcomes_count", len(oset.All()),
				"schema_version", oset.SchemaVersion(),
			)
		}

		// capabilities.toml — optionnel (Phase 1.7a)
		capsPath := filepath.Join(mappingsDir, "capabilities.toml")
		if cset, loadErr := loadCapabilitiesIfExists(capsPath); loadErr != nil {
			logger.Error("mappings_validation_failed", "title_slug", slug, "path", capsPath, "err", loadErr)
			errs = append(errs, fmt.Errorf("load capabilities %s: %w", slug, loadErr))
		} else if cset != nil {
			r.mu.Lock()
			r.capabilities[slug] = cset
			r.mu.Unlock()
			logger.Info("mappings_loaded",
				"title_slug", slug,
				"kind", "capabilities",
				"capabilities_count", len(cset.Keys()),
				"schema_version", cset.SchemaVersion(),
			)
		}

		// constants.toml — optionnel ; section [endpoints] (MT-01). Au niveau
		// du dossier titre (pas mappings/), aligné sur le constants.toml
		// documentaire existant.
		endpointsPath := filepath.Join(titleDir, "constants.toml")
		if eset, loadErr := loadEndpointsIfExists(endpointsPath); loadErr != nil {
			logger.Error("mappings_validation_failed", "title_slug", slug, "path", endpointsPath, "err", loadErr)
			errs = append(errs, fmt.Errorf("load endpoints %s: %w", slug, loadErr))
		} else if eset != nil {
			r.mu.Lock()
			r.endpoints[slug] = eset
			r.mu.Unlock()
			logger.Info("endpoints_loaded",
				"title", slug,
				"count", len(eset.Keys()),
				"schema_version", eset.SchemaVersion(),
			)
		}
	}
	return errs
}

func loadAssetsIfExists(path string) (*AssetMappingSet, error) {
	if _, err := os.Stat(path); errors.Is(err, os.ErrNotExist) {
		return nil, nil
	}
	return LoadAssetsFromFile(path)
}

func loadOutcomesIfExists(path string) (*OutcomeMappingSet, error) {
	if _, err := os.Stat(path); errors.Is(err, os.ErrNotExist) {
		return nil, nil
	}
	return LoadOutcomesFromFile(path)
}

func loadCapabilitiesIfExists(path string) (*CapabilityMappingSet, error) {
	if _, err := os.Stat(path); errors.Is(err, os.ErrNotExist) {
		return nil, nil
	}
	return LoadCapabilitiesFromFile(path)
}

func loadEndpointsIfExists(path string) (*EndpointSet, error) {
	if _, err := os.Stat(path); errors.Is(err, os.ErrNotExist) {
		return nil, nil
	}
	return LoadEndpointsFromFile(path)
}

// Get retourne le FieldMappingSet d'un titre s'il a été chargé.
func (r *Registry) Get(slug string) (*FieldMappingSet, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	v, ok := r.fields[slug]
	return v, ok
}

// GetAssets retourne l'AssetMappingSet d'un titre s'il existe.
func (r *Registry) GetAssets(slug string) (*AssetMappingSet, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	v, ok := r.assets[slug]
	return v, ok
}

// GetOutcomes retourne l'OutcomeMappingSet d'un titre s'il existe.
func (r *Registry) GetOutcomes(slug string) (*OutcomeMappingSet, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	v, ok := r.outcomes[slug]
	return v, ok
}

// GetCapabilities retourne le CapabilityMappingSet d'un titre s'il existe (Phase 1.7a).
func (r *Registry) GetCapabilities(slug string) (*CapabilityMappingSet, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	v, ok := r.capabilities[slug]
	return v, ok
}

// GetEndpoints retourne l'EndpointSet d'un titre s'il a été chargé (MT-01).
func (r *Registry) GetEndpoints(slug string) (*EndpointSet, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	v, ok := r.endpoints[slug]
	return v, ok
}

// Slugs retourne la liste des slugs avec un FieldMappingSet chargé.
func (r *Registry) Slugs() []string {
	r.mu.RLock()
	defer r.mu.RUnlock()
	out := make([]string, 0, len(r.fields))
	for s := range r.fields {
		out = append(out, s)
	}
	return out
}
