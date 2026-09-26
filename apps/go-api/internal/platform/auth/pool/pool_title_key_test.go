package pool

import (
	"context"
	"testing"
)

// titledSources construit des sources taggées avec un titleSlug (Phase 1.6).
func titledSources(titleSlug string, gamertags ...string) []CredentialSource {
	out := make([]CredentialSource, len(gamertags))
	for i, gt := range gamertags {
		out[i] = CredentialSource{
			Gamertag:     gt,
			TitleSlug:    titleSlug,
			XUID:         "xuid_" + gt,
			RefreshToken: "rt_" + gt,
			Source:       "test",
		}
	}
	return out
}

// TestPool_TitleScopedKey vérifie qu'un pool mono-titre indexe ses slots par
// (titleSlug, gamertag) : le lookup pinned/HasPlayer fonctionne avec le titre
// du pool, et le pool expose ce titre.
func TestPool_TitleScopedKey(t *testing.T) {
	resolver := &testResolver{resolved: map[string]*ResolvedTokens{}}
	sources := titledSources("halo_infinite", "Alice", "Bob")

	p, err := NewPool(context.Background(), resolver, sources, PoolOptions{PerTokenRPS: 1})
	if err != nil {
		t.Fatalf("NewPool: %v", err)
	}

	if !p.HasPlayer("Alice") {
		t.Errorf("HasPlayer(Alice) = false, want true (titre du pool)")
	}
	lease, err := p.Acquire(context.Background(), PolicyPinnedPlayer, "Bob")
	if err != nil {
		t.Fatalf("Acquire(pinned Bob): %v", err)
	}
	if lease.Gamertag != "Bob" {
		t.Errorf("lease.Gamertag = %q, want Bob", lease.Gamertag)
	}
}

// TestPool_AddOrUpdateSource_CrossTitleRejected vérifie le garde-fou cœur de la
// Phase 1.6 : un pool halo_infinite refuse une source d'un autre titre — il ne
// doit jamais servir le token d'un titre étranger.
func TestPool_AddOrUpdateSource_CrossTitleRejected(t *testing.T) {
	resolver := &testResolver{resolved: map[string]*ResolvedTokens{}}
	p, err := NewPool(context.Background(), resolver, titledSources("halo_infinite", "Alice"), PoolOptions{PerTokenRPS: 1})
	if err != nil {
		t.Fatalf("NewPool: %v", err)
	}

	foreign := CredentialSource{Gamertag: "Eve", TitleSlug: "other_game", XUID: "x", RefreshToken: "rt", Source: "test"}
	if err := p.AddOrUpdateSource(context.Background(), foreign); err == nil {
		t.Fatalf("AddOrUpdateSource cross-title: attendu une erreur, got nil")
	}
	if p.HasPlayer("Eve") {
		t.Errorf("HasPlayer(Eve) = true après rejet cross-title, want false")
	}
}

// TestPool_AddOrUpdateSource_SameTitleOK vérifie que le hot-add d'une source du
// même titre passe et devient résoluble.
func TestPool_AddOrUpdateSource_SameTitleOK(t *testing.T) {
	resolver := &testResolver{resolved: map[string]*ResolvedTokens{}}
	p, err := NewPool(context.Background(), resolver, titledSources("halo_infinite", "Alice"), PoolOptions{PerTokenRPS: 1})
	if err != nil {
		t.Fatalf("NewPool: %v", err)
	}

	add := CredentialSource{Gamertag: "Carol", TitleSlug: "halo_infinite", XUID: "x", RefreshToken: "rt", Source: "test"}
	if err := p.AddOrUpdateSource(context.Background(), add); err != nil {
		t.Fatalf("AddOrUpdateSource same-title: %v", err)
	}
	if !p.HasPlayer("Carol") {
		t.Errorf("HasPlayer(Carol) = false après hot-add same-title, want true")
	}
}
