//go:build integration

// shared_backfill_is_ranked_test.go — déplacé depuis internal/migration (Phase 1.5 b18).
// shared_backfill_is_ranked_and_season est title-owned : résolu via StepsFor(TargetShared)
// et appliqué en isolation (ApplySchema + ApplyBackfill, sans RunForDB qui chaînerait
// toutes les migrations shared).
package migrations

import (
	"database/sql"
	"testing"
	"time"

	_ "github.com/duckdb/duckdb-go/v2"

	"levelup/go-api/internal/migration"
)

func seedSharedMatchRegistry(t *testing.T, db *sql.DB) {
	t.Helper()
	if _, err := db.Exec(`
		CREATE TABLE IF NOT EXISTS match_registry (
			match_id      VARCHAR PRIMARY KEY,
			start_time    TIMESTAMP NOT NULL,
			playlist_name VARCHAR,
			pair_name     VARCHAR,
			is_ranked     BOOLEAN DEFAULT FALSE
		);
	`); err != nil {
		t.Fatalf("create match_registry: %v", err)
	}
	rows := []struct {
		id        string
		start     time.Time
		playlist  string
		pair      string
		preranked bool
	}{
		{"m_ranked_arena_s13", mustTime("2026-04-01T12:00:00Z"), "Ranked Arena", "Ranked:CTF on Live Fire", false},
		{"m_ranked_slayer_s13", mustTime("2026-03-15T18:00:00Z"), "Ranked Slayer", "Ranked:Slayer on Recharge", false},
		{"m_ranked_arena_s2", mustTime("2022-08-10T20:00:00Z"), "Ranked Arena", "Ranked:Oddball on Live Fire", false},
		{"m_quickplay", mustTime("2026-04-02T12:00:00Z"), "Quick Play", "Slayer on Aquarius", false},
		{"m_btb", mustTime("2026-04-02T13:00:00Z"), "Big Team Battle", "CTF on Fragmentation", false},
		{"m_already_ranked", mustTime("2026-04-03T12:00:00Z"), "Ranked Doubles", "Ranked:Slayer on Empyrean", true},
		{"m_old_pre_s1", mustTime("2021-06-01T12:00:00Z"), "Ranked Arena", "Ranked:Slayer on Bazaar", false},
	}
	for _, r := range rows {
		if _, err := db.Exec(
			`INSERT INTO match_registry (match_id, start_time, playlist_name, pair_name, is_ranked) VALUES (?, ?, ?, ?, ?)`,
			r.id, r.start, r.playlist, r.pair, r.preranked,
		); err != nil {
			t.Fatalf("insert %s: %v", r.id, err)
		}
	}
}

func mustTime(s string) time.Time {
	t, err := time.Parse(time.RFC3339, s)
	if err != nil {
		panic(err)
	}
	return t
}

// applyMigrationInIsolation applique une migration title-owned unique par nom (résolue
// via StepsFor(TargetShared)) sans passer par RunForDB.
func applyMigrationInIsolation(t *testing.T, db *sql.DB, name string) error {
	t.Helper()
	for _, m := range StepsFor(migration.TargetShared) {
		if m.Name != name {
			continue
		}
		if err := m.ApplySchema(db); err != nil {
			return err
		}
		if m.ApplyBackfill != nil {
			if err := m.ApplyBackfill(db); err != nil {
				return err
			}
		}
		return nil
	}
	t.Fatalf("migration introuvable dans StepsFor(TargetShared): %s", name)
	return nil
}

func TestSharedBackfillIsRankedAndSeason_AddsSeasonIDColumn(t *testing.T) {
	db := openEngMemDB(t)
	seedSharedMatchRegistry(t, db)

	if err := applyMigrationInIsolation(t, db, "shared_backfill_is_ranked_and_season"); err != nil {
		t.Fatalf("apply migration: %v", err)
	}

	var cnt int
	err := db.QueryRow(`
		SELECT COUNT(*) FROM information_schema.columns
		WHERE table_name='match_registry' AND column_name='season_id'
	`).Scan(&cnt)
	if err != nil {
		t.Fatalf("query column: %v", err)
	}
	if cnt != 1 {
		t.Errorf("colonne season_id absente après migration (cnt=%d)", cnt)
	}
}

