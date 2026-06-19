// Package halo — endpoints.go : résolution title-aware des hosts d'ingestion du
// HaloProvider (MT-01 / PMT-1 Contract axes 4-6). Miroir de
// internal/sync/endpoint_resolver.go pour le second type de client.
//
// Le holder du resolver partagé vit dans le package games (bas niveau), câblé au
// boot (server.go → games.SetDefaultEndpointResolver). Les const Halo legacy
// restent comme FALLBACK byte-identique pour halo_infinite et les chemins non
// câblés (CLI, tests).
package halo

import (
	"context"
	"log/slog"

	"levelup/go-api/internal/ctxkeys"
	"levelup/go-api/internal/games"
)

// hostFor résout l'host d'un endpoint pour le titre courant (lu dans le ctx).
// Précédence : override d'instance (`override` non vide — champ test du provider)
// → resolver partagé de boot (games.DefaultEndpointResolver) → `legacy` (const
// Halo). Un titre câblé sans cet endpoint logue `endpoint_missing` et retombe sur
// legacy (transition ; la validation boot PMT-12 doit l'empêcher pour un titre ACTIF).
func (p *HaloProvider) hostFor(ctx context.Context, key games.EndpointKey, override, legacy string) string {
	if override != "" {
		return override
	}
	res := games.DefaultEndpointResolver()
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
// dans les chemins d'API du HaloProvider. Fallback games.DefaultGamePrefix ("hi")
// → byte-identique pour Halo Infinite. Miroir de internal/sync.gamePrefix.
func (p *HaloProvider) gamePrefix(ctx context.Context) string {
	return games.GamePrefix(ctxkeys.TitleSlug(ctx))
}
