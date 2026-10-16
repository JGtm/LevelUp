package prestige

// squad_member_guard_test.go — GARDE-RAIL « appartenance à l'escouade ».
//
// Les escouades et leurs défis vivent dans la base sociale PARTAGÉE : l'isolation par
// base joueur (prestigePlayerSlugCtx) ne les couvre pas. La seule barrière contre un
// identifiant d'escouade d'autrui est la vérification d'appartenance du service
// (assertMemberUser, ou membersIfMember quand l'appelant a aussi besoin du roster).
//
// Règle tenue ici, par analyse AST de internal/prestige/ (hors tests) : toute méthode
// EXPORTÉE de *service qui reçoit un identifiant d'escouade — paramètre nommé
// `squadID`, ou paramètre d'un type déclaré dans le paquet qui porte un champ
// `SquadID` — appelle s.assertMemberUser ou s.membersIfMember. Une exception porte sa
// date et sa raison dans squadMemberGuardAllowlist.

import (
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"
)

// squadMemberGuardAllowlist : méthodes exemptées, avec leur justification datée.
var squadMemberGuardAllowlist = map[string]string{
	// 2026-10-15 : aucun acteur dans la signature ; le seul appelant (handler
	// ListMySquads) ne la nourrit qu'avec les escouades rendues par
	// ListSquadsForUser(acteur gardé), donc des escouades dont l'acteur est membre.
	"ListSquadMembers": "roster d'escouades déjà filtrées par appartenance (handler ListMySquads)",
}

// squadMembershipCalls : appels qui valent vérification d'appartenance.
var squadMembershipCalls = map[string]bool{"assertMemberUser": true, "membersIfMember": true}

// typesWithSquadID rend les types struct du paquet qui portent un champ SquadID.
func typesWithSquadID(files []*ast.File) map[string]bool {
	out := map[string]bool{}
	for _, f := range files {
		ast.Inspect(f, func(n ast.Node) bool {
			ts, ok := n.(*ast.TypeSpec)
			if !ok {
				return true
			}
			st, ok := ts.Type.(*ast.StructType)
			if !ok {
				return true
			}
			for _, field := range st.Fields.List {
				for _, name := range field.Names {
					if name.Name == "SquadID" {
						out[ts.Name.Name] = true
					}
				}
			}
			return true
		})
	}
	return out
}

// receivesSquadID dit si la signature reçoit un identifiant d'escouade.
func receivesSquadID(fd *ast.FuncDecl, squadTypes map[string]bool) bool {
	for _, p := range fd.Type.Params.List {
		for _, name := range p.Names {
			if name.Name == "squadID" {
				return true
			}
		}
		if id, ok := p.Type.(*ast.Ident); ok && squadTypes[id.Name] {
			return true
		}
	}
	return false
}

// callsMembershipGuard dit si le corps appelle s.assertMemberUser / s.membersIfMember.
func callsMembershipGuard(body *ast.BlockStmt) bool {
	found := false
	ast.Inspect(body, func(n ast.Node) bool {
		call, ok := n.(*ast.CallExpr)
		if !ok {
			return true
		}
		if sel, ok := call.Fun.(*ast.SelectorExpr); ok && squadMembershipCalls[sel.Sel.Name] {
			found = true
		}
		return !found
	})
	return found
}

func isServiceMethod(fd *ast.FuncDecl) bool {
	if fd.Recv == nil || len(fd.Recv.List) != 1 || fd.Body == nil {
		return false
	}
	star, ok := fd.Recv.List[0].Type.(*ast.StarExpr)
	if !ok {
		return false
	}
	id, ok := star.X.(*ast.Ident)
	return ok && id.Name == "service"
}

// squadMemberGuardViolations analyse des fichiers déjà parsés.
func squadMemberGuardViolations(files []*ast.File) (violations []string, guarded int) {
	squadTypes := typesWithSquadID(files)
	for _, f := range files {
		for _, decl := range f.Decls {
			fd, ok := decl.(*ast.FuncDecl)
			if !ok || !isServiceMethod(fd) || !fd.Name.IsExported() || !receivesSquadID(fd, squadTypes) {
				continue
			}
			if _, allowed := squadMemberGuardAllowlist[fd.Name.Name]; allowed {
				continue
			}
			if callsMembershipGuard(fd.Body) {
				guarded++
				continue
			}
			violations = append(violations, fd.Name.Name)
		}
	}
	sort.Strings(violations)
	return violations, guarded
}

func parsePrestigeSources(t *testing.T) []*ast.File {
	t.Helper()
	entries, err := os.ReadDir(".")
	if err != nil {
		t.Fatalf("lecture du paquet : %v", err)
	}
	fset := token.NewFileSet()
	var files []*ast.File
	for _, e := range entries {
		name := e.Name()
		if e.IsDir() || !strings.HasSuffix(name, ".go") || strings.HasSuffix(name, "_test.go") {
			continue
		}
		f, err := parser.ParseFile(fset, filepath.Join(".", name), nil, 0)
		if err != nil {
			t.Fatalf("parse %s : %v", name, err)
		}
		files = append(files, f)
	}
	return files
}

func TestSquadMethodsAssertMembership(t *testing.T) {
	files := parsePrestigeSources(t)
	violations, guarded := squadMemberGuardViolations(files)
	if guarded < 5 {
		t.Fatalf("%d méthode(s) d'escouade vues gardées : l'analyse ne voit plus les méthodes du service", guarded)
	}
	for _, v := range violations {
		t.Errorf("méthode %s : reçoit un identifiant d'escouade sans appeler assertMemberUser/membersIfMember", v)
	}
	// Allowlist sans entrée périmée.
	present := map[string]bool{}
	for _, f := range files {
		for _, d := range f.Decls {
			if fd, ok := d.(*ast.FuncDecl); ok && isServiceMethod(fd) {
				present[fd.Name.Name] = true
			}
		}
	}
	for name := range squadMemberGuardAllowlist {
		if !present[name] {
			t.Errorf("allowlist périmée : %s n'existe plus", name)
		}
	}
}

// TestSquadMemberGuardDetectsDefects : l'analyse sait échouer — la création d'un défi
// d'escouade telle qu'elle était avant le 2026-10-15 (sans garde) est signalée.
func TestSquadMemberGuardDetectsDefects(t *testing.T) {
	src := `package prestige
type CreateReq struct{ SquadID string }
type service struct{}
func (s *service) Create(ctx int, req CreateReq) error { return nil }
func (s *service) Rename(ctx int, squadID, name string) error { return nil }
func (s *service) Guarded(ctx int, squadID string) error { return s.assertMemberUser(ctx, squadID, "u") }
`
	f, err := parser.ParseFile(token.NewFileSet(), "x.go", src, 0)
	if err != nil {
		t.Fatal(err)
	}
	violations, guarded := squadMemberGuardViolations([]*ast.File{f})
	if strings.Join(violations, ",") != "Create,Rename" || guarded != 1 {
		t.Fatalf("violations %v (gardées %d), attendu [Create Rename] et 1", violations, guarded)
	}
}
