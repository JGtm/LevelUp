//go:build integration

// Package sync — engine_history_pending_integration_test.go : le moteur V1 déclare, par
// SyncResult.HistoryPending, qu'une passe laisse l'historique du joueur à rattraper, et range la
// clé de fin d'historique (internal/sync/historystate) quand la fin a été vue sans perte.
package sync

import (
	"errors"
	"testing"

	"levelup/go-api/internal/domain"
	"levelup/go-api/internal/sync/historystate"
)

func historyPendingOpts() domain.SyncOptions {
	return domain.SyncOptions{
		MatchType:         "matchmaking",
		MaxMatches:        50,
		WithParticipants:  true,
		WithMedals:        true,
		RequestsPerSecond: 100,
	}
}

// historyKeyRanged relit la clé de fin d'historique dans la base joueur du moteur.
func historyKeyRanged(t *testing.T, env *e2eTestEnv) bool {
	t.Helper()
	handle, err := OpenPlayerDB(env.playerPath)
	if err != nil {
		t.Fatalf("OpenPlayerDB: %v", err)
	}
	defer handle.Close()
	return historystate.Complete(t.Context(), handle.SQLDb())
}

// Première passe sur un historique que l'API épuise (page vide) : la fin est rangée, rien n'est
// en attente ; la passe suivante (historique complet) n'a rien en attente non plus.
func TestEngineRun_HistoriqueEpuise_RangeLaFinEtNEstPasEnAttente(t *testing.T) {
	env := newE2EEnv(t)
	env.seedMockMatches("hist-ok", 3)

	result, err := env.engine.RunDelta(t.Context(), historyPendingOpts())
	if err != nil {
		t.Fatalf("RunDelta: %v", err)
	}
	if !result.HistoryExhausted || result.HistoryPending {
		t.Fatalf("HistoryExhausted=%v HistoryPending=%v, attendu true/false", result.HistoryExhausted, result.HistoryPending)
	}
	if !historyKeyRanged(t, env) {
		t.Fatal("la fin d'historique vue sans perte doit être rangée")
	}

	again, err := env.engine.RunDelta(t.Context(), historyPendingOpts())
	if err != nil {
		t.Fatalf("RunDelta (2e passe): %v", err)
	}
	if again.HistoryPending {
		t.Error("historique déjà complet : rien en attente")
	}
}

// Page d'historique perdue : la fin n'est pas vue, la clé reste absente, la passe se déclare en
// attente pour que la suivante rattrape.
func TestEngineRun_HistoriqueInterrompu_RestEnAttente(t *testing.T) {
	env := newE2EEnv(t)
	env.mock.getHistoryErr = errors.New("api indisponible")

	result, _ := env.engine.RunDelta(t.Context(), historyPendingOpts())
	if len(result.Errors) == 0 {
		t.Fatalf("une page perdue doit être comptée en erreur : %+v", result)
	}
	if !result.HistoryPending {
		t.Error("fin d'historique non vue : la passe doit se déclarer en attente")
	}
	if historyKeyRanged(t, env) {
		t.Error("la clé ne doit pas être rangée sur une passe en erreur")
	}
}
