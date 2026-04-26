// Package api — post_sync_deltas_test.go : tests des helpers de delta-detection
// et de l'émetteur EmitPostSyncDeltas via un emitter recording in-memory.
package api

import (
	"context"
	"testing"

	"levelup/go-api/internal/notifications"
)

// recordingEmitter capture les emit pour assertions in-memory.
type recordingEmitter struct {
	emitted []notifications.EmitInput
	failOn  notifications.Category
}

func (r *recordingEmitter) Emit(_ context.Context, in notifications.EmitInput) error {
	if r.failOn != "" && in.Category == r.failOn {
		return errInjected
	}
	r.emitted = append(r.emitted, in)
	return nil
}

var errInjected = error_injected{}

type error_injected struct{}

func (error_injected) Error() string { return "injected" }

// ─── thresholdCrossed ────────────────────────────────────────────────────

func TestThresholdCrossed_Ascending(t *testing.T) {
	cases := []struct {
		name          string
		before, after float64
		step          float64
		wantCrossed   bool
		wantLevel     float64
	}{
		{"crosses 1.0 from 0.99", 0.99, 1.04, 0.05, true, 1.05}, // 0.99/0.05=19, 1.04/0.05=20 → bucket up → level=20*0.05=1.0; en réalité notre helper renvoie afterBucket * step
		{"no crossing, same bucket", 1.01, 1.04, 0.05, false, 0},
		{"descending ignored", 1.10, 0.99, 0.05, false, 0},
		{"equal ignored", 1.00, 1.00, 0.05, false, 0},
		{"large jump multiple steps", 0.50, 1.10, 0.05, true, 0},
		{"step=0 returns false", 1.0, 2.0, 0, false, 0},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			crossed, _ := thresholdCrossed(tc.before, tc.after, tc.step)
			if crossed != tc.wantCrossed {
				t.Errorf("crossed: got %v, want %v", crossed, tc.wantCrossed)
			}
		})
	}
}

func TestThresholdCrossed_LevelAccuracy(t *testing.T) {
	// 0.99 → bucket 19 ; 1.04 → bucket 20 ; level retourné = 20*0.05 = 1.00
	crossed, level := thresholdCrossed(0.99, 1.04, 0.05)
	if !crossed {
		t.Fatal("expected crossed=true")
	}
	if level < 0.99 || level > 1.01 {
		t.Errorf("level expected ~1.00, got %v", level)
	}
}

// ─── EmitPostSyncDeltas ─────────────────────────────────────────────────

func TestEmitPostSyncDeltas_NilGuards(t *testing.T) {
	em := &recordingEmitter{}
	// nil emitter
	EmitPostSyncDeltas(context.Background(), nil, "p1", &PlayerSnapshot{}, &PlayerSnapshot{}, nil)
	// nil before
	EmitPostSyncDeltas(context.Background(), em, "p1", nil, &PlayerSnapshot{}, nil)
	// nil after
	EmitPostSyncDeltas(context.Background(), em, "p1", &PlayerSnapshot{}, nil, nil)
	if len(em.emitted) != 0 {
		t.Errorf("expected no emits with nil args, got %d", len(em.emitted))
	}
}

func TestEmitPostSyncDeltas_NoChange_NoEmit(t *testing.T) {
	em := &recordingEmitter{}
	snap := &PlayerSnapshot{
		CurrentRank:        10,
		PersonalAwardCount: 5,
		CitationsCount:     3,
		KDRatio:            1.20,
		Winrate:            0.55,
	}
	EmitPostSyncDeltas(context.Background(), em, "p1", snap, snap, nil)
	if len(em.emitted) != 0 {
		t.Errorf("expected 0 emits when snapshots equal, got %d", len(em.emitted))
	}
}

func TestEmitPostSyncDeltas_SeasonPassLevel(t *testing.T) {
	em := &recordingEmitter{}
	before := &PlayerSnapshot{CurrentRank: 5, CurrentRankName: "Hero"}
	after := &PlayerSnapshot{CurrentRank: 6, CurrentRankName: "Onyx"}
	EmitPostSyncDeltas(context.Background(), em, "p1", before, after, nil)
	if !hasCategory(em.emitted, notifications.CategorySeasonPassLevel) {
		t.Error("expected season_pass_level emit when rank up")
	}
}

