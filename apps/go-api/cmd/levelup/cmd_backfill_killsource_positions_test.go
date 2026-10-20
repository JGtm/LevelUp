package main

// cmd_backfill_killsource_positions_test.go — UN CATALOGUE DE BORNES ILLISIBLE SE JOURNALISE (revue
// finale, decouverte 4, 2026-10-02). L echec ne passait que par la console (`fmt.Printf`) : aucune
// ligne de journal structure, alors qu il met de cote tous les films de la passe (CLAUDE.md regle 3).
//
// MUTATION VUE ROUGE : retirer le `slog.WarnContext` de l echec du catalogue.

import (
	"bytes"
	"context"
	"log/slog"
	"strings"
	"testing"

	"levelup/go-api/internal/config"
	titlePkg "levelup/go-api/internal/domain/title"
)

func TestPositionCaptureDeps_CatalogueIllisibleJournalise(t *testing.T) {
	var journal bytes.Buffer
	ancien := slog.Default()
	slog.SetDefault(slog.New(slog.NewTextHandler(&journal, nil)))
	t.Cleanup(func() { slog.SetDefault(ancien) })

	// Une racine vide : ni metadata, ni catalogue de bornes.
	capture, fermer := positionCaptureDeps(context.Background(), &config.AppConfig{RepoRoot: t.TempDir()},
		titlePkg.DefaultSlug, nil, nil)
	defer fermer()
	if capture.Cablee() {
		t.Fatal("capture cablee sans catalogue de bornes")
	}
	var ligne string
	for _, l := range strings.Split(journal.String(), "\n") {
		if strings.Contains(l, "catalogue de bornes") {
			ligne = l
		}
	}
	if !strings.Contains(ligne, "level=WARN") || !strings.Contains(ligne, "err=") {
		t.Fatalf("echec du catalogue absent du journal structure (WARN avec err) :\n%s", journal.String())
	}
}
