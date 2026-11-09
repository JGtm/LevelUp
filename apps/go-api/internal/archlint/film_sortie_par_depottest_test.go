package archlint

// film_sortie_par_depottest_test.go — UN TEST DE LA BRIQUE NE SORT DE LA BRIQUE QUE PAR
// `depottest.Racine` (ADR 0041/0043).
//
// `libs/film` est publiee seule, dans son depot. Un test qui lit `data/`, `apps/` ou `config/` du
// depot hote par une chaine de `..` y echoue (au lieu de sauter) : le chemin tombe sur rien. `libs/film/internal/depottest.Racine(t)` est le seul point de sortie : il rend la
// racine du depot hote, ou saute le test quand la brique est prise seule. Le garde-rail voisin
// `film_racine_depot_unique_test.go` interdit les autres remontees vers `go.work` ; celui-ci
// interdit les montees en `..` qui franchissent la racine de la brique.
//
// # CE QU'IL EXIGE
//
// Dans tout `_test.go` de `libs/film` (instruments `research` compris), une chaine de segments
// `".."` consecutifs — arguments successifs d'un `filepath.Join`, ou litteral `"../../x"` — ne
// compte pas plus de segments que le fichier n'a de dossiers sous `libs/film` : au-dela, le
// chemin sort de la brique. Un fichier de `libs/film/replay` peut ecrire `"..", "testdata"`
// (reste dans la brique), pas `"..", "..", "data"`. Allowlist vide.
//
// # CE QU'IL NE VOIT PAS
//
// Une remontee faite par une variable (`filepath.Join(racine, "..", "x")` avec `racine` lue dans
// l'environnement : la chaine est d'un segment) ou par `filepath.Dir(filepath.Dir(...))`.
//
// # MUTATION QUI DOIT LE FAIRE ROUGIR
//
// Dans un `_test.go` de `libs/film/replay`, ecrire
// `filepath.Join("..", "..", "..", "data", "titles")`.

import (
	"fmt"
	"regexp"
	"strings"
	"testing"
)

// chaineDeJoin : `".."` suivi d'autres `".."` separes par des virgules (arguments de Join).
var chaineDeJoin = regexp.MustCompile(`"\.\."(?:\s*,\s*"\.\.")*`)

// chaineDeLitteral : un litteral qui COMMENCE par `../` repete (`"../../x"`, `"../.."`).
var chaineDeLitteral = regexp.MustCompile(`"\.\.(?:/\.\.)*[/"]`)

// segmentsMontants rend la longueur de la plus longue chaine de `..` du code (commentaires exclus).
func segmentsMontants(code string) int {
	plus := 0
	for _, m := range chaineDeJoin.FindAllString(code, -1) {
		plus = max(plus, strings.Count(m, `".."`))
	}
	for _, m := range chaineDeLitteral.FindAllString(code, -1) {
		plus = max(plus, 1+strings.Count(m, "/.."))
	}
	return plus
}

// profondeurSousLaBrique : nombre de dossiers entre `libs/film` et le fichier
// (`libs/film/replay/x_test.go` -> 1).
func profondeurSousLaBrique(rel string) int {
	return strings.Count(strings.TrimPrefix(rel, racineBriqueFilm+"/"), "/")
}

func TestLesTestsDeLaBriqueSortentParDepottest(t *testing.T) {
	var violations []string
	vus := 0
	fichiersGo(t, surDisque(t, racineBriqueFilm), func(rel string, data []byte) {
		if !strings.HasSuffix(rel, "_test.go") {
			return
		}
		vus++
		if n, profondeur := segmentsMontants(stripComments(string(data))), profondeurSousLaBrique(rel); n > profondeur {
			violations = append(violations,
				fmt.Sprintf("%s : %d `..` consecutifs depuis un dossier a %d niveau(x) sous libs/film", rel, n, profondeur))
		}
	})
	if vus == 0 {
		t.Fatal("aucun _test.go sous libs/film : le garde-rail ne garde plus rien")
	}
	if len(violations) > 0 {
		t.Errorf("chemin qui sort de la brique par des `..` — appeler depottest.Racine(t) "+
			"(libs/film/internal/depottest) puis filepath.Join(racine, ...) :\n  %s",
			strings.Join(violations, "\n  "))
	}
}

// TestSegmentsMontantsVoitLesDeuxEcritures fige le comptage : sans lui, une regexp qui ne
// matche plus rien laisserait le garde-rail vert sur n'importe quel arbre.
func TestSegmentsMontantsVoitLesDeuxEcritures(t *testing.T) {
	cas := []struct {
		code string
		want int
	}{
		{`filepath.Join("..", "..", "..", "data", "titles")`, 3},
		{`filepath.Join("..", "testdata", "x")`, 1},
		{`const d = "../../../replay/testdata/x"`, 3},
		{`const d = "../.."`, 2},
		{`filepath.Join(racine, "..", "replays")`, 1},
		{`filepath.Join("data", "x")`, 0},
	}
	for _, c := range cas {
		if got := segmentsMontants(c.code); got != c.want {
			t.Errorf("segmentsMontants(%s) = %d, attendu %d", c.code, got, c.want)
		}
	}
	if profondeurSousLaBrique("libs/film/internal/grammar/x_test.go") != 2 || profondeurSousLaBrique("libs/film/x_test.go") != 0 {
		t.Error("profondeurSousLaBrique ne compte pas les dossiers sous libs/film")
	}
}
