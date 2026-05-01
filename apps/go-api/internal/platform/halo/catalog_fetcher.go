package halo

// catalog_fetcher.go — Phase D du plan PLAN_PLAYLISTS_CATALOG.md.
//
// Wrapper qui adapte halo.HaloProvider à l'interface halo_infinite.AssetFetcher.
// Dépendance directionnelle : platform/halo → games/halo_infinite (sens inverse
// de l'import naturel — éviter le cycle games/halo_infinite ↔ platform/halo).

import (
	"context"
	"fmt"

	"levelup/go-api/internal/games/halo_infinite"
)

// CatalogFetcher adapte HaloProvider à l'interface halo_infinite.AssetFetcher.
type CatalogFetcher struct {
	Provider *HaloProvider
}

// NewCatalogFetcher construit un wrapper autour d'un HaloProvider existant.
func NewCatalogFetcher(p *HaloProvider) *CatalogFetcher {
	return &CatalogFetcher{Provider: p}
}

// FetchAsset traduit l'AssetType local vers halo.AssetType, appelle le provider,
// puis retourne un DiscoveryAssetRaw épuré.
func (f *CatalogFetcher) FetchAsset(
	ctx context.Context,
	assetType halo_infinite.AssetType,
	titleID, assetID, versionID, lang string,
) (*halo_infinite.DiscoveryAssetRaw, error) {
	if f.Provider == nil {
		return nil, fmt.Errorf("CatalogFetcher: provider non injecté")
	}
	platformType, err := mapAssetType(assetType)
	if err != nil {
		return nil, err
	}
	asset, err := f.Provider.FetchAsset(ctx, platformType, titleID, assetID, versionID, lang)
	if err != nil {
		return nil, err
	}
	return &halo_infinite.DiscoveryAssetRaw{
		AssetID:     asset.AssetID,
		VersionID:   asset.VersionID,
		PublicName:  asset.PublicName,
		Description: asset.Description,
	}, nil
}

func mapAssetType(t halo_infinite.AssetType) (AssetType, error) {
	switch t {
	case halo_infinite.AssetTypePlaylist:
		return AssetTypePlaylist, nil
	case halo_infinite.AssetTypeMap:
		return AssetTypeMap, nil
	case halo_infinite.AssetTypePair:
		return AssetTypePair, nil
	case halo_infinite.AssetTypeGameVariant:
		return AssetTypeGameVariant, nil
	}
	return "", fmt.Errorf("AssetType inconnu: %q", t)
}
