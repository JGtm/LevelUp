// Package archlint — i18n_messages_test.go : garde-rail du catalogue des messages Go (ADR 0040,
// D-3/D-4, étape 7.1 du plan `.ai/V8/PLAN_I18N_MULTILANGUE.md`).
//
// Deux règles, tenues par parcours AST des fichiers Go non-test (hors internal/i18n) :
//
//  1. toute clé passée à i18n.Msg / MsgText / MsgTextWith / MsgAll est un littéral (ou un littéral de
//     préfixe suivi d'une partie variable : `"format.month_short." + n`) qui désigne une clé du
//     catalogue ; une clé tapée de travers ne se découvre pas à l'exécution, sous la forme de la clé
//     rendue à l'écran ;
//  2. toute clé du catalogue est appelée quelque part : un message sans appelant est du code mort
//     (règle n°7 de CLAUDE.md), il se supprime du TOML.
//
// La complétude des langues (chaque clé dans chaque langue de config/locales.toml) est tenue à la
// source, par internal/i18n (TestCatalogueComplet).
package archlint

import (
	"go/ast"
	"go/token"
	"sort"
	"strconv"
	"strings"
	"testing"

	"levelup/go-api/internal/i18n"
)

// i18nMessageFuncs : fonctions du catalogue et rang de l'argument qui porte la clé.
var i18nMessageFuncs = map[string]int{
	"Msg":         1, // Msg(l, key, params)
	"MsgText":     0, // MsgText(key, params)
	"MsgTextWith": 0, // MsgTextWith(key, params func)
	"MsgAll":      0, // MsgAll(key)
}

// messageKeyUse est une clé (ou un préfixe de clés) appelée à une position donnée.
type messageKeyUse struct {
	Key    string
	Prefix bool // vrai : `"prefixe." + variable`
	Where  string
}

// collectMessageKeyUses relève les clés appelées, et les appels dont la clé n'est ni un littéral ni un
// préfixe littéral (retournés tels quels dans bad).
func collectMessageKeyUses(srcs []goSource) (uses []messageKeyUse, bad []string) {
	for _, s := range srcs {
		if strings.HasPrefix(s.Rel, "internal/i18n/") {
			continue
		}
		ast.Inspect(s.File, func(n ast.Node) bool {
			call, ok := n.(*ast.CallExpr)
			if !ok {
				return true
			}
			sel, ok := call.Fun.(*ast.SelectorExpr)
			if !ok {
				return true
			}
			pkg, ok := sel.X.(*ast.Ident)
			if !ok || pkg.Name != "i18n" {
				return true
			}
			idx, tracked := i18nMessageFuncs[sel.Sel.Name]
			if !tracked || idx >= len(call.Args) {
				return true
			}
			where := s.Rel + ":" + strconv.Itoa(s.Fset.Position(call.Pos()).Line)
			if key, prefix, ok := literalMessageKey(call.Args[idx]); ok {
				uses = append(uses, messageKeyUse{Key: key, Prefix: prefix, Where: where})
			} else {
				bad = append(bad, where+" : i18n."+sel.Sel.Name+" avec une clé qui n'est pas un littéral")
			}
			return true
		})
	}
	return uses, bad
}

// literalMessageKey reconnaît une clé littérale, ou un littéral de préfixe (terminé par ".") suivi d'une
// partie variable.
func literalMessageKey(e ast.Expr) (key string, prefix, ok bool) {
	switch v := e.(type) {
	case *ast.BasicLit:
		if v.Kind != token.STRING {
			return "", false, false
		}
		s, err := strconv.Unquote(v.Value)
		return s, false, err == nil
	case *ast.BinaryExpr:
		if v.Op != token.ADD {
			return "", false, false
		}
		lit, isLit := v.X.(*ast.BasicLit)
		if !isLit || lit.Kind != token.STRING {
			return "", false, false
		}
		s, err := strconv.Unquote(lit.Value)
		if err != nil || !strings.HasSuffix(s, ".") {
			return "", false, false
		}
		return s, true, true
	}
	return "", false, false
}

func TestMessagesGoKeysExistentEtServent(t *testing.T) {
	uses, bad := collectMessageKeyUses(goSourcesUnder(t, "cmd", "internal", "pkg", "tools", "scripts"))
	for _, b := range bad {
		t.Error(b)
	}
	known := i18n.MessageKeys()
	knownSet := map[string]bool{}
	for _, k := range known {
		knownSet[k] = true
	}
	used := map[string]bool{}
	for _, u := range uses {
		if !u.Prefix {
			if !knownSet[u.Key] {
				t.Errorf("%s : clé %q absente du catalogue (internal/i18n/messages/*.toml)", u.Where, u.Key)
				continue
			}
			used[u.Key] = true
			continue
		}
		matched := false
		for _, k := range known {
			if strings.HasPrefix(k, u.Key) {
				used[k] = true
				matched = true
			}
		}
		if !matched {
			t.Errorf("%s : aucune clé du catalogue ne commence par %q", u.Where, u.Key)
		}
	}
	var unused []string
	for _, k := range known {
		if !used[k] {
			unused = append(unused, k)
		}
	}
	sort.Strings(unused)
	if len(unused) > 0 {
		t.Errorf("%d clé(s) du catalogue sans appelant (code mort, à supprimer des TOML) : %s",
			len(unused), strings.Join(unused, ", "))
	}
	if len(uses) < 20 {
		t.Errorf("auto-preuve : %d appels relevés seulement, le parcours ne voit pas les appels", len(uses))
	}
}

// Auto-preuve de la reconnaissance des clés.
func TestLiteralMessageKeySelfCheck(t *testing.T) {
	lit := &ast.BasicLit{Kind: token.STRING, Value: `"a.b"`}
	if k, prefix, ok := literalMessageKey(lit); !ok || prefix || k != "a.b" {
		t.Errorf("littéral : %q %v %v", k, prefix, ok)
	}
	pre := &ast.BinaryExpr{Op: token.ADD, X: &ast.BasicLit{Kind: token.STRING, Value: `"a."`}, Y: &ast.Ident{Name: "n"}}
	if k, prefix, ok := literalMessageKey(pre); !ok || !prefix || k != "a." {
		t.Errorf("préfixe : %q %v %v", k, prefix, ok)
	}
	bare := &ast.BinaryExpr{Op: token.ADD, X: &ast.BasicLit{Kind: token.STRING, Value: `"a"`}, Y: &ast.Ident{Name: "n"}}
	if _, _, ok := literalMessageKey(bare); ok {
		t.Error("un préfixe sans point final n'est pas une famille de clés")
	}
	if _, _, ok := literalMessageKey(&ast.Ident{Name: "k"}); ok {
		t.Error("un identifiant n'est pas un littéral")
	}
}
