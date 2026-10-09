package service

import (
	"context"
	"errors"
	"testing"
	"time"

	"levelup/go-api/internal/domain"
	"levelup/go-api/internal/games"
	"levelup/go-api/internal/games/canonical"
	"levelup/go-api/internal/port"
	"levelup/go-api/internal/sync"
	"levelup/go-api/internal/util/pointers"
)

const trendsTestSlug = "halo_infinite"

// fakeTrendsMatches est un dépôt de matchs canoniques en mémoire.
type fakeTrendsMatches struct {
	rows []canonical.PlayerMatchRow
	err  error
}

func (f *fakeTrendsMatches) LoadPlayerMatches(_ context.Context, _, _ string, _ port.PlayerMatchFilters) ([]canonical.PlayerMatchRow, error) {
	return f.rows, f.err
}

func (f *fakeTrendsMatches) InvalidatePlayer(_, _ string) {}

var trendsTestNow = time.Date(2026, 9, 27, 2, 0, 0, 0, time.UTC)

func trendsRow(id string, hoursAgo int, ranked, withFriends bool) canonical.PlayerMatchRow {
	return canonical.PlayerMatchRow{
		Summary: canonical.MatchSummary{
			MatchID:      id,
			StartedAtUTC: trendsTestNow.Add(-time.Duration(hoursAgo) * time.Hour),
			IsRanked:     pointers.Ptr(ranked),
		},
		Self:       canonical.MatchParticipant{Outcome: canonical.OutcomeWin, Kills: pointers.Ptr(10), Deaths: pointers.Ptr(5)},
		Enrichment: canonical.PlayerMatchEnrichment{IsWithFriends: withFriends},
	}
}

func newTrendsTestService(repo port.PlayerMatchesRepository) *TrendsService {
	return NewTrendsService(repo, trendsTestSlug, "TestGT").
		WithClock(func() time.Time { return trendsTestNow })
}

// matchCount365 lit le nombre de matchs de l'horizon 365 jours (indicateur match_count).
func matchCount365(t *testing.T, resp domain.TrendsPageResponse) int {
	t.Helper()
	for _, ind := range resp.Indicators {
		if ind.Key != domain.TrendsKeyMatchCount {
			continue
		}
		for _, h := range ind.Horizons {
			if h.Days == 365 {
				return h.Matches
			}
		}
	}
	t.Fatal("indicateur match_count / horizon 365 introuvable")
	return 0
}

func TestTrendsService_SoloKeepsOnlyMatchesWithoutFriends(t *testing.T) {
	repo := &fakeTrendsMatches{rows: []canonical.PlayerMatchRow{
		trendsRow("a", 1, false, false),
		trendsRow("b", 2, false, true),
		trendsRow("c", 3, false, false),
	}}
	resp, err := newTrendsTestService(repo).GetPage(context.Background(), domain.TrendsQueryRequest{})
	if err != nil {
		t.Fatalf("GetPage : %v", err)
	}
	if got := matchCount365(t, resp); got != 2 {
		t.Fatalf("matchs retenus = %d, attendu 2", got)
	}
	if resp.View != domain.TrendsViewSolo {
		t.Fatalf("View = %q", resp.View)
	}
}

func TestTrendsService_GameTypeFilterForwarded(t *testing.T) {
	ranked := sync.GetPerformanceChain(trendsTestSlug, "", true, false)
	repo := &fakeTrendsMatches{rows: []canonical.PlayerMatchRow{
		trendsRow("a", 1, true, false),
		trendsRow("b", 2, false, false),
		trendsRow("c", 3, false, false),
	}}
	resp, err := newTrendsTestService(repo).GetPage(context.Background(), domain.TrendsQueryRequest{GameType: ranked})
	if err != nil {
		t.Fatalf("GetPage : %v", err)
	}
	if resp.GameType != ranked {
		t.Fatalf("GameType = %q, attendu %q", resp.GameType, ranked)
	}
	if got := matchCount365(t, resp); got != 1 {
		t.Fatalf("matchs du type = %d, attendu 1", got)
	}
}

func TestTrendsService_InjectedClockLocationAndCapabilities(t *testing.T) {
	loc, err := time.LoadLocation("Europe/Paris")
	if err != nil {
		t.Skipf("fuseau Europe/Paris indisponible : %v", err)
	}
	caps := domain.TrendsCapabilities{MMR: true, CSR: true, Objectives: true}
	resp, err := newTrendsTestService(&fakeTrendsMatches{}).
		WithLocation(loc).WithCapabilities(caps).
		GetPage(context.Background(), domain.TrendsQueryRequest{})
	if err != nil {
		t.Fatalf("GetPage : %v", err)
	}
	if !resp.AsOf.Equal(trendsTestNow) {
		t.Fatalf("AsOf = %v, attendu %v", resp.AsOf, trendsTestNow)
	}
	if resp.Timezone != "Europe/Paris" {
		t.Fatalf("Timezone = %q", resp.Timezone)
	}
	if resp.Capabilities != caps {
		t.Fatalf("Capabilities = %+v, attendu %+v", resp.Capabilities, caps)
	}
}

func TestTrendsService_RepoErrorPropagated(t *testing.T) {
	boom := errors.New("lecture impossible")
	_, err := newTrendsTestService(&fakeTrendsMatches{err: boom}).GetPage(context.Background(), domain.TrendsQueryRequest{})
	if !errors.Is(err, boom) {
		t.Fatalf("err = %v, attendu l'erreur du dépôt", err)
	}
	_, err = newTrendsTestService(&fakeTrendsMatches{err: games.ErrCapabilityNotSupported}).GetPage(context.Background(), domain.TrendsQueryRequest{})
	if !errors.Is(err, games.ErrCapabilityNotSupported) {
		t.Fatalf("err = %v, ErrCapabilityNotSupported non détectable", err)
	}
}

func TestTrendsService_NoMatchesGivesValidEmptyResponse(t *testing.T) {
	resp, err := newTrendsTestService(&fakeTrendsMatches{}).GetPage(context.Background(), domain.TrendsQueryRequest{})
	if err != nil {
		t.Fatalf("GetPage : %v", err)
	}
	if resp.GameTypes == nil || resp.Indicators == nil || resp.Calendar == nil ||
		resp.WinLoss == nil || resp.Medals == nil || resp.Months == nil {
		t.Fatalf("tableau nil dans la réponse : %+v", resp)
	}
	if len(resp.GameTypes) != 0 || len(resp.Calendar) != 0 || len(resp.Medals) != 0 {
		t.Fatalf("tableaux non vides pour un joueur sans match : %+v", resp)
	}
	if len(resp.Months) != 12 {
		t.Fatalf("Months = %d, attendu 12", len(resp.Months))
	}
}
