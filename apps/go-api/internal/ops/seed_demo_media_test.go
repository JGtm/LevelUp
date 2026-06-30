package ops

import (
	"path/filepath"
	"testing"
)

func TestResolveMediaPath(t *testing.T) {
	base := filepath.Join("repo", "data", "media")

	// Relatif (cas prod : file_path = "JGtm/clip.mp4") → réancré sur la base.
	if got, want := resolveMediaPath(base, "JGtm/clip.mp4"), filepath.Join(base, "JGtm", "clip.mp4"); got != want {
		t.Errorf("relatif: got %q, want %q", got, want)
	}
	// Absolu (cas local : file_path déjà absolu) → inchangé.
	abs := filepath.Join(t.TempDir(), "clip.mp4")
	if got := resolveMediaPath(base, abs); got != abs {
		t.Errorf("absolu: got %q, want %q (inchangé)", got, abs)
	}
	// Base vide → inchangé (rétrocompat / tests).
	if got := resolveMediaPath("", "JGtm/clip.mp4"); got != "JGtm/clip.mp4" {
		t.Errorf("base vide: got %q", got)
	}
	// Path vide → vide (pas de miniature).
	if got := resolveMediaPath(base, ""); got != "" {
		t.Errorf("path vide: got %q", got)
	}
}

// TestNumericMediaID_BigintCastable : l'ID dérivé est un entier positif décimal — donc
// castable en BIGINT par insertDemoMediaRow (média_match_associations_history.media_file_id).
// C'est ce qui manquait au chemin Infinite (id = nom de fichier → CAST échouait).
func TestNumericMediaID_BigintCastable(t *testing.T) {
	for _, name := range []string{"Halo Infinite 2026-01-21 21-30-40", "Halo_5_Guardians-2018-08-08_22h33.mp4"} {
		id := numericMediaID(name)
		if id == "" {
			t.Fatalf("id vide pour %q", name)
		}
		for _, r := range id {
			if r < '0' || r > '9' {
				t.Fatalf("id non numérique %q pour %q (CAST BIGINT échouerait)", id, name)
			}
		}
	}
	// Déterministe (reseed idempotent).
	if numericMediaID("x") != numericMediaID("x") {
		t.Error("numericMediaID doit être déterministe")
	}
}
