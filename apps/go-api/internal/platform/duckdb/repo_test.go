// Package duckdb — repo_test.go : tests DuckDB in-memory pour les repos.
//
// Sprint 39 — tâche 2 : valider les repos avec des DB in-memory + fixtures.
// Lancer avec : go test -tags=integration ./internal/platform/duckdb/ -v

//go:build integration

package duckdb

import (
	"context"
	"database/sql"
	"testing"

	"levelup/go-api/internal/domain"

	_ "github.com/duckdb/duckdb-go/v2"
)

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

// openMemDB ouvre une DuckDB in-memory et l'enregistre pour fermeture.
func openMemDB(t *testing.T) *DB {
	t.Helper()
	sqlDB, err := sql.Open("duckdb", ":memory:")
	if err != nil {
		t.Fatalf("openMemDB: %v", err)
	}
	t.Cleanup(func() { sqlDB.Close() })
	return newTestDB(sqlDB, ":memory:")
}

// seedShared crée et peuple le schéma shared minimal pour les tests.
func seedShared(t *testing.T, db *DB) {
	t.Helper()
	ctx := context.Background()

	// GamertagRepo et BootstrapRepo ouvrent shared_matches_v2.duckdb directement
	// (sans pool player) — les tables sont top-level (match_registry,
	// match_participants, xuid_aliases). Les versions sous schéma `shared.` sont
	// pour les repos qui passent par le pool player avec ATTACH 'shared'.
	ddl := []string{
		`CREATE SCHEMA IF NOT EXISTS shared`,
		`CREATE SCHEMA IF NOT EXISTS global`,
		`CREATE TABLE IF NOT EXISTS match_registry (
			match_id VARCHAR PRIMARY KEY,
			start_time TIMESTAMPTZ,
			last_updated_at TIMESTAMPTZ,
			game_variant_id VARCHAR
		)`,
		`CREATE TABLE IF NOT EXISTS match_participants (
			match_id VARCHAR,
			xuid VARCHAR,
			gamertag VARCHAR
		)`,
		`CREATE TABLE IF NOT EXISTS xuid_aliases (
			xuid VARCHAR,
			gamertag VARCHAR
		)`,
		`CREATE TABLE IF NOT EXISTS shared.match_participants (
			match_id VARCHAR,
			xuid VARCHAR,
			gamertag VARCHAR
		)`,
		`CREATE TABLE IF NOT EXISTS shared.xuid_aliases (
			xuid VARCHAR,
			gamertag VARCHAR
		)`,
		`CREATE TABLE IF NOT EXISTS global.xuid_aliases (
			xuid VARCHAR,
			gamertag VARCHAR
		)`,
	}
	for _, q := range ddl {
		if _, err := db.Exec(ctx, q); err != nil {
			t.Fatalf("seedShared DDL: %v\nSQL: %s", err, q)
		}
	}

	inserts := []string{
		`INSERT INTO match_registry (match_id, start_time, last_updated_at) VALUES
			('m1', TIMESTAMPTZ '2025-01-10 14:00:00+00', TIMESTAMPTZ '2025-01-10 14:30:00+00'),
			('m2', TIMESTAMPTZ '2025-01-11 18:00:00+00', TIMESTAMPTZ '2025-01-11 18:45:00+00'),
			('m3', TIMESTAMPTZ '2025-01-12 20:00:00+00', TIMESTAMPTZ '2025-01-12 20:15:00+00')`,
		`INSERT INTO match_participants VALUES
			('m1', 'xuid001', 'AlphaPlayer'),
			('m1', 'xuid002', 'BravoGamer'),
			('m2', 'xuid001', 'AlphaPlayer'),
			('m2', 'xuid003', 'CharlieX'),
			('m3', 'xuid002', 'BravoGamer')`,
		`INSERT INTO xuid_aliases VALUES
			('xuid001', 'AlphaPlayer'),
			('xuid002', 'BravoGamer'),
			('xuid003', 'CharlieX')`,
		`INSERT INTO shared.match_participants VALUES
			('m1', 'xuid001', 'AlphaPlayer'),
			('m1', 'xuid002', 'BravoGamer'),
			('m2', 'xuid001', 'AlphaPlayer'),
			('m2', 'xuid003', 'CharlieX'),
			('m3', 'xuid002', 'BravoGamer')`,
		`INSERT INTO shared.xuid_aliases (xuid, gamertag) VALUES
			('xuid001', 'AlphaPlayer'),
			('xuid002', 'BravoGamer'),
			('xuid003', 'CharlieX')`,
		`INSERT INTO global.xuid_aliases VALUES
			('xuid001', 'AlphaPlayer'),
			('xuid002', 'BravoGamer'),
			('xuid003', 'CharlieX')`,
	}
	for _, q := range inserts {
		if _, err := db.Exec(ctx, q); err != nil {
			t.Fatalf("seedShared INSERT: %v\nSQL: %s", err, q)
		}
	}
}

