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
	"bytes"
	"compress/zlib"
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	_ "github.com/duckdb/duckdb-go/v2"
)

// zlibCompressForBench : variante zlib-compression pour *testing.B. Ne peut
// pas réutiliser zlibCompress(t *testing.T) directement à cause du type.
// Identique fonctionnellement.
func zlibCompressForBench(b *testing.B, data []byte) []byte {
	b.Helper()
	var buf bytes.Buffer
	w := zlib.NewWriter(&buf)
	if _, err := w.Write(data); err != nil {
		b.Fatalf("zlib write: %v", err)
	}
	if err := w.Close(); err != nil {
		b.Fatalf("zlib close: %v", err)
	}
	return buf.Bytes()
}

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
		CREATE TABLE weapon_kills (
			match_id VARCHAR, xuid VARCHAR,
			time_ms INTEGER, weapon_id UBIGINT,
			reconciled_as UBIGINT, delta_ms INTEGER,
			confidence VARCHAR, attribution_path VARCHAR,
			swap_detected BOOLEAN, delayed_damage BOOLEAN,
			player_index INTEGER
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
// (healParallelismNetworkOnly=24 post-3.6) = ~100ms + overhead. Documente
// le gain Phase 3.0 + 3.6 dans le repo.
func BenchmarkProcessWeaponKillsInline_16Matches(b *testing.B) {
	db := openWeaponDBForBench(b)
	matchIDs := seedBenchMatchesInRegistry(b, db, 16, "bench")
	client := newWeaponLatencyClient(100*time.Millisecond, matchIDs)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _, err := processWeaponKillsInline(context.Background(), db, client, "bench_xuid", matchIDs)
		if err != nil {
			b.Fatalf("processWeaponKillsInline: %v", err)
		}
	}
}

// BenchmarkGetMatchFilm_20Chunks : 20 chunks × 50ms latence CDN simulée.
// Baseline séquentielle = 1s ; parallèle (filmChunkParallelism=8) = ~150ms
// + overhead. Documente le gain Phase 3.1bis dans le repo.
func BenchmarkGetMatchFilm_20Chunks(b *testing.B) {
	const nChunks = 20
	const latency = 50 * time.Millisecond

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.Contains(r.URL.Path, "/spectate") {
			_ = json.NewEncoder(w).Encode(filmManyChunks("http://blobs.test/", nChunks))
			return
		}
		// Sleep + retour de blob zlib-compressé.
		time.Sleep(latency)
		// zlibCompress prend un testing.T ; on encode manuellement via le
		// helper exporté pour éviter le shim T/B.
		_, _ = w.Write(zlibCompressForBench(b, []byte("data-"+r.URL.Path)))
	}))
	b.Cleanup(srv.Close)

	c := newFilmTestClient(srv)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _, err := c.GetMatchFilm(context.Background(), testFilmMatchUUID)
		if err != nil {
			b.Fatalf("GetMatchFilm: %v", err)
		}
	}
}
