package scheduler

import (
	"context"
	"testing"

	syncv2 "levelup/go-api/internal/sync/v2"
)

// stubCycleOrch : orchestrator V2 minimal pour tester le gate shouldUseV2 sans
// dépendre d'un pipeline réel.
type stubCycleOrch struct{}

func (stubCycleOrch) Run(context.Context, []syncv2.PlayerProfile) (syncv2.CycleResult, error) {
	return syncv2.CycleResult{}, nil
}

// TestShouldUseV2_DefaultOnWhenWired : V2 est le pipeline PAR DÉFAUT dès que
// l'orchestrator est câblé et qu'aucun opt-out n'est posé.
func TestShouldUseV2_DefaultOnWhenWired(t *testing.T) {
	t.Setenv("LEVELUP_SYNC_PIPELINE", "")
	s := &AutoSyncScheduler{cycleOrchestrator: stubCycleOrch{}}
	if !s.shouldUseV2() {
		t.Fatal("V2 doit être le pipeline PAR DÉFAUT quand l'orchestrator est câblé (flag absent)")
	}
}

// TestShouldUseV2_OptOutForcesV1 : LEVELUP_SYNC_PIPELINE=v1 (insensible casse/espaces)
// force le legacy V1 — l'échappatoire de rollback.
func TestShouldUseV2_OptOutForcesV1(t *testing.T) {
	s := &AutoSyncScheduler{cycleOrchestrator: stubCycleOrch{}}
	for _, v := range []string{"v1", "V1", " v1 ", "V1 "} {
		t.Setenv("LEVELUP_SYNC_PIPELINE", v)
		if s.shouldUseV2() {
			t.Errorf("LEVELUP_SYNC_PIPELINE=%q doit forcer le legacy V1", v)
		}
	}
}

// TestShouldUseV2_NilOrchestrator : sans orchestrator câblé, V2 n'est jamais tenté
// (fallback V1 structurel, aucun crash).
func TestShouldUseV2_NilOrchestrator(t *testing.T) {
	t.Setenv("LEVELUP_SYNC_PIPELINE", "")
	s := &AutoSyncScheduler{}
	if s.shouldUseV2() {
		t.Fatal("sans orchestrator câblé, V2 ne doit jamais être tenté")
	}
}
