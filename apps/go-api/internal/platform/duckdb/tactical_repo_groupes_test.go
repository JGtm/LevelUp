// Package duckdb — tactical_repo_groupes_test.go : le lecteur tactique lit UN GROUPE de cartes — la clause
// carte de l'univers porte la liste des identifiants du groupe (`TacticalQuery.MapIDs`), et la lecture
// spatiale refuse une liste vide (plan cartes variantes, E3).
package duckdb

import (
	"context"
	"testing"
)

// Deux map_id (la base et sa variante) lus ensemble donnent UN univers et UNE lecture de positions ; un
// seul identifiant ne voit que le sien ; aucune carte (liste vide) = toutes les cartes pour l'univers.
func TestTacticalRepo_UniversEtPositionsSurDeuxIdentifiants(t *testing.T) {
	pdb := newTacticalTestPlayerDB(t)
	seedTacticalCorpus(t, pdb) // m1, m2 sur tacCarteA ; m3 sur tacCarteB (positions m1 et m3)
	repo := NewTacticalRepo(pdb)
	ctx := context.Background()

	groupe := tacQuery("")
	groupe.MapIDs = []string{tacCarteA, tacCarteB}

	univers, err := repo.Univers(ctx, groupe)
	if err != nil {
		t.Fatalf("Univers: %v", err)
	}
	if want := []string{"m1", "m2", "m3"}; !egales(matchIDs(univers.Matchs), want) {
		t.Errorf("univers du groupe = %v, attendu %v", matchIDs(univers.Matchs), want)
	}

	seul, err := repo.Univers(ctx, tacQuery(tacCarteA))
	if err != nil {
		t.Fatalf("Univers (A seul): %v", err)
	}
	if want := []string{"m1", "m2"}; !egales(matchIDs(seul.Matchs), want) {
		t.Errorf("univers de A seul = %v, attendu %v", matchIDs(seul.Matchs), want)
	}

	toutes, err := repo.Univers(ctx, tacQuery(""))
	if err != nil {
		t.Fatalf("Univers (toutes): %v", err)
	}
	if len(toutes.Matchs) != 3 {
		t.Errorf("univers sans carte = %v, attendu les 3 matchs du joueur", matchIDs(toutes.Matchs))
	}

	pos, err := repo.KillPositions(ctx, groupe)
	if err != nil {
		t.Fatalf("KillPositions: %v", err)
	}
	vus := map[string]bool{}
	for _, p := range pos.Points {
		vus[p.MatchID] = true
	}
	if !vus["m1"] || !vus["m3"] {
		t.Errorf("positions du groupe : matchs lus %v, attendu m1 (carte A) et m3 (carte B)", vus)
	}
}

// Le refus d'une lecture spatiale sans carte vaut pour une liste vide comme pour l'ancien identifiant vide.
func TestTacticalRepo_KillPositionsRefuseUneListeDeCartesVide(t *testing.T) {
	pdb := newTacticalTestPlayerDB(t)
	seedTacticalCorpus(t, pdb)
	if _, err := NewTacticalRepo(pdb).KillPositions(context.Background(), tacQuery("")); err == nil {
		t.Error("KillPositions sans carte doit etre refusee")
	}
}
