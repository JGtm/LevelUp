// Package v2 — cycle_test.go : test d'intégration logique du
// CycleOrchestratorImpl avec mocks des 6 dépendances.
//
// Vérifie le wiring complet des 6 phases sans I/O réel :
//   - Discovery + Dedup → unknown matches dispatchés.
//   - FetchShared appelé pour chaque match unique avec le bon fetcher.
//   - FetchPlayer appelé pour chaque (player, match) participant.
//   - Persist appelé UNE seule fois avec le mega-batch.
//   - PostSync appelé en parallèle pour chaque joueur.
package v2

import (
	"context"
	"errors"
	"testing"
	"time"
)

// ─── Mock fetcher implémentant les 5 interfaces nécessaires ────────────

// (les mocks individuels — mockLoader, mockProvider, mockFetcher,
// mockEnrichmentFetcher, mockCyclePersister, mockPostSyncRunner — sont
// définis dans discovery_test.go, fetch_shared_test.go,
// fetch_player_test.go, persist_test.go et post_sync_test.go. On les
// réutilise ici.)

// ─── Tests d'intégration ─────────────────────────────────────────────

func TestCycle_FullPipelineHappyPath(t *testing.T) {
	// Scenario : alice et bob jouent ensemble 2 matchs partagés
	// (m1, m2), plus chacun a 1 match solo (m3 alice, m4 bob).
	// Total unique : 4 matchs.
	players := mkPlayers("alice", "bob")
	playerBy := mkPlayerMap("alice", "bob")

	loader := &mockLoader{
		known: map[string]map[string]bool{
			"alice": {},
			"bob":   {},
		},
	}
	listProvider := &mockProvider{
		allMatches: map[string][]string{
			"alice": {"m1", "m2", "m3"},
			"bob":   {"m1", "m2", "m4"},
		},
	}
	sharedFetcher := &mockFetcher{
		perMatchData: map[string]map[string]any{
			"m1": {"k": "v1"}, "m2": {"k": "v2"},
			"m3": {"k": "v3"}, "m4": {"k": "v4"},
		},
	}
	playerEnr := &mockEnrichmentFetcher{
		dataFor: map[string]map[string]any{
			"alice|m1": {"awards": 1}, "alice|m2": {"awards": 2}, "alice|m3": {"awards": 3},
			"bob|m1": {"awards": 1}, "bob|m2": {"awards": 2}, "bob|m4": {"awards": 4},
		},
	}
	persister := &mockCyclePersister{}
	postSync := &mockPostSyncRunner{
		resultFor: map[string]PlayerPostSyncResult{
			"alice": {CitationsComputed: 5},
			"bob":   {CitationsComputed: 3},
		},
	}

	orch := NewCycleOrchestrator(loader, listProvider, sharedFetcher, playerEnr, persister, postSync, CycleConfig{})
	res, err := orch.Run(context.Background(), players)

	if err != nil {
		t.Fatalf("Run err = %v", err)
	}

	// Phase 1+2 : 4 matchs uniques (m1..m4)
	if res.UniqueMatches != 4 {
		t.Errorf("UniqueMatches = %d, want 4", res.UniqueMatches)
	}

	// Phase 3 : 4 fetchs (1 par match unique, pas 6 = 3+3 joueurs)
	if int(sharedFetcher.totalCalls.Load()) != 4 {
		t.Errorf("sharedFetcher.totalCalls = %d, want 4 (cross-player dedup)", sharedFetcher.totalCalls.Load())
	}

	// Phase 4 : 6 fetchs total (alice: m1,m2,m3 + bob: m1,m2,m4)
	if got := playerEnr.totalCallsFor("alice") + playerEnr.totalCallsFor("bob"); got != 6 {
		t.Errorf("playerEnr total = %d, want 6 (3 alice + 3 bob)", got)
	}

	// Phase 5 : EXACTLY 1 call persister (mega-batch invariant)
	if persister.callCount() != 1 {
		t.Errorf("persister callCount = %d, want 1 (mega-batch)", persister.callCount())
	}
	batch := persister.lastBatch()
	if len(batch.Matches) != 4 {
		t.Errorf("batch.Matches len = %d, want 4", len(batch.Matches))
	}

	// Phase 6 : 1 post-sync par joueur
	if int(postSync.totalCalls.Load()) != 2 {
		t.Errorf("postSync.totalCalls = %d, want 2", postSync.totalCalls.Load())
	}
	if res.PerPlayer["alice"].Status != "ok" {
		t.Errorf("alice status = %q, want ok", res.PerPlayer["alice"].Status)
	}

	// PhaseDurations populées pour les 6 phases
	for _, ph := range []string{PhaseDiscovery, PhaseDedup, PhaseFetchShared, PhaseFetchPlayer, PhasePersist, PhasePostSync} {
		if _, ok := res.PhaseDurations[ph]; !ok {
			t.Errorf("PhaseDurations[%q] absent", ph)
		}
	}

	_ = playerBy // playerBy n'est pas utilisé directement, le orchestrator le construit en interne
}

