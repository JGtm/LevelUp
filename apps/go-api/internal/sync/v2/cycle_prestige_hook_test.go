// Package v2 — cycle_prestige_hook_test.go : garde anti-régression VF-1 / DC-4.
//
// Le hook Prestige post-sync doit tourner APRÈS le post-sync de chaque joueur
// dont le post-sync a réussi (Phase 6), avec le PlayerSlug comme identifiant
// (= user_id des défis Prestige, PAS le gamertag). Ce test échoue si quelqu'un
// débranche le câblage (o.prestigeHook) ou passe le mauvais identifiant.
//
// Contexte : avant le fix, prestige.RunPostSyncHook ne tournait sur AUCUN chemin
// (le hook engine n'est pas atteint en V2 — RunPostSyncForV2 court-circuite
// engine.run()). Cf. .ai/AUDIT_VERIF_FINALE_2026-07-06.md VF-1.
package v2

import (
	"context"
	"errors"
	"sync"
	"testing"
)

// prestigeHookSpy enregistre les (playerSlug, titleSlug) reçus, thread-safe
// (Phase 6 invoque le hook depuis la goroutine du cycle, mais le post-sync est
// parallèle — on protège quand même la map).
type prestigeHookSpy struct {
	mu    sync.Mutex
	calls map[string]string // playerSlug -> titleSlug
}

func newPrestigeHookSpy() *prestigeHookSpy {
	return &prestigeHookSpy{calls: make(map[string]string)}
}

func (s *prestigeHookSpy) hook(_ context.Context, playerSlug, titleSlug string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.calls[playerSlug] = titleSlug
}

func (s *prestigeHookSpy) count() int {
	s.mu.Lock()
	defer s.mu.Unlock()
	return len(s.calls)
}

func (s *prestigeHookSpy) called(playerSlug string) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	_, ok := s.calls[playerSlug]
	return ok
}

// TestCycle_PrestigeHookFiresPerSuccessfulPlayer — le hook est invoqué une fois
// par joueur dont le post-sync a réussi, avec le PlayerSlug (pas le gamertag).
func TestCycle_PrestigeHookFiresPerSuccessfulPlayer(t *testing.T) {
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
			"alice": {CitationsComputed: 5},
			"bob":   {CitationsComputed: 3},
		},
	}
	spy := newPrestigeHookSpy()

	orch := NewCycleOrchestrator(loader, listProvider, sharedFetcher, playerEnr,
		persister, postSync, CycleConfig{}).WithPrestigeHook(spy.hook)
	if _, err := orch.Run(context.Background(), players); err != nil {
		t.Fatalf("Run err = %v", err)
	}

	if spy.count() != 2 {
		t.Fatalf("prestige hook appelé %d fois, want 2 (alice + bob)", spy.count())
	}
	// Identifiant = PlayerSlug (mkPlayers pose Gamertag = slug+"_GT" ≠ slug) :
	// si le câblage passait le gamertag, ces assertions échoueraient.
	for _, slug := range []string{"alice", "bob"} {
		if !spy.called(slug) {
			t.Errorf("prestige hook non appelé pour PlayerSlug %q (identifiant attendu = slug, pas gamertag)", slug)
		}
	}
}

// TestCycle_PrestigeHookSkippedOnPostSyncFailure — un joueur dont le post-sync
// échoue ne déclenche PAS la ré-évaluation Prestige (snapshot incohérent).
func TestCycle_PrestigeHookSkippedOnPostSyncFailure(t *testing.T) {
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
		resultFor: map[string]PlayerPostSyncResult{"bob": {CitationsComputed: 1}},
		errorFor:  map[string]error{"alice": errors.New("post-sync recover")},
	}
	spy := newPrestigeHookSpy()

	orch := NewCycleOrchestrator(loader, listProvider, sharedFetcher, playerEnr,
		persister, postSync, CycleConfig{}).WithPrestigeHook(spy.hook)
	if _, err := orch.Run(context.Background(), players); err != nil {
		t.Fatalf("Run err = %v", err)
	}

	if spy.called("alice") {
		t.Error("prestige hook appelé pour alice alors que son post-sync a échoué")
	}
	if !spy.called("bob") {
		t.Error("prestige hook NON appelé pour bob (post-sync OK)")
	}
}

// TestCycle_PrestigeHookNilNoPanic — pas de hook câblé → aucun panic, cycle OK.
func TestCycle_PrestigeHookNilNoPanic(t *testing.T) {
	players := mkPlayers("alice")
	loader := &mockLoader{known: map[string]map[string]bool{"alice": {}}}
	listProvider := &mockProvider{allMatches: map[string][]string{"alice": {"m1"}}}
	sharedFetcher := &mockFetcher{perMatchData: map[string]map[string]any{"m1": {"k": 1}}}
	playerEnr := &mockEnrichmentFetcher{dataFor: map[string]map[string]any{"alice|m1": {"a": 1}}}
	postSync := &mockPostSyncRunner{resultFor: map[string]PlayerPostSyncResult{"alice": {}}}

	orch := NewCycleOrchestrator(loader, listProvider, sharedFetcher, playerEnr,
		&mockCyclePersister{}, postSync, CycleConfig{}) // pas de WithPrestigeHook
	if _, err := orch.Run(context.Background(), players); err != nil {
		t.Fatalf("Run err = %v", err)
	}
}
