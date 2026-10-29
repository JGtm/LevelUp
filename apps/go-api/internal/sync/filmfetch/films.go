package filmfetch

import (
	"context"
	"fmt"
	"sort"

	"levelup/go-api/internal/haloapi/infinite"

	"github.com/JGtm/LevelUp/libs/film/finalise"
)

// Films lit les films d'Halo Infinite : cache disque d'abord, puis la brique (manifeste et blobs),
// décompression, finalisation. Valeur immuable une fois construite ; sûre en concurrence tant que
// les clients reçus le sont.
type Films struct {
	// manifeste : le client des services authentifiés (manifeste du film).
	manifeste *infinite.Client
	// blobs : le client du CDN des films (blobs, sans jeton) ; porte les compteurs de ce paquet.
	blobs *infinite.Client
	auth  infinite.Auth
	// cache : le cache disque local ; nil = désactivé.
	cache *LocalFilmCache
}

// New construit le lecteur sur deux clients de la brique : `manifeste` (services authentifiés,
// jetons `auth`) et `blobs` (CDN public). Les deux portent la politique de reprise de l'appelant ;
// les compteurs expvar du téléchargement sont posés ici sur `blobs`. Les deux clients sont
// distincts parce que leur sortie réseau est gardée sous deux surfaces différentes.
func New(manifeste, blobs *infinite.Client, auth infinite.Auth) *Films {
	return &Films{manifeste: manifeste, blobs: blobs.WithCrochets(crochets()), auth: auth}
}

// WithLocalFilmCache rend une copie du lecteur sur le cache disque `cache` (nil : désactivé).
func (f *Films) WithLocalFilmCache(cache *LocalFilmCache) *Films {
	clone := *f
	clone.cache = cache
	return &clone
}

// GetMatchFilm télécharge le manifeste du film d'un match et rend ses chunks REPLICATION_DATA
// (ChunkType 2, ceux que lit le scanner d'armes), par index. (nil, false, nil) : film absent
// (404/410), normal pour les vieux matchs. La fonction ne rend la main qu'une fois TOUS les chunks
// obtenus (téléchargement parallèle borné par la brique).
func (f *Films) GetMatchFilm(ctx context.Context, matchID string) (map[int]FilmChunkData, bool, error) {
	chunks, found, err := f.fetchFilmChunks(ctx, matchID, "GetMatchFilm",
		func(t int) bool { return t == FilmChunkTypeReplicationData })
	if err != nil || !found {
		return nil, found, err
	}
	result := make(map[int]FilmChunkData, len(chunks))
	for _, ch := range chunks {
		result[ch.Index] = FilmChunkData{Data: ch.Data, StartMS: ch.StartMS, DurationMS: ch.DurationMS}
	}
	return result, true, nil
}

// GetFilmChunks télécharge TOUS les chunks du film (types 1/2/3) en UNE seule lecture de manifeste,
// triés par index : la séquence complète (l'en-tête compris) que les deux vues par type ne
// rendent pas, sans le second aller-retour de manifeste qu'elles coûteraient.
//
// (nil, false, nil) : film absent (404/410), cas NORMAL.
func (f *Films) GetFilmChunks(ctx context.Context, matchID string) ([]FilmChunk, bool, error) {
	return f.fetchFilmChunks(ctx, matchID, "GetFilmChunks", func(int) bool { return true })
}

// fetchFilmChunks : le chemin de téléchargement COMMUN (cache disque d'abord, puis CDN en
// parallèle par la brique), filtré par type. Les chunks du cache sont rendus tels quels ; les
// autres sont téléchargés (octets compressés) puis décompressés. La première erreur de
// téléchargement annule les autres et est rendue, préfixée par `caller`, l'index et le match.
func (f *Films) fetchFilmChunks(
	ctx context.Context, matchID, caller string, keep func(chunkType int) bool,
) ([]FilmChunk, bool, error) {
	manifest, found, err := f.fetchFilmManifest(ctx, matchID)
	if err != nil || !found {
		return nil, found, err
	}
	if err := refuserSiNonFinalise(manifest, caller, matchID); err != nil {
		return nil, false, err
	}

	var out []FilmChunk
	var toDownload []infinite.FilmChunkMetaDTO
	for _, chunk := range manifest.CustomData.Chunks {
		if !keep(chunk.ChunkType) {
			continue
		}
		// Cache disque d'abord. Le cache hérité ne porte SYSTÉMATIQUEMENT que les
		// REPLICATION_DATA ; l'en-tête et le kill-feed n'y sont que sur les films téléchargés à la
		// main. On tente donc pour tous les types : un miss retombe sur le CDN.
		if cached := f.chunkDuCache(ctx, matchID, chunk.Index); cached != nil {
			out = append(out, FilmChunk{Index: chunk.Index, ChunkType: chunk.ChunkType, Data: cached,
				StartMS: chunk.ChunkStartTimeOffsetMilliseconds, DurationMS: chunk.DurationMilliseconds})
			continue
		}
		toDownload = append(toDownload, chunk)
	}

	if len(toDownload) > 0 {
		downloaded, dErr := f.telecharger(ctx, manifest.BlobStoragePathPrefix, toDownload, caller, matchID)
		if dErr != nil {
			return nil, false, dErr
		}
		out = append(out, downloaded...)
	}

	if len(out) == 0 {
		return nil, false, nil
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Index < out[j].Index })
	return out, true, nil
}

