package prestige

import (
	"testing"
	"time"
)

// ─────────── Baseline ───────────

func TestComputeBaseline_FullData(t *testing.T) {
	tuning := DefaultTuning()
	matches := makeMatches(20, 1.5)
	b := ComputeBaseline(tuning, "u1", "halo_infinite", "FieldKDA", matches)
	if b.Value < 1.49 || b.Value > 1.51 {
		t.Errorf("expected ~1.5 baseline, got %.4f", b.Value)
	}
	if b.MatchCount != 20 {
		t.Errorf("expected 20 matches, got %d", b.MatchCount)
	}
	if b.DataTier != DataFull {
		t.Errorf("expected DataFull, got %s", b.DataTier)
	}
}

func TestComputeBaseline_Estimated(t *testing.T) {
	tuning := DefaultTuning()
	matches := makeMatches(7, 1.0)
	b := ComputeBaseline(tuning, "u1", "halo_infinite", "FieldKDA", matches)
	if b.DataTier != DataEstimated {
		t.Errorf("expected DataEstimated, got %s", b.DataTier)
	}
}

func TestComputeBaseline_Tracking(t *testing.T) {
	tuning := DefaultTuning()
	matches := makeMatches(3, 1.0)
	b := ComputeBaseline(tuning, "u1", "halo_infinite", "FieldKDA", matches)
	if b.DataTier != DataTracking {
		t.Errorf("expected DataTracking, got %s", b.DataTier)
	}
}

func TestComputeBaseline_Empty(t *testing.T) {
	tuning := DefaultTuning()
	b := ComputeBaseline(tuning, "u1", "halo_infinite", "FieldKDA", nil)
	if b.MatchCount != 0 {
		t.Errorf("expected 0 matches, got %d", b.MatchCount)
	}
	if b.DataTier != DataTracking {
		t.Errorf("expected DataTracking, got %s", b.DataTier)
	}
	if b.Value != 0 {
		t.Errorf("expected 0 value, got %.4f", b.Value)
	}
}

func TestComputeBaseline_WindowLimitedTo20(t *testing.T) {
	tuning := DefaultTuning()
	matches := makeMatches(50, 1.0) // 50 matchs, on prend les 20 premiers
	b := ComputeBaseline(tuning, "u1", "halo_infinite", "FieldKDA", matches)
	if b.MatchCount != 20 {
		t.Errorf("expected window 20, got %d", b.MatchCount)
	}
}

func TestCheckStaleness(t *testing.T) {
	tuning := DefaultTuning()
	now := time.Date(2026, 6, 1, 12, 0, 0, 0, time.UTC)
	old := now.Add(-65 * 24 * time.Hour) // 65 jours = stale
	recent := now.Add(-30 * 24 * time.Hour)

	if !CheckStaleness(tuning, &old, now) {
		t.Error("expected stale for >60 days")
	}
	if CheckStaleness(tuning, &recent, now) {
		t.Error("expected not stale for 30 days")
	}
	if CheckStaleness(tuning, nil, now) {
		t.Error("expected not stale for nil last_match (new player)")
	}
}

func TestRecoveryDataTier(t *testing.T) {
	tuning := DefaultTuning()
	st := BaselineState{RecoveryMatchesRemaining: 5}
	// En recovery + beaucoup de matchs : DataEstimated
	if got := RecoveryDataTier(tuning, st, 30); got != DataEstimated {
		t.Errorf("expected Estimated during recovery, got %s", got)
	}
	// Recovery terminé + matchs suffisants : DataFull
	st.RecoveryMatchesRemaining = 0
	if got := RecoveryDataTier(tuning, st, 30); got != DataFull {
		t.Errorf("expected Full after recovery, got %s", got)
	}
}

func TestAdvanceRecovery(t *testing.T) {
	now := time.Now().UTC()
	st := BaselineState{IsStale: true, RecoveryMatchesRemaining: 2}
	updated := AdvanceRecovery(st, now)
	if updated.RecoveryMatchesRemaining != 1 {
		t.Errorf("expected 1 remaining, got %d", updated.RecoveryMatchesRemaining)
	}
	if !updated.IsStale {
		t.Error("expected still stale at recovery=1")
	}
	updated = AdvanceRecovery(updated, now)
	if updated.RecoveryMatchesRemaining != 0 {
		t.Errorf("expected 0 remaining, got %d", updated.RecoveryMatchesRemaining)
	}
	if updated.IsStale {
		t.Error("expected not stale at recovery=0")
	}
}

func TestMarkStale(t *testing.T) {
	tuning := DefaultTuning()
	now := time.Now().UTC()
	st := MarkStale(tuning, BaselineState{}, now)
	if !st.IsStale {
		t.Error("expected IsStale=true")
	}
	if st.RecoveryMatchesRemaining != tuning.AntiSmurf.RecoveryMatches {
		t.Errorf("expected recovery=%d, got %d",
			tuning.AntiSmurf.RecoveryMatches, st.RecoveryMatchesRemaining)
	}
}

// ─────────── Lifecycle ───────────

func TestCanEditTarget(t *testing.T) {
	cases := []struct {
		name string
		c    Challenge
		want bool
	}{
		{"libre + active = true", Challenge{Status: StatusActive, Mode: ModeLibre}, true},
		{"libre + draft = false", Challenge{Status: StatusDraft, Mode: ModeLibre}, false},
		{"libre + completed = false", Challenge{Status: StatusCompleted, Mode: ModeLibre}, false},
		{"pilote + active = false", Challenge{Status: StatusActive, Mode: ModePilote}, false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := CanEditTarget(tc.c); got != tc.want {
				t.Errorf("got %v want %v", got, tc.want)
			}
		})
	}
}

