package sync

import (
	"context"
	"testing"

	"levelup/go-api/internal/domain"
)

// engine_history_complete_test.go — le rattrapage de l'historique : une page vide marque la fin
// de l'historique (HistoryExhausted). La clé sync_meta elle-même est testée dans
// internal/sync/historystate.

// Une page vide = fin de l'historique : le résultat le dit ; une page perdue ne le dit pas.
func TestPaginateAndPersistHistory_PageVide_HistoriqueEpuise(t *testing.T) {
	e := &SyncEngine{gamertag: "J900004", xuid: "2601461636236373"}
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
