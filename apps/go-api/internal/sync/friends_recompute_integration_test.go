//go:build integration

// Package sync — friends_recompute_integration_test.go : tests integration
// pour updateIsWithFriendsBatch avec DuckDB :memory:.
//
// Couvre la régression "Solo qui revient" : les lignes player_match_enrichment
// inserées par le sync avant l'ajout de DEFAULT FALSE sont à NULL. La query
// `WHERE is_with_friends = FALSE` historique skippait NULL en logique 3-valeurs
// SQL, donc le badge "Solo" persistait. Le fix passe à `COALESCE(...) = FALSE`
// qui couvre les deux cas.
package sync

import (
	"context"
	"database/sql"
	"testing"

	_ "github.com/duckdb/duckdb-go/v2"
)

func openPlayerForFriendsRecompute(t *testing.T) *sql.DB {
	t.Helper()
	db, err := sql.Open("duckdb", ":memory:")
	if err != nil {
		t.Fatalf("sql.Open: %v", err)
	}
	db.SetMaxOpenConns(1)
	t.Cleanup(func() { db.Close() })

	if _, err := db.Exec(`
		CREATE TABLE player_match_enrichment (
			match_id        VARCHAR PRIMARY KEY,
			is_with_friends BOOLEAN,
			updated_at      TIMESTAMP DEFAULT CURRENT_TIMESTAMP
		)
	`); err != nil {
		t.Fatalf("create player_match_enrichment: %v", err)
	}
	return db
}

// TestUpdateIsWithFriendsBatch_PromotesNullAndFalse vérifie que le UPDATE
// promeut TANT les lignes is_with_friends=FALSE QUE les lignes
// is_with_friends=NULL. Régression : avant le fix, le filtre `= FALSE`
// skippait NULL → badge Solo permanent sur les matchs récents.
func TestUpdateIsWithFriendsBatch_PromotesNullAndFalse(t *testing.T) {
	db := openPlayerForFriendsRecompute(t)

	if _, err := db.Exec(`
		INSERT INTO player_match_enrichment (match_id, is_with_friends) VALUES
		    ('m_null',  NULL),
		    ('m_false', FALSE),
		    ('m_true',  TRUE),
		    ('m_other', NULL)
	`); err != nil {
		t.Fatalf("seed: %v", err)
	}

	affected, err := updateIsWithFriendsBatch(context.Background(), db, []string{"m_null", "m_false", "m_true"})
	if err != nil {
		t.Fatalf("updateIsWithFriendsBatch: %v", err)
	}
	// Doit toucher m_null et m_false (2). m_true est déjà TRUE → exclu par la garde
	// `COALESCE(is_with_friends, FALSE) = FALSE`. m_other n'est pas dans la batch.
	if affected != 2 {
		t.Errorf("expected 2 rows promoted (NULL + FALSE), got %d", affected)
	}

	rows, err := db.Query(`
		SELECT match_id, is_with_friends
		FROM player_match_enrichment
		ORDER BY match_id
	`)
	if err != nil {
		t.Fatalf("select: %v", err)
	}
	defer rows.Close()

	got := map[string]sql.NullBool{}
	for rows.Next() {
		var mid string
		var v sql.NullBool
		if err := rows.Scan(&mid, &v); err != nil {
			t.Fatalf("scan: %v", err)
		}
		got[mid] = v
	}

	cases := map[string]sql.NullBool{
		"m_null":  {Bool: true, Valid: true},   // NULL → TRUE (le fix)
		"m_false": {Bool: true, Valid: true},   // FALSE → TRUE (comportement existant)
		"m_true":  {Bool: true, Valid: true},   // TRUE → TRUE (garde idempotente)
		"m_other": {Bool: false, Valid: false}, // NULL → NULL (pas dans batch)
	}
	for mid, want := range cases {
		g := got[mid]
		if g.Valid != want.Valid || g.Bool != want.Bool {
			t.Errorf("match=%s : got is_with_friends=%v(valid=%v), want %v(valid=%v)",
				mid, g.Bool, g.Valid, want.Bool, want.Valid)
		}
	}
}

// TestUpdateIsWithFriendsBatch_Idempotent vérifie que rejouer le UPDATE sur un
// set déjà promu ne touche aucune ligne (garde idempotente via COALESCE).
func TestUpdateIsWithFriendsBatch_Idempotent(t *testing.T) {
	db := openPlayerForFriendsRecompute(t)

	if _, err := db.Exec(`
		INSERT INTO player_match_enrichment (match_id, is_with_friends) VALUES ('m1', NULL)
	`); err != nil {
		t.Fatalf("seed: %v", err)
	}

	first, err := updateIsWithFriendsBatch(context.Background(), db, []string{"m1"})
	if err != nil {
		t.Fatalf("first call: %v", err)
	}
	if first != 1 {
		t.Errorf("expected 1 promoted on first call, got %d", first)
	}

	second, err := updateIsWithFriendsBatch(context.Background(), db, []string{"m1"})
	if err != nil {
		t.Fatalf("second call: %v", err)
	}
	if second != 0 {
		t.Errorf("expected 0 promoted on second call (idempotent), got %d", second)
	}
}
