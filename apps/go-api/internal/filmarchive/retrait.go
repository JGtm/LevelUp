package filmarchive

import (
	"errors"
	"fmt"
	"io/fs"
	"os"

	"github.com/JGtm/halo-infinite-film-decoder/filmcache"
)

// Supprimer retire un film de l'archive : son manifeste D'ABORD (c'est le marqueur de commit, le
// film cesse d'etre lisible d'un coup), puis ses chunks. Un film absent n'est pas une erreur.
func Supprimer(root, shortID string) error {
	if err := os.Remove(filmcache.ManifestPath(root, shortID)); err != nil && !errors.Is(err, fs.ErrNotExist) {
		return fmt.Errorf("filmcache: retrait du manifeste de %s : %w", shortID, err)
	}
	if err := os.RemoveAll(filmcache.ChunkDir(root, shortID)); err != nil {
		return fmt.Errorf("filmcache: retrait des chunks de %s : %w", shortID, err)
	}
	return nil
}
