package main

import (
	"os"
	"path/filepath"
	"sort"
	"testing"
)

func TestListOwnerSlugs(t *testing.T) {
	dir := t.TempDir()
	for _, name := range []string{"Madina97294", "JGtm", "Chocoboflor"} {
		if err := os.MkdirAll(filepath.Join(dir, name), 0o755); err != nil {
			t.Fatalf("mkdir %s: %v", name, err)
		}
	}
	// Fichier (pas un dir) — doit être ignoré
	if err := os.WriteFile(filepath.Join(dir, "stray.txt"), []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}
	slugs, err := listOwnerSlugs(dir)
	if err != nil {
		t.Fatalf("listOwnerSlugs: %v", err)
	}
	sort.Strings(slugs)
	want := []string{"Chocoboflor", "JGtm", "Madina97294"}
	if len(slugs) != len(want) {
		t.Fatalf("got %d slugs, want %d (%v)", len(slugs), len(want), slugs)
	}
	for i := range want {
		if slugs[i] != want[i] {
			t.Errorf("slugs[%d] = %q, want %q", i, slugs[i], want[i])
		}
	}
}

func TestDeleteExistingThumbs_OnlyGifAndWebp(t *testing.T) {
	dir := t.TempDir()
	// Mélange : gif + webp à supprimer, autres à conserver
	files := map[string]string{
		"a.gif":  "delete",
		"b.webp": "delete",
		"c.GIF":  "delete", // case-insensitive
		"d.mp4":  "keep",
		"e.txt":  "keep",
	}
	for name := range files {
		if err := os.WriteFile(filepath.Join(dir, name), []byte("x"), 0o644); err != nil {
			t.Fatal(err)
		}
	}

	deleted, err := deleteExistingThumbs(dir, false)
	if err != nil {
		t.Fatalf("deleteExistingThumbs: %v", err)
	}
	if deleted != 3 {
		t.Errorf("deleted=%d, want 3", deleted)
	}
	// Vérifier que les non-gif/webp sont toujours là
	for name, label := range files {
		_, err := os.Stat(filepath.Join(dir, name))
		exists := err == nil
		if label == "keep" && !exists {
			t.Errorf("%s aurait dû être conservé", name)
		}
		if label == "delete" && exists {
			t.Errorf("%s aurait dû être supprimé", name)
		}
	}
}

func TestDeleteExistingThumbs_DryRunDoesNotDelete(t *testing.T) {
	dir := t.TempDir()
	for _, name := range []string{"a.gif", "b.webp"} {
		if err := os.WriteFile(filepath.Join(dir, name), []byte("x"), 0o644); err != nil {
			t.Fatal(err)
		}
	}

	deleted, err := deleteExistingThumbs(dir, true)
	if err != nil {
		t.Fatalf("deleteExistingThumbs: %v", err)
	}
	if deleted != 2 {
		t.Errorf("deleted (count) = %d, want 2", deleted)
	}
	// Fichiers toujours là
	for _, name := range []string{"a.gif", "b.webp"} {
		if _, err := os.Stat(filepath.Join(dir, name)); err != nil {
			t.Errorf("%s aurait dû être conservé en dry-run", name)
		}
	}
}

func TestDeleteExistingThumbs_NoDirIsNoOp(t *testing.T) {
	deleted, err := deleteExistingThumbs(filepath.Join(t.TempDir(), "missing"), false)
	if err != nil {
		t.Fatalf("missing dir devrait être no-op, got err: %v", err)
	}
	if deleted != 0 {
		t.Errorf("deleted=%d, want 0", deleted)
	}
}

func TestLoadCapturesBase_ReadsJSON(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "app_settings.json")
	if err := os.WriteFile(path, []byte(`{"media_captures_base_dir": "/custom"}`), 0o644); err != nil {
		t.Fatal(err)
	}
	got := loadCapturesBase(path)
	if got != "/custom" {
		t.Errorf("loadCapturesBase: got %q, want /custom", got)
	}
}
