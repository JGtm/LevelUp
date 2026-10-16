// Package archlint — invite_code_log_test.go : ratchet « un code d'invitation ne se
// journalise pas » (registre sécurité C5-6, 2026-10-16).
//
// POURQUOI. Un code d'invitation vaut accès : il lève le verrou d'instance et donne
// le droit de créer son profil. Les journaux (fichiers, sauvegardes, exports) ne sont
// pas un canal sûr ; jusqu'au 2026-10-16, l'émission, la révocation et la
// consommation d'une invitation y écrivaient le code en clair.
//
// RÈGLE. Dans les fichiers qui émettent, révoquent ou consomment une invitation,
// aucun appel slog ne porte un attribut de clé "code". Allowlist vide.
package archlint

import (
	"go/ast"
	"go/parser"
	"go/token"
	"path/filepath"
	"runtime"
	"strconv"
	"testing"
)

// inviteFiles : fichiers (relatifs au module) qui manipulent un code d'invitation.
var inviteFiles = []string{
	"internal/api/handlers/admin.go",
	"internal/api/handlers/groups.go",
	"internal/api/handlers/user_auth.go",
	"internal/api/handlers/auth_xbox_oauth.go",
	"internal/service/xbox_auth_service.go",
	"internal/service/xbox_auth_invite.go",
}

// slogCodeAttrs rend les lignes des appels slog qui portent la clé "code".
func slogCodeAttrs(fset *token.FileSet, f *ast.File) []int {
	var lines []int
	ast.Inspect(f, func(n ast.Node) bool {
		call, ok := n.(*ast.CallExpr)
		if !ok {
			return true
		}
		sel, ok := call.Fun.(*ast.SelectorExpr)
		if !ok {
			return true
		}
		if pkg, ok := sel.X.(*ast.Ident); !ok || pkg.Name != "slog" {
			return true
		}
		for _, arg := range call.Args {
			lit, ok := arg.(*ast.BasicLit)
			if !ok || lit.Kind != token.STRING {
				continue
			}
			if v, err := strconv.Unquote(lit.Value); err == nil && v == "code" {
				lines = append(lines, fset.Position(lit.Pos()).Line)
			}
		}
		return true
	})
	return lines
}

func TestNoInviteCodeInLogs(t *testing.T) {
	_, thisFile, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("runtime.Caller a échoué")
	}
	moduleRoot := filepath.Dir(filepath.Dir(filepath.Dir(thisFile)))
	for _, rel := range inviteFiles {
		fset := token.NewFileSet()
		f, err := parser.ParseFile(fset, filepath.Join(moduleRoot, filepath.FromSlash(rel)), nil, 0)
		if err != nil {
			t.Fatalf("lecture de %s : %v", rel, err)
		}
		for _, line := range slogCodeAttrs(fset, f) {
			t.Errorf("%s:%d : attribut slog \"code\" — un code d'invitation ne se journalise pas", rel, line)
		}
	}
}

// TestInviteCodeLogGuardDetectsDefects : l'analyse voit les journaux tels qu'ils
// existaient dans handlers/admin.go avant le 2026-10-16 ; elle ignore la valeur
// "code" hors d'un appel slog et une autre clé.
func TestInviteCodeLogGuardDetectsDefects(t *testing.T) {
	src := `package handlers
func a() {
	slog.Info("admin: invitation générée", "code", invite.Code, "by", createdBy)
	slog.ErrorContext(ctx, "admin: erreur revoke invite", "code", in.Code, "err", err)
	x := map[string]any{"code": 1}
	slog.WarnContext(ctx, "auth: refus", "error_code", code)
}
`
	fset := token.NewFileSet()
	f, err := parser.ParseFile(fset, "x.go", src, 0)
	if err != nil {
		t.Fatal(err)
	}
	if got := slogCodeAttrs(fset, f); len(got) != 2 || got[0] != 3 || got[1] != 4 {
		t.Fatalf("lignes détectées %v, attendu [3 4]", got)
	}
}
