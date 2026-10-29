package grammar

// depot_test.go — OU TROUVER LE DEPOT HOTE POUR LES TESTS DE LA BRIQUE (plan D2, item 2.11,
// regle R2, 2026-10-29).
//
// Certains tests de ce paquet lisent des donnees versionnees du depot hote (catalogues de
// reference sous `data/titles/`, fixtures du front sous `apps/`) ou son parc local
// (`data/cache/`). La brique n'importe plus rien du depot : elle remonte donc a la racine par
// l'EMPLACEMENT DE CE FICHIER SOURCE, jusqu'au premier dossier qui porte `go.work` (le fichier
// d'espace de travail du depot, versionne). Hors depot hote (la brique copiee seule), aucun
// `go.work` n'est trouve et le test se declare absent (t.Skip) : il n'a pas de donnee a lire.
//
// UNE SEULE COPIE PAR PAQUET, nommee pareil partout (`racineDepot`) : une remontee maison par
// fichier est ce que le ratchet `archlint/no_repo_root_walk_test.go` interdit.

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"
)

// racineDepot rend la racine du depot hote (le dossier qui porte `go.work`), ou saute le test.
func racineDepot(t *testing.T) string {
	t.Helper()
	_, ce, _, ok := runtime.Caller(0)
	if !ok {
		t.Skip("racine du depot hote indeductible (runtime.Caller a echoue)")
	}
	for dir := filepath.Dir(ce); ; {
		if _, err := os.Stat(filepath.Join(dir, "go.work")); err == nil {
			return dir
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			t.Skipf("aucun go.work au-dessus de %s : brique hors du depot hote", filepath.Dir(ce))
		}
		dir = parent
	}
}

// referenceDuDepot rend le chemin d'un catalogue de reference versionne du titre
// (`data/titles/halo_infinite/reference/<nom>`) dans le depot hote.
func referenceDuDepot(t *testing.T, nom string) string {
	t.Helper()
	return filepath.Join(racineDepot(t), "data", "titles", "halo_infinite", "reference", nom)
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
	return filepath.Join(racineDepot(t), "data", "cache")
}
