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
	"log/slog"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	_ "github.com/duckdb/duckdb-go/v2"

	"levelup/go-api/internal/domain"
	titlePkg "levelup/go-api/internal/domain/title"
	"levelup/go-api/internal/migration"
	"levelup/go-api/internal/observability"
	"levelup/go-api/internal/platform/duckdb"
	"levelup/go-api/internal/platform/duckdb/sharedprovider"
	"levelup/go-api/internal/platform/halo"
)

// stubScraper retourne des entrées canned et compte les appels.
type stubScraper struct {
	season    string
	seasonErr error
	// seasonErrForPlaylists : playlists de référence qui font échouer FetchActiveSeason
	// (simule un 404 « non classée dans la saison-graine »). Prioritaire sur seasonErr ;
	// les playlists absentes de la map réussissent normalement.
	seasonErrForPlaylists map[string]error
	entries               []domain.LeaderboardEntry
	fetchErr              error
	activeCalls           int
	fetchCalls            int
	// activePlaylists : playlists « découvertes ». nil → le cron retombe sur la liste
	// statique (comportement par défaut des tests historiques).
	activePlaylists     []domain.WorldPlaylistRef
	activePlaylistsErr  error
	activePlaylistCalls int
	// seasons : saisons « découvertes » (C2). nil → season_catalog non rafraîchi
	// (comportement par défaut des tests historiques).
	seasons      []domain.WorldSeasonRef
	seasonsErr   error
	seasonsCalls int
	// seedSeason : dernière graine injectée par le cron (F11). "" = jamais appelée.
	seedSeason string
	// redirectSeason / redirectErr : repli de découverte SANS page-graine (LB-1.2,
	// header Location de la racine des classements). Non configuré = repli
	// INDISPONIBLE (erreur), jamais une saison vide : c'est le défaut des tests
	// historiques, qui doivent continuer à voir un cycle avorté.
	redirectSeason string
	redirectErr    error
	redirectCalls  int
}

func (s *stubScraper) FetchActiveSeasonByRedirect(_ context.Context) (string, error) {
	s.redirectCalls++
	if s.redirectErr != nil {
		return "", s.redirectErr
	}
	if s.redirectSeason == "" {
		return "", errors.New("stub: repli par redirection non configuré")
	}
	return s.redirectSeason, nil
}

// levelRecorder capture (niveau, message) des logs émis pendant un test, pour
// vérifier l'escalade WARN → ERROR sans dépendre d'un format de sortie.
type levelRecorder struct {
	mu      sync.Mutex
	entries []recordedLog
}

type recordedLog struct {
	level slog.Level
	msg   string
}

func (h *levelRecorder) Enabled(_ context.Context, l slog.Level) bool { return l >= slog.LevelWarn }
func (h *levelRecorder) Handle(_ context.Context, r slog.Record) error {
	h.mu.Lock()
	h.entries = append(h.entries, recordedLog{level: r.Level, msg: r.Message})
	h.mu.Unlock()
	return nil
}
func (h *levelRecorder) WithAttrs(_ []slog.Attr) slog.Handler { return h }
func (h *levelRecorder) WithGroup(_ string) slog.Handler      { return h }

// countAtLevel compte les messages capturés d'un niveau donné contenant `substr`.
func (h *levelRecorder) countAtLevel(level slog.Level, substr string) int {
	h.mu.Lock()
	defer h.mu.Unlock()
	n := 0
	for _, e := range h.entries {
		if e.level == level && strings.Contains(e.msg, substr) {
			n++
		}
	}
	return n
}

// captureLogs installe le recorder comme logger par défaut le temps du test.
func captureLogs(t *testing.T) *levelRecorder {
	t.Helper()
	rec := &levelRecorder{}
	old := slog.Default()
	slog.SetDefault(slog.New(rec))
	t.Cleanup(func() { slog.SetDefault(old) })
	return rec
}

func (s *stubScraper) SetSeedSeason(seasonID string) {
	s.seedSeason = seasonID
}

func (s *stubScraper) FetchActiveSeason(_ context.Context, refPlaylistID string) (string, error) {
	s.activeCalls++
	if err, ok := s.seasonErrForPlaylists[refPlaylistID]; ok {
		return "", err
	}
	if s.seasonErr != nil {
		return "", s.seasonErr
	}
	return s.season, nil
}

