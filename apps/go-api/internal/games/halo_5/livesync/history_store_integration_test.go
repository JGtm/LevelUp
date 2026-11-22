//go:build integration

// history_store_integration_test.go : la lecture de la clé de fin en échec remonte en erreur, jamais en
// « complet » (D-25) ; RunDelta en fait un delta ordinaire en attente (catchup_test.go, stateErr).
package livesync

import (
	"context"
	"testing"

	_ "github.com/duckdb/duckdb-go/v2"

	syncpkg "levelup/go-api/internal/sync"
)

func TestPlayerHistory_CleIllisible_ErreurJamaisComplet(t *testing.T) {
	h, path := tempPlayerHistory(t)
	db, err := syncpkg.OpenPlayerDB(path) // base créée : seule la lecture de la clé échoue
	if err != nil {
		t.Fatalf("OpenPlayerDB: %v", err)
	}
	_ = db.Close()

	// Lecture en échec : contexte fini pendant la lecture (la base s'ouvre, la requête échoue).
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	complete, err := h.Complete(ctx)
	if err == nil || complete {
		t.Fatalf("clé illisible : complet=%v err=%v, attendu (faux, erreur)", complete, err)
	}
}
