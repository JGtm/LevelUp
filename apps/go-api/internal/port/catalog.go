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
	"levelup/go-api/internal/games/canonical"
)

// CatalogWriter expose la PERSISTANCE catalogue (drain + upserts) au
// CatalogFetcherService, pour qu'il ne tienne plus de *sql.DB brut (ADR 0025 D-MV2,
// K1j). Interface consumer-side étroite implémentée par platform/duckdb.CatalogRepo.
// Les upserts sont ART-safe (SELECT-then-write, jamais d'ON CONFLICT sur metadata).
type CatalogWriter interface {
	// SelectPending retourne les entrées de catalog_fetch_queue dont l'asset n'est
	// PAS encore dans la table catalogue de son type (append-only : file jamais mutée).
	SelectPending(ctx context.Context, titleSlug string) ([]domain.CatalogQueueEntry, error)
	// UpsertPlaylist persiste une playlist + ses pair_links + ré-enqueue les pairs.
	// isRanked/experience sont déjà résolus par le caller (gate rankedplaylists).
	UpsertPlaylist(ctx context.Context, titleSlug string, pl canonical.CanonicalPlaylist, isRanked bool, experience string) error
	// UpsertPair persiste un pair + ses labels multi-langues + ré-enqueue map/game_variant.
	UpsertPair(ctx context.Context, titleSlug string, p canonical.CanonicalPair) error
	// UpsertMap persiste une map du catalogue.
	UpsertMap(ctx context.Context, titleSlug string, m canonical.CanonicalMap) error
	// UpsertGameVariant persiste un game_variant du catalogue.
	UpsertGameVariant(ctx context.Context, titleSlug string, gv canonical.CanonicalGameVariant) error
}

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