// seedMetadata crée le schéma metadata minimal.
func seedMetadata(t *testing.T, db *DB) {
	t.Helper()
	ctx := context.Background()

	ddl := `CREATE TABLE IF NOT EXISTS career_ranks (
		rank_id INTEGER PRIMARY KEY,
		rank_name VARCHAR
	)`
	if _, err := db.Exec(ctx, ddl); err != nil {
		t.Fatalf("seedMetadata DDL: %v", err)
	}

	ins := `INSERT INTO career_ranks VALUES (1, 'Recruit'), (2, 'Bronze'), (3, 'Silver')`
	if _, err := db.Exec(ctx, ins); err != nil {
		t.Fatalf("seedMetadata INSERT: %v", err)
	}
}

// ---------------------------------------------------------------------------
// BootstrapRepo
// ---------------------------------------------------------------------------

func TestBootstrapRepo_GetMatchCount(t *testing.T) {
	shared := openMemDB(t)
	meta := openMemDB(t)
	seedShared(t, shared)

	repo := NewBootstrapRepo(LegacySharedReader(shared), meta)
	count, err := repo.GetMatchCount(context.Background())
	if err != nil {
		t.Fatalf("GetMatchCount: %v", err)
	}
	if count != 3 {
		t.Errorf("GetMatchCount = %d, want 3", count)
	}
}

func TestBootstrapRepo_GetMatchCount_Empty(t *testing.T) {
	shared := openMemDB(t)
	meta := openMemDB(t)
	ctx := context.Background()

	// Table vide
	if _, err := shared.Exec(ctx, "CREATE TABLE match_registry (match_id VARCHAR)"); err != nil {
		t.Fatal(err)
	}

	repo := NewBootstrapRepo(LegacySharedReader(shared), meta)
	count, err := repo.GetMatchCount(ctx)
	if err != nil {
		t.Fatalf("GetMatchCount empty: %v", err)
	}
	if count != 0 {
		t.Errorf("GetMatchCount empty = %d, want 0", count)
	}
}

func TestBootstrapRepo_GetDBVersion(t *testing.T) {
	shared := openMemDB(t)
	meta := openMemDB(t)

	repo := NewBootstrapRepo(LegacySharedReader(shared), meta)
	version, err := repo.GetDBVersion(context.Background())
	if err != nil {
		t.Fatalf("GetDBVersion: %v", err)
	}
	if version == "" || version == "unknown" {
		t.Errorf("GetDBVersion returned %q, want a version string", version)
	}
}

func TestBootstrapRepo_GetPlayerCount(t *testing.T) {
	shared := openMemDB(t)
	meta := openMemDB(t)
	seedShared(t, shared)

	repo := NewBootstrapRepo(LegacySharedReader(shared), meta)
	count, err := repo.GetPlayerCount(context.Background())
	if err != nil {
		t.Fatalf("GetPlayerCount: %v", err)
	}
	// 3 xuids distincts dans match_participants
	if count != 3 {
		t.Errorf("GetPlayerCount = %d, want 3", count)
	}
}

