package prestige

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"
)

// fakeTelemetryRepo capture les events émis pour vérification.
type fakeTelemetryRepo struct {
	mu     sync.Mutex
	events []PrestigeTelemetry
	failOn string // EventType qui doit faire échouer Emit
}

func (f *fakeTelemetryRepo) Emit(_ context.Context, ev PrestigeTelemetry) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.failOn != "" && ev.EventType == f.failOn {
		return errors.New("simulated telemetry failure")
	}
	f.events = append(f.events, ev)
	return nil
}

func (f *fakeTelemetryRepo) count() int {
	f.mu.Lock()
	defer f.mu.Unlock()
	return len(f.events)
}

func (f *fakeTelemetryRepo) last() PrestigeTelemetry {
	f.mu.Lock()
	defer f.mu.Unlock()
	if len(f.events) == 0 {
		return PrestigeTelemetry{}
	}
	return f.events[len(f.events)-1]
}

func TestTelemetryEmitter_EmitCreated(t *testing.T) {
	repo := &fakeTelemetryRepo{}
	em := NewTelemetryEmitter(repo, nil)
	c := Challenge{ID: "ch1", UserID: "u1", Tier: TierHeroic, Mode: ModeLibre, Cadence: CadenceWeekly}
	em.EmitCreated(context.Background(), c, 1.42, 1.18)
	if repo.count() != 1 {
		t.Fatalf("expected 1 event, got %d", repo.count())
	}
	ev := repo.last()
	if ev.EventType != TelemetryCreated {
		t.Errorf("event_type got %q want %q", ev.EventType, TelemetryCreated)
	}
	if ev.StretchRatio != 1.42 || ev.BaselineValue != 1.18 {
		t.Errorf("stretch/baseline mismatch: %v / %v", ev.StretchRatio, ev.BaselineValue)
	}
}

func TestTelemetryEmitter_EmitTransition(t *testing.T) {
	repo := &fakeTelemetryRepo{}
	now := time.Now().UTC()
	committed := now.Add(-5 * time.Minute)
	em := NewTelemetryEmitter(repo, func() time.Time { return now })
	c := Challenge{ID: "ch2", UserID: "u1", Tier: TierLegendary, CommittedAt: &committed}
	em.EmitTransition(context.Background(), c, TelemetryCompleted)
	ev := repo.last()
	if ev.EventType != TelemetryCompleted {
		t.Errorf("got %q want %q", ev.EventType, TelemetryCompleted)
	}
	if ev.TimeSinceCreateSeconds < 290 || ev.TimeSinceCreateSeconds > 310 {
		t.Errorf("expected ~300s, got %d", ev.TimeSinceCreateSeconds)
	}
}

func TestTelemetryEmitter_EmitRejected(t *testing.T) {
	repo := &fakeTelemetryRepo{}
	em := NewTelemetryEmitter(repo, nil)
	em.EmitRejected(context.Background(), "u1", Challenge{ID: "ch3"}, 1.05, 1.0, RejectTooEasy)
	ev := repo.last()
	if ev.EventType != TelemetryRejected+":"+string(RejectTooEasy) {
		t.Errorf("got %q", ev.EventType)
	}
}

func TestTelemetryEmitter_EmitPalierRecomputed(t *testing.T) {
	repo := &fakeTelemetryRepo{}
	em := NewTelemetryEmitter(repo, nil)
	c := Challenge{ID: "ch4", Tier: TierMythic}
	em.EmitPalierRecomputed(context.Background(), c, TierHeroic, 1.92, 1.05)
	ev := repo.last()
	if ev.EventType != TelemetryPalierRecomputed {
		t.Errorf("got %q want %q", ev.EventType, TelemetryPalierRecomputed)
	}
}

// TestTelemetryEmitter_PropagatesSource : Challenge.Source est recopié sur les
// événements created ET transition (plumbing ADR 0020).
func TestTelemetryEmitter_PropagatesSource(t *testing.T) {
	repo := &fakeTelemetryRepo{}
	em := NewTelemetryEmitter(repo, nil)
	c := Challenge{ID: "ch_src", UserID: "u1", Tier: TierHeroic, Source: ChallengeSourceCoach}

	em.EmitCreated(context.Background(), c, 1.4, 1.1)
	if ev := repo.last(); ev.Source != ChallengeSourceCoach {
		t.Errorf("created source: got %q want %q", ev.Source, ChallengeSourceCoach)
	}

	em.EmitTransition(context.Background(), c, TelemetryCompleted)
	if ev := repo.last(); ev.Source != ChallengeSourceCoach {
		t.Errorf("transition source: got %q want %q", ev.Source, ChallengeSourceCoach)
	}
}

func TestTelemetryEmitter_NilRepoSafe(t *testing.T) {
	em := NewTelemetryEmitter(nil, nil)
	defer func() {
		if r := recover(); r != nil {
			t.Errorf("emitter with nil repo should not panic: %v", r)
		}
	}()
	em.EmitCreated(context.Background(), Challenge{}, 0, 0)
}

func TestTelemetryEmitter_FailureLoggedNotPropagated(t *testing.T) {
	repo := &fakeTelemetryRepo{failOn: TelemetryCreated}
	em := NewTelemetryEmitter(repo, nil)
	// Pas de panic ni de retour d'erreur — best-effort.
	em.EmitCreated(context.Background(), Challenge{ID: "ch5"}, 1.5, 1.0)
}

func TestTelemetryEmitter_NewIDsUnique(t *testing.T) {
	seen := make(map[string]bool)
	for i := 0; i < 100; i++ {
		id := newTelemetryID()
		if seen[id] {
			t.Fatalf("duplicate id: %s", id)
		}
		seen[id] = true
	}
}
