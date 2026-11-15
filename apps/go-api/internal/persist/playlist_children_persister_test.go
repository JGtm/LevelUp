//go:build integration

// Package persist — playlist_children_persister_test.go : les écritures de l'expansion des playlists sur le
// schéma metadata réel (migrations), DuckDB en mémoire.
//
//	CGO_ENABLED=1 go test -tags integration ./internal/persist/ -run PlaylistChildren
package persist

import (
	"context"
	"database/sql"
	"testing"

	_ "github.com/duckdb/duckdb-go/v2"

	halomigrations "levelup/go-api/internal/games/halo_infinite/migrations"
	"levelup/go-api/internal/migration"
)

func openPlaylistChildrenTestDB(t *testing.T) *sql.DB {
	t.Helper()
	db, err := sql.Open("duckdb", ":memory:")
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	t.Cleanup(func() { _ = db.Close() })
	migration.SetTitleStepsProvider(halomigrations.StepsFor)
	if err := migration.RunForDB(db, migration.TargetMetadata); err != nil {
		t.Fatalf("RunForDB(Metadata): %v", err)
	}
	return db
}

// TestPlaylistChildrenPersister_PoidsInsertPuisUpdate valide la table de poids et l'upsert SELECT-then-write
// (insert puis update, sans doublon).
func TestPlaylistChildrenPersister_PoidsInsertPuisUpdate(t *testing.T) {
	db := openPlaylistChildrenTestDB(t)
	ctx := context.Background()
	p := NewPlaylistChildrenPersister(db)

	if err := p.EnsureSchema(ctx); err != nil {
		t.Fatalf("ensure table: %v", err)
	}
	// Idempotent : un 2e appel ne doit pas échouer.
	if err := p.EnsureSchema(ctx); err != nil {
		t.Fatalf("ensure table (2e): %v", err)
	}

	if err := p.RecordWeight(ctx, "halo_infinite", "pl-1", "pair-1", 4.17); err != nil {
		t.Fatalf("RecordWeight (insert): %v", err)
	}
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
	if err := p.RecordWeight(ctx, "halo_infinite", "pl-1", "pair-1", 2.0); err != nil {
		t.Fatalf("RecordWeight (update): %v", err)
	}
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

// TestPlaylistChildrenPersister_EnfilageSansDoublon : la file de récupération (sans clé primaire) ne reçoit
// un couple qu'une fois par (titre, type, identifiant), et garde la version du premier enfilage ; un autre
// titre enfile le même identifiant sans collision.
func TestPlaylistChildrenPersister_EnfilageSansDoublon(t *testing.T) {
	db := openPlaylistChildrenTestDB(t)
	ctx := context.Background()
	p := NewPlaylistChildrenPersister(db)

	for _, v := range []string{"v1", "v2"} { // le doublon est sauté, y compris à version différente
		if err := p.EnqueuePair(ctx, "halo_infinite", "pair-1", v); err != nil {
			t.Fatalf("EnqueuePair: %v", err)
		}
	}
	if err := p.EnqueuePair(ctx, "halo_5", "pair-1", "v9"); err != nil {
		t.Fatalf("EnqueuePair (autre titre): %v", err)
	}

	rows, err := db.QueryContext(ctx,
		`SELECT title_slug, asset_type, asset_id, version_id FROM catalog_fetch_queue ORDER BY title_slug`)
	if err != nil {
		t.Fatalf("select queue: %v", err)
	}
	defer rows.Close()
	var got []string
	for rows.Next() {
		var slug, typ, id, ver string
		if err := rows.Scan(&slug, &typ, &id, &ver); err != nil {
			t.Fatalf("scan: %v", err)
		}
		got = append(got, slug+"|"+typ+"|"+id+"|"+ver)
	}
	want := []string{"halo_5|pair|pair-1|v9", "halo_infinite|pair|pair-1|v1"}
	if len(got) != len(want) || got[0] != want[0] || got[1] != want[1] {
		t.Fatalf("file = %v, want %v", got, want)
	}
}