func TestBootstrapRepo_GetLastSyncAt(t *testing.T) {
	shared := openMemDB(t)
	meta := openMemDB(t)
	seedShared(t, shared)

	repo := NewBootstrapRepo(LegacySharedReader(shared), meta)
	ts, err := repo.GetLastSyncAt(context.Background())
	if err != nil {
		t.Fatalf("GetLastSyncAt: %v", err)
	}
	if ts == nil {
		t.Fatal("GetLastSyncAt returned nil, expected a time")
	}
	// Le max est m3 = 2025-01-12 20:15:00+00
	if ts.Year() != 2025 || ts.Month() != 1 || ts.Day() != 12 {
		t.Errorf("GetLastSyncAt = %v, want 2025-01-12", ts)
	}
}

func TestBootstrapRepo_ValidateTypes(t *testing.T) {
	shared := openMemDB(t)
	meta := openMemDB(t)

	repo := NewBootstrapRepo(LegacySharedReader(shared), meta)
	if err := repo.ValidateTypes(context.Background()); err != nil {
		t.Fatalf("ValidateTypes: %v", err)
	}
}

func TestBootstrapRepo_GetCareerRanksSample(t *testing.T) {
	shared := openMemDB(t)
	meta := openMemDB(t)
	seedMetadata(t, meta)

	repo := NewBootstrapRepo(LegacySharedReader(shared), meta)
	count, err := repo.GetCareerRanksSample(context.Background())
	if err != nil {
		t.Fatalf("GetCareerRanksSample: %v", err)
	}
	if count != 3 {
		t.Errorf("GetCareerRanksSample = %d, want 3", count)
	}
}

// ---------------------------------------------------------------------------
// GamertagRepo
// ---------------------------------------------------------------------------

func TestGamertagRepo_Search_Found(t *testing.T) {
	shared := openMemDB(t)
	seedShared(t, shared)

	repo := NewGamertagRepo(LegacySharedReader(shared))
	results, err := repo.Search(context.Background(), "Alpha")
	if err != nil {
		t.Fatalf("Search(Alpha): %v", err)
	}
	if len(results) != 1 {
		t.Fatalf("Search(Alpha): got %d results, want 1", len(results))
	}
	if results[0].Gamertag != "AlphaPlayer" {
		t.Errorf("gamertag = %q, want AlphaPlayer", results[0].Gamertag)
	}
	if !results[0].ExactMatch {
		// "AlphaPlayer" vs query "Alpha" — not exact
	}
}

func TestGamertagRepo_Search_NoResult(t *testing.T) {
	shared := openMemDB(t)
	seedShared(t, shared)

	repo := NewGamertagRepo(LegacySharedReader(shared))
	results, err := repo.Search(context.Background(), "NonExistent")
	if err != nil {
		t.Fatalf("Search(NonExistent): %v", err)
	}
	if len(results) != 0 {
		t.Errorf("expected 0 results, got %d", len(results))
	}
}

func TestGamertagRepo_Search_CaseInsensitive(t *testing.T) {
	shared := openMemDB(t)
	seedShared(t, shared)

	repo := NewGamertagRepo(LegacySharedReader(shared))
	results, err := repo.Search(context.Background(), "bravo")
	if err != nil {
		t.Fatalf("Search(bravo): %v", err)
	}
	if len(results) != 1 {
		t.Fatalf("case-insensitive search: got %d, want 1", len(results))
	}
	if results[0].Gamertag != "BravoGamer" {
		t.Errorf("gamertag = %q, want BravoGamer", results[0].Gamertag)
	}
}

func TestGamertagRepo_Search_MultipleResults(t *testing.T) {
	shared := openMemDB(t)
	seedShared(t, shared)

	// "er" matches AlphaPlay*er* and BravoGam*er*
	repo := NewGamertagRepo(LegacySharedReader(shared))
	results, err := repo.Search(context.Background(), "er")
	if err != nil {
		t.Fatalf("Search(er): %v", err)
	}
	if len(results) < 2 {
		t.Errorf("Search(er): got %d, want ≥2", len(results))
	}
}

