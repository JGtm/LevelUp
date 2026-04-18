package service

import (
	"context"
	"errors"
	"testing"
	"time"

	"levelup/go-api/internal/domain"
)

// --- mock ---

type mockExplorerRepo struct {
	xuid     string
	xuidErr  error
	matches  []domain.CommonMatchRaw
	matchErr error
}

func (m *mockExplorerRepo) ResolveXUIDByGamertag(_ context.Context, _ string) (string, error) {
	return m.xuid, m.xuidErr
}
func (m *mockExplorerRepo) GetCommonMatches(_ context.Context, _, _ string) ([]domain.CommonMatchRaw, error) {
	return m.matches, m.matchErr
}

// --- tests ---

func TestExplorerService_GetCommonMatches_OK(t *testing.T) {
	now := time.Now()
	tid1, tid2 := 0, 0
	repo := &mockExplorerRepo{
		xuid: "other-xuid",
		matches: []domain.CommonMatchRaw{
			{
				MatchID:       "m1",
				StartTime:     now,
				MapUI:         "Aquarius",
				ModeUI:        "Slayer",
				Player1TeamID: &tid1,
				Player2TeamID: &tid2,
			},
		},
	}
	svc := NewExplorerService(repo, "my-xuid")

	resp, err := svc.GetCommonMatches(context.Background(), "OtherPlayer", 10)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.TargetGamertag != "OtherPlayer" {
		t.Errorf("TargetGamertag = %q, want OtherPlayer", resp.TargetGamertag)
	}
	if resp.TargetXUID != "other-xuid" {
		t.Errorf("TargetXUID = %q, want other-xuid", resp.TargetXUID)
	}
	if resp.Total != 1 {
		t.Errorf("Total = %d, want 1", resp.Total)
	}
	if !resp.CommonMatches[0].WereTeammates {
		t.Error("expected WereTeammates = true (same team ID)")
	}
}

func TestExplorerService_GetCommonMatches_Empty(t *testing.T) {
	repo := &mockExplorerRepo{
		xuid:    "other-xuid",
		matches: []domain.CommonMatchRaw{},
	}
	svc := NewExplorerService(repo, "my-xuid")

	resp, err := svc.GetCommonMatches(context.Background(), "Player", 10)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.Total != 0 {
		t.Errorf("Total = %d, want 0", resp.Total)
	}
}

func TestExplorerService_GetCommonMatches_ResolveError(t *testing.T) {
	repo := &mockExplorerRepo{xuidErr: errors.New("not found")}
	svc := NewExplorerService(repo, "my-xuid")

	_, err := svc.GetCommonMatches(context.Background(), "Unknown", 10)
	if err == nil {
		t.Error("expected error")
	}
}

func TestExplorerService_GetCommonMatches_QueryError(t *testing.T) {
	repo := &mockExplorerRepo{xuid: "other", matchErr: errors.New("db fail")}
	svc := NewExplorerService(repo, "my-xuid")

	_, err := svc.GetCommonMatches(context.Background(), "Player", 10)
	if err == nil {
		t.Error("expected error")
	}
}

// TestExplorerService_GetCommonMatches_WithStats — Sprint 52 C6.
// Vérifie que kills/deaths/kda réels depuis match_participants sont bien propagés.
func TestExplorerService_GetCommonMatches_WithStats(t *testing.T) {
	now := time.Now()
	tid1, tid2 := 0, 1 // équipes différentes
	repo := &mockExplorerRepo{
		xuid: "enemy-xuid",
		matches: []domain.CommonMatchRaw{
			{
				MatchID:        "golden-match-1",
				StartTime:      now,
				MapUI:          "Recharge",
				ModeUI:         "Slayer",
				Player1TeamID:  &tid1,
				Player2TeamID:  &tid2,
				Player1Outcome: 2, // WIN
				Player1Kills:   18,
				Player1Deaths:  7,
				Player1KDA:     2.57,
			},
		},
	}
	svc := NewExplorerService(repo, "my-xuid")

	resp, err := svc.GetCommonMatches(context.Background(), "EnemyPlayer", 10)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.Total != 1 {
		t.Fatalf("Total = %d, want 1", resp.Total)
	}
	m := resp.CommonMatches[0]

	if m.Kills != 18 {
		t.Errorf("Kills = %d, want 18", m.Kills)
	}
	if m.Deaths != 7 {
		t.Errorf("Deaths = %d, want 7", m.Deaths)
	}
	// KDA doit être non nul (valeur réelle)
	if m.KDA == 0.0 {
		t.Error("KDA = 0.0 : kills/deaths/kda ne sont pas propagés depuis match_participants")
	}
	if m.WereTeammates {
		t.Error("WereTeammates attendu false (équipes différentes)")
	}
	if m.PlayerOutcome != 2 {
		t.Errorf("PlayerOutcome = %d, want 2 (WIN)", m.PlayerOutcome)
	}
}

func TestExplorerService_GetCommonMatches_DifferentTeams(t *testing.T) {
	now := time.Now()
	tid1, tid2 := 0, 1
	repo := &mockExplorerRepo{
		xuid: "other",
		matches: []domain.CommonMatchRaw{
			{MatchID: "m1", StartTime: now, Player1TeamID: &tid1, Player2TeamID: &tid2},
		},
	}
	svc := NewExplorerService(repo, "my-xuid")

	resp, err := svc.GetCommonMatches(context.Background(), "Enemy", 10)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.CommonMatches[0].WereTeammates {
		t.Error("expected WereTeammates = false (different teams)")
	}
}
