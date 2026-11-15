// Package archlint — outils_sans_cgo_test.go : les outils que le gate du corpus des rejeux
// compile sans cgo n'importent pas DuckDB.
//
// POURQUOI. `cmd/replay-corpus-gate` compile `cmd/replay-build` et `cmd/replay-verite` à la
// révision de base avec CGO_ENABLED=0. Le pilote DuckDB exige cgo : un import qui le tire, même de
// loin (`internal/config` importe `internal/platform/duckdb`), casse cette compilation, et rien
// d'autre ne le dit, puisque la compilation ordinaire et la CI gardent cgo. Ce test lit la
// fermeture des imports de ces outils et refuse le pilote.
//
// SE PROUVER ROUGE. Importer `levelup/go-api/internal/config` dans `cmd/replay-build/main.go`.
package archlint

import (
	"os/exec"
	"strings"
	"testing"
)

// outilsCompilesSansCgo : les commandes que `cmd/replay-corpus-gate` compile avec CGO_ENABLED=0.
var outilsCompilesSansCgo = []string{"./cmd/replay-build", "./cmd/replay-verite"}

// modulePiloteDuckDB est le préfixe des paquets du pilote DuckDB, qui ne se compile qu'avec cgo.
const modulePiloteDuckDB = "github.com/duckdb/"

func TestOutilsSansCgoNImportentPasDuckDB(t *testing.T) {
	goBin, err := exec.LookPath("go")
	if err != nil {
		t.Skipf("go introuvable dans le PATH : %v", err)
	}
	for _, outil := range outilsCompilesSansCgo {
		args := []string{"list", "-deps", "-f", "{{.ImportPath}}", outil}
		cmd := exec.Command(goBin, args...)
		cmd.Dir = apiRootDepuisIci(t)
		var stderr strings.Builder
		cmd.Stderr = &stderr
		brut, err := cmd.Output()
		if err != nil {
			t.Fatalf("go %s : %v\n%s", strings.Join(args, " "), err, stderr.String())
		}
		paquets := strings.Fields(string(brut))
		if len(paquets) < 20 {
			t.Fatalf("%s : fermeture de %d paquets, la sortie de go list n'est plus celle qu'on lit", outil, len(paquets))
		}
		for _, paquet := range paquets {
			if strings.HasPrefix(paquet, modulePiloteDuckDB) {
				t.Errorf("%s importe %s : cet outil est compilé sans cgo par cmd/replay-corpus-gate et ne "+
					"peut pas tirer DuckDB (go list -deps %s pour trouver le chemin d'import)", outil, paquet, outil)
			}
		}
	}
}
