//go:build windows

package ops

import (
	"context"
	"log/slog"
	"time"

	"levelup/go-api/internal/platform/childproc"
)

// revealLaunchTimeout borne la vie du lanceur : explorer.exe confie la fenêtre à l'explorateur déjà
// en cours et se termine aussitôt.
const revealLaunchTimeout = 15 * time.Second

// systemRevealOpen lance explorer.exe sur target : « /select,"chemin" » pour sélectionner un fichier,
// le chemin seul pour ouvrir un dossier.
//
// La ligne de commande est posée telle quelle (SysProcAttr.CmdLine) : explorer.exe n'analyse pas ses
// arguments comme le fait os/exec, qui échapperait les guillemets de « /select, ». HideWindow est
// remis à zéro : le lanceur de childproc est fait pour des enfants console, et l'explorateur est une
// fenêtre que l'utilisateur doit voir. Le contexte du lanceur n'est pas celui de la requête, qui
// l'arrêterait avant l'ouverture. Le code de sortie d'explorer.exe ne dit rien du succès (il rend
// souvent 1) : seul un échec du lancement est une erreur.
func systemRevealOpen(ctx context.Context, target string, selectIt bool) error {
	launchCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), revealLaunchTimeout)
	cmd := childproc.Command(launchCtx, "explorer.exe")
	line := `explorer.exe "` + target + `"`
	if selectIt {
		line = `explorer.exe /select,"` + target + `"`
	}
	cmd.SysProcAttr.CmdLine = line
	cmd.SysProcAttr.HideWindow = false
	if err := cmd.Start(); err != nil {
		cancel()
		return err
	}
	go func() {
		defer cancel()
		if err := cmd.Wait(); err != nil {
			slog.DebugContext(ctx, "media_reveal: explorer.exe terminé en erreur (code sans valeur)", "err", err)
		}
	}()
	return nil
}
