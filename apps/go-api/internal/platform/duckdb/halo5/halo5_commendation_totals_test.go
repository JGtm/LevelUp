//go:build integration

// halo5_commendation_totals_test.go — test in-memory (DuckDB :memory:) de la lecture
// des totaux à vie des commendations (dernier progress par commendation). NE TOUCHE
// JAMAIS la vraie DB h5.
//
// Lancer : go test -tags=integration ./internal/platform/duckdb/ -run Halo5CommendationTotals

package halo5

import (
	"context"
	"database/sql"
	"testing"

	_ "github.com/duckdb/duckdb-go/v2"
)

func seedCommendationTotals(t *testing.T, db *sql.DB) {
	t.Helper()
	ctx := context.Background()
	for _, q := range []string{
		`CREATE TABLE match_registry (match_id VARCHAR PRIMARY KEY, start_time TIMESTAMP, start_time_utc TIMESTAMPTZ)`,
		`CREATE TABLE match_commendations (match_id VARCHAR, xuid VARCHAR, commendation_id VARCHAR,
			count INTEGER, progress INTEGER, created_at TIMESTAMP,
			PRIMARY KEY (match_id, xuid, commendation_id))`,
	} {
		if _, err := db.ExecContext(ctx, q); err != nil {
			t.Fatalf("ddl: %v", err)
		}
	}
	for _, r := range []struct{ id, t string }{
		{"m1", "2019-01-01 10:00:00+00"},
		{"m2", "2019-02-01 10:00:00+00"},
		{"m3", "2019-03-01 10:00:00+00"},
	} {
		if _, err := db.ExecContext(ctx,
			`INSERT INTO match_registry (match_id, start_time_utc) VALUES (?, ?::TIMESTAMPTZ)`, r.id, r.t); err != nil {
			t.Fatalf("reg %s: %v", r.id, err)
		}
	}
	// uuid-kills : m1=100 (ancien), m3=150 (récent) → latest 150.
	// uuid-asst  : m2=30 (seul). uuid-null : progress NULL → ignoré. xB : autre joueur.
	for _, c := range []struct {
		mid, xuid, cid string
		prog           any
	}{
		{"m1", "xA", "uuid-kills", 100},
		{"m3", "xA", "uuid-kills", 150},
		{"m2", "xA", "uuid-asst", 30},
		{"m1", "xA", "uuid-null", nil},
		{"m3", "xB", "uuid-kills", 999},
	} {
		if _, err := db.ExecContext(ctx,
			`INSERT INTO match_commendations (match_id, xuid, commendation_id, count, progress, created_at)
			 VALUES (?,?,?,?,?,now())`, c.mid, c.xuid, c.cid, 1, c.prog); err != nil {
			t.Fatalf("comm %s/%s: %v", c.mid, c.cid, err)
		}
	}
}

func TestHalo5CommendationTotals_LatestProgressPerCommendation(t *testing.T) {
	mem := openMemSQL(t)
	seedCommendationTotals(t, mem)

	src := NewHalo5CommendationTotalsSource(&memSharedReader{mem}, "xA")
	got, err := src.GetCommendationTotals(context.Background())
	if err != nil {
		t.Fatalf("totals: %v", err)
	}
	byID := map[string]int{}
	for _, c := range got {
		byID[c.ID] = c.Total
	}
	if len(got) != 2 {
		t.Fatalf("totals = %d, want 2 (kills+asst ; null ignoré, xB exclu) — %+v", len(got), got)
	}
	if byID["uuid-kills"] != 150 {
		t.Errorf("uuid-kills = %d, want 150 (dernier match m3)", byID["uuid-kills"])
	}
	if byID["uuid-asst"] != 30 {
		t.Errorf("uuid-asst = %d, want 30", byID["uuid-asst"])
	}
	// ORDER BY progress DESC : kills (150) avant asst (30).
	if len(got) == 2 && got[0].ID != "uuid-kills" {
		t.Errorf("ordre = %s en tête, want uuid-kills (progress DESC)", got[0].ID)
	}
}

func TestHalo5CommendationTotals_NilAndEmpty(t *testing.T) {
	if got, err := NewHalo5CommendationTotalsSource(nil, "x").GetCommendationTotals(context.Background()); err != nil || got != nil {
		t.Errorf("reader nil: got=%v err=%v, want nil neutre", got, err)
	}
	mem := openMemSQL(t)
	seedCommendationTotals(t, mem)
	if got, err := NewHalo5CommendationTotalsSource(&memSharedReader{mem}, "  ").GetCommendationTotals(context.Background()); err != nil || got != nil {
		t.Errorf("xuid vide: got=%v err=%v, want nil neutre", got, err)
	}
}
