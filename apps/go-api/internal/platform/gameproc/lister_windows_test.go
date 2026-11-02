//go:build windows

package gameproc

import (
	"os"
	"path/filepath"
	"testing"
)

// TestSystemLister_VoitLeProcessusDeTest : l'énumération réelle (lecture seule des noms) rend au moins
// l'exécutable du test lui-même, qui est un processus en cours.
func TestSystemLister_VoitLeProcessusDeTest(t *testing.T) {
	self, err := os.Executable()
	if err != nil {
		t.Fatalf("exécutable du test : %v", err)
	}
	names, err := NewSystemLister().Names()
	if err != nil {
		t.Fatalf("énumération : %v", err)
	}
	if !AnyRunning(names, []string{filepath.Base(self)}) {
		t.Fatalf("le processus de test %q n'est pas dans l'énumération (%d processus)", filepath.Base(self), len(names))
	}
}
