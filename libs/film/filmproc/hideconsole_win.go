//go:build windows

package filmproc

import (
	"os/exec"
	"syscall"

	"golang.org/x/sys/windows"
)

// sansFenetre : l'enfant ne recoit aucune console et n'ouvre aucune fenetre, quel que soit le mode
// de lancement du serveur (console visible, console cachee, aucune console ; cas mesures le
// 2026-11-02). Un parent sans aucune console (DETACHED_PROCESS, executable -H windowsgui, service)
// ferait sinon naitre une fenetre de terminal par enfant console ; un parent lance avec
// CREATE_NO_WINDOW partage sa console cachee. Le relais passe par des tubes, que ces drapeaux
// n'affectent pas.
// Copie volontaire de internal/platform/childproc (module separe, ADR 0041 : une brique n'importe
// pas go-api) ; une troisieme copie se centraliserait.
func sansFenetre(cmd *exec.Cmd) {
	if cmd.SysProcAttr == nil {
		cmd.SysProcAttr = &syscall.SysProcAttr{}
	}
	cmd.SysProcAttr.HideWindow = true
	cmd.SysProcAttr.CreationFlags |= windows.CREATE_NO_WINDOW
}
