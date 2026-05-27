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
	repo           port.AssetMetaRepository
	mapImageURL    func(titleID, nameEN string) string // nil = UUID-based URL (fallback)
	weaponImageURL func(titleID, nameEN string) string // nil = pas d'image
}

// NewAssetService crée un AssetService.
func NewAssetService(repo port.AssetMetaRepository) *AssetService {
	return &AssetService{repo: repo}
}

// WithMapImageURL configure une fonction de construction d'URL d'image de map.
// Prioritaire sur l'URL UUID par défaut — utiliser l'adapter TitleAssetURLAdapter.
func (s *AssetService) WithMapImageURL(fn func(titleID, nameEN string) string) *AssetService {
	s.mapImageURL = fn
	return s
}

// WithWeaponImageURL configure une fonction de construction d'URL d'image d'arme.
func (s *AssetService) WithWeaponImageURL(fn func(titleID, nameEN string) string) *AssetService {
	s.weaponImageURL = fn
	return s
}

// ListMaps retourne les maps d'un titre avec image_url.
func (s *AssetService) ListMaps(ctx context.Context, titleID, search string) ([]canonical.AssetMeta, error) {
	items, err := s.repo.ListMapsByTitle(ctx, titleID, search)
	if err != nil {
		return nil, fmt.Errorf("AssetService.ListMaps: %w", err)
	}
	// Init [] plutôt que nil : la valeur sera marshallée à la racine d'une
	// réponse HTTP — un slice nil donne `null` et crashe le front. Cf.
	// testutil.RequireNoNilSlicesWithoutOmitempty.
	out := make([]canonical.AssetMeta, 0, len(items))
	for i := range items {
		if s.mapImageURL != nil {
			items[i].ImageURL = s.mapImageURL(titleID, items[i].NameEN)
			if items[i].ImageURL == "" {
				continue // variante mode+map sans image : exclure du drawer
			}
		} else {
			items[i].ImageURL = fmt.Sprintf("/api/v1/assets/maps/%s/%s/image", titleID, items[i].ID)
		}
		out = append(out, items[i])
	}
	return out, nil
}

// ListWeapons retourne les armes avec image_url.
func (s *AssetService) ListWeapons(ctx context.Context, titleID, search string) ([]canonical.AssetMeta, error) {
	items, err := s.repo.ListWeaponsByTitle(ctx, titleID, search)
	if err != nil {
		return nil, fmt.Errorf("AssetService.ListWeapons: %w", err)
	}
	if s.weaponImageURL != nil {
		for i := range items {
			items[i].ImageURL = s.weaponImageURL(titleID, items[i].NameEN)
		}
	}
	// Init [] plutôt que nil : même raison que ListMaps.
	if items == nil {
		items = []canonical.AssetMeta{}
	}
	return items, nil
}
