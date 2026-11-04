package filmcache

// archive.go — L'ARCHIVE DES FILMS COMME UN TOUT : combien elle pese. La retirer, la deplacer ou
// la recompresser est l'affaire de l'hote.
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
