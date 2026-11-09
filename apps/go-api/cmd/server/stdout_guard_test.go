// stdout_guard_test.go — en -mode desktop, stdout ne porte que les événements du contrat (ADR 0045
// D-2) : la coquille lit chaque ligne comme du JSON. Garde-rail : dans le code de cmd/server, la
// seule référence à la sortie standard est celle qui construit l'écrivain d'événements, et aucun
// fmt.Print* n'écrit dessus. Tout le reste va sur stderr (journal console, bandeaux de démarrage).
package main

import (
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
)

// stdoutWrite repère une écriture sur la sortie standard : os.Stdout, ou fmt.Print/Printf/Println
// (qui écrivent sur os.Stdout).
var stdoutWrite = regexp.MustCompile(`os\.Stdout|fmt\.Print(f|ln)?\(`)

// stdoutAllowed : les seuls usages admis, par fichier source (hors tests).
var stdoutAllowed = map[string]int{
	"boot_listen.go": 1, // newDesktopEvents(os.Stdout)
}

func TestSortieStandard_SeulLEcrivainDEvenementsEcrit(t *testing.T) {
	files, err := filepath.Glob("*.go")
	if err != nil {
		t.Fatal(err)
	}
	for _, f := range files {
		if strings.HasSuffix(f, "_test.go") {
			continue
		}
		src, err := os.ReadFile(f)
		if err != nil {
			t.Fatal(err)
		}
		count := 0
		for _, line := range strings.Split(string(src), "\n") {
			code, _, _ := strings.Cut(line, "//")
			if stdoutWrite.MatchString(code) {
				count++
			}
		}
		if count != stdoutAllowed[f] {
			t.Errorf("%s : %d référence(s) à la sortie standard, %d admise(s) : en -mode desktop, stdout ne porte "+
				"que les événements (desktop_events.go) ; le journal et les bandeaux vont sur stderr", f, count, stdoutAllowed[f])
		}
	}
}
