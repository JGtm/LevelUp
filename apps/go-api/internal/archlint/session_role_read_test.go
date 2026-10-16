// Package archlint — session_role_read_test.go : ratchet « le rôle copié dans la
// session n'est pas une preuve » (registre sécurité C5-3, 2026-10-16).
//
// POURQUOI. La connexion copie le rôle du compte dans la session (`SessionData.Role`,
// un *string). Une décision qui le relit garde un admin rétrogradé ou supprimé admin
// tant qu'il rejoue sa session (TTL glissant de 7 jours). La décision passe par
// authz.IsAdmin, qui relit le rôle dans le magasin d'utilisateurs.
//
// RÈGLE. Aucun déréférencement d'un champ `Role` (`*x.Role`, la seule forme de
// lecture d'un *string) dans le code non-test du module. `domain.User.Role` n'est pas
// un pointeur et n'est jamais déréférencé : seule la lecture du rôle de session est
// visée. Toute entrée de l'allowlist porte une date et une raison.
package archlint

import (
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"runtime"
	"sort"
	"strings"
	"testing"
)

// sessionRoleReadAllowlist : fichier (relatif au module) -> date et raison.
var sessionRoleReadAllowlist = map[string]string{
	// 2026-10-16 (lot sécurité L4) : `is_admin` du bootstrap n'est qu'un AFFICHAGE
	// (menu d'administration) ; toute route d'administration décide par RequireAdmin.
	// Relevé en découverte D-L4 du plan sécurité (aligner l'affichage sur le magasin).
	"internal/service/bootstrap_service.go": "2026-10-16 affichage is_admin, décision par RequireAdmin",
}

// sessionRoleDerefs rend les lignes d'un fichier qui déréférencent un champ Role.
func sessionRoleDerefs(fset *token.FileSet, f *ast.File) []int {
	var lines []int
	ast.Inspect(f, func(n ast.Node) bool {
		star, ok := n.(*ast.StarExpr)
		if !ok {
			return true
		}
		if sel, ok := star.X.(*ast.SelectorExpr); ok && sel.Sel.Name == "Role" {
			lines = append(lines, fset.Position(star.Pos()).Line)
		}
		return true
	})
	return lines
}

func TestNoSessionRoleReadForDecisions(t *testing.T) {
	_, thisFile, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("runtime.Caller a échoué")
	}
	moduleRoot := filepath.Dir(filepath.Dir(filepath.Dir(thisFile)))
	var violations []string
	scanned := 0
	for _, root := range []string{"internal", "cmd"} {
		err := filepath.WalkDir(filepath.Join(moduleRoot, root), func(path string, d os.DirEntry, err error) error {
			if err != nil {
				return err
			}
			if d.IsDir() || !strings.HasSuffix(path, ".go") || strings.HasSuffix(path, "_test.go") {
				return nil
			}
			rel, _ := filepath.Rel(moduleRoot, path)
			rel = filepath.ToSlash(rel)
			fset := token.NewFileSet()
			f, perr := parser.ParseFile(fset, path, nil, 0)
			if perr != nil {
				return perr
			}
			scanned++
			if _, allowed := sessionRoleReadAllowlist[rel]; allowed {
				return nil
			}
			for _, line := range sessionRoleDerefs(fset, f) {
				violations = append(violations, rel+":"+itoa(line))
			}
			return nil
		})
		if err != nil {
			t.Fatalf("parcours de %s : %v", root, err)
		}
	}
	if scanned < 500 {
		t.Fatalf("%d fichier(s) analysé(s) : le parcours ne voit plus le module", scanned)
	}
	sort.Strings(violations)
	for _, v := range violations {
		t.Errorf("%s : rôle lu dans la session — décider par authz.IsAdmin (rôle relu dans le magasin)", v)
	}
	for rel := range sessionRoleReadAllowlist {
		if _, err := os.Stat(filepath.Join(moduleRoot, filepath.FromSlash(rel))); err != nil {
			t.Errorf("allowlist périmée : %s n'existe plus", rel)
		}
	}
}

// TestSessionRoleGuardDetectsDefects : l'analyse voit les lectures telles qu'elles
// existaient dans middleware/require_admin.go et handlers/settings.go avant le
// 2026-10-16 ; elle ignore une écriture et le rôle d'un compte.
func TestSessionRoleGuardDetectsDefects(t *testing.T) {
	src := `package middleware
func a() bool { return sess == nil || sess.Role == nil || *sess.Role != "admin" }
func b() { sess.Role = &role }
func c() bool { return user.Role == domain.RoleAdmin }
func d() bool { return s.Role != nil && *s.Role == string(domain.RoleAdmin) }
`
	fset := token.NewFileSet()
	f, err := parser.ParseFile(fset, "x.go", src, 0)
	if err != nil {
		t.Fatal(err)
	}
	if got := sessionRoleDerefs(fset, f); len(got) != 2 || got[0] != 2 || got[1] != 5 {
		t.Fatalf("lignes détectées %v, attendu [2 5]", got)
	}
}
