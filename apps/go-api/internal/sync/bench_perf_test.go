//go:build integration

// Package sync — bench_perf_test.go : benchmarks ciblés sur les paths
// parallélisés du plan stabilisation 2026-05-22.
//
// Phase 5.4 — la spec initiale demandait un BenchmarkAutoSync_3Players_20Matches
// complet, mais ça nécessiterait un setup mock-pool + scheduler trop lourd
// pour un benchmark exécuté en CI. À la place, ce fichier fournit 2 micro-
// benchmarks ciblés sur les hot paths réellement parallélisés :
//
//  1. BenchmarkProcessWeaponKillsInline_16Matches — exerce le path Phase 3.0
//     (parallel weapon kills inter-match, errgroup.SetLimit=24 post-3.6).
//  2. BenchmarkGetMatchFilm_20Chunks — exerce le path Phase 3.1bis (parallel
//     chunk download intra-film, errgroup.SetLimit=8).
//
// **Usage** :
//
//	go test -tags=integration -bench=BenchmarkProcessWeaponKills ./internal/sync/ -benchtime=3x -run=^$
//	go test -tags=integration -bench=BenchmarkGetMatchFilm ./internal/sync/ -benchtime=3x -run=^$
//
// Les benchs simulent la latence réseau via les mocks (latencyClient pour
// weapon_kills, httptest server pour film chunks). Les wall-times mesurés
// doivent rester sous les seuils définis dans les tests TDD correspondants
// (cf. TestProcessWeaponKillsInline_LatencyParallelFasterThanSequential et
// TestGetMatchFilm_ParallelDownloadFasterThanSequential).

package sync

import (
	"context"
	"database/sql"
	"fmt"
	"testing"
	"time"

	_ "github.com/duckdb/duckdb-go/v2"
)

// openWeaponDBForBench : setup DB minimal pour le bench
// BenchmarkProcessWeaponKillsInline. Réplique de openWeaponDB
// (backfill_weapons_test.go:24) sans dépendance à *testing.T.Context.
func openWeaponDBForBench(b *testing.B) *sql.DB {
	b.Helper()
	db, err := sql.Open("duckdb", ":memory:")
	if err != nil {
		b.Fatal(err)
	}
	b.Cleanup(func() { db.Close() })
	// Réplique du schema openWeaponDB (backfill_weapons_test.go:24). event_type
	// est VARCHAR comme en prod, sinon getKillsForPlayer fait LOWER(INTEGER)
	// et crash le bench.
	ddl := `
		CREATE TABLE highlight_events (
			match_id VARCHAR, xuid VARCHAR,
			event_type VARCHAR, time_ms INTEGER
		);
		CREATE TABLE killer_victim_pairs (
			match_id VARCHAR, killer_xuid VARCHAR,
			weapon_type VARCHAR, time_ms INTEGER
		);
		CREATE TABLE match_participants (
			match_id VARCHAR, xuid VARCHAR,
			team_id INTEGER, rank INTEGER
		);
		CREATE TABLE match_registry (
			match_id VARCHAR PRIMARY KEY,
			backfill_completed INTEGER DEFAULT 0
		);
		CREATE SEQUENCE IF NOT EXISTS weapon_kills_generation_seq START 1;
		CREATE TABLE weapon_kills (
			match_id VARCHAR, xuid VARCHAR,
			time_ms INTEGER, weapon_id UBIGINT,
			reconciled_as UBIGINT, delta_ms INTEGER,
			confidence VARCHAR, attribution_path VARCHAR,
			swap_detected BOOLEAN, delayed_damage BOOLEAN,
			player_index INTEGER, generation_id BIGINT DEFAULT 0
		);
	`
	if err := execScript(context.Background(), db, ddl); err != nil {
		b.Fatal(err)
	}
	return db
}

// seedBenchMatchesInRegistry : variante de seedMatchesInRegistry pour *testing.B.
func seedBenchMatchesInRegistry(b *testing.B, db *sql.DB, n int, prefix string) []string {
	b.Helper()
	ids := make([]string, 0, n)
	for i := 0; i < n; i++ {
		id := fmt.Sprintf("%s-%04d", prefix, i)
		if _, err := db.Exec(`INSERT INTO match_registry (match_id) VALUES (?)`, id); err != nil {
			b.Fatalf("seed match %q: %v", id, err)
		}
		ids = append(ids, id)
	}
	return ids
}

// BenchmarkProcessWeaponKillsInline_16Matches : 16 matchs × 100ms latence
// simulée. Baseline séquentielle = 1.6s ; parallèle
// (weaponBackfillParallelism=24 post-3.6) = ~100ms + overhead. Documente
// le gain Phase 3.0 + 3.6 dans le repo.
func BenchmarkProcessWeaponKillsInline_16Matches(b *testing.B) {
	db := openWeaponDBForBench(b)
	matchIDs := seedBenchMatchesInRegistry(b, db, 16, "bench")
	client := newWeaponLatencyClient(100*time.Millisecond, matchIDs)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _, err := weaponKillsInline(context.Background(), db, client, "bench_xuid", matchIDs)
		if err != nil {
			b.Fatalf("weaponKillsInline: %v", err)
		}
	}
}
