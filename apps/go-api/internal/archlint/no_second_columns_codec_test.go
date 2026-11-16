// Package archlint — no_second_columns_codec_test.go : un seul encodeur de la forme en colonnes du
// rejeu côté Go.
//
// POURQUOI. La forme en colonnes est un contrat entre le serveur et le web : ses marqueurs (`$enc`
// à la racine, `$c` sur une table) sont ce que le décodeur reconnaît. Un second encodeur, ou un
// décodeur Go de production, ferait diverger les deux côtés sans qu'aucun test ne le voie.
//
// RÈGLE. Aucune chaîne littérale de production ne porte les marqueurs sous forme de clé JSON
// (`"$enc"`, `"$c"`) hors des fichiers `internal/service/replayview/columns*.go`, qui sont l'encodeur.
// Le décodeur Go n'existe que dans un `_test.go` (hors de ce balayage) ; le décodeur de production est
// celui du web, tenu par `replayColumns.guard.test.ts`. Allowlist d'exceptions : VIDE. Les chaînes
// qui DÉCRIVENT la forme sans l'écrire (description OpenAPI) ne portent pas la clé entre guillemets.
package archlint

import (
	"go/ast"
	"go/token"
	"regexp"
	"strconv"
	"strings"
	"testing"
)

const dossierEncodeurColonnes = "internal/service/replayview/columns"

// colonnesAllowlist : exceptions au garde-rail. Vide ; une entrée ne s'ajoute pas, elle se retire avec
// le code qui la justifiait.
var colonnesAllowlist = map[string]string{}

var marqueurColonnes = regexp.MustCompile(`"\$(enc|c)"|^\$(enc|c)$`)

func TestUnSeulEncodeurDeLaFormeEnColonnes(t *testing.T) {
	chez := map[string]int{}
	for _, s := range goSourcesUnder(t, "cmd", "internal") {
		ast.Inspect(s.File, func(n ast.Node) bool {
			lit, ok := n.(*ast.BasicLit)
			if !ok || lit.Kind != token.STRING {
				return true
			}
			val, err := strconv.Unquote(lit.Value)
			if err == nil && marqueurColonnes.MatchString(val) {
				chez[s.Rel]++
			}
			return true
		})
	}
	foyer := 0
	for rel, n := range chez {
		switch {
		case strings.HasPrefix(rel, dossierEncodeurColonnes):
			foyer += n
		case colonnesAllowlist[rel] == "":
			t.Errorf("%s écrit un marqueur de la forme en colonnes (%d fois) : seul l'encodeur de replayview/columns*.go "+
				"le fait, et le décodeur de production est celui du web", rel, n)
		}
	}
	if foyer == 0 {
		t.Errorf("l'encodeur (%s*.go) n'écrit plus aucun marqueur : le garde-rail ne garde rien, le mettre à jour avec le code", dossierEncodeurColonnes)
	}
}

func TestMarqueurColonnesAutoPreuve(t *testing.T) {
	for _, s := range []string{`{"$enc":1`, `"$c"`, `$enc`, `$c`} {
		if !marqueurColonnes.MatchString(s) {
			t.Errorf("%q devrait être reconnu comme marqueur", s)
		}
	}
	for _, s := range []string{"`$enc` à la racine", "un objet qui porte `$c`", "columns", "$cache"} {
		if marqueurColonnes.MatchString(s) {
			t.Errorf("%q ne devrait pas être reconnu comme marqueur", s)
		}
	}
}
