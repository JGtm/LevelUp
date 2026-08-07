package duckdb

import (
	"context"
	"database/sql"
	"testing"

	_ "github.com/duckdb/duckdb-go/v2"
)

func openNoDataTestDB(t *testing.T) *sql.DB {
	t.Helper()
	db, err := sql.Open("duckdb", ":memory:")
	if err != nil {
		t.Fatalf("open duckdb: %v", err)
	}
	t.Cleanup(func() { db.Close() })
	if _, err := db.Exec(`CREATE TABLE world_player_no_data (
		title_slug VARCHAR NOT NULL, season_id VARCHAR NOT NULL, gamertag VARCHAR NOT NULL,
		marked_at TIMESTAMP DEFAULT CAST(now() AT TIME ZONE 'UTC' AS TIMESTAMP),
		PRIMARY KEY (title_slug, season_id, gamertag))`); err != nil {
		t.Fatalf("create: %v", err)
	}
	return db
}

func TestWorldNoDataPlayers_RoundTripAndIdempotent(t *testing.T) {
	db := openNoDataTestDB(t)
	ctx := context.Background()

	n, err := InsertWorldNoDataPlayers(ctx, db, "halo_infinite", "csrseason12-1",
		[]string{"PrivateGuy", "SecretPlayer", "", "PrivateGuy"}) // "" ignoré ; doublon dédup
	if err != nil {
		t.Fatalf("InsertWorldNoDataPlayers: %v", err)
	}
	if n != 2 {
		t.Errorf("marques = %d, attendu 2 (vide ignoré, doublon dédupliqué)", n)
	}

	set, err := WorldSeasonNoDataGamertags(ctx, db, "halo_infinite", "csrseason12-1")
	if err != nil {
		t.Fatalf("WorldSeasonNoDataGamertags: %v", err)
	}
	if _, ok := set["PrivateGuy"]; !ok {
		t.Error("PrivateGuy devrait être marqué")
	}
	if len(set) != 2 {
		t.Errorf("set = %d, attendu 2", len(set))
	}

	// Idempotence : re-insérer les mêmes → 0 nouvelle marque, pas de doublon (INSERT-only).
	n2, err := InsertWorldNoDataPlayers(ctx, db, "halo_infinite", "csrseason12-1",
		[]string{"PrivateGuy", "SecretPlayer"})
	if err != nil {
		t.Fatalf("2e insert: %v", err)
	}
	if n2 != 0 {
		t.Errorf("2e passage : %d nouvelles marques, attendu 0", n2)
	}

	// Isolation par saison : autre saison → set vide.
	other, _ := WorldSeasonNoDataGamertags(ctx, db, "halo_infinite", "csrseason11-1")
	if len(other) != 0 {
		t.Errorf("autre saison devrait être vide, got %d", len(other))
	}
}
