//go:build integration

package sync

import (
	"context"
	"database/sql"
	"testing"

	"levelup/go-api/internal/assetnames"

	_ "github.com/duckdb/duckdb-go/v2"
)

// fakeAssetFetcher implémente assetnames.Fetcher pour les tests (zéro réseau).
type fakeAssetFetcher struct {
	names map[string]string // clé assetID|lang → nom ("" = pas de nom)
	calls map[string]int    // clé assetID → nombre d'appels (toutes langues)
}

func (f *fakeAssetFetcher) FetchName(_ context.Context, _, _, assetID, _, lang string) (string, error) {
	if f.calls != nil {
		f.calls[assetID]++
	}
	return f.names[assetID+"|"+lang], nil
}

// setupSharedWithRegistry crée une shared :memory: avec match_registry minimal.
func setupSharedWithRegistry(t *testing.T) *sql.DB {
	t.Helper()
	db, err := sql.Open("duckdb", ":memory:")
	if err != nil {
		t.Fatalf("open shared: %v", err)
	}
	t.Cleanup(func() { db.Close() })
	if _, err := db.Exec(`CREATE TABLE match_registry (
		match_id VARCHAR,
		playlist_id VARCHAR, playlist_name VARCHAR, playlist_version_id VARCHAR,
		map_id VARCHAR, map_name VARCHAR, map_version_id VARCHAR,
		pair_id VARCHAR, pair_name VARCHAR, pair_version_id VARCHAR,
		game_variant_id VARCHAR, game_variant_name VARCHAR, game_variant_version_id VARCHAR)`); err != nil {
		t.Fatalf("create match_registry: %v", err)
	}
	return db
}

// TestResolveRefs_PopulatesTranslationsAndEnriches : le cœur testable peuple
// asset_translations (fr-FR + en-US) pour un asset neuf, puis
// EnrichRegistryFromMetadata résout le registry depuis ce dictionnaire — preuve
// du flux primary-write bout-en-bout (fetcher factice, sans pool/réseau).
func TestResolveRefs_PopulatesTranslationsAndEnriches(t *testing.T) {
	ctx := context.Background()
	meta := setupMetaWithTranslations(t)

	newPlaylist := "playlist-new-uuid" // absent d'asset_translations
	fetcher := &fakeAssetFetcher{
		names: map[string]string{
			newPlaylist + "|fr-FR": "Partie rapide",
			newPlaylist + "|en-US": "Quick Play",
		},
		calls: map[string]int{},
	}
	reg := &MatchRegistryRow{
		PlaylistID:        strPtrNonEmpty(newPlaylist),
		PlaylistName:      strPtrNonEmpty(newPlaylist), // UUID brut
		PlaylistVersionID: strPtrNonEmpty("v-new"),     // version requise par discovery-infiniteugc
	}
	refs := collectAssetRefsFromRegistry(reg)
	res := resolveRefs(ctx, fetcher, meta, "halo_infinite", "test", refs, 0)
	if res.Resolved != 1 {
		t.Fatalf("Resolved = %d, want 1 (%+v)", res.Resolved, res)
	}

	var n int
	if err := meta.QueryRow(`SELECT COUNT(*) FROM asset_translations WHERE asset_id = ?`, newPlaylist).Scan(&n); err != nil {
		t.Fatalf("count: %v", err)
	}
	if n != 2 {
		t.Fatalf("asset_translations rows = %d, want 2 (fr-FR + en-US)", n)
	}

	if err := EnrichRegistryFromMetadata(ctx, meta, reg); err != nil {
		t.Fatalf("enrich: %v", err)
	}
	if got := derefSyncStr(reg.PlaylistName); got != "Quick Play" {
		t.Errorf("PlaylistName = %q, want %q", got, "Quick Play")
	}
}

