package grammar

// marche_trames_unique_test.go — UN SEUL PILOTAGE DE LA MARCHE DES TRAMES (ADR 0037 IR-2).
//
// Le pilotage de la phase delta — chunk par chunk la liaison des images-cles au monde
// ([lierLesImagesClesDuChunk]), paquet par paquet la localisation des listes d evenements
// ([localiserLaListe]) — vit dans `marche_trames.go`. Les etats de mouvement et le tir continu,
// canaux du distributeur ([Distribuer]), et la carte de fermeture le CONSOMMENT
// ([FilmContext.Trames], [marcheurDesTrames.parcourir]) ; aucun fichier de production du paquet ne
// le recopie. La forme instrument de la liaison (`lierLeChunkAuMonde`) vit dans les tests.

import (
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"strings"
	"testing"
)

// fichierDuPilotage est le seul fichier de production qui pilote la marche des trames.
const fichierDuPilotage = "marche_trames.go"

// primitivesDuPilotage sont les appels qui font une marche des trames.
var primitivesDuPilotage = []string{"lierLesImagesClesDuChunk", "localiserLaListe"}

// TestLaMarcheDesTramesEstPiloteeEnUnSeulEndroit : dans le code de production du paquet, les
// primitives du pilotage ne sont appelees que depuis `marche_trames.go`, et elles y sont appelees.
// MUTATION — un appel a [localiserLaListe] dans un autre fichier de production : ROUGE.
func TestLaMarcheDesTramesEstPiloteeEnUnSeulEndroit(t *testing.T) {
	appels := appelsDeProduction(t, primitivesDuPilotage)
	for _, p := range primitivesDuPilotage {
		if len(appels[p]) == 0 {
			t.Errorf("%s n est appelee par aucun fichier de production : le garde-rail ne garde plus rien", p)
		}
		for _, ou := range appels[p] {
			if !strings.HasPrefix(ou, fichierDuPilotage+":") {
				t.Errorf("%s appelle %s : le pilotage de la marche des trames est recopie hors de %s",
					ou, p, fichierDuPilotage)
			}
		}
	}
}

// appelsDeProduction rend, pour chaque fonction de `noms`, les positions (`fichier:ligne`) des
// appels que les fichiers de production du paquet en font.
func appelsDeProduction(t *testing.T, noms []string) map[string][]string {
	t.Helper()
	cherches := map[string]bool{}
	for _, n := range noms {
		cherches[n] = true
	}
	entrees, err := os.ReadDir(".")
	if err != nil {
		t.Fatalf("lecture du paquet : %v", err)
	}
	fset := token.NewFileSet()
	out := map[string][]string{}
	for _, e := range entrees {
		nom := e.Name()
		if e.IsDir() || !strings.HasSuffix(nom, ".go") || strings.HasSuffix(nom, "_test.go") {
			continue
		}
		f, err := parser.ParseFile(fset, nom, nil, parser.SkipObjectResolution)
		if err != nil {
			t.Fatalf("analyse de %s : %v", nom, err)
		}
		ast.Inspect(f, func(n ast.Node) bool {
			if appel, ok := n.(*ast.CallExpr); ok {
				if id, ok := appel.Fun.(*ast.Ident); ok && cherches[id.Name] {
					pos := fset.Position(appel.Pos())
					out[id.Name] = append(out[id.Name], fmt.Sprintf("%s:%d", nom, pos.Line))
				}
			}
			return true
		})
	}
	return out
}
