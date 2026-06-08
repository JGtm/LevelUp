//go:build cgo
// +build cgo

// world_leaderboard_cron_test.go — vérifie le cycle du cron classement mondial :
// découverte saison active, insert append-only via le SharedDBProvider, et
// garde-fou de fraîcheur (skip si snapshot récent / re-scrape si périmé).
//
// Package interne (scheduler) pour piloter les champs non exportés (freshness,
// playlists, limit) sans exposer d'options de test dans l'API publique.
package scheduler

import (
	"context"
	"database/sql"
	"path/filepath"
	"testing"
	"time"

	_ "github.com/duckdb/duckdb-go/v2"

	"levelup/go-api/internal/domain"
	"levelup/go-api/internal/migration"
	"levelup/go-api/internal/platform/duckdb/sharedprovider"
)

// stubScraper retourne des entrées canned et compte les appels.
type stubScraper struct {
	season      string
	seasonErr   error
	entries     []domain.LeaderboardEntry
	fetchErr    error
	activeCalls int
	fetchCalls  int
}

func (s *stubScraper) FetchActiveSeason(_ context.Context, _ string) (string, error) {
	s.activeCalls++
	if s.seasonErr != nil {
		return "", s.seasonErr
	}
	return s.season, nil
}

func (s *stubScraper) FetchCSRLeaderboard(_ context.Context, season, playlist string, _ int) ([]domain.LeaderboardEntry, error) {
	s.fetchCalls++
	if s.fetchErr != nil {
		return nil, s.fetchErr
	}
	// Recopie les entrées en fixant season/playlist (comme le vrai scraper).
	out := make([]domain.LeaderboardEntry, len(s.entries))
	for i, e := range s.entries {
		e.Season = season
		e.Playlist = playlist
		out[i] = e
	}
	return out, nil
}

// newSharedProviderForTest ouvre une shared DuckDB sur fichier temp avec la
// migration TargetShared appliquée, wrappée dans un Provider in-memory.
func newSharedProviderForTest(t *testing.T) (sharedprovider.Provider, *sql.DB) {
	t.Helper()
	dbPath := filepath.Join(t.TempDir(), "shared.duckdb")
	db, err := sql.Open("duckdb", dbPath)
	if err != nil {
		t.Fatalf("open duckdb: %v", err)
	}
	db.SetMaxOpenConns(1) // une seule conn : Get et AcquireWriter partagent le handle
	t.Cleanup(func() { _ = db.Close() })
	if err := migration.RunForDB(db, migration.TargetShared); err != nil {
		t.Fatalf("migration shared: %v", err)
	}
	return sharedprovider.FromInMemoryDB(db, dbPath), db
}

func countSnapshots(t *testing.T, db *sql.DB, season string) int {
	t.Helper()
	var n int
	if err := db.QueryRow(
		`SELECT COUNT(*) FROM world_csr_leaderboard_snapshots WHERE season_id = ?`, season,
	).Scan(&n); err != nil {
		t.Fatalf("count snapshots: %v", err)
	}
	return n
}

func newTestCron(provider sharedprovider.Provider, scraper LeaderboardScraperPort) *WorldLeaderboardCron {
	c := NewWorldLeaderboardCron(provider, scraper, 0)
	c.playlists = func() []string { return []string{"pl-a", "pl-b"} } // 2 playlists déterministes
	c.minEntries = 1                                                  // neutralise le plancher (testé séparément)
	return c
}

