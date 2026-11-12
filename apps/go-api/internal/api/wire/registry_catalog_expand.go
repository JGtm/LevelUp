// Package api — registry_catalog_expand.go : le câblage de la découverte « A à Z » des couples
// map-mode ENFANTS d'une playlist (même jamais joués), via la config playlist discovery-infiniteugc
// (sync.GetPlaylistConfig). Le drain (RunCatalogUGCDrain) ne part que de match_registry
// (joué) ; l'expansion enfile en plus les enfants déclarés dans la config de chaque
// playlist, pour que le drain les nomme, et stocke les poids de rotation. La règle est dans
// service.PlaylistExpansionService ; ce fichier branche ses ports sur les bases et le client Halo.
package wire

import (
	"context"
	"fmt"

	"levelup/go-api/internal/persist"
	"levelup/go-api/internal/platform/duckdb"
	"levelup/go-api/internal/port"
	"levelup/go-api/internal/service"
	syncpkg "levelup/go-api/internal/sync"
)

// playlistExpandRPS : débit du fetch config playlist (discovery-infiniteugc). Conservateur.
const playlistExpandRPS = 3

// ExpandPlaylistChildren enfile les couples map-mode enfants des playlists vues en match (cf.
// service.PlaylistExpansionService). Rend le nombre de couples enfants enfilés.
func (r *ServiceRegistry) ExpandPlaylistChildren(ctx context.Context, titleSlug string) (int, error) {
	return service.NewPlaylistExpansionService(service.PlaylistExpansionDeps{
		Playlists:    duckdb.PlaylistsSeenReader{Bases: r.titleBases()},
		OpenChildren: r.openPlaylistChildStore,
		OpenFetcher:  r.openPlaylistConfigFetcher,
	}).ExpandPlaylistChildren(ctx, titleSlug)
}

// openPlaylistChildStore ouvre metadata du titre (écriture partagée) ; release décrémente son compteur de
// références.
func (r *ServiceRegistry) openPlaylistChildStore(ctx context.Context, titleSlug string) (port.PlaylistChildStore, func(), error) {
	metaSQL, closeMeta := r.titleBases().Metadata(ctx, titleSlug)
	if metaSQL == nil {
		closeMeta()
		return nil, nil, fmt.Errorf("DBs indisponibles pour %s", titleSlug)
	}
	return persist.NewPlaylistChildrenPersister(metaSQL), closeMeta, nil
}

// openPlaylistConfigFetcher prend les jetons Halo du catalogue et construit le client qui lit la configuration
// d'une playlist.
func (r *ServiceRegistry) openPlaylistConfigFetcher(ctx context.Context, titleSlug string) (port.PlaylistConfigFetcher, error) {
	tokens, err := jetonsCatalogue(r, ctx, titleSlug)
	if err != nil {
		return nil, err
	}
	client := syncpkg.NewHaloAPIClient(tokens.SpartanToken, tokens.ClearanceToken, playlistExpandRPS)
	return port.PlaylistConfigFetcherFunc(func(ctx context.Context, p port.PlaylistRef) ([]port.PlaylistChild, error) {
		cfg, err := client.GetPlaylistConfig(ctx, p.ID, p.Version)
		if err != nil {
			return nil, err
		}
		children := make([]port.PlaylistChild, 0, len(cfg.Entries))
		for _, e := range cfg.Entries {
			children = append(children, port.PlaylistChild{PairAssetID: e.MapModePairAssetID, VersionID: e.VersionID, Weight: e.Weight})
		}
		return children, nil
	}), nil
}
