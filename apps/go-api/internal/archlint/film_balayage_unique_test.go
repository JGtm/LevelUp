package archlint

// film_balayage_unique_test.go — LE GARDE-RAIL DU BALAYAGE COMMUN DES GARDE-RAILS DU DECODEUR (regle
// des deux copies, CLAUDE.md n. 6).
//
// Le parcours de la production du decodeur qui ecarte les sous-arbres de recherche
// ([repertoireExcluDuTriTotal]) s ecrit en UN seul endroit : [balayerLaProduction]
// (`film_balayage_test.go`). Ce test interdit, dans les tests d archlint, tout autre fichier qui
// combine un `filepath.WalkDir(` et un appel a `repertoireExcluDuTriTotal(` — la forme recopiee que
// le helper a remplacee —, et exige que l hote la porte.

import (
	"os"
	"path/filepath"
	"regexp"
	"runtime"
	"strings"
	"testing"
)

// hoteBalayage : le seul fichier qui parcourt la production en ecartant les sous-arbres de recherche.
const hoteBalayage = "film_balayage_test.go"

var (
	formeParcours  = regexp.MustCompile(`filepath\.WalkDir\(`)
	formeExclusion = regexp.MustCompile(`repertoireExcluDuTriTotal\(`)
)

// recopieLeBalayage dit si une source combine le parcours et l exclusion des sous-arbres de recherche.
func recopieLeBalayage(src []byte) bool {
	return formeParcours.Match(src) && formeExclusion.Match(src)
}

// TestBalayageDeLaProductionUnique interdit les copies du balayage commun hors de son hote.
func TestBalayageDeLaProductionUnique(t *testing.T) {
	_, ici, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("runtime.Caller a echoue")
	}
	dir := filepath.Dir(ici)
	entrees, err := os.ReadDir(dir)
	if err != nil {
		t.Fatalf("lecture de %s : %v", dir, err)
	}
	vus, hote := 0, false
	for _, e := range entrees {
		nom := e.Name()
		if e.IsDir() || !strings.HasSuffix(nom, "_test.go") || nom == "film_balayage_unique_test.go" {
			continue
		}
		src, err := os.ReadFile(filepath.Join(dir, nom))
		if err != nil {
			t.Fatalf("lecture de %s : %v", nom, err)
		}
		vus++
		if !recopieLeBalayage(src) {
			continue
		}
		if nom == hoteBalayage {
			hote = true
			continue
		}
		t.Errorf("%s : parcours de la production recopie (filepath.WalkDir + repertoireExcluDuTriTotal) — "+
			"appeler balayerLaProduction (%s)", nom, hoteBalayage)
	}
	if vus < 50 {
		t.Fatalf("balayage muet : %d tests d archlint vus", vus)
	}
	if !hote {
		t.Errorf("%s ne porte plus le balayage — deplacer le garde-rail avec lui", hoteBalayage)
	}
}

// TestGardeRailBalayageVecteurs : la forme recopiee rougit ; un appel au helper ou un parcours sans
// l exclusion passent.
func TestGardeRailBalayageVecteurs(t *testing.T) {
	for _, c := range []struct {
		src     string
		recopie bool
	}{
		{"err := filepath.WalkDir(base, f)\nif repertoireExcluDuTriTotal(d.Name(), rel) {", true},
		{"balayerLaProduction(t, racines, visiter)", false},
		{"err := filepath.WalkDir(base, f)", false},
	} {
		if got := recopieLeBalayage([]byte(c.src)); got != c.recopie {
			t.Errorf("%q : recopie %v, attendu %v", c.src, got, c.recopie)
		}
	}
}
