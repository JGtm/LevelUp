package testutil

import (
	"os"
	"testing"
)

// TestCheminBriqueFilmDesigneLeModuleDuDecodeur : le chemin rendu porte le `go.mod` du module.
func TestCheminBriqueFilmDesigneLeModuleDuDecodeur(t *testing.T) {
	if _, err := os.Stat(CheminBriqueFilm(t, "go.mod")); err != nil {
		t.Fatalf("go.mod de la brique du decodeur introuvable : %v", err)
	}
}
