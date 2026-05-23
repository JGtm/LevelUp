//go:build integration

// Package persist — pve_persister_test.go : tests TDD pour PVEPersister.

package persist

import (
	"context"
	"database/sql"
	"testing"

	_ "github.com/duckdb/duckdb-go/v2"

	"levelup/go-api/internal/migration"
)

// openPVETestDB ouvre une shared_pve in-memory + ajoute les colonnes manquantes.
//
// La migration `create_pve_match_stats` ne crée que 14 colonnes ; le code
// prod attend 20 (sentinel_kills, marine_kills, total_kills, deaths,
// damage_dealt, pve_bits proviennent d'une migration Python ancienne).
// Patch test-local en attendant un fix migration dédié.
func openPVETestDB(t *testing.T) *sql.DB {
	t.Helper()
	db, err := sql.Open("duckdb", ":memory:")
	if err != nil {
		t.Fatalf("open duckdb: %v", err)
	}
	t.Cleanup(func() { _ = db.Close() })
	if err := migration.RunForDB(db, migration.TargetSharedPvE); err != nil {
		t.Fatalf("migrate pve: %v", err)
	}
	for _, col := range []string{
		"sentinel_kills INTEGER",
		"marine_kills INTEGER",
		"damage_dealt DOUBLE",
		"pve_bits INTEGER",
	} {
		if _, err := db.Exec("ALTER TABLE pve_match_stats ADD COLUMN IF NOT EXISTS " + col); err != nil {
			t.Fatalf("patch pve %s: %v", col, err)
		}
	}
	return db
}

func helperPVEBatch(matchID, xuid string) *MatchBatch {
	intPtr := func(v int) *int { return &v }
	b := NewBatchBuilder("halo_infinite", "Alice", xuid, "test")
	b.AddPVEStats([]PVEMatchStatsInsert{{
		MatchID:        matchID,
		XUID:           xuid,
		WavesCompleted: 5,
		BossKills:      2,
		GruntKills:     30,
		EliteKills:     5,
		JackalKills:    3,
		BruteKills:     2,
		HunterKills:    1,
		SkimmerKills:   0,
		CrawlerKills:   0,
		SoldierKills:   0,
		KnightKills:    0,
		WardenKills:    0,
		SentinelKills:  0,
		MarineKills:    0,
		TotalKills:     43,
		Deaths:         8,
		DamageDealt:    12500.5,
		PveBits:        intPtr(0xFFFF),
	}})
	return b.Build()
}

// ─── Test 1 : Persist nominal ─────────────────────────────────────────────

func TestPVEPersister_Persist_InsertsRow(t *testing.T) {
	db := openPVETestDB(t)
	p := NewPVEPersister(db)

	batch := helperPVEBatch("pve_001", "1111")
	if err := p.Persist(context.Background(), batch); err != nil {
		t.Fatalf("Persist: %v", err)
	}

	var (
		waves, boss, total, deaths int
		dmg                        float64
		pveBits                    sql.NullInt64
	)
	err := db.QueryRow(`
		SELECT waves_completed, boss_kills, total_kills, deaths, damage_dealt, pve_bits
		FROM pve_match_stats WHERE match_id = ? AND xuid = ?`,
		"pve_001", "1111").
		Scan(&waves, &boss, &total, &deaths, &dmg, &pveBits)
	if err != nil {
		t.Fatalf("query: %v", err)
	}
	if waves != 5 || boss != 2 || total != 43 || deaths != 8 {
		t.Errorf("counts mismatch : waves=%d boss=%d total=%d deaths=%d", waves, boss, total, deaths)
	}
	if dmg != 12500.5 {
		t.Errorf("damage_dealt = %f, want 12500.5", dmg)
	}
	if !pveBits.Valid || pveBits.Int64 != 0xFFFF {
		t.Errorf("pve_bits = %+v, want 0xFFFF", pveBits)
	}
}

// ─── Test 2 : No PVE in batch → no-op ─────────────────────────────────────

func TestPVEPersister_Persist_NoPVE_NoOp(t *testing.T) {
	db := openPVETestDB(t)
	p := NewPVEPersister(db)

	// Batch sans PVEStats
	builder := NewBatchBuilder("halo_infinite", "Alice", "1111", "test")
	if err := p.Persist(context.Background(), builder.Build()); err != nil {
		t.Fatalf("Persist no-op: %v", err)
	}

	var n int
	_ = db.QueryRow(`SELECT COUNT(*) FROM pve_match_stats`).Scan(&n)
	if n != 0 {
		t.Errorf("pve_match_stats doit rester vide (no-op), got %d", n)
	}
}

// ─── Test 3 : Re-persist (PK déjà présent) → INSERT OR IGNORE ─────────────

func TestPVEPersister_Persist_DuplicatePK_Idempotent(t *testing.T) {
	db := openPVETestDB(t)
	p := NewPVEPersister(db)

	if err := p.Persist(context.Background(), helperPVEBatch("pve_dup", "1111")); err != nil {
		t.Fatalf("1er Persist: %v", err)
	}

	// 2e batch même PK, total_kills différent → doit être skip
	b2 := helperPVEBatch("pve_dup", "1111")
	b2.PVE.Stats[0].TotalKills = 99999

	if err := p.Persist(context.Background(), b2); err != nil {
		t.Fatalf("2e Persist (idempotent attendu): %v", err)
	}

	var got int
	if err := db.QueryRow(`SELECT total_kills FROM pve_match_stats WHERE match_id = ? AND xuid = ?`,
		"pve_dup", "1111").Scan(&got); err != nil {
		t.Fatal(err)
	}
	if got != 43 {
		t.Errorf("total_kills = %d, want 43 (INSERT OR IGNORE)", got)
	}
}

// ─── Test 4 : NilBatch → erreur ────────────────────────────────────────────

func TestPVEPersister_Persist_NilBatch_ReturnsError(t *testing.T) {
	db := openPVETestDB(t)
	p := NewPVEPersister(db)
	if err := p.Persist(context.Background(), nil); err == nil {
		t.Error("Persist(nil) devrait retourner une erreur")
	}
}
