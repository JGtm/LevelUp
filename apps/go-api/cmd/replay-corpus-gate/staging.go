package main

// staging.go — LA RACINE DE TRAVAIL TEMPORAIRE : jamais le parc, jamais une jonction du
// systeme de fichiers.
//
// # POURQUOI UNE COPIE, PAS UNE JONCTION
//
// Le balayage du 2026-09-06 a proteg le parc par des
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
//     gate (~54 Mio, quelques secondes), partages par tous les temoins. SEULS LES FICHIERS QUE GIT
//     VOIT sont copies (suivis, ou non suivis et non ignores) : les deux cuissons lisent ainsi le
//     MEME jeu de catalogues, quel que soit le poste. Un fichier de reference ignore par git (les
//     overlays que le runtime ecrit sous `reference/generated/`) n'est fourni a AUCUNE des deux
//     racines : la racine de la base, extraite d'une revision, ne l'a jamais, et le donner au seul
//     HEAD imputerait au diff de revision l'ecart qu'il cree. Meme principe que le catalogue
//     hermetique de `cmd/replay-equiv`.
//   - `data/cache/film_chunks/{short}` et `film_manifests/{short}.json` : depuis `parcRoot` —
//     seule source qui les porte (non verses, jamais dans un checkout frais). Copies PAR
//     TEMOIN, jamais le parc entier.
//
// L'ARTEFACT FRAIS s'ecrit ensuite sous `workRoot/data/cache/replays/{slug}/{short}.json` — un
// chemin qui n'a JAMAIS existe dans le parc reel.

import (
	"context"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"log/slog"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"levelup/go-api/internal/domain/title"

	"github.com/JGtm/halo-infinite-film-decoder/filmcache"
)

// avertirSiCatalogueModifie logue un AVERTISSEMENT explicite si config/titles/{slug} ou
// data/titles/{slug}/reference portent des modifications NON COMMISES dans `sourceRoot` —
// CORPUS-R1 C11 (2026-09-07). Le cote HEAD de ce gate copie DELIBEREMENT l'ARBRE DE TRAVAIL
// (stageReferenceOnce, ci-dessous), jamais un commit strict comme le cote base
// (creerWorktreeBase, base.go) : c'est ce qui permet de tester un changement de catalogue AVANT
// de le committer — exactement le moment ou ce gate sert le plus. Un fichier local NON COMMIS
// serait sinon impute A TORT au diff de revision par un lecteur qui n'aurait aucun moyen de le
// savoir : cet avertissement rend le cas VISIBLE, sans empecher l'usage qui le motive. Un
// echec de la verification elle-meme (pas un depot git, `git` absent) est lui aussi un simple
// avertissement — jamais une raison de faire echouer la cuisson.
func avertirSiCatalogueModifie(ctx context.Context, sourceRoot, titleSlug string) {
	relRef, err := filepath.Rel(sourceRoot,
		filepath.Join(title.NewPathResolver(sourceRoot).TitleDataDir(titleSlug), "reference"))
	if err != nil {
		slog.WarnContext(ctx, "replay-corpus-gate: verification des modifications locales des catalogues (chemin)",
			"err", err)
		return
	}
	relConfig := filepath.Join("config", "titles", titleSlug)
	cmd := exec.CommandContext(ctx, "git", "status", "--porcelain", "--", relConfig, relRef) //nolint:gosec // chemins internes derives du titre
	cmd.Dir = sourceRoot
	out, err := cmd.Output()
	if err != nil {
		slog.WarnContext(ctx, "replay-corpus-gate: verification des modifications locales des catalogues",
			"err", err)
		return
	}
	if modifs := strings.TrimSpace(string(out)); modifs != "" {
		slog.WarnContext(ctx, "replay-corpus-gate: catalogues de reference du HEAD modifies localement "+
			"(non commis) — la cuisson HEAD les inclut ; un ecart contre la base peut donc ne "+
			"PAS venir uniquement du diff de revision",
			"titre", titleSlug, "modifications", modifs)
	}
}

// stageReferenceOnce copie UNE FOIS, vers `workRoot`, les catalogues du titre lus depuis `racine` :
// config/titles/{slug} et data/titles/{slug}/reference, LIMITES AUX FICHIERS QUE GIT VOIT (cf.
// l'en-tete). `racine` doit etre un depot Git : sans lui, le gate ne sait pas ce qui est ignore et
// refuse plutot que de deviner. Les fichiers ignores laisses de cote sont journalises.
func stageReferenceOnce(ctx context.Context, racine, workRoot, titleSlug string) error {
	relRef, err := filepath.Rel(racine, filepath.Join(title.NewPathResolver(racine).TitleDataDir(titleSlug), "reference"))
	if err != nil {
		return fmt.Errorf("catalogues de reference du titre %s : %w", titleSlug, err)
	}
	for _, c := range []struct{ rel, libelle string }{
		{filepath.Join("config", "titles", titleSlug), "config du titre"},
		{relRef, "catalogues de reference du titre"},
	} {
		if err := copierFichiersVisiblesParGit(ctx, racine, workRoot, c.rel); err != nil {
			return fmt.Errorf("%s %s : %w", c.libelle, titleSlug, err)
		}
	}
	return nil
}

