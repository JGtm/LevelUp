package sync

import (
	"context"
	"path/filepath"
	"testing"

	"levelup/go-api/internal/domain"
)

// engine_history_complete_test.go — le rattrapage de l'historique : une page vide marque la fin
// de l'historique (HistoryExhausted), la clé sync_meta se range et se relit.

// Une page vide = fin de l'historique : le résultat le dit ; une page perdue ne le dit pas.
func TestPaginateAndPersistHistory_PageVide_HistoriqueEpuise(t *testing.T) {
	e := &SyncEngine{gamertag: "Nuzzles", xuid: "2533274873954866"}
	in := historyPaginationInputs{
		client: &mockHaloClient{history: nil},
		opts:   domain.SyncOptions{MaxMatches: 25, MatchType: "matchmaking"},
	}
	result := &domain.SyncResult{}
	e.paginateAndPersistHistory(context.Background(), in, result)
	if !result.HistoryExhausted {
		t.Fatal("une page vide doit marquer la fin de l'historique")
	}
	if len(result.Errors) != 0 {
		t.Errorf("aucune erreur attendue, got %v", result.Errors)
	}
}

// La clé de fin d'historique : absente sur une base neuve (rattrapage), présente une fois rangée.
func TestHistoryComplete_AbsentePuisRangee(t *testing.T) {
	handle, err := OpenPlayerDB(filepath.Join(t.TempDir(), "stats.duckdb"))
	if err != nil {
		t.Fatalf("OpenPlayerDB: %v", err)
	}
	defer handle.Close()
	db := handle.SQLDb()
	ctx := context.Background()

	if historyComplete(ctx, db) {
		t.Fatal("base neuve : l'historique ne doit pas être réputé complet (un delta se joue en full)")
	}
	markHistoryComplete(ctx, db, "Nuzzles")
	if !historyComplete(ctx, db) {
		t.Fatal("après markHistoryComplete, l'historique doit être réputé complet")
	}
	if v, ok, err := GetSyncMeta(ctx, db, SyncMetaHistoryCompleteAt); err != nil || !ok || v == "" {
		t.Errorf("GetSyncMeta = (%q, %v, %v), attendu un horodatage", v, ok, err)
	}
	// Base indisponible : réputé complet (delta ordinaire), jamais un parcours intégral aveugle.
	if !historyComplete(ctx, nil) {
		t.Error("sans base, l'historique est réputé complet")
	}
}
