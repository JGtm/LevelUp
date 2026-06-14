// Package halo — asset_name_fetcher.go : adaptateur HaloProvider.FetchAsset →
// assetnames.Fetcher pour la résolution autonome des noms d'assets au sync.
//
// IMPORTANT : la résolution UGC tape discovery-infiniteugc (FetchAsset), qui EXIGE
// un token Spartan + 343-clearance (≠ API publique, ≠ host gamecms-hacs). Le token
// vient du POOL unifié (auth/pool), la même source que tous les syncs (cf.
// internal/sync/assetnames_wiring.go : resolveRefsWithPool).
// Cf. plan « Résolution autonome des noms d'assets Halo ».
package halo

import (
	"context"
	"os"
	"strings"

	"levelup/go-api/internal/assetnames"
	"levelup/go-api/internal/domain"
)

// AssetNameResolveRateLimit : débit du fetch des noms d'assets (req/min), aligné
// sur le drain catalogue. En régime permanent ~0 fetch/cycle (skip-fresh).
const AssetNameResolveRateLimit = 60

// assetNameFetcher adapte *HaloProvider à l'interface assetnames.Fetcher.
type assetNameFetcher struct{ p *HaloProvider }

// NewAssetNameFetcher construit un assetnames.Fetcher AUTHENTIFIÉ (Spartan +
// clearance requis par discovery-infiniteugc). tokens provient du pool unifié.
// Halo-spécifique.
func NewAssetNameFetcher(rateLimitPerMinute int, tokens *domain.HaloTokens) assetnames.Fetcher {
	return assetNameFetcher{p: NewHaloProvider().WithRateLimit(rateLimitPerMinute).WithTokens(tokens)}
}

// AssetNameResolutionEnabled : kill-switch d'urgence. LEVELUP_SYNC_RESOLVE_ASSETS=0
// (ou false) désactive la résolution autonome (ON par défaut). Temporaire — à
// retirer une fois la feature éprouvée (échéance 2026-09-01).
func AssetNameResolutionEnabled() bool {
	v := strings.ToLower(strings.TrimSpace(os.Getenv("LEVELUP_SYNC_RESOLVE_ASSETS")))
	return v != "0" && v != "false"
}

// FetchName récupère le PublicName localisé d'un asset. Nom vide si l'asset
// n'expose pas de libellé pour cette langue (non bloquant côté résolveur).
func (f assetNameFetcher) FetchName(ctx context.Context, assetType, titleID, assetID, versionID, lang string) (string, error) {
	a, err := f.p.FetchAsset(ctx, AssetType(assetType), titleID, assetID, versionID, lang)
	if err != nil {
		// Échec non bloquant (le résolveur compte l'erreur en agrégat). Détail par
		// asset loggué en Debug par FetchAsset (discovery_client.go). Causes connues
		// non corrigeables ici : version_id de rotation périmée (discovery-infiniteugc
		// retire les vieilles versions → 404).
		return "", err
	}
	if a == nil {
		return "", nil
	}
	return a.PublicName, nil
}
