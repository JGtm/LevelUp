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
	var violations []string
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
