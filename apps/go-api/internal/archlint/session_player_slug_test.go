// Package archlint — session_player_slug_test.go : ratchet « le joueur courant de la
// session n'est pas une identité » (registre sécurité C4-1, 2026-10-15).
//
// POURQUOI. Le joueur courant d'une session (`middleware.SessionPlayerSlug`) est
// choisi par l'interface (POST /session/context) parmi les profils LISIBLES, ceux des
// amis compris. Il ne prouve aucune propriété. Jusqu'au 2026-10-15, la suppression et
// le like des médias le prenaient pour le demandeur : un compte fixait son joueur
// courant sur sa victime et supprimait ses médias. Le demandeur vient désormais du
// profil possédé par l'utilisateur (xuid lié, api/requester_resolver.go).
//
// RÈGLE. Aucune référence à `SessionPlayerSlug` (appel ou valeur de fonction) dans
// internal/api/handlers/, internal/service/ et internal/domain/, hors tests. La
// composition (internal/api, internal/api/wire) peut le lire pour l'AFFICHAGE ou le
// mode mono-utilisateur. Allowlist vide ; toute entrée porte une date et une raison.
//
// Le champ de session lui-même (`domain.Session.CurrentPlayerSlug`), que SessionPlayerSlug
// enveloppe, ne se LIT pas non plus dans ces couches (écrire le choix de l'interface reste
// permis) ; currentPlayerSlugAllowlist tolère des lectures d'affichage, datées (ajouté le
// 2026-10-16, F-1a R1-11).
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

// sessionPlayerSlugScanRoots : arborescences interdites, relatives au module.
var sessionPlayerSlugScanRoots = []string{
	"internal/api/handlers",
	"internal/service",
	"internal/domain",
}

// sessionPlayerSlugAllowlist : fichier (relatif au module) -> date et raison.
var sessionPlayerSlugAllowlist = map[string]string{}

// sessionPlayerSlugRefs rend les lignes d'un fichier qui référencent SessionPlayerSlug.
func sessionPlayerSlugRefs(fset *token.FileSet, f *ast.File) []int {
	var lines []int
	ast.Inspect(f, func(n ast.Node) bool {
		if id, ok := n.(*ast.Ident); ok && id.Name == "SessionPlayerSlug" {
			lines = append(lines, fset.Position(id.Pos()).Line)
		}
		return true
	})
	return lines
}

func TestNoSessionPlayerSlugInDecisionLayers(t *testing.T) {
	_, thisFile, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("runtime.Caller a échoué")
	}
	moduleRoot := filepath.Dir(filepath.Dir(filepath.Dir(thisFile)))
	var violations, fieldReads []string
	scanned := 0
	for _, root := range sessionPlayerSlugScanRoots {
		err := filepath.WalkDir(filepath.Join(moduleRoot, filepath.FromSlash(root)), func(path string, d os.DirEntry, err error) error {
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
			if _, allowed := currentPlayerSlugAllowlist[rel]; !allowed {
				for _, line := range currentPlayerSlugReads(fset, f) {
					fieldReads = append(fieldReads, rel+":"+itoa(line))
				}
			}
			if _, allowed := sessionPlayerSlugAllowlist[rel]; allowed {
				return nil
			}
			for _, line := range sessionPlayerSlugRefs(fset, f) {
				violations = append(violations, rel+":"+itoa(line))
			}
			return nil
		})
		if err != nil {
			t.Fatalf("parcours de %s : %v", root, err)
		}
	}
	if scanned < 100 {
		t.Fatalf("%d fichier(s) analysé(s) : le parcours ne voit plus les couches visées", scanned)
	}
	sort.Strings(violations)
	for _, v := range violations {
		t.Errorf("%s : SessionPlayerSlug dans une couche de décision — résoudre le demandeur par le profil possédé (api/requester_resolver.go)", v)
	}
	sort.Strings(fieldReads)
	for _, v := range fieldReads {
		t.Errorf("%s : lecture du champ de session CurrentPlayerSlug dans une couche de décision — le joueur "+
			"courant n'est pas une identité (profil possédé : api/requester_resolver.go)", v)
	}
	for rel := range currentPlayerSlugAllowlist {
		if _, err := os.Stat(filepath.Join(moduleRoot, filepath.FromSlash(rel))); err != nil {
			t.Errorf("allowlist périmée : %s n'existe plus", rel)
		}
	}
	for rel := range sessionPlayerSlugAllowlist {
		if _, err := os.Stat(filepath.Join(moduleRoot, filepath.FromSlash(rel))); err != nil {
			t.Errorf("allowlist périmée : %s n'existe plus", rel)
		}
	}
}

