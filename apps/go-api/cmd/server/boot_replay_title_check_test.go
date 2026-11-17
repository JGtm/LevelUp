package main

import (
	"context"
	"io"
	"log/slog"
	"path/filepath"
	"runtime"
	"sync/atomic"
	"testing"
)

// TestReplayArtifactTitleCheck : le prédicat lit la capacité `film.replay_artifact` du titre (vrai pour un titre qui
// la déclare, faux pour un titre qui ne l'expose pas ou qui n'existe pas) et rend la même réponse au second appel.
func TestReplayArtifactTitleCheck(t *testing.T) {
	_, ce, _, _ := runtime.Caller(0)
	racine := filepath.Join(filepath.Dir(ce), "..", "..", "..", "..")
	check := replayArtifactTitleCheck(racine)
	for range 2 { // second tour : réponse mémoïsée, identique
		if !check("halo_infinite") {
			t.Error("halo_infinite déclare film.replay_artifact : attendu vrai")
		}
		if check("halo_5") {
			t.Error("halo_5 n'expose pas film.replay_artifact : attendu faux")
		}
		if check("titre_inexistant") {
			t.Error("un titre inconnu : attendu faux")
		}
	}
}

// compteurWarn compte les enregistrements de niveau Warn reçus.
type compteurWarn struct {
	slog.Handler
	n *atomic.Int32
}

func (c compteurWarn) Enabled(context.Context, slog.Level) bool { return true }

func (c compteurWarn) Handle(_ context.Context, r slog.Record) error {
	if r.Level == slog.LevelWarn {
		c.n.Add(1)
	}
	return nil
}

// TestReplayArtifactTitleCheck_EchecMemorise : une configuration de titre illisible se journalise UNE fois
// par titre (pas à chaque requête de /bootstrap) et reste fausse aux appels suivants.
func TestReplayArtifactTitleCheck_EchecMemorise(t *testing.T) {
	n := &atomic.Int32{}
	avant := slog.Default()
	slog.SetDefault(slog.New(compteurWarn{Handler: slog.NewTextHandler(io.Discard, nil), n: n}))
	t.Cleanup(func() { slog.SetDefault(avant) })

	check := replayArtifactTitleCheck(t.TempDir()) // racine sans mappings : toute lecture échoue
	for range 5 {
		if check("titre_inexistant") {
			t.Fatal("configuration illisible : attendu faux")
		}
	}
	if got := n.Load(); got != 1 {
		t.Errorf("avertissements journalisés = %d, want 1 (échec mémorisé)", got)
	}
}
