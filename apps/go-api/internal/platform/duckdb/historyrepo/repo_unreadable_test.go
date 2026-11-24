package historyrepo

// repo_unreadable_test.go : une clé de fin illisible rend l'erreur (état inconnu à l'écran, joueur
// compté en rattrapage par l'ordonnanceur), jamais « complet » (D-25).

import (
	"context"
	"database/sql"
	"os"
	"path/filepath"
	"testing"

	_ "github.com/duckdb/duckdb-go/v2"

	"levelup/go-api/internal/platform/duckdb"
)

// newUnreadablePlayerDB crée une base joueur (chemin players/<gamertag>/stats.duckdb, ouvert comme
// l'écrivain du processus) sans table sync_meta : la lecture de la clé de fin échoue.
func newUnreadablePlayerDB(t *testing.T) string {
	t.Helper()
	duckdb.CloseAll()
	t.Cleanup(duckdb.CloseAll)
	path := filepath.Join(t.TempDir(), "players", "JGtm", "stats.duckdb")
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	db, err := sql.Open("duckdb", path)
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	if _, err := db.Exec(`CREATE TABLE autre (x INTEGER)`); err != nil {
		t.Fatalf("create: %v", err)
	}
	if err := db.Close(); err != nil {
		t.Fatalf("close: %v", err)
	}
	return path
}

func TestHistoryComplete_CleIllisible_Erreur(t *testing.T) {
	path := newUnreadablePlayerDB(t)
	r := New(func(string, string) string { return path }, nil)

	complete, err := r.HistoryComplete(context.Background(), "halo_infinite", "JGtm")
	if err == nil || complete {
		t.Errorf("HistoryComplete = (%v, %v), attendu (faux, erreur)", complete, err)
	}
	if _, err := r.ReadHistoryProgress(context.Background(), "halo_infinite", "JGtm", ""); err == nil {
		t.Error("ReadHistoryProgress : attendu l'erreur (l'écran affiche « inconnu »)")
	}
	if left := duckdb.DumpCachedLeaks(); len(left) != 0 {
		t.Errorf("handles encore en cache après la lecture : %v (la référence prise doit être rendue)", left)
	}
}
