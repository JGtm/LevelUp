//go:build windows

package childproc

import (
	"os/exec"
	"syscall"

	"golang.org/x/sys/windows"
)

// sansFenetre pose CREATE_NO_WINDOW (pas de console pour l'enfant) et HideWindow (fenetre
// de l'enfant masquee s'il en ouvre une). Les drapeaux de creation deja poses sont conserves.
func sansFenetre(cmd *exec.Cmd) {
	if cmd.SysProcAttr == nil {
		cmd.SysProcAttr = &syscall.SysProcAttr{}
	}
	cmd.SysProcAttr.HideWindow = true
	cmd.SysProcAttr.CreationFlags |= windows.CREATE_NO_WINDOW
}
