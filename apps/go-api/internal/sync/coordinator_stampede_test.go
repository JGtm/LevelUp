package sync

import (
	"context"
	"sync/atomic"
	"testing"
	"time"
)

// TestStampede_CrossSource_SingleSync simule les 3 sources de sync convergeant
// sur le MÊME joueur quasi-simultanément : le watcher (Submit) lance le sync,
// l'auto-sync et le HTTP (TryClaim) tentent pendant qu'il tourne. Vérifie :
//   - une seule source exécute réellement le RunSync (pas de double fetch) ;
//   - les tentatives auto/HTTP cèdent (TryClaim → !ok), casse incluse ;
//   - le claim est libéré à la fin → une nouvelle source peut l'obtenir.
//
// Sert de garde-fou anti-régression de la dédup cross-source unifiée 2026-06-02.
// À lancer sous -race. Le claim ne prend jamais le leaseMutex (cf. godoc
// coordinator.go) → aucun cycle claim↔lease, donc pas de deadlock à simuler ici.
func TestStampede_CrossSource_SingleSync(t *testing.T) {
	runner := &mockRunner{delay: 150 * time.Millisecond}
	coord := NewCoordinator(runner, 4)

	var completed atomic.Int32
	coord.SetOnComplete(func(_ string, _ error) { completed.Add(1) })

	// 1. Le watcher déclenche le sync (pose le claim + lance run()).
	if !coord.Submit(context.Background(), CoordinatorRequest{Gamertag: "Madina97294", XUID: "x", MatchIDs: []string{"m1"}}) {
		t.Fatal("Submit watcher devrait réussir")
	}
	time.Sleep(20 * time.Millisecond) // laisser run() démarrer et poser le claim

	// 2. Pendant le sync watcher : auto-sync + HTTP tentent le même joueur → cèdent.
	if _, ok := coord.TryClaim("Madina97294"); ok {
		t.Error("auto-sync ne devrait PAS obtenir le claim pendant le sync watcher")
	}
	if _, ok := coord.TryClaim("madina97294"); ok { // casse différente → même clé normalisée
		t.Error("HTTP ne devrait PAS obtenir le claim (clé normalisée)")
	}

	// 3. Attendre la fin du sync watcher.
	deadline := time.Now().Add(2 * time.Second)
	for completed.Load() == 0 && time.Now().Before(deadline) {
		time.Sleep(5 * time.Millisecond)
	}
	if completed.Load() != 1 {
		t.Fatalf("exactement 1 sync devrait s'exécuter, completed=%d", completed.Load())
	}
	if runner.callCount.Load() != 1 {
		t.Errorf("RunSync appelé %d fois, want 1 (auto/HTTP n'exécutent rien)", runner.callCount.Load())
	}

	// 4. Le claim est libéré → une nouvelle source peut l'obtenir.
	if coord.IsInFlight("Madina97294") {
		t.Error("le claim devrait être libéré après la fin du sync")
	}
	release, ok := coord.TryClaim("Madina97294")
	if !ok {
		t.Fatal("TryClaim devrait réussir après libération du claim watcher")
	}
	release()
}
