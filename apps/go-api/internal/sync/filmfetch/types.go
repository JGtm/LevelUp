package filmfetch

import (
	"levelup/go-api/internal/haloapi/infinite"

	"github.com/JGtm/LevelUp/libs/film/finalise"
)

// Constantes pour les types de chunks film Halo.
//
// EXPORTÉES parce qu'un consommateur hors de ce paquet doit pouvoir SÉLECTIONNER un type sans
// recopier le nombre : la ventilation des tirs (`sync/killcollector`) ne scanne que la
// REPLICATION_DATA, la population sur laquelle la loi « un fire-event = un tir » a été mesurée.
// Une copie locale du littéral 2 serait un magic number de plus, et il dériverait.
//
// LE TYPE DES TEMPS FORTS REPREND CELUI DE `filmcache` : c'est le type qui dit qu'un film est
// FINALISÉ ([finalise.Finalise]), et le prédicat vit là où le writer du cache peut le lire. Un
// sélecteur qui compare au type 3 passe par [finalise.EstTempsForts] (garde-rail
// `archlint/film_finalise_predicate_test.go`).
const (
	FilmChunkTypeHeader          = 1
	FilmChunkTypeReplicationData = 2
	FilmChunkTypeHighlightEvents = finalise.ChunkTypeTempsForts
)

// FilmChunk : un chunk du film AVEC son type de manifeste, octets DÉCOMPRESSÉS. C'est la forme dont
// a besoin un consommateur qui veut la SÉQUENCE COMPLÈTE (en-tête + réplication + kill-feed) et pas
// seulement un type — cf. [Films.GetFilmChunks].
type FilmChunk struct {
	Index      int
	ChunkType  int
	Data       []byte
	StartMS    int
	DurationMS int
}

// FilmChunkData encapsule les données binaires (décompressées) d'un chunk film.
type FilmChunkData struct {
	Data       []byte
	StartMS    int
	DurationMS int
}

// FilmChunkRef : un chunk du film et son URL CDN PRÉ-SIGNÉE, SANS ses octets.
type FilmChunkRef = infinite.FilmChunkRef

// BlobHTTPError : échec HTTP DÉFINITIF d'un blob du CDN des films. Type distinct de l'erreur des
// services authentifiés : un 401/403/503 du CDN public ne dit rien des jetons (voir
// [infinite.BlobHTTPError]).
type BlobHTTPError = infinite.BlobHTTPError

// ErrChunkIncomplet : un blob servi d'une autre taille que celle que le manifeste annonce (voir
// [infinite.ErrChunkIncomplet]) ; ce n'est pas un film perdu, il se retente au cycle suivant.
type ErrChunkIncomplet = infinite.ErrChunkIncomplet

// IsFilmGoneErr dit si l'erreur signale un film DÉFINITIVEMENT perdu — manifeste OU blobs, peu
// importe lequel : 404/410 typé (erreur de service pour le manifeste, [BlobHTTPError] pour un blob)
// sur l'un ou l'autre compte comme définitif. Il sert aux appelants qui RETENTENT un film au lieu
// de simplement le classer absent/présent : eux ont besoin de distinguer une erreur transitoire
// (réseau, 5xx, rate limit) d'un lien mort.
//
// EXPIRATION PARTIELLE = DÉFINITIVE, biais assumé. Le manifeste et les blobs pré-signés expirent
// sur des calendriers séparés : UN chunk sur N qui rend 404 remonte seul de la récupération
// parallèle, même si un autre rendait 503 en même temps. Un faux « transitoire » se corrige à la
// passe suivante ; un faux « définitif » serait un film vivant classé perdu : le biais va du bon
// côté, mais un appelant qui a besoin de la distinction fine (un chunk isolé mort à côté de chunks
// lisibles) ne doit pas se fier à ce seul prédicat. Le verdict est TYPÉ, jamais lu dans le texte
// de l'erreur.
func IsFilmGoneErr(err error) bool { return infinite.IsFilmGoneErr(err) }
