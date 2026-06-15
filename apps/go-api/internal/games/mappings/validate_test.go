package mappings

import (
	"os"
	"path/filepath"
	"testing"
)

func TestValidateRequiredTOML(t *testing.T) {
	root := t.TempDir()
	slug := "test_title"
	dir := filepath.Join(root, "config", "titles", slug, "mappings")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}

	// Aucun fichier → 2 erreurs (fields + capabilities).
	if errs := ValidateRequiredTOML(root, slug); len(errs) != 2 {
		t.Fatalf("attendu 2 erreurs, got %d : %v", len(errs), errs)
	}

	// fields.toml seul → 1 erreur (capabilities manquant).
	if err := os.WriteFile(filepath.Join(dir, "fields.toml"), []byte("x = 1\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if errs := ValidateRequiredTOML(root, slug); len(errs) != 1 {
		t.Fatalf("attendu 1 erreur, got %d : %v", len(errs), errs)
	}

	// Les deux présents → 0 erreur.
	if err := os.WriteFile(filepath.Join(dir, "capabilities.toml"), []byte("x = 1\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if errs := ValidateRequiredTOML(root, slug); len(errs) != 0 {
		t.Errorf("attendu 0 erreur, got %d : %v", len(errs), errs)
	}
}
