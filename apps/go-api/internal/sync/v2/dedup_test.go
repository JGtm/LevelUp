// Package v2 — dedup_test.go : tests Phase 2 (Dedup).
package v2

import (
	"sort"
	"testing"
)

func TestRunDedup_EmptyDiscovery(t *testing.T) {
	res := RunDedup(DiscoveryResult{PerPlayer: map[string][]string{}})
	if len(res.UniqueMatches) != 0 {
		t.Errorf("UniqueMatches = %v, want []", res.UniqueMatches)
	}
	if len(res.CanonicalFetcher) != 0 {
		t.Errorf("CanonicalFetcher = %v, want empty", res.CanonicalFetcher)
	}
}

func TestRunDedup_SinglePlayerNoSharedMatches(t *testing.T) {
	disc := DiscoveryResult{
		PerPlayer: map[string][]string{
			"alice": {"m1", "m2", "m3"},
		},
	}
	res := RunDedup(disc)
	if want := []string{"m1", "m2", "m3"}; !equalSlice(res.UniqueMatches, want) {
		t.Errorf("UniqueMatches = %v, want %v", res.UniqueMatches, want)
	}
	for _, mID := range res.UniqueMatches {
		if got := res.CanonicalFetcher[mID]; got != "alice" {
			t.Errorf("CanonicalFetcher[%s] = %s, want alice", mID, got)
		}
	}
}

func TestRunDedup_TwoPlayersAllSharedMatches(t *testing.T) {
	// alice et bob ont les mêmes 4 matchs. Le load balancing doit
	// répartir 2-2 entre eux.
	disc := DiscoveryResult{
		PerPlayer: map[string][]string{
			"alice": {"m1", "m2", "m3", "m4"},
			"bob":   {"m1", "m2", "m3", "m4"},
		},
	}
	res := RunDedup(disc)
	if len(res.UniqueMatches) != 4 {
		t.Fatalf("UniqueMatches len = %d, want 4 (dedup)", len(res.UniqueMatches))
	}
	// Compter le workload par joueur
	workload := map[string]int{}
	for _, mID := range res.UniqueMatches {
		workload[res.CanonicalFetcher[mID]]++
	}
	if workload["alice"] != 2 || workload["bob"] != 2 {
		t.Errorf("workload mal réparti : %v (want alice=2, bob=2)", workload)
	}
}

func TestRunDedup_PartialOverlap(t *testing.T) {
	// alice : m1, m2, m3 (m1 et m2 partagés avec bob)
	// bob   : m1, m2, m4
	// Unique : m1, m2, m3, m4 (4 total)
	disc := DiscoveryResult{
		PerPlayer: map[string][]string{
			"alice": {"m1", "m2", "m3"},
			"bob":   {"m1", "m2", "m4"},
		},
	}
	res := RunDedup(disc)
	if want := []string{"m1", "m2", "m3", "m4"}; !equalSlice(res.UniqueMatches, want) {
		t.Errorf("UniqueMatches = %v, want %v", res.UniqueMatches, want)
	}
	// Vérifier participants
	if got := res.ParticipantsByMatch["m1"]; !equalSlice(got, []string{"alice", "bob"}) {
		t.Errorf("ParticipantsByMatch[m1] = %v, want [alice bob]", got)
	}
	if got := res.ParticipantsByMatch["m3"]; !equalSlice(got, []string{"alice"}) {
		t.Errorf("ParticipantsByMatch[m3] = %v, want [alice]", got)
	}
}