func TestCommit_FromDraft(t *testing.T) {
	c := Challenge{Status: StatusDraft}
	now := time.Now().UTC()
	updated, err := Commit(c, now)
	if err != nil {
		t.Fatalf("unexpected err: %v", err)
	}
	if updated.Status != StatusActive {
		t.Errorf("got %s want active", updated.Status)
	}
	if updated.CommittedAt == nil {
		t.Error("CommittedAt should be set")
	}
}

func TestCommit_FromNonDraftFails(t *testing.T) {
	c := Challenge{Status: StatusActive}
	_, err := Commit(c, time.Now())
	if err == nil {
		t.Error("expected error from non-draft commit")
	}
}

func TestMarkCompleted_Expired_Abandoned(t *testing.T) {
	now := time.Now().UTC()
	c := Challenge{Status: StatusActive}
	completed, err := MarkCompleted(c, now)
	if err != nil || completed.Status != StatusCompleted {
		t.Errorf("MarkCompleted failed: %v / %s", err, completed.Status)
	}
	expired, err := MarkExpired(c, now)
	if err != nil || expired.Status != StatusExpired {
		t.Errorf("MarkExpired failed: %v / %s", err, expired.Status)
	}
	abandoned, err := MarkAbandoned(c, now)
	if err != nil || abandoned.Status != StatusAbandoned {
		t.Errorf("MarkAbandoned failed: %v / %s", err, abandoned.Status)
	}
}

func TestCooldownEndsAt(t *testing.T) {
	tuning := DefaultTuning()
	now := time.Date(2026, 5, 1, 12, 0, 0, 0, time.UTC)
	abandonedAt := now.Add(-10 * time.Hour)
	c := Challenge{
		Status:      StatusAbandoned,
		Mode:        ModePilote,
		AbandonedAt: &abandonedAt,
	}
	end := CooldownEndsAt(tuning, c)
	expected := abandonedAt.Add(48 * time.Hour)
	if !end.Equal(expected) {
		t.Errorf("got %v want %v", end, expected)
	}
}

func TestCooldownEndsAt_LibreNoCooldown(t *testing.T) {
	tuning := DefaultTuning()
	abandonedAt := time.Now().UTC()
	c := Challenge{
		Status:      StatusAbandoned,
		Mode:        ModeLibre,
		AbandonedAt: &abandonedAt,
	}
	if end := CooldownEndsAt(tuning, c); !end.IsZero() {
		t.Errorf("libre mode should have no cooldown, got %v", end)
	}
}

func TestIsCooldownActive(t *testing.T) {
	tuning := DefaultTuning()
	now := time.Date(2026, 5, 1, 12, 0, 0, 0, time.UTC)
	abandonedRecent := now.Add(-2 * time.Hour) // cooldown 48h → toujours actif
	abandonedOld := now.Add(-100 * time.Hour)  // cooldown 48h → expiré

	prevActive := []Challenge{
		{Status: StatusAbandoned, Mode: ModePilote, AbandonedAt: &abandonedRecent},
	}
	if !IsCooldownActive(tuning, prevActive, now) {
		t.Error("expected active cooldown")
	}

	prevExpired := []Challenge{
		{Status: StatusAbandoned, Mode: ModePilote, AbandonedAt: &abandonedOld},
	}
	if IsCooldownActive(tuning, prevExpired, now) {
		t.Error("expected expired cooldown")
	}
}

// ─────────── Squad target ───────────

func TestCollectiveTargetTotal(t *testing.T) {
	if got := CollectiveTargetTotal(10.0, 3); got != 30.0 {
		t.Errorf("got %.2f want 30", got)
	}
	if got := CollectiveTargetTotal(10.0, 0); got != 0 {
		t.Errorf("zero members should yield 0, got %.2f", got)
	}
}

func TestCollectiveBaseline_IgnoresTracking(t *testing.T) {
	baselines := []Baseline{
		{Value: 4.0, DataTier: DataFull},
		{Value: 5.0, DataTier: DataFull},
		{Value: 100.0, DataTier: DataTracking}, // ignoré
	}
	total, contributing := CollectiveBaseline(baselines)
	if total != 9.0 {
		t.Errorf("got %.2f want 9", total)
	}
	if contributing != 2 {
		t.Errorf("got %d contributing want 2", contributing)
	}
}

func TestValidateResizeForRemoval(t *testing.T) {
	// 3 membres × 10 = 30 cible. Progression 25 → retirer un membre
	// → cible 20, progression 25 dépasse → ErrTargetWouldGoBelowProgress
	err := ValidateResizeForRemoval(10.0, 3, 25.0)
	if err != ErrTargetWouldGoBelowProgress {
		t.Errorf("expected ErrTargetWouldGoBelowProgress, got %v", err)
	}
	// Progression 15 → cible 20 après retrait → OK
	err = ValidateResizeForRemoval(10.0, 3, 15.0)
	if err != nil {
		t.Errorf("expected nil, got %v", err)
	}
}

// ─────────── Helpers ───────────

func makeMatches(n int, value float64) []MatchData {
	out := make([]MatchData, n)
	base := time.Date(2026, 4, 1, 12, 0, 0, 0, time.UTC)
	for i := 0; i < n; i++ {
		out[i] = MatchData{
			MatchID:     "m" + intToStr(i),
			StartedAt:   base.Add(time.Duration(i) * time.Hour),
			MetricValue: value,
		}
	}
	return out
}

func intToStr(n int) string {
	if n == 0 {
		return "0"
	}
	digits := []byte{}
	if n < 0 {
		digits = append(digits, '-')
		n = -n
	}
	rev := []byte{}
	for n > 0 {
		rev = append(rev, byte('0'+n%10))
		n /= 10
	}
	for i := len(rev) - 1; i >= 0; i-- {
		digits = append(digits, rev[i])
	}
	return string(digits)
}
