package ops

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"testing"
)

// writeFileAt crée un fichier et ses répertoires parents.
func writeFileAt(t *testing.T, path string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("mkdir %s: %v", path, err)
	}
	if err := os.WriteFile(path, []byte("x"), 0o644); err != nil {
		t.Fatalf("write %s: %v", path, err)
	}
}

// TestOSMediaFileRemover_RemovesStoredPaths : les chemins relatifs stockés
// ({owner}/{rel}) sont résolus contre la base de captures puis retirés.
func TestOSMediaFileRemover_RemovesStoredPaths(t *testing.T) {
	base := t.TempDir()
	clip := filepath.Join(base, "JGtm", "clip.mp4")
	thumb := filepath.Join(base, "JGtm", "thumbs", "clip.webp")
	writeFileAt(t, clip)
	writeFileAt(t, thumb)

	r := NewOSMediaFileRemover(func() string { return base })
	n, err := r.RemoveMediaFiles(context.Background(),
		[]string{"JGtm/clip.mp4", "JGtm/thumbs/clip.webp"})
	if err != nil {
		t.Fatalf("RemoveMediaFiles: %v", err)
	}
	if n != 2 {
		t.Errorf("removed = %d, want 2", n)
	}
	for _, p := range []string{clip, thumb} {
		if _, err := os.Stat(p); !errors.Is(err, os.ErrNotExist) {
			t.Errorf("fichier toujours présent: %s", p)
		}
	}
}

// TestOSMediaFileRemover_Idempotent : un fichier déjà absent n'est pas une
// erreur et n'est pas compté — une suppression rejouée après un échec partiel
// doit converger, pas échouer.
func TestOSMediaFileRemover_Idempotent(t *testing.T) {
	base := t.TempDir()
	r := NewOSMediaFileRemover(func() string { return base })

	n, err := r.RemoveMediaFiles(context.Background(), []string{"JGtm/absent.mp4"})
	if err != nil {
		t.Fatalf("un fichier absent ne doit pas produire d'erreur: %v", err)
	}
	if n != 0 {
		t.Errorf("removed = %d, want 0", n)
	}
}

// TestOSMediaFileRemover_RemovesHLSDirectory : supprimer la seule playlist
// laisserait tous les segments (des centaines de Mo, invisibles et jamais
// réclamés). Le répertoire de transcodage part avec elle.
func TestOSMediaFileRemover_RemovesHLSDirectory(t *testing.T) {
	base := t.TempDir()
	hlsDir := filepath.Join(base, "JGtm", "hls", "clip")
	writeFileAt(t, filepath.Join(hlsDir, "master.m3u8"))
	writeFileAt(t, filepath.Join(hlsDir, "seg0.m4s"))
	writeFileAt(t, filepath.Join(hlsDir, "seg1.m4s"))

	r := NewOSMediaFileRemover(func() string { return base })
	n, err := r.RemoveMediaFiles(context.Background(), []string{"JGtm/hls/clip/master.m3u8"})
	if err != nil {
		t.Fatalf("RemoveMediaFiles: %v", err)
	}
	if n != 1 {
		t.Errorf("removed = %d, want 1 (le répertoire compte pour un)", n)
	}
	if _, err := os.Stat(hlsDir); !errors.Is(err, os.ErrNotExist) {
		t.Errorf("répertoire HLS toujours présent: %s", hlsDir)
	}
}

// TestOSMediaFileRemover_PlaylistOutsideHLSDir : une playlist qui n'est PAS dans
// un répertoire `hls/` ne doit entraîner AUCUNE suppression de répertoire — un
// file_path aberrant ne doit pas emporter un dossier de captures entier.
func TestOSMediaFileRemover_PlaylistOutsideHLSDir(t *testing.T) {
	base := t.TempDir()
	playlist := filepath.Join(base, "JGtm", "master.m3u8")
	sibling := filepath.Join(base, "JGtm", "autre-clip.mp4")
	writeFileAt(t, playlist)
	writeFileAt(t, sibling)

	r := NewOSMediaFileRemover(func() string { return base })
	if _, err := r.RemoveMediaFiles(context.Background(), []string{"JGtm/master.m3u8"}); err != nil {
		t.Fatalf("RemoveMediaFiles: %v", err)
	}
	if _, err := os.Stat(sibling); err != nil {
		t.Errorf("un fichier voisin a été emporté : %v", err)
	}
	if _, err := os.Stat(filepath.Dir(playlist)); err != nil {
		t.Errorf("le répertoire du joueur a été supprimé : %v", err)
	}
}

// TestOSMediaFileRemover_NoBase_RelativePathLeftAlone : sans base de captures,
// un chemin relatif n'est pas résolvable — on ne supprime RIEN à l'aveugle.
func TestOSMediaFileRemover_NoBase_RelativePathLeftAlone(t *testing.T) {
	r := NewOSMediaFileRemover(func() string { return "" })
	n, err := r.RemoveMediaFiles(context.Background(), []string{"JGtm/clip.mp4"})
	if err != nil {
		t.Fatalf("RemoveMediaFiles: %v", err)
	}
	if n != 0 {
		t.Errorf("removed = %d, want 0 (chemin non résolvable)", n)
	}
}

// TestOSMediaFileRemover_AbsoluteLegacyPath : les chemins absolus legacy
// (antérieurs à la migration des paths) restent traitables.
func TestOSMediaFileRemover_AbsoluteLegacyPath(t *testing.T) {
	base := t.TempDir()
	legacy := filepath.Join(base, "legacy-clip.mp4")
	writeFileAt(t, legacy)

	r := NewOSMediaFileRemover(func() string { return base })
	n, err := r.RemoveMediaFiles(context.Background(), []string{legacy})
	if err != nil {
		t.Fatalf("RemoveMediaFiles: %v", err)
	}
	if n != 1 {
		t.Errorf("removed = %d, want 1", n)
	}
	if _, err := os.Stat(legacy); !errors.Is(err, os.ErrNotExist) {
		t.Error("le chemin absolu legacy n'a pas été supprimé")
	}
}

// TestOSMediaFileRemover_EmptyAndBlankPathsIgnored : les entrées vides
// (thumbnail_path NULL) ne produisent ni erreur ni appel disque.
func TestOSMediaFileRemover_EmptyAndBlankPathsIgnored(t *testing.T) {
	r := NewOSMediaFileRemover(func() string { return t.TempDir() })
	n, err := r.RemoveMediaFiles(context.Background(), []string{"", "   "})
	if err != nil {
		t.Fatalf("RemoveMediaFiles: %v", err)
	}
	if n != 0 {
		t.Errorf("removed = %d, want 0", n)
	}
}