// TestResolveRefs_SkipsAlreadyKnown : un asset déjà présent dans asset_translations
// n'est jamais re-fetché (skip-fresh).
func TestResolveRefs_SkipsAlreadyKnown(t *testing.T) {
	ctx := context.Background()
	meta := setupMetaWithTranslations(t) // "playlist-known-uuid" déjà seedé (en-US)

	fetcher := &fakeAssetFetcher{names: map[string]string{}, calls: map[string]int{}}
	reg := &MatchRegistryRow{
		PlaylistID:        strPtrNonEmpty("playlist-known-uuid"),
		PlaylistName:      strPtrNonEmpty("playlist-known-uuid"),
		PlaylistVersionID: strPtrNonEmpty("v-known"),
	}
	refs := collectAssetRefsFromRegistry(reg)
	resolveRefs(ctx, fetcher, meta, "halo_infinite", "test", refs, 0)
	// en-US déjà frais → pas de re-fetch en-US (au plus 1 tentative fr-FR).
	if c := fetcher.calls["playlist-known-uuid"]; c > 1 {
		t.Errorf("fetch calls = %d, want <= 1 (en-US déjà frais)", c)
	}
}

// TestCollectUnresolvedRefs_AndResolve : le balayage trouve une playlist restée
// en UUID (name == id) dans match_registry et la résout — même si elle n'est dans
// aucun nouveau match (filet pour la traîne). La playlist déjà résolue (name != id)
// est exclue de la collecte.
func TestCollectUnresolvedRefs_AndResolve(t *testing.T) {
	ctx := context.Background()
	meta := setupMetaWithTranslations(t)
	shared := setupSharedWithRegistry(t)
	if _, err := shared.Exec(`INSERT INTO match_registry (match_id, playlist_id, playlist_name, playlist_version_id) VALUES
		('m1', 'pl-orphan', 'pl-orphan', 'v-orphan'),
		('m2', 'playlist-resolved', 'Quick Play', 'v-resolved')`); err != nil {
		t.Fatalf("seed registry: %v", err)
	}

	refs, err := collectUnresolvedRefs(ctx, shared, "playlist", "playlist_id", "playlist_name", "playlist_version_id")
	if err != nil {
		t.Fatalf("collect: %v", err)
	}
	if len(refs) != 1 || refs[0].AssetID != "pl-orphan" {
		t.Fatalf("collectUnresolvedRefs = %+v, want [pl-orphan] (résolu exclu)", refs)
	}

	fetcher := &fakeAssetFetcher{
		names: map[string]string{"pl-orphan|fr-FR": "Événement", "pl-orphan|en-US": "Event"},
		calls: map[string]int{},
	}
	res := resolveRefs(ctx, fetcher, meta, "halo_infinite", "sweep", refs, 500)
	if res.Resolved != 1 {
		t.Fatalf("Resolved = %d, want 1 (%+v)", res.Resolved, res)
	}
	var n int
	if err := meta.QueryRow(`SELECT COUNT(*) FROM asset_translations WHERE asset_id = 'pl-orphan'`).Scan(&n); err != nil {
		t.Fatalf("count: %v", err)
	}
	if n != 2 {
		t.Fatalf("asset_translations[pl-orphan] = %d rows, want 2", n)
	}
}

// TestResolveRefs_SkipsRefsWithoutVersion : une ref sans version_id (non fetchable
// sur discovery-infiniteugc, qui 404 sans) est écartée — ni résolue ni comptée en
// erreur — sans appeler le fetcher.
func TestResolveRefs_SkipsRefsWithoutVersion(t *testing.T) {
	ctx := context.Background()
	meta := setupMetaWithTranslations(t)
	fetcher := &fakeAssetFetcher{names: map[string]string{}, calls: map[string]int{}}
	refs := []assetnames.AssetRef{{AssetType: "pair", AssetID: "pair-no-version", VersionID: ""}}
	res := resolveRefs(ctx, fetcher, meta, "halo_infinite", "test", refs, 0)
	if res.Requested != 0 || res.Resolved != 0 || res.Errors != 0 {
		t.Fatalf("ref sans version: %+v, want tout à 0 (écartée avant fetch)", res)
	}
	if c := fetcher.calls["pair-no-version"]; c != 0 {
		t.Errorf("fetcher appelé %d fois pour une ref sans version, want 0", c)
	}
}

// TestResolveRefs_NilFetcher : fetcher nil → no-op.
func TestResolveRefs_NilFetcher(t *testing.T) {
	refs := collectAssetRefsFromRegistry(&MatchRegistryRow{PlaylistID: strPtrNonEmpty("x"), PlaylistName: strPtrNonEmpty("x")})
	res := resolveRefs(context.Background(), nil, nil, "halo_infinite", "test", refs, 0)
	if res.Requested != 0 || res.Resolved != 0 {
		t.Fatalf("nil fetcher: %+v", res)
	}
}
