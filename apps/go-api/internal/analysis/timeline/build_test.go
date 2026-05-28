package timeline

import (
	"testing"

	"levelup/go-api/internal/domain"
)

func ptrInt(v int) *int { return &v }

func TestBuildFromRegistry_Phase1_AlwaysT0Zero(t *testing.T) {
	reg := domain.MatchRegistryRow{
		MatchID:         "test-match",
		DurationSeconds: ptrInt(447),
	}
	tl := BuildFromRegistry(reg)
	if tl.T0Ms != 0 {
		t.Errorf("Phase 1: T0Ms must be 0, got %d", tl.T0Ms)
	}
	if tl.DurationMs != 447000 {
		t.Errorf("DurationMs = %d, want 447000", tl.DurationMs)
	}
}

func TestBuildFromRegistry_NilDurationDefaultsToZero(t *testing.T) {
	reg := domain.MatchRegistryRow{MatchID: "no-dur"}
	tl := BuildFromRegistry(reg)
	if tl.DurationMs != 0 {
		t.Errorf("nil DurationSeconds should yield DurationMs=0, got %d", tl.DurationMs)
	}
	if tl.T0Ms != 0 {
		t.Errorf("T0Ms = %d, want 0", tl.T0Ms)
	}
}

func TestBuildFromRegistry_GameplayEqualsDurationInPhase1(t *testing.T) {
	// Property: as long as T0=0 (Phase 1), GameplayDuration == Duration.
	reg := domain.MatchRegistryRow{DurationSeconds: ptrInt(600)}
	tl := BuildFromRegistry(reg)
	if tl.GameplayDurationMs() != tl.DurationMs {
		t.Errorf("Phase 1 invariant violated: gameplay=%d, duration=%d", tl.GameplayDurationMs(), tl.DurationMs)
	}
}
