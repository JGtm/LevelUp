//go:build gamefiles

package himap

// aides_corpus_gamefiles_test.go — petites aides de test dupliquées depuis le paquet de tests de
// LevelUp `internal/gamefilescorpus` (ecritPNG, mesureAncresToits) ou partagées avec lui
// (chainesLisibles). Une brique ne peut pas importer ce paquet ; chaque côté porte sa copie
// (deux copies au plus, règle 6 de CLAUDE.md).

import (
	"image"
	"image/png"
	"os"
	"testing"
)

// chainesLisibles rend les suites d'ASCII imprimables d'au moins minLen caracteres, au plus
// maxOut d'entre elles.
func chainesLisibles(b []byte, minLen, maxOut int) []string {
	var out []string
	cur := make([]byte, 0, 64)
	vide := func() {
		if len(cur) >= minLen && len(out) < maxOut {
			out = append(out, string(cur))
		}
		cur = cur[:0]
	}
	for _, c := range b {
		if c >= 0x20 && c < 0x7f {
			cur = append(cur, c)
			continue
		}
		vide()
	}
	vide()
	return out
}

func ecritPNG(t *testing.T, chemin string, img image.Image) {
	t.Helper()
	f, err := os.Create(chemin)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = f.Close() }()
	if err := png.Encode(f, img); err != nil {
		t.Fatal(err)
	}
}

// mesureAncresToits journalise, ancre par ancre, la surface HAUTE dessinee a son aplomb, en
// metres au-dessus de son sol. Un plafond se lit ici directement.
func mesureAncresToits(t *testing.T, r *Rendu, ancres [][3]float64) {
	t.Helper()
	var dessus []float64
	couvertes := 0
	for _, a := range ancres {
		i := int((a[0] - r.Min[0]) / r.Cell)
		j := int((a[1] - r.Min[1]) / r.Cell)
		if z, ok := r.Altitude(i, j); ok {
			d := z - (a[2] - AncrageDecalageSol)
			dessus = append(dessus, d)
			if d >= 4 {
				couvertes++
			}
		}
	}
	if len(dessus) > 0 {
		t.Logf("ANCRES : %d/%d sous une surface a >=4 m au-dessus de leur sol · surplomb q50 %.1f m · q90 %.1f m",
			couvertes, len(dessus), Centile(dessus, 0.5), Centile(dessus, 0.90))
	}
}
