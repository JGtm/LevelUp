package main

// roots.go — LES TROIS RACINES DU GATE, ET POURQUOI ELLES SONT DISTINCTES.
//
//   - `sourceRoot` : le depot ou CE BINAIRE tourne, donc le CODE et les catalogues de
//     reference VERSIONNES (config/titles/{slug}, data/titles/{slug}/reference) au HEAD qu'on
//     est en train de verifier. C'est la seule racine qui peut varier de branche en branche.
//     Resolue par defaut via `git rev-parse --show-toplevel` — PAS `title.FindRepoRoot`
//     (cherche `db_profiles.json`, une donnee joueur non versionnee qu'un worktree DEDIE ne
//     porte jamais en copie locale : la racine du CODE n'a besoin d'aucune donnee joueur pour
//     etre trouvee, seulement de savoir dans quel depot Git ce binaire tourne — mesure
//     CORPUS-R1 C6, 2026-09-07).
//   - `parcRoot` : le depot qui porte le PARC de developpement (chunks de film non verses,
//     artefacts deja cuits). Ordre de resolution : (1) `sourceRoot` LUI-MEME, s'il porte deja
//     la base partagee du titre — le cas le plus courant, un gate lance depuis le depot de
//     developpement qui EST le parc ; (2) a defaut, le `.git` COMMUN a tous les worktrees (il
//     vit dans le depot principal, jamais dans un worktree secondaire) — SAUF si ce
//     « principal » est LUI-MEME un worktree d'un ancetre .git renomme (topologie mesuree le
//     2026-09-06 sur ce depot : `.git` commun de `LevelUp-go-migration` vit dans `LevelUp`, un
//     clone historique SANS le parc). Les DEUX candidats sont VALIDES par la presence de la
//     base partagee du titre (pas seulement d'un dossier `data/` : un ancetre .git renomme
//     peut porter un `data/` PERIME sans etre le vrai parc) ; `--parc-root` reste la methode
//     SURE sur une topologie inhabituelle (un worktree dedie sans base locale, par exemple).
//     Lecture SEULE sur les DONNEES : ce gate n'y ecrit jamais ni chunk ni artefact. UNE
//     exception intentionnelle : `lockRoot`, ci-dessous, y pose PAR DEFAUT un verrou de
//     decodage — corrige le 2026-09-07 (CORPUS-R1 C9) apres qu'un commentaire ait affirme
//     « ni verrou par defaut », contredit par `resolveLockRoot` trois lignes plus bas.
//   - `lockRoot` : ou vit le verrou de decodage partage (filmproc.AcquireSolo/Wait). Par
//     defaut, CacheRootDir() du PARC — le MEME chemin que tout autre outil de cuisson de ce
//     depot (cmd/replay-build, backfill-replay) y pose deja le sien : deux cuissons lancees
//     depuis deux checkouts differents s'excluent donc mutuellement, comme demande. Un fichier
//     `film_decode.lock` y est cree puis retire A CHAQUE cuisson (bake.go) — c'est le SEUL
//     ecrit que ce gate fait sous le parc, et il est voulu.
//
// AUCUNE DE CES RACINES N'EST LA RACINE DE TRAVAIL DE LA CUISSON : celle-la (`workRoot`) est
// temporaire et jetable, batie par staging.go a partir des deux premieres — cf. son en-tete.

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"levelup/go-api/internal/domain/title"
)

