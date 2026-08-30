package killcollector

// remote_films.go — LA SOURCE DE FILMS EN LIGNE, QUI ARCHIVE CE QU ELLE LIT.
//
// # POURQUOI ELLE EXISTE — LE CACHE AVAIT CESSE D ETRE ALIMENTE
//
// Jusqu ici, le decodage de kill-source n avait qu UNE source : [LocalCacheFilms], le cache
// disque. Or ce cache etait herite du projet Python supprime a la migration, et AUCUN code Go
// ne creait de nouveau manifeste. Mesure du 2026-08-29 (registre
// `.ai/V7.5/REGISTRE_ASSISTANCES_2026-08-29.md`) : le cache s arrete le 2026-04-07, et depuis
// cette date `assist_known` vaut FALSE sur 100 % des matchs synchronises — parce que
// l assistant ne se lit QUE dans le kill-feed du film. Deux blocs de l app (« qui assiste
// qui » de la page Escouade, assistances de la vue match) se sont retires en silence pendant
// cinq mois.
//
// Cette source ferme le trou par le seul endroit ou il pouvait l etre : elle va CHERCHER le
// film quand il n est pas sur disque.
//
// # ELLE ARCHIVE, ET CE N EST PAS UN EFFET DE BORD — C EST LA MOITIE DE SON TRAVAIL
//
// Les films EXPIRENT cote serveur Halo. Un film telecharge puis jete devrait etre
// re-telecharge a chaque re-decodage, et le jour ou il expire il serait perdu alors qu on l a
// EU. Chaque film lu est donc ecrit au cache par [filmcache.Write] (chunks d abord, manifeste
// en dernier : le manifeste est le marqueur de commit). Une passe suivante — decodeur revise,
// nouvelle mesure — retombe alors sur [LocalCacheFilms] et ne paie plus le reseau.
//
// L echec d archivage n est PAS fatal : les octets sont en memoire, le decodage peut se faire.
// Il est logge et compte (`killsource_films_archive_erreurs`), jamais avale — un disque plein
// qui ferait echouer silencieusement l archivage rendrait la passe suivante aussi chere que
// celle-ci, sans que rien ne le dise.
//
// # ELLE PREFERE LE DISQUE, TOUJOURS
//
// Un film deja en cache n est jamais retelecharge : le film est IMMUABLE cote serveur, et une
// passe `--force` sur 400 matchs qui re-tirerait 10 Go du CDN serait une faute. Le repli
// disque -> reseau est donc dans ce sens, et dans ce sens seulement.

import (
	"context"
	"log/slog"

	titlePkg "levelup/go-api/internal/domain/title"
	"levelup/go-api/internal/games/halo_infinite/film/filmcache"
	"levelup/go-api/internal/observability"
	"levelup/go-api/internal/sync/haloclient"
)

// Compteurs de la source en ligne, publies en expvar (ADR 0009).
const (
	CompteurFilmsDepuisCache = "killsource_films_depuis_cache"
	CompteurFilmsTelecharges = "killsource_films_telecharges"
	CompteurFilmsArchives    = "killsource_films_archives"
	CompteurArchiveErreurs   = "killsource_films_archive_erreurs"
)

// RemoteFilms : le cache disque d abord, le reseau ensuite, et le reseau archive au passage.
//
// Elle implemente [filmChunkFetcher], exactement comme [LocalCacheFilms] : le collecteur ne
// sait pas d ou viennent les octets. C est ce qui permet a la MEME chaine de decodage et
// d ecriture de servir la passe hors ligne et la passe en ligne.
type RemoteFilms struct {
	local     *LocalCacheFilms
	distant   filmChunkFetcher
	cacheRoot string
}

// NewRemoteFilms construit la source.
//
//	local     : peut etre nil (machine sans cache) — tout part alors au reseau.
//	distant   : le client Halo authentifie. nil = source inerte (aucun film), pas une panne.
//	cacheRoot : la racine ou archiver. Vide = aucun archivage (et un WARN au premier film,
//	            parce qu une passe qui telecharge sans archiver perd une donnee irremplacable).
func NewRemoteFilms(local *LocalCacheFilms, distant filmChunkFetcher, cacheRoot string) *RemoteFilms {
	return &RemoteFilms{local: local, distant: distant, cacheRoot: cacheRoot}
}

// GetFilmChunks rend tous les chunks du film, avec leur type de manifeste.
//
// `found = false` quand le film n existe ni sur disque ni cote serveur (404/410) : c est un
// ETAT NORMAL — un film expire n est pas une panne — et le collecteur le compte en
// `killsource_films_absents`.
func (r *RemoteFilms) GetFilmChunks(
	ctx context.Context, matchID string,
) ([]haloclient.FilmChunk, bool, error) {
	if r == nil {
		return nil, false, nil
	}
	// Le disque d abord : le film est immuable, le retelecharger serait payer pour rien.
	if chunks, found, err := r.local.GetFilmChunks(ctx, matchID); err == nil && found {
		observability.IncCounter(CompteurFilmsDepuisCache)
		return chunks, true, nil
	} else if err != nil {
		// Un cache illisible (manifeste corrompu) ne doit pas empecher le reseau de servir :
		// on le signale et on continue, plutot que de faire echouer tout le match.
		slog.WarnContext(ctx, "killsource_cache_illisible_repli_reseau",
			"match_id", matchID, "err", err)
	}
	if r.distant == nil {
		return nil, false, nil
	}

	chunks, found, err := r.distant.GetFilmChunks(ctx, matchID)
	if err != nil || !found || len(chunks) == 0 {
		return chunks, found && len(chunks) > 0, err
	}
	observability.IncCounter(CompteurFilmsTelecharges)
	r.archiver(ctx, matchID, chunks)
	return chunks, true, nil
}

// archiver ecrit le film au cache. Best-effort SIGNALE : l echec n empeche pas le decodage
// (les octets sont en memoire) mais il est logge ET compte — sans quoi un disque plein
// ferait payer le reseau a chaque passe sans que personne ne le sache.
func (r *RemoteFilms) archiver(ctx context.Context, matchID string, chunks []haloclient.FilmChunk) {
	if r.cacheRoot == "" {
		slog.WarnContext(ctx, "killsource_film_non_archive_faute_de_cache",
			"match_id", matchID,
			"consequence", "film irremplacable non conserve — la passe suivante repaiera le reseau")
		return
	}
	wc := make([]filmcache.WriteChunk, 0, len(chunks))
	for _, c := range chunks {
		wc = append(wc, filmcache.WriteChunk{
			Index: c.Index, ChunkType: c.ChunkType, StartMS: c.StartMS, DurationMS: c.DurationMS,
			Data: c.Data,
		})
	}
	if err := filmcache.Write(r.cacheRoot, titlePkg.FilmShortMatchID(matchID), wc); err != nil {
		observability.IncCounter(CompteurArchiveErreurs)
		slog.ErrorContext(ctx, "killsource_film_archive_echec", "match_id", matchID, "err", err)
		return
	}
	observability.IncCounter(CompteurFilmsArchives)
}
