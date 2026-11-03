//go:build cgo

package transfer

import (
	"archive/zip"
	"context"
	"database/sql"
	"errors"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"levelup/go-api/internal/domain/title"
)

const (
	slug    = "halo_infinite"
	meXuid  = "2533274800000001"
	herXuid = "2533274800000002"
	meTag   = "SpartanB"
)

var sharedDDL = []string{
	`CREATE TABLE match_registry (match_id VARCHAR PRIMARY KEY, map_name VARCHAR NOT NULL, team_0_score INTEGER)`,
	`CREATE TABLE match_participants (match_id VARCHAR, xuid VARCHAR, gamertag VARCHAR, kills INTEGER, PRIMARY KEY (match_id, xuid))`,
	`CREATE TABLE medals_earned (match_id VARCHAR, xuid VARCHAR, medal_id BIGINT, count INTEGER)`,
	`CREATE TABLE highlight_events (match_id VARCHAR, xuid VARCHAR, event_type VARCHAR, details_json VARCHAR)`,
}

var playerDDL = []string{`CREATE TABLE player_match_enrichment (match_id VARCHAR PRIMARY KEY, performance_score DOUBLE)`}

func mustExec(t *testing.T, path string, stmts ...string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	db, err := sql.Open("duckdb", path)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = db.Close() }()
	for _, s := range stmts {
		if _, err := db.Exec(s); err != nil {
			t.Fatalf("%s : %v", s, err)
		}
	}
}

func roots(t *testing.T) *title.PathResolver {
	root := t.TempDir()
	return title.NewPathResolverRoots(title.Roots{Resources: root, Data: filepath.Join(root, "data"), DataSet: true})
}

func count(t *testing.T, path, query string) int {
	t.Helper()
	db, err := sql.Open("duckdb", path+"?access_mode=read_only")
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = db.Close() }()
	var n int
	if err := db.QueryRow(query).Scan(&n); err != nil {
		t.Fatalf("%s : %v", query, err)
	}
	return n
}

// hostedFixture : une instance hébergée avec deux joueurs ; me joue m1 et m2, her joue m2 et m3.
func hostedFixture(t *testing.T) *title.PathResolver {
	pr := roots(t)
	mustExec(t, pr.SharedDBPath(slug), append(append([]string{}, sharedDDL...),
		`INSERT INTO match_registry VALUES ('m1','Aquarius',50),('m2','Streets',48),('m3','Recharge',50)`,
		`INSERT INTO match_participants VALUES ('m1','`+meXuid+`','SpartanB',20),('m2','`+meXuid+`','SpartanB',11),('m2','`+herXuid+`','Her',9),('m3','`+herXuid+`','Her',14)`,
		`INSERT INTO medals_earned VALUES ('m1','`+meXuid+`',1,2),('m2','`+meXuid+`',2,1),('m2','`+herXuid+`',2,3),('m3','`+herXuid+`',5,1)`,
		`INSERT INTO highlight_events VALUES ('m1','`+meXuid+`','kill','{}'),('m3','`+herXuid+`','kill','{}')`)...)
	mustExec(t, pr.PlayerDBPath(slug, meTag), append(append([]string{}, playerDDL...),
		`INSERT INTO player_match_enrichment VALUES ('m1',71.5),('m2',64.0)`)...)
	return pr
}

func exportFor(t *testing.T, pr *title.PathResolver, xuid, tag string) (string, *Manifest) {
	t.Helper()
	dest := filepath.Join(t.TempDir(), "export.zip")
	m, err := Export(context.Background(), ExportInput{Paths: pr, Slug: slug, Xuid: xuid, Gamertag: tag, AppVersion: "9.9.9-test", Dest: dest})
	if err != nil {
		t.Fatalf("Export : %v", err)
	}
	return dest, m
}

// acquire ouvre une base de fichier en écriture (le test tient le rôle du fournisseur de l'application).
func acquire(path string) Acquire {
	return func(context.Context) (*sql.DB, func(), error) {
		db, err := sql.Open("duckdb", path)
		if err != nil {
			return nil, nil, err
		}
		return db, func() { _ = db.Close() }, nil
	}
}

