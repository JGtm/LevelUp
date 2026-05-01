package service

import (
	"context"
	"testing"

	"levelup/go-api/internal/games/canonical"
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
	if maps[0].NameEN != "Aquarius" {
		t.Errorf("NameEN=%q, attendu Aquarius", maps[0].NameEN)
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
	if weapons[0].NameEN != "BR75 Battle Rifle" {
		t.Errorf("NameEN=%q", weapons[0].NameEN)
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

func testMaps() []canonical.AssetMeta {
	return []canonical.AssetMeta{
		{ID: "map-001", NameEN: "Aquarius", NameFR: "Aquarius"},
		{ID: "map-002", NameEN: "Breaker", NameFR: "Breaker"},
		{ID: "map-003", NameEN: "Streets", NameFR: "Streets"},
	}
}

func testWeapons() []canonical.AssetMeta {
	return []canonical.AssetMeta{
		{ID: "100", NameEN: "BR75 Battle Rifle", NameFR: "Fusil BR75"},
		{ID: "200", NameEN: "Skewer", NameFR: "Brochette"},
	}
}
