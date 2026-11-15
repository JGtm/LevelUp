// Package archlint — frontieres_sql_test.go : garde-rail « le câblage HTTP ne porte pas de SQL ».
//
// `internal/api/wire` (le registre de services) et la racine de `internal/api` assemblent des dépôts et des
// services ; ils ne lisent ni n'écrivent la base. Toute requête vit dans `platform/duckdb` (lectures, derrière
// un port de `internal/port`), `persist/` et `migration/` (écritures et DDL), ou dans `ops/` et `sync/`.
// Changer de moteur de base ne doit pas toucher la couche HTTP.
//
// Le garde-rail lit l'AST : il refuse (1) l'import de `database/sql`, (2) un littéral de chaîne qui est une
// instruction SQL (toute casse, WITH, MERGE et TRUNCATE compris), (3) l'appel d'une méthode d'exécution de
// requête : avec contexte (QueryContext, ExecContext...) ou sans (Query, QueryRow, Exec, Prepare, reconnus par
// heuristique : voir requeteSansContexte). Limite assumée : un SQL assemblé à l'exécution à partir de morceaux
// qui ne sont pas des instructions ne se voit pas.
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

// instructionSQL reconnaît un littéral qui COMMENCE par une instruction SQL, quelle que soit la casse : INSERT
// INTO, UPDATE x SET, DELETE FROM, CREATE / ALTER / DROP, WITH x AS (, MERGE INTO, TRUNCATE TABLE, un SELECT qui
// désigne ses colonnes (`*`, DISTINCT, COUNT(, liste, ou colonne puis FROM), PRAGMA, ATTACH, DETACH, CHECKPOINT,
// COPY x TO. Chaque forme exige son second mot-clé : un texte français ou anglais (« Select a player from the
// list ») ne les satisfait pas.
var instructionSQL = regexp.MustCompile(
	`(?i)^\s*(INSERT\s+(OR\s+\w+\s+)?INTO\s|UPDATE\s+\w+\s+SET\s|DELETE\s+FROM\s|` +
		`CREATE\s+(OR\s+REPLACE\s+)?(TEMP\s+|TEMPORARY\s+)?(TABLE|VIEW|INDEX|SEQUENCE)\s|ALTER\s+TABLE\s|` +
		`DROP\s+(TABLE|VIEW|INDEX|SEQUENCE)\s|WITH\s+(RECURSIVE\s+)?\w+\s+AS\s*\(|MERGE\s+INTO\s|TRUNCATE\s+TABLE\s|` +
		`SELECT\s+(\*|DISTINCT\s|COUNT\s*\(|[\w.]+(\s+AS\s+\w+)?\s*,|[\w.]+\s+FROM\s)|` +
		`PRAGMA\s+\w|ATTACH\s+['"]|DETACH\s+\w|CHECKPOINT\s*;?\s*$|COPY\s+\w+\s+TO\s)` +
		`|(?-i:^\s*SELECT\b|\bSELECT\b[\s\S]*\bFROM\b)`)

// methodesDeRequete : les méthodes qui exécutent une requête sur un handle de base, avec contexte.
var methodesDeRequete = map[string]bool{
	"QueryContext": true, "ExecContext": true, "QueryRowContext": true,
	"QueryRecovered": true, "QueryRowRecovered": true, "PrepareContext": true, "BeginTx": true,
}

// methodesDeRequeteSansContexte : les mêmes sans contexte (`database/sql` : Query, QueryRow, Exec, Prepare ;
// l'enveloppe `duckdb.DB` les prend avec un contexte en premier argument).
var methodesDeRequeteSansContexte = map[string]bool{"Query": true, "QueryRow": true, "Exec": true, "Prepare": true}

// recepteurDeBase reconnaît, par son dernier nom, un récepteur qui est un handle de base : `db`, `sqlDB`,
// `sharedSQL`, `metaDB`, `tx`, `conn`, `pdb.Player`, `Shared`, `Metadata`, `SharedSocial`.
var recepteurDeBase = regexp.MustCompile(`(?i)(db|sql|tx|conn|player|shared|metadata|social)$`)

