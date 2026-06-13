// Package halo — asset_name_fetcher.go : adaptateur HaloProvider.FetchAsset →
// assetnames.Fetcher pour la résolution autonome des noms d'assets au sync.
//
// FetchAsset tape l'API publique GameCMS (gamecms-hacs) qui ne requiert PAS
// d'authentification Spartan (cf. discovery_client.go) — la résolution des noms
// est donc token-free, robuste face à la péremption des tokens (contrairement
// au drain catalogue DiscoveryUGC). Cf. plan « Résolution autonome des noms
// d'assets Halo » (2026-06-13).
package halo

import (
	"context"
	"os"
	"strings"

	"levelup/go-api/internal/assetnames"
)

// assetNameResolveRateLimit : débit du fetch des noms d'assets (req/min), aligné
// sur le drain catalogue. En régime permanent ~0 fetch/cycle (skip-fresh).
const assetNameResolveRateLimit = 60

// assetNameFetcher adapte *HaloProvider à l'interface assetnames.Fetcher.
type assetNameFetcher struct{ p *HaloProvider }

// NewAssetNameFetcher construit un assetnames.Fetcher token-free (API publique
// GameCMS) avec le rate limit donné (req/min). Halo-spécifique : fourni par le
// titre au wiring de sync.
func NewAssetNameFetcher(rateLimitPerMinute int) assetnames.Fetcher {
	return assetNameFetcher{p: NewHaloProvider().WithRateLimit(rateLimitPerMinute)}
}

// NewAssetNameFetcherIfEnabled retourne le fetcher de noms d'assets. ACTIVÉ PAR
// DÉFAUT : sans cette résolution, le contenu neuf affiche des UUID bruts (feature
// critique). LEVELUP_SYNC_RESOLVE_ASSETS=0|false agit en KILL-SWITCH d'urgence
// (couper la résolution sans redéploiement, p.ex. si l'API GameCMS dégrade les
// temps de cycle) → retourne nil. Le kill-switch est temporaire, à retirer une
// fois la feature éprouvée (cf. plan, échéance 2026-09-01). Centralisé ici pour
// partager le même gate entre les wirings V1 (scheduler) et V2 (cmd/server).
func NewAssetNameFetcherIfEnabled() assetnames.Fetcher {
	v := strings.ToLower(strings.TrimSpace(os.Getenv("LEVELUP_SYNC_RESOLVE_ASSETS")))
	if v == "0" || v == "false" {
		return nil // kill-switch explicite
	}
	return NewAssetNameFetcher(assetNameResolveRateLimit)
}

// FetchName récupère le PublicName localisé d'un asset. Nom vide si l'asset
// n'expose pas de libellé pour cette langue (non bloquant côté résolveur).
func (f assetNameFetcher) FetchName(ctx context.Context, assetType, titleID, assetID, versionID, lang string) (string, error) {
	a, err := f.p.FetchAsset(ctx, AssetType(assetType), titleID, assetID, versionID, lang)
	if err != nil {
		return "", err
	}
	if a == nil {
		return "", nil
	}
	return a.PublicName, nil
}
