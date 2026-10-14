// seasons_catalog_reader_test.go — l'échec du fetch live est celui d'UN lecteur (D5 du lot
// aj14-d, 2026-10-14) : il ne prive pas les autres du catalogue, et le journal ne dit que les
// changements d'état.
package service

import (
	"bytes"
	"context"
	"errors"
	"log/slog"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"levelup/go-api/internal/ctxkeys"
	"levelup/go-api/internal/domain"
)

// readerSeasonProvider refuse le lecteur « périmé » (jeton rejeté) et sert les autres.
type readerSeasonProvider struct {
	calls  atomic.Int32
	stale  atomic.Bool // vrai : le lecteur « périmé » est rejeté
	result []domain.SeasonCalendar
}

func (p *readerSeasonProvider) FetchSeasonCalendar(ctx context.Context, _ string) ([]domain.SeasonCalendar, []byte, error) {
	p.calls.Add(1)
	if ctxkeys.HaloXUID(ctx) == "stale" && p.stale.Load() {
		return nil, nil, errors.New("HTTP 401")
	}
	return p.result, nil, nil
}

func seasonsCtxFor(xuid string) context.Context {
	return ctxkeys.WithHaloAuth(context.Background(), &domain.HaloTokens{SpartanToken: "s"}, xuid)
}

func nouvelleSaison() []domain.SeasonCalendar {
	return []domain.SeasonCalendar{dbSeason("season14", "Skyfall", time.Date(2026, 4, 15, 0, 0, 0, 0, time.UTC), nil)}
}

// TestSeasonsCatalog_ReaderFailureDoesNotBlockOthers : le jeton d'un lecteur est rejeté ; un autre
// lecteur, arrivé tout de suite après, retente et sert la saison fetchée — il n'attend pas 30 min.
func TestSeasonsCatalog_ReaderFailureDoesNotBlockOthers(t *testing.T) {
	repo := &syncMetadataRepo{}
	prov := &readerSeasonProvider{result: nouvelleSaison()}
	prov.stale.Store(true)
	clock := newTestClock()
	cat := NewSeasonsCatalog(makeAssetSetWithSeasons(t), repo, prov, nil)
	cat.now = clock.now

	if got := cat.Load(seasonsCtxFor("stale"), "halo_infinite"); len(got) != 2 {
		t.Fatalf("lecteur périmé : %d entrées, want 2 (repli TOML)", len(got))
	}
	clock.advance(time.Minute)
	if got := cat.Load(seasonsCtxFor("sain"), "halo_infinite"); len(got) != 3 {
		t.Fatalf("lecteur sain une minute après : %d entrées, want 3 (saison fetchée)", len(got))
	}
	if prov.calls.Load() != 2 {
		t.Errorf("appels live = %d, want 2 (un par lecteur)", prov.calls.Load())
	}
	// Le catalogue sain prime sur le repli du lecteur périmé.
	if got := cat.Load(seasonsCtxFor("stale"), "halo_infinite"); len(got) != 3 {
		t.Errorf("lecteur périmé après un succès d'un autre : %d entrées, want 3", len(got))
	}
}

// TestSeasonsCatalog_ReaderWaitsOwnBackoff : le lecteur en échec ne rappelle pas Waypoint pendant
// son attente, même quand un autre lecteur a échoué avant lui sur le même titre.
func TestSeasonsCatalog_ReaderWaitsOwnBackoff(t *testing.T) {
	prov := &readerSeasonProvider{result: nouvelleSaison()}
	prov.stale.Store(true)
	clock := newTestClock()
	cat := NewSeasonsCatalog(makeAssetSetWithSeasons(t), &syncMetadataRepo{}, prov, nil)
	cat.now = clock.now
	ctx := seasonsCtxFor("stale")

	cat.Load(ctx, "halo_infinite")
	clock.advance(29 * time.Minute)
	cat.Load(ctx, "halo_infinite")
	if prov.calls.Load() != 1 {
		t.Errorf("appels live pendant l'attente = %d, want 1", prov.calls.Load())
	}
	clock.advance(2 * time.Minute)
	cat.Load(ctx, "halo_infinite")
	if prov.calls.Load() != 2 {
		t.Errorf("appels live après l'attente = %d, want 2", prov.calls.Load())
	}
}

// TestSeasonsCatalog_FailureLogsOnStateChange : WARN au passage sain -> en échec, plus rien de
// niveau WARN/ERROR tant que le lecteur y reste, INFO au retour, puis WARN de nouveau.
func TestSeasonsCatalog_FailureLogsOnStateChange(t *testing.T) {
	var buf bytes.Buffer
	logger := slog.New(slog.NewTextHandler(&buf, &slog.HandlerOptions{Level: slog.LevelDebug}))
	prov := &readerSeasonProvider{result: nouvelleSaison()}
	prov.stale.Store(true)
	clock := newTestClock()
	repo := &syncMetadataRepo{}
	cat := NewSeasonsCatalog(makeAssetSetWithSeasons(t), repo, prov, logger)
	cat.now = clock.now
	ctx := seasonsCtxFor("stale")
	compte := func(niveau string) int {
		return strings.Count(buf.String(), "level="+niveau+" msg=\"seasons_catalog: fetch live")
	}

	cat.Load(ctx, "halo_infinite")
	clock.advance(31 * time.Minute)
	cat.Load(ctx, "halo_infinite") // toujours en échec : DEBUG seulement
	clock.advance(31 * time.Minute)
	cat.Load(ctx, "halo_infinite")
	if compte("WARN") != 1 || compte("ERROR") != 0 || compte("DEBUG") != 2 {
		t.Fatalf("après trois échecs : WARN=%d ERROR=%d DEBUG=%d, want 1, 0, 2\n%s", compte("WARN"), compte("ERROR"), compte("DEBUG"), buf.String())
	}

	prov.stale.Store(false)
	clock.advance(31 * time.Minute)
	cat.Load(ctx, "halo_infinite")
	if !strings.Contains(buf.String(), "fetch live retabli") {
		t.Errorf("retour à la normale non journalisé :\n%s", buf.String())
	}

	repo.mu.Lock()
	repo.rows = nil // base de nouveau vide : le fetch live est de nouveau tenté
	repo.mu.Unlock()
	prov.stale.Store(true)
	clock.advance(2 * time.Hour)
	cat.Load(ctx, "halo_infinite")
	if compte("WARN") != 2 {
		t.Errorf("nouvel échec après retour à la normale : WARN=%d, want 2 (nouveau changement d'état)", compte("WARN"))
	}
}
