//go:build cgo

// Package ops — tests anti-régression pour le bug WAL corruption shared_social
// (bug DuckDB #7659). Ces tests complètent media_associate_test.go (algo pur)
// avec deux niveaux de protection :
//
//  1. TestNoATTACHInMediaPackage : sentinelle code-level. Parse les fichiers
//     ops/media*.go via go/parser et vérifie qu'aucune string literal SQL ne
//     contient "ATTACH" (sauf allow-listées). Anti-régression triviale : si
//     quelqu'un réintroduit un ATTACH dans ce package, ce test échoue.
//
//  2. TestAssociateMediaWithMatches_RestartCycle : simule le scénario
//     opérationnel "ouvrir via NewConnector(timezone) → Associate → Close →
//     reopen via NewConnector(timezone)". Avec l'ancienne implémentation
//     (ATTACH), ce cycle laissait un WAL non-rejouable. Avec la nouvelle, le
//     reopen doit réussir et les associations doivent persister.
//
// Limite assumée : ces tests ne reproduisent PAS un kill brutal (SIGKILL) du
// process sans CHECKPOINT, qui est la condition exacte du bug observé sous
// Air rebuild Windows. Reproduire ça en `go test` n'est pas faisable car
// db.Close() de Go force un CHECKPOINT propre. La sentinelle code-level
// compense en garantissant qu'aucun ATTACH ne peut être réintroduit dans le
// package.

package ops

import (
	"context"
	"database/sql"
	"database/sql/driver"
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
	"time"

	duckdb "github.com/duckdb/duckdb-go/v2"

	duckdbpkg "levelup/go-api/internal/platform/duckdb"
)

// ─────────────────────────────────────────────────────────────────────────────
// Test 1 — Sentinelle code-level : aucun ATTACH dans ops/media*.go
// ─────────────────────────────────────────────────────────────────────────────

// TestNoATTACHInMediaPackage parse les fichiers Go de ops/media*.go et vérifie
// qu'aucune string literal ne contient le mot "ATTACH" précédé d'un espace ou
// d'un guillemet (motif typique du SQL `ATTACH 'path' AS alias`).
//
// Si quelqu'un réintroduit un ATTACH dans ce package — même pour un cas qui
// "semble inoffensif" — ce test échoue avec un message qui pointe vers
// l'issue DuckDB #7659 et le commit fix initial.
//
// Allow-list : aucun fichier media*.go ne devrait avoir besoin d'ATTACH. Si
// un cas légitime apparaît (ex. backup tool), il faudra modifier ce test
// explicitement et documenter pourquoi le scénario WAL ne s'applique pas.
func TestNoATTACHInMediaPackage(t *testing.T) {
	// Chercher les fichiers media*.go dans le package courant.
	pattern := "media*.go"
	files, err := filepath.Glob(pattern)
	if err != nil {
		t.Fatalf("glob: %v", err)
	}
	if len(files) == 0 {
		t.Fatalf("aucun fichier %s trouvé — le test doit tourner depuis internal/ops/", pattern)
	}

	fset := token.NewFileSet()
	for _, file := range files {
		// Skip les fichiers de test eux-mêmes (qui peuvent contenir "ATTACH"
		// dans des commentaires ou strings de doc).
		if strings.HasSuffix(file, "_test.go") {
			continue
		}

		src, err := os.ReadFile(file)
		if err != nil {
			t.Errorf("read %s: %v", file, err)
			continue
		}

		f, err := parser.ParseFile(fset, file, src, parser.ParseComments)
		if err != nil {
			t.Errorf("parse %s: %v", file, err)
			continue
		}

		ast.Inspect(f, func(n ast.Node) bool {
			lit, ok := n.(*ast.BasicLit)
			if !ok || lit.Kind != token.STRING {
				return true
			}
			// strconv.Unquote gère " et `, et décode les escapes.
			raw, err := strconv.Unquote(lit.Value)
			if err != nil {
				return true
			}
			// Pattern SQL : "ATTACH " suivi d'un délimiteur de chemin (' " ou variable).
			// On veut catcher `ATTACH 'path' AS alias`, `ATTACH "path"`, `ATTACH %s ...`.
			upper := strings.ToUpper(raw)
			if !strings.Contains(upper, "ATTACH ") {
				return true
			}
			// Faux positifs possibles : si le mot ATTACH apparaît dans un message
			// d'erreur (genre "attach failed"). On filtre : il faut un quote ou %
			// après ATTACH.
			idx := strings.Index(upper, "ATTACH ")
			rest := upper[idx+len("ATTACH "):]
			if len(rest) == 0 {
				return true
			}
			next := rest[0]
			if next == '\'' || next == '"' || next == '%' {
				pos := fset.Position(lit.Pos())
				t.Errorf(
					"%s:%d : string literal SQL contient `ATTACH` — cf. bug DuckDB #7659.\n"+
						"  String: %q\n"+
						"  Tout ATTACH cross-DB depuis ops/media*.go écrit une entrée WAL non-rejouable au reboot.\n"+
						"  Fix : faire la jointure cross-DB en Go (cf. computeAssociations dans media_associate.go).",
					pos.Filename, pos.Line, raw,
				)
			}
			return true
		})
	}
}