// requeteSansContexte dit si un appel Query / QueryRow / Exec / Prepare est une requête de base. HEURISTIQUE sans
// types : il faut au moins un argument (`r.URL.Query()` de net/http n'en a pas), et soit un premier argument qui
// n'est pas un contexte (forme de `database/sql`), soit un récepteur nommé comme un handle de base (forme
// `db.Query(ctx, requête)`). Un faux positif connu est écarté par construction : `h.Query(ctx, q, live)` d'un
// handler de recherche (api/huma_routes.go) n'a pas de récepteur de base et passe un contexte.
func requeteSansContexte(sel *ast.SelectorExpr, args []ast.Expr) bool {
	if !methodesDeRequeteSansContexte[sel.Sel.Name] || len(args) == 0 {
		return false
	}
	if id, ok := args[0].(*ast.Ident); !ok || id.Name != "ctx" {
		return true
	}
	switch x := sel.X.(type) {
	case *ast.Ident:
		return recepteurDeBase.MatchString(x.Name)
	case *ast.SelectorExpr:
		return recepteurDeBase.MatchString(x.Sel.Name)
	}
	return false
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
			sel, ok := v.Fun.(*ast.SelectorExpr)
			if ok && (methodesDeRequete[sel.Sel.Name] || requeteSansContexte(sel, v.Args)) {
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
func f(db *sql.DB, r *http.Request) {
	_ = fmt.Sprint("load players: SELECT en minuscules ou texte, select * from t")
	_, _ = db.QueryContext(nil, "SELECT 1")
	_, _ = db.ExecContext(nil, "insert into t values (1)")
	_ = "CREATE TABLE IF NOT EXISTS t (a INT)"
	_ = "WITH base AS (SELECT 1) SELECT * FROM base"
	_ = "x " + "SELECT a" + "FROM t"
	_, _ = db.Query("x")
	_ = db.QueryRow("x")
	_, _ = db.Exec("x")
	_, _ = db.Prepare("x")
	_ = r.URL.Query()
	_, _ = pdb.Player.Query(ctx, q)
	_, _ = h.Query(ctx, q, live)
	_ = "merge into t using u on (1=1)"
	_ = "truncate table t"
	_ = "with recursive r as (select 1)"
	_ = "select a, b from t"
	_ = "Select a player from the list"
	_ = "with it as a guide"
	_ = "Update the settings"
}
`)
	got := strings.Join(sqlDansSource(src), "|")
	want := strings.Join([]string{
		"appel de Exec ligne 18", "appel de ExecContext ligne 12", "appel de Prepare ligne 19", "appel de Query ligne 16",
		"appel de Query ligne 21", "appel de QueryContext ligne 11", "appel de QueryRow ligne 17", "import de database/sql",
		"littéral SQL ligne 11", "littéral SQL ligne 12", "littéral SQL ligne 13", "littéral SQL ligne 14", "littéral SQL ligne 15",
		"littéral SQL ligne 23", "littéral SQL ligne 24", "littéral SQL ligne 25", "littéral SQL ligne 26",
	}, "|")
	if got != want {
		t.Fatalf("détecté\n  %s\nattendu\n  %s\n(ni le commentaire, ni la prose, ni r.URL.Query(), ni h.Query(ctx, ...) d'un handler ne sont du SQL)", got, want)
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

// racinesMetierSansSQL : la couche métier et ses abords, sous-dossiers compris. Aucune ne parle au moteur de
// base : `service/` orchestre, `analysis/` calcule, `progression/` et `prestige/` portent des règles,
// `scheduler/` cadence des tâches, `api/` (handlers et middleware compris) décode et encode. Toute lecture et
// toute écriture passent par un port de `internal/port`, implémenté dans `platform/duckdb`, `persist`,
// `importstore`, `cronstore` ou `migration`.
var racinesMetierSansSQL = []string{
	"internal/service", "internal/analysis", "internal/progression", "internal/scheduler", "internal/prestige", "internal/api",
}

// metierAvecSQLAutorises : fichiers de ces dossiers qui portent encore du SQL ou `database/sql`, avec la date
// et la raison. Vide depuis le 2026-11-12 : plafond 0. Une entrée ne s'ajoute pas, elle se supprime avec le
// code qui la justifiait ; une entrée sans violation ou sans fichier fait échouer le test.
var metierAvecSQLAutorises = map[string]string{}

// TestMetierSansSQL : aucun fichier non-test de la couche métier n'importe `database/sql`, ne porte de
// littéral SQL ni n'appelle une méthode de requête (même détecteur que TestApiSansSQL), hors
// `metierAvecSQLAutorises`.
func TestMetierSansSQL(t *testing.T) {
	var violations []string
	vus := map[string]bool{}
	for _, src := range goSourcesUnder(t, racinesMetierSansSQL...) {
		vus[src.Rel] = true
		trouves := sqlDansSource(src)
		_, autorise := metierAvecSQLAutorises[src.Rel]
		switch {
		case len(trouves) == 0 && autorise:
			violations = append(violations, src.Rel+" : entrée autorisée sans SQL — la retirer de metierAvecSQLAutorises")
		case len(trouves) > 0 && !autorise:
			violations = append(violations, src.Rel+" : "+strings.Join(trouves, ", "))
		}
	}
	for rel := range metierAvecSQLAutorises {
		if !vus[rel] {
			violations = append(violations, rel+" : entrée autorisée sans fichier — la retirer de metierAvecSQLAutorises")
		}
	}
	sort.Strings(violations)
	if len(violations) > 0 {
		t.Errorf("du SQL ou database/sql dans la couche métier — passer par un port de internal/port :\n  %s",
			strings.Join(violations, "\n  "))
	}
}

// portAvecTypeSQLAutorises : fichiers de `port/` qui importent encore `database/sql`. Vide depuis le
// 2026-11-12 (DBExecutor et DBWriter vivent dans platform/dblease) : plafond 0.
var portAvecTypeSQLAutorises = map[string]string{}

// TestPortSansTypeSQL : aucun fichier non-test de `internal/port` n'importe `database/sql` : une interface de
// port ne nomme ni `*sql.DB`, ni `*sql.Tx`, ni `*sql.Rows`, ni `sql.Result`. Changer de moteur de base ne
// change pas les contrats.
func TestPortSansTypeSQL(t *testing.T) {
	var violations []string
	vus := map[string]bool{}
	for _, src := range goSourcesUnder(t, "internal/port") {
		vus[src.Rel] = true
		trouves := importsInterdits(src, []string{"database/sql"})
		_, autorise := portAvecTypeSQLAutorises[src.Rel]
		switch {
		case len(trouves) == 0 && autorise:
			violations = append(violations, src.Rel+" : entrée autorisée sans violation — la retirer de portAvecTypeSQLAutorises")
		case len(trouves) > 0 && !autorise:
			violations = append(violations, src.Rel+" importe database/sql")
		}
	}
	for rel := range portAvecTypeSQLAutorises {
		if !vus[rel] {
			violations = append(violations, rel+" : entrée autorisée sans fichier — la retirer de portAvecTypeSQLAutorises")
		}
	}
	sort.Strings(violations)
	if len(violations) > 0 {
		t.Errorf("un port nomme un type SQL — l'interface doit parler métier :\n  %s", strings.Join(violations, "\n  "))
	}
}
