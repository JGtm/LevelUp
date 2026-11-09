// uninstall_cleanup_mode.go — le mode -mode uninstall-cleanup : retire ce que l'application a posé HORS de son
// dossier, puis sort (ADR 0045 D-28).
//
// Le désinstalleur NSIS (crochet NSIS_HOOK_PREUNINSTALL, apps/desktop/src-tauri/windows/hooks.nsh) n'a sous la main
// que le serveur livré : il l'appelle ici, avant d'effacer les fichiers du programme. Le mode ne charge aucune
// configuration serveur, n'ouvre AUCUNE base, ne prend aucun verrou et ne lie aucun port. Il retire :
//   - l'entrée « démarrer avec Windows » de l'utilisateur (valeur `LevelUp` de HKCU Run et son approbation), si elle
//     vise le dossier de CE programme (internal/platform/startupentry) ;
//   - la tâche planifiée `LevelUp\Refresh-<empreinte>` de la racine de données de l'environnement
//     (internal/platform/schedtask), le nom étant propre à la racine, si son action lance le serveur de CE dossier :
//     deux installations d'un même utilisateur partagent la racine par défaut, donc le nom de la tâche, et la tâche
//     de l'autre (`tache=foreign`) n'est pas retirée ;
//   - le paquet de ressources installé sous le dossier du programme (internal/resourcepack.Remove : les fichiers que son
//     manifeste liste, rien d'autre) : l'installeur « moteur seul » d'une mise à jour ne les liste pas, le désinstalleur
//     NSIS ne les connaît donc plus après une mise à jour courante.
//
// La racine de données n'est JAMAIS touchée (ADR 0045, A22). Le bilan va sur stderr (stdout est réservé aux
// événements de la coquille). Une entrée absente ou étrangère est un succès (sortie 0) ; un échec de lecture ou de
// suppression donne la sortie 1, que le crochet NSIS ignore : la désinstallation ne dépend jamais de ce nettoyage.
//
// Si la variable LEVELUP_UNINSTALL_WITNESS porte un chemin, le bilan (entrées, tâche, racine, dossier, `ok` et le code
// de sortie `exit`) y est aussi écrit : le test de bout en bout de l'installeur (apps/desktop/scripts/e2e-install.ps1)
// prouve ainsi que le désinstalleur a bien lancé le mode, et ce qu'il a rendu.
package main

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"

	"levelup/go-api/internal/config"
	"levelup/go-api/internal/domain/title"
	"levelup/go-api/internal/platform/schedtask"
	"levelup/go-api/internal/platform/startupentry"
	"levelup/go-api/internal/resourcepack"
)

// uninstallCleanupFailure : code de sortie quand une lecture ou une suppression a échoué.
const uninstallCleanupFailure = 1

// envUninstallWitness : chemin du fichier témoin du bilan (essais de l'installeur ; absent en usage normal).
const envUninstallWitness = "LEVELUP_UNINSTALL_WITNESS"

// uninstallCleanupDeps : ce que le nettoyage lit et supprime (le registre et schtasks, simulés dans les tests).
type uninstallCleanupDeps struct {
	// Startup : le registre de démarrage de l'utilisateur.
	Startup startupentry.Store
	// Tasks : la tâche planifiée de la racine de données ; nil quand la racine est inconnue.
	Tasks interface {
		TaskName() string
		RemoveIfTargets(ctx context.Context, installDir string) (schedtask.RemoveOutcome, error)
	}
	// InstallDir : le dossier du programme désinstallé (celui du serveur en cours).
	InstallDir string
	// DataRoot : la racine de données dont la tâche est retirée ; vide = inconnue (LEVELUP_DATA_DIR absente).
	DataRoot string
	// Out : le bilan (stderr).
	Out io.Writer
	// Witness : le fichier témoin ; vide = aucun.
	Witness string
}

// say écrit une ligne du bilan. Une sortie d'erreur fermée n'a aucun autre canal où se plaindre : l'échec d'écriture est
// ignoré, le code de sortie et le fichier témoin portent le résultat.
func (d uninstallCleanupDeps) say(format string, args ...any) {
	_, _ = fmt.Fprintf(d.Out, format+"\n", args...)
}

// runUninstallCleanupMode rend le code de sortie du processus.
func runUninstallCleanupMode(ctx context.Context) int {
	exe, err := os.Executable()
	if err != nil {
		fmt.Fprintf(os.Stderr, "uninstall-cleanup : exécutable introuvable : %v\n", err)
		return uninstallCleanupFailure
	}
	deps := newUninstallCleanupDeps(exe, config.ResolveRoots(), startupentry.NewRegistryStore(), schedtask.ExecRunner{})
	deps.Out, deps.Witness = os.Stderr, os.Getenv(envUninstallWitness)
	return uninstallCleanup(ctx, deps)
}

// newUninstallCleanupDeps câble le nettoyage : le dossier du programme est celui de l'exécutable en cours, et la
// tâche (si la racine de données est connue) est celle de roots.Data, que le serveur de cet exécutable aurait posée.
func newUninstallCleanupDeps(exe string, roots title.Roots, startup startupentry.Store, runner schedtask.Runner) uninstallCleanupDeps {
	deps := uninstallCleanupDeps{Startup: startup, InstallDir: filepath.Dir(exe)}
	if roots.DataSet {
		deps.DataRoot = roots.Data
		deps.Tasks = &schedtask.Manager{
			Runner: runner,
			Spec:   schedtask.Spec{ExePath: exe, DataRoot: roots.Data, ResourcesRoot: roots.Resources, LogsRoot: roots.Logs},
		}
	}
	return deps
}

// uninstallCleanup retire l'entrée de démarrage puis la tâche planifiée ; les deux sont tentées même si la
// première échoue.
func uninstallCleanup(ctx context.Context, d uninstallCleanupDeps) int {
	var failures []error
	startup := "erreur"
	if outcome, err := startupentry.RemoveForInstall(d.Startup, d.InstallDir); err != nil {
		failures = append(failures, err)
	} else {
		startup = string(outcome)
	}
	task := "racine-inconnue"
	if d.Tasks != nil {
		task = "erreur"
		if outcome, err := d.Tasks.RemoveIfTargets(ctx, d.InstallDir); err != nil {
			failures = append(failures, err)
		} else {
			task = string(outcome)
		}
	}
	pack := "erreur"
	if n, err := resourcepack.Remove(d.InstallDir); err != nil {
		failures = append(failures, err)
	} else {
		pack = fmt.Sprintf("%d-fichiers", n)
	}
	taskName := ""
	if d.Tasks != nil {
		taskName = d.Tasks.TaskName()
	}
	code := 0
	if len(failures) > 0 {
		code = uninstallCleanupFailure
	}
	report := fmt.Sprintf("uninstall-cleanup: demarrage=%s tache=%s ressources=%s nom_tache=%q data_root=%q install_dir=%q ok=%t exit=%d",
		startup, task, pack, taskName, d.DataRoot, d.InstallDir, code == 0, code)
	d.say("%s", report)
	if err := errors.Join(failures...); err != nil {
		d.say("uninstall-cleanup : %v", err)
	}
	if d.Witness != "" {
		if err := os.WriteFile(d.Witness, []byte(report+"\n"), 0o600); err != nil {
			d.say("uninstall-cleanup : fichier témoin %s non écrit : %v", d.Witness, err)
		}
	}
	return code
}
