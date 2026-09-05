// tactical_pure_test.go — `internal/analysis/tactical` ET `internal/analysis/coordination`
// SONT PURS, ET ILS DOIVENT LE RESTER.
//
// # POURQUOI CE GARDE-RAIL EXISTE, ET IL A UNE DATE
//
// 2026-09-06, phase 1 de PLAN_TACTIQUE_2026-09-06. Ces deux paquets calculent les lectures de
// placement (ou je meurs, ou je tue, ou je gagne) et les mesures de coordination (l'echange).
// Leur en-tete annonce « aucune I/O, aucun SQL, aucun reseau » — et jusqu'a ce fichier, rien ne
// le verifiait. Trois interdits, chacun pour une raison qui se paie plus tard :
//
//   - `internal/analysis/replay` : le DOCUMENT de rejeu est un artefact, et une lecture
//     tactique doit rester calculable sans artefact (les positions mesurees de `kill_positions`
//     suffisent aux trois premieres lectures). L'importer ferait dependre le socle d'un
//     document qui n'existe que pour les matchs cuits, et l'onglet s'eteindrait pour tous les
//     autres. L'appelant PROJETTE ce qu'il a vers `domain.PositionSample` ;
//   - `internal/platform/duckdb` : un algo qui ouvre une base n'est plus testable a la main, et
//     il viole le modele mono-process (ADR 0013/0016) depuis un endroit ou personne ne
//     l'attend ;
//   - `database/sql` : la meme porte, un cran plus bas — c'est par la qu'un « juste une petite
//     requete » entrerait sans citer DuckDB.
//
// Le jour ou quelqu'un ajoutera l'un de ces imports, il l'apprendra ici, et non a la premiere
// page qui ne s'affiche plus faute d'artefact.
//
// # CE QU'IL VERIFIE, ET COMMENT
//
// Il PARSE les imports (go/parser, ImportsOnly) des fichiers non-test des deux paquets : un
// test grep se ferait tromper par un chemin cite en commentaire — et les deux en citent
// plusieurs, `analysis/replay` en tete. Les `_test.go` sont hors perimetre : un test peut avoir
// besoin d'une fixture sans que la production, elle, ne depende de rien.
package archlint

import (
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

// tacticalPurePkgs : les paquets purs, relatifs a apps/go-api.
var tacticalPurePkgs = []string{
	"internal/analysis/tactical",
	"internal/analysis/coordination",
}

// tacticalPureInterdits : les imports qui feraient de ces paquets autre chose que des algos.
var tacticalPureInterdits = []string{
	"levelup/go-api/internal/analysis/replay",
	"levelup/go-api/internal/platform/duckdb",
	"database/sql",
}

func TestTacticalEtCoordinationSontPurs(t *testing.T) {
	_, thisFile, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("runtime.Caller a echoue")
	}
	apiRoot := filepath.Dir(filepath.Dir(filepath.Dir(thisFile))) // internal/archlint -> internal -> apps/go-api

	fset := token.NewFileSet()
	var violations []string
	for _, pkg := range tacticalPurePkgs {
		pkgDir := filepath.Join(apiRoot, filepath.FromSlash(pkg))
		entries, err := os.ReadDir(pkgDir)
		if err != nil {
			t.Fatalf("paquet %s introuvable (%v) : s'il a DEMENAGE, deplacer ce garde-rail avec lui — "+
				"la contrainte de purete tient a la lecture sans artefact, pas au chemin", pkg, err)
		}

		fichiers := 0
		for _, e := range entries {
			name := e.Name()
			if e.IsDir() || !strings.HasSuffix(name, ".go") || strings.HasSuffix(name, "_test.go") {
				continue
			}
			fichiers++
			f, err := parser.ParseFile(fset, filepath.Join(pkgDir, name), nil, parser.ImportsOnly)
			if err != nil {
				t.Fatalf("analyse de %s/%s : %v", pkg, name, err)
			}
			for _, imp := range f.Imports {
				chemin := strings.Trim(imp.Path.Value, `"`)
				for _, interdit := range tacticalPureInterdits {
					if chemin == interdit || strings.HasPrefix(chemin, interdit+"/") {
						violations = append(violations, pkg+"/"+name+" -> "+chemin)
					}
				}
			}
		}
		if fichiers == 0 {
			t.Fatalf("aucun fichier non-test dans %s : ce garde-rail n'aurait plus d'objet", pkg)
		}
	}

	if len(violations) > 0 {
		t.Fatalf("le socle tactique n'est plus PUR — %d import(s) interdit(s) :\n  %s\n"+
			"Ces paquets ne lisent ni artefact ni base : l'appelant projette ce qu'il a vers "+
			"domain.PositionSample / domain.KillEvent et le leur passe. Une lecture tactique doit "+
			"rester calculable sans artefact de rejeu (cf. l'en-tete de ce fichier).",
			len(violations), strings.Join(violations, "\n  "))
	}
}
