// Package duckdb — slog_capture_test.go : helper commun de capture du logger
// slog par défaut, partagé entre les tests du gate PAR DÉFAUT et les tests
// taggés `integration`. Fichier SANS build tag (condition du partage — un
// fichier sans contrainte de build est compilé dans les deux configurations).
//
// Lot D micro-hygiène (2026-08-26) : `captureSlog` (encodage JSON, dans
// match_view_scoreboard_objective_degrade_test.go, taggé `integration`) et
// `captureSlogText` (encodage texte, ci-dessous, utilisé par
// read_recovery_test.go, sans tag) réimplémentaient la même mécanique
// save/restore. Elle est centralisée ici via captureSlogWith.
//
// captureSlog reste défini dans son fichier taggé `integration` (et non ici) :
// golangci-lint (linter `unused`) lint le build PAR DÉFAUT, où ce fichier
// n'existe pas — y déplacer un wrapper qui n'y a aucun appelant le ferait
// classer à tort comme mort.
package duckdb

import (
	"bytes"
	"log/slog"
	"testing"
)

// captureSlogWith redirige le logger par défaut vers un buffer construit par
// newHandler pour la durée du test, et le restaure ensuite.
func captureSlogWith(t *testing.T, newHandler func(buf *bytes.Buffer) slog.Handler) *bytes.Buffer {
	t.Helper()
	var buf bytes.Buffer
	prev := slog.Default()
	slog.SetDefault(slog.New(newHandler(&buf)))
	t.Cleanup(func() { slog.SetDefault(prev) })
	return &buf
}

// captureSlogText redirige le logger par défaut vers un buffer TEXTE pour la
// durée du test et le restaure ensuite. Les logs du helper passent par le logger
// PACKAGE (slog.WarnContext / slog.ErrorContext), donc capturer le défaut suffit.
func captureSlogText(t *testing.T) *bytes.Buffer {
	return captureSlogWith(t, func(buf *bytes.Buffer) slog.Handler {
		return slog.NewTextHandler(buf, &slog.HandlerOptions{Level: slog.LevelDebug})
	})
}
