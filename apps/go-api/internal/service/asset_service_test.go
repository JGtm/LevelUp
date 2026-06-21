package service

import (
	"context"
	"errors"
	"testing"

	"levelup/go-api/internal/games/canonical"
)

// mockAssetMetaRepo implémente port.AssetMetaRepository pour les tests.
type mockAssetMetaRepo struct {
	maps    []canonical.AssetMeta
	weapons []canonical.AssetMeta
	err     error
}

func (m *mockAssetMetaRepo) ListMapsByTitle(_ context.Context, _, _ string) ([]canonical.AssetMeta, error) {
	return m.maps, m.err
}

func (m *mockAssetMetaRepo) ListWeaponsByTitle(_ context.Context, _, _ string) ([]canonical.AssetMeta, error) {
	return m.weapons, m.err
}

func (m *mockAssetMetaRepo) ListMedalsByTitle(_ context.Context, _, _ string) ([]canonical.AssetMeta, error) {
	return nil, m.err
}

func TestAssetService_ListMaps_EnrichesImageURL(t *testing.T) {
	repo := &mockAssetMetaRepo{
		maps: []canonical.AssetMeta{
			{ID: "map-001", NameEN: "Aquarius", NameFR: "Aquarius"},
			{ID: "map-002", NameEN: "Breaker", NameFR: "Breaker"},
		},
	}
	svc := NewAssetService(repo)

	items, err := svc.ListMaps(context.Background(), "halo_infinite", "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(items) != 2 {
		t.Fatalf("len=%d, want 2", len(items))
	}
	want := "/api/v1/assets/maps/halo_infinite/map-001/image"
	if items[0].ImageURL != want {
		t.Errorf("ImageURL=%q, want %q", items[0].ImageURL, want)
	}
	want2 := "/api/v1/assets/maps/halo_infinite/map-002/image"
	if items[1].ImageURL != want2 {
		t.Errorf("ImageURL=%q, want %q", items[1].ImageURL, want2)
	}
}

func TestAssetService_ListMaps_RepoError(t *testing.T) {
	repo := &mockAssetMetaRepo{err: errors.New("db fail")}
	svc := NewAssetService(repo)

	_, err := svc.ListMaps(context.Background(), "halo_infinite", "")
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestAssetService_ListMaps_EmptyResult(t *testing.T) {
	repo := &mockAssetMetaRepo{maps: nil}
	svc := NewAssetService(repo)

	items, err := svc.ListMaps(context.Background(), "unknown_title", "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(items) != 0 {
		t.Errorf("len=%d, want 0", len(items))
	}
}

func TestAssetService_ListWeapons_NoImageURL(t *testing.T) {
	repo := &mockAssetMetaRepo{
		weapons: []canonical.AssetMeta{
			{ID: "100", NameEN: "BR75 Battle Rifle", NameFR: "Fusil BR75"},
		},
	}
	svc := NewAssetService(repo)

	items, err := svc.ListWeapons(context.Background(), "halo_infinite", "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(items) != 1 {
		t.Fatalf("len=%d, want 1", len(items))
	}
	if items[0].ImageURL != "" {
		t.Errorf("ImageURL=%q, want empty (B2 gap — no weapon_id→file mapping in V1)", items[0].ImageURL)
	}
}

func TestAssetService_ListWeapons_RepoError(t *testing.T) {
	repo := &mockAssetMetaRepo{err: errors.New("db fail")}
	svc := NewAssetService(repo)

	_, err := svc.ListWeapons(context.Background(), "halo_infinite", "")
	if err == nil {
		t.Fatal("expected error")
	}
}
