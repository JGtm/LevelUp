package atomicfile

import (
	"os"
	"path/filepath"
	"syscall"
	"testing"
)

// withRename remplace le point d'injection du rename pour la durée du test.
func withRename(t *testing.T, fn func(oldpath, newpath string) error) {
	t.Helper()
	prev := renameFile
	renameFile = fn
	t.Cleanup(func() { renameFile = prev })
}

// withCreateTemp remplace le point d'injection de création du temporaire.
func withCreateTemp(t *testing.T, fn func(dir, pattern string) (*os.File, error)) {
	t.Helper()
	prev := createTemp
	createTemp = fn
	t.Cleanup(func() { createTemp = prev })
}

func TestWriteFile_AtomicPath_WritesContentAndLeavesNoTemp(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "app_settings.json")

	if err := WriteFile(path, []byte(`{"a":1}`), 0o644); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}
	got, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("ReadFile: %v", err)
	}
	if string(got) != `{"a":1}` {
		t.Errorf("contenu = %q, attendu %q", got, `{"a":1}`)
	}
	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatalf("ReadDir: %v", err)
	}
	if len(entries) != 1 {
		t.Errorf("%d entrée(s) dans le répertoire, attendu 1 (temporaire non nettoyé ?)", len(entries))
	}
}

// TestWriteFile_RenameEBUSY_FallsBackInPlace — LE cas prod (bind-mount fichier) :
// le rename échoue en EBUSY, l'écriture doit quand même aboutir, sur le MÊME
// fichier, avec un contenu intact et sans temporaire résiduel.
func TestWriteFile_RenameEBUSY_FallsBackInPlace(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "app_settings.json")
	if err := os.WriteFile(path, []byte(`{"last_notified_version":"7.2.0"}`), 0o644); err != nil {
		t.Fatalf("seed: %v", err)
	}
	before, err := os.Stat(path)
	if err != nil {
		t.Fatalf("stat avant: %v", err)
	}

	var renameCalls int
	withRename(t, func(_, _ string) error {
		renameCalls++
		return &os.LinkError{Op: "rename", Err: syscall.EBUSY}
	})

	want := `{"last_notified_version":"7.3.0"}`
	if err := WriteFile(path, []byte(want), 0o644); err != nil {
		t.Fatalf("WriteFile (repli attendu) : %v", err)
	}
	if renameCalls != 1 {
		t.Errorf("rename appelé %d fois, attendu 1", renameCalls)
	}
	got, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("ReadFile: %v", err)
	}
	if string(got) != want {
		t.Errorf("contenu après repli = %q, attendu %q", got, want)
	}
	after, err := os.Stat(path)
	if err != nil {
		t.Fatalf("stat après: %v", err)
	}
	if after.Mode().Perm() != before.Mode().Perm() {
		t.Errorf("mode modifié par le repli : %v → %v", before.Mode().Perm(), after.Mode().Perm())
	}
	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatalf("ReadDir: %v", err)
	}
	if len(entries) != 1 {
		t.Errorf("%d entrée(s) après repli, attendu 1 (temporaire non nettoyé)", len(entries))
	}
}

// TestWriteFile_RenameOtherError_IsReported — un échec de rename qui n'est PAS
// EBUSY ne doit PAS être masqué par le repli (le repli couvre une contrainte
// d'environnement connue, pas un diagnostic manquant).
func TestWriteFile_RenameOtherError_IsReported(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "app_settings.json")
	if err := os.WriteFile(path, []byte(`{"v":1}`), 0o644); err != nil {
		t.Fatalf("seed: %v", err)
	}
	withRename(t, func(_, _ string) error {
		return &os.LinkError{Op: "rename", Err: syscall.ENOSPC}
	})

	if err := WriteFile(path, []byte(`{"v":2}`), 0o644); err == nil {
		t.Fatal("erreur attendue (ENOSPC ne doit pas déclencher le repli)")
	}
	got, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("ReadFile: %v", err)
	}
	if string(got) != `{"v":1}` {
		t.Errorf("le fichier a été modifié malgré l'échec : %q", got)
	}
	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatalf("ReadDir: %v", err)
	}
	if len(entries) != 1 {
		t.Errorf("%d entrée(s), attendu 1 (temporaire non nettoyé après échec)", len(entries))
	}
}

// TestWriteFile_TempCreationImpossible_FallsBackInPlace — répertoire parent non
// inscriptible : le temporaire est impossible mais l'inode cible reste ouvrable.
func TestWriteFile_TempCreationImpossible_FallsBackInPlace(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "app_settings.json")
	if err := os.WriteFile(path, []byte(`{"v":1}`), 0o644); err != nil {
		t.Fatalf("seed: %v", err)
	}
	withCreateTemp(t, func(_, _ string) (*os.File, error) {
		return nil, &os.PathError{Op: "open", Path: dir, Err: syscall.EACCES}
	})

	want := `{"v":2}`
	if err := WriteFile(path, []byte(want), 0o644); err != nil {
		t.Fatalf("WriteFile (repli attendu) : %v", err)
	}
	got, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("ReadFile: %v", err)
	}
	if string(got) != want {
		t.Errorf("contenu = %q, attendu %q", got, want)
	}
}

// TestWriteFile_InPlace_TruncatesLongerContent — le repli doit TRONQUER : écrire
// un contenu plus court ne doit pas laisser la queue de l'ancien contenu.
func TestWriteFile_InPlace_TruncatesLongerContent(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "app_settings.json")
	if err := os.WriteFile(path, []byte(`{"long":"aaaaaaaaaaaaaaaaaaaaaaaaaaaa"}`), 0o644); err != nil {
		t.Fatalf("seed: %v", err)
	}
	withRename(t, func(_, _ string) error {
		return &os.LinkError{Op: "rename", Err: syscall.EBUSY}
	})

	want := `{"a":1}`
	if err := WriteFile(path, []byte(want), 0o644); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}
	got, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("ReadFile: %v", err)
	}
	if string(got) != want {
		t.Errorf("contenu = %q, attendu %q (troncature manquée)", got, want)
	}
}

// TestWriteFile_CreatesMissingTarget — cible absente : le repli comme le chemin
// atomique doivent créer le fichier.
func TestWriteFile_CreatesMissingTarget(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "app_settings.json")
	withRename(t, func(_, _ string) error {
		return &os.LinkError{Op: "rename", Err: syscall.EBUSY}
	})

	if err := WriteFile(path, []byte(`{}`), 0o644); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}
	if got, err := os.ReadFile(path); err != nil || string(got) != `{}` {
		t.Fatalf("fichier non créé correctement : %q / %v", got, err)
	}
}
