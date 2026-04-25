package mappings

import (
	"fmt"
	"log/slog"
	"path/filepath"
	"sync"
)

// Registry est le registre title-aware des FieldMappingSet chargés au boot.
//
// Construction : NewRegistry() puis LoadFromConfigDir(repoRoot, slugs...).
// Lecture : Get(slug) → lookup atomique.
// Une erreur de chargement pour un titre n'invalide pas les autres.
type Registry struct {
	mu  sync.RWMutex
	set map[string]*FieldMappingSet
}

// NewRegistry crée un registre vide.
func NewRegistry() *Registry {
	return &Registry{set: make(map[string]*FieldMappingSet)}
}

// LoadFromConfigDir charge config/titles/{slug}/mappings/fields.toml pour
// chaque slug fourni. Retourne la liste agrégée des erreurs (une par titre)
// et nil si tout charge correctement.
func (r *Registry) LoadFromConfigDir(repoRoot string, slugs []string, logger *slog.Logger) []error {
	if logger == nil {
		logger = slog.Default()
	}
	var errs []error
	for _, slug := range slugs {
		path := filepath.Join(repoRoot, "config", "titles", slug, "mappings", "fields.toml")
		set, err := LoadFieldsFromFile(path)
		if err != nil {
			logger.Error("mappings_load_failed", "title_slug", slug, "path", path, "err", err.Error())
			errs = append(errs, fmt.Errorf("load %s: %w", slug, err))
			continue
		}
		r.mu.Lock()
		r.set[slug] = set
		r.mu.Unlock()
		logger.Info("mappings_loaded",
			"title_slug", slug,
			"fields_count", len(set.All()),
			"schema_version", set.SchemaVersion(),
		)
	}
	return errs
}

// Get retourne le FieldMappingSet d'un titre s'il a été chargé.
func (r *Registry) Get(slug string) (*FieldMappingSet, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	v, ok := r.set[slug]
	return v, ok
}

// Slugs retourne la liste des slugs chargés avec succès.
func (r *Registry) Slugs() []string {
	r.mu.RLock()
	defer r.mu.RUnlock()
	out := make([]string, 0, len(r.set))
	for s := range r.set {
		out = append(out, s)
	}
	return out
}
