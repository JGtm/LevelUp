package timeline

import (
	"testing"
	"time"
)

func TestComputeTimePlayed_FullMatchPlayer(t *testing.T) {
	start := time.Date(2026, 1, 1, 12, 0, 0, 0, time.UTC)
	gameplayStart := start.Add(28 * time.Second) // T0=28s countdown
	gameplayEnd := start.Add(600 * time.Second)  // duration 600s

	// Joueur présent dès le countdown (first_joined ~ start), pas de last_leave.
	joined := start.Add(2 * time.Second)
	secs, q := ComputeTimePlayed(TimePlayedInput{FirstJoinedTime: joined}, gameplayStart, gameplayEnd)
	if q != TimePlayedOK {
		t.Fatalf("quality = %q, want ok", q)
	}
	// Clampé à gameplayStart → gameplay complet = 600 − 28 = 572s.
	if secs != 572 {
		t.Errorf("time_played = %d, want 572 (gameplay duration)", secs)
	}
}

func TestComputeTimePlayed_Quitter(t *testing.T) {
	start := time.Date(2026, 1, 1, 12, 0, 0, 0, time.UTC)
	gameplayStart := start.Add(28 * time.Second)
	gameplayEnd := start.Add(600 * time.Second)

	// Quitter : présent au début, part à 250s après gameplayStart.
	joined := start.Add(5 * time.Second)
	leave := gameplayStart.Add(250 * time.Second)
	secs, q := ComputeTimePlayed(TimePlayedInput{FirstJoinedTime: joined, LastLeaveTime: &leave}, gameplayStart, gameplayEnd)
	if q != TimePlayedOK {
		t.Fatalf("quality = %q, want ok", q)
	}
	if secs != 250 {
		t.Errorf("time_played = %d, want 250 (quitter)", secs)
	}
}

func TestComputeTimePlayed_Latecomer(t *testing.T) {
	start := time.Date(2026, 1, 1, 12, 0, 0, 0, time.UTC)
	gameplayStart := start.Add(28 * time.Second)
	gameplayEnd := start.Add(600 * time.Second)

	// Latecomer : rejoint 100s après le début du gameplay, reste jusqu'à la fin.
	joined := gameplayStart.Add(100 * time.Second)
	secs, q := ComputeTimePlayed(TimePlayedInput{FirstJoinedTime: joined}, gameplayStart, gameplayEnd)
	if q != TimePlayedOK {
		t.Fatalf("quality = %q, want ok", q)
	}
	// 572 − 100 = 472s.
	if secs != 472 {
		t.Errorf("time_played = %d, want 472 (latecomer)", secs)
	}
}

func TestComputeTimePlayed_LeaveAfterEndClamped(t *testing.T) {
	start := time.Date(2026, 1, 1, 12, 0, 0, 0, time.UTC)
	gameplayStart := start.Add(28 * time.Second)
	gameplayEnd := start.Add(600 * time.Second)

	joined := start.Add(5 * time.Second)
	leave := gameplayEnd.Add(30 * time.Second) // last_leave après la fin (film)
	secs, q := ComputeTimePlayed(TimePlayedInput{FirstJoinedTime: joined, LastLeaveTime: &leave}, gameplayStart, gameplayEnd)
	if q != TimePlayedOK {
		t.Fatalf("quality = %q, want ok", q)
	}
	if secs != 572 {
		t.Errorf("time_played = %d, want 572 (clampé sur gameplayEnd)", secs)
	}
}

func TestComputeTimePlayed_NoFirstJoined(t *testing.T) {
	start := time.Date(2026, 1, 1, 12, 0, 0, 0, time.UTC)
	_, q := ComputeTimePlayed(TimePlayedInput{}, start, start.Add(600*time.Second))
	if q != TimePlayedNoData {
		t.Errorf("quality = %q, want no_data (first_joined absent)", q)
	}
	if q.Computed() {
		t.Errorf("NoData ne doit pas être Computed()")
	}
}

func TestComputeTimePlayed_InvalidWindow(t *testing.T) {
	start := time.Date(2026, 1, 1, 12, 0, 0, 0, time.UTC)
	joined := start
	_, q := ComputeTimePlayed(TimePlayedInput{FirstJoinedTime: joined}, start, start) // gameplayEnd == start
	if q != TimePlayedNoData {
		t.Errorf("quality = %q, want no_data (fenêtre invalide)", q)
	}
}

func TestComputeTimePlayed_JoinedAfterLeave(t *testing.T) {
	start := time.Date(2026, 1, 1, 12, 0, 0, 0, time.UTC)
	gameplayStart := start.Add(28 * time.Second)
	gameplayEnd := start.Add(600 * time.Second)

	// Incohérence : rejoint après son last_leave → time_played 0.
	joined := gameplayStart.Add(300 * time.Second)
	leave := gameplayStart.Add(100 * time.Second)
	secs, q := ComputeTimePlayed(TimePlayedInput{FirstJoinedTime: joined, LastLeaveTime: &leave}, gameplayStart, gameplayEnd)
	if q != TimePlayedClampedZero {
		t.Errorf("quality = %q, want clamped_zero", q)
	}
	if secs != 0 {
		t.Errorf("time_played = %d, want 0", secs)
	}
}
