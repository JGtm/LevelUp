//go:build windows

package childproc

import (
	"context"
	"testing"

	"golang.org/x/sys/windows"
)

func TestCommandSansFenetreSousWindows(t *testing.T) {
	cmd := Command(context.Background(), "cmd.exe", "/c", "exit", "0")
	attr := cmd.SysProcAttr
	if attr == nil {
		t.Fatal("SysProcAttr nil : l'enfant ouvrirait une console")
	}
	if !attr.HideWindow {
		t.Error("HideWindow doit etre vrai")
	}
	if attr.CreationFlags&windows.CREATE_NO_WINDOW == 0 {
		t.Errorf("CREATE_NO_WINDOW absent : %#x", attr.CreationFlags)
	}
}

func TestCommandConserveLesDrapeauxDeCreation(t *testing.T) {
	cmd := Command(context.Background(), "cmd.exe")
	cmd.SysProcAttr.CreationFlags |= windows.CREATE_NEW_PROCESS_GROUP
	sansFenetre(cmd)
	if cmd.SysProcAttr.CreationFlags&windows.CREATE_NEW_PROCESS_GROUP == 0 ||
		cmd.SysProcAttr.CreationFlags&windows.CREATE_NO_WINDOW == 0 {
		t.Fatalf("drapeaux perdus : %#x", cmd.SysProcAttr.CreationFlags)
	}
}
