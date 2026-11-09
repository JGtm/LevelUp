package service

import (
	"context"
	"testing"

	"levelup/go-api/internal/games/canonical"
	"levelup/go-api/internal/i18n"
)

func TestStaticAssetMetaRepo_ListMaps_All(t *testing.T) {
	repo := NewStaticAssetMetaRepo(testMaps(), testWeapons())
	maps, err := repo.ListMapsByTitle(context.Background(), "halo_infinite", "")
	if err != nil {
		t.Fatal(err)
	}
	if len(maps) != 3 {
		t.Errorf("attendu 3, obtenu %d", len(maps))
	}
}

func TestStaticAssetMetaRepo_ListMaps_Search(t *testing.T) {
	repo := NewStaticAssetMetaRepo(testMaps(), testWeapons())
	maps, err := repo.ListMapsByTitle(context.Background(), "halo_infinite", "aqu")
	if err != nil {
		t.Fatal(err)
	}
	if len(maps) != 1 {
		t.Errorf("attendu 1, obtenu %d", len(maps))
	}
	if maps[0].CanonicalName() != "Aquarius" {
		t.Errorf("nom canonique=%q, attendu Aquarius", maps[0].CanonicalName())
	}
}

func TestStaticAssetMetaRepo_ListMaps_SearchCaseInsensitive(t *testing.T) {
	repo := NewStaticAssetMetaRepo(testMaps(), testWeapons())
	maps, err := repo.ListMapsByTitle(context.Background(), "halo_infinite", "AQU")
	if err != nil {
		t.Fatal(err)
	}
	if len(maps) != 1 {
		t.Errorf("attendu 1, obtenu %d (recherche insensible à la casse)", len(maps))
	}
}

func TestStaticAssetMetaRepo_ListMaps_SearchNoMatch(t *testing.T) {
	repo := NewStaticAssetMetaRepo(testMaps(), testWeapons())
	maps, err := repo.ListMapsByTitle(context.Background(), "halo_infinite", "zzz")
	if err != nil {
		t.Fatal(err)
	}
	if len(maps) != 0 {
		t.Errorf("attendu 0, obtenu %d", len(maps))
	}
}

func TestStaticAssetMetaRepo_ListWeapons_All(t *testing.T) {
	repo := NewStaticAssetMetaRepo(testMaps(), testWeapons())
	weapons, err := repo.ListWeaponsByTitle(context.Background(), "halo_infinite", "")
	if err != nil {
		t.Fatal(err)
	}
	if len(weapons) != 2 {
		t.Errorf("attendu 2, obtenu %d", len(weapons))
	}
}

func TestStaticAssetMetaRepo_ListWeapons_Search(t *testing.T) {
	repo := NewStaticAssetMetaRepo(testMaps(), testWeapons())
	weapons, err := repo.ListWeaponsByTitle(context.Background(), "halo_infinite", "BR75")
	if err != nil {
		t.Fatal(err)
	}
	if len(weapons) != 1 {
		t.Errorf("attendu 1, obtenu %d", len(weapons))
	}
	if weapons[0].CanonicalName() != "BR75 Battle Rifle" {
		t.Errorf("nom canonique=%q", weapons[0].CanonicalName())
	}
}

func TestStaticAssetMetaRepo_Empty(t *testing.T) {
	repo := NewStaticAssetMetaRepo(nil, nil)
	maps, _ := repo.ListMapsByTitle(context.Background(), "any", "")
	weapons, _ := repo.ListWeaponsByTitle(context.Background(), "any", "")
	if len(maps) != 0 || len(weapons) != 0 {
		t.Errorf("repo vide : attendu 0/0, obtenu %d/%d", len(maps), len(weapons))
	}
}

// TestStaticAssetMetaRepo_MedalsEmptyWithoutFallback : sans WithFallbackMedals, le
// titre par défaut (Infinite) n'a AUCUNE médaille — c'était le bug du tab vide.
func TestStaticAssetMetaRepo_MedalsEmptyWithoutFallback(t *testing.T) {
	repo := NewStaticAssetMetaRepo(testMaps(), testWeapons())
	medals, _ := repo.ListMedalsByTitle(context.Background(), "halo_infinite", "")
	if len(medals) != 0 {
		t.Errorf("sans fallback medals : attendu 0, obtenu %d", len(medals))
	}
}

// TestStaticAssetMetaRepo_FallbackMedals : WithFallbackMedals sert les médailles du
// titre par défaut/inconnu ; un override WithTitle garde ses propres médailles.
func TestStaticAssetMetaRepo_FallbackMedals(t *testing.T) {
	ctx := context.Background()
	repo := NewStaticAssetMetaRepo(testMaps(), testWeapons()).
		WithFallbackMedals(testMedals()).
		WithTitle("halo_5", nil, nil, []canonical.AssetMeta{{ID: "9", Names: i18n.FromColumns("Killjoy", "Rabat-joie")}})

	hinf, _ := repo.ListMedalsByTitle(ctx, "halo_infinite", "")
	if len(hinf) != 2 {
		t.Fatalf("Infinite (fallback) : attendu 2 médailles, obtenu %d", len(hinf))
	}
	if hinf[0].Names[i18n.LocaleFR] != "Vengeur" {
		t.Errorf("Infinite médaille[0] name_fr = %q, want Vengeur", hinf[0].Names[i18n.LocaleFR])
	}

	h5, _ := repo.ListMedalsByTitle(ctx, "halo_5", "")
	if len(h5) != 1 || h5[0].CanonicalName() != "Killjoy" {
		t.Errorf("h5 (override) : attendu [Killjoy], obtenu %+v", h5)
	}

	// Recherche locale-aware (name_fr) sur le fallback.
	found, _ := repo.ListMedalsByTitle(ctx, "halo_infinite", "veng")
	if len(found) != 1 || found[0].CanonicalName() != "Avenger" {
		t.Errorf("recherche 'veng' : attendu [Avenger], obtenu %+v", found)
	}
}

func testMedals() []canonical.AssetMeta {
	return []canonical.AssetMeta{
		{ID: "9000000001", Names: i18n.FromColumns("Avenger", "Vengeur"), ImageURL: "/static/medals/halo_infinite/9000000001.png"},
		{ID: "3565443938", Names: i18n.FromColumns("Perfect", "Parfait"), ImageURL: "/static/medals/halo_infinite/3565443938.png"},
	}
}

func testMaps() []canonical.AssetMeta {
	return []canonical.AssetMeta{
		{ID: "map-001", Names: i18n.FromColumns("Aquarius", "Aquarius")},
		{ID: "map-002", Names: i18n.FromColumns("Breaker", "Breaker")},
		{ID: "map-003", Names: i18n.FromColumns("Streets", "Streets")},
	}
}

func testWeapons() []canonical.AssetMeta {
	return []canonical.AssetMeta{
		{ID: "100", Names: i18n.FromColumns("BR75 Battle Rifle", "Fusil BR75")},
		{ID: "200", Names: i18n.FromColumns("Skewer", "Brochette")},
	}
}
