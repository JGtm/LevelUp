package filmarchive

// deplacement.go — LE DÉPLACEMENT DE L'ARCHIVE VERS UN AUTRE DOSSIER (lot L14, `levelup film
// archive-move`).
//
// UN FILM EST ENTIER D'UN CÔTÉ OU DE L'AUTRE, JAMAIS DES DEUX. Un film s'écrit d'abord dans un
// dossier temporaire de la cible (`<id>.part/`) ; ce dossier est renommé en un seul geste, puis le
// manifeste (le marqueur de commit) ; l'origine n'est retirée qu'ensuite. Une erreur avant le
// renommage (disque plein, interruption) retire le temporaire et laisse l'origine intacte. Sur un
// même volume, le renommage direct des deux éléments remplace la copie ; un arrêt entre les deux
// renommages laisse les chunks à la cible et le manifeste à l'origine, et la reprise achève ce
// geste (état reconnu par [deplacerFilm]).
//
// UNE REPRISE SE RECONNAÎT À UN MARQUEUR posé dans la cible au début du déplacement et retiré à
// sa fin. Sans lui, `reprise` ne saute rien : la cible est contrôlée comme pour un premier
// déplacement, de sorte qu'un `--resume` posé à la légère ne mélange pas deux archives.

import (
	"context"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"strings"

	"github.com/JGtm/LevelUp/libs/film/filmcache"
)

// marqueurDeplacement : le fichier, dans la cible, qui dit « un déplacement depuis <origine> est en
// cours ».
const marqueurDeplacement = ".deplacement-archive-en-cours"

// suffixePartiel : le suffixe des éléments temporaires d'un film en cours de copie.
const suffixePartiel = ".part"

// Points d'injection des tests (échec d'écriture au milieu d'un film, volumes distincts). Jamais
// réassignés en production.
var (
	renommerDossier = os.Rename
	copieDeFichier  = copierFichier
)

// ErrCibleNonVide : la destination d'un déplacement contient déjà quelque chose. Déplacer sur un
// dossier non vide mélangerait deux archives ; l'appelant le refuse et le dit à l'utilisateur.
var ErrCibleNonVide = errors.New("filmcache: le dossier cible n'est pas vide")

