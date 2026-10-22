//go:build integration

// Package duckdb — match_view_repo_assist_pairs_errors_test.go : le lecteur RÉEL de Q21d remonte
// ses échecs de lecture. Une portée à zéro rendue sur échec se lirait « journal des morts non
// publiable » sur la Vue match ; l'erreur, elle, donne l'état « lecture indisponible ». Deux échecs :
// contexte annulé, base non migrée.
package duckdb

import (
	"context"
	"errors"
	"testing"

	"levelup/go-api/internal/domain"
)

func TestMatchViewRepo_GetMatchAssistPairs_EchecRemonte(t *testing.T) {
	pdb := newTestPlayerDB(t)
	repo := NewMatchViewRepo(pdb, pTestXUID)

	ctx, cancel := context.WithCancel(context.Background())
	cancel() // la lecture ne peut pas aboutir : contexte annulé
	pairs, scope, err := repo.GetMatchAssistPairs(ctx, "m1")
	if err == nil {
		t.Fatalf("lecture sur contexte annulé : erreur attendue, obtenu nil (portée %+v)", scope)
	}
	if !errors.Is(err, context.Canceled) {
		t.Errorf("err = %v, attendu une erreur enveloppant context.Canceled", err)
	}
	if pairs != nil || scope != (domain.MatchAssistScopeRaw{}) {
		t.Errorf("sur échec : paires %v, portée %+v, attendu rien", pairs, scope)
	}
}

// TestMatchViewRepo_GetMatchAssistPairs_BaseNonMigreeRemonte — une base sans les résumés du journal
// (schéma antérieur à la migration de l'ADR 0039) : la lecture échoue et l'échec remonte, au lieu
// d'une portée nulle qui se lirait « non publiable ». Le cas nominal (schéma de production) est tenu
// par `match_view_repo_assist_pairs_test.go`.
func TestMatchViewRepo_GetMatchAssistPairs_BaseNonMigreeRemonte(t *testing.T) {
	pdb := newTestPlayerDB(t)
	if _, err := pdb.Shared.SQLDb().Exec(`DROP TABLE IF EXISTS match_kill_heads CASCADE`); err != nil {
		t.Fatalf("retrait des têtes : %v", err)
	}
	if _, err := pdb.Player.SQLDb().Exec(`DROP TABLE IF EXISTS match_kill_heads CASCADE`); err != nil {
		t.Fatalf("retrait des têtes (joueur) : %v", err)
	}
	repo := NewMatchViewRepo(pdb, pTestXUID)
	pairs, scope, err := repo.GetMatchAssistPairs(context.Background(), "m1")
	if err == nil {
		t.Fatalf("base non migrée : erreur attendue, obtenu nil (portée %+v)", scope)
	}
	if pairs != nil || scope != (domain.MatchAssistScopeRaw{}) {
		t.Errorf("sur échec : paires %v, portée %+v, attendu rien", pairs, scope)
	}
}
