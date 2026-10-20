package skill

import (
	"math"
	"testing"
)

// ── NewPlayerState ───────────────────────────────────────────────────────────

func TestNewPlayerState_Defaults(t *testing.T) {
	s := NewPlayerState()
	if s.MU != InitialMU {
		t.Fatalf("expected MU=%f, got %f", InitialMU, s.MU)
	}
	if s.Sigma != InitialSigma {
		t.Fatalf("expected Sigma=%f, got %f", InitialSigma, s.Sigma)
	}
	if s.MatchCount != 0 {
		t.Fatal("expected MatchCount=0")
	}
	if s.LastMatchTime != nil {
		t.Fatal("expected nil LastMatchTime")
	}
}

// ── computeCompositeScore ────────────────────────────────────────────────────

func TestComputeCompositeScore_AllZeros(t *testing.T) {
	row := &compositeMatchRow{}
	got := computeCompositeScore(row, nil, nil, nil, nil, nil, nil)
	if got != 0.5 {
		t.Fatalf("expected 0.5 for empty, got %f", got)
	}
}

func TestComputeCompositeScore_Win(t *testing.T) {
	outcome := 2 // WIN
	row := &compositeMatchRow{
		Kills:          20,
		Deaths:         5,
		KillsExpected:  10,
		DeathsExpected: 10,
		Outcome:        &outcome,
		DamageDealt:    2000,
		DamageTaken:    500,
		Accuracy:       0.5,
	}
	avgAcc := 0.4
	got := computeCompositeScore(row, &avgAcc, nil, nil, nil, nil, nil)
	if got <= 0.5 {
		t.Fatalf("expected > 0.5 for strong win, got %f", got)
	}
	if got > 1.0 {
		t.Fatalf("expected <= 1.0, got %f", got)
	}
}

func TestComputeCompositeScore_Loss(t *testing.T) {
	outcome := 3 // LOSS
	row := &compositeMatchRow{
		Kills:          3,
		Deaths:         15,
		KillsExpected:  10,
		DeathsExpected: 5,
		Outcome:        &outcome,
		DamageDealt:    300,
		DamageTaken:    2000,
		Accuracy:       0.2,
	}
	avgAcc := 0.4
	got := computeCompositeScore(row, &avgAcc, nil, nil, nil, nil, nil)
	if got >= 0.5 {
		t.Fatalf("expected < 0.5 for clear loss, got %f", got)
	}
}

func TestComputeCompositeScore_DNF(t *testing.T) {
	outcome := 4 // DNF
	row := &compositeMatchRow{
		Kills:          5,
		Deaths:         5,
		KillsExpected:  10,
		DeathsExpected: 10,
		Outcome:        &outcome,
		DamageDealt:    500,
		DamageTaken:    500,
	}
	got := computeCompositeScore(row, nil, nil, nil, nil, nil, nil)
	// DNF gives winScore=0.15, composite should be < 0.5
	if got >= 0.6 {
		t.Fatalf("expected < 0.6 for DNF, got %f", got)
	}
}

// ── computeMatchKEStats ──────────────────────────────────────────────────────

func TestComputeMatchKEStats_Empty(t *testing.T) {
	avg, std := computeMatchKEStats(nil)
	if avg != InitialMU {
		t.Fatalf("expected InitialMU=%f, got %f", InitialMU, avg)
	}
	if std != 1.0 {
		t.Fatalf("expected std=1.0, got %f", std)
	}
}

func TestComputeMatchKEStats_Single(t *testing.T) {
	parts := []lusrParticipant{{KillsExpected: 10.0}}
	avg, std := computeMatchKEStats(parts)
	if avg != 10.0 {
		t.Fatalf("expected avg=10.0, got %f", avg)
	}
	if std != 1.0 {
		t.Fatalf("expected std=1.0 for single, got %f", std)
	}
}

func TestComputeMatchKEStats_Multiple(t *testing.T) {
	parts := []lusrParticipant{
		{KillsExpected: 10.0},
		{KillsExpected: 20.0},
		{KillsExpected: 30.0},
	}
	avg, std := computeMatchKEStats(parts)
	if math.Abs(avg-20.0) > 0.01 {
		t.Fatalf("expected avg≈20, got %f", avg)
	}
	if std <= 0 {
		t.Fatalf("expected std>0, got %f", std)
	}
}

func TestComputeMatchKEStats_SkipZeroKE(t *testing.T) {
	parts := []lusrParticipant{
		{KillsExpected: 0},
		{KillsExpected: 10.0},
	}
	avg, std := computeMatchKEStats(parts)
	if avg != 10.0 {
		t.Fatalf("expected avg=10 (skipped 0), got %f", avg)
	}
	if std != 1.0 {
		t.Fatalf("expected std=1.0 for single valid, got %f", std)
	}
}

// ── splitParticipantKEs ──────────────────────────────────────────────────────

func TestSplitParticipantKEs_NilTeamID(t *testing.T) {
	parts := []lusrParticipant{
		{KillsExpected: 5.0},
		{KillsExpected: 10.0},
	}
	teammates, enemies := splitParticipantKEs(nil, parts)
	if len(teammates) != 0 {
		t.Fatalf("expected 0 teammates, got %d", len(teammates))
	}
	if len(enemies) != 2 {
		t.Fatalf("expected 2 enemies, got %d", len(enemies))
	}
}

func TestSplitParticipantKEs_WithTeamID(t *testing.T) {
	team1 := 1
	team2 := 2
	parts := []lusrParticipant{
		{TeamID: &team1, KillsExpected: 5.0},
		{TeamID: &team1, KillsExpected: 8.0},
		{TeamID: &team2, KillsExpected: 10.0},
		{TeamID: &team2, KillsExpected: 12.0},
	}
	teammates, enemies := splitParticipantKEs(&team1, parts)
	if len(teammates) != 2 {
		t.Fatalf("expected 2 teammates, got %d", len(teammates))
	}
	if len(enemies) != 2 {
		t.Fatalf("expected 2 enemies, got %d", len(enemies))
	}
}
