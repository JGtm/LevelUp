// Package archlint — no_local_portee_comparison_test.go : garde-rail de la comparaison « une
// distance au coequipier le plus proche est-elle a portee du radar ? » (CLAUDE.md regle 6).
//
// La regle tient en une ligne — une distance MESUREE, a la portee ou en deca (borne INCLUSIVE) ;
// une distance absente n'est jamais a portee — et sa SOURCE UNIQUE est la fonction
// `coordination.APortee` (`internal/analysis/coordination/vies_pres_ou_seul.go`). Toutes les
// lectures qui rangent une mort « pres » ou « seule » la lisent : l'isolement de la Tactique, les
// vies pres d'un coequipier, le detail d'une zone et son badge de placement. Une copie qui ecrirait
// `<` au lieu de `<=`, son contraire (`d >= rayon`, `!(d < rayon)`), ou qui rangerait une distance
// absente a portee, ferait dire « pres » a une page et « seul » a l'autre pour la meme mort.
//
// Les empreintes, telles qu'une copie s'ecrit :
//  1. une DISTANCE comparee a une portee ou un rayon par `<`, `<=`, `>` ou `>=`, dans un sens ou
//     dans l'autre ; une distance est une valeur DEREFERENCEE (`*m.PlusProcheM`), un identifiant
//     lie a une valeur dereferencee (`d := *m.PlusProcheM`) ou un identifiant teste `!= nil` dans
//     le meme fichier ;
//  2. le test de presence de la distance au plus proche (`PlusProcheM != nil &&`).
//
// Fichiers de production seulement (un test peut fabriquer ses distances comme il veut). Seul le
// corps de la fonction `APortee` est exclu, pas son fichier : une copie posee a cote du helper
// est une copie. L'auto-test prouve que l'empreinte reconnait les formes connues d'une copie.
package archlint

import (
	"os"
	"path/filepath"
	"regexp"
	"runtime"
	"strings"
	"testing"
)

// aPorteeHelper : le fichier de la source unique (cite par le message d'erreur).
const aPorteeHelper = "internal/analysis/coordination/vies_pres_ou_seul.go"

var (
	// corpsAPortee : la fonction source unique, seule exclue du balayage.
	corpsAPortee = regexp.MustCompile(`(?s)func APortee\([^)]*\)[^{]*\{.*?\n\}`)
	// distanceAvantPortee : `<operande> <op> <portee>` ; distanceApresPortee : l'inverse.
	// La portee peut etre lue sur un champ (`regle.RayonM`) et passer par une conversion
	// (`float64(rayon)`).
	distanceAvantPortee = regexp.MustCompile(`(\*?\s*[\w.]+)\s*(?:<=|>=|<|>)\s*(?:\w+\()?[\w.]*(?i:rayon|portee)\w*`)
	distanceApresPortee = regexp.MustCompile(`[\w.]*(?i:rayon|portee)\w*\)?\s*(?:<=|>=|<|>)\s*(\*?\s*[\w.]+)`)
	// lieeADeref : un identifiant lie a une valeur dereferencee, avec ou sans son type
	// (`d := *m.PlusProcheM`, `var d float64 = *m.PlusProcheM`).
	lieeADeref = regexp.MustCompile(`(?:\bvar\s+)?(\w+)(?:\s+[\w.]+)?\s*:?=\s*\*\s*[\w.]+`)
	// testeeNonNil : un identifiant teste `!= nil`.
	testeeNonNil = regexp.MustCompile(`(\w+)\s*!=\s*nil`)
	// presencePlusProche : empreinte 2.
	presencePlusProche = regexp.MustCompile(`PlusProcheM\s*!=\s*nil\s*&&`)
)

// distancesDuFichier rend les identifiants qu'un fichier traite comme des distances : lies a une
// valeur dereferencee, ou testes `!= nil`.
func distancesDuFichier(source string) map[string]bool {
	out := map[string]bool{}
	for _, re := range []*regexp.Regexp{lieeADeref, testeeNonNil} {
		for _, m := range re.FindAllStringSubmatch(source, -1) {
			out[m[1]] = true
		}
	}
	return out
}

// estUneDistance dit si l'operande d'une comparaison est une distance au sens de l'empreinte 1.
func estUneDistance(operande string, distances map[string]bool) bool {
	operande = strings.TrimSpace(operande)
	if strings.HasPrefix(operande, "*") {
		return true
	}
	return distances[operande]
}

// empreintesDeComparaison rend les empreintes de comparaison locale presentes dans un texte, le
// corps de `APortee` mis a part.
func empreintesDeComparaison(source string) []string {
	source = corpsAPortee.ReplaceAllString(source, "")
	distances := distancesDuFichier(source)
	var out []string
	compare := false
	for _, re := range []*regexp.Regexp{distanceAvantPortee, distanceApresPortee} {
		for _, m := range re.FindAllStringSubmatch(source, -1) {
			if estUneDistance(m[1], distances) {
				compare = true
			}
		}
	}
	if compare {
		out = append(out, "distance comparee a une portee")
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
		// les autres operateurs, et le sens inverse
		"\tseul := *m.PlusProcheM > rayon",
		"\tif rayonRadar < *mort.PlusProcheM {",
		// le contraire de la regle, sur une distance deja dereferencee (revue ronde 1, G1)
		"\td := *m.PlusProcheM\n\tbadge := Badge{Seul: d >= rayon}",
		"\td := *mort.PlusProcheM\n\tpres := !(d < portee)",
		// une distance testee non nulle puis comparee
		"\tif d != nil {\n\t\tpres = !(*d < rayon)\n\t}",
		// les trois formes de la revue, ronde 2 : portee lue sur un champ, portee convertie, distance
		// declaree avec son type
		"\tseul := *m.PlusProcheM > regle.RayonM",
		"\tif *d <= float64(rayon) {",
		"\tvar d float64 = *m.PlusProcheM\n\tif d >= rayon {",
		// une copie posee dans le fichier du helper, hors de la fonction
		"func APortee(d *float64, rayon float64) bool {\n\treturn d != nil && *d <= rayon\n}\n\nfunc seul(d *float64, rayon float64) bool {\n\treturn d == nil || *d > rayon\n}",
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
		"\t\t\td := distanceAZone(x, y, z)\n\t\t\tif d >= RayonZoneM {",
		"\tif !connu || rayon <= 0 {",
		// le helper lui-meme
		"func APortee(d *float64, rayon float64) bool {\n\treturn d != nil && *d <= rayon\n}",
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
