package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/JGtm/halo-infinite-files-reader/mapvar"
)

func TestAncresDesObjectifsRendLaPositionMonde(t *testing.T) {
	var a, b mapvar.Objective
	a.Pos.X, a.Pos.Y, a.Pos.Z = 1, 2, 3
	b.Pos.X, b.Pos.Y, b.Pos.Z = -4, 5.5, 0
	got := ancresDesObjectifs([]mapvar.Objective{a, b})
	if len(got) != 2 || got[0] != [3]float64{1, 2, 3} || got[1] != [3]float64{-4, 5.5, 0} {
		t.Errorf("ancres = %v", got)
	}
	if got := ancresDesObjectifs(nil); len(got) != 0 {
		t.Errorf("sans objectif : %v", got)
	}
}

// Garde-rail : une position d'objectif ne devient une ancre qu'à UN endroit. Les trois sites de
// la cuisson native, de la cuisson Forge et du plan seul l'écrivaient chacun, et une règle écrite
// trois fois diverge (un objectif ajouté à l'une n'arrivait pas aux autres).
func TestAncresConstruitesEnUnSeulPoint(t *testing.T) {
	// Écrit en deux morceaux : le littéral entier ne doit se lire qu'une fois dans le paquet.
	const litteral = "[3]float64{o.Pos" + ".X"
	fichiers, err := filepath.Glob("*.go")
	if err != nil {
		t.Fatal(err)
	}
	var trouves []string
	for _, f := range fichiers {
		if strings.HasSuffix(f, "_test.go") {
			continue
		}
		blob, err := os.ReadFile(f)
		if err != nil {
			t.Fatal(err)
		}
		if strings.Contains(string(blob), litteral) {
			trouves = append(trouves, f)
		}
	}
	if len(trouves) != 1 || trouves[0] != "cuisson.go" {
		t.Errorf("le littéral %q doit n'apparaître que dans ancresDesObjectifs (cuisson.go), trouvé dans %v", litteral, trouves)
	}
}
