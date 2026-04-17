package service

import (
	"context"
	"errors"
	"testing"
	"time"

	"levelup/go-api/internal/domain"
)

// --- mock ---

type mockHomeRepo struct {
	matches    []domain.HomeMatchRow
	matchErr   error
	sessions   []domain.HomeSessionRow
	sessionErr error
	media      []domain.HomeMediaRow
	mediaErr   error
}

func (m *mockHomeRepo) LoadHomeMatches(_ context.Context) ([]domain.HomeMatchRow, error) {
	return m.matches, m.matchErr
}
func (m *mockHomeRepo) LoadHomeSessions(_ context.Context) ([]domain.HomeSessionRow, error) {
	return m.sessions, m.sessionErr
}
func (m *mockHomeRepo) LoadRecentMedia(_ context.Context, _ int) ([]domain.HomeMediaRow, error) {
	return m.media, m.mediaErr
}

// --- tests ---

func TestHomeService_GetHomePage_OK(t *testing.T) {
	now := time.Now()
	repo := &mockHomeRepo{
		matches: []domain.HomeMatchRow{
			{MatchID: "m1", StartTime: now, MapName: "Aquarius", PairName: "Slayer", Outcome: 2, Kills: 10, Deaths: 5, Assists: 3},
			{MatchID: "m2", StartTime: now.Add(-1 * time.Hour), MapName: "Streets", PairName: "CTF", Outcome: 3, Kills: 5, Deaths: 10, Assists: 1},
		},
		sessions: []domain.HomeSessionRow{
			{MatchID: "m1", SessionLabel: strPtr("Session 1"), StartTime: &now},
		},
		media: []domain.HomeMediaRow{
			{FileName: "clip1.mp4"},
		},
	}
	svc := NewHomeService(repo)

	resp, err := svc.GetHomePage(context.Background(), "TestGT")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp == nil {
		t.Fatal("expected non-nil response")
	}
}

func TestHomeService_GetHomePage_Empty(t *testing.T) {
	repo := &mockHomeRepo{
		matches:  []domain.HomeMatchRow{},
		sessions: []domain.HomeSessionRow{},
		media:    []domain.HomeMediaRow{},
	}
	svc := NewHomeService(repo)

	resp, err := svc.GetHomePage(context.Background(), "TestGT")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp == nil {
		t.Fatal("expected non-nil response even with empty data")
	}
}

func TestHomeService_GetHomePage_MatchesError(t *testing.T) {
	repo := &mockHomeRepo{matchErr: errors.New("fail")}
	svc := NewHomeService(repo)

	_, err := svc.GetHomePage(context.Background(), "GT")
	if err == nil {
		t.Error("expected error when matches fail")
	}
}

func TestHomeService_GetHomePage_SessionsError(t *testing.T) {
	repo := &mockHomeRepo{
		matches:    []domain.HomeMatchRow{{MatchID: "m1", Outcome: 2, StartTime: time.Now()}},
		sessionErr: errors.New("fail"),
	}
	svc := NewHomeService(repo)

	_, err := svc.GetHomePage(context.Background(), "GT")
	if err == nil {
		t.Error("expected error when sessions fail")
	}
}

func TestHomeService_GetHomePage_MediaGraceful(t *testing.T) {
	now := time.Now()
	repo := &mockHomeRepo{
		matches:  []domain.HomeMatchRow{{MatchID: "m1", Outcome: 2, Kills: 10, Deaths: 5, StartTime: now}},
		sessions: []domain.HomeSessionRow{},
		mediaErr: errors.New("media unavailable"),
	}
	svc := NewHomeService(repo)

	resp, err := svc.GetHomePage(context.Background(), "GT")
	if err != nil {
		t.Fatalf("expected graceful degradation on media error, got: %v", err)
	}
	if resp == nil {
		t.Fatal("expected non-nil response")
	}
}

func TestHomeService_GetBattlePass(t *testing.T) {
	repo := &mockHomeRepo{}
	svc := NewHomeService(repo)
	bp := svc.GetBattlePass(context.Background())
	// Default provider returns available=false
	if bp.Available {
		t.Error("expected Available=false from default provider")
	}
}

func TestHomeService_GetChallenges(t *testing.T) {
	repo := &mockHomeRepo{}
	svc := NewHomeService(repo)
	ch := svc.GetChallenges(context.Background())
	if ch.Available {
		t.Error("expected Available=false from default provider")
	}
}