// ─────────────────────────────────────────────────────────────────────────────
// Test 2 — Cycle Associate → Close → reopen via NewConnector(timezone)
// ─────────────────────────────────────────────────────────────────────────────

// newConnectorWithTimezone reproduit la façon dont le pool ouvre shared_social :
// duckdb.NewConnector avec une init function qui SET TimeZone. C'est ce
// connecteur qui, post-bug, tentait de rejouer un WAL contaminé par ATTACH et
// échouait avec "DatabaseManager::GetDefaultDatabase".
func newConnectorWithTimezone(t *testing.T, path, tz string) *sql.DB {
	t.Helper()
	connector, err := duckdb.NewConnector(path, func(execer driver.ExecerContext) error {
		_, initErr := execer.ExecContext(context.Background(), "SET TimeZone='"+tz+"'", nil)
		return initErr
	})
	if err != nil {
		t.Fatalf("NewConnector(%s): %v", path, err)
	}
	return sql.OpenDB(connector)
}

// TestAssociateMediaWithMatches_RestartCycle simule le scénario opérationnel
// post-bug :
//  1. Ouvrir shared_social via NewConnector(timezone) — comme le pool prod
//  2. Setup média + match
//  3. Run AssociateMediaWithMatches (ancienne version : ATTACH ; nouvelle :
//     load match en Go puis bulk insert)
//  4. Close la connexion (CHECKPOINT propre — limite du test : ne simule pas
//     un kill brutal SIGKILL)
//  5. Réouvrir via NewConnector(timezone) — comme le pool au reboot serveur
//  6. Ping doit réussir, COUNT(*) media_match_associations doit être correct
//
// Avec l'ancienne implémentation ATTACH, ce cycle pouvait échouer au step 5
// avec "INTERNAL Error: Failure while replaying WAL file". Avec la nouvelle
// implémentation, le WAL ne contient plus d'entrée ATTACH et le reopen est
// garanti propre.
func TestAssociateMediaWithMatches_RestartCycle(t *testing.T) {
	dir := t.TempDir()
	socialPath := filepath.Join(dir, "shared_social.duckdb")
	matchesPath := filepath.Join(dir, "shared_matches.duckdb")
	ctx := context.Background()
	const tz = "Europe/Paris"

	// Setup shared_matches avec un match dans une fenêtre connue.
	matchDB, err := sql.Open("duckdb", matchesPath)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := matchDB.Exec(`CREATE TABLE match_registry (
		match_id       VARCHAR PRIMARY KEY,
		start_time     TIMESTAMP,
		end_time       TIMESTAMP,
		start_time_utc TIMESTAMPTZ,
		end_time_utc   TIMESTAMPTZ
	)`); err != nil {
		t.Fatal(err)
	}
	// Match 2026-04-19 15:00-15:15 UTC (=17:00-17:15 Paris CEST).
	if _, err := matchDB.Exec(
		`INSERT INTO match_registry VALUES (?, ?, ?, ?, ?)`,
		"match-restart-cycle",
		time.Date(2026, 4, 19, 17, 0, 0, 0, time.UTC),
		time.Date(2026, 4, 19, 17, 15, 0, 0, time.UTC),
		time.Date(2026, 4, 19, 15, 0, 0, 0, time.UTC),
		time.Date(2026, 4, 19, 15, 15, 0, 0, time.UTC),
	); err != nil {
		t.Fatal(err)
	}
	matchDB.Close()

	// Phase 1 : ouvrir shared_social via NewConnector(timezone) — comme le pool.
	socialDB := newConnectorWithTimezone(t, socialPath, tz)

	// Setup shared_social : créer media_files + média à 15:10 UTC (dans la fenêtre).
	if err := ensureMediaTables(ctx, socialDB); err != nil {
		t.Fatal(err)
	}
	captureAt := time.Date(2026, 4, 19, 15, 10, 0, 0, time.UTC)
	if _, err := socialDB.Exec(
		`INSERT INTO media_files (player_slug, file_path, file_name, file_hash, kind, capture_start_utc)
		 VALUES ('spartan', '/cap/clip.mp4', 'clip.mp4', 'hash-restart', 'video', ?)`,
		captureAt,
	); err != nil {
		t.Fatal(err)
	}

	// Phase 2 : run AssociateMediaWithMatches (nouvelle impl sans ATTACH).
	nAssocs, err := AssociateMediaWithMatches(ctx, socialDB, matchesPath, 2, tz)
	if err != nil {
		t.Fatalf("AssociateMediaWithMatches: %v", err)
	}
	if nAssocs != 1 {
		t.Errorf("nAssocs = %d, want 1", nAssocs)
	}

	// Phase 3 : close la connexion (CHECKPOINT propre Go).
	if err := socialDB.Close(); err != nil {
		t.Fatalf("first Close: %v", err)
	}

	// Phase 4 : reopen via NewConnector(timezone) — comme le pool au reboot.
	// AVEC l'ancien code (ATTACH), cette étape pouvait planter avec :
	// "INTERNAL Error: Failure while replaying WAL file: Calling
	// DatabaseManager::GetDefaultDatabase with no default database set"
	socialDB2 := newConnectorWithTimezone(t, socialPath, tz)
	defer socialDB2.Close()

	if err := socialDB2.PingContext(ctx); err != nil {
		t.Fatalf("RestartCycle: ping après reopen échoué (régression bug DuckDB #7659 ?) : %v", err)
	}

	// Phase 5 : vérifier que les associations sont toujours là.
	var count int
	if err := socialDB2.QueryRowContext(ctx,
		`SELECT COUNT(*) FROM media_match_associations WHERE match_id = ?`,
		"match-restart-cycle",
	).Scan(&count); err != nil {
		t.Fatalf("query après reopen: %v", err)
	}
	if count != 1 {
		t.Errorf("associations perdues après reopen : count=%d, want 1", count)
	}
}

