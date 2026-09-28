package atomicfile

import (
	"os"
	"path/filepath"
	"syscall"
	"testing"
)

// assertSeulFichier vérifie que `dir` ne contient que la cible (aucun temporaire
// résiduel) et que son contenu vaut `want`.
func assertSeulFichier(t *testing.T, dir, path, want string) {
	t.Helper()
	got, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("ReadFile: %v", err)
	}
	if string(got) != want {
		t.Errorf("contenu = %q, attendu %q", got, want)
	}
	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatalf("ReadDir: %v", err)
	}
	if len(entries) != 1 {
		t.Errorf("%d entrée(s) dans le répertoire, attendu 1 (temporaire résiduel ?)", len(entries))
	}
}

func TestWriteFileStrict_Atomique(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "chunk_00.bin")

	if err := WriteFileStrict(path, []byte("octets"), 0o644); err != nil {
		t.Fatalf("WriteFileStrict: %v", err)
	}
	assertSeulFichier(t, dir, path, "octets")
}

// TestWriteFileStrict_RenameRefuseRendUneErreur — EBUSY au rename : l'écriture
// stricte ne replie JAMAIS sur l'in-place (troncature = perte irréversible).
func TestWriteFileStrict_RenameRefuseRendUneErreur(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "chunk_00.bin")
	if err := os.WriteFile(path, []byte("ancien"), 0o644); err != nil {
		t.Fatalf("seed: %v", err)
	}
	withRename(t, func(_, _ string) error {
		return &os.LinkError{Op: "rename", Err: syscall.EBUSY}
	})

	if err := WriteFileStrict(path, []byte("nouveau"), 0o644); err == nil {
		t.Fatal("erreur attendue : un rename refusé ne doit pas replier sur l'in-place")
	}
	assertSeulFichier(t, dir, path, "ancien")
}

// TestWriteFileStrict_TemporaireImpossibleRendUneErreur — temporaire impossible :
// erreur, cible intacte.
func TestWriteFileStrict_TemporaireImpossibleRendUneErreur(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "chunk_00.bin")
	if err := os.WriteFile(path, []byte("ancien"), 0o644); err != nil {
		t.Fatalf("seed: %v", err)
	}
	withCreateTemp(t, func(_, _ string) (*os.File, error) {
		return nil, &os.PathError{Op: "open", Path: dir, Err: syscall.EACCES}
	})

	if err := WriteFileStrict(path, []byte("nouveau"), 0o644); err == nil {
		t.Fatal("erreur attendue : un temporaire impossible ne doit pas replier sur l'in-place")
	}
	assertSeulFichier(t, dir, path, "ancien")
}

// TestWriteFileStrict_CibleExistanteRemplaceeEntiere — un contenu plus court
// remplace entièrement l'ancien (aucune queue résiduelle).
func TestWriteFileStrict_CibleExistanteRemplaceeEntiere(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "chunk_00.bin")
	if err := os.WriteFile(path, []byte("un contenu nettement plus long"), 0o644); err != nil {
		t.Fatalf("seed: %v", err)
	}

	if err := WriteFileStrict(path, []byte("court"), 0o644); err != nil {
		t.Fatalf("WriteFileStrict: %v", err)
	}
	assertSeulFichier(t, dir, path, "court")
}
