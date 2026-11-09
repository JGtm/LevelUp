package replay

// sieges_et_occupants_test.go — LE SYNC ET LA CUISSON NOMMENT AVEC LE MEME REGISTRE.
//
//	SO-UNIQUE   la table des sieges du film et les occupants par entite ne se lisent, en production,
//	            que dans [lireSiegesEtOccupants], et ses DEUX appelants sont l'etage de balayage de la
//	            cuisson et le document interne du sync ;
//	SO-OPTIONS  le document interne pose les quatre lectures dans les options de l'assemblage — celles
//	            que la cuisson y pose par `FilmInputs.applyTo`.
//
// La parite sur de vrais films (le sync et la cuisson nomment les memes vies) est mesuree par
// `sync/killcollector/porteurs_au_sync_parite_research_test.go`.

import (
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"slices"
	"strings"
	"testing"

	"github.com/JGtm/LevelUp/libs/film/internal/grammar"
)

// lecturesDuNommage : les deux lectures du film que [lireSiegesEtOccupants] est seule a appeler.
var lecturesDuNommage = []string{"ScanFilmPlayerTable", "ScanPlayerTeams"}

// appelsParFonction rend, pour chaque fonction de production du paquet, les noms simples qu'elle
// appelle.
func appelsParFonction(t *testing.T) map[string][]string {
	t.Helper()
	entrees, err := os.ReadDir(".")
	if err != nil {
		t.Fatalf("lecture du paquet : %v", err)
	}
	fset := token.NewFileSet()
	out := map[string][]string{}
	for _, e := range entrees {
		nom := e.Name()
		if !strings.HasSuffix(nom, ".go") || strings.HasSuffix(nom, "_test.go") {
			continue
		}
		f, err := parser.ParseFile(fset, nom, nil, 0)
		if err != nil {
			t.Fatalf("analyse de %s : %v", nom, err)
		}
		for _, d := range f.Decls {
			fn, ok := d.(*ast.FuncDecl)
			if !ok || fn.Body == nil {
				continue
			}
			ast.Inspect(fn.Body, func(n ast.Node) bool {
				if call, ok := n.(*ast.CallExpr); ok {
					if appele, ok := nomAppele(call); ok {
						out[fn.Name.Name] = append(out[fn.Name.Name], appele)
					}
				}
				return true
			})
		}
	}
	return out
}

// TestLeSyncEtLaCuissonLisentLesMemesSiegesEtOccupants (SO-UNIQUE). Mutations vues rouges : rappeler
// `grammar.ScanPlayerTeams` dans la garde du drapeau du sync ; lire la table du film a part dans
// `balayerPont` ; ne plus appeler [lireSiegesEtOccupants] depuis `DocumentInterne`.
func TestLeSyncEtLaCuissonLisentLesMemesSiegesEtOccupants(t *testing.T) {
	var appelants []string
	for fn, appels := range appelsParFonction(t) {
		for _, lecture := range lecturesDuNommage {
			if fn != "lireSiegesEtOccupants" && slices.Contains(appels, lecture) {
				t.Errorf("%s appelle %s hors de lireSiegesEtOccupants : le sync et la cuisson ne liraient "+
					"plus le meme registre", fn, lecture)
			}
		}
		if slices.Contains(appels, "lireSiegesEtOccupants") {
			appelants = append(appelants, fn)
		}
	}
	slices.Sort(appelants)
	if want := []string{"DocumentInterne", "balayerPont"}; !slices.Equal(appelants, want) {
		t.Errorf("appelants de lireSiegesEtOccupants = %v, attendu %v (cuisson et document interne du sync)",
			appelants, want)
	}
}

// TestLeDocumentInternePoseLesSiegesEtLesOccupants (SO-OPTIONS). Mutation vue rouge : retirer un des
// quatre champs de `optionsDuRegistre`.
func TestLeDocumentInternePoseLesSiegesEtLesOccupants(t *testing.T) {
	so := siegesEtOccupants{
		table:   grammar.FilmPlayerTable{Occupied: 3},
		equipes: map[int]int{8: 1},
		rapport: grammar.TeamScanReport{Read: 4},
		entites: grammar.PlayerEntityScan{Scanned: true, Entities: []grammar.PlayerEntity{{Slot: 1550, Index: 8}}},
	}
	opt := EntreePorteursAuSync{}.optionsDuRegistre(nil, so)
	if opt.FilmTable.Occupied != 3 || opt.PlayerTeams[8] != 1 || opt.TeamScan.Read != 4 ||
		!opt.PlayerEntities.Scanned || len(opt.PlayerEntities.Entities) != 1 {
		t.Fatalf("options du document interne : table %+v, equipes %v, rapport %+v, entites %+v — attendu "+
			"les quatre lectures de lireSiegesEtOccupants", opt.FilmTable, opt.PlayerTeams, opt.TeamScan,
			opt.PlayerEntities)
	}
}
