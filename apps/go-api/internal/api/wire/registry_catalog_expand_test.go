//go:build cgo

package wire

import (
	"context"
	"database/sql"
	"testing"

	_ "github.com/duckdb/duckdb-go/v2"
)

// TestPlaylistWeights_UpsertInsertThenUpdate valide la table de poids + l'upsert
// SELECT-then-write (insert puis update, sans doublon), sur DuckDB :memory:.
func TestPlaylistWeights_UpsertInsertThenUpdate(t *testing.T) {
	db, err := sql.Open("duckdb", ":memory:")
	if err != nil {
		t.Fatalf("open duckdb: %v", err)
	}
	defer db.Close()
	ctx := context.Background()

	if err := ensurePlaylistWeightsTable(ctx, db); err != nil {
		t.Fatalf("ensure table: %v", err)
	}
	// Idempotent : un 2e appel ne doit pas échouer.
	if err := ensurePlaylistWeightsTable(ctx, db); err != nil {
		t.Fatalf("ensure table (2e): %v", err)
	}

	upsertPlaylistWeight(ctx, db, "halo_infinite", "pl-1", "pair-1", 4.17)

	weightOf := func() float64 {
		var w float64
		if err := db.QueryRowContext(ctx,
			`SELECT weight FROM playlist_map_mode_weights WHERE title_slug='halo_infinite' AND playlist_asset_id='pl-1' AND pair_asset_id='pair-1'`,
		).Scan(&w); err != nil {
			t.Fatalf("select weight: %v", err)
		}
		return w
	}
	if got := weightOf(); got != 4.17 {
		t.Errorf("poids après insert = %v, want 4.17", got)
	}

	// Update du même couple → écrase, pas de doublon.
	upsertPlaylistWeight(ctx, db, "halo_infinite", "pl-1", "pair-1", 2.0)
	if got := weightOf(); got != 2.0 {
		t.Errorf("poids après update = %v, want 2.0", got)
	}

	var n int
	if err := db.QueryRowContext(ctx, `SELECT COUNT(*) FROM playlist_map_mode_weights`).Scan(&n); err != nil {
		t.Fatalf("count: %v", err)
	}
	if n != 1 {
		t.Errorf("lignes = %d, want 1 (upsert idempotent, pas de doublon)", n)
	}
}
