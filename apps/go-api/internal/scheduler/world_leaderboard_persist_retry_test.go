//go:build cgo
// +build cgo

// world_leaderboard_persist_retry_test.go — lot B1 (2026-09-26) : le classement mondial
// scrapé ne se perd plus sur une vidange expirée du provider (nouvelle tentative bornée,
// scrape gardé en mémoire), et le premier tir attend le délai de boot.
package scheduler

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"testing"
	"time"

	"levelup/go-api/internal/domain"
	"levelup/go-api/internal/observability"
	"levelup/go-api/internal/platform/duckdb/sharedprovider"
)

// flakyWriterProvider enveloppe un provider réel : l'appel i d'AcquireWriter rend
// errs[i] quand elle est non nil, sinon délègue (writer réel, INSERT réels).
type flakyWriterProvider struct {
	sharedprovider.Provider
	errs  []error
	calls int
}

func (p *flakyWriterProvider) AcquireWriter(ctx context.Context) (*sharedprovider.WriterHandle, error) {
	i := p.calls
	p.calls++
	if i < len(p.errs) && p.errs[i] != nil {
		return nil, p.errs[i]
	}
	return p.Provider.AcquireWriter(ctx)
}

// drainTimeoutErr reproduit l'erreur du provider réel sur une vidange expirée.
func drainTimeoutErr() error {
	return fmt.Errorf("sharedprovider: drain inflight readers: %w: %w",
		sharedprovider.ErrDrainTimeout, context.DeadlineExceeded)
}

// recordWaits remplace l'attente entre deux tentatives par un enregistreur : aucune
// suite ne dort pour de vrai.
func recordWaits(t *testing.T) *[]time.Duration {
	t.Helper()
	waits := []time.Duration{}
	orig := worldLeaderboardSleep
	worldLeaderboardSleep = func(ctx context.Context, d time.Duration) error {
		waits = append(waits, d)
		return ctx.Err()
	}
	t.Cleanup(func() { worldLeaderboardSleep = orig })
	return &waits
}

// newFlakyCron monte un cron (2 playlists × 1 entrée) sur un provider dont les
// premiers AcquireWriter rendent errs.
func newFlakyCron(t *testing.T, errs ...error) (*WorldLeaderboardCron, *flakyWriterProvider, *stubScraper) {
	t.Helper()
	observability.ResetCronStatus()
	t.Cleanup(observability.ResetCronStatus)
	inner, _ := newSharedProviderForTest(t)
	provider := &flakyWriterProvider{Provider: inner, errs: errs}
	scraper := &stubScraper{
		season: "csrseason13-2",
		entries: []domain.LeaderboardEntry{{Rank: 1, Gamertag: "Alpha", XUID: "2535000000000001",
			CSRValue: 1500, FetchedAt: time.Now().UTC()}},
	}
	return newTestCron(provider, scraper), provider, scraper
}

// TestWorldLeaderboardCron_RetriesPersistOnDrainTimeout : deux vidanges expirées puis
// succès → le lot scrapé UNE fois est inséré à la 3e tentative, après deux attentes de
// 30 s ; le cycle est rapporté en succès.
func TestWorldLeaderboardCron_RetriesPersistOnDrainTimeout(t *testing.T) {
	c, provider, scraper := newFlakyCron(t, drainTimeoutErr(), drainTimeoutErr())
	inner := provider.Provider
	waits := recordWaits(t)
	rec := captureLogs(t)

	c.RunOnce(context.Background())

	db, release, err := inner.Get(context.Background())
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	defer release()
	var rows, batches int
	if err := db.QueryRow(`SELECT COUNT(*), COUNT(DISTINCT fetched_at) FROM world_csr_leaderboard_snapshots
		WHERE season_id = ?`, "csrseason13-2").Scan(&rows, &batches); err != nil {
		t.Fatalf("count: %v", err)
	}
	if rows != 2 || batches != 1 {
		t.Errorf("snapshots = %d lignes / %d lot(s), attendu 2 / 1 (le scrape gardé en mémoire est persisté)", rows, batches)
	}
	if scraper.fetchCalls != 2 {
		t.Errorf("FetchCSRLeaderboard = %d, attendu 2 (un seul scrape, pas de re-scrape par tentative)", scraper.fetchCalls)
	}
	if provider.calls != 3 {
		t.Errorf("AcquireWriter = %d appels, attendu 3", provider.calls)
	}
	if len(*waits) != 2 || (*waits)[0] != worldLeaderboardPersistRetryDelay || (*waits)[1] != worldLeaderboardPersistRetryDelay {
		t.Errorf("attentes = %v, attendu 2 × %v", *waits, worldLeaderboardPersistRetryDelay)
	}
	if got := rec.countAtLevel(slog.LevelWarn, "nouvelle tentative"); got != 2 {
		t.Errorf("WARN de nouvelle tentative = %d, attendu 2", got)
	}
	if got := cronRecord(t, "world_leaderboard"); got.LastError != "" {
		t.Errorf("cycle rapporté en échec (%q) alors que la 3e tentative a réussi", got.LastError)
	}
}

