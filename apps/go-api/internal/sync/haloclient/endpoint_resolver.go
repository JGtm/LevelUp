// Package sync — endpoint_resolver.go : résolution title-aware des hosts
// d'ingestion API (MT-01 / PMT-1 Contract).
//
// Les HaloAPIClient consultent un games.EndpointResolver pour router chaque
// appel vers l'host du titre courant (lu dans le ctx). Source de vérité : la
// section [endpoints] de config/titles/{slug}/constants.toml, chargée au boot.
//
// Le resolver est câblé UNE fois au boot (server.go → games.SetDefaultEndpointResolver)
// et partagé par tous les clients (le holder vit dans le package games, bas
// niveau, pour être consultable par sync ET platform/halo ET assets). Les const
// Halo legacy (haloStatsHost, …) restent comme FALLBACK byte-identique pour les
// chemins non câblés (binaires CLI hors boot, tests) — pour halo_infinite, le
// resolver rend exactement la même valeur (golden de parité), donc zéro
// changement de comportement.
package haloclient

import (
	"context"
	"log/slog"

	"levelup/go-api/internal/ctxkeys"
	"levelup/go-api/internal/games"
)

// hostFor résout l'host d'un endpoint pour le titre courant (lu dans le ctx).
// Précédence : resolver d'instance (override de test) → resolver partagé de boot
// (games.DefaultEndpointResolver). Fallback sur `legacy` (const Halo) si aucun
// resolver n'est câblé OU si le titre ne déclare pas cet endpoint (warn
// `endpoint_missing` ; transition — la validation boot PMT-12 garantit qu'un
// titre ACTIF déclare ses endpoints requis).
func (c *HaloAPIClient) hostFor(ctx context.Context, key games.EndpointKey, legacy string) string {
	res := c.endpoints
	if res == nil {
		res = games.DefaultEndpointResolver()
	}
	if res == nil {
		return legacy
	}
	slug := ctxkeys.TitleSlug(ctx)
	if host, ok := res.HostFor(slug, key); ok {
		return host
	}
	slog.WarnContext(ctx, "endpoint_missing", "title", slug, "endpoint_key", string(key))
	return legacy
}

// gamePrefix résout le segment d'URL de jeu du titre courant ("hi"/"h5") injecté
// dans les chemins d'API d'ingestion. Fallback games.DefaultGamePrefix ("hi") →
// byte-identique pour Halo Infinite. Même précédence que hostFor : resolver
// d'instance (override de test) prioritaire, sinon resolver partagé de boot.
func (c *HaloAPIClient) gamePrefix(ctx context.Context) string {
	if c.endpoints != nil {
		return games.GamePrefixFromResolver(c.endpoints, ctxkeys.TitleSlug(ctx))
	}
	return gamePrefixForCtx(ctx)
}

// gamePrefixForCtx est la variante free-function de gamePrefix (résolveurs hors
// HaloAPIClient, ex. nameplate). Consulte le resolver partagé de boot, fallback
// games.DefaultGamePrefix ("hi").
func gamePrefixForCtx(ctx context.Context) string {
	return games.GamePrefix(ctxkeys.TitleSlug(ctx))
}
