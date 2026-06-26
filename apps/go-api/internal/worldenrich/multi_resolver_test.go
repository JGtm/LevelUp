package worldenrich

import (
	"context"
	"errors"
	"strings"
	"testing"
)

// fakeResolver — XUIDResolver de test : retourne soit un xuid, soit une erreur.
type fakeResolver struct {
	xuid string
	err  error
	hits int
}

func (f *fakeResolver) ResolveXUID(_ context.Context, _ string) (string, error) {
	f.hits++
	if f.err != nil {
		return "", f.err
	}
	return f.xuid, nil
}

func TestChainResolver_FallbackToSecond(t *testing.T) {
	// 1er (PeopleHub) échoue (gamertag hors graphe social) → 2e (profil) résout.
	first := &fakeResolver{err: errors.New("gamertag absent des résultats peoplehub")}
	second := &fakeResolver{xuid: "2533274895653213"}
	chain := chainResolver{resolvers: []XUIDResolver{first, second}}

	got, err := chain.ResolveXUID(context.Background(), "RandomOpponent")
	if err != nil {
		t.Fatalf("ResolveXUID: %v", err)
	}
	if got != "2533274895653213" {
		t.Errorf("xuid = %q, want 2533274895653213 (résolu par le fallback profil)", got)
	}
	if first.hits != 1 || second.hits != 1 {
		t.Errorf("hits: first=%d second=%d, want 1/1 (fallback essayé après échec PeopleHub)", first.hits, second.hits)
	}
}

func TestChainResolver_FirstWins(t *testing.T) {
	first := &fakeResolver{xuid: "111"}
	second := &fakeResolver{xuid: "222"}
	chain := chainResolver{resolvers: []XUIDResolver{first, second}}

	got, err := chain.ResolveXUID(context.Background(), "Friend")
	if err != nil {
		t.Fatalf("ResolveXUID: %v", err)
	}
	if got != "111" {
		t.Errorf("xuid = %q, want 111 (PeopleHub d'abord)", got)
	}
	if second.hits != 0 {
		t.Errorf("le fallback ne doit PAS être appelé quand le 1er résout (hits=%d)", second.hits)
	}
}

func TestChainResolver_AllThrottledPropagates429(t *testing.T) {
	first := &fakeResolver{err: errors.New("peoplehub HTTP 429: rate limited")}
	second := &fakeResolver{err: errors.New("xbox profile HTTP 429: rate limited")}
	chain := chainResolver{resolvers: []XUIDResolver{first, second}}

	_, err := chain.ResolveXUID(context.Background(), "X")
	if err == nil {
		t.Fatal("attendu une erreur quand tous les résolveurs throttlent")
	}
	// Le 429 doit rester en surface pour que le round-robin du CachingResolver rote.
	if !strings.Contains(err.Error(), "429") {
		t.Errorf("erreur = %q, doit conserver le marqueur 429", err.Error())
	}
}

func TestCachingResolver_CacheHitSkipsResolvers(t *testing.T) {
	r := &fakeResolver{xuid: "999"}
	c := NewCachingResolver([]XUIDResolver{r}, map[string]string{"Seeded": "777"}, nil)

	// Graine → cache hit, aucun appel résolveur.
	got, err := c.ResolveXUID(context.Background(), "seeded") // casse insensible
	if err != nil || got != "777" {
		t.Fatalf("seeded: got=%q err=%v, want 777", got, err)
	}
	if r.hits != 0 {
		t.Errorf("cache hit ne doit pas appeler le résolveur (hits=%d)", r.hits)
	}

	// Miss → résout via le résolveur + met en cache.
	got, err = c.ResolveXUID(context.Background(), "New")
	if err != nil || got != "999" {
		t.Fatalf("new: got=%q err=%v, want 999", got, err)
	}
	if r.hits != 1 {
		t.Errorf("miss doit appeler le résolveur une fois (hits=%d)", r.hits)
	}
	// 2e fois → cache.
	_, _ = c.ResolveXUID(context.Background(), "New")
	if r.hits != 1 {
		t.Errorf("2e résolution de New doit venir du cache (hits=%d, want 1)", r.hits)
	}
}
