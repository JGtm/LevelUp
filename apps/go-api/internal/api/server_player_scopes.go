package api

// server_player_scopes.go : adaptateurs de composition entre le résolveur de joueurs (slug -> bases
// DuckDB) et les ports de progression et de campagne. Aucun SQL ici : les dépôts sont ceux de
// `platform/duckdb`.

import (
	"context"

	"levelup/go-api/internal/api/wire"
	platform_duckdb "levelup/go-api/internal/platform/duckdb"
	"levelup/go-api/internal/port"
)

// progressionStoreResolver adapte le résolveur de joueurs en résolveur de dépôt de progression.
func progressionStoreResolver(resolve wire.PlayerResolver) port.ProgressionStoreResolver {
	return func(ctx context.Context, playerSlug string) (port.ProgressionStore, error) {
		pdb, err := resolve(ctx, playerSlug)
		if err != nil {
			return nil, err
		}
		return platform_duckdb.NewPlayerProgressionStore(pdb), nil
	}
}

// campaignScopeResolver adapte le résolveur de joueurs en résolveur de dépôts de campagne.
func campaignScopeResolver(resolve wire.PlayerResolver) port.CampaignScopeResolver {
	return func(ctx context.Context, playerSlug string) (port.CampaignScope, error) {
		pdb, err := resolve(ctx, playerSlug)
		if err != nil {
			return port.CampaignScope{}, err
		}
		return platform_duckdb.NewCampaignScope(pdb), nil
	}
}