// ValiderCible contrôle la destination d'un déplacement de l'archive : un chemin ABSOLU. Un dossier
// explicite doit être absent ou vide. Le dossier par défaut (`parDefaut`, le cache applicatif) ne
// l'est jamais — il porte les faits, les verrous, les images : il suffit qu'il ne contienne aucun
// film. [ErrCibleNonVide] sinon.
func ValiderCible(cible string, parDefaut bool) error {
	if strings.TrimSpace(cible) == "" || !filepath.IsAbs(cible) {
		return fmt.Errorf("filmcache: le dossier cible doit etre un chemin absolu (%q)", cible)
	}
	if parDefaut {
		return aucunFilm(cible)
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

// aucunFilm dit que `racine` ne porte aucun film : `film_chunks/` et `film_manifests/` absents ou
// vides (les temporaires d'une copie interrompue ne comptent pas).
func aucunFilm(racine string) error {
	for _, dossier := range []string{filmcache.ChunksRoot(racine), filmcache.ManifestsRoot(racine)} {
		entrees, err := os.ReadDir(dossier)
		if errors.Is(err, fs.ErrNotExist) {
			continue
		}
		if err != nil {
			return fmt.Errorf("filmcache: dossier cible illisible (%s) : %w", dossier, err)
		}
		for _, e := range entrees {
			if !strings.HasSuffix(e.Name(), suffixePartiel) {
				return fmt.Errorf("%w : %s contient des films", ErrCibleNonVide, racine)
			}
		}
	}
	return nil
}

// OptionsDeplacement règle [Deplacer].
type OptionsDeplacement struct {
	// Reprise : poursuivre un déplacement interrompu. Sans marqueur de déplacement depuis la même
	// origine dans la cible, elle est sans effet et la cible est contrôlée normalement.
	Reprise bool
	// CibleParDefaut : la cible est l'emplacement par défaut de l'archive (cf. [ValiderCible]).
	CibleParDefaut bool
}

// Deplacer range les films de l'archive `origine` dans `cible`, un film à la fois (cf. l'en-tête).
// Rend le nombre de films déplacés. Le contexte est contrôlé avant chaque film et chaque chunk.
func Deplacer(ctx context.Context, origine, cible string, opt OptionsDeplacement) (int, error) {
	reprise := opt.Reprise && deplacementEnCours(cible, origine)
	if !reprise {
		if err := ValiderCible(cible, opt.CibleParDefaut); err != nil {
			return 0, err
		}
	}
	films, err := filmcache.ListShortIDs(origine)
	if err != nil {
		return 0, err
	}
	if err := EnsureDirs(cible); err != nil {
		return 0, err
	}
	if err := os.WriteFile(filepath.Join(cible, marqueurDeplacement), []byte(filepath.Clean(origine)), 0o644); err != nil {
		return 0, fmt.Errorf("filmcache: marqueur de deplacement : %w", err)
	}
	deplaces := 0
	for _, id := range films {
		if err := ctx.Err(); err != nil {
			return deplaces, err
		}
		if err := deplacerFilm(ctx, origine, cible, id); err != nil {
			return deplaces, err
		}
		deplaces++
	}
	if err := os.Remove(filepath.Join(cible, marqueurDeplacement)); err != nil && !errors.Is(err, fs.ErrNotExist) {
		return deplaces, fmt.Errorf("filmcache: marqueur de deplacement : %w", err)
	}
	return deplaces, nil
}

// deplacementEnCours dit si `cible` porte le marqueur d'un déplacement depuis `origine`.
func deplacementEnCours(cible, origine string) bool {
	brut, err := os.ReadFile(filepath.Join(cible, marqueurDeplacement)) //nolint:gosec // chemin compose
	return err == nil && filepath.Clean(string(brut)) == filepath.Clean(origine)
}

func existe(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}

// deplacerFilm range UN film, ou reconnaît l'état qu'un arrêt a laissé et l'achève.
func deplacerFilm(ctx context.Context, origine, cible, id string) error {
	manifesteCible := filmcache.ManifestPath(cible, id)
	chunksOrigine, chunksCible := filmcache.ChunkDir(origine, id), filmcache.ChunkDir(cible, id)
	switch {
	case existe(manifesteCible):
		// Film complet à la cible (arrêt après son commit, avant le retrait de l'origine).
		return retirerOrigine(origine, id)
	case !existe(chunksOrigine) && existe(chunksCible) && existe(filmcache.ManifestPath(origine, id)):
		// Arrêt entre les deux renommages d'un déplacement sur le même volume.
		return deplacerFichier(filmcache.ManifestPath(origine, id), manifesteCible)
	}
	if err := renommerDossier(chunksOrigine, chunksCible); err == nil {
		if err := deplacerFichier(filmcache.ManifestPath(origine, id), manifesteCible); err != nil {
			_ = renommerDossier(chunksCible, chunksOrigine) // le film reste entier à l'origine
			return fmt.Errorf("filmcache: manifeste de %s : %w", id, err)
		}
		return nil
	}
	return copierFilm(ctx, origine, cible, id)
}

// copierFilm copie un film vers un temporaire de la cible, le valide d'un seul renommage, puis
// retire l'origine. Toute erreur avant le commit retire le temporaire.
func copierFilm(ctx context.Context, origine, cible, id string) (err error) {
	chunksCible := filmcache.ChunkDir(cible, id)
	partiel := chunksCible + suffixePartiel
	manifestePartiel := filmcache.ManifestPath(cible, id) + suffixePartiel
	defer func() {
		if err != nil {
			_ = os.RemoveAll(partiel)
			_ = os.Remove(manifestePartiel)
		}
	}()
	_ = os.RemoveAll(partiel)
	_ = os.Remove(manifestePartiel)
	if err = os.MkdirAll(partiel, 0o755); err != nil {
		return fmt.Errorf("filmcache: dossier temporaire de %s : %w", id, err)
	}
	entrees, err := os.ReadDir(filmcache.ChunkDir(origine, id))
	if err != nil && !errors.Is(err, fs.ErrNotExist) {
		return fmt.Errorf("filmcache: chunks de %s : %w", id, err)
	}
	for _, e := range entrees {
		if e.IsDir() {
			continue
		}
		if err = ctx.Err(); err != nil {
			return err
		}
		if err = copieDeFichier(filepath.Join(filmcache.ChunkDir(origine, id), e.Name()), filepath.Join(partiel, e.Name())); err != nil {
			return fmt.Errorf("filmcache: %s/%s : %w", id, e.Name(), err)
		}
	}
	if err = copieDeFichier(filmcache.ManifestPath(origine, id), manifestePartiel); err != nil {
		return fmt.Errorf("filmcache: manifeste de %s : %w", id, err)
	}
	// Un dossier de chunks sans manifeste à la cible est un reste invisible de notre propre
	// passage : on le remplace.
	if err = os.RemoveAll(chunksCible); err != nil {
		return fmt.Errorf("filmcache: reste de %s à la cible : %w", id, err)
	}
	if err = os.Rename(partiel, chunksCible); err != nil {
		return fmt.Errorf("filmcache: validation des chunks de %s : %w", id, err)
	}
	if err = os.Rename(manifestePartiel, filmcache.ManifestPath(cible, id)); err != nil {
		_ = os.RemoveAll(chunksCible)
		return fmt.Errorf("filmcache: validation du manifeste de %s : %w", id, err)
	}
	return retirerOrigine(origine, id)
}

// retirerOrigine retire un film de l'origine : manifeste d'abord (le film cesse d'y être lisible
// d'un coup), puis ses chunks. Le film est alors déjà entier à la cible.
func retirerOrigine(origine, id string) error {
	return Supprimer(origine, id)
}

// deplacerFichier renomme `de` en `vers`, ou le copie puis le retire si les deux dossiers ne sont
// pas sur le même volume.
func deplacerFichier(de, vers string) error {
	if err := os.Rename(de, vers); err == nil {
		return nil
	}
	if err := copieDeFichier(de, vers); err != nil {
		return err
	}
	return os.Remove(de)
}

// copierFichier copie `de` vers `vers` (via un `.part`, fsync, relecture de la taille).
func copierFichier(de, vers string) (err error) {
	in, err := os.Open(de) //nolint:gosec // chemin compose par le cache
	if err != nil {
		return err
	}
	defer func() { _ = in.Close() }()
	part := vers + suffixePartiel
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
