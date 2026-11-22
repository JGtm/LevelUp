// Package archlint — history_key_single_source_test.go : la clé du rattrapage de l'historique
// d'un joueur a UNE source (CLAUDE.md règle 6).
//
// La clé `sync_meta` qui dit « la fin de l'historique a été vue » (`history_complete_at`), sa
// lecture, son écriture, le budget par passe et la borne de profondeur vivent dans
// `internal/sync/historystate`. Le moteur V1, le cycle V2 et le runner Halo 5 l'appellent. Une
// seconde copie du littéral est une seconde règle libre de diverger : un chemin qui range la fin
// que l'autre ne lit pas, ou un nom de clé changé d'un seul côté.
//
// Portée : tout fichier Go non-test de `internal/` et `cmd/`, littéraux chaîne lus dans l'AST
// (les commentaires n'en sont pas). Seul le paquet `historystate` peut porter le littéral.
package archlint

import (
	"go/ast"
	"go/token"
	"strconv"
	"strings"
	"testing"
)

const (
	// historyKeyLiteral : la clé de `sync_meta` du rattrapage de l'historique.
	historyKeyLiteral = "history_complete_at"
	// historyStatePackage : la source unique (préfixe de chemin relatif à apps/go-api).
	historyStatePackage = "internal/sync/historystate/"
)

// historyKeyViolations rend « fichier:ligne » de chaque littéral chaîne contenant la clé, hors
// du paquet historystate.
func historyKeyViolations(sources []goSource) []string {
	var out []string
	for _, src := range sources {
		if strings.HasPrefix(src.Rel, historyStatePackage) {
			continue
		}
		ast.Inspect(src.File, func(n ast.Node) bool {
			lit, ok := n.(*ast.BasicLit)
			if !ok || lit.Kind != token.STRING {
				return true
			}
			if val, err := strconv.Unquote(lit.Value); err == nil && strings.Contains(val, historyKeyLiteral) {
				out = append(out, src.Rel+":"+strconv.Itoa(src.Fset.Position(lit.Pos()).Line))
			}
			return true
		})
	}
	return out
}

func TestHistoryKeyLiteralOnlyInHistorystate(t *testing.T) {
	sources := goSourcesUnder(t, "internal", "cmd")
	if violations := historyKeyViolations(sources); len(violations) > 0 {
		t.Errorf("littéral %q hors de %s — passer par historystate.Complete / MarkComplete :\n  %s",
			historyKeyLiteral, historyStatePackage, strings.Join(violations, "\n  "))
	}
	// Le garde-rail ne garde rien si la source unique a perdu la clé.
	found := false
	for _, src := range sources {
		if strings.HasPrefix(src.Rel, historyStatePackage) && strings.Contains(string(src.Src), `"`+historyKeyLiteral+`"`) {
			found = true
		}
	}
	if !found {
		t.Errorf("%s ne déclare plus le littéral %q : le garde-rail est sans objet", historyStatePackage, historyKeyLiteral)
	}
}

// Auto-preuve : le garde-rail voit un littéral (chaîne ou SQL) hors du paquet, pas un commentaire,
// et laisse passer le paquet source.
func TestHistoryKeyLiteralOnlyInHistorystate_AutoPreuve(t *testing.T) {
	dehors := parseGoSnippet(t, "internal/sync/v2/x.go", "package v2\n"+
		"// history_complete_at en commentaire : toléré\n"+
		"var k = \"history_complete_at\"\n"+
		"var q = `SELECT 1 FROM sync_meta WHERE key = 'history_complete_at'`\n")
	if got := historyKeyViolations([]goSource{dehors}); len(got) != 2 {
		t.Errorf("violations = %v, attendu 2 (la chaîne et le SQL, pas le commentaire)", got)
	}
	dedans := parseGoSnippet(t, historyStatePackage+"x.go", "package historystate\nconst Key = \"history_complete_at\"\n")
	if got := historyKeyViolations([]goSource{dedans}); len(got) != 0 {
		t.Errorf("le paquet historystate ne doit pas être signalé : %v", got)
	}
}
