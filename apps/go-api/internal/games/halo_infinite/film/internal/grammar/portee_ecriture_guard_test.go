package grammar

// portee_ecriture_guard_test.go — LA PORTEE `DAT_144e61ea0` N EST POSEE QUE PAR LA MARCHE D ETAT
// COMPLET.
//
// [Lecteur.portee] change la largeur de tous les lecteurs de position du moteur. Le jeu la pose dans
// ses lecteurs d etat complet et nulle part ailleurs : la boucle delta (`FUN_14076cb60`), le lecteur
// de record NEW (`FUN_1408f1aa4`) et la vue A ne la touchent pas. Ce ratchet tient la meme regle sur
// le paquet : une ECRITURE du champ n existe que dans `keyframe_fullstate_loop.go`, la marche d etat
// complet ([consumeFullStateDefaultBlock], [traverserSousLaPortee]), et dans l assistant de test
// `harnais_portee_test.go` ([sousLaPortee]). [TraverseEntity], [decodeDelta] et la vue A
// (`vue_a_charges*.go`) n en portent donc aucune.
//
// Mecanique : l AST de chaque fichier Go du paquet, tests et instruments compris ; une ecriture est
// une affectation dont une cible est un selecteur `.portee`, ou un litteral compose qui nomme le
// champ `portee`. Le compte de chaque fichier autorise est exact : une ecriture de plus, ou de moins,
// rougit aussi.
//
// Mutation jouee le 2026-10-08, rouge puis retiree : `br.portee = true` dans [TraverseEntity].

import (
	"go/ast"
	"go/parser"
	"go/token"
	"path/filepath"
	"sort"
	"testing"
)

// ecrituresDeLaPortee : les seuls fichiers qui ecrivent [Lecteur.portee], et combien de fois.
var ecrituresDeLaPortee = map[string]int{
	"keyframe_fullstate_loop.go": 4, // poser et retirer, autour de l etat par defaut et de la boucle
	"harnais_portee_test.go":     1, // l assistant de test
}

// TestLaPorteeNEstEcriteQueParLaMarcheDEtatComplet — LE RATCHET.
func TestLaPorteeNEstEcriteQueParLaMarcheDEtatComplet(t *testing.T) {
	fichiers, err := filepath.Glob("*.go")
	if err != nil || len(fichiers) == 0 {
		t.Fatalf("glob : %v (%d fichiers)", err, len(fichiers))
	}
	fset := token.NewFileSet()
	vues := map[string]int{}
	for _, f := range fichiers {
		af, err := parser.ParseFile(fset, f, nil, parser.SkipObjectResolution)
		if err != nil {
			t.Fatalf("%s : %v", f, err)
		}
		ast.Inspect(af, func(n ast.Node) bool {
			switch x := n.(type) {
			case *ast.AssignStmt:
				for _, cible := range x.Lhs {
					if sel, ok := cible.(*ast.SelectorExpr); ok && sel.Sel.Name == "portee" {
						vues[f]++
					}
				}
			case *ast.KeyValueExpr:
				if id, ok := x.Key.(*ast.Ident); ok && id.Name == "portee" {
					vues[f]++
				}
			}
			return true
		})
	}
	noms := make([]string, 0, len(vues))
	for f := range vues {
		noms = append(noms, f)
	}
	sort.Strings(noms)
	for _, f := range noms {
		if _, ok := ecrituresDeLaPortee[f]; !ok {
			t.Errorf("%s ecrit la portee DAT_144e61ea0 (%d fois) : seule la marche d etat complet la pose ; "+
				"un test passe par sousLaPortee", f, vues[f])
		}
	}
	for f, attendu := range ecrituresDeLaPortee {
		if vues[f] != attendu {
			t.Errorf("%s ecrit la portee %d fois, attendu %d", f, vues[f], attendu)
		}
	}
}
