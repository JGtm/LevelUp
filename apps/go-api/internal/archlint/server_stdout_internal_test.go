package archlint

// server_stdout_internal_test.go — la sortie standard du serveur est reservee a ses evenements.
//
// En -mode desktop la coquille lit stdout ligne a ligne comme des evenements JSON (ADR 0045 D-2).
// cmd/server tient deja la regle pour son propre code (stdout_guard_test.go) ; ce garde-rail la
// porte sur internal/ : aucun code non-test n'y reference os.Stdout ni n'appelle fmt.Print*. Le cas
// qui l'a motive : le lanceur d'enfants de cuisson (internal/replaychild) relayait la sortie de
// l'enfant sur os.Stdout du serveur. Un compte rendu va sur stderr (fmt.Fprint*(os.Stderr, ...)) ou
// dans le journal. Allowlist : "fichier" -> justification datee.

import (
	"go/ast"
	"testing"
)

var serverStdoutAllowlist = map[string]string{
	"internal/filmrun/runner.go": "2026-11-04 (lanceur de films venu de libs/film/filmproc) : EmitPeak et EmitRaison " +
		"sont appeles par l'ENFANT, dont la sortie standard est le tube que le parent lit et ne relaie pas ; " +
		"le defaut out=os.Stdout ne sert qu'aux outils de cmd/, le serveur passe os.Stderr (replaychild.go). " +
		"Retrait : le jour ou le protocole parent/enfant passe par un descripteur dedie.",
}

func TestInternalNEcritJamaisSurLaSortieStandard(t *testing.T) {
	for _, s := range goSourcesUnder(t, "internal") {
		if _, ok := serverStdoutAllowlist[s.Rel]; ok {
			continue
		}
		ast.Inspect(s.File, func(n ast.Node) bool {
			sel, ok := n.(*ast.SelectorExpr)
			if !ok {
				return true
			}
			pkg, ok := sel.X.(*ast.Ident)
			if !ok {
				return true
			}
			isStdout := pkg.Name == "os" && sel.Sel.Name == "Stdout"
			isPrint := pkg.Name == "fmt" && (sel.Sel.Name == "Print" || sel.Sel.Name == "Printf" || sel.Sel.Name == "Println")
			if isStdout || isPrint {
				t.Errorf("%s:%d : %s.%s ecrit sur la sortie standard du serveur, reservee aux evenements du "+
					"mode desktop : utiliser os.Stderr ou le journal", s.Rel, s.Fset.Position(sel.Pos()).Line, pkg.Name, sel.Sel.Name)
			}
			return true
		})
	}
}