// telecharger obtient les blobs `chunks` par la brique (en parallèle) et les décompresse ; l'erreur
// d'un blob illisible porte le même préfixe que celle d'un blob en échec.
func (f *Films) telecharger(ctx context.Context, prefixe string, chunks []infinite.FilmChunkMetaDTO, caller, matchID string) ([]FilmChunk, error) {
	bruts, err := f.blobs.DownloadChunks(ctx, prefixe, chunks, caller, matchID)
	if err != nil {
		return nil, err
	}
	// Chaque blob est décompressé (une reprise ne compte que sur un blob lisible), puis la première
	// erreur de décompression est rendue.
	out := make([]FilmChunk, len(bruts))
	var premiere error
	for i, b := range bruts {
		data, iErr := inflateBlob(b.Brut)
		if iErr != nil {
			if premiere == nil {
				premiere = fmt.Errorf("%s chunk %d(%s): %w", caller, b.Index, matchID, iErr)
			}
			continue
		}
		compterReprise(b.Essais)
		out[i] = FilmChunk{Index: b.Index, ChunkType: b.ChunkType, Data: data, StartMS: b.StartMS, DurationMS: b.DurationMS}
	}
	if premiere != nil {
		return nil, premiere
	}
	return out, nil
}

// GetFilmChunkURLs résout le manifeste d'un match et rend les RÉFÉRENCES de ses chunks, sans
// télécharger un seul octet : le manifeste exige les jetons Halo, les blobs non (CDN pré-signé).
// Le serveur résout donc ici et l'ouvrier distant décode sans le moindre secret Halo.
//
// (nil, false, nil) : film absent (404/410) — cas NORMAL.
func (f *Films) GetFilmChunkURLs(ctx context.Context, matchID string) ([]FilmChunkRef, bool, error) {
	manifest, found, err := f.fetchFilmManifest(ctx, matchID)
	if err != nil || !found {
		return nil, found, err
	}
	// MÊME RÈGLE QUE LE TÉLÉCHARGEMENT : l'ouvrier archive puis cuit ce qu'on lui confie, et son
	// writer refuserait un film non finalisé — autant ne pas mettre en file un travail condamné.
	if err := refuserSiNonFinalise(manifest, "GetFilmChunkURLs", matchID); err != nil {
		return nil, false, err
	}
	out := manifest.Refs() // un chunk sans préfixe de blob (cache écrit localement) est omis
	if len(out) == 0 {
		return nil, false, nil
	}
	return out, true, nil
}

// GetHighlightEventsChunk télécharge le chunk des temps forts (ChunkType 3) du film.
// (data, filmMajorVersion, true, nil) si disponible ; (nil, 0, false, nil) si le film est absent,
// sans chunk des temps forts, ou si son blob a expiré du CDN (manifeste venu du cache).
func (f *Films) GetHighlightEventsChunk(ctx context.Context, matchID string) ([]byte, int, bool, error) {
	manifest, found, err := f.fetchFilmManifest(ctx, matchID)
	if err != nil || !found {
		return nil, 0, found, err
	}
	for _, chunk := range manifest.CustomData.Chunks {
		if !finalise.EstTempsForts(chunk.ChunkType) {
			continue
		}
		// Cache disque d'abord (rarement présent — le cache hérité ne porte que REPLICATION_DATA).
		if cached := f.chunkDuCache(ctx, matchID, chunk.Index); cached != nil {
			return cached, manifest.CustomData.FilmMajorVersion, true, nil
		}
		data, err := f.downloadBlob(ctx, infinite.BuildChunkURL(manifest.BlobStoragePathPrefix, chunk.FileRelativePath), chunk.ChunkSize)
		if err != nil {
			// Blob expiré du CDN alors que le manifeste (du cache) répond : équivalent à « film
			// absent », pas une erreur.
			if infinite.IsNotFound(err) {
				return nil, 0, false, nil
			}
			return nil, 0, false, fmt.Errorf("GetHighlightEventsChunk(%s): %w", matchID, err)
		}
		return data, manifest.CustomData.FilmMajorVersion, true, nil
	}
	return nil, 0, false, nil
}
