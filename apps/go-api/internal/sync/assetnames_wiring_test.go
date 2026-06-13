//go:build integration

package sync

import (
	"context"
	"testing"

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

// TestResolveCycleAssets_PopulatesTranslationsAndEnriches : le pré-pass peuple
// asset_translations (fr-FR + en-US) pour un asset neuf, puis
// EnrichRegistryFromMetadata résout le registry depuis ce dictionnaire — preuve
// du flux primary-write bout-en-bout.
func TestResolveCycleAssets_PopulatesTranslationsAndEnriches(t *testing.T) {
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
	e := &SyncEngine{gamertag: "Tester", titleSlug: "halo_infinite", metaDB: meta, assetFetcher: fetcher}

	reg := &MatchRegistryRow{
		PlaylistID:   strPtr(newPlaylist),
		PlaylistName: strPtr(newPlaylist), // UUID brut
	}
	e.resolveCycleAssets(ctx, []*fetchedMatch{{MatchID: "m1", Registry: reg}})

	// asset_translations doit contenir les 2 langues du nouvel asset.
	var n int
	if err := meta.QueryRow(
		`SELECT COUNT(*) FROM asset_translations WHERE asset_id = ?`, newPlaylist,
	).Scan(&n); err != nil {
		t.Fatalf("count: %v", err)
	}
	if n != 2 {
		t.Fatalf("asset_translations rows = %d, want 2 (fr-FR + en-US)", n)
	}

	// EnrichRegistryFromMetadata résout désormais le registry depuis le dico
	// fraîchement peuplé (en-US prioritaire pour le nom canonique stocké).
	if err := EnrichRegistryFromMetadata(ctx, meta, reg); err != nil {
		t.Fatalf("enrich: %v", err)
	}
	if got := derefSyncStr(reg.PlaylistName); got != "Quick Play" {
		t.Errorf("PlaylistName = %q, want %q", got, "Quick Play")
	}
}

// TestResolveCycleAssets_SkipsAlreadyKnown : un asset déjà présent dans
// asset_translations n'est jamais re-fetché (skip-fresh).
func TestResolveCycleAssets_SkipsAlreadyKnown(t *testing.T) {
	ctx := context.Background()
	meta := setupMetaWithTranslations(t) // "playlist-known-uuid" déjà seedé (en-US)

	fetcher := &fakeAssetFetcher{names: map[string]string{}, calls: map[string]int{}}
	e := &SyncEngine{gamertag: "Tester", titleSlug: "halo_infinite", metaDB: meta, assetFetcher: fetcher}

	reg := &MatchRegistryRow{
		PlaylistID:   strPtr("playlist-known-uuid"),
		PlaylistName: strPtr("playlist-known-uuid"), // UUID brut → candidat, mais déjà dans le dico (en-US)
	}
	e.resolveCycleAssets(ctx, []*fetchedMatch{{MatchID: "m1", Registry: reg}})

	// en-US était frais → pas de fetch en-US. fr-FR absent → 1 tentative fr-FR
	// (le fake renvoie "" → rien d'écrit). On vérifie surtout qu'aucun fetch
	// en-US n'a eu lieu (skip-fresh effectif) : le seul appel possible est fr-FR.
	if c := fetcher.calls["playlist-known-uuid"]; c > 1 {
		t.Errorf("fetch calls = %d, want <= 1 (en-US déjà frais → pas re-fetché)", c)
	}
}

// TestResolveCycleAssets_Disabled : assetFetcher nil → no-op total (parité legacy).
func TestResolveCycleAssets_Disabled(t *testing.T) {
	ctx := context.Background()
	meta := setupMetaWithTranslations(t)
	e := &SyncEngine{gamertag: "Tester", titleSlug: "halo_infinite", metaDB: meta} // pas de fetcher

	reg := &MatchRegistryRow{PlaylistID: strPtr("x"), PlaylistName: strPtr("x")}
	e.resolveCycleAssets(ctx, []*fetchedMatch{{MatchID: "m1", Registry: reg}})

	var n int
	if err := meta.QueryRow(`SELECT COUNT(*) FROM asset_translations WHERE asset_id = 'x'`).Scan(&n); err != nil {
		t.Fatalf("count: %v", err)
	}
	if n != 0 {
		t.Fatalf("résolution désactivée mais %d rows écrites", n)
	}
}
