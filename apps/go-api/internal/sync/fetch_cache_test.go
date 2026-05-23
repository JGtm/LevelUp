// Package sync — fetch_cache_test.go : tests TDD pour cachedHaloClient.

package sync

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"
)

// ─── Helper : mock HaloClient compteur d'appels ────────────────────────────

type countingMockClient struct {
	*mockHaloClient
	statsCalls int
	skillCalls int
	chunkCalls int
}

func (c *countingMockClient) GetMatchStats(ctx context.Context, matchID string) (map[string]any, error) {
	c.statsCalls++
	return c.mockHaloClient.GetMatchStats(ctx, matchID)
}

func (c *countingMockClient) GetMatchSkill(ctx context.Context, matchID string, xuids []string) (map[string]*MatchSkillData, error) {
	c.skillCalls++
	return c.mockHaloClient.GetMatchSkill(ctx, matchID, xuids)
}

func (c *countingMockClient) GetHighlightEventsChunk(ctx context.Context, matchID string) ([]byte, int, bool, error) {
	c.chunkCalls++
	return c.mockHaloClient.GetHighlightEventsChunk(ctx, matchID)
}

// ─── Test 1 : cache hit → 0 appel API au 2e fetch ─────────────────────────

func TestCachedHaloClient_GetMatchStats_CachesAndReusesResponse(t *testing.T) {
	dir := t.TempDir()
	inner := &countingMockClient{
		mockHaloClient: &mockHaloClient{
			statsBody: map[string]map[string]any{
				"m1": {"MatchId": "m1", "Players": []any{}},
			},
		},
	}
	cached := NewCachedHaloClient(inner, FetchCacheConfig{CacheDir: dir}).(*cachedHaloClient)

	// 1er appel : API hit + write cache
	r1, err := cached.GetMatchStats(context.Background(), "m1")
	if err != nil {
		t.Fatal(err)
	}
	if r1["MatchId"] != "m1" {
		t.Errorf("r1.MatchId = %v, want m1", r1["MatchId"])
	}
	if inner.statsCalls != 1 {
		t.Errorf("inner.statsCalls = %d, want 1", inner.statsCalls)
	}

	// Fichier cache écrit
	cachePath := filepath.Join(dir, "match_m1_stats.json")
	if _, err := os.Stat(cachePath); err != nil {
		t.Errorf("cache file absent post-write: %v", err)
	}

	// 2e appel : cache hit → pas de nouveau call API
	r2, err := cached.GetMatchStats(context.Background(), "m1")
	if err != nil {
		t.Fatal(err)
	}
	if r2["MatchId"] != "m1" {
		t.Errorf("r2.MatchId = %v, want m1", r2["MatchId"])
	}
	if inner.statsCalls != 1 {
		t.Errorf("inner.statsCalls = %d, want 1 (cache hit), got %d", inner.statsCalls, inner.statsCalls)
	}
}

// ─── Test 2 : cache GetMatchSkill ─────────────────────────────────────────

func TestCachedHaloClient_GetMatchSkill_Cached(t *testing.T) {
	dir := t.TempDir()
	inner := &countingMockClient{
		mockHaloClient: &mockHaloClient{
			skillBody: map[string]map[string]*MatchSkillData{
				"m1": {"xuid1": {}},
			},
		},
	}
	cached := NewCachedHaloClient(inner, FetchCacheConfig{CacheDir: dir})

	_, _ = cached.GetMatchSkill(context.Background(), "m1", []string{"xuid1"})
	_, _ = cached.GetMatchSkill(context.Background(), "m1", []string{"xuid1"})
	if inner.skillCalls != 1 {
		t.Errorf("skillCalls = %d, want 1 (cache hit)", inner.skillCalls)
	}
}

// ─── Test 3 : Disabled config → pass-through, pas de cache ────────────────