// copierFichiersVisiblesParGit copie, de `racine` vers `dstRoot` (meme chemin relatif `rel`), les
// fichiers suivis et les fichiers non suivis NON ignores : l'arbre de travail tel que Git le voit,
// modifications non commises comprises, jamais ce qu'il ignore. Un fichier suivi mais supprime de
// l'arbre de travail n'est pas copie (suppression non commise). Un dossier absent ne copie rien.
func copierFichiersVisiblesParGit(ctx context.Context, racine, dstRoot, rel string) error {
	visibles, err := gitListerFichiers(ctx, racine, rel, "--cached", "--others", "--exclude-standard")
	if err != nil {
		return err
	}
	ignores, err := gitListerFichiers(ctx, racine, rel, "--others", "--ignored", "--exclude-standard")
	if err != nil {
		return err
	}
	if len(ignores) > 0 {
		slog.InfoContext(ctx, "replay-corpus-gate: fichiers de reference ignores par git, fournis a aucune "+
			"des deux racines de cuisson", "racine", racine, "dossier", rel, "nombre", len(ignores), "premier", ignores[0])
	}
	for _, f := range visibles {
		src := filepath.Join(racine, f)
		if _, err := os.Stat(src); errors.Is(err, fs.ErrNotExist) {
			continue
		} else if err != nil {
			return err
		}
		if err := copierFichier(src, filepath.Join(dstRoot, f)); err != nil {
			return fmt.Errorf("copie de %s : %w", f, err)
		}
	}
	return nil
}

// gitListerFichiers rend les chemins (relatifs a `racine`, separateur natif) que `git ls-files`
// liste sous `rel` pour les options donnees. Un echec de git (pas un depot) est une erreur.
func gitListerFichiers(ctx context.Context, racine, rel string, options ...string) ([]string, error) {
	args := append([]string{"ls-files", "-z"}, options...)
	args = append(args, "--", filepath.ToSlash(rel))
	cmd := exec.CommandContext(ctx, "git", args...) //nolint:gosec // options constantes, chemin interne derive du titre
	cmd.Dir = racine
	out, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("git ls-files sous %s (%s) : %w — la racine doit etre un depot Git", rel, racine, err)
	}
	var fichiers []string
	for _, f := range strings.Split(string(out), "\x00") {
		if f != "" {
			fichiers = append(fichiers, filepath.FromSlash(f))
		}
	}
	return fichiers, nil
}

// stageFilm copie le manifeste et les chunks d'UN film depuis le parc reel vers la racine de
// travail. Ne rend PAS le dossier de chunks (CORPUS-R1 C7, unparam) : depuis que la cuisson
// passe par un sous-processus (bake.go), ce chemin n'est JAMAIS transmis explicitement — le
// binaire replay-build le deduit lui-meme depuis LEVELUP_REPO_ROOT (filmcache.ChunkDir), cf.
// l'en-tete de cuireUneCarte. Un film absent du parc (aucun chunk) est signale par une erreur
// nommee — l'appelant la traduit en avertissement `slog` et saute le temoin, sans faire
// echouer les autres.
func stageFilm(parcRoot, workRoot, short string) error {
	srcCache := title.NewPathResolver(parcRoot).FilmArchiveDir()
	dstCache := title.NewPathResolver(workRoot).FilmArchiveDir()

	srcManifest := filmcache.ManifestPath(srcCache, short)
	if _, statErr := os.Stat(srcManifest); statErr != nil {
		return fmt.Errorf("manifeste du film %s absent du parc (%s) : %w", short, srcManifest, statErr)
	}
	dstManifest := filmcache.ManifestPath(dstCache, short)
	if err := copierFichier(srcManifest, dstManifest); err != nil {
		return fmt.Errorf("copie du manifeste %s : %w", short, err)
	}

	srcChunks := filmcache.ChunkDir(srcCache, short)
	dstChunks := filmcache.ChunkDir(dstCache, short)
	n, err := copierArbreCompte(srcChunks, dstChunks)
	if err != nil {
		return fmt.Errorf("copie des chunks %s : %w", short, err)
	}
	if n == 0 {
		return fmt.Errorf("film %s absent du parc (aucun chunk sous %s)", short, srcChunks)
	}
	return nil
}

// copierArbreCompte copie recursivement et rend le nombre de FICHIERS copies (0 si `src`
// n'existe pas : certains titres n'ont pas de reference — cf. capability film.replay_artifact — et le gate
// refuse alors la cuisson faute de catalogue, avec un message qui le dit).
func copierArbreCompte(src, dst string) (int, error) {
	info, err := os.Stat(src)
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) {
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
