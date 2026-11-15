package port

// catalog_expansion.go — l'expansion des playlists du catalogue : pour chaque playlist vue en match, les
// couples carte-mode que sa configuration déclare (même jamais joués) sont enfilés dans la file de
// récupération du catalogue, avec leur poids de rotation.

import "context"

// PlaylistRef : une playlist vue en match, avec sa version (l'API la requiert : 404 sans version, 400 sur
// « latest »).
type PlaylistRef struct {
	ID      string
	Version string
}

// PlaylistChild : un couple carte-mode déclaré par la configuration d'une playlist.
type PlaylistChild struct {
	// PairAssetID identifie le couple ; vide, le couple est ignoré.
	PairAssetID string
	// VersionID est la version du couple.
	VersionID string
	// Weight est le poids du couple dans la rotation de la playlist.
	Weight float64
}

// PlaylistsSeenReader liste les playlists distinctes que les matchs d'un titre référencent.
type PlaylistsSeenReader interface {
	// PlaylistsSeenInMatches rend les playlists (avec version) du registre des matchs du titre. La base
	// partagée n'est tenue que le temps de la lecture.
	PlaylistsSeenInMatches(ctx context.Context, titleSlug string) ([]PlaylistRef, error)
}

// PlaylistConfigFetcher lit la configuration d'une playlist auprès du service de catalogue.
type PlaylistConfigFetcher interface {
	// PlaylistChildren rend les couples carte-mode de la playlist.
	PlaylistChildren(ctx context.Context, playlist PlaylistRef) ([]PlaylistChild, error)
}

// PlaylistConfigFetcherFunc adapte une fonction en PlaylistConfigFetcher.
type PlaylistConfigFetcherFunc func(ctx context.Context, playlist PlaylistRef) ([]PlaylistChild, error)

// PlaylistChildren appelle la fonction.
func (f PlaylistConfigFetcherFunc) PlaylistChildren(ctx context.Context, playlist PlaylistRef) ([]PlaylistChild, error) {
	return f(ctx, playlist)
}

// PlaylistChildStore écrit dans metadata ce que l'expansion découvre.
type PlaylistChildStore interface {
	// EnsureSchema crée la table des poids de rotation si elle est absente (idempotent).
	EnsureSchema(ctx context.Context) error
	// EnqueuePair enfile un couple carte-mode dans la file de récupération du titre, sans doublon.
	EnqueuePair(ctx context.Context, titleSlug, pairID, versionID string) error
	// RecordWeight inscrit le poids d'un couple dans une playlist.
	RecordWeight(ctx context.Context, titleSlug, playlistID, pairID string, weight float64) error
}