// TestWorldLeaderboardCron_InsertAndFreshness couvre : insert au 1er cycle, skip
// au 2e (snapshot frais), re-scrape quand le snapshot est périmé.
func TestWorldLeaderboardCron_InsertAndFreshness(t *testing.T) {
	provider, db := newSharedProviderForTest(t)
	scraper := &stubScraper{
		season: "csrseason13-2",
		entries: []domain.LeaderboardEntry{
			{Rank: 1, Gamertag: "Alpha", CSRValue: 1800, Tier: "Onyx", FetchedAt: time.Now().UTC()},
			{Rank: 2, Gamertag: "Bravo", CSRValue: 1700, Tier: "Diamond", FetchedAt: time.Now().UTC()},
		},
	}
	c := newTestCron(provider, scraper)
	ctx := context.Background()

	// Cycle 1 : insert (2 playlists × 2 entrées = 4 lignes).
	c.RunOnce(ctx)
	if got := countSnapshots(t, db, "csrseason13-2"); got != 4 {
		t.Fatalf("après cycle 1 : %d lignes, attendu 4", got)
	}

	// Cycle 2 : snapshot frais (< freshness 20h) → skip, aucune nouvelle ligne,
	// et le scraper n'est PAS appelé pour les playlists.
	fetchBefore := scraper.fetchCalls
	c.RunOnce(ctx)
	if got := countSnapshots(t, db, "csrseason13-2"); got != 4 {
		t.Fatalf("après cycle 2 (frais) : %d lignes, attendu 4 (skip)", got)
	}
	if scraper.fetchCalls != fetchBefore {
		t.Errorf("scraper.FetchCSRLeaderboard appelé malgré le skip fraîcheur (%d → %d)",
			fetchBefore, scraper.fetchCalls)
	}

	// Cycle 3 : on force la péremption (freshness ~0) → re-scrape + insert.
	c.freshness = time.Nanosecond
	c.RunOnce(ctx)
	if got := countSnapshots(t, db, "csrseason13-2"); got != 8 {
		t.Fatalf("après cycle 3 (périmé) : %d lignes, attendu 8 (re-scrape)", got)
	}
}

// TestWorldLeaderboardCron_SeasonDiscoveryError : si la découverte de saison
// échoue, le cycle est ignoré proprement (pas de panic, pas d'insert, pas de
// scrape de playlist).
func TestWorldLeaderboardCron_SeasonDiscoveryError(t *testing.T) {
	provider, db := newSharedProviderForTest(t)
	scraper := &stubScraper{seasonErr: context.DeadlineExceeded}
	c := newTestCron(provider, scraper)

	c.RunOnce(context.Background())

	if scraper.fetchCalls != 0 {
		t.Errorf("FetchCSRLeaderboard ne doit pas être appelé si la saison est indécouvrable (%d)", scraper.fetchCalls)
	}
	if got := countSnapshots(t, db, "csrseason13-2"); got != 0 {
		t.Errorf("aucune ligne attendue sur échec de découverte, got %d", got)
	}
}

// TestWorldLeaderboardCron_SanityFloor : un scrape non vide mais sous le plancher
// de cohérence est ignoré (glitch probable) → aucun insert.
func TestWorldLeaderboardCron_SanityFloor(t *testing.T) {
	provider, db := newSharedProviderForTest(t)
	scraper := &stubScraper{
		season: "csrseason13-2",
		entries: []domain.LeaderboardEntry{ // 3 entrées seulement
			{Rank: 1, Gamertag: "A", CSRValue: 1800, Tier: "Onyx", FetchedAt: time.Now().UTC()},
			{Rank: 2, Gamertag: "B", CSRValue: 1700, Tier: "Diamond", FetchedAt: time.Now().UTC()},
			{Rank: 3, Gamertag: "C", CSRValue: 1600, Tier: "Diamond", FetchedAt: time.Now().UTC()},
		},
	}
	c := newTestCron(provider, scraper)
	c.minEntries = 25 // plancher actif : 3 < 25 → skip

	c.RunOnce(context.Background())

	if got := countSnapshots(t, db, "csrseason13-2"); got != 0 {
		t.Errorf("snapshot trop court devrait être ignoré, mais %d lignes insérées", got)
	}
}

// TestWorldLeaderboardCron_NilGuards : provider/scraper nil → noop sans panic.
func TestWorldLeaderboardCron_NilGuards(t *testing.T) {
	(&WorldLeaderboardCron{}).RunOnce(context.Background())
	NewWorldLeaderboardCron(nil, nil, 0).RunOnce(context.Background())
}
