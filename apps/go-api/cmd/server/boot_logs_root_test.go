package main

import (
	"io/fs"
	"log/slog"
	"os"
	"path/filepath"
	"runtime/debug"
	"strings"
	"testing"

	"levelup/go-api/internal/observability/logging"
)

// Racine de journaux NEUVE : le dossier est créé avant l'ouverture de la capture des plantages,
// qui est donc présente dès le premier démarrage (ADR 0008, amendement « two roots », lot L11).
func TestSetupLogging_RacineNeuveCreeLaCaptureDesPlantages(t *testing.T) {
	base, err := os.MkdirTemp("", "levelup-logs-neuve-")
	if err != nil {
		t.Fatal(err)
	}
	logs := filepath.Join(base, "racine", "neuve", "logs") // aucun parent n'existe
	t.Setenv("LEVELUP_LOGS_DIR", logs)
	t.Setenv("LEVELUP_LOGS_ENABLED", "")

	prev := slog.Default()
	st := &bootState{}
	setupLogging(st)
	t.Cleanup(func() {
		slog.SetDefault(prev)
		if st.multiHandler != nil {
			_ = st.multiHandler.Close()
		}
		debug.SetCrashOutput(nil, debug.CrashOptions{})
		// La capture garde son descripteur ouvert : sous Windows la suppression peut échouer,
		// ce que le test tolère (dossier temporaire du système).
		_ = os.RemoveAll(base)
	})

	var crash string
	_ = filepath.WalkDir(logs, func(path string, d fs.DirEntry, err error) error {
		if err == nil && !d.IsDir() && d.Name() == "server.crash.log" {
			crash = path
		}
		return nil
	})
	if crash == "" {
		t.Fatalf("server.crash.log absent sous %s après le premier démarrage", logs)
	}
	data, err := os.ReadFile(crash)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(data), "=== server start") {
		t.Errorf("en-tête de démarrage absent de la capture : %q", string(data))
	}
}

func TestEnsureLogsRoot_JournauxDesactivesNeCreeRien(t *testing.T) {
	// LogsDir vide = journaux désactivés : aucun dossier, aucune panique.
	ensureLogsRoot(logging.Config{})
}
