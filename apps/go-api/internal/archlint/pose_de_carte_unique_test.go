package archlint

// pose_de_carte_unique_test.go — LE CONTEXTE DE CARTE D UNE CUISSON SE POSE PAR UN SEUL GESTE.
//
// # CE QUE CE RATCHET TIENT
//
// Le contexte de carte d une cuisson — les largeurs d axe de la carte du match posees sur le profil
// de balayage, puis le decoupage MPP du film — se pose par `grammar.FilmContext.PoserLaCarteEtLeDecoupage`
// (`grammar/contexte_de_carte.go`). La cuisson et la lecture des porteurs au sync l appellent
// (`replay.installWorldObjectPrecision`), et le ratchet de fermeture d image-cle en contexte de
// cuisson aussi : il ne peut donc pas mesurer un autre contexte que celui de la cuisson. Un appel de
// production a `PoserLargeursObjetDuMondeDepuisDecoupage` hors de la liste fermee ci-dessous est une
// COPIE de ce geste, qui divergerait en silence : il rougit (CLAUDE.md regle 6).
//
// Mecanique : l AST des fichiers Go non-test d `internal/` et de `cmd/` (parcours commun
// [parcourirSourcesProduction]), tout appel dont la fonction est un selecteur de ce nom. Les
// `_test.go` ne sont pas balayes : les instruments de mesure posent un decoupage detecte ou celui
// d une carte choisie, c est leur objet. Une entree de la liste qui ne porte plus d appel rougit aussi.
//
// Mutation jouee le 2026-10-08, rouge puis retiree : un appel
// `bal.PoserLargeursObjetDuMondeDepuisDecoupage(e.Layout())` ajoute a `replay/world_object_precision.go`.

import (
	"go/ast"
	"go/parser"
	"go/token"
	"sort"
	"testing"
)

// nomDeLaPoseDeCarte : la methode qui pose les largeurs d une carte sur un profil de balayage (et,
// sur un contexte, sa jumelle du meme nom).
const nomDeLaPoseDeCarte = "PoserLargeursObjetDuMondeDepuisDecoupage"

// posesDeCarteAutorisees — LISTE FERMEE ET DATEE des fichiers de production qui appellent la pose.
// Chemin relatif a apps/go-api.
var posesDeCarteAutorisees = map[string]string{
	"internal/games/halo_infinite/film/internal/grammar/contexte_de_carte.go": "2026-10-08 — LE geste " +
		"unique du contexte de carte de la cuisson, du sync et du ratchet de fermeture en contexte de cuisson",
	"internal/games/halo_infinite/film/internal/grammar/film_context.go": "2026-10-08 — la methode du " +
		"contexte qui pose un decoupage DONNE, et l enveloppe D2 `contexteDeBobine`, qui y pose celui LU DANS " +
		"LE FILM (hors cuisson : la cuisson prend le catalogue de la carte)",
	"internal/games/halo_infinite/film/internal/facts/killsource/decode.go": "2026-10-08 — EXCEPTION : " +
		"killsource pose la carte sur son PROFIL DE DEPART, avant la calibration (`ProfilDeDepartPourCarte`), " +
		"pas sur un contexte ; la faire passer par le geste unique changerait l ordre de ses gestes et son " +
		"perimetre hache (`killsource.Rev`), hors du lot qui a pose ce ratchet (plan LK, decision E-5)",
}

// TestLaCarteSePoseParUnSeulGeste — LE RATCHET.
func TestLaCarteSePoseParUnSeulGeste(t *testing.T) {
	racine := racineGoAPI(t)
	fset := token.NewFileSet()
	appels := map[string]int{}
	parcourirSourcesProduction(t, racine, func(rel, chemin string) {
		f, err := parser.ParseFile(fset, chemin, nil, parser.SkipObjectResolution)
		if err != nil {
			t.Fatalf("%s : %v", rel, err)
		}
		ast.Inspect(f, func(n ast.Node) bool {
			if call, ok := n.(*ast.CallExpr); ok {
				if sel, ok := call.Fun.(*ast.SelectorExpr); ok && sel.Sel.Name == nomDeLaPoseDeCarte {
					appels[rel]++
				}
			}
			return true
		})
	})
	fichiers := make([]string, 0, len(appels))
	for rel := range appels {
		fichiers = append(fichiers, rel)
	}
	sort.Strings(fichiers)
	for _, rel := range fichiers {
		if _, ok := posesDeCarteAutorisees[rel]; !ok {
			t.Errorf("%s pose les largeurs d une carte sur un profil (%d appel(s) a %s) hors de la liste "+
				"fermee : c est une copie du contexte de carte de la cuisson. Appeler "+
				"grammar.FilmContext.PoserLaCarteEtLeDecoupage, ou justifier l exception, datee, dans "+
				"posesDeCarteAutorisees.", rel, appels[rel], nomDeLaPoseDeCarte)
		}
	}
	for rel := range posesDeCarteAutorisees {
		if appels[rel] == 0 {
			t.Errorf("%s est dans posesDeCarteAutorisees mais n appelle plus %s : retirer l entree",
				rel, nomDeLaPoseDeCarte)
		}
	}
}