// TestAssociateMediaWithMatches_SharedMatchesHeldRW est le guard anti-régression
// de l'incident 2026-06-03 (médias uploadés sans association). loadMatchTimeWindows
// doit RÉUTILISER le handle process-wide quand shared_matches est déjà tenu en
// RW dans le même process (cas prod : le pool ouvre shared_matches au boot), au
// lieu de forcer un nouvel open `?access_mode=read_only` — lequel échoue avec
// "Can't open ... with a different configuration" / "file is being used".
//
// Mécanisme garanti par OpenReadForQuery (LookupCachedDB → handle caché).
// Si quelqu'un re-régresse loadMatchTimeWindows vers un sql.Open RO non
// cache-aware, ce test échoue : l'open RO entrera en conflit avec le handle RW
// ci-dessous tenu ouvert pour toute la durée du test.
func TestAssociateMediaWithMatches_SharedMatchesHeldRW(t *testing.T) {
	dir := t.TempDir()
	socialPath := filepath.Join(dir, "shared_social.duckdb")
	matchesPath := filepath.Join(dir, "shared_matches_held_rw.duckdb")
	ctx := context.Background()

	// Setup shared_matches avec un match dans une fenêtre connue, puis le tenir
	// OUVERT EN RW via le pool process-wide (clé "rw:"+matchesPath) pour toute la
	// durée du test — exactement comme le pool serveur en prod.
	matchHandle, err := duckdbpkg.OpenReadWrite(matchesPath)
	if err != nil {
		t.Fatalf("OpenReadWrite(matches): %v", err)
	}
	defer matchHandle.Close()
	matchDB := matchHandle.SQLDb()
	if _, err := matchDB.ExecContext(ctx, `CREATE TABLE match_registry (
		match_id VARCHAR PRIMARY KEY,
		start_time TIMESTAMP, end_time TIMESTAMP,
		start_time_utc TIMESTAMPTZ, end_time_utc TIMESTAMPTZ
	)`); err != nil {
		t.Fatal(err)
	}
	if _, err := matchDB.ExecContext(ctx,
		`INSERT INTO match_registry VALUES (?, ?, ?, ?, ?)`,
		"m-held-rw",
		time.Date(2026, 5, 11, 21, 0, 0, 0, time.UTC),
		time.Date(2026, 5, 11, 21, 15, 0, 0, time.UTC),
		time.Date(2026, 5, 11, 21, 0, 0, 0, time.UTC),
		time.Date(2026, 5, 11, 21, 15, 0, 0, time.UTC),
	); err != nil {
		t.Fatal(err)
	}

	// shared_social : un média dans la fenêtre (21:10 UTC).
	socialDB, err := sql.Open("duckdb", socialPath)
	if err != nil {
		t.Fatal(err)
	}
	defer socialDB.Close()
	if err := ensureMediaTables(ctx, socialDB); err != nil {
		t.Fatal(err)
	}
	if _, err := socialDB.ExecContext(ctx,
		`INSERT INTO media_files (player_slug, file_path, file_name, file_hash, kind, capture_start_utc)
		 VALUES ('spartan', '/cap/held.mp4', 'held.mp4', 'hash-held', 'video', ?)`,
		time.Date(2026, 5, 11, 21, 10, 0, 0, time.UTC),
	); err != nil {
		t.Fatal(err)
	}

	// L'association DOIT réussir bien que shared_matches soit tenu RW in-process.
	n, err := AssociateMediaWithMatches(ctx, socialDB, matchesPath, 2, "")
	if err != nil {
		t.Fatalf("AssociateMediaWithMatches a échoué alors que shared_matches est tenu RW "+
			"(régression handle reuse / incident 2026-06-03) : %v", err)
	}
	if n != 1 {
		t.Errorf("associations = %d, want 1", n)
	}
}

