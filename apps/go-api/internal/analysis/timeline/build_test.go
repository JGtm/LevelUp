package timeline

import (
	"testing"
	"time"

	"levelup/go-api/internal/domain"
	"levelup/go-api/internal/games/canonical"
)

func ptrInt(v int) *int       { return &v }
func ptrInt64(v int64) *int64 { return &v }

func TestBuildFromRegistry_NilRealStartTimeYieldsT0Zero(t *testing.T) {
	reg := domain.MatchRegistryRow{
		MatchID:         "test-match",
		DurationSeconds: ptrInt(447),
	}
	tl := BuildFromRegistry(reg)
	if tl.T0Ms != 0 {
		t.Errorf("nil RealStartTime: T0Ms must be 0, got %d", tl.T0Ms)
	}
	if tl.DurationMs != 447000 {
		t.Errorf("DurationMs = %d, want 447000", tl.DurationMs)
	}
}

func TestBuildFromRegistry_DerivesT0FromRealStartTime(t *testing.T) {
	start := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	real := start.Add(28 * time.Second)
	reg := domain.MatchRegistryRow{
		MatchID:         "with-t0",
		StartTime:       start,
		RealStartTime:   &real,
		DurationSeconds: ptrInt(600),
	}
	tl := BuildFromRegistry(reg)
	if tl.T0Ms != 28000 {
		t.Errorf("T0Ms want 28000 (28s countdown), got %d", tl.T0Ms)
	}
	// Gameplay = duration − T0.
	if tl.GameplayDurationMs() != 600000-28000 {
		t.Errorf("GameplayDurationMs = %d, want %d", tl.GameplayDurationMs(), 600000-28000)
	}
}

func TestBuildFromRegistry_NilDurationDefaultsToZero(t *testing.T) {
	reg := domain.MatchRegistryRow{MatchID: "no-dur"}
	tl := BuildFromRegistry(reg)
	if tl.DurationMs != 0 {
		t.Errorf("nil DurationSeconds should yield DurationMs=0, got %d", tl.DurationMs)
	}
}

func TestBuildTimelinesFromPlayerMatches_ReadsSummaryT0Ms(t *testing.T) {
	rows := []canonical.PlayerMatchRow{
		{Summary: canonical.MatchSummary{
			MatchID: "m_t0", DurationSeconds: ptrInt(600), T0Ms: ptrInt64(28000),
		}},
		{Summary: canonical.MatchSummary{
			MatchID: "m_no_t0", DurationSeconds: ptrInt(600), T0Ms: nil,
		}},
	}
	got := BuildTimelinesFromPlayerMatches(rows)
	if got["m_t0"].T0Ms != 28000 {
		t.Errorf("m_t0: T0Ms want 28000, got %d", got["m_t0"].T0Ms)
	}
	if got["m_no_t0"].T0Ms != 0 {
		t.Errorf("m_no_t0: T0Ms want 0 (nil fallback), got %d", got["m_no_t0"].T0Ms)
	}
}

func TestBuildTimelinesFromSquadRows_ReadsT0Ms(t *testing.T) {
	rows := []domain.SquadMatchRow{
		{MatchID: "m_t0", DurationSeconds: 600, T0Ms: ptrInt64(28000)},
		{MatchID: "m_no_t0", DurationSeconds: 600, T0Ms: nil},
		// Même match, autre coéquipier : même T0, écrasement sans effet.
		{MatchID: "m_t0", DurationSeconds: 600, T0Ms: ptrInt64(28000)},
	}
	got := BuildTimelinesFromSquadRows(rows)
	if got["m_t0"].T0Ms != 28000 {
		t.Errorf("m_t0: T0Ms want 28000, got %d", got["m_t0"].T0Ms)
	}
	if got["m_t0"].DurationMs != 600000 {
		t.Errorf("m_t0: DurationMs want 600000, got %d", got["m_t0"].DurationMs)
	}
	if got["m_no_t0"].T0Ms != 0 {
		t.Errorf("m_no_t0: T0Ms want 0 (nil fallback), got %d", got["m_no_t0"].T0Ms)
	}
}

func TestBuildForMatchMs_UsesT0Ms(t *testing.T) {
	tl := BuildForMatchMs(600000, 28000)
	if tl.T0Ms != 28000 {
		t.Errorf("T0Ms want 28000, got %d", tl.T0Ms)
	}
	if tl.GameplayDurationMs() != 572000 {
		t.Errorf("GameplayDurationMs want 572000, got %d", tl.GameplayDurationMs())
	}
	// T0=0 → identité (chronologie brute).
	if BuildForMatchMs(600000, 0).GameplayDurationMs() != 600000 {
		t.Errorf("T0=0 should leave gameplay == duration")
	}
}
