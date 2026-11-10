package grammar

// depot_test.go — LES CHEMINS DU DEPOT HOTE que lisent les tests de ce paquet (catalogues de
// reference sous `data/titles/`, parc local sous `data/cache/`). La racine vient de
// [depottest.Racine] : hors depot hote, le test se declare absent.

import (
	"path/filepath"
	"testing"

	"github.com/JGtm/halo-infinite-film-decoder/internal/depottest"
)

// referenceDuDepot rend le chemin d'un catalogue de reference versionne du titre
// (`data/titles/halo_infinite/reference/<nom>`) dans le depot hote.
func referenceDuDepot(t testing.TB, nom string) string {
	t.Helper()
	return filepath.Join(depottest.Racine(t), "data", "titles", "halo_infinite", "reference", nom)
}

// bornesDeCartes rend le chemin du catalogue des bornes de quantification (`map_quant_bounds.json`)
// que la brique porte dans ses propres fixtures (`testdata/`) : les tests qui en ont besoin tournent
// sans depot hote. Le catalogue LIVRE par l'hote est [bornesLivreesParLHote] ; le test
// `TestLesBornesDeTestSontLeCatalogueLivre` garde les deux octet a octet.
func bornesDeCartes(t testing.TB) string {
	t.Helper()
	return filepath.Join("testdata", "map_quant_bounds.json")
}

// bornesLivreesParLHote rend le chemin du catalogue des bornes tel que le depot hote le livre
// (`data/titles/halo_infinite/reference/map_quant_bounds.json`) ; saute le test hors depot hote.
func bornesLivreesParLHote(t testing.TB) string {
	t.Helper()
	return referenceDuDepot(t, "map_quant_bounds.json")
}

// cacheDuDepot rend le dossier `data/cache` du depot hote (films decoupes, artefacts : le parc
// local, non versionne). Les instruments qui le lisent sautent quand il est vide.
func cacheDuDepot(t *testing.T) string {
	t.Helper()
	return filepath.Join(depottest.Racine(t), "data", "cache")
}