// TestAssociateMediaWithMatches_RestartCycle_NoWALLeft vérifie qu'après un
// Close propre, aucun fichier .wal résiduel ne traîne. C'est une condition
// nécessaire (mais pas suffisante) pour que le reopen suivant ne déclenche
// pas le bug DuckDB #7659. Le WAL est attendu vide ou inexistant après Close.
func TestAssociateMediaWithMatches_RestartCycle_NoWALLeft(t *testing.T) {
	dir := t.TempDir()
	socialPath := filepath.Join(dir, "shared_social.duckdb")
	matchesPath := filepath.Join(dir, "shared_matches.duckdb")
	ctx := context.Background()
	const tz = "Europe/Paris"

	// Setup minimal matches (1 match).
	matchDB, _ := sql.Open("duckdb", matchesPath)
	_, _ = matchDB.Exec(`CREATE TABLE match_registry (
		match_id VARCHAR PRIMARY KEY,
		start_time TIMESTAMP, end_time TIMESTAMP,
		start_time_utc TIMESTAMPTZ, end_time_utc TIMESTAMPTZ
	)`)
	_, _ = matchDB.Exec(
		`INSERT INTO match_registry VALUES (?, ?, ?, ?, ?)`,
		"m1",
		time.Now().UTC(), time.Now().UTC().Add(15*time.Minute),
		time.Now().UTC(), time.Now().UTC().Add(15*time.Minute),
	)
	matchDB.Close()

	// Setup social + run Associate.
	socialDB := newConnectorWithTimezone(t, socialPath, tz)
	_ = ensureMediaTables(ctx, socialDB)
	captureAt := time.Now().UTC().Add(5 * time.Minute)
	_, _ = socialDB.Exec(
		`INSERT INTO media_files (player_slug, file_path, file_name, file_hash, kind, capture_start_utc)
		 VALUES ('p', '/c.mp4', 'c.mp4', 'h', 'video', ?)`,
		captureAt,
	)
	_, _ = AssociateMediaWithMatches(ctx, socialDB, matchesPath, 2, tz)
	_ = socialDB.Close()

	// Vérifier l'état du WAL post-close.
	walPath := socialPath + ".wal"
	info, err := os.Stat(walPath)
	if err != nil {
		if os.IsNotExist(err) {
			return // WAL n'existe pas → parfait, CHECKPOINT a tout fusionné
		}
		t.Fatalf("stat WAL: %v", err)
	}
	// WAL existe : tolérer si vide (CHECKPOINT a vidé le contenu). Échouer si
	// le WAL contient des données (signe d'opérations non-checkpointées qui
	// pourraient causer un replay au reopen).
	if info.Size() > 0 {
		t.Errorf("WAL non-vide après Close (size=%d) — risque de replay au prochain reopen.\n"+
			"Path: %s", info.Size(), walPath)
	}
}
