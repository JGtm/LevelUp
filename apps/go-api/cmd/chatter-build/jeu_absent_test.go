//go:build !gamefiles

package main

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"testing"
)

func TestSansLeTagGamefilesLesEtapesDuJeuLeDisent(t *testing.T) {
	travail := t.TempDir()
	if err := os.WriteFile(filepath.Join(travail, "sons.json"), []byte(`{"films":1,"sons":{"0000000a":1}}`), 0o644); err != nil {
		t.Fatal(err)
	}
	err := executer(context.Background(), []string{"textes", "-travail", travail}, journalDeTest(t))
	if !errors.Is(err, errSansTag) {
		t.Errorf("erreur = %v : une étape qui lit le jeu doit dire qu'il faut le tag gamefiles", err)
	}
}
