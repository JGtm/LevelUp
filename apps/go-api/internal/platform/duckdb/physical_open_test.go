package duckdb_test

// physical_open_test.go — l'ouverture physique en écriture d'une base joueur aligne ses
// séquences avant de publier le handle (contrat : physical_open.go). Base de chaque test : une
// séquence msr_seq qui rend 1 alors que les ids 1..3 de sa colonne sont pris, préparée hors du
// cache du paquet puis fermée.

import (
	"context"
	"database/sql"
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"strings"
	"testing"

	_ "github.com/duckdb/duckdb-go/v2"

	titlePkg "levelup/go-api/internal/domain/title"
	"levelup/go-api/internal/observability"
	duckdbpkg "levelup/go-api/internal/platform/duckdb"
)

const (
	careCounter        = "duckdb_player_open_care_total"
	alignFailedCounter = "duckdb_sequence_align_failed_total"
	insertSansID       = `INSERT INTO msr (match_id) VALUES ('m-neuf')`
)

// lagSQL — séquence en retard sur sa colonne : prochaine valeur 1, ids 1..3 déjà pris.
var lagSQL = []string{
	`CREATE SEQUENCE msr_seq START 1`,
	`CREATE TABLE msr (id BIGINT PRIMARY KEY DEFAULT nextval('msr_seq'), match_id VARCHAR)`,
	`INSERT INTO msr (id, match_id) SELECT range + 1, 'legacy-' || range FROM range(3)`,
}

// playerPath rend un chemin de la forme d'une base joueur, sous un répertoire temporaire.
func playerPath(t *testing.T) string {
	t.Helper()
	return titlePkg.NewPathResolver(t.TempDir()).PlayerDBPath(titlePkg.DefaultSlug, "Retard")
}

// prepareFile crée path hors du cache du paquet, y exécute stmts, puis ferme le fichier.
func prepareFile(t *testing.T, path string, stmts ...string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	db, err := sql.Open("duckdb", path)
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	defer func() { _ = db.Close() }()
	for _, s := range stmts {
		if _, err := db.Exec(s); err != nil {
			t.Fatalf("préparation %q: %v", s, err)
		}
	}
}

func mustInsertSansID(t *testing.T, h *duckdbpkg.DB, what string) {
	t.Helper()
	if _, err := h.SQLDb().ExecContext(context.Background(), insertSansID); err != nil {
		t.Fatalf("%s : insertion sans id refusée : %v", what, err)
	}
}

func mustCollideSansID(t *testing.T, h *duckdbpkg.DB, what string) {
	t.Helper()
	_, err := h.SQLDb().ExecContext(context.Background(), insertSansID)
	if err == nil || !strings.Contains(err.Error(), "Duplicate key") {
		t.Fatalf("%s : collision attendue (séquence non alignée), err=%v", what, err)
	}
}

// Les deux ouvreurs en écriture (pool mono-connexion et pool partagé, même clé de cache)
// alignent la base joueur à l'ouverture physique.
func TestOuverturePhysique_BaseJoueurEnEcriture_AligneLesSequences(t *testing.T) {
	for name, open := range map[string]func(string, ...string) (*duckdbpkg.DB, error){
		"OpenReadWrite":       duckdbpkg.OpenReadWrite,
		"OpenReadWriteShared": duckdbpkg.OpenReadWriteShared,
	} {
		t.Run(name, func(t *testing.T) {
			path := playerPath(t)
			prepareFile(t, path, lagSQL...)
			h, err := open(path)
			if err != nil {
				t.Fatalf("%s: %v", name, err)
			}
			t.Cleanup(func() { _ = h.Close() })
			mustInsertSansID(t, h, name)
		})
	}
}

// Une fois par ouverture PHYSIQUE : un emprunt du handle en cache ne repasse pas par le soin ;
// la réouverture après fermeture complète, si.
func TestOuverturePhysique_UneFoisParOuverturePhysique(t *testing.T) {
	path := playerPath(t)
	prepareFile(t, path, lagSQL...)
	base := observability.LoadCounter(careCounter)

	h1, err := duckdbpkg.OpenReadWrite(path)
	if err != nil {
		t.Fatalf("ouverture 1: %v", err)
	}
	if d := observability.LoadCounter(careCounter) - base; d != 1 {
		t.Fatalf("ouverture physique : %s +%d, attendu +1", careCounter, d)
	}
	h2, err := duckdbpkg.OpenReadWrite(path)
	if err != nil {
		t.Fatalf("emprunt: %v", err)
	}
	if d := observability.LoadCounter(careCounter) - base; d != 1 {
		t.Fatalf("emprunt du cache : %s +%d, attendu +1 (pas de second soin)", careCounter, d)
	}
	_ = h2.Close()
	_ = h1.Close()

	// Retard recréé hors du cache, fichier fermé : la prochaine ouverture est physique.
	prepareFile(t, path, append([]string{`DROP TABLE msr`, `DROP SEQUENCE msr_seq`}, lagSQL...)...)
	h3, err := duckdbpkg.OpenReadWrite(path)
	if err != nil {
		t.Fatalf("réouverture: %v", err)
	}
	t.Cleanup(func() { _ = h3.Close() })
	if d := observability.LoadCounter(careCounter) - base; d != 2 {
		t.Fatalf("réouverture physique : %s +%d, attendu +2", careCounter, d)
	}
	mustInsertSansID(t, h3, "réouverture")
}

