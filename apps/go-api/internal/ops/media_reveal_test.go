package ops

// media_reveal_test.go — ce que Reveal demande à l'explorateur : le fichier sélectionné s'il existe,
// sinon le dossier existant le plus proche. Le lanceur est une doublure : aucun test n'ouvre un
// explorateur.

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"testing"
)

type revealCall struct {
	target   string
	selectIt bool
}

func revealerWith(calls *[]revealCall, err error) *OSFileRevealer {
	return &OSFileRevealer{open: func(_ context.Context, target string, selectIt bool) error {
		*calls = append(*calls, revealCall{target, selectIt})
		return err
	}}
}

func TestReveal_ExistingFileIsSelected(t *testing.T) {
	file := filepath.Join(t.TempDir(), "clip.mp4")
	if err := os.WriteFile(file, []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}
	var calls []revealCall
	if err := revealerWith(&calls, nil).Reveal(context.Background(), file); err != nil {
		t.Fatalf("Reveal: %v", err)
	}
	if len(calls) != 1 || calls[0] != (revealCall{file, true}) {
		t.Errorf("appels = %+v, attendu la sélection de %s", calls, file)
	}
}

func TestReveal_MissingFileOpensNearestExistingFolder(t *testing.T) {
	root := t.TempDir()
	missing := filepath.Join(root, "sub", "deeper", "clip.mp4")
	var calls []revealCall
	if err := revealerWith(&calls, nil).Reveal(context.Background(), missing); err != nil {
		t.Fatalf("Reveal: %v", err)
	}
	if len(calls) != 1 || calls[0] != (revealCall{root, false}) {
		t.Errorf("appels = %+v, attendu le dossier %s sans sélection", calls, root)
	}
}

func TestReveal_MissingFileInExistingFolder(t *testing.T) {
	root := t.TempDir()
	var calls []revealCall
	if err := revealerWith(&calls, nil).Reveal(context.Background(), filepath.Join(root, "gone.mp4")); err != nil {
		t.Fatalf("Reveal: %v", err)
	}
	if len(calls) != 1 || calls[0] != (revealCall{root, false}) {
		t.Errorf("appels = %+v, attendu le dossier %s", calls, root)
	}
}

func TestReveal_RelativePathRefused(t *testing.T) {
	var calls []revealCall
	for _, p := range []string{"", "JGtm/clip.mp4"} {
		if err := revealerWith(&calls, nil).Reveal(context.Background(), p); err == nil {
			t.Errorf("chemin %q : erreur attendue", p)
		}
	}
	if len(calls) != 0 {
		t.Errorf("explorateur lancé pour un chemin non absolu : %+v", calls)
	}
}

func TestReveal_LaunchFailureIsReturned(t *testing.T) {
	boom := errors.New("explorateur indisponible")
	var calls []revealCall
	err := revealerWith(&calls, boom).Reveal(context.Background(), t.TempDir())
	if !errors.Is(err, boom) {
		t.Errorf("err = %v, attendu l'échec du lancement", err)
	}
}

func TestNearestExistingDir(t *testing.T) {
	root := t.TempDir()
	if got, ok := nearestExistingDir(filepath.Join(root, "a", "b", "c.mp4")); !ok || got != root {
		t.Errorf("nearestExistingDir = %q, %v ; attendu %q", got, ok, root)
	}
}
