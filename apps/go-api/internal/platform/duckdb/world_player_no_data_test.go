package duckdb

import (
	"context"
	"database/sql"
	"fmt"
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

// markWorldNoDataPlayers pose des marques « privé / sans données » dans
// world_player_no_data (INSERT-only, idempotent via WHERE NOT EXISTS) : amorce des
// tests de lecture, la table n'ayant plus d'écrivain de production. Retourne le
// nombre de nouvelles marques.
func markWorldNoDataPlayers(ctx context.Context, db *sql.DB, titleSlug, season string, gamertags []string) (int, error) {
	n := 0
	for _, gt := range gamertags {
		if gt == "" {
			continue
		}
		res, err := db.ExecContext(ctx,
			`INSERT INTO world_player_no_data (title_slug, season_id, gamertag)
			 SELECT ?, ?, ?
			 WHERE NOT EXISTS (
				SELECT 1 FROM world_player_no_data
				WHERE title_slug = ? AND season_id = ? AND gamertag = ?
			 )`,
			titleSlug, season, gt, titleSlug, season, gt)
		if err != nil {
			return n, fmt.Errorf("markWorldNoDataPlayers(%s,%s): %w", season, gt, err)
		}
		if a, _ := res.RowsAffected(); a > 0 {
			n++
		}
	}
	return n, nil
}

func TestWorldNoDataPlayers_RoundTripAndIdempotent(t *testing.T) {
	db := openNoDataTestDB(t)
	ctx := context.Background()

	n, err := markWorldNoDataPlayers(ctx, db, "halo_infinite", "csrseason12-1",
		[]string{"PrivateGuy", "SecretPlayer", "", "PrivateGuy"}) // "" ignoré ; doublon dédup
	if err != nil {
		t.Fatalf("markWorldNoDataPlayers: %v", err)
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
	n2, err := markWorldNoDataPlayers(ctx, db, "halo_infinite", "csrseason12-1",
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
