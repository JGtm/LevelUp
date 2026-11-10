// Package depottest : OU TROUVER LES DONNEES DE TEST LUES HORS DE LA BRIQUE.
//
// Certains tests de la brique lisent des donnees qu'elle ne porte pas : catalogues de reference sous
// `data/titles/`, sources du front sous `apps/`, parc local sous `data/cache/`. Elles vivent dans un
// depot qui embarque la brique ; [Racine] cherche la racine de ce depot dans cet ordre.
//
//  1. La variable d'environnement [EnvRacineDonnees] (`DECODEUR_FILM_RACINE_DONNEES`) : la racine
//     d'un depot qui fournit les donnees de test lues hors de la brique, un dossier qui existe. C'est
//     la voie d'une integration continue, et de tout poste qui tient la brique hors de l'espace de
//     travail de ce depot. Une valeur qui ne designe pas un dossier est une ERREUR (le test echoue),
//     jamais un saut : une variable posee qui ne mene nulle part ne doit pas se lire comme une brique
//     prise seule.
//  2. Sans variable, le premier dossier au-dessus de CE FICHIER SOURCE qui porte `go.work`. Cette voie
//     n'existe que tant que la brique vit dans un espace de travail ; elle disparait avec lui.
//  3. Sinon la brique est prise seule : le test se declare absent (t.Skip), il n'a pas de donnee a
//     lire.
//
// C'est une aide de test : aucun fichier de production de la brique ne l'importe (garde-rail de son
// depot) et c'est la seule remontee vers la racine de la brique.
package depottest

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"
)

// EnvRacineDonnees nomme la variable d'environnement qui donne la racine du depot de donnees.
const EnvRacineDonnees = "DECODEUR_FILM_RACINE_DONNEES"

// Racine rend la racine du depot qui fournit les donnees de test (voir la documentation du paquet pour
// l'ordre de recherche), ou saute le test quand la brique est prise seule.
func Racine(t testing.TB) string {
	t.Helper()
	if dir := os.Getenv(EnvRacineDonnees); dir != "" {
		abs, err := filepath.Abs(dir)
		if err != nil {
			t.Fatalf("%s=%q : chemin inexploitable : %v", EnvRacineDonnees, dir, err)
		}
		info, err := os.Stat(abs)
		if err != nil || !info.IsDir() {
			t.Fatalf("%s=%q ne designe pas un dossier existant (err=%v)", EnvRacineDonnees, dir, err)
		}
		return abs
	}
	_, ce, _, ok := runtime.Caller(0)
	if !ok {
		t.Skip("racine des donnees de test indeductible (runtime.Caller a echoue)")
	}
	for dir := filepath.Dir(ce); ; {
		if _, err := os.Stat(filepath.Join(dir, "go.work")); err == nil {
			return dir
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			t.Skipf("%s absente et aucun go.work au-dessus de %s : brique prise seule", EnvRacineDonnees, filepath.Dir(ce))
		}
		dir = parent
	}
}
