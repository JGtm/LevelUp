package importstore

import (
	"context"
	"database/sql"

	"levelup/go-api/internal/platform/dblease"
	"levelup/go-api/internal/platform/duckdb/sharedprovider"
	"levelup/go-api/internal/sync"
	"levelup/go-api/internal/transfer"
)

// SiteShared rend l'acquisition de l'écriture de la base partagée du titre pour l'import depuis le site :
// par le fournisseur de l'application, coordonnée avec la synchronisation et les lecteurs (bascule B-swap
// comprise).
func SiteShared(provider sharedprovider.Provider, sharedDBPath string) transfer.Acquire {
	return func(ctx context.Context) (*sql.DB, func(), error) {
		return sync.AcquireSharedWriterStandalone(ctx, provider, sharedDBPath)
	}
}

// SitePlayer rend l'acquisition de l'écriture de la base du joueur, sous son bail (un seul écrivain), par
// l'ouverture du cache du processus qui applique aussi le schéma.
func SitePlayer(playerDBPath string) transfer.Acquire {
	return func(ctx context.Context) (*sql.DB, func(), error) {
		lease, err := dblease.AcquireWriterCtx(ctx, nil, playerDBPath, dblease.KindPlayer)
		if err != nil {
			return nil, nil, err
		}
		handle, err := sync.OpenPlayerDB(playerDBPath)
		if err != nil {
			lease.Release()
			return nil, nil, err
		}
		return handle.SQLDb(), func() { _ = handle.Close(); lease.Release() }, nil
	}
}
