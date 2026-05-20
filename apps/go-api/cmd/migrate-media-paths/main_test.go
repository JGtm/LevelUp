package main

import (
	"os"
	"path/filepath"
	"testing"

	"levelup/go-api/internal/ops"
)

func TestConvertPath_StandardMultiPlayer(t *testing.T) {
	store := ops.MediaPathStore{CapturesBase: filepath.FromSlash("C:/Captures")}

	abs := filepath.FromSlash("C:/Captures/Madina97294/thumbs/x.webp")
	got := convertPath(store, abs, "Madina97294")
	want := "Madina97294/thumbs/x.webp"
	if got != want {
		t.Errorf("standard multi-player: got %q, want %q", got, want)
	}
}

func TestConvertPath_LegacyInternalLayout(t *testing.T) {
	// Path legacy hors capturesBase (ex: ancien data/players/halo_infinite/JGtm/captures/thumbs/xxx.gif).
	// Stratégie heuristique : extraire à partir de /{slug}/.
	store := ops.MediaPathStore{CapturesBase: filepath.FromSlash("C:/Captures")}

	abs := filepath.FromSlash("D:/old/data/players/halo_infinite/JGtm/captures/thumbs/Replay_bd811d72870d.gif")
	got := convertPath(store, abs, "JGtm")
	want := "JGtm/captures/thumbs/Replay_bd811d72870d.gif"
	if got != want {
		t.Errorf("legacy internal layout: got %q, want %q", got, want)
	}
}

func TestConvertPath_SlugAtStart(t *testing.T) {
	// Cas dégénéré : path qui commence par {slug}/ (déjà partiellement relatif).
	store := ops.MediaPathStore{CapturesBase: filepath.FromSlash("C:/Captures")}

	// On considère ça comme déjà relatif et on retourne tel quel (forward slashes).
	abs := filepath.FromSlash("JGtm/thumbs/x.webp")
	got := convertPath(store, abs, "JGtm")
	want := "JGtm/thumbs/x.webp"
	if got != want {
		t.Errorf("slug at start: got %q, want %q", got, want)
	}
}

func TestConvertPath_NoMatchingSlug(t *testing.T) {
	store := ops.MediaPathStore{CapturesBase: filepath.FromSlash("C:/Captures")}

	abs := filepath.FromSlash("D:/random/file.mp4")
	got := convertPath(store, abs, "JGtm")
	if got != "" {
		t.Errorf("no match: got %q, want empty", got)
	}
}

func TestConvertPath_EmptySlug(t *testing.T) {
	store := ops.MediaPathStore{CapturesBase: filepath.FromSlash("C:/Captures")}
	got := convertPath(store, filepath.FromSlash("C:/Captures/x.mp4"), "")
	if got != "" {
		t.Errorf("empty slug: got %q, want empty", got)
	}
}

func TestLoadCapturesBase_ReadsJSON(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "app_settings.json")
	content := `{"media_captures_base_dir": "C:\\Custom\\Path"}`
	if err := writeFile(path, content); err != nil {
		t.Fatalf("write: %v", err)
	}
	got := loadCapturesBase(path)
	if got != `C:\Custom\Path` {
		t.Errorf("loadCapturesBase: got %q, want %q", got, `C:\Custom\Path`)
	}
}

func TestLoadCapturesBase_MissingFileReturnsEmpty(t *testing.T) {
	if got := loadCapturesBase(filepath.Join(t.TempDir(), "missing.json")); got != "" {
		t.Errorf("missing file: got %q, want empty", got)
	}
}

func writeFile(path, content string) error {
	return os.WriteFile(path, []byte(content), 0o644)
}
