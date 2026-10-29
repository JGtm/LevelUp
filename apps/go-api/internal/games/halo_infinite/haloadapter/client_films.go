package haloadapter

// client_films.go — playlists (Discovery UGC) et films. La lecture des films (cache disque,
// décompression, finalisation, version) est à `sync/filmfetch` : le client lui passe deux clients
// de la brique, gardés sous deux surfaces (manifeste authentifié, blobs du CDN public), et ses
// jetons.

import (
	"context"

	"levelup/go-api/internal/haloapi/infinite"
	"levelup/go-api/internal/sync/filmfetch"
)

// PlaylistEntry : un couple carte-mode d'une playlist, sa version et son poids de rotation.
type PlaylistEntry = infinite.PlaylistEntry

// PlaylistConfig : le nom d'une playlist et ses couples carte-mode (même jamais joués).
type PlaylistConfig = infinite.PlaylistConfig

// GetPlaylistConfig récupère la configuration d'une playlist. `versionID` est requis (le service
// rend 404 sans, 400 pour « latest »).
func (c *Client) GetPlaylistConfig(ctx context.Context, playlistID, versionID string) (*PlaylistConfig, error) {
	return c.api(ctx, surfaceAPIGet, infinite.CleDiscoveryUGC).GetPlaylistConfig(ctx, c.auth(), playlistID, versionID)
}

// films rend le lecteur de films de ce client, pour le titre du contexte.
func (c *Client) films(ctx context.Context) *filmfetch.Films {
	manifeste := c.api(ctx, surfaceAPIGet, infinite.CleUGCFilm)
	blobs := c.api(ctx, surfaceAPIBlob)
	return filmfetch.New(manifeste, blobs, c.auth()).WithLocalFilmCache(c.localFilmCache)
}

// GetMatchFilm rend les chunks REPLICATION_DATA du film d'un match (voir
// [filmfetch.Films.GetMatchFilm]).
func (c *Client) GetMatchFilm(ctx context.Context, matchID string) (map[int]filmfetch.FilmChunkData, bool, error) {
	return c.films(ctx).GetMatchFilm(ctx, matchID)
}

// GetFilmChunks rend tous les chunks du film en une lecture de manifeste (voir
// [filmfetch.Films.GetFilmChunks]).
func (c *Client) GetFilmChunks(ctx context.Context, matchID string) ([]filmfetch.FilmChunk, bool, error) {
	return c.films(ctx).GetFilmChunks(ctx, matchID)
}

// GetFilmChunkURLs rend les références des chunks du film, sans octets (voir
// [filmfetch.Films.GetFilmChunkURLs]).
func (c *Client) GetFilmChunkURLs(ctx context.Context, matchID string) ([]filmfetch.FilmChunkRef, bool, error) {
	return c.films(ctx).GetFilmChunkURLs(ctx, matchID)
}

// GetHighlightEventsChunk rend le chunk des temps forts du film et sa version (voir
// [filmfetch.Films.GetHighlightEventsChunk]).
func (c *Client) GetHighlightEventsChunk(ctx context.Context, matchID string) ([]byte, int, bool, error) {
	return c.films(ctx).GetHighlightEventsChunk(ctx, matchID)
}
