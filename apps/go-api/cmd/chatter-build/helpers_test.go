package main

import (
	"io"
	"log/slog"
	"testing"
)

// journalDeTest : un journal qui n'écrit nulle part (les tests regardent les résultats, pas les messages).
func journalDeTest(t *testing.T) *slog.Logger {
	t.Helper()
	return slog.New(slog.NewTextHandler(io.Discard, nil))
}

// languesDeTest : les langues de l'application, telles que `config/locales.toml` les donne.
func languesDeTest(t *testing.T) []langueApp {
	t.Helper()
	l, err := languesDeLApp()
	if err != nil {
		t.Fatal(err)
	}
	return l
}
