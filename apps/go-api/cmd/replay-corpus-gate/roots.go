package main

// roots.go — LES TROIS RACINES DU GATE, ET POURQUOI ELLES SONT DISTINCTES.
//
//   - `sourceRoot` : le depot ou CE BINAIRE tourne (title.FindRepoRoot), donc le CODE et les
//     catalogues de reference VERSIONNES (config/titles/{slug}, data/titles/{slug}/reference)
//     au HEAD qu'on est en train de verifier. C'est la seule racine qui peut varier de branche
//     en branche.
//   - `parcRoot` : le depot qui porte le PARC de developpement (chunks de film non verses,
//     artefacts deja cuits) — en pratique le checkout PRINCIPAL, partage par tous les
//     worktrees. Resolu par defaut via `git rev-parse --git-common-dir` : le `.git` COMMUN a
//     tous les worktrees vit dans le principal, jamais dans un worktree secondaire — SAUF si
//     ce « principal » est LUI-MEME un worktree d'un ancetre .git renomme (topologie mesuree
//     le 2026-09-06 sur ce depot : `.git` commun de `LevelUp-go-migration` vit dans
//     `LevelUp`, un clone historique SANS le parc). L'auto-detection est donc VALIDEE (un
//     dossier `data/` doit exister) avant d'etre acceptee ; `--parc-root` reste la methode
//     SURE sur une topologie inhabituelle. Lecture SEULE : ce gate n'y ecrit jamais rien (ni
//     artefact, ni verrou par defaut — cf. lockRoot).
//   - `lockRoot` : ou vit le verrou de decodage partage (filmproc.AcquireSolo). Par defaut,
//     CacheRootDir() du PARC — le MEME chemin que tout autre outil de cuisson de ce depot
//     (cmd/replay-build, backfill-replay) y pose deja son verrou : deux cuissons lancees
//     depuis deux checkouts differents s'excluent donc mutuellement, comme demande.
//
// AUCUNE DE CES RACINES N'EST LA RACINE DE TRAVAIL DE LA CUISSON : celle-la (`workRoot`) est
// temporaire et jetable, batie par staging.go a partir des deux premieres — cf. son en-tete.

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"levelup/go-api/internal/domain/title"
)

// resolveParcRoot rend la racine du parc de developpement (chunks, artefacts de reference).
// Ordre : flag explicite, variable d'environnement, puis auto-detection par le `.git` commun —
// VALIDEE par la presence de la base partagee du titre (pas seulement d'un dossier `data/`,
// cf. l'en-tete du fichier : un ancetre .git renomme peut porter un `data/` PERIME sans etre
// le vrai parc).
func resolveParcRoot(flagValue, titleSlug string) (string, error) {
	if flagValue != "" {
		return filepath.Clean(flagValue), nil
	}
	if v := os.Getenv("REPLAY_CORPUS_GATE_PARC_ROOT"); v != "" {
		return filepath.Clean(v), nil
	}
	out, err := exec.Command("git", "rev-parse", "--path-format=absolute", "--git-common-dir").Output()
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
	sharedDB := title.NewPathResolver(racine).SharedDBPath(titleSlug)
	if _, err := os.Stat(sharedDB); err != nil {
		return "", fmt.Errorf(
			"auto-detection du parc (%s, via le .git commun) ne porte pas %s : %w — "+
				"topologie inhabituelle (le .git commun n'est pas le vrai parc de developpement ; "+
				"mesure le 2026-09-06 quand un depot dit « principal » est lui-meme un worktree "+
				"d'un ancetre renomme), passer --parc-root explicitement", racine, sharedDB, err)
	}
	return racine, nil
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
// `title.FindRepoRoot` cherche `db_profiles.json` (config joueurs, non versionnee) en
// remontant depuis le cwd : un worktree DEDIE fraichement cree n'en porte pas de copie locale
// (il vit uniquement dans le checkout principal, jamais duplique — donnee potentiellement
// personnelle). L'echec est alors EXPLICITE plutot qu'une racine fausse silencieuse : passer
// --source-root leve l'ambiguite.
func resolveSourceRoot(flagValue string) (string, error) {
	if flagValue != "" {
		return filepath.Clean(flagValue), nil
	}
	root, err := title.FindRepoRoot()
	if err != nil {
		return "", fmt.Errorf("%w — un worktree sans db_profiles.json local exige --source-root explicite", err)
	}
	return root, nil
}
