package port

// catalog.go — interface CatalogRepo (Phase C plan catalogue).
//
// Lecture seule du catalogue persisté dans metadata.duckdb. Le fetch
// DiscoveryUGC est exclusivement du ressort du CatalogFetcherService (Phase F)
// qui utilise le TitleCatalogAdapter — pas via cette interface.
//
// Implémenté par platform/duckdb.CatalogRepo (Phase F).

import (
	"context"

	"levelup/go-api/internal/domain"
)

// CatalogRepo expose la lecture catalogue title-aware vers les services.
type CatalogRepo interface {
	// PlaylistsByTitle retourne les playlists du catalogue pour un titre donné.
	// Si onlyPlayed = true, ne retourne que les playlists ayant au moins un match
	// joué par le xuid (JOIN match_registry filtré sur match_participants).
	// Sinon retourne le catalogue complet (is_active = true).
	PlaylistsByTitle(ctx context.Context, titleSlug, xuid string, onlyPlayed bool) ([]domain.CatalogPlaylist, error)

	// PairsByPlaylist retourne les pairs (map+mode) liés à une playlist.
	PairsByPlaylist(ctx context.Context, titleSlug, playlistAssetID string) ([]domain.CatalogPair, error)

	// MapsByTitle retourne les maps du catalogue pour un titre donné.
	// Si xuid != "" et onlyPlayed = true, ne retourne que les maps jouées par le xuid.
	MapsByTitle(ctx context.Context, titleSlug, xuid string, onlyPlayed bool) ([]domain.CatalogMap, error)

	// CountCatalogEntries retourne le nombre total de playlists actives pour un titre.
	// Utilisé par FiltersService.Resolve() comme guard de fallback (Phase I) :
	// si le catalogue est vide, retomber sur le scan match_participants legacy.
	CountCatalogEntries(ctx context.Context, titleSlug string) (int, error)
}
