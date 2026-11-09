package wire

import (
	"context"

	"levelup/go-api/internal/platform/duckdb"
	"levelup/go-api/internal/port"
)

// PlayerReadCaches adapte les caches de lecture joueur du paquet de données au port
// [port.PlayerReadCacheInvalidator] : c'est ce qu'on injecte aux services qui écrivent hors synchronisation.
type PlayerReadCaches struct{}

var _ port.PlayerReadCacheInvalidator = PlayerReadCaches{}

// InvalidatePlayerReadCaches délègue à duckdb.InvalidatePlayerReadCaches.
func (PlayerReadCaches) InvalidatePlayerReadCaches(ctx context.Context, xuid, titleSlug string) {
	duckdb.InvalidatePlayerReadCaches(ctx, xuid, titleSlug)
}
