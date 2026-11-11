package replay

// depot_test.go — LES CHEMINS DU DEPOT HOTE que lisent les tests de ce paquet (catalogues de
// reference sous `data/titles/`, parc local sous `data/cache/`). La racine vient de
// [depottest.Racine] : hors depot hote, le test se declare absent.

import (
	"path/filepath"
	"testing"

	"github.com/JGtm/halo-infinite-film-decoder/internal/depottest"
)

// titreDeReference : le titre dont les tests de ce paquet lisent les donnees de reference du
// depot hote. Un test de la brique Halo Infinite le dit en clair : aucune branche sur le titre.
const titreDeReference = "halo_infinite"

// bornesDeCartes rend le chemin du catalogue versionne des bornes de quantification
// (`data/titles/halo_infinite/reference/map_quant_bounds.json`) dans le depot hote.
func bornesDeCartes(t testing.TB) string {
	t.Helper()
	return filepath.Join(depottest.Racine(t), "data", "titles", titreDeReference, "reference", "map_quant_bounds.json")
}

// cacheDuDepot rend le dossier `data/cache` du depot hote (films decoupes, artefacts : le parc
// local, non versionne). Les instruments qui le lisent sautent quand il est vide.
func cacheDuDepot(t *testing.T) string {
	t.Helper()
	return filepath.Join(depottest.Racine(t), "data", "cache")
}
