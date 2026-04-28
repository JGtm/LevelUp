// Package service — asset_service.go : Asset Drawer (maps, armes).
package service

import (
	"context"
	"fmt"

	"levelup/go-api/internal/games/canonical"
	"levelup/go-api/internal/port"
)

// AssetService construit les métadonnées d'assets pour l'Asset Drawer.
type AssetService struct {
	repo port.AssetMetaRepository
}

// NewAssetService crée un AssetService.
func NewAssetService(repo port.AssetMetaRepository) *AssetService {
	return &AssetService{repo: repo}
}

// ListMaps retourne les maps d'un titre avec image_url.
func (s *AssetService) ListMaps(ctx context.Context, titleID, search string) ([]canonical.AssetMeta, error) {
	items, err := s.repo.ListMapsByTitle(ctx, titleID, search)
	if err != nil {
		return nil, fmt.Errorf("AssetService.ListMaps: %w", err)
	}
	for i := range items {
		items[i].ImageURL = fmt.Sprintf("/api/v1/assets/maps/%s/%s/image", titleID, items[i].ID)
	}
	return items, nil
}

// ListWeapons retourne les armes avec image_url.
// ImageURL est vide en V1 — les images armes n'ont pas de mapping weapon_id→fichier (B2).
func (s *AssetService) ListWeapons(ctx context.Context, titleID, search string) ([]canonical.AssetMeta, error) {
	items, err := s.repo.ListWeaponsByTitle(ctx, titleID, search)
	if err != nil {
		return nil, fmt.Errorf("AssetService.ListWeapons: %w", err)
	}
	return items, nil
}
