//go:build integration

// shared_commendations_test.go — garde-fou de la table match_commendations
// (AXE B prod-gate : commendations NATIVES Halo 5 par match). Vérifie :
//   - la table est créée par la migration title-owned shared_create_match_commendations ;
//   - la PK naturelle (match_id, xuid, commendation_id) est posée ;
//   - INSERT OR IGNORE est ART-safe : un doublon est ignoré SANS muter le count
//     existant ni DELETE/UPDATE (parité medals_earned, anti-régression #23046).
package migrations

import (
	"database/sql"
	"testing"

	_ "github.com/duckdb/duckdb-go/v2"
)

func applyCommendationsMigration(t *testing.T, db *sql.DB) {
	t.Helper()
	if err := applyMigrationInIsolation(t, db, "shared_create_match_commendations"); err != nil {
		t.Fatalf("apply migration match_commendations: %v", err)
	}
}

func TestSharedCreateMatchCommendations_TableAndPK(t *testing.T) {
	db := openEngMemDB(t)
	applyCommendationsMigration(t, db)

	// Table présente.
	var tcnt int
	if err := db.QueryRow(`
		SELECT COUNT(*) FROM information_schema.tables
		WHERE table_name = 'match_commendations'
	`).Scan(&tcnt); err != nil {
		t.Fatalf("query table: %v", err)
	}
	if tcnt != 1 {
		t.Fatalf("table match_commendations absente (cnt=%d)", tcnt)
	}

	// Colonnes attendues.
	wantCols := map[string]bool{
		"match_id": false, "xuid": false, "commendation_id": false,
		"count": false, "created_at": false,
	}
	rows, err := db.Query(`
		SELECT column_name FROM information_schema.columns
		WHERE table_name = 'match_commendations'
	`)
	if err != nil {
		t.Fatalf("query columns: %v", err)
	}
	defer rows.Close()
	for rows.Next() {
		var c string
		if err := rows.Scan(&c); err != nil {
			t.Fatalf("scan col: %v", err)
		}
		if _, ok := wantCols[c]; ok {
			wantCols[c] = true
		}
	}
	for c, seen := range wantCols {
		if !seen {
			t.Errorf("colonne manquante: %s", c)
		}
	}
}

// TestMatchCommendations_InsertOrIgnore_ARTSafe : un INSERT OR IGNORE sur la même
// clé naturelle (match_id, xuid, commendation_id) est NO-OP (count préservé), sans
// UPDATE de colonne indexée ni DELETE — exactement le pattern medals_earned.
func TestMatchCommendations_InsertOrIgnore_ARTSafe(t *testing.T) {
	db := openEngMemDB(t)
	applyCommendationsMigration(t, db)

	ins := func(matchID, xuid, cid string, count int) error {
		_, err := db.Exec(`
			INSERT OR IGNORE INTO match_commendations (match_id, xuid, commendation_id, count, created_at)
			VALUES (?, ?, ?, ?, CURRENT_TIMESTAMP)`,
			matchID, xuid, cid, count)
		return err
	}

	if err := ins("m1", "xA", "uuid-1", 3); err != nil {
		t.Fatalf("insert 1: %v", err)
	}
	// Re-insert même clé avec un count DIFFÉRENT → ignoré (le 1er count gagne).
	if err := ins("m1", "xA", "uuid-1", 99); err != nil {
		t.Fatalf("insert dup: %v", err)
	}
	// Clés distinctes (autre commendation / autre joueur) → rows additionnelles.
	if err := ins("m1", "xA", "uuid-2", 1); err != nil {
		t.Fatalf("insert distinct cid: %v", err)
	}
	if err := ins("m1", "xB", "uuid-1", 5); err != nil {
		t.Fatalf("insert distinct xuid: %v", err)
	}

	var total int
	if err := db.QueryRow(`SELECT COUNT(*) FROM match_commendations`).Scan(&total); err != nil {
		t.Fatalf("count: %v", err)
	}
	if total != 3 {
		t.Errorf("rows = %d, want 3 (dup ignoré)", total)
	}

	var firstCount int
	if err := db.QueryRow(`
		SELECT count FROM match_commendations
		WHERE match_id = 'm1' AND xuid = 'xA' AND commendation_id = 'uuid-1'
	`).Scan(&firstCount); err != nil {
		t.Fatalf("read count: %v", err)
	}
	if firstCount != 3 {
		t.Errorf("count muté par le dup = %d, want 3 (INSERT OR IGNORE NE met PAS à jour)", firstCount)
	}
}
