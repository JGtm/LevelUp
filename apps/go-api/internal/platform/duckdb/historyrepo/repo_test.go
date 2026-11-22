package historyrepo

import (
	"context"
	"database/sql"
	"path/filepath"
	"testing"

	_ "github.com/duckdb/duckdb-go/v2"

	"levelup/go-api/internal/sync/historystate"
)

// newPlayerDB crée sur disque une base joueur avec sa table sync_meta (une seule connexion), la
// referme et rend son chemin : la lecture rouvre le fichier par OpenReadForQuery.
func newPlayerDB(t *testing.T, markComplete bool) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "stats.duckdb")
	db, err := sql.Open("duckdb", path)
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	db.SetMaxOpenConns(1)
	if _, err := db.Exec(`CREATE TABLE sync_meta (key VARCHAR PRIMARY KEY, value VARCHAR, updated_at TIMESTAMP)`); err != nil {
		t.Fatalf("create sync_meta: %v", err)
	}
	if markComplete && !historystate.MarkComplete(context.Background(), db, "JGtm") {
		t.Fatal("la fin d'historique n'a pas été rangée")
	}
	if err := db.Close(); err != nil {
		t.Fatalf("close: %v", err)
	}
	return path
}

func TestHistoryComplete(t *testing.T) {
	cases := []struct {
		name string
		path func(t *testing.T) string
		want bool
	}{
		{"base joueur absente : aucune passe n'a parcouru l'historique", func(t *testing.T) string {
			return filepath.Join(t.TempDir(), "absente.duckdb")
		}, false},
		{"base sans clé de fin : rattrapage en cours", func(t *testing.T) string { return newPlayerDB(t, false) }, false},
		{"clé de fin rangée : complet", func(t *testing.T) string { return newPlayerDB(t, true) }, true},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			path := c.path(t)
			var gotTitle, gotGamertag string
			r := New(func(titleSlug, gamertag string) string {
				gotTitle, gotGamertag = titleSlug, gamertag
				return path
			})
			got, err := r.HistoryComplete(context.Background(), "halo_infinite", "JGtm")
			if err != nil {
				t.Fatalf("HistoryComplete: %v", err)
			}
			if got != c.want {
				t.Errorf("HistoryComplete = %v, attendu %v", got, c.want)
			}
			if gotTitle != "halo_infinite" || gotGamertag != "JGtm" {
				t.Errorf("chemin demandé pour (%q, %q)", gotTitle, gotGamertag)
			}
		})
	}
}
