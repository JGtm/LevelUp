//go:build windows && e2e_windows

package childproc

// Temoin de bout en bout : un parent lance SANS console (CREATE_NO_WINDOW, comme la coquille lance le
// serveur) fait-il naitre une fenetre de terminal visible quand il lance un enfant console ?
//
// Le test se re-execute comme parent sans console (role "parent"), qui lance `ping` (assez long pour
// etre observe) par childproc.Command ou par exec.Command nu ; le test exterieur compte les fenetres
// de console qui apparaissent entre-temps (EnumWindows). Lancer :
//
//	go test -tags=e2e_windows -run TestFenetreEnfantConsole -v ./internal/platform/childproc/

import (
	"context"
	"os"
	"os/exec"
	"strings"
	"syscall"
	"testing"
	"time"
	"unsafe"

	"golang.org/x/sys/windows"
)

const (
	envRole    = "CHILDPROC_E2E_ROLE"
	envLanceur = "CHILDPROC_E2E_LANCEUR" // "childproc" ou "exec"
)

// classesDeConsole : les classes de fenetre qu'une console visible peut prendre (conhost classique,
// hote de Windows Terminal).
var classesDeConsole = map[string]bool{
	"ConsoleWindowClass":            true,
	"CASCADIA_HOSTING_WINDOW_CLASS": true,
	"PseudoConsoleWindow":           false,
}

// fenetresDeConsoleVisibles rend l'ensemble des fenetres de console visibles du bureau.
func fenetresDeConsoleVisibles(t *testing.T) map[windows.HWND]string {
	t.Helper()
	trouvees := map[windows.HWND]string{}
	cb := windows.NewCallback(func(h windows.HWND, _ uintptr) uintptr {
		var buf [256]uint16
		n, err := windows.GetClassName(h, &buf[0], int32(len(buf)))
		if err != nil || n == 0 || !windows.IsWindowVisible(h) {
			return 1
		}
		if classe := windows.UTF16ToString(buf[:n]); classesDeConsole[classe] {
			trouvees[h] = classe
		}
		return 1
	})
	if err := windows.EnumWindows(cb, unsafe.Pointer(nil)); err != nil {
		t.Fatalf("EnumWindows : %v", err)
	}
	return trouvees
}

// TestParentSansConsole est le CORPS du parent sans console : inerte hors du role "parent".
func TestParentSansConsole(t *testing.T) {
	if os.Getenv(envRole) != "parent" {
		t.Skip("corps du parent, lance par TestFenetreEnfantConsole")
	}
	args := []string{"/c", "ping", "-n", "5", "127.0.0.1"}
	var cmd *exec.Cmd
	if os.Getenv(envLanceur) == "childproc" {
		cmd = Command(context.Background(), "cmd.exe", args...)
	} else {
		cmd = exec.Command("cmd.exe", args...)
	}
	if err := cmd.Start(); err != nil {
		t.Fatalf("lancement : %v", err)
	}
	_ = cmd.Wait()
}

// compterFenetres lance un parent sans console avec le lanceur demande et rend le nombre de
// fenetres de console apparues pendant la vie de son enfant.
func compterFenetres(t *testing.T, lanceur string, drapeauxDuParent uint32) int {
	t.Helper()
	avant := fenetresDeConsoleVisibles(t)
	parent := exec.Command(os.Args[0], "-test.run=^TestParentSansConsole$", "-test.count=1")
	parent.Env = append(os.Environ(), envRole+"=parent", envLanceur+"="+lanceur)
	parent.SysProcAttr = &syscall.SysProcAttr{HideWindow: true, CreationFlags: drapeauxDuParent}
	if err := parent.Start(); err != nil {
		t.Fatalf("lancement du parent : %v", err)
	}
	time.Sleep(2 * time.Second) // l'enfant ping vit ~4 s
	apres := fenetresDeConsoleVisibles(t)
	_ = parent.Wait()
	nouvelles := 0
	var classes []string
	for h, c := range apres {
		if _, deja := avant[h]; !deja {
			nouvelles++
			classes = append(classes, c)
		}
	}
	t.Logf("parent %#x, lanceur=%s : %d nouvelle(s) fenetre(s) de console %s", drapeauxDuParent, lanceur, nouvelles, strings.Join(classes, ","))
	return nouvelles
}

// TestFenetreEnfantConsole : un parent SANS console (DETACHED_PROCESS) fait naitre une fenetre par
// enfant console lance par exec.Command nu (temoin negatif) et aucune par childproc.Command.
//
// Un parent lance avec CREATE_NO_WINDOW, lui, recoit une console cachee que ses enfants partagent :
// aucune fenetre, avec ou sans childproc (observe le 2026-11-02, Windows 11 26200). C'est le cas du
// serveur lance par la coquille ; childproc couvre le parent qui n'a, lui, aucune console.
func TestFenetreEnfantConsole(t *testing.T) {
	if os.Getenv(envRole) != "" {
		t.Skip("role enfant")
	}
	if n := compterFenetres(t, "childproc", windows.DETACHED_PROCESS); n != 0 {
		t.Errorf("childproc.Command : %d fenetre(s) de console, attendu 0", n)
	}
	if n := compterFenetres(t, "exec", windows.DETACHED_PROCESS); n == 0 {
		t.Errorf("temoin negatif : exec.Command nu n'a ouvert aucune fenetre, le test ne prouve rien")
	}
}