// TestWorldLeaderboardCron_NoRetryOnNonTransientWriterError : une erreur autre qu'une
// vidange expirée (ex. dblease) n'est PAS retentée : un seul appel, aucune attente.
func TestWorldLeaderboardCron_NoRetryOnNonTransientWriterError(t *testing.T) {
	c, provider, _ := newFlakyCron(t, errors.New("dblease: verrou indisponible"))
	waits := recordWaits(t)

	c.RunOnce(context.Background())

	if provider.calls != 1 {
		t.Errorf("AcquireWriter = %d appels, attendu 1 (erreur non transitoire)", provider.calls)
	}
	if len(*waits) != 0 {
		t.Errorf("attentes = %v, attendu aucune", *waits)
	}
	if got := cronRecord(t, "world_leaderboard"); got.LastError == "" {
		t.Error("cycle rapporté en succès malgré l'échec du writer")
	}
}

// TestWorldLeaderboardCron_PersistFailsAfterThreeDrainTimeouts : trois vidanges expirées
// → trois tentatives au total, deux attentes, aucune ligne, UN seul ERROR final et
// l'erreur remonte à ReportCronRun.
func TestWorldLeaderboardCron_PersistFailsAfterThreeDrainTimeouts(t *testing.T) {
	c, provider, _ := newFlakyCron(t, drainTimeoutErr(), drainTimeoutErr(), drainTimeoutErr())
	inner := provider.Provider
	waits := recordWaits(t)
	rec := captureLogs(t)

	c.RunOnce(context.Background())

	if provider.calls != 3 {
		t.Errorf("AcquireWriter = %d appels, attendu 3 (bornage)", provider.calls)
	}
	if len(*waits) != 2 {
		t.Errorf("attentes = %v, attendu 2 (pas d'attente après la dernière tentative)", *waits)
	}
	db, release, err := inner.Get(context.Background())
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	defer release()
	if n := countSnapshots(t, db, "csrseason13-2"); n != 0 {
		t.Errorf("snapshots = %d, attendu 0", n)
	}
	if got := rec.countAtLevel(slog.LevelError, ""); got != 1 {
		t.Errorf("ERROR = %d, attendu 1 (échec final seulement)", got)
	}
	got := cronRecord(t, "world_leaderboard")
	if got.LastError == "" || got.ConsecutiveFailures != 1 {
		t.Errorf("statut cron = %+v, attendu un échec avec cause", got)
	}
}

// TestWorldLeaderboardCron_RetryWaitIsCancellable : un arrêt pendant l'attente met fin
// aux tentatives (pas de 2e acquisition) et l'échec remonte au statut du cron.
func TestWorldLeaderboardCron_RetryWaitIsCancellable(t *testing.T) {
	c, provider, _ := newFlakyCron(t, drainTimeoutErr(), drainTimeoutErr())
	orig := worldLeaderboardSleep
	worldLeaderboardSleep = func(context.Context, time.Duration) error { return context.Canceled }
	t.Cleanup(func() { worldLeaderboardSleep = orig })

	c.RunOnce(context.Background())

	if provider.calls != 1 {
		t.Errorf("AcquireWriter = %d appels, attendu 1 (attente interrompue)", provider.calls)
	}
	if got := cronRecord(t, "world_leaderboard"); got.LastError == "" {
		t.Error("cycle rapporté en succès malgré l'attente interrompue")
	}
}

