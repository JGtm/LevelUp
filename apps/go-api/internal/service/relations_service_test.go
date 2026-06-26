package service

import (
	"context"
	"errors"
	"testing"
	"time"

	"levelup/go-api/internal/domain"
)

type mockRelationsRepo struct {
	rows []domain.RelationRawRow
	err  error
}

func (m *mockRelationsRepo) GetRelations(_ context.Context) ([]domain.RelationRawRow, error) {
	return m.rows, m.err
}

func fixedNow() func() time.Time {
	t := time.Date(2026, 6, 26, 0, 0, 0, 0, time.UTC)
	return func() time.Time { return t }
}

func TestGetRelationsPage_Empty(t *testing.T) {
	svc := NewRelationsService(&mockRelationsRepo{}).withNow(fixedNow())
	page, err := svc.GetRelationsPage(context.Background())
	if err != nil {
		t.Fatalf("unexpected err: %v", err)
	}
	if page.Overview.DistinctPlayers != 0 {
		t.Fatalf("distinct=%d want 0", page.Overview.DistinctPlayers)
	}
	if page.Relations == nil {
		t.Fatal("relations must be non-nil empty slice, not null")
	}
	if len(page.Relations) != 0 {
		t.Fatalf("relations len=%d want 0", len(page.Relations))
	}
	if page.Overview.TopAlly != nil || page.Overview.TopNemesis != nil {
		t.Fatal("top refs must be nil on empty")
	}
}

func TestGetRelationsPage_Error(t *testing.T) {
	svc := NewRelationsService(&mockRelationsRepo{err: errors.New("boom")})
	if _, err := svc.GetRelationsPage(context.Background()); err == nil {
		t.Fatal("expected error propagation")
	}
}

func TestGetRelationsPage_Enriched(t *testing.T) {
	now := time.Date(2026, 6, 26, 0, 0, 0, 0, time.UTC)
	old := now.AddDate(0, -8, 0)
	rows := []domain.RelationRawRow{
		{
			XUID: "x1", Gamertag: "Ally", TotalMatches: 15,
			TeammateCount: 15, TeammateWins: 11, TeammateLosses: 4,
			KillsDealt: 10, DeathsSuffered: 5, FirstSeen: old, LastSeen: now,
		},
		{
			XUID: "x2", Gamertag: "Nemesis", TotalMatches: 12,
			EnemyCount: 12, EnemyWins: 3, EnemyLosses: 9,
			KillsDealt: 4, DeathsSuffered: 20, FirstSeen: old, LastSeen: now,
		},
		{
			XUID: "x3", Gamertag: "Mix", TotalMatches: 4,
			TeammateCount: 2, EnemyCount: 2, FirstSeen: now, LastSeen: now,
		},
	}
	svc := NewRelationsService(&mockRelationsRepo{rows: rows}).withNow(func() time.Time { return now })
	page, err := svc.GetRelationsPage(context.Background())
	if err != nil {
		t.Fatalf("unexpected err: %v", err)
	}

	if page.Overview.DistinctPlayers != 3 {
		t.Fatalf("distinct=%d want 3", page.Overview.DistinctPlayers)
	}
	if page.Overview.AlliesCount != 2 { // Ally + Mix
		t.Fatalf("allies=%d want 2", page.Overview.AlliesCount)
	}
	if page.Overview.RivalsCount != 2 { // Nemesis + Mix
		t.Fatalf("rivals=%d want 2", page.Overview.RivalsCount)
	}

	// top_ally : Ally (teammate 15 >= 8, win rate 11/15 ~0.733)
	if page.Overview.TopAlly == nil || page.Overview.TopAlly.Gamertag != "Ally" {
		t.Fatalf("top ally=%v want Ally", page.Overview.TopAlly)
	}
	// top_nemesis : Nemesis (enemy 12 >= 8, win rate 3/12 = 0.25)
	if page.Overview.TopNemesis == nil || page.Overview.TopNemesis.Gamertag != "Nemesis" {
		t.Fatalf("top nemesis=%v want Nemesis", page.Overview.TopNemesis)
	}

	// Categories
	byGT := map[string]domain.RelationInsight{}
	for _, r := range page.Relations {
		byGT[r.Gamertag] = r
	}
	if byGT["Ally"].Category != "ally" {
		t.Fatalf("Ally category=%q", byGT["Ally"].Category)
	}
	if byGT["Nemesis"].Category != "enemy" {
		t.Fatalf("Nemesis category=%q", byGT["Nemesis"].Category)
	}
	if byGT["Mix"].Category != "mixed" {
		t.Fatalf("Mix category=%q", byGT["Mix"].Category)
	}

	// Win rates
	if byGT["Ally"].TeammateWinRate == nil || *byGT["Ally"].TeammateWinRate < 0.73 {
		t.Fatalf("Ally teammate win rate=%v", byGT["Ally"].TeammateWinRate)
	}
	if byGT["Ally"].EnemyWinRate != nil {
		t.Fatal("Ally enemy win rate must be nil")
	}
	// Duel ratio Ally = 10/5 = 2.0
	if byGT["Ally"].DuelRatio == nil || *byGT["Ally"].DuelRatio != 2.0 {
		t.Fatalf("Ally duel ratio=%v want 2.0", byGT["Ally"].DuelRatio)
	}
	// Mix duel ratio nil (0 kills 0 deaths)
	if byGT["Mix"].DuelRatio != nil {
		t.Fatalf("Mix duel ratio=%v want nil", byGT["Mix"].DuelRatio)
	}
	// first_seen_at / last_seen_at present
	if byGT["Ally"].FirstSeenAt == nil || byGT["Ally"].LastSeenAt == nil {
		t.Fatal("Ally timestamps must be set")
	}
}
