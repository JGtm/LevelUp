//go:build integration

package persist

import (
	"context"
	"database/sql"
	"testing"

	"levelup/go-api/internal/migration"

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
			created_at TIMESTAMP DEFAULT CAST(now() AT TIME ZONE 'UTC' AS TIMESTAMP),
			updated_at TIMESTAMP DEFAULT CAST(now() AT TIME ZONE 'UTC' AS TIMESTAMP)
		)
	`); err != nil {
		t.Fatalf("create: %v", err)
	}
	if err := migration.EnsurePlayerMatchEnrichmentAppendOnly(db); err != nil {
		t.Fatalf("EnsurePlayerMatchEnrichmentAppendOnly: %v", err)
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
		_ = db.QueryRow(`SELECT dominance_flag FROM player_match_enrichment_latest WHERE match_id = ?`, tc.mid).Scan(&got)
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
	_ = db.QueryRow(`SELECT dominance_flag, performance_score FROM player_match_enrichment_latest WHERE match_id = 'm1'`).Scan(&dom, &perf)
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

	// Append-only #23046 : BatchUpdateMulti écrit plusieurs colonnes d'UN SEUL stage
	// par INSERT partiel (un INSERT = un stage). On teste perf={score, chain} — ce que
	// fait le vrai caller (performance.go). Mélanger des stages (dominance+perf) est
	// désormais rejeté par deriveEnrichmentStage (cf. test dédié ci-dessous).
	rows := []EnrichmentMultiColumnUpdate{
		{MatchID: "m1", Fields: map[string]any{"performance_score": 75.0, "performance_chain": "WWL"}},
		{MatchID: "m2", Fields: map[string]any{"performance_score": 82.5, "performance_chain": "LWW"}},
		{MatchID: "m3", Fields: map[string]any{"performance_score": 91.2, "performance_chain": "WWW"}},
	}
	n, err := p.BatchUpdateMulti(context.Background(), rows)
	if err != nil {
		t.Fatalf("BatchUpdateMulti: %v", err)
	}
	if n != int64(len(rows)) {
		t.Errorf("affected rows=%d, want %d", n, len(rows))
	}

	for _, tc := range []struct {
		mid   string
		perf  float64
		chain string
	}{{"m1", 75.0, "WWL"}, {"m2", 82.5, "LWW"}, {"m3", 91.2, "WWW"}} {
		var perf float64
		var chain string
		_ = db.QueryRow(`SELECT performance_score, performance_chain FROM player_match_enrichment_latest WHERE match_id = ?`, tc.mid).Scan(&perf, &chain)
		if perf != tc.perf || chain != tc.chain {
			t.Errorf("%s: perf=%f chain=%q, want perf=%f chain=%q", tc.mid, perf, chain, tc.perf, tc.chain)
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
	_, err := p.BatchUpdateMulti(context.Background(), rows)
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
	_ = db.QueryRow(`SELECT dominance_flag FROM player_match_enrichment_latest WHERE match_id = 'm1'`).Scan(&dom)
	if dom.Valid {
		t.Errorf("dominance_flag = %+v, want NULL", dom)
	}
}
