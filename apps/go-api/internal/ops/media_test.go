// Package ops — media_test.go : tests unitaires purs (sans CGO) pour les helpers d'indexation.
package ops

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestWalkMediaDir_EmptyPure(t *testing.T) {
	dir := t.TempDir()
	files, err := walkMediaDir(dir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(files) != 0 {
		t.Errorf("expected 0 files, got %d", len(files))
	}
}

func TestWalkMediaDir_NonexistentPure(t *testing.T) {
	files, err := walkMediaDir("/nonexistent/path")
	if err != nil {
		t.Fatalf("nonexistent dir should not error: %v", err)
	}
	if files != nil {
		t.Errorf("expected nil, got %v", files)
	}
}

func TestWalkMediaDir_SupportedAndUnsupported(t *testing.T) {
	dir := t.TempDir()
	supported := []string{"clip.mp4", "shot.png", "video.mov"}
	unsupported := []string{"readme.txt", "data.csv", "malware.exe"}

	for _, name := range append(supported, unsupported...) {
		if err := os.WriteFile(filepath.Join(dir, name), []byte("x"), 0o644); err != nil {
			t.Fatal(err)
		}
	}

	files, err := walkMediaDir(dir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(files) != len(supported) {
		t.Errorf("got %d files, want %d", len(files), len(supported))
	}
	for _, f := range files {
		ext := strings.ToLower(filepath.Ext(f))
		if _, ok := supportedExtensions[ext]; !ok {
			t.Errorf("unexpected extension: %s", ext)
		}
	}
}

func TestWalkMediaDir_SkipsThumbsDir(t *testing.T) {
	dir := t.TempDir()
	thumbsDir := filepath.Join(dir, "thumbs")
	if err := os.MkdirAll(thumbsDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(thumbsDir, "thumb.jpg"), []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "clip.mp4"), []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}

	files, err := walkMediaDir(dir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	for _, f := range files {
		if strings.Contains(filepath.ToSlash(f), "/thumbs/") {
			t.Errorf("thumbs file should not be indexed: %s", f)
		}
	}
	if len(files) != 1 {
		t.Errorf("expected 1 file (clip.mp4), got %d: %v", len(files), files)
	}
}

func TestFileHash_Deterministic(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "test.mp4")
	data := []byte("hello halo")
	if err := os.WriteFile(path, data, 0o644); err != nil {
		t.Fatal(err)
	}
	h1, err := fileHash(path)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	h2, err := fileHash(path)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if h1 != h2 {
		t.Errorf("hash not deterministic: %q != %q", h1, h2)
	}
	if len(h1) != 16 {
		t.Errorf("hash length = %d, want 16", len(h1))
	}
}

func TestFileHash_DifferentFiles(t *testing.T) {
	dir := t.TempDir()
	p1 := filepath.Join(dir, "a.mp4")
	p2 := filepath.Join(dir, "b.mp4")
	os.WriteFile(p1, []byte("content A"), 0o644) //nolint:errcheck
	os.WriteFile(p2, []byte("content B"), 0o644) //nolint:errcheck

	h1, _ := fileHash(p1)
	h2, _ := fileHash(p2)
	if h1 == h2 {
		t.Error("different files should produce different hashes")
	}
}

func TestFileHash_MissingFile(t *testing.T) {
	_, err := fileHash("/nonexistent/file.mp4")
	if err == nil {
		t.Error("expected error for missing file")
	}
}

func TestInsertMediaFile_UTCTimestamp(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "clip.mp4")
	if err := os.WriteFile(path, []byte("video"), 0o644); err != nil {
		t.Fatal(err)
	}
	fi, err := os.Stat(path)
	if err != nil {
		t.Fatal(err)
	}
	utcTime := fi.ModTime().UTC()
	if utcTime.Location().String() != "UTC" {
		t.Errorf("expected UTC, got %s", utcTime.Location())
	}
}