// TestSessionPlayerSlugGuardDetectsDefects : l'analyse voit l'appel tel qu'il existait
// dans handlers/media_delete.go avant le 2026-10-15, et la valeur de fonction ; elle
// ignore un commentaire.
func TestSessionPlayerSlugGuardDetectsDefects(t *testing.T) {
	src := `package handlers
// SessionPlayerSlug n'est pas une identité.
func a(ctx any) { req.RequesterSlug = middleware.SessionPlayerSlug(ctx) }
var b = middleware.SessionPlayerSlug
`
	fset := token.NewFileSet()
	f, err := parser.ParseFile(fset, "x.go", src, 0)
	if err != nil {
		t.Fatal(err)
	}
	if got := sessionPlayerSlugRefs(fset, f); len(got) != 2 || got[0] != 3 || got[1] != 4 {
		t.Fatalf("lignes détectées %v, attendu [3 4]", got)
	}
}

// TestCurrentPlayerSlugGuardDetectsDefects : l'analyse voit la lecture directe du champ de
// session (déréférencée, comparée, passée en valeur), et ignore son écriture, sa déclaration
// et un appel de méthode homonyme.
func TestCurrentPlayerSlugGuardDetectsDefects(t *testing.T) {
	src := `package handlers
type Session struct{ CurrentPlayerSlug *string }
func a(sess *Session) { req.RequesterSlug = *sess.CurrentPlayerSlug }
func b(sess *Session) bool { return sess.CurrentPlayerSlug != nil }
func c(sess *Session) any { return resp{CurrentPlayerSlug: sess.CurrentPlayerSlug} }
func d(sess *Session, s string) { sess.CurrentPlayerSlug = &s }
func e(repo any) string { return repo.CurrentPlayerSlug() }
`
	fset := token.NewFileSet()
	f, err := parser.ParseFile(fset, "x.go", src, 0)
	if err != nil {
		t.Fatal(err)
	}
	got := currentPlayerSlugReads(fset, f)
	if len(got) != 3 || got[0] != 3 || got[1] != 4 || got[2] != 5 {
		t.Fatalf("lignes détectées %v, attendu [3 4 5]", got)
	}
}

// currentPlayerSlugAllowlist : fichier (relatif au module) -> date et raison. Lecture du
// champ de session CurrentPlayerSlug tolérée pour l'AFFICHAGE seulement.
var currentPlayerSlugAllowlist = map[string]string{
	// 2026-10-16 : affichage — le profil montré par défaut au bootstrap (current_player) est
	// choisi parmi les profils déjà lisibles par la session ; aucune décision d'accès.
	"internal/service/bootstrap_service.go": "affichage du joueur courant",
	// 2026-10-16 : affichage — POST /session/context renvoie au front le contexte qu'il vient
	// de poser (le choix lui-même est contrôlé par canSelectPlayer).
	"internal/api/handlers/session_context.go": "renvoi du contexte de session",
	// 2026-10-16 : persistance — Session.IsEmpty dit si la session porte un état à conserver.
	"internal/domain/session.go": "test de session vide",
}

// currentPlayerSlugReads rend les lignes qui LISENT le champ CurrentPlayerSlug d'une valeur
// (sélecteur hors partie gauche d'une affectation et hors appel de méthode).
func currentPlayerSlugReads(fset *token.FileSet, f *ast.File) []int {
	written := map[*ast.SelectorExpr]bool{}
	called := map[*ast.SelectorExpr]bool{}
	ast.Inspect(f, func(n ast.Node) bool {
		switch x := n.(type) {
		case *ast.AssignStmt:
			for _, lhs := range x.Lhs {
				if sel, ok := lhs.(*ast.SelectorExpr); ok {
					written[sel] = true
				}
			}
		case *ast.CallExpr:
			if sel, ok := x.Fun.(*ast.SelectorExpr); ok {
				called[sel] = true
			}
		}
		return true
	})
	var lines []int
	ast.Inspect(f, func(n ast.Node) bool {
		if sel, ok := n.(*ast.SelectorExpr); ok && sel.Sel.Name == "CurrentPlayerSlug" && !written[sel] && !called[sel] {
			lines = append(lines, fset.Position(sel.Pos()).Line)
		}
		return true
	})
	return lines
}