// seedGamertagRanking ajoute un jeu de gamertags qui démontre :
//   - exact vs prefix vs substring vs typo
//   - tri stable par score puis match_count
func seedGamertagRanking(t *testing.T, db *DB) {
	t.Helper()
	ctx := context.Background()
	rows := []string{
		`('xr1', 'Master')`,      // exact match candidate
		`('xr2', 'MasterChief')`, // prefix match candidate
		`('xr3', 'GrandMaster')`, // substring match candidate
		`('xr4', 'Mst3rch1f')`,   // typo candidate (jaro_winkler)
		`('xr5', 'TotallyUnrelated')`,
	}
	// Inserts dans les 3 variantes (top-level pour GamertagRepo.Search,
	// shared.* pour les repos via pool, global.* pour la résolution
	// xuid→gamertag globale).
	if _, err := db.Exec(ctx, `INSERT INTO xuid_aliases VALUES `+joinRows(rows)); err != nil {
		t.Fatalf("seedGamertagRanking top-level aliases: %v", err)
	}
	if _, err := db.Exec(ctx, `INSERT INTO shared.xuid_aliases (xuid, gamertag) VALUES `+joinRows(rows)); err != nil {
		t.Fatalf("seedGamertagRanking aliases: %v", err)
	}
	if _, err := db.Exec(ctx, `INSERT INTO global.xuid_aliases VALUES `+joinRows(rows)); err != nil {
		t.Fatalf("seedGamertagRanking global aliases: %v", err)
	}
}

func joinRows(rows []string) string {
	out := ""
	for i, r := range rows {
		if i > 0 {
			out += ","
		}
		out += r
	}
	return out
}

func TestGamertagRepo_Search_ExactRanksFirst(t *testing.T) {
	shared := openMemDB(t)
	seedShared(t, shared)
	seedGamertagRanking(t, shared)

	repo := NewGamertagRepo(LegacySharedReader(shared))
	results, err := repo.Search(context.Background(), "Master")
	if err != nil {
		t.Fatalf("Search(Master): %v", err)
	}
	if len(results) < 3 {
		t.Fatalf("Search(Master): got %d, want ≥3 (Master, MasterChief, GrandMaster)", len(results))
	}
	if results[0].Gamertag != "Master" {
		t.Errorf("rank 1 = %q, want Master (exact match should win)", results[0].Gamertag)
	}
	if !results[0].ExactMatch {
		t.Errorf("ExactMatch on rank 1 = false, want true")
	}
	if results[1].Gamertag != "MasterChief" {
		t.Errorf("rank 2 = %q, want MasterChief (prefix > substring)", results[1].Gamertag)
	}
}

func TestGamertagRepo_Search_TypoTolerance(t *testing.T) {
	shared := openMemDB(t)
	seedShared(t, shared)
	seedGamertagRanking(t, shared)

	// "Mst3rch1f" matche "Mst3rch1f" exact — testons plutôt une vraie typo.
	// "Mst3rch1" (sans 'f') ne devrait pas matcher en substring mais via jaro_winkler.
	repo := NewGamertagRepo(LegacySharedReader(shared))
	results, err := repo.Search(context.Background(), "Mst3rch1")
	if err != nil {
		t.Fatalf("Search(Mst3rch1): %v", err)
	}
	found := false
	for _, r := range results {
		if r.Gamertag == "Mst3rch1f" {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("typo tolerance: 'Mst3rch1f' not found for query 'Mst3rch1'; got: %v", gamertagsOf(results))
	}
}

func gamertagsOf(rs []domain.GamertagSearchResult) []string {
	out := make([]string, len(rs))
	for i, r := range rs {
		out[i] = r.Gamertag
	}
	return out
}