func TestCycle_EmptyPlayers(t *testing.T) {
	orch := NewCycleOrchestrator(
		&mockLoader{}, &mockProvider{}, &mockFetcher{},
		&mockEnrichmentFetcher{}, &mockCyclePersister{}, &mockPostSyncRunner{},
		CycleConfig{},
	)
	res, err := orch.Run(context.Background(), nil)
	if err != nil {
		t.Fatalf("err = %v", err)
	}
	if len(res.PerPlayer) != 0 {
		t.Errorf("PerPlayer non vide : %v", res.PerPlayer)
	}
}

func TestCycle_PhaseDiscoveryFailureProperlyCaptured(t *testing.T) {
	// alice load known fail → status failed, autres OK.
	players := mkPlayers("alice", "bob")
	loader := &mockLoader{
		known: map[string]map[string]bool{
			"bob": {},
		},
		failFor: map[string]error{
			"alice": errors.New("db lock"),
		},
	}
	listProvider := &mockProvider{
		allMatches: map[string][]string{
			"bob": {"m1"},
		},
	}
	orch := NewCycleOrchestrator(loader, listProvider,
		&mockFetcher{perMatchData: map[string]map[string]any{"m1": {"k": 1}}},
		&mockEnrichmentFetcher{},
		&mockCyclePersister{},
		&mockPostSyncRunner{},
		CycleConfig{},
	)
	res, err := orch.Run(context.Background(), players)
	if err != nil {
		t.Fatalf("err = %v", err)
	}
	if res.PerPlayer["alice"].Status != "failed" {
		t.Errorf("alice status = %q, want failed", res.PerPlayer["alice"].Status)
	}
	if res.PerPlayer["alice"].FirstError == "" {
		t.Errorf("alice FirstError should be set")
	}
	if res.PerPlayer["bob"].Status != "ok" {
		t.Errorf("bob status = %q, want ok", res.PerPlayer["bob"].Status)
	}
}

func TestCycle_PhasePersistGlobalErrorAbortsPostSync(t *testing.T) {
	// Persist échoue → ne lance pas post-sync, tous les joueurs en failed.
	players := mkPlayers("alice")
	loader := &mockLoader{known: map[string]map[string]bool{"alice": {}}}
	listProvider := &mockProvider{allMatches: map[string][]string{"alice": {"m1"}}}
	sharedFetcher := &mockFetcher{perMatchData: map[string]map[string]any{"m1": {"k": 1}}}
	playerEnr := &mockEnrichmentFetcher{dataFor: map[string]map[string]any{"alice|m1": {"awards": 1}}}
	persister := &mockCyclePersister{err: errors.New("shared write timeout")}
	postSync := &mockPostSyncRunner{}

	orch := NewCycleOrchestrator(loader, listProvider, sharedFetcher, playerEnr, persister, postSync, CycleConfig{})
	res, err := orch.Run(context.Background(), players)
	if err == nil {
		t.Fatal("Run should return err on persist failure")
	}
	if int(postSync.totalCalls.Load()) != 0 {
		t.Errorf("postSync should NOT have been called (persist failed), got %d calls", postSync.totalCalls.Load())
	}
	if res.PerPlayer["alice"].Status != "failed" {
		t.Errorf("alice status = %q, want failed", res.PerPlayer["alice"].Status)
	}
}

