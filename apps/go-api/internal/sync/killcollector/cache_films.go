package killcollector

// cache_films.go — LA SOURCE DE FILMS HORS LIGNE : le cache disque, et rien d autre.
//
// # POURQUOI ELLE EXISTE — UNE PREMISSE DU PLAN A ETE DEMENTIE PAR LA MESURE
//
// Le guide du decodeur affirmait que « le cache disque local ne stocke QUE les chunks de
// replication » et que le kill-feed etait re-telecharge a chaque appel. **C est FAUX sur le
// cache d aujourd hui** : sur les 949 films utilisables (951 repertoires, 2 vides), les 949
// portent sur disque leur en-tete (type 1), TOUTES leurs replications (type 2) ET leur kill-feed
// (type 3) — croisement manifeste x fichiers presents, 949/949 sur les trois criteres.
//
// Consequence, et elle RETIRE un risque plutot que d en ajouter un : **le backfill n a besoin ni
// de reseau, ni de tokens, ni du CDN**, donc ni de la fenetre d authentification, ni du risque
// qu un film expire cote serveur pendant les heures ou la passe tourne.
//
// # CE QU ELLE N EST PAS
//
// Ce n est PAS un client degrade. Elle n emet aucune requete, ne connait aucun jeton, et rend
// `found = false` des qu un film manque — exactement comme un 404. Un appelant ne peut donc pas
// « retomber sur le reseau » par accident : il n y a pas de reseau derriere.

import (
	"context"
	"fmt"
	"log/slog"

	"levelup/go-api/internal/games/halo_infinite/film/filmcache"
	"levelup/go-api/internal/observability"
	"levelup/go-api/internal/sync/haloclient"
)

// CompteurCacheNonFinalise : manifestes du cache refuses parce qu'ils ne portent pas le morceau
// des temps forts (lot L3, 2026-09-23, cf. `filmcache/finalise.go`). Depuis ce lot le writer ne
// valide plus de tel manifeste : le compteur ne voit donc que ceux ecrits AVANT lui (un seul au
// parc du 2026-09-23, `ab526724`, repare le soir meme). Au-dessus de zero en exploitation, un
// autre chemin d'ecriture contourne `filmcache.Write`.
const CompteurCacheNonFinalise = "killsource_cache_manifestes_non_finalises"

// LocalCacheFilms : les films du cache disque, servis comme le ferait le client HTTP.
//
// Elle implemente [filmChunkFetcher] — le SEUL besoin du pont cote client — donc le collecteur
// ne sait pas d ou viennent les octets, et c est le but : la meme chaine de decodage et
// d ecriture sert la passe en ligne et le backfill hors ligne.
type LocalCacheFilms struct {
	cache *haloclient.LocalFilmCache
}

// NewLocalCacheFilms construit la source. `cache` nil = aucun film (toutes les lectures rendent
// `found = false`), ce qui est le comportement correct d une machine sans cache : une passe qui
// ne trouve aucun film est une passe qui ne fait rien, pas une panne.
func NewLocalCacheFilms(cache *haloclient.LocalFilmCache) *LocalCacheFilms {
	return &LocalCacheFilms{cache: cache}
}

// GetFilmChunks rend TOUS les chunks du film, avec leur type de manifeste.
//
// LE TYPE VIENT DU MANIFESTE CACHE, jamais d une deduction sur la position. La tentation
// (« chunk 0 = en-tete, dernier = kill-feed ») est fausse en general : le kill-feed d un film BTB
// est le chunk n62 sur 63, mais rien ne garantit qu il soit le DERNIER, et un film tronque
// n aurait meme pas de kill-feed du tout.
//
// Un chunk declare au manifeste mais ABSENT du disque est saute, pas fatal : le decodeur lit ce
// qu on lui donne et localise le kill-feed par son CONTENU. Un film incomplet rend donc ce qu il
// a — et si le kill-feed manque, `killsource.Decode` rendra `ErrNoKillFeed`, qui est un ETAT
// et pas une panne.
//
// UN MANIFESTE NON FINALISE N EST PAS SERVI (lot L3, 2026-09-23) : sans le morceau des temps
// forts, il decrit un film en cours de publication, archive trop tot. Le servir decoderait un
// film TRONQUE a chaque cycle (`ab526724` : 276 passes « sans kill-feed » en 13 h) ; le refus
// rend [filmcache.ErrFilmNonFinalise]. EN LIGNE, `RemoteFilms` retombe alors sur le reseau, et
// son archivage COMPLETE le manifeste : le film se repare seul. HORS LIGNE, l erreur ne pose
// AUCUN marqueur de registre — la ou un « absent » poserait `MBitFilmAbsent`, terminal.
func (l *LocalCacheFilms) GetFilmChunks(
	ctx context.Context, matchID string,
) ([]haloclient.FilmChunk, bool, error) {
	if l == nil || l.cache == nil {
		return nil, false, nil
	}
	manifest, err := l.cache.LoadManifest(matchID)
	if err != nil {
		return nil, false, fmt.Errorf("manifeste cache %s: %w", matchID, err)
	}
	if manifest == nil || len(manifest.Chunks) == 0 {
		return nil, false, nil
	}
	if !filmcache.Finalise(manifest.Chunks, func(c haloclient.CachedChunk) int { return c.ChunkType }) {
		observability.IncCounter(CompteurCacheNonFinalise)
		slog.InfoContext(ctx, "killsource: manifeste du cache non finalise (sans temps forts) — "+
			"film non servi par le disque", "match_id", matchID, "entrees", len(manifest.Chunks))
		return nil, false, fmt.Errorf("manifeste cache %s (%d entrees) : %w", matchID,
			len(manifest.Chunks), filmcache.ErrFilmNonFinalise)
	}

	out := make([]haloclient.FilmChunk, 0, len(manifest.Chunks))
	for _, ch := range manifest.Chunks {
		data, cErr := l.cache.LoadChunk(matchID, ch.Index)
		if cErr != nil {
			return nil, false, fmt.Errorf("chunk %d du film %s: %w", ch.Index, matchID, cErr)
		}
		if len(data) == 0 {
			continue // declare au manifeste, absent du disque
		}
		out = append(out, haloclient.FilmChunk{
			Index:      ch.Index,
			ChunkType:  ch.ChunkType,
			Data:       data,
			StartMS:    ch.StartMS,
			DurationMS: ch.DurationMS,
		})
	}
	if len(out) == 0 {
		return nil, false, nil
	}
	return out, true, nil
}
