package wire

// registry_spawn_families.go — LA TABLE DES FAMILLES DE MODES D'UN TITRE (spawn_families.toml), lue UNE
// fois par titre et gardee en memoire : elle dit quels modes la prevision de reapparition couvre, et
// l'onglet Tactique la consulte a chaque lecture « reapparitions adverses ».

import (
	"errors"
	"io/fs"
	"log/slog"
	"path/filepath"

	"levelup/go-api/internal/games/mappings"
)

// fichierFamillesDeModes est le nom du fichier dans le dossier des mappings d'un titre.
const fichierFamillesDeModes = "spawn_families.toml"

// spawnFamiliesFor rend la table du titre, ou nil quand il n'en a pas (aucun mode couvert). Un titre
// sans fichier est l'etat normal (Halo 5) et ne se journalise pas ; un fichier present mais illisible se
// journalise, une seule fois, et le titre retombe sur « aucun mode couvert ».
func (r *ServiceRegistry) spawnFamiliesFor(slug string) *mappings.SpawnFamilySet {
	if v, ok := r.spawnFamilies.Load(slug); ok {
		set, _ := v.(*mappings.SpawnFamilySet)
		return set
	}
	set := r.chargerSpawnFamilies(slug)
	r.spawnFamilies.Store(slug, set)
	return set
}

func (r *ServiceRegistry) chargerSpawnFamilies(slug string) *mappings.SpawnFamilySet {
	if r.cfg == nil {
		return nil
	}
	path := filepath.Join(r.cfg.PathResolver().TitleMappingsDir(slug), fichierFamillesDeModes)
	set, err := mappings.LoadSpawnFamiliesFromFile(path)
	switch {
	case errors.Is(err, fs.ErrNotExist):
		return nil
	case err != nil:
		slog.Error("familles de modes illisibles : aucun mode couvert par la prevision de reapparition",
			"titleSlug", slug, "path", path, "err", err)
		return nil
	}
	return set
}
