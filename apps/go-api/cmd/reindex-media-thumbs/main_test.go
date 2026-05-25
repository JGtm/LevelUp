package main

import (
	"os"
	"path/filepath"
	"testing"
)

// helper : crée thumbsDir et y dépose des fichiers vides aux noms donnés.
func seedThumbs(t *testing.T, names ...string) string {
	t.Helper()
	dir := t.TempDir()
	thumbsDir := filepath.Join(dir, "thumbs")
	if err := os.MkdirAll(thumbsDir, 0o755); err != nil {
		t.Fatal(err)
	}
	for _, n := range names {
		p := filepath.Join(thumbsDir, n)
		if err := os.WriteFile(p, []byte(""), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	return thumbsDir
}

func TestFindThumb_MatchesExactStem(t *testing.T) {
	thumbs := seedThumbs(t, "Replay 2026-03-27 23-17-43.webp")
	got := findThumb(thumbs, "Replay 2026-03-27 23-17-43.mp4")
	want := "Replay 2026-03-27 23-17-43.webp"
	if got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}

// Cas du bug observé : DB porte `Replay X_<hash>.gif`, disque a `Replay X.webp`.
// Le strip du suffixe hash doit permettre le match.
func TestFindThumb_StripsHashSuffix(t *testing.T) {
	thumbs := seedThumbs(t, "Replay 2026-03-27 23-17-43.webp")
	got := findThumb(thumbs, "Replay 2026-03-27 23-17-43_bd811d72870d.mp4")
	want := "Replay 2026-03-27 23-17-43.webp"
	if got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}

// Cas inverse : DB porte le stem nu, disque porte hash. Doit aussi matcher
// (strip appliqué des deux côtés).
func TestFindThumb_DiskHasHashSuffix(t *testing.T) {
	thumbs := seedThumbs(t, "Replay 2026-03-27 23-17-43_a1b2c3d4e5f6.webp")
	got := findThumb(thumbs, "Replay 2026-03-27 23-17-43.mp4")
	want := "Replay 2026-03-27 23-17-43_a1b2c3d4e5f6.webp"
	if got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}

func TestFindThumb_PrefersWebpOverGif(t *testing.T) {
	thumbs := seedThumbs(t,
		"Halo Infinite.gif",
		"Halo Infinite.webp",
	)
	got := findThumb(thumbs, "Halo Infinite.mp4")
	if got != "Halo Infinite.webp" {
		t.Errorf("got %q, want webp priority", got)
	}
}

func TestFindThumb_FallsBackToGifWhenOnlyGif(t *testing.T) {
	thumbs := seedThumbs(t, "Replay legacy.gif")
	got := findThumb(thumbs, "Replay legacy.mp4")
	if got != "Replay legacy.gif" {
		t.Errorf("got %q, want gif fallback", got)
	}
}

func TestFindThumb_NoMatchReturnsEmpty(t *testing.T) {
	thumbs := seedThumbs(t, "Other.webp")
	got := findThumb(thumbs, "Replay missing.mp4")
	if got != "" {
		t.Errorf("got %q, want empty (no match)", got)
	}
}

func TestFindThumb_MissingDirReturnsEmpty(t *testing.T) {
	got := findThumb(filepath.Join(t.TempDir(), "does-not-exist"), "any.mp4")
	if got != "" {
		t.Errorf("got %q, want empty (missing dir)", got)
	}
}

// Sanity : un nom sans extension dans la DB ne doit pas paniquer.
func TestFindThumb_NoExtensionInDB(t *testing.T) {
	thumbs := seedThumbs(t, "raw.webp")
	got := findThumb(thumbs, "raw")
	if got != "raw.webp" {
		t.Errorf("got %q, want raw.webp", got)
	}
}