// resolveParcRoot rend la racine du parc de developpement (chunks, artefacts de reference).
// Ordre : flag explicite, variable d'environnement, `sourceRoot` s'il porte deja la base
// partagee du titre, puis auto-detection par le `.git` commun — TOUJOURS VALIDEE par la
// presence de la base partagee du titre (pas seulement d'un dossier `data/`, cf. l'en-tete du
// fichier). `sourceRoot` vide (appelant qui ne l'a pas encore resolu) saute simplement cette
// premiere tentative.
func resolveParcRoot(ctx context.Context, flagValue, sourceRoot, titleSlug string) (string, error) {
	if flagValue != "" {
		return filepath.Clean(flagValue), nil
	}
	if v := os.Getenv("REPLAY_CORPUS_GATE_PARC_ROOT"); v != "" {
		return filepath.Clean(v), nil
	}
	if sourceRoot != "" && racinePorteLaBase(sourceRoot, titleSlug) {
		return sourceRoot, nil
	}
	out, err := exec.CommandContext(ctx, "git", "rev-parse", "--path-format=absolute", "--git-common-dir").Output()
	if err != nil {
		return "", fmt.Errorf(
			"resolution automatique du parc impossible (git rev-parse --git-common-dir) : %w — "+
				"passer --parc-root explicitement", err)
	}
	commonDir := strings.TrimSpace(string(out))
	if commonDir == "" {
		return "", fmt.Errorf("git rev-parse --git-common-dir a rendu un chemin vide — passer --parc-root")
	}
	// Le `.git` commun vit A LA RACINE du depot principal (checkout ordinaire ou premier
	// worktree) : son parent EST cette racine, worktree ou pas.
	racine := filepath.Dir(commonDir)
	if !racinePorteLaBase(racine, titleSlug) {
		return "", fmt.Errorf(
			"ni la racine source (%s) ni l'auto-detection par le .git commun (%s) ne portent la "+
				"base partagee du titre %s — topologie inhabituelle (le .git commun n'est pas le "+
				"vrai parc de developpement ; mesure le 2026-09-06 quand un depot dit « principal » "+
				"est lui-meme un worktree d'un ancetre renomme), passer --parc-root explicitement",
			sourceRoot, racine, titleSlug)
	}
	return racine, nil
}

// racinePorteLaBase dit si `racine` porte la base partagee du titre — la VALIDATION commune
// aux deux candidats de resolveParcRoot (jamais la simple presence d'un dossier `data/`, cf.
// l'en-tete du fichier).
func racinePorteLaBase(racine, titleSlug string) bool {
	_, err := os.Stat(title.NewPathResolver(racine).SharedDBPath(titleSlug))
	return err == nil
}

// resolveLockRoot rend la racine ou poser le verrou de decodage partage. Ordre : flag,
// variable d'environnement, puis CacheRootDir() du parc — cf. l'en-tete du fichier.
func resolveLockRoot(flagValue, parcRoot string) string {
	if flagValue != "" {
		return filepath.Clean(flagValue)
	}
	if v := os.Getenv("REPLAY_CORPUS_GATE_LOCK_ROOT"); v != "" {
		return filepath.Clean(v)
	}
	return title.NewPathResolver(parcRoot).CacheRootDir()
}

// resolveSourceRoot rend le depot ou ce binaire tourne — le code et la config AU HEAD teste.
//
// PAS `title.FindRepoRoot` (cherche `db_profiles.json`, config joueurs non versionnee) : un
// worktree DEDIE fraichement cree n'en porte pas de copie locale (il vit uniquement dans le
// checkout principal, jamais duplique — donnee potentiellement personnelle), alors qu'il porte
// TOUJOURS son propre depot Git (c'est ce qui en fait un worktree). Trouver le CODE ne demande
// aucune donnee joueur — seulement `git rev-parse --show-toplevel` depuis le cwd, qui reussit
// dans N'IMPORTE QUEL worktree (corrige le 2026-09-07, CORPUS-R1 C6 : `make replay-corpus-gate`
// sortait en 2 depuis un worktree dedie faute de db_profiles.json local, alors que rien dans ce
// gate n'a besoin de ce fichier). L'echec (hors d'un depot Git) reste EXPLICITE : passer
// --source-root leve l'ambiguite.
func resolveSourceRoot(ctx context.Context, flagValue string) (string, error) {
	if flagValue != "" {
		return filepath.Clean(flagValue), nil
	}
	out, err := exec.CommandContext(ctx, "git", "rev-parse", "--path-format=absolute", "--show-toplevel").Output()
	if err != nil {
		return "", fmt.Errorf("racine du depot introuvable (git rev-parse --show-toplevel) : %w — "+
			"hors d'un depot Git, passer --source-root explicitement", err)
	}
	root := strings.TrimSpace(string(out))
	if root == "" {
		return "", fmt.Errorf("git rev-parse --show-toplevel a rendu un chemin vide — passer --source-root")
	}
	return filepath.Clean(root), nil
}
