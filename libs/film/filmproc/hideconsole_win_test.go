//go:build windows

package filmproc

import (
	"os/exec"
	"testing"

	"golang.org/x/sys/windows"
)

func TestSansFenetreSousWindows(t *testing.T) {
	cmd := exec.Command("cmd.exe")
	cmd.SysProcAttr = nil
	lowPriority(cmd)
	sansFenetre(cmd)
	if !cmd.SysProcAttr.HideWindow {
		t.Error("HideWindow doit etre vrai")
	}
	const attendus = windows.CREATE_NO_WINDOW | belowNormalPriorityClass
	if cmd.SysProcAttr.CreationFlags&attendus != attendus {
		t.Errorf("drapeaux = %#x, attendus %#x au moins (priorite basse conservee)",
			cmd.SysProcAttr.CreationFlags, attendus)
	}
}
