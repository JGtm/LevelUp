package grammar

// generations_vivantes_datees_ratchet_test.go — LE FILTRE DE GENERATION ATEMPOREL N A PLUS QUE DES
// APPELANTS NOMMES (lot R2-bis du PLAN_SUITE_AUDIT_DECODEUR_FILM_2026-09-25, 2026-09-29).
//
// # CE QUE CE RATCHET TIENT
//
// [FilmContext.GenerationsVivantes] rend le masque ATEMPOREL des corps du film : il dit qu un corps
// (slot, generation) a existe, pas QUAND. Un lecteur de records delta bipedes qui le passe tel quel
// accepte l en-tete d un corps AVANT le record de creation de ce corps — la replication d aucun
// corps (constat C2 du G-corpus J11.1, corrige pour les positions au lot R2 ; decouverte 2 du lot R2
// pour les huit canaux, la recuperation d equipement et la visee seule, corrigee ici). Les lecteurs
// prennent donc leur filtre par [FilmContext.GenerationsVivantesA] (l instant du paquet porteur).
//
// Le test releve, par l AST des sources de PRODUCTION de l arbre `film/` (hors `_test.go` et hors
// `research/`), chaque appel `.GenerationsVivantes()` et sa fonction englobante, et exige que
// l ensemble soit EXACTEMENT la liste ci-dessous — un appelant neuf rougit, un appelant retire aussi
// (la liste se met a jour et ne peut que diminuer). Chaque appelant legitime porte sa raison, et le
// test verifie la condition qui la rend vraie :
//
//   - `ScanBipedPositionsForBand` (offline_biped_band.go) : filtre par defaut des options, DATE
//     paquet par paquet par `scanBipedChunks` (`opt.Generations.A(pk.TimestampUS)`, lot R2) ;
//   - `etageDuPont.lire` (pont_identite.go) : l etage le passe au balayage des positions (qui le
//     date, ci-dessus) et le rend a la cuisson, qui n en tire que le compte du repli nomme
//     (`SlotsEnRepli`, aucun record juge) — verifie : hors de `grammar`, un filtre ne sert qu a
//     `SlotsEnRepli` ou `Connue` ;
//   - `GenerationsVivantesA` (generations_vivantes.go) : le definisseur de l acces date.
//
// # MUTATIONS JOUEES (2026-09-29), ROUGES, PUIS RETIREES — consignees au rapport du lot R2-bis.

import (
	"go/ast"
	"go/parser"
	"go/token"
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"slices"
	"sort"
	"strings"
	"testing"
)

// appelantsDuFiltreAtemporel : LA LISTE NOMMEE, DATEE (2026-09-29, lot R2-bis). Cle : chemin relatif
// a `film/` (separateurs `/`) + « : » + fonction englobante.
var appelantsDuFiltreAtemporel = map[string]string{
	"internal/grammar/offline_biped_band.go:balayerLesPositions":    "defaut des options, date par paquet dans scanBipedChunks (lot R2) ; l ancrage du contexte date le sien par GenerationsVivantesA (corps de ScanBipedPositionsForBand depuis le lot 2.4 de la representation intermediaire)",
	"internal/grammar/pont_identite.go:lire":                        "etage du pont : positions (datees) et compte du repli SlotsEnRepli (aucun record juge)",
	"internal/grammar/generations_vivantes.go:GenerationsVivantesA": "definisseur de l acces date",
}

// racineDuFilm : l arbre `film/`, depuis ce paquet (`film/internal/grammar`).
const racineDuFilm = "../.."

func TestFiltreDeGenerationAtemporel_AppelantsNommes(t *testing.T) {
	var vus []string
	for _, f := range sourcesDeProductionDuFilm(t) {
		fset := token.NewFileSet()
		af, err := parser.ParseFile(fset, filepath.Join(racineDuFilm, f), nil, parser.SkipObjectResolution)
		if err != nil {
			t.Fatalf("%s : %v", f, err)
		}
		for _, d := range af.Decls {
			fn, ok := d.(*ast.FuncDecl)
			if !ok || fn.Body == nil {
				continue
			}
			ast.Inspect(fn.Body, func(n ast.Node) bool {
				c, ok := n.(*ast.CallExpr)
				if !ok || len(c.Args) != 0 {
					return true
				}
				if s, ok := c.Fun.(*ast.SelectorExpr); ok && s.Sel.Name == "GenerationsVivantes" {
					vus = append(vus, f+":"+fn.Name.Name)
				}
				return true
			})
		}
	}
	sort.Strings(vus)
	vus = slices.Compact(vus)
	var attendus []string
	for k := range appelantsDuFiltreAtemporel {
		attendus = append(attendus, k)
	}
	sort.Strings(attendus)
	if strings.Join(vus, "\n") != strings.Join(attendus, "\n") {
		t.Errorf("appelants de production de FilmContext.GenerationsVivantes() (filtre ATEMPOREL) :\n  %s\n"+
			"attendus :\n  %s\nun lecteur de records delta bipedes prend son filtre par "+
			"FilmContext.GenerationsVivantesA(pk.TimestampUS) (lot R2-bis) ; un appelant retire sort de la liste",
			strings.Join(vus, "\n  "), strings.Join(attendus, "\n  "))
	}
}

// TestFiltreDeGenerationAtemporel_RaisonsTenues verifie la condition qui rend chaque appelant nomme
// legitime.
func TestFiltreDeGenerationAtemporel_RaisonsTenues(t *testing.T) {
	band := lireSource(t, "internal/grammar/offline_biped_band.go")
	if !strings.Contains(band, "o.Generations = opt.Generations.A(pk.TimestampUS)") {
		t.Errorf("offline_biped_band.go ne date plus le filtre par paquet (lot R2) : son appel atemporel " +
			"n est plus legitime")
	}
	// Hors de `grammar`, le filtre rendu par l etage ne juge aucun record.
	usage := regexp.MustCompile(`\.Generations\.(\w+)\(`)
	permis := map[string]bool{"SlotsEnRepli": true, "Connue": true}
	for _, f := range sourcesDeProductionDuFilm(t) {
		if strings.HasPrefix(f, "internal/grammar/") {
			continue
		}
		for _, m := range usage.FindAllStringSubmatch(lireSource(t, f), -1) {
			if !permis[m[1]] {
				t.Errorf("%s : .Generations.%s( — hors de grammar, le filtre de l etage du pont ne sert qu au "+
					"compte du repli (SlotsEnRepli, Connue) ; juger un record y demande le filtre date", f, m[1])
			}
		}
	}
}

// sourcesDeProductionDuFilm rend les `.go` de production de l arbre `film/`, relatifs a lui.
func sourcesDeProductionDuFilm(t *testing.T) []string {
	t.Helper()
	var out []string
	err := filepath.WalkDir(racineDuFilm, func(p string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		rel, _ := filepath.Rel(racineDuFilm, p)
		rel = filepath.ToSlash(rel)
		if d.IsDir() {
			if rel == "research" || d.Name() == "testdata" {
				return filepath.SkipDir
			}
			return nil
		}
		if strings.HasSuffix(rel, ".go") && !strings.HasSuffix(rel, "_test.go") {
			out = append(out, rel)
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(out) < 100 {
		t.Fatalf("%d source(s) de production sous %s : la racine est-elle la bonne ?", len(out), racineDuFilm)
	}
	return out
}

func lireSource(t *testing.T, rel string) string {
	t.Helper()
	b, err := os.ReadFile(filepath.Join(racineDuFilm, rel)) //nolint:gosec // sources du depot
	if err != nil {
		t.Fatal(err)
	}
	return string(b)
}
