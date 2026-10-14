package grammar

// portee_ecriture_guard_test.go — LA PORTEE `DAT_144e61ea0` ET L ETAT COMPLET NE SONT POSES QUE PAR
// LA MARCHE D ETAT COMPLET ET PAR L ASSISTANT DE RELECTURE A L ETENDUE.
//
// [Lecteur.portee] change la largeur de tous les lecteurs de position du moteur ; [Lecteur.etatComplet]
// dit au lecteur qu il lit un record sans masque (`FUN_142e2c690`). Le jeu les pose dans ses lecteurs
// d etat complet et nulle part ailleurs : la boucle delta (`FUN_14076cb60`), le lecteur de record NEW
// (`FUN_1408f1aa4`) et la vue A ne les touchent pas. Ce ratchet tient la meme regle sur le paquet :
// une ECRITURE de l un de ces champs n existe que dans
//   - `keyframe_fullstate_loop.go`, la marche d etat complet ([consumeFullStateDefaultBlock],
//     [traverserSousLaPortee]) ;
//   - `relecture_a_l_etendue.go`, l assistant unique qui RELIT une occurrence sous le cadre ou la
//     marche l a traversee ([relecteurDEtatComplet], lot D1.1 de 2.7.d1 : regle 6, la troisieme copie
//     du motif « lecteur, contexte, etat complet, portee, relecture » y est centralisee) ;
//   - `harnais_portee_test.go`, les deux assistants des tests unitaires de lecteurs ([sousLaPortee],
//     [enEtatComplet]).
// [TraverseEntity], [decodeDelta] et la vue A (`vue_a_charges*.go`) n en portent donc aucune, et une
// relecture ne recopie pas le motif.
//
// Mecanique : l AST de chaque fichier Go du paquet, tests et instruments compris ; une ecriture est
// une affectation dont une cible est un selecteur du champ, ou un litteral compose qui le nomme. Le
// compte de chaque fichier autorise est exact : une ecriture de plus, ou de moins, rougit aussi.
//
// Mutations jouees, rouges puis retirees : `br.portee = true` dans [TraverseEntity] (2026-10-08) ;
// `br.etatComplet = true` dans [decodeDelta] (2026-10-09).

import (
	"go/ast"
	"go/parser"
	"go/token"
	"path/filepath"
	"sort"
	"testing"
)

// ecrituresAutorisees : par champ, les seuls fichiers qui l ecrivent, et combien de fois.
var ecrituresAutorisees = map[string]map[string]int{
	"portee": {
		"keyframe_fullstate_loop.go": 4, // poser et retirer, autour de l etat par defaut et de la boucle
		"relecture_a_l_etendue.go":   1, // le relecteur d etat complet
		"harnais_portee_test.go":     1, // l assistant de test
	},
	"etatComplet": {
		"keyframe_fullstate_loop.go": 1, // la boucle de composants d etat complet
		"relecture_a_l_etendue.go":   1, // le relecteur d etat complet
		"harnais_portee_test.go":     1, // l assistant de test
	},
}

// ecrituresDuPaquet compte, par champ et par fichier, les ecritures des champs gardes.
func ecrituresDuPaquet(t *testing.T) map[string]map[string]int {
	t.Helper()
	fichiers, err := filepath.Glob("*.go")
	if err != nil || len(fichiers) == 0 {
		t.Fatalf("glob : %v (%d fichiers)", err, len(fichiers))
	}
	fset := token.NewFileSet()
	vues := map[string]map[string]int{}
	compter := func(champ, f string) {
		if _, garde := ecrituresAutorisees[champ]; !garde {
			return
		}
		if vues[champ] == nil {
			vues[champ] = map[string]int{}
		}
		vues[champ][f]++
	}
	for _, f := range fichiers {
		af, err := parser.ParseFile(fset, f, nil, parser.SkipObjectResolution)
		if err != nil {
			t.Fatalf("%s : %v", f, err)
		}
		ast.Inspect(af, func(n ast.Node) bool {
			switch x := n.(type) {
			case *ast.AssignStmt:
				for _, cible := range x.Lhs {
					if sel, ok := cible.(*ast.SelectorExpr); ok {
						compter(sel.Sel.Name, f)
					}
				}
			case *ast.KeyValueExpr:
				if id, ok := x.Key.(*ast.Ident); ok {
					compter(id.Name, f)
				}
			}
			return true
		})
	}
	return vues
}

// TestEtatCompletEtPorteeNeSontEcritsQueParLaMarche — LE RATCHET.
func TestEtatCompletEtPorteeNeSontEcritsQueParLaMarche(t *testing.T) {
	vues := ecrituresDuPaquet(t)
	for champ, autorises := range ecrituresAutorisees {
		noms := make([]string, 0, len(vues[champ]))
		for f := range vues[champ] {
			noms = append(noms, f)
		}
		sort.Strings(noms)
		for _, f := range noms {
			if _, ok := autorises[f]; !ok {
				t.Errorf("%s ecrit %s (%d fois) : seules la marche d etat complet et la relecture a "+
					"l etendue le posent ; un test passe par sousLaPortee / enEtatComplet", f, champ, vues[champ][f])
			}
		}
		for f, attendu := range autorises {
			if vues[champ][f] != attendu {
				t.Errorf("%s ecrit %s %d fois, attendu %d", f, champ, vues[champ][f], attendu)
			}
		}
	}
}
