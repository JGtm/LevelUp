// Package archlint — no_local_portee_comparison_test.go : garde-rail de la comparaison « une
// distance au coequipier le plus proche est-elle a portee du radar ? » (CLAUDE.md regle 6).
//
// La regle tient en une ligne — une distance MESUREE, a la portee ou en deca (borne INCLUSIVE) ;
// une distance absente n'est jamais a portee — et sa SOURCE UNIQUE est `coordination.APortee`
// (`internal/analysis/coordination/vies_pres_ou_seul.go`). Toutes les lectures qui rangent une
// mort « pres » ou « seule » la lisent : l'isolement de la Tactique, les vies pres d'un coequipier,
// le detail d'une zone et son badge de placement. Une copie qui ecrirait `<` au lieu de `<=`, ou qui
// rangerait une distance absente a portee, ferait dire « pres » a une page et « seul » a l'autre
// pour la meme mort.
//
// Les deux empreintes, telles qu'une copie s'ecrit :
//  1. une distance DEREFERENCEE comparee par `<=` a une portee (`*m.PlusProcheM <= rayon`) ;
//  2. le test de presence de la distance au plus proche (`PlusProcheM != nil &&`).
//
// Fichiers de production seulement (un test peut fabriquer ses distances comme il veut). Aucune
// allowlist hors du helper ; l'auto-test prouve que l'empreinte reconnait l'ancienne copie du
// detail de cellule.
package archlint

import (
	"os"
	"path/filepath"
	"regexp"
	"runtime"
	"strings"
	"testing"
)

// aPorteeHelper : le seul fichier qui compare une distance a une portee.
const aPorteeHelper = "internal/analysis/coordination/vies_pres_ou_seul.go"

var (
	// distanceDerefComparee : empreinte 1.
	distanceDerefComparee = regexp.MustCompile(`\*\s*[\w.]+\s*<=\s*\w*(?i:rayon|portee)\w*`)
	// presencePlusProche : empreinte 2.
	presencePlusProche = regexp.MustCompile(`PlusProcheM\s*!=\s*nil\s*&&`)
)

// empreintesDeComparaison rend les empreintes de comparaison locale presentes dans un texte.
func empreintesDeComparaison(source string) []string {
	var out []string
	if distanceDerefComparee.MatchString(source) {
		out = append(out, "distance dereferencee comparee a une portee")
	}
	if presencePlusProche.MatchString(source) {
		out = append(out, "presence de la distance au plus proche testee a la main")
	}
	return out
}

func TestNoLocalPorteeComparison_ReconnaitLesCopies(t *testing.T) {
	copies := []string{
		// service/tactical_service_cellule.go, celluleIsole (avant le plan Tactique v2, lot L1)
		"\t\taccompagnee := m.PlusProcheM != nil && *m.PlusProcheM <= rayon",
		// la forme du helper lui-meme, recopiee ailleurs
		"\treturn d != nil && *d <= rayon",
		"\tif dist != nil && *dist <= portee {",
	}
	for _, c := range copies {
		if len(empreintesDeComparaison(c)) == 0 {
			t.Errorf("le garde-rail ne reconnait pas une copie :\n%s", c)
		}
	}
	for _, sain := range []string{
		"\t\taccompagnee := coordination.APortee(m.PlusProcheM, rayon)",
		"\tfor d := -rayon; d <= rayon && !out[j*r.NX+i]; d++ {",
		"\t\tout[k] = v <= rayon",
		"\t\tif distance(p.X, p.Y, p.Z, c[0], c[1], &z) <= rayonPorteM {",
	} {
		if e := empreintesDeComparaison(sain); len(e) != 0 {
			t.Errorf("faux positif sur %q : %v", sain, e)
		}
	}
}

func TestNoLocalPorteeComparison(t *testing.T) {
	_, thisFile, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("runtime.Caller a echoue")
	}
	goAPIRoot := filepath.Dir(filepath.Dir(filepath.Dir(thisFile)))
	var violations []string
	for _, sub := range []string{"internal", "cmd"} {
		err := filepath.WalkDir(filepath.Join(goAPIRoot, sub), func(path string, d os.DirEntry, err error) error {
			if err != nil || d.IsDir() || !strings.HasSuffix(path, ".go") || strings.HasSuffix(path, "_test.go") {
				return err
			}
			rel, _ := filepath.Rel(goAPIRoot, path)
			rel = filepath.ToSlash(rel)
			if rel == aPorteeHelper {
				return nil
			}
			data, readErr := os.ReadFile(path)
			if readErr != nil {
				return readErr
			}
			for _, e := range empreintesDeComparaison(string(data)) {
				violations = append(violations, rel+" : "+e)
			}
			return nil
		})
		if err != nil {
			t.Fatalf("parcours de %s/ : %v", sub, err)
		}
	}
	if len(violations) > 0 {
		t.Errorf("comparaison locale d'une distance a la portee du radar interdite — appeler "+
			"coordination.APortee (%s) :\n  %s", aPorteeHelper, strings.Join(violations, "\n  "))
	}
}
