//go:build windows

package filmproc

import (
	"os/exec"
	"syscall"

	"golang.org/x/sys/windows"
)

// sansFenetre : l'enfant ne recoit aucune console. Un parent sans console (le serveur du profil
// desktop, lance par la coquille avec CREATE_NO_WINDOW) fait sinon naitre une fenetre de terminal
// vide pour chaque enfant console. Le relais passe par des tubes, que ces drapeaux n'affectent pas.
// Copie volontaire de internal/platform/childproc (module separe, ADR 0041 : une brique n'importe
// pas go-api) ; une troisieme copie se centraliserait.
func sansFenetre(cmd *exec.Cmd) {
	if cmd.SysProcAttr == nil {
		cmd.SysProcAttr = &syscall.SysProcAttr{}
	}
	cmd.SysProcAttr.HideWindow = true
	cmd.SysProcAttr.CreationFlags |= windows.CREATE_NO_WINDOW
}
