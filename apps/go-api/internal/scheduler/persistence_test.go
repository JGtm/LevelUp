package scheduler

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"levelup/go-api/internal/domain"
	"levelup/go-api/internal/platform/adminstate"
)

// newPersistScheduler crée un scheduler minimal branché sur un store JSON temp.
func newPersistScheduler(t *testing.T) (*AutoSyncScheduler, string) {
	t.Helper()
	path := filepath.Join(t.TempDir(), "post_sync_snapshot.json")
	s := &AutoSyncScheduler{
		playerOutcomes:  map[string]PlayerOutcomeDetail{},
		postSyncHistory: map[string][]int64{},
	}
	s.WithSnapshotStore(adminstate.NewFileStore(path))
	return s, path
}

// TestPersistThenRehydrate : le snapshot post-sync survit à un « reboot »
// (nouveau scheduler, même fichier) — timeline + matrice + horodatage restaurés.
func TestPersistThenRehydrate(t *testing.T) {
	src, path := newPersistScheduler(t)
	src.recordOutcome(PlayerOutcomeDetail{
		Gamertag:    "JGtm",
		XUID:        "2533",
		Outcome:     "ok",
		AttemptedAt: time.Now(),
		PostSync:    &domain.PostSyncResult{DurationMs: 1234},
	})
	src.persistSnapshot(context.Background())

	if _, err := os.Stat(path); err != nil {
		t.Fatalf("snapshot non écrit sur disque: %v", err)
	}

	// « Reboot » : nouveau scheduler, même fichier.
	dst, _ := newPersistScheduler(t)
	dst.snapshotStore = adminstate.NewFileStore(path)
	dst.RehydrateFromDisk(context.Background())

	dst.snapshotMu.RLock()
	rec, ok := dst.playerOutcomes["JGtm"]
	hist := dst.postSyncHistory["JGtm"]
	sinceBoot := dst.cycleRanSinceBoot
	dst.snapshotMu.RUnlock()

	if !ok || rec.PostSync == nil || rec.PostSync.DurationMs != 1234 {
		t.Fatalf("outcome joueur non réhydraté: ok=%v rec=%+v", ok, rec)
	}
	// La durée post-sync alimente le ring (sparkline) — persisté séparément.
	if len(hist) != 1 || hist[0] != 1234 {
		t.Fatalf("ring post-sync non réhydraté: %v", hist)
	}
	// Réhydraté seul (aucun cycle depuis ce boot) → SinceBoot doit rester false.
	if sinceBoot {
		t.Fatalf("cycleRanSinceBoot=true après simple réhydratation (attendu false)")
	}
}

// TestSinceBootFlipsAfterCycle : un cycle effectif depuis le boot bascule
// SinceBoot à true (distingue « live » de « réhydraté »).
func TestSinceBootFlipsAfterCycle(t *testing.T) {
	s, _ := newPersistScheduler(t)
	s.RehydrateFromDisk(context.Background()) // fichier absent → no-op

	s.snapshotMu.RLock()
	before := s.cycleRanSinceBoot
	s.snapshotMu.RUnlock()
	if before {
		t.Fatalf("cycleRanSinceBoot=true avant tout cycle")
	}

	s.storeCycleResult(context.Background(), &RunOnceResult{Total: 1, Synced: 1, Duration: time.Second}, "tick", cycleLoadSnapshot{})

	s.snapshotMu.RLock()
	after := s.cycleRanSinceBoot
	s.snapshotMu.RUnlock()
	if !after {
		t.Fatalf("cycleRanSinceBoot=false après un cycle effectif")
	}
}

// TestRehydrateNoStoreIsNoop : sans store câblé, la réhydratation ne panique pas.
func TestRehydrateNoStoreIsNoop(t *testing.T) {
	s := &AutoSyncScheduler{}
	s.RehydrateFromDisk(context.Background())
	s.persistSnapshot(context.Background()) // idem, no-op sans store
}

// TestRehydrateCorruptFileDegrades : un fichier corrompu est loggé et ignoré
// (démarrage sans historique, jamais de panic).
func TestRehydrateCorruptFileDegrades(t *testing.T) {
	path := filepath.Join(t.TempDir(), "post_sync_snapshot.json")
	if err := os.WriteFile(path, []byte("not json {"), 0o644); err != nil {
		t.Fatalf("seed corrompu: %v", err)
	}
	s := &AutoSyncScheduler{playerOutcomes: map[string]PlayerOutcomeDetail{}}
	s.snapshotStore = adminstate.NewFileStore(path)
	s.RehydrateFromDisk(context.Background())

	s.snapshotMu.RLock()
	defer s.snapshotMu.RUnlock()
	if len(s.playerOutcomes) != 0 || !s.lastCycleAt.IsZero() {
		t.Fatalf("fichier corrompu: état partiellement chargé (players=%d)", len(s.playerOutcomes))
	}
}
