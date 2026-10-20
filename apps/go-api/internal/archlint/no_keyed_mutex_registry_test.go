package archlint

// no_keyed_mutex_registry_test.go — UN VERROU PAR CLÉ S'OBTIENT PAR `platform/verrous`, JAMAIS PAR
// UN REGISTRE RÉÉCRIT À LA MAIN (règle des deux copies, CLAUDE.md n. 6).
//
// Le motif « une map de verrous par clé, créés à la demande, jamais retirés » vivait en trois
// exemplaires (bail d'écriture par chemin de base, passe post-sync par titre, indexation des
// médias par chemin) quand le rattrapage des zones nommées en a demandé un quatrième. Il est
// centralisé dans `verrous.Registre` ; ce garde-rail refuse ses deux formes manuscrites :
//
//	map[string]*sync.Mutex (ou *sync.RWMutex)    le registre à la main ;
//	LoadOrStore(…, &sync.Mutex{})                 sa variante en sync.Map.

import (
	"bufio"
	"bytes"
	"fmt"
	"os"
	"regexp"
	"sort"
	"strings"
	"testing"
)

// paquetDuRegistreDeVerrous : le seul endroit où le registre a le droit d'être écrit.
const paquetDuRegistreDeVerrous = "internal/platform/verrous/"

// formesManuscritesDuRegistre : les deux écritures du motif que le garde-rail refuse.
var formesManuscritesDuRegistre = []*regexp.Regexp{
	regexp.MustCompile(`map\[string\]\*sync\.(RW)?Mutex`),
	regexp.MustCompile(`LoadOrStore\(.*(&sync\.(RW)?Mutex\{\}|new\(sync\.(RW)?Mutex\))`),
}

// registresManuscrits rend les lignes de code (commentaires écartés) qui réécrivent le registre.
func registresManuscrits(rel string, src []byte) []string {
	var out []string
	sc := bufio.NewScanner(bytes.NewReader(src))
	sc.Buffer(make([]byte, 0, 1024*1024), 1024*1024)
	for n := 1; sc.Scan(); n++ {
		ligne := sc.Text()
		if i := strings.Index(ligne, "//"); i >= 0 {
			ligne = ligne[:i]
		}
		for _, forme := range formesManuscritesDuRegistre {
			if forme.MatchString(ligne) {
				out = append(out, fmt.Sprintf("%s:%d  %s", rel, n, strings.TrimSpace(ligne)))
				break
			}
		}
	}
	return out
}

// TestRegistreDeVerrousCentralise — LE GARDE-RAIL.
func TestRegistreDeVerrousCentralise(t *testing.T) {
	var violations []string
	visites := 0
	balayerLaProduction(t, []string{"internal", "cmd"}, func(rel, chemin string) error {
		if strings.HasPrefix(rel, paquetDuRegistreDeVerrous) {
			return nil
		}
		src, err := os.ReadFile(chemin) //nolint:gosec // chemin issu du parcours du module
		if err != nil {
			return err
		}
		visites++
		violations = append(violations, registresManuscrits(rel, src)...)
		return nil
	})
	if visites == 0 {
		t.Fatal("balayage muet : aucune source de production lue")
	}
	sort.Strings(violations)
	if len(violations) > 0 {
		t.Errorf("registre de verrous par clé réécrit à la main : %d.\n  - %s\n\nEmployer "+
			"verrous.Registre (internal/platform/verrous) : `var r verrous.Registre ; r.De(cle)`.",
			len(violations), strings.Join(violations, "\n  - "))
	}
}

// TestRegistreDeVerrousGardeRailMord — la preuve que le garde-rail mord, et qu'il ne mord pas sur
// un commentaire ni sur un verrou unique.
func TestRegistreDeVerrousGardeRailMord(t *testing.T) {
	refuses := map[string]string{
		"registre en map":      "var verrous = map[string]*sync.Mutex{}",
		"registre RW en map":   "type r struct{ m map[string]*sync.RWMutex }",
		"registre en sync.Map": "v, _ := m.LoadOrStore(cle, &sync.Mutex{})",
		"sync.Map et new":      "v, _ := m.LoadOrStore(cle, new(sync.Mutex))",
	}
	for nom, ligne := range refuses {
		if len(registresManuscrits("x.go", []byte(ligne))) == 0 {
			t.Errorf("le garde-rail n'a rien vu : %s", nom)
		}
	}
	acceptes := []string{
		"var mu sync.Mutex",
		"// autrefois map[string]*sync.Mutex",
		"var leases verrous.Registre",
	}
	for _, ligne := range acceptes {
		if v := registresManuscrits("x.go", []byte(ligne)); len(v) > 0 {
			t.Errorf("faux positif : %v", v)
		}
	}
}