// Le soin est réservé aux bases joueur en écriture : une autre base (partagée, metadata…) ou
// une ouverture en lecture n'est ni alignée ni comptée.
func TestOuverturePhysique_HorsBaseJoueurOuEnLecture_SansSoin(t *testing.T) {
	base := observability.LoadCounter(careCounter)

	shared := titlePkg.NewPathResolver(t.TempDir()).SharedDBPath(titlePkg.DefaultSlug)
	prepareFile(t, shared, lagSQL...)
	hs, err := duckdbpkg.OpenReadWrite(shared)
	if err != nil {
		t.Fatalf("base partagée: %v", err)
	}
	t.Cleanup(func() { _ = hs.Close() })
	mustCollideSansID(t, hs, "base partagée")

	player := playerPath(t)
	prepareFile(t, player, lagSQL...)
	ro, err := duckdbpkg.OpenReadOnly(player)
	if err != nil {
		t.Fatalf("lecture: %v", err)
	}
	_ = ro.Close()

	if d := observability.LoadCounter(careCounter) - base; d != 0 {
		t.Fatalf("%s +%d, attendu +0 hors base joueur en écriture", careCounter, d)
	}
}

// Reopen (après invalidation) est une ouverture physique : la base joueur y est réalignée.
func TestOuverturePhysique_ReopenRealigne(t *testing.T) {
	path := playerPath(t)
	prepareFile(t, path, lagSQL[0], lagSQL[1])
	h, err := duckdbpkg.OpenReadWrite(path)
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	t.Cleanup(func() { _ = h.Close() })
	if _, err := h.SQLDb().Exec(lagSQL[2]); err != nil {
		t.Fatalf("pose des ids: %v", err)
	}
	mustCollideSansID(t, h, "avant Reopen")
	if err := h.Reopen(); err != nil {
		t.Fatalf("Reopen: %v", err)
	}
	mustInsertSansID(t, h, "après Reopen")
}

// Un alignement en échec (avance au-delà de MAXVALUE) ne bloque jamais l'ouverture : le
// handle est rendu, l'échec compté, et la séquence voisine alignée quand même.
func TestOuverturePhysique_EchecDAlignementNonBloquant(t *testing.T) {
	path := playerPath(t)
	prepareFile(t, path, append([]string{
		`CREATE SEQUENCE zz_borne MAXVALUE 100`,
		`CREATE TABLE zz_borne_t (id BIGINT DEFAULT nextval('zz_borne'))`,
		`INSERT INTO zz_borne_t (id) VALUES (100)`,
	}, lagSQL...)...)
	before := observability.LoadCounter(alignFailedCounter)
	h, err := duckdbpkg.OpenReadWrite(path)
	if err != nil {
		t.Fatalf("l'échec d'alignement ne doit pas bloquer l'ouverture : %v", err)
	}
	t.Cleanup(func() { _ = h.Close() })
	if d := observability.LoadCounter(alignFailedCounter) - before; d != 1 {
		t.Fatalf("%s +%d, attendu +1", alignFailedCounter, d)
	}
	mustInsertSansID(t, h, "séquence voisine")
}

// TestOuverturePhysiqueUnique — garde-rail : dans le paquet, la connexion DuckDB ne se
// construit qu'en un endroit (openSQLDBFor, seul porteur de NewConnector / sql.OpenDB /
// sql.Open), et openSQLDBFor n'est appelé que par openPhysicalSQLDB, qui porte le soin des
// bases joueur. Une nouvelle ouverture physique qui contournerait le soin échoue ici.
func TestOuverturePhysiqueUnique(t *testing.T) {
	fset := token.NewFileSet()
	files, err := filepath.Glob("*.go")
	if err != nil {
		t.Fatal(err)
	}
	type site struct{ fn, file string }
	callers := map[string][]site{} // appelé -> sites (fonction englobante, fichier)
	for _, f := range files {
		if strings.HasSuffix(f, "_test.go") {
			continue
		}
		af, err := parser.ParseFile(fset, f, nil, 0)
		if err != nil {
			t.Fatalf("parse %s: %v", f, err)
		}
		for _, d := range af.Decls {
			fd, ok := d.(*ast.FuncDecl)
			if !ok || fd.Body == nil {
				continue
			}
			ast.Inspect(fd.Body, func(n ast.Node) bool {
				call, ok := n.(*ast.CallExpr)
				if !ok {
					return true
				}
				if name := calleeName(call); name != "" {
					callers[name] = append(callers[name], site{fd.Name.Name, f})
				}
				return true
			})
		}
	}
	want := map[string]string{
		"openSQLDBFor":       "openPhysicalSQLDB",
		"NewConnector":       "openSQLDBFor",
		"sql.OpenDB":         "openSQLDBFor",
		"sql.Open":           "",
		"careOnPhysicalOpen": "openPhysicalSQLDB",
	}
	for callee, only := range want {
		for _, s := range callers[callee] {
			if s.fn != only {
				t.Errorf("%s appelé par %s (%s) : seul %q y a droit — toute ouverture physique passe par "+
					"openPhysicalSQLDB (physical_open.go), qui porte le soin des bases joueur", callee, s.fn, s.file, only)
			}
		}
	}
	if len(callers["openSQLDBFor"]) != 1 || len(callers["careOnPhysicalOpen"]) != 1 {
		t.Errorf("openSQLDBFor et careOnPhysicalOpen : un seul site d'appel chacun, obtenu %d et %d",
			len(callers["openSQLDBFor"]), len(callers["careOnPhysicalOpen"]))
	}
}

// calleeName rend le nom d'un appel : `f` pour f(...), `pkg.F` pour sql.X(...), `F` pour tout
// autre sélecteur (méthode ou paquet aliasé, ex. duckdb.NewConnector).
func calleeName(call *ast.CallExpr) string {
	switch fn := call.Fun.(type) {
	case *ast.Ident:
		return fn.Name
	case *ast.SelectorExpr:
		if x, ok := fn.X.(*ast.Ident); ok && x.Name == "sql" {
			return "sql." + fn.Sel.Name
		}
		return fn.Sel.Name
	}
	return ""
}
