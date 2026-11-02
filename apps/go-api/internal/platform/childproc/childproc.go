// Package childproc : le lanceur unique des processus enfants console du serveur.
//
// POURQUOI. Windows donne une console NEUVE et visible a tout processus console lance par un
// parent qui n'en a aucune (DETACHED_PROCESS, executable -H windowsgui, service). Un parent lance
// avec CREATE_NO_WINDOW recoit au contraire une console cachee que ses enfants partagent : pas de
// fenetre, avec ou sans ce paquet (cas mesures le 2026-11-02 sur Windows 11, temoin
// fenetre_e2e_win_test.go).
//
// CONTRAT. [Command] rend la meme commande que `exec.CommandContext`, avec, sous Windows,
// CREATE_NO_WINDOW et HideWindow : l'enfant ne recoit aucune console et n'ouvre aucune fenetre,
// quel que soit le mode de lancement du serveur (console visible, console cachee, aucune console).
// Ce paquet ne dit rien d'autre : il n'explique pas, a lui seul, une fenetre vue sur l'application
// installee. Hors Windows, rien ne change.
// Ces drapeaux n'empechent ni les tubes ni la redirection : un enfant dont le parent relaie la sortie
// (cmd.StdoutPipe, cmd.Stdout = tampon) la garde. Un enfant qui a besoin d'une console propre, ou
// qui doit en partager une, n'est pas lance ici.
//
// GARDE-RAIL. Tout lancement de processus sous internal/ passe par ce paquet
// (archlint/childproc_single_spawn_test.go) : un `exec.Command` direct reouvre la fenetre.
package childproc

import (
	"context"
	"os/exec"
)

// Command construit l'enfant `name args...` lie a ctx, sans fenetre de console sous Windows.
// L'appelant garde la main sur l'environnement, les tubes et le demarrage, comme avec os/exec.
func Command(ctx context.Context, name string, args ...string) *exec.Cmd {
	cmd := exec.CommandContext(ctx, name, args...)
	sansFenetre(cmd)
	return cmd
}
