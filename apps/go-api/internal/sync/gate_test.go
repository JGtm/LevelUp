package sync

import (
	"context"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

// =============================================================================
// SyncGate (dédup cross-source) — unification 2026-06-02
// =============================================================================

func TestSyncGate_TryClaim_Dedup(t *testing.T) {
	coord := NewCoordinator(&mockRunner{}, 2)

	release, ok := coord.TryClaim("Madina97294")
	if !ok {
		t.Fatal("premier TryClaim devrait réussir")
	}
	if _, ok2 := coord.TryClaim("Madina97294"); ok2 {
		t.Error("second TryClaim sur le même joueur devrait échouer (déjà en vol)")
	}
	release()
	if _, ok3 := coord.TryClaim("Madina97294"); !ok3 {
		t.Error("TryClaim après release devrait réussir")
	}
}

func TestSyncGate_Release_Idempotent(t *testing.T) {
	coord := NewCoordinator(&mockRunner{}, 2)

	release, ok := coord.TryClaim("p1")
	if !ok {
		t.Fatal("TryClaim devrait réussir")
	}
	release()
	release() // double release : no-op (sync.OnceFunc)
	release()
	if coord.IsInFlight("p1") {
		t.Error("p1 ne devrait plus être en vol après release")
	}
	// claimWG ne doit pas être passé en négatif (sinon Done() paniquerait) :
	// WaitInFlight retourne immédiatement.
	done := make(chan struct{})
	go func() { coord.WaitInFlight(); close(done) }()
	select {
	case <-done:
	case <-time.After(time.Second):
		t.Fatal("WaitInFlight devrait retourner immédiatement (aucun claim)")
	}
}

func TestSyncGate_CaseInsensitive(t *testing.T) {
	coord := NewCoordinator(&mockRunner{}, 2)

	release, ok := coord.TryClaim("JGtm")
	if !ok {
		t.Fatal("premier TryClaim devrait réussir")
	}
	defer release()
	if _, ok2 := coord.TryClaim("jgtm"); ok2 {
		t.Error("TryClaim 'jgtm' devrait être bloqué par le claim 'JGtm' (clé normalisée)")
	}
	if !coord.IsInFlight("  JGTM  ") {
		t.Error("IsInFlight devrait normaliser casse + espaces")
	}
}

func TestSyncGate_Concurrent_3sources(t *testing.T) {
	coord := NewCoordinator(&mockRunner{}, 2)

	const sources = 3
	var wins atomic.Int32
	var wg sync.WaitGroup
	releases := make(chan func(), sources)

	for i := 0; i < sources; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			if release, ok := coord.TryClaim("Chocoboflor"); ok {
				wins.Add(1)
				releases <- release
			}
		}()
	}
	wg.Wait()
	close(releases)

	if got := wins.Load(); got != 1 {
		t.Fatalf("exactement 1 source devrait gagner le claim, got %d", got)
	}
	for release := range releases {
		release()
	}
	if coord.IsInFlight("Chocoboflor") {
		t.Error("plus aucun claim ne devrait subsister après release")
	}
}

func TestSyncGate_WaitInFlight_BlocksUntilRelease(t *testing.T) {
	coord := NewCoordinator(&mockRunner{}, 2)

	release, ok := coord.TryClaim("p1")
	if !ok {
		t.Fatal("TryClaim devrait réussir")
	}

	done := make(chan struct{})
	go func() { coord.WaitInFlight(); close(done) }()

	select {
	case <-done:
		t.Fatal("WaitInFlight ne devrait pas retourner tant que le claim est tenu")
	case <-time.After(50 * time.Millisecond):
		// attendu : toujours bloqué
	}

	release()
	select {
	case <-done:
		// attendu : débloqué après release
	case <-time.After(time.Second):
		t.Fatal("WaitInFlight devrait retourner après release du claim")
	}
}

