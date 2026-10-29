//go:build windows

package filmproc

// selfpriority_windows.go — LA PRIORITE CPU BASSE DU PROCESSUS COURANT, sur Windows.
//
// POURQUOI CE FICHIER EXISTE A COTE DE `priority_windows.go`. Celui-la baisse la priorite d'un
// ENFANT qu'on va lancer (`exec.Cmd.SysProcAttr`), et c'est la seule forme dont le paquet avait
// besoin tant que tout decodage passait par un parent. Un CLI d'operateur comme
// `cmd/replay-build` n'a PAS de parent : il est lui-meme le processus qui decode, et il n'avait
// donc aucun moyen de se ranger derriere l'interface de l'utilisateur (sinistre du 2026-08-31).
//
// Meme choix de classe que pour l'enfant : `BELOW_NORMAL_PRIORITY_CLASS` cede le pas a
// l'interface sans condamner le decodage, la ou `IDLE` ne finirait jamais.

import (
	"log/slog"
	"syscall"
)

// LowerOwnPriority range le processus COURANT en priorite basse. Best-effort : un echec est
// journalise et n'arrete rien — la priorite protege le CONFORT d'usage, le plafond memoire
// protege la machine, et c'est lui qui a le droit de tuer.
func LowerOwnPriority(tool string) {
	kernel32 := syscall.NewLazyDLL("kernel32.dll")
	setPriorityClass := kernel32.NewProc("SetPriorityClass")
	getCurrentProcess := kernel32.NewProc("GetCurrentProcess")
	h, _, _ := getCurrentProcess.Call()
	if r, _, err := setPriorityClass.Call(h, uintptr(belowNormalPriorityClass)); r == 0 {
		slog.Warn("priorite basse non appliquee au processus courant", "err", err, "outil", tool)
		return
	}
	slog.Info("priorite CPU abaissee pour ce decodage", "outil", tool, "classe", "below_normal")
}
