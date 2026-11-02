// shutdown_stdin.go — arrêt propre à la fermeture de stdin (drapeau -arret-sur-stdin).
//
// Le processus qui lance le serveur (coquille desktop) tient le bout écrivain du tube de
// stdin : il le ferme pour demander l'arrêt, et l'OS le ferme à sa place s'il meurt. Aucun
// signal n'est nécessaire, ce qui est le seul arrêt propre sous Windows (os.Interrupt n'y
// atteint pas un processus sans console partagée, SIGTERM n'y existe pas).
package main

import (
	"context"
	"io"
	"log/slog"
)

// Causes d'arrêt journalisées (reason de "arrêt: demandé" et "arrêt: terminé").
const (
	shutdownReasonSignal = "signal"
	shutdownReasonStdin  = "stdin"
)

// watchStdin lit r (stdin du processus) en arrière-plan jusqu'à EOF ou erreur, puis demande
// l'arrêt par requestShutdown. Seule la fermeture demande l'arrêt : les lignes lues avant sont
// des commandes de la coquille (stdin_commands.go), jamais un ordre d'arrêt. La goroutine rend
// la main dès la demande ; si l'arrêt vient d'ailleurs avant, elle reste bloquée sur la lecture
// jusqu'à la fin du processus.
func (b *bootServer) watchStdin(r io.Reader) {
	go func() {
		ctx := context.Background()
		if err := drainStdin(ctx, r, &b.commands); err != nil {
			slog.WarnContext(ctx, "arrêt: lecture de stdin en erreur, arrêt demandé", "err", err)
		} else {
			slog.InfoContext(ctx, "arrêt: stdin fermé")
		}
		b.requestShutdown(shutdownReasonStdin)
	}()
}
