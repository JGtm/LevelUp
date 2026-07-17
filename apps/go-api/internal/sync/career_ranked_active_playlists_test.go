//go:build integration

package sync

// career_ranked_active_playlists_test.go — LB1 (revue 2026-07). Verrouille la partition
// PAR playlist de activeRankedPlaylists : deux playlists scrapées à des fetched_at
// DISTINCTS (batch réel, chaque playlist estampillée à son propre instant) doivent
// TOUTES être retournées. L'ancien MAX(fetched_at) GLOBAL n'en rendait qu'une (la
// dernière scrapée du batch) → augment CSR limité à 1 playlist sur ~7.

import (
	"context"
	"database/sql"
	"testing"
	"time"

	_ "github.com/duckdb/duckdb-go/v2"

	"levelup/go-api/internal/platform/duckdb/sharedprovider"
)

func TestActiveRankedPlaylists_PartitionsPerPlaylist(t *testing.T) {
	db, err := sql.Open("duckdb", ":memory:")
	if err != nil {
		t.Fatalf("open duckdb: %v", err)
	}
	t.Cleanup(func() { _ = db.Close() })

	// Schéma minimal + vue partitionnée (identique à la migration shared active).
	if _, err := db.Exec(`
		CREATE TABLE world_csr_leaderboard_snapshots (
			title_slug VARCHAR,
			season_id VARCHAR,
			playlist_id VARCHAR,
			fetched_at TIMESTAMP
		);
		CREATE VIEW world_csr_leaderboard_latest AS
			SELECT s.* FROM world_csr_leaderboard_snapshots s
			WHERE s.fetched_at = (
				SELECT max(s2.fetched_at) FROM world_csr_leaderboard_snapshots s2
				WHERE s2.title_slug = s.title_slug
				  AND s2.season_id = s.season_id
				  AND s2.playlist_id = s.playlist_id
			);
	`); err != nil {
		t.Fatalf("setup schéma: %v", err)
	}

	t0 := time.Date(2026, 7, 15, 10, 0, 0, 0, time.UTC)
	rowsIn := []struct {
		playlist string
		at       time.Time
	}{
		{"playlist-A", t0},
		{"playlist-B", t0.Add(37 * time.Second)}, // dernière scrapée du batch
	}
	for _, r := range rowsIn {
		if _, err := db.Exec(
			`INSERT INTO world_csr_leaderboard_snapshots VALUES ('halo_infinite','csrseason13-2',?,?)`,
			r.playlist, r.at,
		); err != nil {
			t.Fatalf("insert %s: %v", r.playlist, err)
		}
	}

	e := &SyncEngine{titleSlug: "halo_infinite", sharedProvider: sharedprovider.FromInMemoryDB(db, ":memory:")}
	got := e.activeRankedPlaylists(context.Background())

	ids := map[string]bool{}
	for _, p := range got {
		ids[p.AssetID] = true
	}
	if !ids["playlist-A"] || !ids["playlist-B"] {
		t.Errorf("les 2 playlists à fetched_at distincts doivent TOUTES être retournées, got %v", got)
	}
	if len(got) != 2 {
		t.Errorf("attendu exactement 2 playlists (pas de fallback statique), got %d : %v", len(got), got)
	}
}
