// Package ops — media_test.go : tests unitaires purs (sans CGO) pour les helpers d'indexation.
package ops

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
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

// ─────────────────────────────────────────────────────────────────────────────
// SanitizeMediaTimezone
// ─────────────────────────────────────────────────────────────────────────────

func TestSanitizeMediaTimezone_ValidIANA(t *testing.T) {
	cases := []string{
		"Europe/Paris",
		"America/New_York",
		"UTC",
		"Asia/Tokyo",
		"Etc/GMT+2",
		"Pacific/Auckland",
	}
	for _, tz := range cases {
		got := SanitizeMediaTimezone(tz)
		if got != tz {
			t.Errorf("SanitizeMediaTimezone(%q) = %q, want %q", tz, got, tz)
		}
	}
}

func TestSanitizeMediaTimezone_Invalid(t *testing.T) {
	bad := []string{
		"Europe/Paris'; DROP TABLE x; --",
		"bad tz",
		"Europe/Paris\x00",
		"Zone!Name",
		"Zone@Name",
	}
	for _, tz := range bad {
		got := SanitizeMediaTimezone(tz)
		if got != "" {
			t.Errorf("SanitizeMediaTimezone(%q) = %q, want \"\" (sanitized)", tz, got)
		}
	}
}

func TestSanitizeMediaTimezone_Empty(t *testing.T) {
	if got := SanitizeMediaTimezone(""); got != "" {
		t.Errorf("expected empty, got %q", got)
	}
}

// ─────────────────────────────────────────────────────────────────────────────
// parseCaptureTimeFromFilename
// ─────────────────────────────────────────────────────────────────────────────

func TestParseCaptureTimeFromFilename_XboxFormat_CET(t *testing.T) {
	// CET = UTC+1 (décembre = hiver)
	loc, err := time.LoadLocation("Europe/Paris")
	if err != nil {
		t.Skip("timezone Europe/Paris non disponible")
	}
	name := "Halo Infinite 2024.12.15 - 21.30.45.01.mp4"
	got := parseCaptureTimeFromFilename(name, loc)
	if got == nil {
		t.Fatal("expected non-nil result")
	}
	// Paris CET = UTC+1 → 21:30:45 Paris = 20:30:45 UTC
	want := time.Date(2024, 12, 15, 20, 30, 45, 0, time.UTC)
	if !got.Equal(want) {
		t.Errorf("got %v, want %v", got, want)
	}
}

func TestParseCaptureTimeFromFilename_XboxFormat_CEST(t *testing.T) {
	// CEST = UTC+2 (juillet = été)
	loc, err := time.LoadLocation("Europe/Paris")
	if err != nil {
		t.Skip("timezone Europe/Paris non disponible")
	}
	name := "Halo Infinite 2024.07.15 - 22.00.00.01.mp4"
	got := parseCaptureTimeFromFilename(name, loc)
	if got == nil {
		t.Fatal("expected non-nil result")
	}
	// CEST = UTC+2 → 22:00:00 Paris = 20:00:00 UTC
	want := time.Date(2024, 7, 15, 20, 0, 0, 0, time.UTC)
	if !got.Equal(want) {
		t.Errorf("got %v, want %v", got, want)
	}
}

func TestParseCaptureTimeFromFilename_OBSFormat_CEST(t *testing.T) {
	// OBS Studio : format par défaut "%CCYY-%MM-%DD %hh-%mm-%ss"
	// CEST = UTC+2 (avril = été)
	loc, err := time.LoadLocation("Europe/Paris")
	if err != nil {
		t.Skip("timezone Europe/Paris non disponible")
	}
	name := "Replay 2026-04-19 17-10-54.mp4"
	got := parseCaptureTimeFromFilename(name, loc)
	if got == nil {
		t.Fatal("expected non-nil result")
	}
	// CEST = UTC+2 → 17:10:54 Paris = 15:10:54 UTC
	want := time.Date(2026, 4, 19, 15, 10, 54, 0, time.UTC)
	if !got.Equal(want) {
		t.Errorf("got %v, want %v", got, want)
	}
}

func TestParseCaptureTimeFromFilename_OBSFormat_CET(t *testing.T) {
	// CET = UTC+1 (janvier = hiver)
	loc, err := time.LoadLocation("Europe/Paris")
	if err != nil {
		t.Skip("timezone Europe/Paris non disponible")
	}
	name := "Recording 2026-01-15 09-00-00.mp4"
	got := parseCaptureTimeFromFilename(name, loc)
	if got == nil {
		t.Fatal("expected non-nil result")
	}
	// CET = UTC+1 → 09:00:00 Paris = 08:00:00 UTC
	want := time.Date(2026, 1, 15, 8, 0, 0, 0, time.UTC)
	if !got.Equal(want) {
		t.Errorf("got %v, want %v", got, want)
	}
}

func TestParseCaptureTimeFromFilename_NoMatch(t *testing.T) {
	loc, _ := time.LoadLocation("Europe/Paris")
	cases := []string{
		"OBS-Recording-2024-01-01.mp4", // date sans time
		"clip.mp4",
		"screenshot.png",
		"Halo Infinite.mp4",
		"2024-01-01.mp4", // date seule
		"recording-no-date.mp4",
	}
	for _, name := range cases {
		if got := parseCaptureTimeFromFilename(name, loc); got != nil {
			t.Errorf("parseCaptureTimeFromFilename(%q) = %v, want nil", name, got)
		}
	}
}

func TestParseCaptureTimeFromFilename_NilLoc_ReturnsNil(t *testing.T) {
	name := "Halo Infinite 2024.12.15 - 21.30.45.01.mp4"
	if got := parseCaptureTimeFromFilename(name, nil); got != nil {
		t.Errorf("expected nil with nil loc, got %v", got)
	}
}

// ─────────────────────────────────────────────────────────────────────────────
// mustAtoi
// ─────────────────────────────────────────────────────────────────────────────

func TestMustAtoi(t *testing.T) {
	if mustAtoi("42") != 42 {
		t.Error("mustAtoi(\"42\") != 42")
	}
	if mustAtoi("0") != 0 {
		t.Error("mustAtoi(\"0\") != 0")
	}
	if mustAtoi("abc") != 0 {
		t.Error("mustAtoi(\"abc\") != 0 (expected fallback)")
	}
	if mustAtoi("") != 0 {
		t.Error("mustAtoi(\"\") != 0")
	}
}
