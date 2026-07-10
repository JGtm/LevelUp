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

// TestShouldUseV2_WhenWired : V2 pilote le cycle dès que l'orchestrator est câblé
// (unique moteur depuis la suppression du pipeline V1, lot D1c).
func TestShouldUseV2_WhenWired(t *testing.T) {
	s := &AutoSyncScheduler{cycleOrchestrator: stubCycleOrch{}}
	if !s.shouldUseV2() {
		t.Fatal("V2 doit piloter le cycle quand l'orchestrator est câblé")
	}
}

// TestShouldUseV2_NilOrchestrator : sans orchestrator câblé, V2 n'est pas piloté
// (le cycle bascule sur le filet syncPlayer de boot, aucun crash).
func TestShouldUseV2_NilOrchestrator(t *testing.T) {
	s := &AutoSyncScheduler{}
	if s.shouldUseV2() {
		t.Fatal("sans orchestrator câblé, V2 ne doit pas être piloté")
	}
}
