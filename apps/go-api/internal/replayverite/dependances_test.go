package replayverite

import (
	"go/parser"
	"go/token"
	"os"
	"strconv"
	"strings"
	"testing"
)

// TestDependances_AucunImportDuDecodeur : le banc juge le decodeur, il ne lui emprunte rien. Un
// import de `games/halo_infinite/film/...` (types, faits, facade) ou de `replaybuild` (qui porte la
// cuisson) ferait partager au juge les types et les erreurs du juge — interdit, tests compris sauf
// la lecture du fichier de faits par `replaybuild.ReadFactsFile` dans reel_test.go (le lecteur
// unique de ce format, sans lien avec l'artefact juge).
func TestDependances_AucunImportDuDecodeur(t *testing.T) {
	entrees, err := os.ReadDir(".")
	if err != nil {
		t.Fatal(err)
	}
	fset := token.NewFileSet()
	lus := 0
	for _, e := range entrees {
		nom := e.Name()
		if !strings.HasSuffix(nom, ".go") {
			continue
		}
		f, err := parser.ParseFile(fset, nom, nil, parser.ImportsOnly)
		if err != nil {
			t.Fatal(err)
		}
		lus++
		for _, imp := range f.Imports {
			chemin, _ := strconv.Unquote(imp.Path.Value)
			if strings.Contains(chemin, "/games/halo_infinite/film") {
				t.Errorf("%s importe %s : le banc n'importe rien du decodeur", nom, chemin)
			}
			if strings.HasSuffix(chemin, "/internal/replaybuild") && nom != "reel_test.go" {
				t.Errorf("%s importe %s : seule la lecture des faits d'un test y est admise", nom, chemin)
			}
		}
	}
	if lus < 10 {
		t.Fatalf("seulement %d fichiers lus : le parcours est casse", lus)
	}
}
