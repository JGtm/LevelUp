// Package childproc : le lanceur unique des processus enfants console du serveur.
//
// POURQUOI. Windows donne une console NEUVE et visible a tout processus console lance par un
// parent qui n'en a pas. Le profil desktop lance le serveur depuis la coquille sans console
// (CREATE_NO_WINDOW) : sans precaution, chaque ffmpeg, ffprobe ou enfant de cuisson du serveur
// ouvrirait une fenetre de terminal vide a l'ecran de l'utilisateur.
//
// CONTRAT. [Command] rend la meme commande que `exec.CommandContext`, avec, sous Windows,
// CREATE_NO_WINDOW et HideWindow : l'enfant ne recoit aucune console. Hors Windows, rien ne change.
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
