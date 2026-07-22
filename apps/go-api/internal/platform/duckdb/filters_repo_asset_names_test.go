//go:build integration

// filters_repo_asset_names_test.go — tests d'intégration de la résolution
// read-side ID->nom des assets du pipeline de filtres (applyAssetNamesFromMetadata).
//
// Couvre la voie « metadata-side » (Halo 5 : noms registry NULL, ids remplis,
// libellés dans asset_translations) et vérifie le no-op strict de la voie
// « registry » (Halo Infinite : noms déjà présents → aucune écrasure).
package duckdb

import (
	"context"
	"testing"

	"levelup/go-api/internal/domain"
)

// seedFilterAssetTranslations insère des libellés bilingues dans
// asset_translations. rows = {asset_id, asset_type, lang, name}.
func seedFilterAssetTranslations(t *testing.T, meta *DB, rows [][4]string) {
	t.Helper()
	ctx := context.Background()
	for _, row := range rows {
		if _, err := meta.Exec(ctx,
			`INSERT INTO asset_translations (asset_id, asset_type, lang, name, description, fetched_at)
			 VALUES (?, ?, ?, ?, '', now())`,
			row[0], row[1], row[2], row[3],
		); err != nil {
			t.Fatalf("seed asset_translations (%s/%s): %v", row[1], row[2], err)
		}
	}
}

// nullifyRegistryNamesForH5 met le registry de m1 dans l'état « voie Halo 5 » :
// noms NULL, ids remplis, pas de pair (game_variant = source de mode).
func nullifyRegistryNamesForH5(t *testing.T, pdb *PlayerDB, mapID, playlistID, variantID string) {
	t.Helper()
	if _, err := pdb.Player.Exec(context.Background(), `
		UPDATE shared.match_registry
		SET map_name = NULL, map_name_fr = NULL, map_id = ?,
		    pair_name = NULL, pair_name_fr = NULL, pair_id = NULL,
		    playlist_name = NULL, playlist_name_fr = NULL, playlist_id = ?,
		    game_variant_name = NULL, game_variant_id = ?
		WHERE match_id = ?
	`, mapID, playlistID, variantID, "m1"); err != nil {
		t.Fatalf("UPDATE shared.match_registry (voie H5): %v", err)
	}
}

// assertFilterName vérifie qu'un champ *string vaut la valeur attendue.
func assertFilterName(t *testing.T, field string, got *string, want string) {
	t.Helper()
	if got == nil {
		t.Errorf("%s = nil, want %q", field, want)
		return
	}
	if *got != want {
		t.Errorf("%s = %q, want %q", field, *got, want)
	}
}

// TestFiltersRepo_ApplyAssetNamesFromMetadata_H5Path : registry aux noms NULL +
// ids remplis + asset_translations seedées (en-US + fr-FR) → LoadMatchesForFilters
// renvoie les noms EN et FR (map, playlist, game_variant) résolus depuis metadata.
func TestFiltersRepo_ApplyAssetNamesFromMetadata_H5Path(t *testing.T) {
	pdb := newTestPlayerDB(t)
	ctx := context.Background()

	nullifyRegistryNamesForH5(t, pdb, "map-h5", "playlist-h5", "variant-h5")
	seedFilterAssetTranslations(t, pdb.Metadata, [][4]string{
		{"map-h5", "map", "en-US", "Truth"},
		{"map-h5", "map", "fr-FR", "Vérité"},
		{"playlist-h5", "playlist", "en-US", "Team Arena"},
		{"playlist-h5", "playlist", "fr-FR", "Arène en équipe"},
		{"variant-h5", "game_variant", "en-US", "Slayer"},
		{"variant-h5", "game_variant", "fr-FR", "Assassin"},
	})

	rows, err := NewFiltersRepo(pdb).LoadMatchesForFilters(ctx)
	if err != nil {
		t.Fatalf("LoadMatchesForFilters: %v", err)
	}
	if len(rows) != 1 {
		t.Fatalf("attendu 1 row, obtenu %d", len(rows))
	}
	m := rows[0]
	assertFilterName(t, "MapName", m.MapName, "Truth")
	assertFilterName(t, "MapNameFR", m.MapNameFR, "Vérité")
	assertFilterName(t, "PlaylistNameEN", m.PlaylistNameEN, "Team Arena")
	assertFilterName(t, "PlaylistName", m.PlaylistName, "Arène en équipe")
	assertFilterName(t, "GameVariantName", m.GameVariantName, "Slayer")
	assertFilterName(t, "GameVariantNameFR", m.GameVariantNameFR, "Assassin")
}

