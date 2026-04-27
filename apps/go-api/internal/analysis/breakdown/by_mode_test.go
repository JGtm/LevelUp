package breakdown

import (
	"testing"

	"levelup/go-api/internal/games/canonical"
)

func TestByMode_Empty(t *testing.T) {
	t.Parallel()
	if got := ByMode(nil); len(got) != 0 {
		t.Errorf("nil: want 0, got %d", len(got))
	}
}

func TestByMode_GroupsBySubmode(t *testing.T) {
	t.Parallel()
	rows := []Row{
		{ModeName: "Slayer", Outcome: canonical.OutcomeWin},
		{ModeName: "Slayer", Outcome: canonical.OutcomeWin},
		{ModeName: "CTF", Outcome: canonical.OutcomeLoss},
		{ModeName: "CTF", Outcome: canonical.OutcomeWin},
	}
	got := ByMode(rows)
	if len(got) != 2 {
		t.Fatalf("want 2 modes, got %d", len(got))
	}
	if got[0].ModeName != "Slayer" || got[0].WinRate != 1 {
		t.Errorf("Slayer expected first (WR=1), got %s WR=%v", got[0].ModeName, got[0].WinRate)
	}
	if got[1].ModeName != "CTF" || got[1].WinRate != 0.5 {
		t.Errorf("CTF expected second (WR=0.5), got %s WR=%v", got[1].ModeName, got[1].WinRate)
	}
}

func TestByMode_IgnoresEmptyModeName(t *testing.T) {
	t.Parallel()
	rows := []Row{
		{ModeName: "", Outcome: canonical.OutcomeWin},
		{ModeName: "Slayer", Outcome: canonical.OutcomeWin},
	}
	got := ByMode(rows)
	if len(got) != 1 || got[0].ModeName != "Slayer" {
		t.Errorf("empty ModeName ignored expected, got %v", got)
	}
}

func TestByMode_TieBreakerAlphabetic(t *testing.T) {
	t.Parallel()
	rows := []Row{
		{ModeName: "Zebra", Outcome: canonical.OutcomeWin},
		{ModeName: "Alpha", Outcome: canonical.OutcomeWin},
	}
	got := ByMode(rows)
	if got[0].ModeName != "Alpha" {
		t.Errorf("alphabetic tie-breaker: got %s first", got[0].ModeName)
	}
}

func TestByModeCategory_GroupsByCategory(t *testing.T) {
	t.Parallel()
	rows := []Row{
		{ModeName: "Slayer", ModeCategory: "Assassin", Outcome: canonical.OutcomeWin},
		{ModeName: "Tactical", ModeCategory: "Assassin", Outcome: canonical.OutcomeLoss},
		{ModeName: "BTB", ModeCategory: "BTB", Outcome: canonical.OutcomeWin},
	}
	got := ByModeCategory(rows)
	if len(got) != 2 {
		t.Fatalf("want 2 categories, got %d", len(got))
	}
	// BTB (WR=1) > Assassin (WR=0.5)
	if got[0].ModeName != "BTB" || got[1].ModeName != "Assassin" {
		t.Errorf("order should be BTB then Assassin, got %s / %s", got[0].ModeName, got[1].ModeName)
	}
	if got[1].Played != 2 {
		t.Errorf("Assassin should aggregate 2 rows (Slayer + Tactical), got %d", got[1].Played)
	}
}

func TestByModeCategory_IgnoresEmptyCategory(t *testing.T) {
	t.Parallel()
	rows := []Row{
		{ModeName: "Unknown", ModeCategory: "", Outcome: canonical.OutcomeWin},
		{ModeName: "Slayer", ModeCategory: "Assassin", Outcome: canonical.OutcomeWin},
	}
	got := ByModeCategory(rows)
	if len(got) != 1 || got[0].ModeName != "Assassin" {
		t.Errorf("empty category ignored expected, got %v", got)
	}
}
