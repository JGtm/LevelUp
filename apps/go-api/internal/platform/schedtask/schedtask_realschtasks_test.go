//go:build e2e_windows

package schedtask

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"os"
	"path/filepath"
	"testing"
	"time"
)

// TestRealSchtasks_PoseInterrogueSupprime joue le vrai schtasks.exe de la machine : la définition XML est
// acceptée par le planificateur, la liste porte la tâche, un second Ensure la remplace, la suppression la
// retire (et une seconde est un succès). Seul test du paquet qui touche au planificateur réel :
//
//   - la tâche porte un nom de test aléatoire (`LevelUp-test-<hex>`), JAMAIS `LevelUp\Refresh` ;
//   - son premier déclenchement est dans 48 heures et son exécutable est inoffensif : elle ne part jamais ;
//   - elle est supprimée en fin de test, même en cas d'échec.
//
// Étiquette e2e_windows : lancé à la main (`go test -tags=e2e_windows ./internal/platform/schedtask/`).
func TestRealSchtasks_PoseInterrogueSupprime(t *testing.T) {
	suffix := make([]byte, 4)
	if _, err := rand.Read(suffix); err != nil {
		t.Fatal(err)
	}
	name := "LevelUp-test-" + hex.EncodeToString(suffix)
	root := t.TempDir()
	m := &Manager{
		Runner: ExecRunner{},
		Name:   name,
		Spec: Spec{
			ExePath:       filepath.Join(os.Getenv("SystemRoot"), "System32", "whoami.exe"),
			DataRoot:      root,
			ResourcesRoot: root,
			LogsRoot:      root,
		},
		Now: func() time.Time { return time.Now().Add(48 * time.Hour) },
	}
	ctx := context.Background()
	t.Cleanup(func() {
		if err := m.Remove(ctx); err != nil {
			t.Errorf("nettoyage de la tâche de test %s : %v", name, err)
		}
	})

	if state, err := m.Query(ctx); err != nil || state != StateAbsent {
		t.Fatalf("avant : %q, %v ; attendu absent", state, err)
	}
	if err := m.Ensure(ctx); err != nil {
		t.Fatalf("Ensure : %v", err)
	}
	if state, err := m.Query(ctx); err != nil || state != StatePresent {
		t.Fatalf("après Ensure : %q, %v ; attendu present", state, err)
	}
	if err := m.Ensure(ctx); err != nil {
		t.Fatalf("second Ensure (remplacement) : %v", err)
	}
	if err := m.Remove(ctx); err != nil {
		t.Fatalf("Remove : %v", err)
	}
	if state, err := m.Query(ctx); err != nil || state != StateAbsent {
		t.Fatalf("après Remove : %q, %v ; attendu absent", state, err)
	}
	if err := m.Remove(ctx); err != nil {
		t.Fatalf("second Remove (déjà absente) : %v", err)
	}
}