// Le claim posé par Submit (watcher) doit bloquer un TryClaim (auto/HTTP) sur le
// même joueur — c'est le cœur de la dédup cross-source.
func TestSyncGate_SubmitBlocksTryClaim(t *testing.T) {
	runner := &mockRunner{delay: 200 * time.Millisecond}
	coord := NewCoordinator(runner, 2)

	ok := coord.Submit(context.Background(), CoordinatorRequest{Gamertag: "p1", XUID: "x1", MatchIDs: []string{"m1"}})
	if !ok {
		t.Fatal("Submit watcher devrait réussir")
	}
	time.Sleep(10 * time.Millisecond) // laisser run() démarrer

	if _, ok2 := coord.TryClaim("p1"); ok2 {
		t.Error("TryClaim devrait être bloqué par le claim watcher en cours")
	}
	// Joueur différent → autorisé.
	if release, ok3 := coord.TryClaim("p2"); !ok3 {
		t.Error("TryClaim sur un autre joueur devrait réussir")
	} else {
		release()
	}
}

func TestNopSyncGate(t *testing.T) {
	var gate SyncGate = NopSyncGate{}

	gate.BeginShutdown() // no-op, ne doit pas paniquer
	release, ok := gate.TryClaim("anyone")
	if !ok {
		t.Error("NopSyncGate.TryClaim devrait toujours réussir")
	}
	release() // ne doit pas paniquer
	if gate.IsInFlight("anyone") {
		t.Error("NopSyncGate.IsInFlight devrait toujours être false")
	}
	gate.WaitInFlight() // ne doit jamais bloquer
}

// TestSyncGate_TryClaimDoesNotBlockSubmit — INVARIANT D'ASYMÉTRIE (fix behavior-1).
// Un claim auto/HTTP (TryClaim → gateClaims) ne doit PAS empêcher le watcher de
// poser son claim (Submit → inFlight) : le watcher reste prioritaire fraîcheur.
func TestSyncGate_TryClaimDoesNotBlockSubmit(t *testing.T) {
	runner := &mockRunner{delay: 100 * time.Millisecond}
	coord := NewCoordinator(runner, 2)

	// auto/HTTP réserve d'abord le joueur.
	release, ok := coord.TryClaim("Madina97294")
	if !ok {
		t.Fatal("TryClaim initial devrait réussir")
	}
	defer release()

	// Le watcher Submit le MÊME joueur → doit réussir malgré le gateClaim auto/HTTP.
	if !coord.Submit(context.Background(), CoordinatorRequest{Gamertag: "Madina97294", XUID: "x", MatchIDs: []string{"m1"}}) {
		t.Error("Submit watcher NE doit PAS être bloqué par un claim auto/HTTP (priorité watcher)")
	}
	// Symétrie inverse : tant que le watcher est en vol, un TryClaim auto/HTTP cède.
	if _, ok2 := coord.TryClaim("Madina97294"); ok2 {
		t.Error("TryClaim devrait céder au sync watcher en cours")
	}
}

// TestSyncGate_BeginShutdown_RefusesClaim — après BeginShutdown, plus aucun claim.
func TestSyncGate_BeginShutdown_RefusesClaim(t *testing.T) {
	coord := NewCoordinator(&mockRunner{}, 2)
	coord.BeginShutdown()
	if _, ok := coord.TryClaim("p1"); ok {
		t.Error("TryClaim devrait échouer après BeginShutdown")
	}
}

// TestSyncGate_WaitInFlight_NoRaceWithLateClaim — valide le fix concurrency-1 :
// après BeginShutdown, WaitInFlight() ne peut pas courir concurremment à un
// gateWG.Add (TryClaim refuse → pas d'Add). À lancer sous -race.
func TestSyncGate_WaitInFlight_NoRaceWithLateClaim(t *testing.T) {
	coord := NewCoordinator(&mockRunner{}, 2)

	// Un claim en vol au moment du shutdown.
	release, ok := coord.TryClaim("p1")
	if !ok {
		t.Fatal("TryClaim devrait réussir")
	}

	coord.BeginShutdown()

	var wg sync.WaitGroup
	wg.Add(2)
	// Goroutine A : draine.
	go func() { defer wg.Done(); coord.WaitInFlight() }()
	// Goroutine B : tente des claims tardifs concurremment (tous refusés → aucun Add).
	go func() {
		defer wg.Done()
		for i := 0; i < 100; i++ {
			if r, claimed := coord.TryClaim("p2"); claimed {
				r()
				t.Error("aucun claim ne devrait réussir après BeginShutdown")
			}
		}
	}()

	release() // débloque WaitInFlight
	wg.Wait()
}
