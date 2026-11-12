// Package service — playlist_expansion_service.go : l'expansion « A à Z » du catalogue.
//
// Le drain du catalogue ne part que de match_registry (ce qui a été joué). Ce service enfile en plus les
// couples carte-mode ENFANTS déclarés par la configuration de chaque playlist vue en match, pour que le
// drain les nomme, et inscrit leurs poids de rotation. Il ne connaît ni la base ni le réseau : la lecture
// des playlists, la configuration de chaque playlist et l'écriture dans metadata arrivent par des ports.
package service

import (
	"context"
	"log/slog"

	"levelup/go-api/internal/observability"
	"levelup/go-api/internal/observability/logging"
	"levelup/go-api/internal/port"
)

// PlaylistExpansionDeps assemble ce que le service ne sait pas construire lui-même.
type PlaylistExpansionDeps struct {
	// Playlists lit les playlists vues en match.
	Playlists port.PlaylistsSeenReader
	// OpenChildren ouvre le dépôt d'écriture du titre (metadata) ; release le rend. Une erreur arrête
	// l'expansion avant tout accès réseau.
	OpenChildren func(ctx context.Context, titleSlug string) (store port.PlaylistChildStore, release func(), err error)
	// OpenFetcher prend les jetons et rend le lecteur de configuration de playlist : le premier accès
	// réseau de la tâche.
	OpenFetcher func(ctx context.Context, titleSlug string) (port.PlaylistConfigFetcher, error)
}

// PlaylistExpansionService enfile les couples carte-mode enfants des playlists d'un titre.
type PlaylistExpansionService struct {
	deps PlaylistExpansionDeps
}

// NewPlaylistExpansionService construit le service.
func NewPlaylistExpansionService(deps PlaylistExpansionDeps) *PlaylistExpansionService {
	return &PlaylistExpansionService{deps: deps}
}

// ExpandPlaylistChildren lit la configuration de chaque playlist vue en match (avec sa version, requise par
// l'API) et enfile ses couples carte-mode enfants dans la file de récupération du catalogue (le drain les
// nommera ensuite), en stockant les poids de rotation. Best-effort par playlist : un fetch raté est
// journalisé et n'interrompt pas les autres. Rend le nombre de couples enfants enfilés. In-process (handles
// de bases du serveur) : aucun verrou.
func (s *PlaylistExpansionService) ExpandPlaylistChildren(ctx context.Context, titleSlug string) (int, error) {
	// La base partagée n'est lue que le temps de lister les playlists, puis rendue : la boucle réseau qui
	// suit dure plusieurs minutes (320 playlists à 3 req/s) et un lecteur en vol fait échouer toute prise
	// d'écriture de la base (ADR 0016).
	playlists, err := s.deps.Playlists.PlaylistsSeenInMatches(ctx, titleSlug)
	if err != nil {
		return 0, err
	}
	store, release, err := s.deps.OpenChildren(ctx, titleSlug)
	if err != nil {
		return 0, err
	}
	defer release()
	if err := store.EnsureSchema(ctx); err != nil {
		return 0, err
	}

	fetcher, err := s.deps.OpenFetcher(ctx, titleSlug)
	if err != nil {
		return 0, err
	}

	slog.InfoContext(ctx, "catalog_expand: démarré", "module", logging.ModuleCatalog,
		"title", titleSlug, "playlists", len(playlists))

	enqueued, failed, err := expandPlaylists(ctx, titleSlug, playlists, fetcher, store)
	if err != nil {
		return enqueued, err
	}
	observability.IncCounter("catalog_playlist_expand_total")
	// Résumé unique de fin de cycle : WARN si au moins une playlist a échoué (signal agrégé visible), INFO
	// sinon.
	if failed > 0 {
		slog.WarnContext(ctx, "catalog_expand: terminé avec échecs", "module", logging.ModuleCatalog,
			"playlists", len(playlists), "children_enqueued", enqueued, "playlists_failed", failed)
	} else {
		slog.InfoContext(ctx, "catalog_expand: terminé", "module", logging.ModuleCatalog,
			"playlists", len(playlists), "children_enqueued", enqueued, "playlists_failed", failed)
	}
	return enqueued, nil
}

// expandPlaylists parcourt les playlists : pour chacune, enfile ses couples enfants et inscrit leurs poids.
// Rend les couples enfilés et le nombre de playlists dont la configuration n'a pas pu être lue ; l'erreur
// n'est que l'annulation du contexte.
func expandPlaylists(
	ctx context.Context, titleSlug string, playlists []port.PlaylistRef,
	fetcher port.PlaylistConfigFetcher, store port.PlaylistChildStore,
) (enqueued, failed int, err error) {
	for _, p := range playlists {
		if err := ctx.Err(); err != nil {
			return enqueued, failed, err
		}
		children, ferr := fetcher.PlaylistChildren(ctx, p)
		if ferr != nil {
			failed++
			// Best-effort par playlist : DEBUG. Le total est remonté une fois en fin de cycle (WARN si
			// failed>0), pas N WARN ici. Un fetch raté au boot est souvent un simple 429 transitoire (cf.
			// cooldown pool).
			slog.DebugContext(ctx, "catalog_expand: config playlist échouée (best-effort)",
				"module", logging.ModuleCatalog, "playlist", p.ID, "err", ferr)
			continue
		}
		for _, e := range children {
			if e.PairAssetID == "" {
				continue
			}
			// catalog_fetch_queue n'a plus de clé primaire (surface ART retirée) : l'enfilage dédoublonne
			// par NOT EXISTS (cf. persist.PlaylistChildrenPersister).
			if eerr := store.EnqueuePair(ctx, titleSlug, e.PairAssetID, e.VersionID); eerr != nil {
				slog.WarnContext(ctx, "catalog_expand: enqueue enfant échoué",
					"module", logging.ModuleCatalog, "pair", e.PairAssetID, "err", eerr)
				continue
			}
			enqueued++
			// Poids best-effort, recalculé au prochain cycle : l'erreur est délibérément ignorée (comme
			// l'upsert ART-safe canonique qu'elle remplace).
			_ = store.RecordWeight(ctx, titleSlug, p.ID, e.PairAssetID, e.Weight)
		}
	}
	return enqueued, failed, nil
}
