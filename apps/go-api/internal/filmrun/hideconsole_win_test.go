//go:build windows

package filmrun

import (
	"context"
	"testing"

	"golang.org/x/sys/windows"
)

// TestSansFenetreSousWindows : l'enfant du lanceur nait sans console (childproc) ET en priorite
// basse ; les deux drapeaux de creation se cumulent, aucun n'efface l'autre.
func TestSansFenetreSousWindows(t *testing.T) {
	r, err := NewRunner("racine", nil)
	if err != nil {
		t.Fatal(err)
	}
	cmd := r.command(context.Background(), nil)
	if !cmd.SysProcAttr.HideWindow {
		t.Error("HideWindow doit etre vrai")
	}
	const attendus = windows.CREATE_NO_WINDOW | belowNormalPriorityClass
	if cmd.SysProcAttr.CreationFlags&attendus != attendus {
		t.Errorf("drapeaux = %#x, attendus %#x au moins (priorite basse conservee)",
			cmd.SysProcAttr.CreationFlags, attendus)
	}
}