func TestExportThenImport_EndToEnd(t *testing.T) {
	ctx := context.Background()
	hosted := hostedFixture(t)
	zipPath, m := exportFor(t, hosted, meXuid, meTag)

	// Le périmètre : les lignes du joueur et ses matchs, jamais celles de l'autre joueur.
	rows := map[string]int64{}
	for _, tb := range m.Tables {
		rows[tb.Scope+"/"+tb.Name] = tb.Rows
	}
	want := map[string]int64{"shared/match_registry": 2, "shared/match_participants": 2, "shared/medals_earned": 2,
		"shared/highlight_events": 1, "player/player_match_enrichment": 2}
	for k, v := range want {
		if rows[k] != v {
			t.Errorf("%s : %d lignes exportées, attendu %d", k, rows[k], v)
		}
	}

	// Le bureau : bases locales vides, sauf m1 déjà synchronisé.
	local := roots(t)
	mustExec(t, local.SharedDBPath(slug), append(append([]string{}, sharedDDL...),
		`INSERT INTO match_registry VALUES ('m1','Aquarius',50)`,
		`INSERT INTO match_participants VALUES ('m1','`+meXuid+`','SpartanB',20)`)...)
	mustExec(t, local.PlayerDBPath(slug, meTag), playerDDL...)
	in := ImportInput{ZipPath: zipPath, ExpectXuid: meXuid,
		AcquireShared: acquire(local.SharedDBPath(slug)), AcquirePlayer: acquire(local.PlayerDBPath(slug, meTag))}

	res, err := Import(ctx, in)
	if err != nil {
		t.Fatalf("Import : %v", err)
	}
	if res.Inserted["shared/match_registry"] != 1 || res.Inserted["shared/match_participants"] != 1 {
		t.Errorf("m1 est déjà là, seul m2 s'ajoute : %v", res.Inserted)
	}
	if got := count(t, local.SharedDBPath(slug), `SELECT count(*) FROM match_registry`); got != 2 {
		t.Errorf("registre local = %d matchs, attendu 2 (m1 sans doublon, m2 ajouté)", got)
	}
	if got := count(t, local.SharedDBPath(slug), `SELECT count(*) FROM match_registry WHERE match_id = 'm3'`); got != 0 {
		t.Error("m3 est un match de l'autre joueur : il ne doit pas être importé")
	}
	if got := count(t, local.SharedDBPath(slug), `SELECT count(*) FROM match_participants WHERE xuid = '`+herXuid+`'`); got != 0 {
		t.Error("aucune ligne de l'autre joueur ne doit être importée")
	}
	if got := count(t, local.PlayerDBPath(slug, meTag), `SELECT count(*) FROM player_match_enrichment`); got != 2 {
		t.Errorf("enrichissements joueur = %d, attendu 2", got)
	}

	// Idempotence : rejouer n'insère rien.
	again, err := Import(ctx, in)
	if err != nil {
		t.Fatalf("second Import : %v", err)
	}
	for k, v := range again.Inserted {
		if v != 0 {
			t.Errorf("%s : %d lignes au second import, attendu 0", k, v)
		}
	}
}

func TestImport_RefusesOtherPlayersArchive(t *testing.T) {
	zipPath, _ := exportFor(t, hostedFixture(t), meXuid, meTag)
	local := roots(t)
	mustExec(t, local.SharedDBPath(slug), sharedDDL...)
	_, err := Import(context.Background(), ImportInput{ZipPath: zipPath, ExpectXuid: herXuid, AcquireShared: acquire(local.SharedDBPath(slug))})
	if !errors.Is(err, ErrXuidMismatch) {
		t.Fatalf("archive d'un autre joueur : %v, attendu ErrXuidMismatch", err)
	}
	if got := count(t, local.SharedDBPath(slug), `SELECT count(*) FROM match_registry`); got != 0 {
		t.Fatalf("rien ne doit être écrit : %d matchs", got)
	}
}

func rewriteZip(t *testing.T, src string, edit func(name string, data []byte) (string, []byte)) string {
	t.Helper()
	zr, err := zip.OpenReader(src)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = zr.Close() }()
	dst := filepath.Join(t.TempDir(), "edited.zip")
	f, err := os.Create(dst)
	if err != nil {
		t.Fatal(err)
	}
	zw := zip.NewWriter(f)
	for _, e := range zr.File {
		rc, _ := e.Open()
		buf := make([]byte, 0, e.UncompressedSize64)
		tmp := make([]byte, 4096)
		for {
			n, err := rc.Read(tmp)
			buf = append(buf, tmp[:n]...)
			if err != nil {
				break
			}
		}
		_ = rc.Close()
		name, data := edit(e.Name, buf)
		w, _ := zw.Create(name)
		_, _ = w.Write(data)
	}
	_ = zw.Close()
	_ = f.Close()
	return dst
}

