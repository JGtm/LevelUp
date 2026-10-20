// Package archlint — no_demo_layout_translation_test.go : une seule disposition de l'arbre
// démo (backlog 2026-09-26, lot B5.1 ; CLAUDE.md règle n°6).
//
// La traduction « chemin de production -> chemin de la fixture démo » vivait en TROIS copies
// (cmd/server/demo_paths.go demoWarehouseDBPath, internal/config/player_resolver.go
// demoTitleDir, internal/ops/seed_demo_multititle.go demoTitleSubdir). Elle vit désormais dans
// internal/domain/title/demo_layout.go (title.DemoLayout), et nulle part ailleurs.
//
// Interdit, hors de demo_layout.go et hors des fichiers de test :
//  1. une fonction nommée comme un traducteur de disposition démo (demoTitleDir,
//     demoTitleSubdir, demoWarehouseDBPath, demoSharedDBPath, demoMetaDBPath…) ;
//  2. un `filepath.Join(X, …)` dont la base X désigne la racine démo (texte contenant « demo »
//     ou « fixture ») et qui ajoute un segment de la disposition (warehouse, titles, players,
//     auth, runtime) ;
//  3. dans les fichiers qui fabriquent ou résolvent l'arbre démo (seed de la démo, résolveur
//     démo de config, cmd/server/demo_paths.go) : TOUT `filepath.Join` qui ajoute un segment
//     de la disposition, quel que soit le nom de sa base.
package archlint

import (
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"regexp"
	"runtime"
	"sort"
	"strconv"
	"strings"
	"testing"
)

// dispositionDemoCanonique : LE fichier où vit la disposition.
const dispositionDemoCanonique = "internal/domain/title/demo_layout.go"

// nomDeTraducteurDemo : noms des anciennes copies et de leurs variantes plausibles.
var nomDeTraducteurDemo = regexp.MustCompile(
	`(?i)^demo(title(dir|subdir)|warehouse\w*|(shared|meta|metadata|social|sharedsocial|pve|sharedpve)\w*path)$`)

// baseRacineDemo : texte d'une base de Join qui désigne la racine démo.
var baseRacineDemo = regexp.MustCompile(`(?i)demo|fixture`)

// segmentsDeLaDispositionDemo : segments que seule la disposition a le droit d'ajouter.
var segmentsDeLaDispositionDemo = map[string]bool{
	"warehouse": true, "titles": true, "players": true, "auth": true, "runtime": true,
}

// fichiersDeLArbreDemo : fichiers qui fabriquent ou résolvent l'arbre démo — règle 3.
var fichiersDeLArbreDemo = regexp.MustCompile(
	`^(internal/ops/seed_demo[^/]*\.go|internal/config/player_resolver\.go|internal/config/config_players\.go|cmd/server/demo_paths\.go)$`)

func TestNoDemoLayoutTranslationOutsideHelper(t *testing.T) {
	_, thisFile, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("runtime.Caller a échoué")
	}
	goAPIRoot := filepath.Dir(filepath.Dir(filepath.Dir(thisFile)))

	var violations []string
	canoniqueVu := false
	for _, sub := range []string{"internal", "cmd"} {
		err := filepath.WalkDir(filepath.Join(goAPIRoot, sub), func(path string, d os.DirEntry, err error) error {
			if err != nil {
				return err
			}
			if d.IsDir() || !strings.HasSuffix(path, ".go") || strings.HasSuffix(path, "_test.go") {
				return nil
			}
			rel, _ := filepath.Rel(goAPIRoot, path)
			rel = filepath.ToSlash(rel)
			if rel == dispositionDemoCanonique {
				canoniqueVu = true
				return nil
			}
			data, readErr := os.ReadFile(path)
			if readErr != nil {
				return readErr
			}
			violations = append(violations, traductionsDemoDuFichier(t, rel, data)...)
			return nil
		})
		if err != nil {
			t.Fatalf("walk %s: %v", sub, err)
		}
	}
	if !canoniqueVu {
		t.Fatalf("%s introuvable : la disposition démo a déménagé, mettre ce garde-rail à jour", dispositionDemoCanonique)
	}
	sort.Strings(violations)
	if len(violations) > 0 {
		t.Errorf("traduction de la disposition démo hors de %s (règle n°6) — passer par title.DemoLayout :\n  %s",
			dispositionDemoCanonique, strings.Join(violations, "\n  "))
	}
}

// traductionsDemoDuFichier rend les sites interdits d'un fichier source (règles 1 à 3).
func traductionsDemoDuFichier(t *testing.T, rel string, data []byte) []string {
	t.Helper()
	bas := strings.ToLower(string(data))
	arbreDemo := fichiersDeLArbreDemo.MatchString(rel)
	if !arbreDemo && !strings.Contains(bas, "demo") && !strings.Contains(bas, "fixture") {
		return nil
	}
	fset := token.NewFileSet()
	f, err := parser.ParseFile(fset, rel, data, parser.SkipObjectResolution)
	if err != nil {
		t.Fatalf("parse %s: %v", rel, err)
	}
	var out []string
	ast.Inspect(f, func(n ast.Node) bool {
		switch x := n.(type) {
		case *ast.FuncDecl:
			if nomDeTraducteurDemo.MatchString(x.Name.Name) {
				out = append(out, rel+":"+strconv.Itoa(fset.Position(x.Pos()).Line)+
					" fonction "+x.Name.Name+" (traducteur de disposition démo)")
			}
		case *ast.CallExpr:
			if !estFilepathJoin(x) || len(x.Args) < 2 || !ajouteUnSegmentDemo(x.Args[1:]) {
				return true
			}
			// Base littérale = chemin RELATIF, enraciné nulle part (ex. le champ db_path que le
			// seed inscrit dans le db_profiles.json de la fixture, que le résolveur démo ignore) :
			// ce n'est pas une traduction de la racine démo.
			if lit, isLit := x.Args[0].(*ast.BasicLit); isLit && lit.Kind == token.STRING {
				return true
			}
			base := string(data[fset.Position(x.Args[0].Pos()).Offset:fset.Position(x.Args[0].End()).Offset])
			if arbreDemo || baseRacineDemo.MatchString(base) {
				out = append(out, rel+":"+strconv.Itoa(fset.Position(x.Pos()).Line)+
					" filepath.Join("+base+", …) ajoute un segment de la disposition démo")
			}
		}
		return true
	})
	return out
}

func estFilepathJoin(c *ast.CallExpr) bool {
	sel, ok := c.Fun.(*ast.SelectorExpr)
	if !ok || sel.Sel.Name != "Join" {
		return false
	}
	id, ok := sel.X.(*ast.Ident)
	return ok && id.Name == "filepath"
}

func ajouteUnSegmentDemo(args []ast.Expr) bool {
	for _, a := range args {
		lit, ok := a.(*ast.BasicLit)
		if !ok || lit.Kind != token.STRING {
			continue
		}
		if v, err := strconv.Unquote(lit.Value); err == nil && segmentsDeLaDispositionDemo[v] {
			return true
		}
	}
	return false
}
