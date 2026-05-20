// auto_sync_zero_insert_test.go — counter ConsecutiveZeroInserts.
//
// Ajouté suite à l'incident 2026-05-20 (sync delta retournait inserted=0
// pendant 14 jours sans alerte). Vérifie que le compteur :
//   - reste à 0 tant qu'au moins 1 match est inséré
//   - incrémente cycle après cycle quand inserted=0
//   - se reset à 0 dès qu'un cycle réussit à insérer >=1 match
//   - préserve la valeur précédente sur outcome=skipped (joueur absent du pool, etc.)
//   - log WarnContext au franchissement de ConsecutiveZeroInsertWarnThreshold

package scheduler_test

import (
	"context"
	"testing"

	"levelup/go-api/internal/domain"
	"levelup/go-api/internal/scheduler"
)

func TestConsecutiveZeroInserts_IncrementsThenResets(t *testing.T) {
	repoRoot := t.TempDir()
	touchPlayerDB(t, repoRoot, "Player1")

	p := &fakePool{hasPlayerMap: map[string]bool{"Player1": true}, size: 1}
	s := newSchedulerForTest(t, repoRoot, p)

	// Cycles 1..3 : inserted=0 chaque coup → compteur 1,2,3.
	s.RunnerFactory = func(_ context.Context, _, _ string) scheduler.DeltaRunner {
		return &mockRunner{result: domain.SyncResult{MatchesInserted: 0, MatchesSkipped: 25}}
	}
	for cycle, want := range []int{1, 2, 3} {
		s.RunOnce(context.Background())
		snap := s.Snapshot()
		if got := snap.Players[0].ConsecutiveZeroInserts; got != want {
			t.Errorf("cycle %d: ConsecutiveZeroInserts = %d, want %d", cycle+1, got, want)
		}
	}

	// Cycle 4 : un match inséré → reset à 0.
	s.RunnerFactory = func(_ context.Context, _, _ string) scheduler.DeltaRunner {
		return &mockRunner{result: domain.SyncResult{MatchesInserted: 2}}
	}
	s.RunOnce(context.Background())
	snap := s.Snapshot()
	if got := snap.Players[0].ConsecutiveZeroInserts; got != 0 {
		t.Errorf("cycle 4 après insert>0: ConsecutiveZeroInserts = %d, want 0", got)
	}

	// Cycles 5..6 : ré-incrément depuis 0 si on retombe à inserted=0.
	s.RunnerFactory = func(_ context.Context, _, _ string) scheduler.DeltaRunner {
		return &mockRunner{result: domain.SyncResult{MatchesInserted: 0}}
	}
	s.RunOnce(context.Background())
	snap = s.Snapshot()
	if got := snap.Players[0].ConsecutiveZeroInserts; got != 1 {
		t.Errorf("cycle 5 après reset: ConsecutiveZeroInserts = %d, want 1", got)
	}
}

func TestConsecutiveZeroInserts_PreservedOnSkipped(t *testing.T) {
	repoRoot := t.TempDir()
	touchPlayerDB(t, repoRoot, "Player1")

	// Cycle 1+2 : joueur dans le pool, inserted=0 → compteur = 2.
	p := &fakePool{hasPlayerMap: map[string]bool{"Player1": true}, size: 1}
	s := newSchedulerForTest(t, repoRoot, p)
	s.RunnerFactory = func(_ context.Context, _, _ string) scheduler.DeltaRunner {
		return &mockRunner{result: domain.SyncResult{MatchesInserted: 0}}
	}
	s.RunOnce(context.Background())
	s.RunOnce(context.Background())
	if got := s.Snapshot().Players[0].ConsecutiveZeroInserts; got != 2 {
		t.Fatalf("setup: ConsecutiveZeroInserts = %d, want 2", got)
	}

	// Cycle 3 : joueur retiré du pool → outcome=skipped, compteur préservé à 2.
	p.hasPlayerMap = map[string]bool{}
	p.size = 0
	s.RunOnce(context.Background())
	snap := s.Snapshot()
	if snap.Players[0].Outcome != "skipped" {
		t.Errorf("Outcome = %q, want skipped", snap.Players[0].Outcome)
	}
	if got := snap.Players[0].ConsecutiveZeroInserts; got != 2 {
		t.Errorf("après skip: ConsecutiveZeroInserts = %d, want 2 (préservé)", got)
	}
}

func TestConsecutiveZeroInserts_ReachesWarnThreshold(t *testing.T) {
	repoRoot := t.TempDir()
	touchPlayerDB(t, repoRoot, "Player1")

	p := &fakePool{hasPlayerMap: map[string]bool{"Player1": true}, size: 1}
	s := newSchedulerForTest(t, repoRoot, p)
	s.RunnerFactory = func(_ context.Context, _, _ string) scheduler.DeltaRunner {
		return &mockRunner{result: domain.SyncResult{MatchesInserted: 0}}
	}

	// Boucle jusqu'au seuil + 1 pour franchir.
	for i := 0; i <= scheduler.ConsecutiveZeroInsertWarnThreshold; i++ {
		s.RunOnce(context.Background())
	}
	snap := s.Snapshot()
	got := snap.Players[0].ConsecutiveZeroInserts
	if got < scheduler.ConsecutiveZeroInsertWarnThreshold {
		t.Errorf("ConsecutiveZeroInserts = %d, want >= threshold %d",
			got, scheduler.ConsecutiveZeroInsertWarnThreshold)
	}
	// Le WARN log lui-même est observable via slog handler test, mais on se
	// contente ici de valider que la valeur atteint bien le seuil sans crash.
}