func TestRunDedup_CanonicalFetcherIsParticipant(t *testing.T) {
	// Property : pour chaque match, CanonicalFetcher est dans ParticipantsByMatch.
	disc := DiscoveryResult{
		PerPlayer: map[string][]string{
			"alice":   {"m1", "m2", "m3"},
			"bob":     {"m2", "m3", "m4"},
			"charlie": {"m3", "m4", "m5"},
		},
	}
	res := RunDedup(disc)
	for _, mID := range res.UniqueMatches {
		fetcher := res.CanonicalFetcher[mID]
		participants := res.ParticipantsByMatch[mID]
		found := false
		for _, p := range participants {
			if p == fetcher {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("match %s : fetcher %s pas dans participants %v", mID, fetcher, participants)
		}
	}
}

func TestRunDedup_DeterministicAcrossRuns(t *testing.T) {
	// Property : 2 runs identiques produisent le même output exact.
	disc := DiscoveryResult{
		PerPlayer: map[string][]string{
			"alice":   {"m3", "m1", "m5"},
			"charlie": {"m1", "m5", "m4"},
			"bob":     {"m2", "m1", "m3"},
		},
	}
	r1 := RunDedup(disc)
	r2 := RunDedup(disc)
	if !equalSlice(r1.UniqueMatches, r2.UniqueMatches) {
		t.Errorf("UniqueMatches differ : r1=%v r2=%v", r1.UniqueMatches, r2.UniqueMatches)
	}
	for mID := range r1.CanonicalFetcher {
		if r1.CanonicalFetcher[mID] != r2.CanonicalFetcher[mID] {
			t.Errorf("CanonicalFetcher[%s] differ : r1=%s r2=%s",
				mID, r1.CanonicalFetcher[mID], r2.CanonicalFetcher[mID])
		}
	}
}

func TestRunDedup_PropertyCountUniqueLessOrEqualSumUnknown(t *testing.T) {
	// Property cardinale : count(unique) <= sum(unknownByPlayer).
	// Couvre la garantie de dedup.
	cases := []DiscoveryResult{
		{PerPlayer: map[string][]string{"a": {"m1"}, "b": {"m1"}}},             // overlap total
		{PerPlayer: map[string][]string{"a": {"m1"}, "b": {"m2"}}},             // disjoint
		{PerPlayer: map[string][]string{"a": {"m1", "m2"}, "b": {"m2", "m3"}}}, // partiel
		{PerPlayer: map[string][]string{}},                                     // vide
		{PerPlayer: map[string][]string{"solo": {"m1", "m2", "m3", "m4"}}},     // solo
	}
	for i, disc := range cases {
		res := RunDedup(disc)
		var sum int
		for _, ms := range disc.PerPlayer {
			sum += len(ms)
		}
		if len(res.UniqueMatches) > sum {
			t.Errorf("case %d : count(unique)=%d > sum(unknown)=%d", i, len(res.UniqueMatches), sum)
		}
	}
}

func TestRunDedup_UniqueMatchesSorted(t *testing.T) {
	disc := DiscoveryResult{
		PerPlayer: map[string][]string{
			"alice": {"zzz", "aaa", "mmm"},
		},
	}
	res := RunDedup(disc)
	sorted := make([]string, len(res.UniqueMatches))
	copy(sorted, res.UniqueMatches)
	sort.Strings(sorted)
	if !equalSlice(res.UniqueMatches, sorted) {
		t.Errorf("UniqueMatches pas trié : %v", res.UniqueMatches)
	}
}

func TestRunDedup_ParticipantsSorted(t *testing.T) {
	disc := DiscoveryResult{
		PerPlayer: map[string][]string{
			"zoe":   {"m1"},
			"alice": {"m1"},
			"max":   {"m1"},
		},
	}
	res := RunDedup(disc)
	got := res.ParticipantsByMatch["m1"]
	sorted := []string{"alice", "max", "zoe"}
	if !equalSlice(got, sorted) {
		t.Errorf("ParticipantsByMatch[m1] = %v, want %v (trié)", got, sorted)
	}
}

func TestRunDedup_IgnoresErrorsField(t *testing.T) {
	// Les joueurs qui ont échoué Phase 1 (présents dans Errors) ne doivent
	// pas apparaître dans participants. Phase 2 ne lit que PerPlayer.
	disc := DiscoveryResult{
		PerPlayer: map[string][]string{
			"alice": {"m1"},
		},
		Errors: map[string]error{
			"bob": errInjected("phase 1 fail"),
		},
	}
	res := RunDedup(disc)
	if res.CanonicalFetcher["m1"] != "alice" {
		t.Errorf("CanonicalFetcher[m1] = %s, want alice (bob exclu car erreur)", res.CanonicalFetcher["m1"])
	}
}

// errInjected helper local pour ne pas dépendre d'errors.New dans le test
// (au cas où on veut une sentinel typée plus tard).
type errInjected string

func (e errInjected) Error() string { return string(e) }