func TestCycle_PhasePostSyncPartialFailure(t *testing.T) {
	// Post-sync échoue pour alice mais Phase 1-5 OK → status = "partial".
	players := mkPlayers("alice", "bob")
	loader := &mockLoader{known: map[string]map[string]bool{"alice": {}, "bob": {}}}
	listProvider := &mockProvider{allMatches: map[string][]string{
		"alice": {"m1"}, "bob": {"m2"},
	}}
	sharedFetcher := &mockFetcher{perMatchData: map[string]map[string]any{
		"m1": {"k": 1}, "m2": {"k": 2},
	}}
	playerEnr := &mockEnrichmentFetcher{dataFor: map[string]map[string]any{
		"alice|m1": {"a": 1}, "bob|m2": {"a": 2},
	}}
	persister := &mockCyclePersister{}
	postSync := &mockPostSyncRunner{
		resultFor: map[string]PlayerPostSyncResult{
			"bob": {CitationsComputed: 1},
		},
		errorFor: map[string]error{
			"alice": errors.New("post-sync recover"),
		},
	}

	orch := NewCycleOrchestrator(loader, listProvider, sharedFetcher, playerEnr, persister, postSync, CycleConfig{})
	res, _ := orch.Run(context.Background(), players)

	if res.PerPlayer["alice"].Status != "partial" {
		t.Errorf("alice status = %q, want partial (post-sync failed but rest OK)", res.PerPlayer["alice"].Status)
	}
	if res.PerPlayer["bob"].Status != "ok" {
		t.Errorf("bob status = %q, want ok", res.PerPlayer["bob"].Status)
	}
}

func TestCycle_NoMatchesNoPersistCall(t *testing.T) {
	// Tous les joueurs sont à jour (no new matches) → pas d'appel persister
	// (RunPersist skip si inputs vides).
	players := mkPlayers("alice")
	loader := &mockLoader{known: map[string]map[string]bool{"alice": {"m_old": true}}}
	listProvider := &mockProvider{allMatches: map[string][]string{
		"alice": {"m_old"}, // déjà connu → liste vide après filtre
	}}
	persister := &mockCyclePersister{}
	postSync := &mockPostSyncRunner{}

	orch := NewCycleOrchestrator(loader, listProvider,
		&mockFetcher{}, &mockEnrichmentFetcher{}, persister, postSync, CycleConfig{})
	res, err := orch.Run(context.Background(), players)
	if err != nil {
		t.Fatalf("err = %v", err)
	}
	if persister.callCount() != 0 {
		t.Errorf("persister called inutilement : %d", persister.callCount())
	}
	if res.UniqueMatches != 0 {
		t.Errorf("UniqueMatches = %d, want 0", res.UniqueMatches)
	}
	// PostSync est toujours appelé pour les heals (même sans nouveaux matchs).
	if int(postSync.totalCalls.Load()) != 1 {
		t.Errorf("postSync should be called for alice (heals)")
	}
}

