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
	"errors"
	"path/filepath"
	"testing"
	"time"

	_ "github.com/duckdb/duckdb-go/v2"

	"levelup/go-api/internal/domain"
	titlePkg "levelup/go-api/internal/domain/title"
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
	// activePlaylists : playlists « découvertes ». nil → le cron retombe sur la liste
	// statique (comportement par défaut des tests historiques).
	activePlaylists     []domain.WorldPlaylistRef
	activePlaylistsErr  error
	activePlaylistCalls int
}

func (s *stubScraper) FetchActiveSeason(_ context.Context, _ string) (string, error) {
	s.activeCalls++
	if s.seasonErr != nil {
		return "", s.seasonErr
	}
	return s.season, nil
}

func (s *stubScraper) FetchActivePlaylists(_ context.Context, _ string) ([]domain.WorldPlaylistRef, error) {
	s.activePlaylistCalls++
	return s.activePlaylists, s.activePlaylistsErr
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
	// Registre frais (halo_infinite seul, avec CapWorldLeaderboard) : isole les
	// tests d'un éventuel SetDefaultRegistry global posé par un autre test.
	c.registry = titlePkg.NewRegistry()
	return c
}

// TestWorldLeaderboardCron_DiscoversActivePlaylists vérifie que le cron scrape les
// playlists ACTIVES découvertes sur Waypoint et non la liste statique (2 en test) :
// sans ça, seules les playlists scrapées ont des snapshots, donc la page classement
// n'en affiche qu'une poignée (cause racine A2).
func TestWorldLeaderboardCron_DiscoversActivePlaylists(t *testing.T) {
	provider, db := newSharedProviderForTest(t)
	scraper := &stubScraper{
		season:  "csrseason13-2",
		entries: []domain.LeaderboardEntry{{Rank: 1, Gamertag: "Alpha", XUID: "2535000000000001", CSRValue: 1500}},
		activePlaylists: []domain.WorldPlaylistRef{
			{AssetID: "pl-1", DisplayName: "Ranked Arena"},
			{AssetID: "pl-2", DisplayName: "Ranked Slayer"},
			{AssetID: "pl-3", DisplayName: "Ranked Snipers"},
		},
	}
	c := newTestCron(provider, scraper)
	c.RunOnce(context.Background())

	if scraper.activePlaylistCalls == 0 {
		t.Errorf("FetchActivePlaylists jamais appelé (découverte non câblée)")
	}
	if scraper.fetchCalls != 3 {
		t.Errorf("FetchCSRLeaderboard = %d, want 3 (playlists découvertes, pas les 2 statiques)", scraper.fetchCalls)
	}
	if n := countSnapshots(t, db, "csrseason13-2"); n != 3 {
		t.Errorf("snapshots = %d, want 3 (une entrée par playlist découverte)", n)
	}
}

// TestWorldLeaderboardCron_FallbackStaticPlaylists : si la découverte échoue, le cron
// retombe sur la liste statique (jamais zéro playlist scrapée).
func TestWorldLeaderboardCron_FallbackStaticPlaylists(t *testing.T) {
	provider, _ := newSharedProviderForTest(t)
	scraper := &stubScraper{
		season:             "csrseason13-2",
		entries:            []domain.LeaderboardEntry{{Rank: 1, Gamertag: "Alpha", CSRValue: 1500}},
		activePlaylistsErr: errors.New("page cassée"),
	}
	c := newTestCron(provider, scraper) // statique = 2 playlists
	c.RunOnce(context.Background())

	if scraper.fetchCalls != 2 {
		t.Errorf("FetchCSRLeaderboard = %d, want 2 (fallback statique sur erreur découverte)", scraper.fetchCalls)
	}
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

// TestWorldLeaderboardCron_CapabilityGated vérifie la brique title-aware (PMT-7) :
// le cron itère les titres actifs et ne produit un snapshot QUE pour ceux qui
// déclarent CapWorldLeaderboard. Un titre actif SANS la cap (ex. Halo 5) est
// skippé proprement — aucun scrape, aucun InsertWorldCSRSnapshot pour lui —
// tandis que halo_infinite (avec la cap) reste traité normalement.
func TestWorldLeaderboardCron_CapabilityGated(t *testing.T) {
	provider, db := newSharedProviderForTest(t)
	scraper := &stubScraper{
		season: "csrseason13-2",
		entries: []domain.LeaderboardEntry{
			{Rank: 1, Gamertag: "Alpha", CSRValue: 1800, Tier: "Onyx", FetchedAt: time.Now().UTC()},
			{Rank: 2, Gamertag: "Bravo", CSRValue: 1700, Tier: "Diamond", FetchedAt: time.Now().UTC()},
		},
	}
	c := newTestCron(provider, scraper)

	// Registre à 2 titres ACTIFS : halo_infinite (built-in, a CapWorldLeaderboard)
	// + un titre sans la cap (modèle Halo 5 : pas de classement mondial scrapable).
	reg := titlePkg.NewRegistry() // halo_infinite déjà enregistré avec la cap
	reg.Register(&titlePkg.TitleDescriptor{
		Slug:         "title_no_leaderboard",
		Name:         "Title Sans Classement",
		Provider:     "title_no_leaderboard",
		Status:       titlePkg.StatusActive,
		Capabilities: []titlePkg.Capability{titlePkg.CapMatchmaking}, // pas de CapWorldLeaderboard
	})
	c.registry = reg

	c.RunOnce(context.Background())

	// Le titre avec la cap a été scrapé une fois par playlist (2) → 2 fetchs ;
	// le titre sans la cap ne déclenche aucun fetch supplémentaire.
	if scraper.fetchCalls != 2 {
		t.Errorf("FetchCSRLeaderboard attendu 2× (halo_infinite seul), got %d", scraper.fetchCalls)
	}
	// La saison n'est découverte qu'une fois (titre avec cap), pas pour le capless.
	if scraper.activeCalls != 1 {
		t.Errorf("FetchActiveSeason attendu 1× (halo_infinite seul), got %d", scraper.activeCalls)
	}

	// halo_infinite traité : 2 playlists × 2 entrées = 4 lignes sous son slug.
	if got := countSnapshotsForTitle(t, db, "halo_infinite", "csrseason13-2"); got != 4 {
		t.Fatalf("halo_infinite : %d lignes, attendu 4", got)
	}
	// Aucun snapshot pour le titre sans la cap (ni sous son slug, ni global).
	if got := countSnapshotsForTitle(t, db, "title_no_leaderboard", "csrseason13-2"); got != 0 {
		t.Errorf("titre sans CapWorldLeaderboard : %d lignes, attendu 0 (skippé)", got)
	}
}

// countSnapshotsForTitle compte les snapshots d'une saison filtrés par title_slug.
func countSnapshotsForTitle(t *testing.T, db *sql.DB, titleSlug, season string) int {
	t.Helper()
	var n int
	if err := db.QueryRow(
		`SELECT COUNT(*) FROM world_csr_leaderboard_snapshots WHERE title_slug = ? AND season_id = ?`,
		titleSlug, season,
	).Scan(&n); err != nil {
		t.Fatalf("count snapshots by title: %v", err)
	}
	return n
}
