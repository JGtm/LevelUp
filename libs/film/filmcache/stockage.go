package filmcache

// stockage.go — LA FORME D'UN CHUNK SUR DISQUE (decision D-18, 2026-11-01).
//
// Un chunk du cache est brut, zlib (cache historique) ou zstd-3 (ecrit par l'ecrivain de l'hote depuis D-18) ;
// la couche `source` reconnait les trois a l'en-tete et c'est elle seule qui touche aux octets
// (cf. `source/zstd.go`). Ce fichier porte ce qu'en deduit le cache : la TAILLE LOGIQUE d'un
// fichier, celle qu'on compare au `size_bytes` du manifeste.
//
// `size_bytes` est la taille de ce que le writer a RECU (les octets decompresses), jamais celle du
// fichier ecrit : un manifeste ecrit avant D-18 reste donc valide apres la recompression d'un
// chunk brut, et la recompression n'a pas a le reecrire.

import (
	"errors"
	"fmt"
	"io"
	"os"

	"github.com/JGtm/LevelUp/libs/film/internal/source"
)

// TailleLogiqueInconnue : le fichier est une trame zstd qui ne declare pas sa taille (ecrite par
// un autre compresseur que le notre). Aucun controle de troncature n'est possible dessus.
const TailleLogiqueInconnue int64 = -1

// TailleLogique rend la taille logique du chunk `path`, de taille disque `tailleDisque` :
// la taille declaree par l'en-tete pour une trame zstd, la taille du fichier sinon.
func TailleLogique(path string, tailleDisque int64) (int64, error) {
	if tailleDisque < 4 {
		return tailleDisque, nil
	}
	f, err := os.Open(path) //nolint:gosec // chemin compose par CheminDuChunk
	if err != nil {
		return 0, err
	}
	defer func() { _ = f.Close() }()
	debut := make([]byte, min(tailleDisque, int64(source.TailleEnteteZstd)))
	if _, err := io.ReadFull(f, debut); err != nil {
		return 0, fmt.Errorf("lecture de l'en-tete de %s : %w", path, err)
	}
	if !source.EstZstd(debut) {
		return tailleDisque, nil
	}
	taille, err := source.TailleLogiqueZstd(debut)
	if errors.Is(err, source.ErrZstdTailleInconnue) {
		return TailleLogiqueInconnue, nil
	}
	if err != nil {
		return 0, fmt.Errorf("en-tete zstd de %s : %w", path, err)
	}
	return taille, nil
}

// octetsLogiques rend les octets du chunk tels que le writer les avait recus : decodes quand le
// fichier est une trame zstd, tels quels sinon (brut ou zlib, que la couche `source` decompresse).
func octetsLogiques(fichier []byte) ([]byte, error) {
	if !source.EstZstd(fichier) {
		return fichier, nil
	}
	return source.DecompresserZstd(fichier)
}

// OctetsLogiquesStricts rend les octets d'un chunk NON zstd : decompresses s'il est en zlib, tels
// quels sinon. Un flux zlib dont l'en-tete est bon mais qui casse est une ERREUR : celui qui
// recompresserait ces octets figerait un partiel.
func OctetsLogiquesStricts(fichier []byte) ([]byte, error) {
	if len(fichier) < 2 || fichier[0] != 0x78 {
		return fichier, nil
	}
	out, err := source.Decompresser(fichier)
	if errors.Is(err, source.ErrEnTeteZlib) {
		return fichier, nil // 0x78 par hasard : un chunk clair
	}
	return out, err
}

// LA FORME ZSTD D'UN CHUNK, pour qui ecrit ou recompresse le cache (l'hote : la brique ne fait que
// lire). Ce sont les fonctions de la couche `source`, qui seule touche aux octets d'un film.

// TailleEnteteZstd : le nombre d'octets de tete qui suffisent a [EstZstd] et [TailleLogiqueZstd].
const TailleEnteteZstd = source.TailleEnteteZstd

// ErrZstdTailleInconnue : la trame zstd ne declare pas sa taille dans son en-tete.
var ErrZstdTailleInconnue = source.ErrZstdTailleInconnue

// EstZstd dit si les octets commencent par l'en-tete d'une trame zstd.
func EstZstd(octets []byte) bool { return source.EstZstd(octets) }

// CompresserZstd rend la trame zstd (niveau du cache) des octets.
func CompresserZstd(octets []byte) ([]byte, error) { return source.CompresserZstd(octets) }

// DecompresserZstd rend les octets d'une trame zstd entiere.
func DecompresserZstd(trame []byte) ([]byte, error) { return source.DecompresserZstd(trame) }

// TailleLogiqueZstd rend la taille que l'en-tete de la trame declare ([ErrZstdTailleInconnue] s'il
// n'en declare pas).
func TailleLogiqueZstd(debut []byte) (int64, error) { return source.TailleLogiqueZstd(debut) }