// stubEnricher rend des stats fixes pour chaque saison enrichie.
type stubEnricher struct{}

func (stubEnricher) EnrichSeason(_ context.Context, season string, players []domain.WorldPlayerRef) ([]domain.WorldPlayerSeasonStats, []error) {
	out := make([]domain.WorldPlayerSeasonStats, 0, len(players))
	for _, p := range players {
		out = append(out, domain.WorldPlayerSeasonStats{Gamertag: p.Gamertag, SeasonID: season, MatchCount: 10})
	}
	return out, nil
}

// TestWorldLeaderboardCron_RetriesPersistStatsOnDrainTimeout : la persistance des stats
// enrichies passe par la même nouvelle tentative que celle du classement.
func TestWorldLeaderboardCron_RetriesPersistStatsOnDrainTimeout(t *testing.T) {
	// Appel 1 (classement) passe ; appel 2 (stats) expire ; appel 3 (stats) passe.
	c, provider, _ := newFlakyCron(t, nil, drainTimeoutErr())
	c.WithStatsEnricher(stubEnricher{})
	inner := provider.Provider
	waits := recordWaits(t)

	c.RunOnce(context.Background())

	if provider.calls != 3 || len(*waits) != 1 {
		t.Errorf("AcquireWriter = %d appels / %d attente(s), attendu 3 / 1", provider.calls, len(*waits))
	}
	db, release, err := inner.Get(context.Background())
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	defer release()
	var n int
	if err := db.QueryRow(`SELECT COUNT(*) FROM world_player_season_stats WHERE season_id = ?`,
		"csrseason13-2").Scan(&n); err != nil {
		t.Fatalf("count stats: %v", err)
	}
	if n != 1 {
		t.Errorf("stats persistées = %d, attendu 1 (Alpha)", n)
	}
}

// TestWorldLeaderboardCron_RunWaitsBootDelay : le premier tir attend bootDelay (défaut
// 2 min) ; un arrêt pendant ce délai ne lance aucun cycle ; à 0 le cycle part aussitôt.
func TestWorldLeaderboardCron_RunWaitsBootDelay(t *testing.T) {
	provider, _ := newSharedProviderForTest(t)
	scraper := &stubScraper{seasonErr: errors.New("hors ligne"), redirectErr: errors.New("hors ligne")}
	c := newTestCron(provider, scraper)
	if c.bootDelay != worldLeaderboardBootDelay || worldLeaderboardBootDelay != 2*time.Minute {
		t.Fatalf("bootDelay défaut = %v (constante %v), attendu 2m", c.bootDelay, worldLeaderboardBootDelay)
	}

	// Délai non écoulé : Run rend la main à l'arrêt sans avoir touché le scraper.
	c.bootDelay = time.Hour
	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()
	c.Run(ctx)
	if scraper.activePlaylistCalls != 0 || scraper.activeCalls != 0 {
		t.Fatalf("cycle lancé avant le délai de boot (playlists=%d, saison=%d)",
			scraper.activePlaylistCalls, scraper.activeCalls)
	}

	// Option de test à 0 : le cycle part aussitôt ; l'arrêt est demandé par le cycle lui-même.
	c.bootDelay = 0
	ctx2, cancel2 := context.WithCancel(context.Background())
	defer cancel2()
	c.scraper = &cancelOnDiscoveryScraper{stubScraper: scraper, cancel: cancel2}
	c.Run(ctx2)
	if scraper.activePlaylistCalls == 0 {
		t.Error("aucun cycle lancé avec bootDelay = 0")
	}
}

// cancelOnDiscoveryScraper arrête le contexte de Run dès que le cycle démarre.
type cancelOnDiscoveryScraper struct {
	*stubScraper
	cancel context.CancelFunc
}

func (s *cancelOnDiscoveryScraper) FetchActivePlaylists(ctx context.Context, ref string) ([]domain.WorldPlaylistRef, error) {
	s.cancel()
	return s.stubScraper.FetchActivePlaylists(ctx, ref)
}
