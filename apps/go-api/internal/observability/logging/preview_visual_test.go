package logging

import (
	"log/slog"
	"os"
	"testing"
	"time"
)

// TestVisualPreview_ConsoleFormat affiche un échantillon de logs sur stderr
// pour validation visuelle du format compact. Activé via :
//
//	go test -v -run TestVisualPreview_ConsoleFormat ./internal/observability/logging/
//
// Skippé en CI : ne sert qu'à l'inspection humaine. Pas d'assertion.
func TestVisualPreview_ConsoleFormat(t *testing.T) {
	if os.Getenv("LEVELUP_LOG_PREVIEW") != "1" {
		t.Skip("set LEVELUP_LOG_PREVIEW=1 pour afficher le preview")
	}
	h := NewConsoleHandler(os.Stderr, ConsoleHandlerOptions{
		Level:    slog.LevelDebug,
		MaxWidth: 200,
		Color:    false,
	})
	logger := slog.New(h)

	logger.Info("sync.postSync: pipeline démarré",
		"gamertag", "Madina97294",
		"matches_inserted", 3,
		"event_id", "sync.RunDelta:abc123", // skipped on console
	)
	logger.Warn("halo_api: GET HTTP error",
		"status", 429,
		"url", "https://halo.api/spnkr",
		"retry_after", 30*time.Second,
	)
	logger.Error("provider.swap: reopen RO failed",
		"attempts", 3,
		"path", "C:\\Users\\GuillaumeSITBON\\data\\shared.duckdb",
		"err", "file locked by other process",
	)
	logger.Debug("scheduler.tick: tour terminé",
		"players_evaluated", 5,
		"skipped", 2,
		"duration_ms", 125,
	)
	// Ligne très longue pour montrer le tronquage.
	longURL := "https://halo.api/spnkr/very/long/path/that/exceeds/200/chars/abcdefghijklmnopqrstuvwxyz/0123456789/0123456789/0123456789/0123456789/0123456789/0123456789/0123456789/end"
	logger.Info("watcher.rta: presence event",
		"xuid", "2535401234567890",
		"url", longURL,
	)
}
