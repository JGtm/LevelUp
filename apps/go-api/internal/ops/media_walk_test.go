package ops

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestWalkMediaDir_SkipsHLSAndThumbs garde-rail anti-régression (PLAN_MEDIA_HLS
// piège n°1) : les fichiers de l'arbre HLS — en particulier init.mp4 dont
// l'extension .mp4 est pourtant supportée — ne doivent JAMAIS être indexés comme
// médias. thumbs/ reste également ignoré.
func TestWalkMediaDir_SkipsHLSAndThumbs(t *testing.T) {
	dir := t.TempDir()
	hlsDir := filepath.Join(dir, "hls", "clip")
	thumbsDir := filepath.Join(dir, "thumbs")
	for _, d := range []string{hlsDir, thumbsDir} {
		if err := os.MkdirAll(d, 0o755); err != nil {
			t.Fatal(err)
		}
	}
	fixtures := []string{
		filepath.Join(dir, "clip.mp4"),          // seul média légitime
		filepath.Join(hlsDir, "init_a0.mp4"),    // .mp4 supporté MAIS sous hls/ → ignoré
		filepath.Join(hlsDir, "seg_a0_000.m4s"), // extension non supportée
		filepath.Join(hlsDir, "master.m3u8"),    // playlist
		filepath.Join(thumbsDir, "clip.webp"),   // miniature
	}
	for _, f := range fixtures {
		if err := os.WriteFile(f, []byte("x"), 0o644); err != nil {
			t.Fatal(err)
		}
	}

	files, err := walkMediaDir(dir)
	if err != nil {
		t.Fatalf("walkMediaDir: %v", err)
	}
	if len(files) != 1 || filepath.Base(files[0]) != "clip.mp4" {
		t.Fatalf("walkMediaDir = %v, want [clip.mp4]", files)
	}
	for _, f := range files {
		if strings.Contains(filepath.ToSlash(f), "/hls/") {
			t.Errorf("fichier de l'arbre HLS indexé par erreur: %s", f)
		}
	}
}
