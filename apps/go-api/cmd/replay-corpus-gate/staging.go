package main

// staging.go — LA RACINE DE TRAVAIL TEMPORAIRE : jamais le parc, jamais une jonction du
// systeme de fichiers.
//
// # POURQUOI UNE COPIE, PAS UNE JONCTION
//
// Le balayage du 2026-09-06 (.ai/V7.5/v2/BALAYAGE_PARC_2026-09-06.md) a proteg le parc par des
// jonctions NTFS EN LECTURE et un dossier `data/cache/replays` REEL pour l'ecriture — sur et
// mesure, mais VERIFIE apres coup (mtimes compares avant/apres) plutot que garanti par
// construction : une jonction ne bloque aucune ecriture, elle redirige simplement, et
// `mklink /J` n'est pas portable hors Windows. La copie, elle, rend l'isolation
// STRUCTURELLE : `workRoot` est un arbre de fichiers physiquement DISTINCT du parc — aucune
// configuration erronee ne peut faire ecrire la cuisson dans le parc, parce qu'aucun chemin
// n'est partage entre les deux arbres.
//
// # CE QUI EST COPIE, ET DEPUIS OU
//
//   - `config/titles/{slug}` et `data/titles/{slug}/reference` : depuis `sourceRoot` (le depot
//     ou ce binaire tourne) — c'est le CODE ET LES CATALOGUES DE REFERENCE au HEAD qu'on
//     verifie, pas ceux, potentiellement differents, du parc. Copies UNE FOIS par execution du
//     gate (~54 Mio, quelques secondes), partages par tous les temoins.
//   - `data/cache/film_chunks/{short}` et `film_manifests/{short}.json` : depuis `parcRoot` —
//     seule source qui les porte (non verses, jamais dans un checkout frais). Copies PAR
//     TEMOIN, jamais le parc entier.
//
// L'ARTEFACT FRAIS s'ecrit ensuite sous `workRoot/data/cache/replays/{slug}/{short}.json` — un
// chemin qui n'a JAMAIS existe dans le parc reel.

import (
	"fmt"
	"io"
	"os"
	"path/filepath"

	"levelup/go-api/internal/domain/title"
	"levelup/go-api/internal/games/halo_infinite/film/filmcache"
)

// stageReferenceOnce copie UNE FOIS, vers `workRoot`, les catalogues VERSIONNES du titre lus
// depuis `sourceRoot` : config/titles/{slug} et data/titles/{slug}/reference.
func stageReferenceOnce(sourceRoot, workRoot, titleSlug string) error {
	src := title.NewPathResolver(sourceRoot)
	dst := title.NewPathResolver(workRoot)

	if err := copierArbre(
		filepath.Join(sourceRoot, "config", "titles", titleSlug),
		filepath.Join(workRoot, "config", "titles", titleSlug),
	); err != nil {
		return fmt.Errorf("config du titre %s : %w", titleSlug, err)
	}
	if err := copierArbre(
		filepath.Join(src.TitleDataDir(titleSlug), "reference"),
		filepath.Join(dst.TitleDataDir(titleSlug), "reference"),
	); err != nil {
		return fmt.Errorf("catalogues de reference du titre %s : %w", titleSlug, err)
	}
	return nil
}

// stageFilm copie le manifeste et les chunks d'UN film depuis le parc reel vers la racine de
// travail. Rend le dossier de chunks a passer a `replaybuild.Builder.BuildMatch`. Un film
// absent du parc (aucun chunk) est signale par une erreur nommee — l'appelant la traduit en
// avertissement `slog` et saute le temoin, sans faire echouer les autres.
func stageFilm(parcRoot, workRoot, short string) (filmDir string, err error) {
	srcCache := title.NewPathResolver(parcRoot).CacheRootDir()
	dstCache := title.NewPathResolver(workRoot).CacheRootDir()

	srcManifest := filmcache.ManifestPath(srcCache, short)
	if _, statErr := os.Stat(srcManifest); statErr != nil {
		return "", fmt.Errorf("manifeste du film %s absent du parc (%s) : %w", short, srcManifest, statErr)
	}
	dstManifest := filmcache.ManifestPath(dstCache, short)
	if err := copierFichier(srcManifest, dstManifest); err != nil {
		return "", fmt.Errorf("copie du manifeste %s : %w", short, err)
	}

	srcChunks := filmcache.ChunkDir(srcCache, short)
	dstChunks := filmcache.ChunkDir(dstCache, short)
	n, err := copierArbreCompte(srcChunks, dstChunks)
	if err != nil {
		return "", fmt.Errorf("copie des chunks %s : %w", short, err)
	}
	if n == 0 {
		return "", fmt.Errorf("film %s absent du parc (aucun chunk sous %s)", short, srcChunks)
	}
	return dstChunks, nil
}

// copierArbre copie recursivement un repertoire. Un repertoire source absent n'est PAS une
// erreur (certains titres n'ont pas de reference — cf. capability film.replay_artifact) : le
// gate refuse alors la cuisson faute de catalogue, avec un message qui le dit.
func copierArbre(src, dst string) error {
	_, err := copierArbreCompte(src, dst)
	return err
}

// copierArbreCompte copie recursivement et rend le nombre de FICHIERS copies (0 si `src`
// n'existe pas — un repertoire absent n'est pas une erreur, cf. copierArbre).
func copierArbreCompte(src, dst string) (int, error) {
	info, err := os.Stat(src)
	if err != nil {
		if os.IsNotExist(err) {
			return 0, nil
		}
		return 0, err
	}
	if !info.IsDir() {
		return 0, fmt.Errorf("%s n'est pas un repertoire", src)
	}
	entries, err := os.ReadDir(src)
	if err != nil {
		return 0, err
	}
	if err := os.MkdirAll(dst, 0o750); err != nil {
		return 0, err
	}
	total := 0
	for _, e := range entries {
		s, d := filepath.Join(src, e.Name()), filepath.Join(dst, e.Name())
		if e.IsDir() {
			n, err := copierArbreCompte(s, d)
			if err != nil {
				return total, err
			}
			total += n
			continue
		}
		if err := copierFichier(s, d); err != nil {
			return total, err
		}
		total++
	}
	return total, nil
}

// copierFichier copie un fichier UNIQUE, en creant son repertoire parent au besoin.
func copierFichier(src, dst string) error {
	if err := os.MkdirAll(filepath.Dir(dst), 0o750); err != nil {
		return err
	}
	in, err := os.Open(src) //nolint:gosec // chemins internes au gate, source dans le parc local
	if err != nil {
		return err
	}
	defer func() { _ = in.Close() }()
	out, err := os.Create(dst) //nolint:gosec // chemins internes au gate, racine de travail jetable
	if err != nil {
		return err
	}
	if _, err := io.Copy(out, in); err != nil {
		_ = out.Close()
		return err
	}
	return out.Close()
}
