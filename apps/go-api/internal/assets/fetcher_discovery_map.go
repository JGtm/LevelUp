package assets

// fetcher_discovery_map.go — fetcher KindMapImage via DiscoveryUGC (source REMOTE).
//
// Le local-first (static curé / cache resolver) reste géré en amont : la vue match
// passe par sa cascade map_images_registry → AssetURLAdapter et n'utilise CE fetcher
// (endpoint /api/v1/assets/maps/...) qu'en dernier recours, pour les cartes inconnues.
//
// DiscoveryUGC exige le version_id de la map (404 sans) ; il est porté par ref.Variant
// (la vue match le fournit via ?v=). La résolution réseau elle-même est INJECTÉE
// (MapImageURLFetcher) pour éviter le cycle d'import assets→halo (halo→assets existe) :
// la closure vit côté caller (api/server.go) et réutilise halo.FetchAsset.

import (
	"context"
	"fmt"
	"strings"
)

// MapImageURLFetcher résout l'URL CDN de l'image d'une map via DiscoveryUGC.
// (titleID, mapID, versionID) → URL absolue, ou erreur. Nil → fetcher no-op.
type MapImageURLFetcher func(ctx context.Context, titleID, mapID, versionID string) (string, error)

// discoveryMapFetcher implémente Fetcher pour KindMapImage (source DiscoveryUGC).
type discoveryMapFetcher struct {
	fetchURL MapImageURLFetcher
}

// NewDiscoveryMapFetcher construit le fetcher KindMapImage. fetchURL nil → le
// fetcher répond ErrNotFound (aucune image distante résolue).
func NewDiscoveryMapFetcher(fetchURL MapImageURLFetcher) Fetcher {
	return &discoveryMapFetcher{fetchURL: fetchURL}
}

func (f *discoveryMapFetcher) Supports(k Kind) bool { return k == KindMapImage }

func (f *discoveryMapFetcher) Fetch(ctx context.Context, ref Ref) (Payload, error) {
	if f.fetchURL == nil {
		return nil, ErrNotFound
	}
	mapID := strings.TrimSpace(ref.ID)
	versionID := strings.TrimSpace(ref.Variant) // version portée par Variant (?v=)
	if mapID == "" || versionID == "" {
		// Sans version_id, DiscoveryUGC 404 — inutile d'appeler.
		return nil, ErrNotFound
	}
	url, err := f.fetchURL(ctx, ref.TitleID, mapID, versionID)
	if err != nil {
		return nil, fmt.Errorf("%w: discovery map image %s: %v", ErrUpstreamUnavailable, mapID, err)
	}
	url = strings.TrimSpace(url)
	if url == "" {
		return nil, ErrNotFound
	}
	return URLPayload{URL: url, ContentType: mimeFromImageURL(url)}, nil
}

// mimeFromImageURL devine le MIME depuis l'extension de l'URL (défaut JPEG : les
// miniatures DiscoveryUGC sont majoritairement .jpg).
func mimeFromImageURL(u string) string {
	if strings.Contains(strings.ToLower(u), ".png") {
		return MimeImagePNG
	}
	return MimeImageJPEG
}
