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
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
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

// ErrCibleNonVide : la destination d'un deplacement contient deja quelque chose. Deplacer sur un
// dossier non vide melangerait deux archives ; l'appelant le refuse et le dit a l'utilisateur.
var ErrCibleNonVide = errors.New("filmcache: le dossier cible n'est pas vide")

// ValiderCible controle la destination d'un deplacement de l'archive : un chemin ABSOLU, absent
// ou vide. [ErrCibleNonVide] quand il contient quoi que ce soit.
func ValiderCible(cible string) error {
	if strings.TrimSpace(cible) == "" || !filepath.IsAbs(cible) {
		return fmt.Errorf("filmcache: le dossier cible doit etre un chemin absolu (%q)", cible)
	}
	entrees, err := os.ReadDir(cible)
	switch {
	case errors.Is(err, fs.ErrNotExist):
		return nil
	case err != nil:
		return fmt.Errorf("filmcache: dossier cible illisible (%s) : %w", cible, err)
	case len(entrees) > 0:
		return fmt.Errorf("%w : %s", ErrCibleNonVide, cible)
	}
	return nil
}

// Deplacer range les films de l'archive `origine` dans `cible`, un film a la fois : chunks d'abord,
// manifeste en dernier (le marqueur de commit), puis le film est retire de la source. Interrompu,
// il laisse chaque film entier d'un cote ou de l'autre : relancer reprend les films restants
// (la cible n'est alors plus vide, d'ou le parametre `reprise`). Rend le nombre de films deplaces.
//
// Le renommage est tente d'abord (meme volume : instantane) ; sinon le film est copie, relu a la
// taille, puis retire de la source.
func Deplacer(ctx context.Context, origine, cible string, reprise bool) (int, error) {
	if !reprise {
		if err := ValiderCible(cible); err != nil {
			return 0, err
		}
	}
	films, err := ListShortIDs(origine)
	if err != nil {
		return 0, err
	}
	if err := EnsureDirs(cible); err != nil {
		return 0, err
	}
	deplaces := 0
	for _, id := range films {
		if err := ctx.Err(); err != nil {
			return deplaces, err
		}
		if err := deplacerFilm(origine, cible, id); err != nil {
			return deplaces, err
		}
		deplaces++
	}
	return deplaces, nil
}

// deplacerFilm range UN film. Chunks puis manifeste a la cible, puis retrait de la source.
func deplacerFilm(origine, cible, id string) error {
	if err := os.MkdirAll(ChunkDir(cible, id), 0o755); err != nil {
		return fmt.Errorf("filmcache: dossier de %s : %w", id, err)
	}
	entrees, err := os.ReadDir(ChunkDir(origine, id))
	if err != nil && !errors.Is(err, fs.ErrNotExist) {
		return fmt.Errorf("filmcache: chunks de %s : %w", id, err)
	}
	for _, e := range entrees {
		if e.IsDir() {
			continue
		}
		if err := deplacerFichier(filepath.Join(ChunkDir(origine, id), e.Name()), filepath.Join(ChunkDir(cible, id), e.Name())); err != nil {
			return fmt.Errorf("filmcache: %s/%s : %w", id, e.Name(), err)
		}
	}
	if err := deplacerFichier(ManifestPath(origine, id), ManifestPath(cible, id)); err != nil {
		return fmt.Errorf("filmcache: manifeste de %s : %w", id, err)
	}
	_ = os.Remove(ChunkDir(origine, id)) // vide : ses fichiers sont partis un a un
	return nil
}

// deplacerFichier renomme `de` en `vers`, ou le copie puis le retire si les deux dossiers ne sont
// pas sur le meme volume. La copie est relue a la taille avant que la source ne soit retiree.
func deplacerFichier(de, vers string) error {
	if err := os.Rename(de, vers); err == nil {
		return nil
	}
	if err := copierFichier(de, vers); err != nil {
		return err
	}
	return os.Remove(de)
}

func copierFichier(de, vers string) (err error) {
	in, err := os.Open(de) //nolint:gosec // chemin compose par le cache
	if err != nil {
		return err
	}
	defer func() { _ = in.Close() }()
	part := vers + ".part"
	out, err := os.OpenFile(part, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0o644) //nolint:gosec // idem
	if err != nil {
		return err
	}
	defer func() {
		if err != nil {
			_ = os.Remove(part)
		}
	}()
	n, err := io.Copy(out, in)
	if err != nil {
		_ = out.Close()
		return err
	}
	if err = out.Sync(); err != nil {
		_ = out.Close()
		return err
	}
	if err = out.Close(); err != nil {
		return err
	}
	if info, serr := os.Stat(de); serr != nil || info.Size() != n {
		return fmt.Errorf("copie de %s : %d octets recopies, source illisible ou de taille differente", de, n)
	}
	return os.Rename(part, vers)
}