func TestCycle_CycleConfigDefaults(t *testing.T) {
	orch := NewCycleOrchestrator(
		&mockLoader{}, &mockProvider{}, &mockFetcher{},
		&mockEnrichmentFetcher{}, &mockCyclePersister{}, &mockPostSyncRunner{},
		CycleConfig{},
	)
	if orch.cfg.FetchSharedParallelism != 8 {
		t.Errorf("default FetchSharedParallelism = %d, want 8", orch.cfg.FetchSharedParallelism)
	}
	if orch.cfg.FetchPlayerParallelism != 4 {
		t.Errorf("default FetchPlayerParallelism = %d, want 4", orch.cfg.FetchPlayerParallelism)
	}
	if orch.cfg.PostSyncParallelism != 0 {
		t.Errorf("default PostSyncParallelism = %d, want 0 (unlimited)", orch.cfg.PostSyncParallelism)
	}
}

func TestCycle_CycleConfigOverrides(t *testing.T) {
	orch := NewCycleOrchestrator(
		&mockLoader{}, &mockProvider{}, &mockFetcher{},
		&mockEnrichmentFetcher{}, &mockCyclePersister{}, &mockPostSyncRunner{},
		CycleConfig{
			FetchSharedParallelism: 2,
			FetchPlayerParallelism: 1,
			PostSyncParallelism:    3,
		},
	)
	if orch.cfg.FetchSharedParallelism != 2 {
		t.Errorf("FetchSharedParallelism = %d, want 2", orch.cfg.FetchSharedParallelism)
	}
	if orch.cfg.FetchPlayerParallelism != 1 {
		t.Errorf("FetchPlayerParallelism = %d, want 1", orch.cfg.FetchPlayerParallelism)
	}
}

func TestCycle_PerformanceVsSequentialLowerBound(t *testing.T) {
	// 4 joueurs, dataset moderne avec délais simulés sur chaque phase.
	// Vérifier que le wall time est cohérent avec parallélisme (pas du
	// 4× les délais individuels).
	players := mkPlayers("a", "b", "c", "d")
	loader := &mockLoader{
		known: map[string]map[string]bool{"a": {}, "b": {}, "c": {}, "d": {}},
		delay: 20 * time.Millisecond,
	}
	listProvider := &mockProvider{
		allMatches: map[string][]string{
			"a": {"m1"}, "b": {"m2"}, "c": {"m3"}, "d": {"m4"},
		},
		delay: 20 * time.Millisecond,
	}
	delays := map[string]time.Duration{
		"m1": 30 * time.Millisecond, "m2": 30 * time.Millisecond,
		"m3": 30 * time.Millisecond, "m4": 30 * time.Millisecond,
	}
	sharedFetcher := &mockFetcher{
		perMatchData:  map[string]map[string]any{"m1": {}, "m2": {}, "m3": {}, "m4": {}},
		perMatchDelay: delays,
	}
	playerEnr := &mockEnrichmentFetcher{delayAll: 20 * time.Millisecond}
	persister := &mockCyclePersister{delay: 30 * time.Millisecond}
	postSync := &mockPostSyncRunner{delayFor: map[string]time.Duration{
		"a": 50 * time.Millisecond, "b": 50 * time.Millisecond,
		"c": 50 * time.Millisecond, "d": 50 * time.Millisecond,
	}}
	orch := NewCycleOrchestrator(loader, listProvider, sharedFetcher, playerEnr, persister, postSync, CycleConfig{})

	start := time.Now()
	_, err := orch.Run(context.Background(), players)
	elapsed := time.Since(start)
	if err != nil {
		t.Fatalf("err = %v", err)
	}
	// Lower bound théorique :
	//   - P1 : ~20ms (parallel sur load + 20ms list)
	//   - P2 : ~0
	//   - P3 : ~30ms (4 matchs / 8 parallel = 1 batch)
	//   - P4 : ~20ms (1 match/player)
	//   - P5 : 30ms
	//   - P6 : 50ms (parallel sur 4 joueurs)
	// Total séquentiel : ~150-200ms attendu (marge généreuse pour CI).
	// Si sequentiel-total des délais : 20+20+30+20+30+50 = 170ms minimum.
	if elapsed > 500*time.Millisecond {
		t.Errorf("cycle too slow: %v (want < 500ms, parallelism not effective)", elapsed)
	}
}