func TestEmitPostSyncDeltas_ObjectiveCompleted_AggregatedDelta(t *testing.T) {
	em := &recordingEmitter{}
	before := &PlayerSnapshot{PersonalAwardCount: 10}
	after := &PlayerSnapshot{PersonalAwardCount: 13}
	EmitPostSyncDeltas(context.Background(), em, "p1", before, after, nil)
	// Doit emit objective_completed ET objective_assigned (delta>0 sur PSA)
	if !hasCategory(em.emitted, notifications.CategoryObjectiveCompleted) {
		t.Error("expected objective_completed")
	}
	if !hasCategory(em.emitted, notifications.CategoryObjectiveAssigned) {
		t.Error("expected objective_assigned (MVP : delta = both)")
	}
}

func TestEmitPostSyncDeltas_ChallengeCompleted_AndAdded(t *testing.T) {
	em := &recordingEmitter{}
	before := &PlayerSnapshot{CitationsCount: 1, ChallengePathsCount: 5}
	after := &PlayerSnapshot{CitationsCount: 4, ChallengePathsCount: 7}
	EmitPostSyncDeltas(context.Background(), em, "p1", before, after, nil)
	if !hasCategory(em.emitted, notifications.CategoryChallengeCompleted) {
		t.Error("expected challenge_completed")
	}
	if !hasCategory(em.emitted, notifications.CategoryChallengeAdded) {
		t.Error("expected challenge_added")
	}
}

func TestEmitPostSyncDeltas_ThresholdCrossed_KDRatio(t *testing.T) {
	em := &recordingEmitter{}
	before := &PlayerSnapshot{KDRatio: 0.99, Winrate: 0.40}
	after := &PlayerSnapshot{KDRatio: 1.04, Winrate: 0.43}
	EmitPostSyncDeltas(context.Background(), em, "p1", before, after, nil)
	if !hasCategory(em.emitted, notifications.CategoryThresholdCrossed) {
		t.Error("expected threshold_crossed (KD up)")
	}
	// Doit avoir 1 emit threshold (KD), pas 2 (winrate ne franchit pas le palier 0.05)
	count := countCategory(em.emitted, notifications.CategoryThresholdCrossed)
	if count != 1 {
		t.Errorf("expected exactly 1 threshold emit, got %d", count)
	}
}

func TestEmitPostSyncDeltas_ThresholdCrossed_NoEmitOnDescent(t *testing.T) {
	em := &recordingEmitter{}
	before := &PlayerSnapshot{KDRatio: 1.10, Winrate: 0.55}
	after := &PlayerSnapshot{KDRatio: 0.99, Winrate: 0.49}
	EmitPostSyncDeltas(context.Background(), em, "p1", before, after, nil)
	if hasCategory(em.emitted, notifications.CategoryThresholdCrossed) {
		t.Error("threshold_crossed should NOT emit on descent")
	}
}

func TestEmitPostSyncDeltas_PersonalRecord_SkippedWithoutPDB(t *testing.T) {
	em := &recordingEmitter{}
	before := &PlayerSnapshot{BestKDA: 2.0}
	after := &PlayerSnapshot{BestKDA: 4.5, BestKDAMatchID: "abc"}
	// pdb nil → personal_record skip
	EmitPostSyncDeltas(context.Background(), em, "p1", before, after, nil)
	if hasCategory(em.emitted, notifications.CategoryPersonalRecord) {
		t.Error("personal_record should be skipped when pdb=nil")
	}
}

// ─── helpers de test ────────────────────────────────────────────────────

func hasCategory(items []notifications.EmitInput, c notifications.Category) bool {
	for _, it := range items {
		if it.Category == c {
			return true
		}
	}
	return false
}

func countCategory(items []notifications.EmitInput, c notifications.Category) int {
	n := 0
	for _, it := range items {
		if it.Category == c {
			n++
		}
	}
	return n
}