func TestSharedBackfillIsRankedAndSeason_MarksRankedFromName(t *testing.T) {
	db := openEngMemDB(t)
	seedSharedMatchRegistry(t, db)

	if err := applyMigrationInIsolation(t, db, "shared_backfill_is_ranked_and_season"); err != nil {
		t.Fatalf("apply migration: %v", err)
	}

	cases := map[string]bool{
		"m_ranked_arena_s13":  true,
		"m_ranked_slayer_s13": true,
		"m_ranked_arena_s2":   true,
		"m_quickplay":         false,
		"m_btb":               false,
		"m_already_ranked":    true,
		"m_old_pre_s1":        true,
	}
	for matchID, want := range cases {
		var got bool
		if err := db.QueryRow(`SELECT is_ranked FROM match_registry WHERE match_id=?`, matchID).Scan(&got); err != nil {
			t.Errorf("scan %s: %v", matchID, err)
			continue
		}
		if got != want {
			t.Errorf("match=%s is_ranked=%v, want %v", matchID, got, want)
		}
	}
}

func TestSharedBackfillIsRankedAndSeason_DerivesSeasonIDFromStartTime(t *testing.T) {
	db := openEngMemDB(t)
	seedSharedMatchRegistry(t, db)

	if err := applyMigrationInIsolation(t, db, "shared_backfill_is_ranked_and_season"); err != nil {
		t.Fatalf("apply migration: %v", err)
	}

	cases := map[string]struct {
		want   string
		isNull bool
	}{
		"m_ranked_arena_s13":  {want: "CsrSeason13-1"},
		"m_ranked_slayer_s13": {want: "CsrSeason13-1"},
		"m_ranked_arena_s2":   {want: "CsrSeason2"},
		"m_already_ranked":    {want: "CsrSeason13-1"},
		"m_old_pre_s1":        {isNull: true},
		"m_quickplay":         {isNull: true},
		"m_btb":               {isNull: true},
	}
	for matchID, c := range cases {
		var sid sql.NullString
		if err := db.QueryRow(`SELECT season_id FROM match_registry WHERE match_id=?`, matchID).Scan(&sid); err != nil {
			t.Errorf("scan %s: %v", matchID, err)
			continue
		}
		if c.isNull {
			if sid.Valid {
				t.Errorf("match=%s season_id=%q, want NULL", matchID, sid.String)
			}
			continue
		}
		if !sid.Valid || sid.String != c.want {
			t.Errorf("match=%s season_id=%v, want %q", matchID, sid, c.want)
		}
	}
}

func TestSharedBackfillIsRankedAndSeason_Idempotent(t *testing.T) {
	db := openEngMemDB(t)
	seedSharedMatchRegistry(t, db)

	if err := applyMigrationInIsolation(t, db, "shared_backfill_is_ranked_and_season"); err != nil {
		t.Fatalf("1er apply: %v", err)
	}
	if err := applyMigrationInIsolation(t, db, "shared_backfill_is_ranked_and_season"); err != nil {
		t.Fatalf("2e apply: %v", err)
	}

	var ranked int
	if err := db.QueryRow(`SELECT COUNT(*) FROM match_registry WHERE is_ranked=TRUE`).Scan(&ranked); err != nil {
		t.Fatalf("count ranked: %v", err)
	}
	if ranked < 4 {
		t.Errorf("après 2e run, expected ≥4 ranked rows, got %d", ranked)
	}
}

func TestSharedBackfillIsRankedAndSeason_PreservesPreRanked(t *testing.T) {
	db := openEngMemDB(t)
	seedSharedMatchRegistry(t, db)

	if err := applyMigrationInIsolation(t, db, "shared_backfill_is_ranked_and_season"); err != nil {
		t.Fatalf("apply migration: %v", err)
	}

	var got bool
	if err := db.QueryRow(`SELECT is_ranked FROM match_registry WHERE match_id='m_already_ranked'`).Scan(&got); err != nil {
		t.Fatalf("scan: %v", err)
	}
	if !got {
		t.Error("m_already_ranked devrait rester is_ranked=TRUE")
	}
}
