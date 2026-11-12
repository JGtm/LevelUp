// Package archlint — frontieres_sql_test.go : garde-rail « le câblage HTTP ne porte pas de SQL ».
//
// `internal/api/wire` (le registre de services) et la racine de `internal/api` assemblent des dépôts et des
// services ; ils ne lisent ni n'écrivent la base. Toute requête vit dans `platform/duckdb` (lectures, derrière
// un port de `internal/port`), `persist/` et `migration/` (écritures et DDL), ou dans `ops/` et `sync/`.
// Changer de moteur de base ne doit pas toucher la couche HTTP.
//
// Le garde-rail lit l'AST : il refuse (1) l'import de `database/sql`, (2) un littéral de chaîne qui est une
// instruction SQL, (3) l'appel d'une méthode d'exécution de requête (QueryContext, ExecContext...).
package archlint

import (
	"go/ast"
	"path"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"testing"
)

// racinesApiSansSQL : les deux dossiers couverts, sans leurs sous-dossiers (`api/handlers`, `api/middleware`
// ont leurs propres garde-rails de frontière).
var racinesApiSansSQL = []string{"internal/api", "internal/api/wire"}

// apiAvecSQLAutorises : fichiers de ces dossiers qui portent encore du SQL ou `database/sql`, avec la date et
// la raison. Vide : plafond 0. Une entrée ne s'ajoute pas, elle se supprime avec le code qui la justifiait ;
// une entrée sans violation ou sans fichier fait échouer le test.
var apiAvecSQLAutorises = map[string]string{}

// instructionSQL reconnaît un littéral qui COMMENCE par une instruction SQL (majuscules : c'est l'écriture de
// tout le dépôt), ou qui contient un `SELECT ... FROM`. Un texte français ou un message de journal ne les
// satisfait pas.
var instructionSQL = regexp.MustCompile(
	`^\s*(SELECT|INSERT\s+(OR\s+\w+\s+)?INTO|UPDATE\s+\w+\s+SET|DELETE\s+FROM|` +
		`CREATE\s+(OR\s+REPLACE\s+)?(TEMP\s+|TEMPORARY\s+)?(TABLE|VIEW|INDEX|SEQUENCE)|ALTER\s+TABLE|` +
		`DROP\s+(TABLE|VIEW|INDEX|SEQUENCE)|WITH\s+\w+\s+AS|PRAGMA|ATTACH|DETACH|CHECKPOINT|COPY\s+\w+\s+TO)\b` +
		`|\bSELECT\b[\s\S]*\bFROM\b`)

// methodesDeRequete : les méthodes qui exécutent une requête sur un handle de base.
var methodesDeRequete = map[string]bool{
	"QueryContext": true, "ExecContext": true, "QueryRowContext": true,
	"QueryRecovered": true, "QueryRowRecovered": true, "PrepareContext": true, "BeginTx": true,
}

// sqlDansSource rend, triées, les formes de SQL d'un fichier : import de database/sql, littéral SQL (avec
// sa ligne), appel de méthode de requête.
func sqlDansSource(src goSource) []string {
	var out []string
	for _, imp := range src.File.Imports {
		if chemin, err := strconv.Unquote(imp.Path.Value); err == nil && chemin == "database/sql" {
			out = append(out, "import de database/sql")
		}
	}
	ast.Inspect(src.File, func(n ast.Node) bool {
		switch v := n.(type) {
		case *ast.BasicLit:
			if v.Kind.String() != "STRING" {
				return true
			}
			if s, err := strconv.Unquote(v.Value); err == nil && instructionSQL.MatchString(s) {
				out = append(out, "littéral SQL ligne "+strconv.Itoa(src.Fset.Position(v.Pos()).Line))
			}
		case *ast.CallExpr:
			if sel, ok := v.Fun.(*ast.SelectorExpr); ok && methodesDeRequete[sel.Sel.Name] {
				out = append(out, "appel de "+sel.Sel.Name+" ligne "+strconv.Itoa(src.Fset.Position(v.Pos()).Line))
			}
		}
		return true
	})
	sort.Strings(out)
	return out
}

func TestOutilSQLDansSource(t *testing.T) {
	src := parseGoSnippet(t, "x.go", `package x

import (
	"database/sql"
	"fmt"
)

// SELECT x FROM y : un commentaire n'est pas du SQL
func f(db *sql.DB) {
	_ = fmt.Sprint("load players: SELECT en minuscules ou texte, select * from t")
	_, _ = db.QueryContext(nil, "SELECT 1")
	_, _ = db.ExecContext(nil, "insert into t values (1)")
	_ = "CREATE TABLE IF NOT EXISTS t (a INT)"
	_ = "WITH base AS (SELECT 1) SELECT * FROM base"
	_ = "x " + "SELECT a" + "FROM t"
}
`)
	got := strings.Join(sqlDansSource(src), "|")
	want := strings.Join([]string{
		"appel de ExecContext ligne 12", "appel de QueryContext ligne 11", "import de database/sql",
		"littéral SQL ligne 11", "littéral SQL ligne 13", "littéral SQL ligne 14", "littéral SQL ligne 15",
	}, "|")
	if got != want {
		t.Fatalf("détecté\n  %s\nattendu\n  %s\n(ni le commentaire, ni le texte, ni le SQL en minuscules d'une chaîne ne sont du SQL)", got, want)
	}
}

// TestApiSansSQL : ni `internal/api` (racine) ni `internal/api/wire` ne portent de SQL ni n'importent
// `database/sql`, hors `apiAvecSQLAutorises`.
func TestApiSansSQL(t *testing.T) {
	racines := map[string]bool{}
	for _, r := range racinesApiSansSQL {
		racines[r] = true
	}
	var violations []string
	vus := map[string]bool{}
	for _, src := range goSourcesUnder(t, "internal/api") {
		if !racines[path.Dir(src.Rel)] {
			continue
		}
		vus[src.Rel] = true
		trouves := sqlDansSource(src)
		if _, ok := apiAvecSQLAutorises[src.Rel]; ok || len(trouves) == 0 {
			continue
		}
		violations = append(violations, src.Rel+" : "+strings.Join(trouves, ", "))
	}
	for rel := range apiAvecSQLAutorises {
		if !vus[rel] {
			violations = append(violations, rel+" : entrée autorisée sans fichier — la retirer de apiAvecSQLAutorises")
			continue
		}
		for _, src := range goSourcesUnder(t, path.Dir(rel)) {
			if src.Rel == rel && len(sqlDansSource(src)) == 0 {
				violations = append(violations, rel+" : entrée autorisée sans SQL — la retirer de apiAvecSQLAutorises")
			}
		}
	}
	if len(violations) > 0 {
		t.Errorf("du SQL ou database/sql dans le câblage HTTP — passer par un port de internal/port, "+
			"implémenté dans platform/duckdb, persist ou migration :\n  %s", strings.Join(violations, "\n  "))
	}
}