func TestImport_RefusesTamperedAndTraversalArchives(t *testing.T) {
	zipPath, _ := exportFor(t, hostedFixture(t), meXuid, meTag)
	local := roots(t)
	mustExec(t, local.SharedDBPath(slug), sharedDDL...)
	in := func(p string) ImportInput {
		return ImportInput{ZipPath: p, ExpectXuid: meXuid, AcquireShared: acquire(local.SharedDBPath(slug))}
	}
	tampered := rewriteZip(t, zipPath, func(name string, data []byte) (string, []byte) {
		if name == "shared/match_registry.parquet" {
			return name, append([]byte("x"), data...)
		}
		return name, data
	})
	if _, err := Import(context.Background(), in(tampered)); !errors.Is(err, ErrBadArchive) {
		t.Errorf("archive altérée : %v, attendu ErrBadArchive", err)
	}
	traversal := rewriteZip(t, zipPath, func(name string, data []byte) (string, []byte) {
		if name == ManifestFile {
			return name, []byte(strings.ReplaceAll(string(data), `"shared/match_registry.parquet"`, `"../../evil.parquet"`))
		}
		return name, data
	})
	if _, err := Import(context.Background(), in(traversal)); !errors.Is(err, ErrBadArchive) {
		t.Errorf("entrée hors périmètre : %v, attendu ErrBadArchive", err)
	}
	if _, err := Import(context.Background(), ImportInput{ZipPath: zipPath, ExpectXuid: meXuid, MaxUncompressed: 10,
		AcquireShared: acquire(local.SharedDBPath(slug))}); !errors.Is(err, ErrTooLarge) {
		t.Errorf("taille bornée : %v, attendu ErrTooLarge", err)
	}
}

func TestFetchThenImport_ThroughHTTP(t *testing.T) {
	hosted := hostedFixture(t)
	zipPath, _ := exportFor(t, hosted, meXuid, meTag)
	store := NewTokenStore()
	token, _, err := store.Issue(Grant{Xuid: meXuid, Gamertag: meTag}, time.Now())
	if err != nil {
		t.Fatal(err)
	}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != ExportPath {
			http.NotFound(w, r)
			return
		}
		raw := strings.TrimPrefix(r.Header.Get("Authorization"), "Bearer ")
		if _, err := store.Redeem(raw, time.Now()); err != nil {
			http.Error(w, "refus", http.StatusUnauthorized)
			return
		}
		http.ServeFile(w, r, zipPath)
	}))
	defer srv.Close()

	local := roots(t)
	mustExec(t, local.SharedDBPath(slug), sharedDDL...)
	mustExec(t, local.PlayerDBPath(slug, meTag), playerDDL...)
	runner := NewRunner()
	input := RunInput{
		Fetch: FetchInput{BaseURL: srv.URL, Token: token, Client: srv.Client()},
		Import: ImportInput{ExpectXuid: meXuid,
			AcquireShared: acquire(local.SharedDBPath(slug)), AcquirePlayer: acquire(local.PlayerDBPath(slug, meTag))},
		WorkDir: t.TempDir(),
	}
	if err := runner.Start(context.Background(), nil, input); err != nil {
		t.Fatal(err)
	}
	waitFor(t, runner, PhaseDone)
	if got := count(t, local.SharedDBPath(slug), `SELECT count(*) FROM match_registry`); got != 2 {
		t.Errorf("registre local = %d, attendu 2", got)
	}

	// Le jeton est à usage unique : un second téléchargement avec lui est refusé.
	if err := runner.Start(context.Background(), nil, input); err != nil {
		t.Fatal(err)
	}
	st := waitFor(t, runner, PhaseFailed)
	if st.ErrorCode != CodeTokenInvalid {
		t.Errorf("jeton rejoué : code %q, attendu %q", st.ErrorCode, CodeTokenInvalid)
	}
}

func waitFor(t *testing.T, r *Runner, want Phase) Status {
	t.Helper()
	deadline := time.Now().Add(20 * time.Second)
	for time.Now().Before(deadline) {
		if st := r.Status(); st.Phase == want {
			return st
		} else if st.Phase == PhaseFailed || st.Phase == PhaseDone {
			t.Fatalf("phase %s (code %q), attendu %s", st.Phase, st.ErrorCode, want)
		}
		time.Sleep(20 * time.Millisecond)
	}
	t.Fatalf("délai dépassé en attendant %s (phase %s)", want, r.Status().Phase)
	return Status{}
}
