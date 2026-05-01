package service

import (
	"context"
	"errors"
	"fmt"
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
	kv       domain.KillerVictimAggregate
	kvErr    error
}

func (m *mockExplorerRepo) ResolveXUIDByGamertag(_ context.Context, _ string) (string, error) {
	return m.xuid, m.xuidErr
}
func (m *mockExplorerRepo) GetCommonMatches(_ context.Context, _, _ string) ([]domain.CommonMatchRaw, error) {
	return m.matches, m.matchErr
}
func (m *mockExplorerRepo) GetKillerVictimBetween(_ context.Context, _, _ string) (domain.KillerVictimAggregate, error) {
	return m.kv, m.kvErr
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

	resp, err := svc.GetCommonMatches(context.Background(), "OtherPlayer", 1)
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

	resp, err := svc.GetCommonMatches(context.Background(), "Player", 1)
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

	_, err := svc.GetCommonMatches(context.Background(), "Unknown", 1)
	if err == nil {
		t.Error("expected error")
	}
}

func TestExplorerService_GetCommonMatches_QueryError(t *testing.T) {
	repo := &mockExplorerRepo{xuid: "other", matchErr: errors.New("db fail")}
	svc := NewExplorerService(repo, "my-xuid")

	_, err := svc.GetCommonMatches(context.Background(), "Player", 1)
	if err == nil {
		t.Error("expected error")
	}
}

// TestExplorerService_GetCommonMatches_WithStats — vérifie kills/deaths/kda + OutcomeLabel.
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

	resp, err := svc.GetCommonMatches(context.Background(), "EnemyPlayer", 1)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.TotalCount != 1 {
		t.Fatalf("TotalCount = %d, want 1", resp.TotalCount)
	}
	m := resp.CommonMatches[0]
	if m.Kills != 18 {
		t.Errorf("Kills = %d, want 18", m.Kills)
	}
	if m.Deaths != 7 {
		t.Errorf("Deaths = %d, want 7", m.Deaths)
	}
	if m.KDA == 0.0 {
		t.Error("KDA = 0.0 : kills/deaths/kda ne sont pas propagés depuis match_participants")
	}
	if m.WereTeammates {
		t.Error("WereTeammates attendu false (équipes différentes)")
	}
	if m.PlayerOutcome != 2 {
		t.Errorf("PlayerOutcome = %d, want 2 (WIN)", m.PlayerOutcome)
	}
	if m.OutcomeLabel == "" {
		t.Error("OutcomeLabel vide — doit être résolu via outcomeLabel()")
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

	resp, err := svc.GetCommonMatches(context.Background(), "Enemy", 1)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.CommonMatches[0].WereTeammates {
		t.Error("expected WereTeammates = false (different teams)")
	}
}

// TestExplorerService_GetCommonMatches_Pagination vérifie la pagination à 20 éléments.
func TestExplorerService_GetCommonMatches_Pagination(t *testing.T) {
	now := time.Now()
	tid1, tid2 := 0, 0
	// 25 matchs — page 1 = 20, page 2 = 5
	matches := make([]domain.CommonMatchRaw, 25)
	for i := range matches {
		matches[i] = domain.CommonMatchRaw{
			MatchID:       fmt.Sprintf("m%d", i),
			StartTime:     now,
			Player1TeamID: &tid1,
			Player2TeamID: &tid2,
		}
	}
	repo := &mockExplorerRepo{xuid: "other", matches: matches}
	svc := NewExplorerService(repo, "my-xuid")

	p1, err := svc.GetCommonMatches(context.Background(), "Player", 1)
	if err != nil {
		t.Fatalf("page 1: %v", err)
	}
	if len(p1.CommonMatches) != 20 {
		t.Errorf("page 1 items = %d, want 20", len(p1.CommonMatches))
	}
	if p1.TotalCount != 25 {
		t.Errorf("TotalCount = %d, want 25", p1.TotalCount)
	}
	if p1.Page != 1 {
		t.Errorf("Page = %d, want 1", p1.Page)
	}

	p2, err := svc.GetCommonMatches(context.Background(), "Player", 2)
	if err != nil {
		t.Fatalf("page 2: %v", err)
	}
	if len(p2.CommonMatches) != 5 {
		t.Errorf("page 2 items = %d, want 5", len(p2.CommonMatches))
	}
}

// TestExplorerService_GetCommonMatches_AllyPlusBadge vérifie que le badge ally_plus
// est émis quand le win rate dépasse le seuil.
func TestExplorerService_GetCommonMatches_AllyPlusBadge(t *testing.T) {
	now := time.Now()
	tid := 0
	// 4 matchs en équipe, 3 victoires → winrate 0.75 > 0.70
	matches := []domain.CommonMatchRaw{
		{MatchID: "a1", StartTime: now, Player1TeamID: &tid, Player2TeamID: &tid, Player1Outcome: 2},
		{MatchID: "a2", StartTime: now, Player1TeamID: &tid, Player2TeamID: &tid, Player1Outcome: 2},
		{MatchID: "a3", StartTime: now, Player1TeamID: &tid, Player2TeamID: &tid, Player1Outcome: 2},
		{MatchID: "a4", StartTime: now, Player1TeamID: &tid, Player2TeamID: &tid, Player1Outcome: 3},
	}
	repo := &mockExplorerRepo{xuid: "ally-xuid", matches: matches}
	svc := NewExplorerService(repo, "my-xuid")

	resp, err := svc.GetCommonMatches(context.Background(), "AllyPlayer", 1)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	var hasAllyPlus bool
	for _, b := range resp.Badges {
		if b.Kind == "ally_plus" {
			hasAllyPlus = true
		}
	}
	if !hasAllyPlus {
		t.Error("badge ally_plus attendu (winrate 75% > 70%)")
	}
}