// TestFiltersRepo_ApplyAssetNamesFromMetadata_ENOnlyFallsBackToEN : un id sans
// fr-FR (seul en-US existe) → le champ FR retombe sur l'EN, jamais vide ni UUID.
func TestFiltersRepo_ApplyAssetNamesFromMetadata_ENOnlyFallsBackToEN(t *testing.T) {
	pdb := newTestPlayerDB(t)
	ctx := context.Background()

	nullifyRegistryNamesForH5(t, pdb, "map-en", "playlist-en", "variant-en")
	seedFilterAssetTranslations(t, pdb.Metadata, [][4]string{
		{"map-en", "map", "en-US", "Plaza"},
		{"variant-en", "game_variant", "en-US", "CTF"},
		// playlist-en : aucune traduction → doit rester vide (cas couvert plus bas).
	})

	rows, err := NewFiltersRepo(pdb).LoadMatchesForFilters(ctx)
	if err != nil {
		t.Fatalf("LoadMatchesForFilters: %v", err)
	}
	m := rows[0]
	assertFilterName(t, "MapName", m.MapName, "Plaza")
	assertFilterName(t, "MapNameFR", m.MapNameFR, "Plaza") // FR retombe sur EN
	assertFilterName(t, "GameVariantName", m.GameVariantName, "CTF")
	assertFilterName(t, "GameVariantNameFR", m.GameVariantNameFR, "CTF")
}

// TestFiltersRepo_ApplyAssetNamesFromMetadata_InfinitePathNoOverwrite : quand les
// noms registry sont déjà présents (voie Halo Infinite), l'enrichisseur ne doit
// RIEN écraser, même si asset_translations contient d'autres valeurs pour ces ids.
func TestFiltersRepo_ApplyAssetNamesFromMetadata_InfinitePathNoOverwrite(t *testing.T) {
	pdb := newTestPlayerDB(t)
	ctx := context.Background()

	// m1 par défaut : map_name=Aquarius, playlist_name=Ranked Slayer,
	// game_variant_name=Arena:Slayer (noms présents). On seede des libellés
	// PIÈGES sur les mêmes ids pour prouver qu'ils ne fuient pas.
	seedFilterAssetTranslations(t, pdb.Metadata, [][4]string{
		{"aquarius", "map", "en-US", "PIEGE-MAP"},
		{"aquarius", "map", "fr-FR", "PIEGE-MAP-FR"},
		{"playlist-ranked-slayer", "playlist", "en-US", "PIEGE-PL"},
		{"variant-slayer", "game_variant", "en-US", "PIEGE-VAR"},
	})

	rows, err := NewFiltersRepo(pdb).LoadMatchesForFilters(ctx)
	if err != nil {
		t.Fatalf("LoadMatchesForFilters: %v", err)
	}
	m := rows[0]
	assertFilterName(t, "MapName", m.MapName, "Aquarius")
	assertFilterName(t, "PlaylistNameEN", m.PlaylistNameEN, "Ranked Slayer")
	assertFilterName(t, "GameVariantName", m.GameVariantName, "Arena:Slayer")
}

// TestFiltersRepo_ApplyAssetNamesFromMetadata_MissingTranslationStaysEmpty : un id
// sans aucune traduction → le champ reste vide, aucun UUID n'est écrit comme nom.
func TestFiltersRepo_ApplyAssetNamesFromMetadata_MissingTranslationStaysEmpty(t *testing.T) {
	pdb := newTestPlayerDB(t)
	ctx := context.Background()

	nullifyRegistryNamesForH5(t, pdb, "map-unknown", "playlist-unknown", "variant-unknown")
	// Aucune asset_translations seedée pour ces ids.

	rows, err := NewFiltersRepo(pdb).LoadMatchesForFilters(ctx)
	if err != nil {
		t.Fatalf("LoadMatchesForFilters: %v", err)
	}
	m := rows[0]
	for _, c := range []struct {
		field string
		got   *string
	}{
		{"MapName", m.MapName},
		{"MapNameFR", m.MapNameFR},
		{"PlaylistNameEN", m.PlaylistNameEN},
		{"GameVariantName", m.GameVariantName},
		{"GameVariantNameFR", m.GameVariantNameFR},
	} {
		if v := derefString(c.got); v != "" {
			t.Errorf("%s = %q, want vide (id sans traduction, aucun UUID ne doit fuiter)", c.field, v)
		}
	}
}

// TestFiltersRepo_ApplyAssetNamesFromMetadata_NilMetadataNoOp : sans metadata DB,
// l'enrichisseur retourne immédiatement sans erreur ni écriture.
func TestFiltersRepo_ApplyAssetNamesFromMetadata_NilMetadataNoOp(t *testing.T) {
	repo := &FiltersRepo{pdb: &PlayerDB{}} // Metadata = nil
	mapID := "map-x"
	rows := []domain.FilterMatchRow{{MatchID: "m1", MapID: &mapID}}

	repo.applyAssetNamesFromMetadata(context.Background(), rows)

	if rows[0].MapName != nil {
		t.Errorf("MapName = %q, want nil (Metadata nil doit être no-op)", *rows[0].MapName)
	}
}
