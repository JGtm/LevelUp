package filmcache

// stockage.go — LA FORME D'UN CHUNK SUR DISQUE (decision D-18, 2026-11-01).
//
// Un chunk du cache est brut, zlib (cache historique) ou zstd-3 (ecrit par [Write] depuis D-18) ;
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

// tailleLogiqueInconnue : le fichier est une trame zstd qui ne declare pas sa taille (ecrite par
// un autre compresseur que le notre). Aucun controle de troncature n'est possible dessus.
const tailleLogiqueInconnue int64 = -1

// tailleLogiqueFichier rend la taille logique du chunk `path`, de taille disque `tailleDisque` :
// la taille declaree par l'en-tete pour une trame zstd, la taille du fichier sinon.
func tailleLogiqueFichier(path string, tailleDisque int64) (int64, error) {
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
		return tailleLogiqueInconnue, nil
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

// octetsAEcrire rend la forme de stockage d'un chunk : zstd-3 quand elle est plus courte que les
// octets, les octets tels quels sinon (un chunk vide ou incompressible). `comprime` dit laquelle.
func octetsAEcrire(donnees []byte) (stocke []byte, comprime bool, err error) {
	z, err := source.CompresserZstd(donnees)
	if err != nil {
		return donnees, false, err
	}
	if len(z) == 0 || len(z) >= len(donnees) {
		return donnees, false, nil
	}
	return z, true, nil
}
