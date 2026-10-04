package grammar

// marche_images_cles_unique_test.go — UN SEUL PILOTAGE DE LA PHASE DES IMAGES-CLES (ADR 0037 IR-3 ;
// lot 2.2 du plan de l etape 2).
//
// Les lecteurs d image-cle de la cuisson pilotaient chacun leur boucle « chunks -> paquets
// d image-cle -> marche d ancres ». Depuis le lot 2.2, ils sont des canaux de la phase des
// images-cles ([Distribuer]) : le seul fichier de production du paquet qui choisit les paquets
// d image-cle est `marche_images_cles.go`. Les sites qui restent sont ceux que le lot n a pas
// convertis (decouverte 7 du plan de l etape 2) ; la liste ne fait que descendre.

import (
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"sort"
	"strings"
	"testing"
)

// fichierDeLaPhase est le fichier de production qui pilote la phase des images-cles.
const fichierDeLaPhase = "marche_images_cles.go"

// pilotagesRestants : les fichiers de production qui choisissent encore eux-memes les paquets
// d image-cle, avec la raison (2026-10-03). Une entree se RETIRE quand son site est converti ; une
// entree morte fait rougir le test.
var pilotagesRestants = map[string]string{
	"birth_loadouts.go":          "la marche des naissances (sa marche d ancres et `LierTableDeDatums`) : lot 3.1",
	"keyframe_datums.go":         "`LierTableDeDatums`, la liaison de la marche des naissances : lot 3.1",
	"keyframe_ground_weapons.go": "enveloppe D2 des armes au sol, sans appelant de production",
	"navpoint_radial_scan.go":    "l anneau de la bombe, etat complet de ti=12 : lot 3.1",
	"object_deaths.go":           "`marchPacketsOf`, la marche des morts d objet : lot LU de la campagne de grammaire",
	"objective_scan.go":          "les objectifs, etat complet de ti=11 : lot 3.1",
}

// TestLaPhaseDesImagesClesEstPiloteeEnUnSeulEndroit : dans le code de production du paquet, seuls
// `marche_images_cles.go` et les sites restants nommes choisissent les paquets d image-cle.
// MUTATION — une boucle `if pk.Type != PacketTypeKeyframe` dans un autre fichier de production :
// ROUGE.
func TestLaPhaseDesImagesClesEstPiloteeEnUnSeulEndroit(t *testing.T) {
	vus := fichiersQuiChoisissentLesImagesCles(t)
	if !vus[fichierDeLaPhase] {
		t.Fatalf("%s ne choisit plus les paquets d image-cle : le garde-rail ne garde plus rien", fichierDeLaPhase)
	}
	var enTrop, morts []string
	for f := range vus {
		if _, ok := pilotagesRestants[f]; !ok && f != fichierDeLaPhase {
			enTrop = append(enTrop, f)
		}
	}
	for f := range pilotagesRestants {
		if !vus[f] {
			morts = append(morts, f)
		}
	}
	sort.Strings(enTrop)
	sort.Strings(morts)
	if len(enTrop) > 0 {
		t.Errorf("pilotage des images-cles recopie hors de %s : %s — un lecteur d image-cle est un canal "+
			"de la phase des images-cles (`Distribuer`)", fichierDeLaPhase, strings.Join(enTrop, ", "))
	}
	if len(morts) > 0 {
		t.Errorf("sites restants qui ne pilotent plus les images-cles : %s — les retirer de la liste",
			strings.Join(morts, ", "))
	}
}

// fichiersQuiChoisissentLesImagesCles rend les fichiers de production du paquet qui comparent un
// type de paquet a [PacketTypeKeyframe] (comparaison ou cas d un `switch`).
func fichiersQuiChoisissentLesImagesCles(t *testing.T) map[string]bool {
	t.Helper()
	entrees, err := os.ReadDir(".")
	if err != nil {
		t.Fatalf("lecture du paquet : %v", err)
	}
	fset := token.NewFileSet()
	out := map[string]bool{}
	for _, e := range entrees {
		nom := e.Name()
		if e.IsDir() || !strings.HasSuffix(nom, ".go") || strings.HasSuffix(nom, "_test.go") {
			continue
		}
		f, err := parser.ParseFile(fset, nom, nil, parser.SkipObjectResolution)
		if err != nil {
			t.Fatalf("analyse de %s : %v", nom, err)
		}
		ast.Inspect(f, func(n ast.Node) bool {
			switch x := n.(type) {
			case *ast.BinaryExpr:
				if estLeTypeImageCle(x.X) || estLeTypeImageCle(x.Y) {
					out[nom] = true
				}
			case *ast.CaseClause:
				for _, v := range x.List {
					if estLeTypeImageCle(v) {
						out[nom] = true
					}
				}
			}
			return true
		})
	}
	return out
}

// estLeTypeImageCle dit si l expression est l identifiant [PacketTypeKeyframe].
func estLeTypeImageCle(e ast.Expr) bool {
	id, ok := e.(*ast.Ident)
	return ok && id.Name == "PacketTypeKeyframe"
}
