package analysis

import (
	"testing"
	"time"

	"levelup/go-api/internal/domain"
)

// ── ComputeTeammateProfile ───────────────────────────────────────────────────

func TestComputeTeammateProfile_EmptyExtra(t *testing.T) {
	p := ComputeTeammateProfile(nil, "test", "#fff")
	if p.Name != "test" {
		t.Fatal("expected name")
	}
	if len(p.Values) != 0 {
		t.Fatal("expected empty values")
	}
}

func TestComputeTeammateProfile_WithData(t *testing.T) {
	acc := 65.0
	ratio := 2.0
	rows := []domain.TeammateMatchRow{
		{
			MatchID: "m1", StartTime: time.Now(), Kills: 20, Deaths: 10, Assists: 5,
			Accuracy: &acc, Ratio: &ratio, TimePlayedSecs: 600,
		},
		{
			MatchID: "m2", StartTime: time.Now(), Kills: 10, Deaths: 5, Assists: 3,
			Accuracy: &acc, Ratio: &ratio, TimePlayedSecs: 300,
		},
	}
	p := ComputeTeammateProfile(rows, "Player", "#0f0")
	if p.Values["kills"] != 15 { // avg(20,10)
		t.Fatalf("expected kills=15, got %v", p.Values["kills"])
	}
	if p.Values["accuracy"] != 65 {
		t.Fatalf("expected accuracy=65, got %v", p.Values["accuracy"])
	}
}

// ── ComputeTeammateRecords ───────────────────────────────────────────────────

func TestComputeTeammateRecords_EmptyExtra(t *testing.T) {
	r := ComputeTeammateRecords(nil)
	if r["kills"] != nil {
		t.Fatal("expected nil kills")
	}
}

func TestComputeTeammateRecords_WithData(t *testing.T) {
	acc1, acc2 := 70.0, 80.0
	ratio1, ratio2 := 1.5, 3.0
	rows := []domain.TeammateMatchRow{
		{Kills: 10, Deaths: 8, Assists: 4, Accuracy: &acc1, Ratio: &ratio1},
		{Kills: 25, Deaths: 3, Assists: 9, Accuracy: &acc2, Ratio: &ratio2},
	}
	r := ComputeTeammateRecords(rows)
	if r["kills"] == nil || *r["kills"] != 25 {
		t.Fatalf("expected kills record=25, got %v", r["kills"])
	}
	if r["deaths"] == nil || *r["deaths"] != 3 {
		t.Fatalf("expected deaths record=3 (min), got %v", r["deaths"])
	}
	if r["kda"] == nil || *r["kda"] != 3.0 {
		t.Fatalf("expected kda record=3.0, got %v", r["kda"])
	}
}
