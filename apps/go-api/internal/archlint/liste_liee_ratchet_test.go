// Package archlint — liste_liee_ratchet_test.go : RATCHET du paramètre liste `?::VARCHAR[]`
// (item B.7 du plan perf « compaction et périmètre joueur », 2026-09-27 ; règle CLAUDE.md n°6).
//
// Une liste liée en UN paramètre a deux formes, et le choix n'est pas libre (constante sous une
// fenêtre `_latest`, semi-jointure partout ailleurs : cf. l'en-tête de analysis/sql_liste.go).
// Leur texte vit à UN endroit, `analysis/sql_liste.go` ; les lectures l'appellent (directement dans
// `analysis`, par `clauseListe…` dans `platform/duckdb`). Ce test compte, fichier par fichier, les
// littéraux de chaîne Go qui contiennent `::VARCHAR[]` sous internal/ (hors tests) : une nouvelle
// copie à la main le fait échouer, et le retrait d'une copie consignée aussi (la table descend).
package archlint

import (
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"testing"
)

// listeLieeAutorisee : fichier (relatif à apps/go-api) -> littéraux permis, datée du 2026-09-27.
var listeLieeAutorisee = map[string]int{
	// La source unique des deux formes.
	"internal/analysis/sql_liste.go": 1,
	// Deux semi-jointures `match_id IN (SELECT UNNEST(?::VARCHAR[]))` antérieures au helper, non
	// migrées au lot B (hors de son périmètre) : à passer par clauseListeParJointure, puis à retirer.
	"internal/platform/duckdb/fanout_repo.go": 2,
}

func TestListeLiee_TexteEnUnSeulEndroit(t *testing.T) {
	_, ici, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("runtime.Caller a échoué")
	}
	goAPI := filepath.Join(filepath.Dir(ici), "..", "..")
	fset := token.NewFileSet()
	trouves := map[string]int{}
	err := filepath.WalkDir(filepath.Join(goAPI, "internal"), func(chemin string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() || !strings.HasSuffix(chemin, ".go") || strings.HasSuffix(chemin, "_test.go") {
			return nil
		}
		f, err := parser.ParseFile(fset, chemin, nil, parser.SkipObjectResolution)
		if err != nil {
			return err
		}
		ast.Inspect(f, func(n ast.Node) bool {
			lit, ok := n.(*ast.BasicLit)
			if !ok || lit.Kind != token.STRING {
				return true
			}
			if texte, err := strconv.Unquote(lit.Value); err == nil && strings.Contains(texte, "::VARCHAR[]") {
				rel, _ := filepath.Rel(goAPI, chemin)
				trouves[filepath.ToSlash(rel)]++
			}
			return true
		})
		return nil
	})
	if err != nil {
		t.Fatalf("parcours de internal/ : %v", err)
	}
	for fichier, n := range trouves {
		if permis := listeLieeAutorisee[fichier]; n > permis {
			t.Errorf("%s : %d littéral(aux) `::VARCHAR[]`, %d permis — passer par analysis/sql_liste.go "+
				"(SQLDansListe, SQLDansListeParJointure) ou clauseListe… de platform/duckdb", fichier, n, permis)
		}
	}
	for fichier, permis := range listeLieeAutorisee {
		if n := trouves[fichier]; n < permis {
			t.Errorf("%s : %d littéral(aux) `::VARCHAR[]` pour %d permis — descendre la table", fichier, n, permis)
		}
	}
}
