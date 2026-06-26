//go:build integration

package duckdb

import (
	"context"
	"testing"
)

// seedCrossTitleShared crée un mini-catalogue "autre titre" : match_participants
// avec xuid global. Le joueur courant = xuidMe.
//   - x1 : 4 matchs communs avec me (>= seuil)
//   - x2 : 2 matchs communs (< seuil par défaut 3)
//   - bid(bot) : présent mais exclu par le filtre NOT LIKE 'bid(%'
func seedCrossTitleShared(t *testing.T, db *DB) {
	t.Helper()
	ctx := context.Background()
	ddl := `CREATE TABLE match_participants (match_id VARCHAR, xuid VARCHAR)`
	if _, err := db.Exec(ctx, ddl); err != nil {
		t.Fatalf("seedCrossTitleShared DDL: %v", err)
	}
	ins := `INSERT INTO match_participants VALUES
		('a1','xuidMe'), ('a1','x1'), ('a1','x2'),
		('a2','xuidMe'), ('a2','x1'), ('a2','x2'),
		('a3','xuidMe'), ('a3','x1'),
		('a4','xuidMe'), ('a4','x1'), ('a4','bid(12345)'),
		('a5','x1'), ('a5','x2')` // a5 sans me → ne compte pas
	if _, err := db.Exec(ctx, ins); err != nil {
		t.Fatalf("seedCrossTitleShared INSERT: %v", err)
	}
}

func TestCountCrossTitleCooccurrences(t *testing.T) {
	db := openMemDB(t)
	seedCrossTitleShared(t, db)

	counts, err := CountCrossTitleCooccurrences(
		context.Background(),
		db.SQLDb(),
		"xuidMe",
		[]string{"x1", "x2", "bid(12345)"},
		3,
	)
	if err != nil {
		t.Fatalf("CountCrossTitleCooccurrences: %v", err)
	}
	// x1 : matchs communs a1,a2,a3,a4 = 4 (>= 3) → présent
	if got := counts["x1"]; got != 4 {
		t.Fatalf("x1 count=%d want 4", got)
	}
	// x2 : matchs communs a1,a2 = 2 (< 3) → filtré par HAVING
	if _, ok := counts["x2"]; ok {
		t.Fatalf("x2 should be filtered by threshold, got %d", counts["x2"])
	}
	// bot : exclu par NOT LIKE 'bid(%'
	if _, ok := counts["bid(12345)"]; ok {
		t.Fatal("bot xuid should be excluded")
	}
}

func TestCountCrossTitleCooccurrences_EmptyInputs(t *testing.T) {
	db := openMemDB(t)
	seedCrossTitleShared(t, db)

	// oppXUIDs vide → map vide, pas d'erreur (pas de requête).
	counts, err := CountCrossTitleCooccurrences(context.Background(), db.SQLDb(), "xuidMe", nil, 3)
	if err != nil {
		t.Fatalf("empty inputs: %v", err)
	}
	if len(counts) != 0 {
		t.Fatalf("expected empty map, got %v", counts)
	}

	// db nil → map vide, pas de panique.
	counts, err = CountCrossTitleCooccurrences(context.Background(), nil, "xuidMe", []string{"x1"}, 3)
	if err != nil || len(counts) != 0 {
		t.Fatalf("nil db should yield empty map no err, got %v %v", counts, err)
	}
}
