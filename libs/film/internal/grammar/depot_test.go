package grammar

// depot_test.go — LES CHEMINS DU DEPOT HOTE que lisent les tests de ce paquet (catalogues de
// reference sous `data/titles/`, parc local sous `data/cache/`). La racine vient de
// [depottest.Racine] : hors depot hote, le test se declare absent.

import (
	"path/filepath"
	"testing"

	"github.com/JGtm/LevelUp/libs/film/internal/depottest"
)

// referenceDuDepot rend le chemin d'un catalogue de reference versionne du titre
// (`data/titles/halo_infinite/reference/<nom>`) dans le depot hote.
func referenceDuDepot(t *testing.T, nom string) string {
	t.Helper()
	return filepath.Join(depottest.Racine(t), "data", "titles", "halo_infinite", "reference", nom)
}

// bornesDeCartes rend le chemin du catalogue des bornes de quantification (`map_quant_bounds.json`).
func bornesDeCartes(t *testing.T) string {
	t.Helper()
	return referenceDuDepot(t, "map_quant_bounds.json")
}

// cacheDuDepot rend le dossier `data/cache` du depot hote (films decoupes, artefacts : le parc
// local, non versionne). Les instruments qui le lisent sautent quand il est vide.
func cacheDuDepot(t *testing.T) string {
	t.Helper()
	return filepath.Join(depottest.Racine(t), "data", "cache")
}
