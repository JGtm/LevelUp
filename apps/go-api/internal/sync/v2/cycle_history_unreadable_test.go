// Package v2 — cycle_history_unreadable_test.go : un état de l'historique illisible (lecture de la clé de
// fin en échec) n'est jamais pris pour « complet » ni pour un rattrapage (D-25) : delta ordinaire, aucune
// écriture de l'état, HistoryPending vrai pour que la cadence courte réessaie.
package v2

import (
	"context"
	"database/sql"
	gosync "sync"
	"testing"
)

func TestCycleHistory_EtatIllisible_DeltaSansEcritureEtEnAttente(t *testing.T) {
	e := newHistEnv(t, 300)
	if _, err := e.db.Exec(`DROP TABLE sync_meta`); err != nil {
		t.Fatal(err)
	}
	var mu gosync.Mutex
	writes := 0
	e.writer = func(context.Context, string) (*sql.DB, func(), error) {
		mu.Lock()
		writes++
		mu.Unlock()
		return e.db, func() {}, nil
	}

	out, err := e.cycle()
	if err != nil {
		t.Fatal(err)
	}
	if !out.HistoryPending {
		t.Error("HistoryPending = faux : un état illisible garde le joueur en attente (cadence courte)")
	}
	if writes != 0 {
		t.Errorf("%d ouvertures en écriture de l'état : rien ne se range sur un état illisible", writes)
	}
	// Delta ordinaire : la liste entière jusqu'à sa fin (300), pas une passe de rattrapage au budget (200).
	if got := e.knownCount(); got != 300 {
		t.Errorf("%d matchs rangés, want 300 (delta ordinaire, pas une passe de rattrapage)", got)
	}
}
