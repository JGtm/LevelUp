// Package archlint — known_set_single_source_test.go : garde-rail de l'ENSEMBLE DES MATCHS
// CONNUS de la sync (CLAUDE.md règle 6).
//
// La sync décide « match déjà connu, ne pas le récupérer » par UNE règle,
// `internal/sync/knownset.Load` : connu seulement s'il est au registre partagé pour le joueur ;
// base partagée illisible = erreur typée, jamais un ensemble partiel. Le moteur V1
// (`internal/sync/engine.go`) et le pipeline V2 (`internal/sync/v2/known_loader.go`) l'appellent.
// Une seconde construction de l'ensemble est une seconde règle, libre de diverger : un match que
// l'une dit connu et l'autre non est sauté par un moteur et récupéré par l'autre.
//
// Empreinte d'une construction, par fonction (AST, hors `_test.go`) : la fonction lit une des
// tables de l'ensemble (`match_participants`, `match_registry`, `player_match_enrichment` dans un
// littéral chaîne) ET fabrique une map dont le nom contient « known » (`make(map…)` ou littéral de
// map). Hors du paquet `knownset`, seules les entrées de `constructionsAutorisees` en portent.
package archlint

import (
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"runtime"
	"sort"
	"strconv"
	"strings"
	"testing"
)

// knownSetPackage : la source unique (préfixe de chemin relatif à apps/go-api).
const knownSetPackage = "internal/sync/knownset/"

// knownSetTables : les tables dont la lecture compose l'ensemble connu.
var knownSetTables = []string{"match_participants", "match_registry", "player_match_enrichment"}

// constructionsAutorisees : « fichier:fonction » portant l'empreinte hors du paquet knownset,
// datées et justifiées. Une entrée devenue sans empreinte doit sortir (contrôlé plus bas).
//
//   - internal/games/halo_5/livesync/persist.go:loadKnownMatchIDs (2026-10-08) : arrêt delta du
//     sync live Halo 5, un autre titre et un autre moteur. Sa règle est déjà « connu = au registre
//     partagé » (tout le registre du titre, sans borne par xuid) et sa lecture échoue en erreur ;
//     elle n'a donc pas le défaut corrigé par knownset. L'aligner sur knownset (borne par xuid)
//     est une décision du chantier multi-titre, hors de ce lot.
var constructionsAutorisees = map[string]string{
	"internal/games/halo_5/livesync/persist.go:loadKnownMatchIDs": "Halo 5 : registre entier, autre moteur (2026-10-08)",
}

// appelantsKnownset : les deux moteurs, qui doivent appeler la règle unique.
var appelantsKnownset = []string{
	"internal/sync/engine.go",
	"internal/sync/v2/known_loader.go",
}

func TestKnownSetSingleSource(t *testing.T) {
	_, thisFile, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("runtime.Caller a échoué")
	}
	goAPIRoot := filepath.Dir(filepath.Dir(filepath.Dir(thisFile)))

	trouvees := map[string]bool{}
	var violations []string
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
			if strings.HasPrefix(rel, knownSetPackage) {
				return nil
			}
			fset := token.NewFileSet()
			file, perr := parser.ParseFile(fset, path, nil, 0)
			if perr != nil {
				return perr
			}
			for _, decl := range file.Decls {
				fn, isFunc := decl.(*ast.FuncDecl)
				if !isFunc || fn.Body == nil || !construitEnsembleConnu(fn.Body) {
					continue
				}
				cle := rel + ":" + fn.Name.Name
				if _, autorisee := constructionsAutorisees[cle]; autorisee {
					trouvees[cle] = true
					continue
				}
				violations = append(violations, cle+" (ligne "+strconv.Itoa(fset.Position(fn.Pos()).Line)+")")
			}
			return nil
		})
		if err != nil {
			t.Fatalf("balayage %s/: %v", sub, err)
		}
	}
	sort.Strings(violations)
	if len(violations) > 0 {
		t.Errorf("ensemble des matchs connus reconstruit hors de %s — appeler knownset.Load :\n  %s",
			knownSetPackage, strings.Join(violations, "\n  "))
	}
	for cle := range constructionsAutorisees {
		if !trouvees[cle] {
			t.Errorf("entrée autorisée sans empreinte (supprimée ou renommée) : %s — la retirer de constructionsAutorisees", cle)
		}
	}
	for _, rel := range appelantsKnownset {
		data, err := os.ReadFile(filepath.Join(goAPIRoot, filepath.FromSlash(rel)))
		if err != nil {
			t.Fatalf("lecture %s: %v", rel, err)
		}
		if !strings.Contains(string(data), "knownset.Load(") {
			t.Errorf("%s n'appelle plus knownset.Load : le moteur a perdu la règle unique", rel)
		}
	}
}

// construitEnsembleConnu : le corps lit une table de l'ensemble (littéral chaîne) ET fabrique une
// map nommée « …known… ».
func construitEnsembleConnu(body *ast.BlockStmt) bool {
	litTable, mapKnown := false, false
	ast.Inspect(body, func(n ast.Node) bool {
		switch x := n.(type) {
		case *ast.BasicLit:
			if x.Kind == token.STRING && citeTableConnue(x.Value) {
				litTable = true
			}
		case *ast.AssignStmt:
			for i, lhs := range x.Lhs {
				if id, isIdent := lhs.(*ast.Ident); isIdent && nomKnown(id.Name) && i < len(x.Rhs) && fabriqueMap(x.Rhs[i]) {
					mapKnown = true
				}
			}
		case *ast.ValueSpec:
			for i, name := range x.Names {
				if !nomKnown(name.Name) {
					continue
				}
				if _, isMap := x.Type.(*ast.MapType); isMap {
					mapKnown = true
				}
				if i < len(x.Values) && fabriqueMap(x.Values[i]) {
					mapKnown = true
				}
			}
		}
		return true
	})
	return litTable && mapKnown
}

func citeTableConnue(lit string) bool {
	for _, table := range knownSetTables {
		if strings.Contains(lit, table) {
			return true
		}
	}
	return false
}

func nomKnown(name string) bool {
	return strings.Contains(strings.ToLower(name), "known")
}

// fabriqueMap : `make(map[...]...)` ou littéral `map[...]...{...}`.
func fabriqueMap(expr ast.Expr) bool {
	switch x := expr.(type) {
	case *ast.CallExpr:
		if id, isIdent := x.Fun.(*ast.Ident); isIdent && id.Name == "make" && len(x.Args) > 0 {
			_, isMap := x.Args[0].(*ast.MapType)
			return isMap
		}
	case *ast.CompositeLit:
		_, isMap := x.Type.(*ast.MapType)
		return isMap
	}
	return false
}
