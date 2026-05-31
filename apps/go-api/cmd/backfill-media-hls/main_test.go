package main

import (
	"database/sql"
	"path/filepath"
	"testing"

	_ "github.com/duckdb/duckdb-go/v2"
)

// TestSelectCandidates vérifie le filtrage : seules les vidéos sans hls_path
// sont candidates ; images et clips déjà transcodés exclus ; filtre slug.
func TestSelectCandidates(t *testing.T) {
	dir := t.TempDir()
	dbPath := filepath.Join(dir, "shared_social.duckdb")

	func() {
		db, err := sql.Open("duckdb", dbPath)
		if err != nil {
			t.Fatal(err)
		}
		defer db.Close()
		if _, err := db.Exec(`CREATE TABLE media_files (id INTEGER, player_slug VARCHAR, file_path VARCHAR, hls_path VARCHAR, kind VARCHAR)`); err != nil {
			t.Fatal(err)
		}
		if _, err := db.Exec(`INSERT INTO media_files VALUES
			(1, 'A', 'A/clip.mkv',  NULL,                       'video'),
			(2, 'A', 'A/done.mkv',  'A/hls/done/master.m3u8',   'video'),
			(3, 'B', 'B/other.mkv', NULL,                       'video'),
			(4, 'A', 'A/img.png',   NULL,                       'image')`); err != nil {
			t.Fatal(err)
		}
	}()

	all, err := selectCandidates(dbPath, "")
	if err != nil {
		t.Fatalf("selectCandidates: %v", err)
	}
	if len(all) != 2 {
		t.Errorf("tous: %d candidats, want 2 (clip.mkv + other.mkv ; done.mkv et img.png exclus)", len(all))
	}

	a, err := selectCandidates(dbPath, "A")
	if err != nil {
		t.Fatal(err)
	}
	if len(a) != 1 || a[0].filePath != "A/clip.mkv" {
		t.Errorf("filtre slug A: got %+v, want [A/clip.mkv]", a)
	}
}
