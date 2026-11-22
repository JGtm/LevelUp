//go:build integration

// Package livesync — history_integration_test.go : la clé history_complete_at d'une player DB
// Halo 5 se lit et se pose par les hooks câblés dans Deps (sonde puis marqueur), sur un vrai
// fichier DuckDB.
package livesync

import (
	"context"
	"path/filepath"
	"testing"

	syncpkg "levelup/go-api/internal/sync"
)

func TestHistoryHooks_RoundTrip(t *testing.T) {
	ctx := context.Background()
	playerPath := filepath.Join(t.TempDir(), "players", "JGtm", "stats.duckdb")

	probe := historyCompleteProbe(playerPath, "JGtm")
	mark := historyCompleteMarker(playerPath, "JGtm")

	if probe(ctx) {
		t.Fatal("clé absente : l'historique ne doit pas être réputé complet")
	}
	mark(ctx)
	if !probe(ctx) {
		t.Fatal("après la pose, l'historique doit être réputé complet")
	}

	db, err := syncpkg.OpenPlayerDB(playerPath)
	if err != nil {
		t.Fatalf("OpenPlayerDB: %v", err)
	}
	defer db.Close()
	v, ok, err := syncpkg.GetSyncMeta(ctx, db.SQLDb(), syncpkg.SyncMetaHistoryCompleteAt)
	if err != nil || !ok || v == "" {
		t.Fatalf("GetSyncMeta(%s) = %q ok=%v err=%v, want valeur posée", syncpkg.SyncMetaHistoryCompleteAt, v, ok, err)
	}
}
