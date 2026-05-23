//go:build integration

package persist

import (
	"context"
	"database/sql"
	"testing"

	_ "github.com/duckdb/duckdb-go/v2"
)

func openEnrichmentTestDB(t *testing.T) *sql.DB {
	t.Helper()
	db, err := sql.Open("duckdb", ":memory:")
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	t.Cleanup(func() { _ = db.Close() })
	if _, err := db.Exec(`
		CREATE TABLE player_match_enrichment (
			match_id VARCHAR PRIMARY KEY,
			dominance_flag INTEGER,
			performance_score FLOAT,
			session_id VARCHAR,
			is_with_friends BOOLEAN,
			had_bot_teammate BOOLEAN,
			created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
			updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
		)
	`); err != nil {
		t.Fatalf("create: %v", err)
	}
	for _, mid := range []string{"m1", "m2", "m3"} {
		if _, err := db.Exec(`INSERT INTO player_match_enrichment (match_id) VALUES (?)`, mid); err != nil {
			t.Fatalf("seed %s: %v", mid, err)
		}
	}
	return db
}

func TestPostSyncEnrichmentPersister_BatchUpdateColumn_UpdatesAllRows(t *testing.T) {
	db := openEnrichmentTestDB(t)
	p := NewPostSyncEnrichmentPersister(db)

	err := p.BatchUpdateColumn(context.Background(), EnrichmentColumnUpdate{
		Column: "dominance_flag",
		Rows: []EnrichmentColumnRow{
			{MatchID: "m1", Value: 1},
			{MatchID: "m2", Value: 3},
			{MatchID: "m3", Value: 5},
		},
	})
	if err != nil {
		t.Fatalf("BatchUpdateColumn: %v", err)
	}

	for _, tc := range []struct {
		mid  string
		want int
	}{{"m1", 1}, {"m2", 3}, {"m3", 5}} {
		var got int
		_ = db.QueryRow(`SELECT dominance_flag FROM player_match_enrichment WHERE match_id = ?`, tc.mid).Scan(&got)
		if got != tc.want {
			t.Errorf("%s dominance_flag = %d, want %d", tc.mid, got, tc.want)
		}
	}
}

func TestPostSyncEnrichmentPersister_BatchUpdateColumn_RejectsUnknownColumn(t *testing.T) {
	db := openEnrichmentTestDB(t)
	p := NewPostSyncEnrichmentPersister(db)

	err := p.BatchUpdateColumn(context.Background(), EnrichmentColumnUpdate{
		Column: "evil_column; DROP TABLE x;",
		Rows:   []EnrichmentColumnRow{{MatchID: "m1", Value: 1}},
	})
	if err == nil {
		t.Error("colonne non-whitelistée doit échouer")
	}
}

func TestPostSyncEnrichmentPersister_BatchUpdateColumn_EmptyRows_NoOp(t *testing.T) {
	db := openEnrichmentTestDB(t)
	p := NewPostSyncEnrichmentPersister(db)

	if err := p.BatchUpdateColumn(context.Background(), EnrichmentColumnUpdate{
		Column: "dominance_flag",
		Rows:   nil,
	}); err != nil {
		t.Errorf("empty rows doit être no-op, got %v", err)
	}
}

func TestPostSyncEnrichmentPersister_BatchUpdateColumn_PreservesOtherColumns(t *testing.T) {
	db := openEnrichmentTestDB(t)
	// Pré-seed performance_score sur m1
	if _, err := db.Exec(`UPDATE player_match_enrichment SET performance_score = 75.5 WHERE match_id = 'm1'`); err != nil {
		t.Fatal(err)
	}

	p := NewPostSyncEnrichmentPersister(db)
	if err := p.BatchUpdateColumn(context.Background(), EnrichmentColumnUpdate{
		Column: "dominance_flag",
		Rows:   []EnrichmentColumnRow{{MatchID: "m1", Value: 2}},
	}); err != nil {
		t.Fatal(err)
	}

	var dom int
	var perf sql.NullFloat64
	_ = db.QueryRow(`SELECT dominance_flag, performance_score FROM player_match_enrichment WHERE match_id = 'm1'`).Scan(&dom, &perf)
	if dom != 2 {
		t.Errorf("dominance_flag = %d, want 2", dom)
	}
	if !perf.Valid || perf.Float64 != 75.5 {
		t.Errorf("performance_score = %+v, want 75.5 (autre col préservée)", perf)
	}
}

func TestPostSyncEnrichmentPersister_BatchUpdateMulti_UpdatesMultiCols(t *testing.T) {
	db := openEnrichmentTestDB(t)
	p := NewPostSyncEnrichmentPersister(db)

	rows := []EnrichmentMultiColumnUpdate{
		{MatchID: "m1", Fields: map[string]any{"dominance_flag": 1, "performance_score": 75.0}},
		{MatchID: "m2", Fields: map[string]any{"dominance_flag": 3, "performance_score": 82.5}},
		{MatchID: "m3", Fields: map[string]any{"dominance_flag": 5, "performance_score": 91.2}},
	}
	if err := p.BatchUpdateMulti(context.Background(), rows); err != nil {
		t.Fatalf("BatchUpdateMulti: %v", err)
	}

	for _, tc := range []struct {
		mid  string
		flag int
		perf float64
	}{{"m1", 1, 75.0}, {"m2", 3, 82.5}, {"m3", 5, 91.2}} {
		var flag int
		var perf float64
		_ = db.QueryRow(`SELECT dominance_flag, performance_score FROM player_match_enrichment WHERE match_id = ?`, tc.mid).Scan(&flag, &perf)
		if flag != tc.flag || perf != tc.perf {
			t.Errorf("%s: flag=%d perf=%f, want flag=%d perf=%f", tc.mid, flag, perf, tc.flag, tc.perf)
		}
	}
}

func TestPostSyncEnrichmentPersister_BatchUpdateMulti_RejectsInconsistentRows(t *testing.T) {
	db := openEnrichmentTestDB(t)
	p := NewPostSyncEnrichmentPersister(db)

	rows := []EnrichmentMultiColumnUpdate{
		{MatchID: "m1", Fields: map[string]any{"dominance_flag": 1, "performance_score": 75.0}},
		{MatchID: "m2", Fields: map[string]any{"dominance_flag": 3}}, // ← cols différentes
	}
	err := p.BatchUpdateMulti(context.Background(), rows)
	if err == nil {
		t.Error("rows inhomogènes doit échouer")
	}
}

func TestPostSyncEnrichmentPersister_BatchUpdateColumn_HandlesNilValue(t *testing.T) {
	db := openEnrichmentTestDB(t)
	if _, err := db.Exec(`UPDATE player_match_enrichment SET dominance_flag = 5 WHERE match_id = 'm1'`); err != nil {
		t.Fatal(err)
	}

	p := NewPostSyncEnrichmentPersister(db)
	if err := p.BatchUpdateColumn(context.Background(), EnrichmentColumnUpdate{
		Column: "dominance_flag",
		Rows:   []EnrichmentColumnRow{{MatchID: "m1", Value: nil}},
	}); err != nil {
		t.Fatal(err)
	}

	var dom sql.NullInt64
	_ = db.QueryRow(`SELECT dominance_flag FROM player_match_enrichment WHERE match_id = 'm1'`).Scan(&dom)
	if dom.Valid {
		t.Errorf("dominance_flag = %+v, want NULL", dom)
	}
}
