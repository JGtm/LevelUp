// Package archlint — history_key_single_source_test.go : les clés du rattrapage de l'historique
// d'un joueur ont UNE source (CLAUDE.md règle 6).
//
// Les clés `sync_meta` qui disent « la fin de l'historique a été vue » (`history_complete_at`) et
// « le rattrapage en est là » (`history_catchup_cursor`, le marque-page), « ce point a échoué N
// passes de suite » (`history_failure_streak`), « ce match n'est plus demandé » (`history_skip_set`), leur lecture, leur
// écriture, le budget par passe et la borne de profondeur vivent dans
// `internal/sync/historystate`. Le cycle V2 et le runner Halo 5 l'appellent (le moteur V1 non : delta pur). Une
// seconde copie d'un littéral est une seconde règle libre de diverger : un chemin qui range la fin
// que l'autre ne lit pas, ou un nom de clé changé d'un seul côté.
//
// Portée : tout fichier Go non-test de `internal/` et `cmd/`, littéraux chaîne lus dans l'AST
// (les commentaires n'en sont pas). Seul le paquet `historystate` peut porter ces littéraux.
package archlint

import (
	"go/ast"
	"go/token"
	"strconv"
	"strings"
	"testing"
)

// historyStatePackage : la source unique (préfixe de chemin relatif à apps/go-api).
const historyStatePackage = "internal/sync/historystate/"

// historyKeyLiterals : les clés de `sync_meta` du rattrapage de l'historique (fin vue, marque-page).
var historyKeyLiterals = []string{"history_complete_at", "history_catchup_cursor", "history_failure_streak", "history_skip_set"}

// historyKeyViolations rend « fichier:ligne » de chaque littéral chaîne contenant l'une des clés,
// hors du paquet historystate.
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
			val, err := strconv.Unquote(lit.Value)
			if err != nil {
				return true
			}
			for _, key := range historyKeyLiterals {
				if strings.Contains(val, key) {
					out = append(out, src.Rel+":"+strconv.Itoa(src.Fset.Position(lit.Pos()).Line))
					break
				}
			}
			return true
		})
	}
	return out
}

func TestHistoryKeyLiteralOnlyInHistorystate(t *testing.T) {
	sources := goSourcesUnder(t, "internal", "cmd")
	if violations := historyKeyViolations(sources); len(violations) > 0 {
		t.Errorf("littéraux %q hors de %s — passer par historystate (Complete, MarkComplete, LoadCursor, SaveCursor) :\n  %s",
			historyKeyLiterals, historyStatePackage, strings.Join(violations, "\n  "))
	}
	// Le garde-rail ne garde rien si la source unique a perdu une clé.
	for _, key := range historyKeyLiterals {
		found := false
		for _, src := range sources {
			if strings.HasPrefix(src.Rel, historyStatePackage) && strings.Contains(string(src.Src), `"`+key+`"`) {
				found = true
			}
		}
		if !found {
			t.Errorf("%s ne déclare plus le littéral %q : le garde-rail est sans objet", historyStatePackage, key)
		}
	}
}

// Auto-preuve : le garde-rail voit un littéral (chaîne ou SQL) hors du paquet, pas un commentaire,
// et laisse passer le paquet source.
func TestHistoryKeyLiteralOnlyInHistorystate_AutoPreuve(t *testing.T) {
	dehors := parseGoSnippet(t, "internal/sync/v2/x.go", "package v2\n"+
		"// history_complete_at en commentaire : toléré\n"+
		"var k = \"history_complete_at\"\n"+
		"var q = `SELECT 1 FROM sync_meta WHERE key = 'history_complete_at'`\n"+
		"var c = \"history_catchup_cursor\"\n"+
		"var s = \"history_failure_streak\"\n"+
		"var k2 = \"history_skip_set\"\n")
	if got := historyKeyViolations([]goSource{dehors}); len(got) != 5 {
		t.Errorf("violations = %v, attendu 5 (la chaîne, le SQL, le marque-page, le compte d'échecs et l'ensemble des matchs à ne plus demander, pas le commentaire)", got)
	}
	dedans := parseGoSnippet(t, historyStatePackage+"x.go", "package historystate\n"+
		"const Key = \"history_complete_at\"\n"+
		"const CursorKey = \"history_catchup_cursor\"\n"+
		"const StreakKey = \"history_failure_streak\"\n"+
		"const SkipSetKey = \"history_skip_set\"\n")
	if got := historyKeyViolations([]goSource{dedans}); len(got) != 0 {
		t.Errorf("le paquet historystate ne doit pas être signalé : %v", got)
	}
}