func (s *stubScraper) FetchActivePlaylists(_ context.Context, _ string) ([]domain.WorldPlaylistRef, error) {
	s.activePlaylistCalls++
	return s.activePlaylists, s.activePlaylistsErr
}

func (s *stubScraper) FetchSeasons(_ context.Context, _ string) ([]domain.WorldSeasonRef, error) {
	s.seasonsCalls++
	return s.seasons, s.seasonsErr
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

// TestWorldLeaderboardCron_PersistsSeasonCatalog (C2) vérifie que les saisons
// découvertes (nom d'Operation + FR) sont upsertées dans season_catalog dans la
// même fenêtre writer que le snapshot CSR.
func TestWorldLeaderboardCron_PersistsSeasonCatalog(t *testing.T) {
	provider, db := newSharedProviderForTest(t)
	scraper := &stubScraper{
		season:          "csrseason13-2",
		entries:         []domain.LeaderboardEntry{{Rank: 1, Gamertag: "Alpha", XUID: "2535000000000001", CSRValue: 1500}},
		activePlaylists: []domain.WorldPlaylistRef{{AssetID: "pl-1", DisplayName: "Ranked Arena"}},
		seasons: []domain.WorldSeasonRef{
			{SeasonID: "csrseason13-2", DisplayName: "Infinite", NameFR: "Infinite"},
			{SeasonID: "csrseason12-1", DisplayName: "Shadows", NameFR: "Ombres"},
		},
	}
	c := newTestCron(provider, scraper)
	c.RunOnce(context.Background())

	if scraper.seasonsCalls == 0 {
		t.Errorf("FetchSeasons jamais appelé (season_catalog non câblé)")
	}
	var count int
	if err := db.QueryRow(`SELECT COUNT(*) FROM season_catalog WHERE title_slug = ?`, "halo_infinite").Scan(&count); err != nil {
		t.Fatalf("count season_catalog: %v", err)
	}
	if count != 2 {
		t.Errorf("season_catalog = %d lignes, want 2", count)
	}
	var nameFR string
	if err := db.QueryRow(`SELECT name_fr FROM season_catalog WHERE season_id = ?`, "csrseason12-1").Scan(&nameFR); err != nil {
		t.Fatalf("select FR: %v", err)
	}
	if nameFR != "Ombres" {
		t.Errorf("name_fr csrseason12-1 = %q, want \"Ombres\"", nameFR)
	}
}

// TestWorldLeaderboardCron_SeedFromLatestPersistedSeason (F11/LB3) : la graine de
// découverte injectée au scraper est la DERNIÈRE saison persistée (rang NUMÉRIQUE max),
// pas la constante figée. Les snapshots pré-existants incluent csrseason6-1 (lexicalement
// > 13-2) pour piéger un MAX(season_id) naïf : la graine correcte reste csrseason13-2.
func TestWorldLeaderboardCron_SeedFromLatestPersistedSeason(t *testing.T) {
	provider, db := newSharedProviderForTest(t)
	ctx := context.Background()
	if _, err := duckdb.InsertWorldCSRSnapshot(ctx, db, "halo_infinite", []domain.LeaderboardEntry{
		{Season: "csrseason6-1", Playlist: "pl-a", Rank: 1, Gamertag: "Old", CSRValue: 1400, FetchedAt: time.Now().UTC()},
		{Season: "csrseason13-2", Playlist: "pl-a", Rank: 1, Gamertag: "New", CSRValue: 1500, FetchedAt: time.Now().UTC()},
	}); err != nil {
		t.Fatalf("seed snapshots: %v", err)
	}

	scraper := &stubScraper{
		season:  "csrseason13-2",
		entries: []domain.LeaderboardEntry{{Rank: 1, Gamertag: "Alpha", XUID: "2535000000000001", CSRValue: 1500}},
	}
	c := newTestCron(provider, scraper)
	c.RunOnce(ctx)

	if scraper.seedSeason != "csrseason13-2" {
		t.Errorf("graine injectée = %q, want csrseason13-2 (dernière saison persistée, rang max)", scraper.seedSeason)
	}
}

// TestWorldLeaderboardCron_SeedFallbackOnEmptyDB (F11/LB3) : sans aucun snapshot, le
// cron n'injecte PAS de graine — le scraper conserve sa constante par défaut (fallback).
func TestWorldLeaderboardCron_SeedFallbackOnEmptyDB(t *testing.T) {
	provider, _ := newSharedProviderForTest(t)
	scraper := &stubScraper{
		season:  "csrseason13-2",
		entries: []domain.LeaderboardEntry{{Rank: 1, Gamertag: "Alpha", XUID: "2535000000000001", CSRValue: 1500}},
	}
	c := newTestCron(provider, scraper)
	c.RunOnce(context.Background())

	if scraper.seedSeason != "" {
		t.Errorf("graine injectée = %q sur DB vide, want \"\" (fallback constante scraper)", scraper.seedSeason)
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

// TestWorldLeaderboardCron_SeasonDiscoveryFallsThrough404 : une playlist de
// référence non classée dans la saison-graine renvoie un 404 (cas nominal). Le cron
// NE DOIT PAS avorter le cycle dessus — il essaie la playlist candidate suivante
// jusqu'à trouver une page rendue. Régression du bruit prod B3.1 (ERROR quotidienne
// « découverte saison active échouée »).
func TestWorldLeaderboardCron_SeasonDiscoveryFallsThrough404(t *testing.T) {
	provider, db := newSharedProviderForTest(t)
	scraper := &stubScraper{
		season:  "csrseason13-2",
		entries: []domain.LeaderboardEntry{{Rank: 1, Gamertag: "Alpha", XUID: "2535000000000001", CSRValue: 1500}},
		activePlaylists: []domain.WorldPlaylistRef{
			{AssetID: "pl-new", DisplayName: "Ranked Nouvelle"}, // récente : 404 sur la saison-graine
			{AssetID: "pl-old", DisplayName: "Ranked Arène"},    // ancienne : rend la page
		},
		// Les statiques (pl-a, pl-b, essayées d'abord) et la playlist récente 404 ;
		// seule pl-old rend le menu de saisons.
		seasonErrForPlaylists: map[string]error{
			"pl-a":   halo.ErrLeaderboardPageNotFound,
			"pl-b":   halo.ErrLeaderboardPageNotFound,
			"pl-new": halo.ErrLeaderboardPageNotFound,
		},
	}
	c := newTestCron(provider, scraper)
	c.RunOnce(context.Background())

	// La saison a été découverte via pl-old → le cycle a scrapé + inséré (pas avorté).
	if got := countSnapshots(t, db, "csrseason13-2"); got == 0 {
		t.Fatalf("aucun snapshot : le cycle a avorté sur le 404 de la 1re playlist au lieu de basculer sur une candidate valide")
	}
	// Plusieurs playlists de référence ont été essayées avant le succès (repli).
	if scraper.activeCalls < 2 {
		t.Errorf("FetchActiveSeason appelé %d× — le cron n'a pas essayé de playlist de repli après le 404", scraper.activeCalls)
	}
}

// TestWorldLeaderboardCron_SeasonDiscoveryFallsBackToRedirect (LB-1.2) : quand la
// saison-graine a été RETIRÉE de Waypoint, TOUTES les candidates page-graine
// renvoient 404 et l'ancienne découverte était un point fixe mort (267 cycles à vide
// en prod). Le cron doit alors basculer sur le repli SANS graine (header Location de
// la racine des classements) et poursuivre le cycle NORMALEMENT : scrape + persist
// sous la saison découverte par ce repli.
func TestWorldLeaderboardCron_SeasonDiscoveryFallsBackToRedirect(t *testing.T) {
	provider, db := newSharedProviderForTest(t)
	scraper := &stubScraper{
		seasonErr:      halo.ErrLeaderboardPageNotFound, // saison-graine retirée du site
		redirectSeason: "csrseason13-3",                 // seule source encore vivante
		entries: []domain.LeaderboardEntry{
			{Rank: 1, Gamertag: "Alpha", XUID: "2535000000000001", CSRValue: 1800},
		},
	}
	c := newTestCron(provider, scraper) // statique = pl-a, pl-b
	c.RunOnce(context.Background())

	if scraper.redirectCalls == 0 {
		t.Fatal("repli par redirection jamais appelé — le cron reste bloqué sur la page-graine morte")
	}
	// Le cycle est allé jusqu'au bout : 2 playlists × 1 entrée, sous la saison du repli.
	if got := countSnapshots(t, db, "csrseason13-3"); got != 2 {
		t.Errorf("snapshots csrseason13-3 = %d, attendu 2 (le repli doit alimenter le cycle comme une découverte normale)", got)
	}
	// Découverte réussie (fût-ce par repli) → aucune série d'échecs en cours.
	if got := c.seasonDiscoveryFails.Load(); got != 0 {
		t.Errorf("série d'échecs = %d après un repli réussi, attendu 0", got)
	}
}

// TestWorldLeaderboardCron_SeasonDiscoveryEscalatesAfterStreak (LB-1.3) : une saison
// indécouvrable N'EST PAS auto-résolutive. Les 3 premiers cycles consécutifs restent
// en WARN (hoquet Waypoint plausible), le 4e passe en ERROR — sans quoi une panne
// durable reste invisible dans le bruit quotidien. Un succès remet la série à zéro.
func TestWorldLeaderboardCron_SeasonDiscoveryEscalatesAfterStreak(t *testing.T) {
	provider, _ := newSharedProviderForTest(t)
	scraper := &stubScraper{
		seasonErr:   halo.ErrLeaderboardPageNotFound,
		redirectErr: errors.New("waypoint 500"), // les DEUX chemins sont morts
	}
	c := newTestCron(provider, scraper)
	rec := captureLogs(t)
	ctx := context.Background()

	const msgFragment = "saison active indécouvrable"
	for i := 1; i <= 4; i++ {
		c.RunOnce(ctx)
		if got := c.seasonDiscoveryFails.Load(); got != int64(i) {
			t.Fatalf("après %d cycles en échec : série = %d, attendu %d", i, got, i)
		}
	}
	if got := observability.LoadCounter(seasonDiscoveryStreakMetric); got != 4 {
		t.Errorf("jauge expvar %s = %d, attendu 4", seasonDiscoveryStreakMetric, got)
	}
	// 3 WARN tolérés, puis escalade au 4e cycle.
	if got := rec.countAtLevel(slog.LevelWarn, msgFragment); got != maxSilentSeasonDiscoveryFails {
		t.Errorf("WARN de découverte = %d, attendu %d", got, maxSilentSeasonDiscoveryFails)
	}
	if got := rec.countAtLevel(slog.LevelError, msgFragment); got != 1 {
		t.Errorf("ERROR de découverte = %d, attendu 1 (escalade au-delà de %d cycles consécutifs)",
			got, maxSilentSeasonDiscoveryFails)
	}

	// Retour à la normale : la découverte repasse, la série est remise à zéro.
	scraper.seasonErr = nil
	scraper.season = "csrseason13-3"
	scraper.entries = []domain.LeaderboardEntry{{Rank: 1, Gamertag: "Alpha", XUID: "2535000000000001", CSRValue: 1500}}
	c.RunOnce(ctx)

	if got := c.seasonDiscoveryFails.Load(); got != 0 {
		t.Errorf("série d'échecs = %d après un cycle réussi, attendu 0 (reset)", got)
	}
	if got := observability.LoadCounter(seasonDiscoveryStreakMetric); got != 0 {
		t.Errorf("jauge expvar %s = %d après un cycle réussi, attendu 0", seasonDiscoveryStreakMetric, got)
	}
}

// TestWorldLeaderboardCron_WarnsWhenEnricherMissing (LB-1.4) : sans enricher câblé,
// le cycle reste scrape-only — dégradation de wiring qui ne doit PLUS être
// silencieuse (colonnes détaillées vides côté UI sans trace dans les logs).
func TestWorldLeaderboardCron_WarnsWhenEnricherMissing(t *testing.T) {
	provider, _ := newSharedProviderForTest(t)
	scraper := &stubScraper{
		season:  "csrseason13-3",
		entries: []domain.LeaderboardEntry{{Rank: 1, Gamertag: "Alpha", XUID: "2535000000000001", CSRValue: 1500}},
	}
	c := newTestCron(provider, scraper) // enricher nil (wiring absent)
	rec := captureLogs(t)

	c.RunOnce(context.Background())

	if got := rec.countAtLevel(slog.LevelWarn, "enrichissement inactif"); got != 1 {
		t.Errorf("WARN « enrichissement inactif » = %d, attendu 1 (retour silencieux ?)", got)
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
