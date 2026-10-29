//go:build windows

package filmproc

// priority_windows.go — LA PRIORITE CPU BASSE DE L'ENFANT, sur Windows.
//
// POURQUOI CETTE LIGNE EXISTE (leçon du 2026-08-26). Une mesure bornee en memoire mais lancee a
// priorite NORMALE prend quand meme la machine : le decodage d'un film BTB sature les coeurs, et
// l'utilisateur ne peut plus travailler pendant que sa propre mesure tourne. Le plafond memoire
// protege la MACHINE de la mort ; la priorite basse protege son USAGE.
//
// `BELOW_NORMAL_PRIORITY_CLASS` plutot que `IDLE_PRIORITY_CLASS` : au ralenti absolu, un film
// lourd ne finirait jamais tant qu'autre chose tourne. « En dessous de la normale » cede le pas
// a l'interface sans condamner la mesure.

import (
	"os/exec"
	"syscall"
)

// belowNormalPriorityClass — la constante Windows (winbase.h). Elle n'est pas exposee par le
// paquet `syscall` de Go : elle est ecrite ici avec sa source plutot qu'importee d'ailleurs.
const belowNormalPriorityClass = 0x00004000

// lowPriority fait naitre l'enfant en priorite BASSE.
func lowPriority(cmd *exec.Cmd) {
	if cmd.SysProcAttr == nil {
		cmd.SysProcAttr = &syscall.SysProcAttr{}
	}
	cmd.SysProcAttr.CreationFlags |= belowNormalPriorityClass
}
