// Package scheduler — auto_sync_history_test.go : ring buffer d'historique
// des cycles (dashboard monitoring). Tests white-box (storeCycleResult est
// interne, alimenté par RunOnceTrigger).
package scheduler

import (
	"context"
	"fmt"
	"testing"
	"time"
)

// TestCycleHistory_OrderAndTrigger : History() retourne le plus récent en
// premier, avec le trigger et les compteurs du cycle.
func TestCycleHistory_OrderAndTrigger(t *testing.T) {
	s := &AutoSyncScheduler{}
	s.storeCycleResult(context.Background(), &RunOnceResult{Total: 3, Synced: 2, Skipped: 1, Duration: 1500 * time.Millisecond}, "tick", cycleLoadSnapshot{})
	s.storeCycleResult(context.Background(), &RunOnceResult{Total: 3, Failed: 3, Duration: 200 * time.Millisecond}, "manual", cycleLoadSnapshot{})

	hist := s.History()
	if len(hist) != 2 {
		t.Fatalf("len(History) = %d (attendu 2)", len(hist))
	}
	if hist[0].Trigger != "manual" || hist[0].Failed != 3 || hist[0].DurationMs != 200 {
		t.Errorf("entrée la plus récente inattendue : %+v", hist[0])
	}
	if hist[1].Trigger != "tick" || hist[1].Synced != 2 || hist[1].Total != 3 {
		t.Errorf("entrée la plus ancienne inattendue : %+v", hist[1])
	}
	if hist[0].At.Before(hist[1].At) {
		t.Error("History doit être trié du plus récent au plus ancien")
	}
}

// TestCycleHistory_BoundedRing : l'historique est borné à cycleHistorySize —
// les plus anciens cycles sortent de la fenêtre.
func TestCycleHistory_BoundedRing(t *testing.T) {
	s := &AutoSyncScheduler{}
	total := cycleHistorySize + 10
	for i := 0; i < total; i++ {
		s.storeCycleResult(context.Background(), &RunOnceResult{Total: i}, "tick", cycleLoadSnapshot{})
	}
	hist := s.History()
	if len(hist) != cycleHistorySize {
		t.Fatalf("len(History) = %d (attendu %d)", len(hist), cycleHistorySize)
	}
	// Le plus récent porte Total = total-1 ; le plus ancien conservé = total-cycleHistorySize.
	if hist[0].Total != total-1 {
		t.Errorf("plus récent Total = %d (attendu %d)", hist[0].Total, total-1)
	}
	if hist[len(hist)-1].Total != total-cycleHistorySize {
		t.Errorf("plus ancien Total = %d (attendu %d)", hist[len(hist)-1].Total, total-cycleHistorySize)
	}
}

// TestCycleHistory_CopySemantics : muter le slice retourné ne corrompt pas
// l'état interne (le snapshot est une copie).
func TestCycleHistory_CopySemantics(t *testing.T) {
	s := &AutoSyncScheduler{}
	s.storeCycleResult(context.Background(), &RunOnceResult{Total: 7}, "tick", cycleLoadSnapshot{})
	first := s.History()
	first[0].Total = 999

	if again := s.History(); again[0].Total != 7 {
		t.Fatalf("History doit retourner une copie — état interne corrompu : %+v", again[0])
	}
}

// TestCycleHistory_StoreAlsoUpdatesSnapshot : storeCycleResult alimente aussi
// lastCycleAt/lastCycleResult (le point de passage unique remplace l'ancien
// bloc snapshotMu inline de RunOnce — non-régression Snapshot()). Lecture
// white-box des champs (Snapshot() exigerait un settings store complet).
func TestCycleHistory_StoreAlsoUpdatesSnapshot(t *testing.T) {
	s := &AutoSyncScheduler{playerOutcomes: map[string]PlayerOutcomeDetail{}}
	res := &RunOnceResult{Total: 4, Synced: 4, Duration: time.Second}
	s.storeCycleResult(context.Background(), res, "tick", cycleLoadSnapshot{})

	// Muter le résultat source ne doit pas affecter l'état mémorisé (copie).
	res.Synced = 0

	s.snapshotMu.RLock()
	defer s.snapshotMu.RUnlock()
	if s.lastCycleAt.IsZero() {
		t.Fatal("lastCycleAt doit être posé par storeCycleResult")
	}
	if s.lastCycleResult == nil || s.lastCycleResult.Synced != 4 {
		t.Fatalf("lastCycleResult inattendu : %+v", s.lastCycleResult)
	}
}

// TestCycleHistory_EmptyByDefault : aucun cycle → historique vide non-nil
// (contrat JSON : tableau vide, pas null).
func TestCycleHistory_EmptyByDefault(t *testing.T) {
	s := &AutoSyncScheduler{}
	if hist := s.History(); hist == nil || len(hist) != 0 {
		t.Fatalf("History par défaut = %v (attendu slice vide)", hist)
	}
	// Sanity : le JSON d'un slice vide est bien `[]`.
	if got := fmt.Sprintf("%v", s.History()); got != "[]" {
		t.Fatalf("représentation inattendue : %s", got)
	}
}

// TestCycleLoadDelta : les deltas avant/après cycle sont attribués au cycle
// et clampés à 0 (un Reset expvar en cours de route ne produit jamais de
// valeurs négatives).
func TestCycleLoadDelta(t *testing.T) {
	before := cycleLoadSnapshot{blockedMs: 1000, swapCount: 2, readsRejected: 1, apiMs: 5000, persistWriteMs: 300}
	after := cycleLoadSnapshot{blockedMs: 4500, swapCount: 5, readsRejected: 1, apiMs: 65000, persistWriteMs: 1200}

	d := after.deltaSince(before)
	if d.blockedMs != 3500 || d.swapCount != 3 || d.readsRejected != 0 || d.apiMs != 60000 || d.persistWriteMs != 900 {
		t.Fatalf("deltas inattendus : %+v", d)
	}

	// Cumuls remis à zéro (Reset en test) → clamp à 0, jamais négatif.
	d = (cycleLoadSnapshot{}).deltaSince(before)
	if d.blockedMs != 0 || d.apiMs != 0 || d.persistWriteMs != 0 {
		t.Fatalf("clamp à 0 attendu : %+v", d)
	}

	// Les deltas se retrouvent dans le CycleRecord.
	s := &AutoSyncScheduler{}
	s.storeCycleResult(context.Background(), &RunOnceResult{Total: 1, Duration: time.Second}, "tick",
		cycleLoadSnapshot{blockedMs: 800, swapCount: 2, readsRejected: 3, apiMs: 600, persistWriteMs: 150})
	rec := s.History()[0]
	if rec.BlockedMs != 800 || rec.SwapCount != 2 || rec.ReadsRejected != 3 || rec.APIMs != 600 || rec.PersistWriteMs != 150 {
		t.Fatalf("CycleRecord charge inattendue : %+v", rec)
	}
}
