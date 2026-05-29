// Package domain — match_timeline_test.go : tests purs de MatchTimeline.
package domain

import (
	"testing"
	"time"
)

func TestMatchTimeline_AbsoluteClock(t *testing.T) {
	start := time.Date(2026, 1, 1, 12, 0, 0, 0, time.UTC)
	// duration 600s film, countdown 28s.
	tl := NewMatchTimelineAt(start, 600_000, 28_000)

	if !tl.HasClock() {
		t.Fatal("HasClock() doit être vrai avec StartUTC renseigné")
	}
	if got := tl.GameplayStartUTC(); !got.Equal(start.Add(28 * time.Second)) {
		t.Errorf("GameplayStartUTC = %v, want start+28s", got)
	}
	if got := tl.GameplayEndUTC(); !got.Equal(start.Add(600 * time.Second)) {
		t.Errorf("GameplayEndUTC = %v, want start+600s", got)
	}
	// Vraie durée jouable = 600 − 28 = 572s.
	if got := tl.GameplayDurationSeconds(); got != 572 {
		t.Errorf("GameplayDurationSeconds = %d, want 572", got)
	}
}

func TestMatchTimeline_NoClock(t *testing.T) {
	tl := NewMatchTimeline(600_000, 28_000) // sans StartUTC
	if tl.HasClock() {
		t.Error("HasClock() doit être faux sans StartUTC")
	}
}

func TestNewMatchTimeline_ClampsNegativeValues(t *testing.T) {
	tl := NewMatchTimeline(-100, -50)
	if tl.DurationMs != 0 || tl.T0Ms != 0 {
		t.Errorf("expected clamp to 0,0, got %d,%d", tl.DurationMs, tl.T0Ms)
	}
}

func TestNewMatchTimeline_PreservesValidValues(t *testing.T) {
	tl := NewMatchTimeline(447000, 28000)
	if tl.DurationMs != 447000 || tl.T0Ms != 28000 {
		t.Errorf("expected 447000,28000, got %d,%d", tl.DurationMs, tl.T0Ms)
	}
}

func TestMatchTimeline_IsValid(t *testing.T) {
	cases := []struct {
		name    string
		dur, t0 int64
		want    bool
	}{
		{"canonical Fortress", 447000, 28000, true},
		{"t0 zero (no countdown)", 600000, 0, true},
		{"duration zero", 0, 0, false},
		{"t0 equals duration", 100, 100, true},
		{"t0 exceeds duration", 100, 200, false},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			tl := NewMatchTimeline(c.dur, c.t0)
			if got := tl.IsValid(); got != c.want {
				t.Errorf("IsValid() = %v, want %v", got, c.want)
			}
		})
	}
}

func TestMatchTimeline_GameplayDurationMs(t *testing.T) {
	cases := []struct {
		name    string
		dur, t0 int64
		want    int64
	}{
		{"Fortress 447s - 28s = 419s", 447000, 28000, 419000},
		{"t0=0", 600000, 0, 600000},
		{"t0 > dur clamped to 0", 100, 200, 0},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			tl := NewMatchTimeline(c.dur, c.t0)
			if got := tl.GameplayDurationMs(); got != c.want {
				t.Errorf("GameplayDurationMs() = %d, want %d", got, c.want)
			}
		})
	}
}

func TestMatchTimeline_GameplayDurationSeconds(t *testing.T) {
	tl := NewMatchTimeline(447523, 28047)
	got := tl.GameplayDurationSeconds()
	if got != 419 {
		t.Errorf("GameplayDurationSeconds() = %d, want 419", got)
	}
}

func TestMatchTimeline_CorrectEventTime(t *testing.T) {
	tl := NewMatchTimeline(447000, 28000)
	cases := []struct {
		raw, want int64
	}{
		{28000, 0},
		{30000, 2000},
		{447000, 419000},
		{15000, -13000}, // event before T0 (rare, caller filters)
	}
	for _, c := range cases {
		got := tl.CorrectEventTime(c.raw)
		if got != c.want {
			t.Errorf("CorrectEventTime(%d) = %d, want %d", c.raw, got, c.want)
		}
	}
}

func TestMatchTimeline_RoundTrip(t *testing.T) {
	tl := NewMatchTimeline(447000, 28000)
	for _, raw := range []int64{0, 1000, 28000, 100000, 446999} {
		corrected := tl.CorrectEventTime(raw)
		back := tl.RawTimeFromCorrected(corrected)
		if back != raw {
			t.Errorf("round-trip failed: raw=%d → corrected=%d → back=%d", raw, corrected, back)
		}
	}
}

func TestMatchTimeline_T0ZeroIsIdentity(t *testing.T) {
	tl := NewMatchTimeline(600000, 0)
	for _, raw := range []int64{0, 100, 12345, 599999} {
		if got := tl.CorrectEventTime(raw); got != raw {
			t.Errorf("with T0=0, CorrectEventTime should be identity: %d → %d", raw, got)
		}
	}
}
