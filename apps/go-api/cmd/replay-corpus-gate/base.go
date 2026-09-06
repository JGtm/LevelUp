package main

// base.go — LA REFERENCE PAR DEFAUT : UNE CUISSON A LA BASE, PAS LE PARC.
//
// # POURQUOI CE CHANGEMENT DE REFERENCE (decision superviseur, 2026-09-06)
//
// Un gate qui rend PERTE sur 7 temoins sur 7 au MEILLEUR etat connu (HEAD contre le parc, qui
// n'est jamais a jour) ne gate rien : personne ne peut lire un tableau tout rouge et savoir si
// SON changement a introduit une regression. La reference devient donc une CUISSON A LA BASE —
// le meme code que HEAD, un commit plus tot — et le signal redevient binaire : toute perte
// HEAD-contre-base est necessairement due au diff en cours de revue, jamais a l'age du parc.
//
// # LA RESOLUTION DE LA BASE
//
// Par defaut, `origin/feat/v75` SI le HEAD courant en differe (le cas normal : une branche de
// travail en cours de revue) — sinon `HEAD^` (le cas d'un gate lance directement sur la branche
// d'integration, pour verifier le dernier commit qui vient d'y atterrir). `--base` explicite
// remplace cette resolution.
//
// # LE WORKTREE DETACHE, ET SA SUPPRESSION SURE
//
// La base est materialisee par `git worktree add --detach` sous la racine de travail — jamais
// un checkout en place (qui deplacerait le HEAD du depot ou le gate tourne). Retire A LA FIN,
// meme en echec (`defer`). AVANT `git worktree remove`, le worktree est balaye pour une
// JONCTION : `git worktree remove` la SUIT (piege deja mesure sur ce depot,
// reference_worktree_remove_follows_junctions.md) et supprimerait alors recursivement des
// fichiers de l'AUTRE cote de la jonction. Ce gate ne pose jamais de jonction dans ce worktree
// (copie systematique, cf. staging.go) : la verification est une garde DEFENSIVE, pas un cas
// attendu — sa seule action si elle mord est de refuser de supprimer et de le dire fort.

import (
	"bytes"
	"fmt"
	"log/slog"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// resolveBaseRevision rend la revision de base a cuire, cf. l'en-tete du fichier.
func resolveBaseRevision(explicit, sourceRoot string) (string, error) {
	if explicit != "" {
		return explicit, nil
	}
	head, err := gitRevParse(sourceRoot, "HEAD")
	if err != nil {
		return "", fmt.Errorf("resolution de la base : HEAD illisible : %w", err)
	}
	origin, err := gitRevParse(sourceRoot, "origin/feat/v75")
	if err != nil {
		slog.Warn("replay-corpus-gate: origin/feat/v75 introuvable — repli sur HEAD^", "err", err)
		return "HEAD^", nil
	}
	if head != origin {
		return "origin/feat/v75", nil
	}
	return "HEAD^", nil
}

func gitRevParse(dir, rev string) (string, error) {
	cmd := exec.Command("git", "rev-parse", rev)
	cmd.Dir = dir
	out, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("git rev-parse %s : %w", rev, err)
	}
	return strings.TrimSpace(string(out)), nil
}

// worktreeBase est le worktree Git detache temporaire d'une revision de base.
type worktreeBase struct {
	Chemin   string
	GoAPIDir string
}

// creerWorktreeBase cree le worktree detache et rend sa fonction de nettoyage — a `defer` par
// l'appelant, MEME EN ECHEC (le worktree, une fois cree, doit toujours etre retire).
func creerWorktreeBase(sourceRoot, workDir, revision string) (worktreeBase, func(), error) {
	chemin := filepath.Join(workDir, "base-worktree")
	cmd := exec.Command("git", "worktree", "add", "--detach", chemin, revision) //nolint:gosec // revision resolue par resolveBaseRevision
	cmd.Dir = sourceRoot
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		return worktreeBase{}, nil, fmt.Errorf("git worktree add --detach %s %s : %w\n%s",
			chemin, revision, err, stderr.String())
	}
	wt := worktreeBase{Chemin: chemin, GoAPIDir: filepath.Join(chemin, "apps", "go-api")}
	cleanup := func() {
		if trouve, err := contientUneJonction(chemin); err != nil {
			slog.Warn("replay-corpus-gate: balayage de jonctions avant suppression du worktree base",
				"chemin", chemin, "err", err)
		} else if trouve {
			slog.Error("replay-corpus-gate: worktree base NON SUPPRIME — une jonction y a ete "+
				"detectee (git worktree remove la suivrait et supprimerait l'autre cote) ; "+
				"nettoyage manuel requis", "chemin", chemin)
			return
		}
		rmCmd := exec.Command("git", "worktree", "remove", "--force", chemin)
		rmCmd.Dir = sourceRoot
		if out, err := rmCmd.CombinedOutput(); err != nil {
			slog.Warn("replay-corpus-gate: suppression du worktree base", "chemin", chemin,
				"err", err, "sortie", string(out))
		}
	}
	return wt, cleanup, nil
}

// contientUneJonction balaie recursivement un repertoire a la recherche d'un reparse point
// (jonction NTFS ou symlink — Go les rend tous deux via ModeSymlink). Un balayage qui echoue
// EN COURS DE ROUTE ne doit pas faire passer un dossier suspect pour propre : l'erreur remonte,
// l'appelant refuse alors la suppression par prudence.
func contientUneJonction(racine string) (bool, error) {
	trouve := false
	err := filepath.WalkDir(racine, func(_ string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		info, err := d.Info()
		if err != nil {
			return err
		}
		if info.Mode()&os.ModeSymlink != 0 {
			trouve = true
			return filepath.SkipAll
		}
		return nil
	})
	if err != nil {
		return false, err
	}
	return trouve, nil
}

// compilerReplayBuild compile cmd/replay-build depuis `goAPIDir` vers `sortie`, avec un
// GOCACHE dedie (jamais celui du HEAD : les deux binaires peuvent differer, un cache partage
// les ferait courir apres le meme paquet compile sous deux revisions).
func compilerReplayBuild(goAPIDir, gocache, sortie string) error {
	if err := os.MkdirAll(gocache, 0o750); err != nil {
		return fmt.Errorf("GOCACHE dedie (%s) : %w", gocache, err)
	}
	cmd := exec.Command("go", "build", "-o", sortie, "./cmd/replay-build")
	cmd.Dir = goAPIDir
	cmd.Env = append(os.Environ(), "GOCACHE="+gocache, "CGO_ENABLED=0")
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("compilation de cmd/replay-build (%s) : %w\n%s", goAPIDir, err, stderr.String())
	}
	return nil
}
