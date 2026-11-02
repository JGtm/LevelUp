package filmcache

// archive.go — L'ARCHIVE DES FILMS COMME UN TOUT : combien elle pese, comment on retire un film,
// comment on la deplace (lot L14, plan Desktop).
//
// L'UNITE DE L'ARCHIVE EST LE FILM : `film_chunks/<short8>/` ET `film_manifests/<short8>.json`
// voyagent ensemble, sous une meme racine (la racine que prend chaque fonction de ce paquet).
// Les faits decodes (`film_facts`) ne sont pas dans l'archive : ils se refont d'un film, pas
// l'inverse.

import (
	"context"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
)

// Occupation est la mesure de l'archive : combien de films, combien d'octets sur disque.
type Occupation struct {
	Films  int   // films dont le manifeste est present
	Octets int64 // taille disque des chunks et des manifestes
	// Partielle : la mesure a ete interrompue (delai ou annulation) ; Films et Octets sont des
	// minorants.
	Partielle bool
}

// MesurerOccupation parcourt l'archive `root`. Un parcours interrompu par `ctx` rend la mesure
// faite jusque-la, marquee [Occupation.Partielle], sans erreur : c'est un affichage, pas un
// controle. Une archive absente est vide.
func MesurerOccupation(ctx context.Context, root string) (Occupation, error) {
	var occ Occupation
	films, err := ListShortIDs(root)
	if err != nil {
		return occ, err
	}
	occ.Films = len(films)
	for _, dossier := range []string{ChunksRoot(root), ManifestsRoot(root)} {
		err := filepath.WalkDir(dossier, func(path string, d fs.DirEntry, err error) error {
			if cerr := ctx.Err(); cerr != nil {
				return cerr
			}
			if err != nil {
				if errors.Is(err, fs.ErrNotExist) {
					return nil // un film retire pendant la mesure
				}
				return err
			}
			if d.IsDir() {
				return nil
			}
			info, err := d.Info()
			if errors.Is(err, fs.ErrNotExist) {
				return nil
			}
			if err != nil {
				return err
			}
			occ.Octets += info.Size()
			return nil
		})
		if ctxErr := ctx.Err(); ctxErr != nil {
			occ.Partielle = true
			return occ, nil
		}
		if err != nil && !errors.Is(err, fs.ErrNotExist) {
			return occ, fmt.Errorf("filmcache: mesure de %s : %w", dossier, err)
		}
	}
	return occ, nil
}

// Supprimer retire un film de l'archive : son manifeste D'ABORD (c'est le marqueur de commit, le
// film cesse d'etre lisible d'un coup), puis ses chunks. Un film absent n'est pas une erreur.
func Supprimer(root, shortID string) error {
	if err := os.Remove(ManifestPath(root, shortID)); err != nil && !errors.Is(err, fs.ErrNotExist) {
		return fmt.Errorf("filmcache: retrait du manifeste de %s : %w", shortID, err)
	}
	if err := os.RemoveAll(ChunkDir(root, shortID)); err != nil {
		return fmt.Errorf("filmcache: retrait des chunks de %s : %w", shortID, err)
	}
	return nil
}