func TestCachedHaloClient_Disabled_PassesThrough(t *testing.T) {
	inner := &countingMockClient{
		mockHaloClient: &mockHaloClient{
			statsBody: map[string]map[string]any{"m1": {"MatchId": "m1"}},
		},
	}
	// Disabled=true → wrapping retourne directement inner
	client := NewCachedHaloClient(inner, FetchCacheConfig{Disabled: true})
	if client != inner {
		t.Error("NewCachedHaloClient(disabled) doit retourner inner directement")
	}
}

// ─── Test 4 : CacheDir vide → pass-through ────────────────────────────────

func TestCachedHaloClient_EmptyCacheDir_PassesThrough(t *testing.T) {
	inner := &countingMockClient{mockHaloClient: &mockHaloClient{}}
	client := NewCachedHaloClient(inner, FetchCacheConfig{CacheDir: ""})
	if client != inner {
		t.Error("NewCachedHaloClient(empty dir) doit retourner inner directement")
	}
}

// ─── Test 5 : cache corrompu → re-fetch + overwrite ───────────────────────

func TestCachedHaloClient_CorruptedCache_RefetchesAndOverwrites(t *testing.T) {
	dir := t.TempDir()
	// Pré-seed un cache corrompu
	cachePath := filepath.Join(dir, "match_m1_stats.json")
	if err := os.WriteFile(cachePath, []byte("{not json"), 0o644); err != nil {
		t.Fatal(err)
	}
	inner := &countingMockClient{
		mockHaloClient: &mockHaloClient{
			statsBody: map[string]map[string]any{"m1": {"MatchId": "m1", "fresh": true}},
		},
	}
	cached := NewCachedHaloClient(inner, FetchCacheConfig{CacheDir: dir})

	r, err := cached.GetMatchStats(context.Background(), "m1")
	if err != nil {
		t.Fatal(err)
	}
	if r["fresh"] != true {
		t.Errorf("cache corrompu → devrait avoir refetch et obtenu data fresh, got %+v", r)
	}
	if inner.statsCalls != 1 {
		t.Errorf("inner.statsCalls = %d, want 1 (refetch sur cache corrompu)", inner.statsCalls)
	}

	// Le cache devrait avoir été ré-écrit (valide JSON)
	data, _ := os.ReadFile(cachePath)
	var out map[string]any
	if err := json.Unmarshal(data, &out); err != nil {
		t.Errorf("cache file pas re-écrit en JSON valide post-corruption: %v", err)
	}
}

// ─── Test 6 : PurgeOldFetchCache ──────────────────────────────────────────

func TestPurgeOldFetchCache_RemovesOldCycleDirs(t *testing.T) {
	root := t.TempDir()
	// Crée 2 sous-dossiers : un vieux + un récent
	oldDir := filepath.Join(root, "cycle_old")
	if err := os.MkdirAll(oldDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(oldDir, "match_x_stats.json"), []byte(`{}`), 0o644); err != nil {
		t.Fatal(err)
	}
	old := time.Now().Add(-10 * 24 * time.Hour)
	_ = os.Chtimes(oldDir, old, old)

	recentDir := filepath.Join(root, "cycle_recent")
	if err := os.MkdirAll(recentDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(recentDir, "match_y_stats.json"), []byte(`{}`), 0o644); err != nil {
		t.Fatal(err)
	}

	purged, err := PurgeOldFetchCache(root, 7*24*time.Hour)
	if err != nil {
		t.Fatalf("PurgeOldFetchCache: %v", err)
	}
	if purged != 1 {
		t.Errorf("purged = %d, want 1", purged)
	}

	if _, err := os.Stat(oldDir); !os.IsNotExist(err) {
		t.Errorf("vieux cycle dir devrait être supprimé, err=%v", err)
	}
	if _, err := os.Stat(recentDir); err != nil {
		t.Errorf("recent cycle dir doit être préservé, err=%v", err)
	}
}

func TestPurgeOldFetchCache_NonExistentDir_NoError(t *testing.T) {
	purged, err := PurgeOldFetchCache(filepath.Join(t.TempDir(), "missing"), 7*24*time.Hour)
	if err != nil {
		t.Errorf("dir absent doit pas être une erreur, got %v", err)
	}
	if purged != 0 {
		t.Errorf("purged = %d, want 0", purged)
	}
}
