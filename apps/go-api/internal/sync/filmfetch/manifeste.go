package filmfetch

import (
	"context"
	"fmt"
	"log/slog"

	"levelup/go-api/internal/haloapi/infinite"

	"github.com/JGtm/LevelUp/libs/film/decfilm"
	"github.com/JGtm/LevelUp/libs/film/finalise"
)

// fetchFilmManifest rend le manifeste du film d'un match : (manifeste, true, nil) si disponible,
// (nil, false, nil) si absent (404/410).
//
// Le manifeste du cache disque passe AVANT l'API : il survit à l'expiration de l'adresse du
// manifeste côté Halo (les blobs du CDN vivent plus longtemps). LA VERSION DU FILM VIENT ALORS DU
// FILM LUI-MÊME : le manifeste en cache ne porte pas `FilmMajorVersion`, elle est lue dans
// l'en-tête du registre (`chunk_00`), qui est le film et vaut pour tout le cache sans migration.
//
// UN MANIFESTE DU CACHE NON FINALISÉ NE MASQUE PAS L'API : écrit avant la règle « seul un film
// finalisé se valide », il décrit un film tronqué ; le servir ferait refuser le film à chaque
// cycle sans que le serveur, qui l'a finalisé depuis, soit jamais consulté.
func (f *Films) fetchFilmManifest(ctx context.Context, matchID string) (*infinite.FilmManifestDTO, bool, error) {
	if !infinite.EstUUID(matchID) {
		return nil, false, fmt.Errorf("fetchFilmManifest: matchID invalide %q", matchID)
	}
	cm, err := f.cache.LoadManifest(matchID)
	if err == nil && cm != nil && len(cm.Chunks) > 0 && !finalise.Finalise(cm.Chunks, typeDuCache) {
		slog.InfoContext(ctx, "film: manifeste du cache non finalisé (sans temps forts) — relu à l'API",
			"match_id", matchID, "entrees", len(cm.Chunks))
		cm = nil
	}
	if err == nil && cm != nil {
		return f.manifesteDuCache(ctx, matchID, cm), true, nil
	}

	manifest, err := f.manifeste.GetFilmManifest(ctx, f.auth, matchID)
	if err != nil {
		if infinite.IsNotFound(err) {
			return nil, false, nil
		}
		return nil, false, err
	}
	return manifest, true, nil
}

// manifesteDuCache rend le manifeste de la brique décrit par un manifeste du cache disque (sans
// taille annoncée : le cache ne la porte pas, aucun contrôle de taille).
func (f *Films) manifesteDuCache(ctx context.Context, matchID string, cm *CachedManifest) *infinite.FilmManifestDTO {
	manifest := &infinite.FilmManifestDTO{BlobStoragePathPrefix: cm.BlobPrefix}
	manifest.CustomData.FilmMajorVersion = f.filmMajorVersionDuCache(ctx, matchID)
	manifest.CustomData.Chunks = make([]infinite.FilmChunkMetaDTO, 0, len(cm.Chunks))
	for _, ch := range cm.Chunks {
		manifest.CustomData.Chunks = append(manifest.CustomData.Chunks, infinite.FilmChunkMetaDTO{
			Index:                            ch.Index,
			ChunkType:                        ch.ChunkType,
			ChunkStartTimeOffsetMilliseconds: ch.StartMS,
			DurationMilliseconds:             ch.DurationMS,
			FileRelativePath:                 ch.FileRelativePath,
		})
	}
	return manifest
}

// filmMajorVersionDuCache lit le `FilmMajorVersion` d'un film en cache dans l'en-tête de son
// registre (`chunk_00`). Registre absent ou illisible : [decfilm.FilmMajorVersionUnknown], et la
// dégradation est consignée — le décodeur du kill-feed retombe alors sur le découpage historique
// du gamertag, ce qui est faux sur un film de version 39-40.
//
// `decfilm.Inflate` rend le tampon inchangé quand il n'est pas zlib (le registre du cache ne l'est
// pas) : la tolérance ne coûte rien et couvre le cache historique.
func (f *Films) filmMajorVersionDuCache(ctx context.Context, matchID string) int {
	registre, err := f.cache.LoadChunk(matchID, 0)
	if err != nil {
		slog.WarnContext(ctx, "film: registre illisible, version de film inconnue",
			"match_id", matchID, "err", err)
		return decfilm.FilmMajorVersionUnknown
	}
	version, ok := decfilm.FilmMajorVersionFromHeader(decfilm.Inflate(registre))
	if !ok {
		slog.WarnContext(ctx, "film: registre absent du cache, version de film inconnue",
			"match_id", matchID)
		return decfilm.FilmMajorVersionUnknown
	}
	return version
}

// refuserSiNonFinalise rend [finalise.ErrFilmNonFinalise] quand le manifeste décrit un film en
// cours de publication : des morceaux, mais pas celui des temps forts.
//
// NI 404 NI PANNE. Le film existe et sera complet dans une minute : l'appelant reporte au cycle
// suivant. Un manifeste à ZÉRO morceau, lui, reste un film ABSENT (expiré) : `killcollector` en
// tire son marqueur terminal.
func refuserSiNonFinalise(manifest *infinite.FilmManifestDTO, caller, matchID string) error {
	chunks := manifest.CustomData.Chunks
	if len(chunks) == 0 || finalise.Finalise(chunks, typeDuManifesteAPI) {
		return nil
	}
	return fmt.Errorf("%s(%s) : %d morceaux au manifeste : %w", caller, matchID, len(chunks),
		finalise.ErrFilmNonFinalise)
}

// typeDuManifesteAPI / typeDuCache : les accesseurs de type que [finalise.Finalise] reçoit.
func typeDuManifesteAPI(c infinite.FilmChunkMetaDTO) int { return c.ChunkType }
func typeDuCache(c CachedChunk) int                      { return c.ChunkType }
