// Package duckdb — tactical_repo_ownership_test.go : tests TacticalRepo.MatchsOuvrables
// (ADR 0029, lot M1 — detail d'une cellule, Tactique S.1).
//
// Meme corpus que tactical_repo_test.go (seedTacticalCorpus) : m1/m2/m3 sont a moi
// (tacXUIDMoi), m4 est un match entre DEUX TIERS (tacXUIDTier vs tacXUIDAdv) — c'est
// le cas de reference « un match d'un autre joueur ».
package duckdb

import (
	"context"
	"testing"
	"time"
)

// TestTacticalRepo_MatchsOuvrables_MienOuvrableTiersNon : LE test du lot. Je demande
// l'ouvrabilite de m1 (a moi) ET m4 (a deux tiers) : seul m1 doit revenir dans la map.
func TestTacticalRepo_MatchsOuvrables_MienOuvrableTiersNon(t *testing.T) {
	pdb := newTacticalTestPlayerDB(t)
	seedTacticalCorpus(t, pdb)

	got, err := NewTacticalRepo(pdb).MatchsOuvrables(context.Background(), tacXUIDMoi, []string{"m1", "m4"})
	if err != nil {
		t.Fatalf("MatchsOuvrables: %v", err)
	}
	if _, ok := got["m1"]; !ok {
		t.Errorf("m1 doit etre ouvrable (j'y ai participe) : %+v", got)
	}
	if _, ok := got["m4"]; ok {
		t.Errorf("m4 ne doit PAS etre ouvrable (match entre deux tiers) : %+v", got)
	}
	if len(got) != 1 {
		t.Fatalf("map = %+v, want exactement 1 entree (m1)", got)
	}
}

// TestTacticalRepo_MatchsOuvrables_DateCanonique : la date rendue est celle du registre
// (start_time_utc canonique), pour le tri des contributions.
func TestTacticalRepo_MatchsOuvrables_DateCanonique(t *testing.T) {
	pdb := newTacticalTestPlayerDB(t)
	seedTacticalCorpus(t, pdb)
	base := time.Date(2026, 8, 1, 12, 0, 0, 0, time.UTC)

	got, err := NewTacticalRepo(pdb).MatchsOuvrables(context.Background(), tacXUIDMoi, []string{"m1", "m3"})
	if err != nil {
		t.Fatalf("MatchsOuvrables: %v", err)
	}
	if !got["m1"].Equal(base) {
		t.Errorf("m1 start_time = %v, want %v", got["m1"], base)
	}
	if !got["m3"].Equal(base.Add(2 * time.Hour)) {
		t.Errorf("m3 start_time = %v, want %v", got["m3"], base.Add(2*time.Hour))
	}
}

// TestTacticalRepo_MatchsOuvrables_ListeVideOuXUIDVide : deux refus DEFENSIFS —
// aucune requete n'est jouee, la map rendue est vide.
func TestTacticalRepo_MatchsOuvrables_ListeVideOuXUIDVide(t *testing.T) {
	pdb := newTacticalTestPlayerDB(t)
	seedTacticalCorpus(t, pdb)
	repo := NewTacticalRepo(pdb)

	got, err := repo.MatchsOuvrables(context.Background(), tacXUIDMoi, nil)
	if err != nil || len(got) != 0 {
		t.Fatalf("liste vide: got=%+v err=%v, want map vide sans erreur", got, err)
	}
	got, err = repo.MatchsOuvrables(context.Background(), "", []string{"m1"})
	if err != nil || len(got) != 0 {
		t.Fatalf("xuid vide: got=%+v err=%v, want map vide sans erreur", got, err)
	}
}

// TestTacticalRepo_MatchsOuvrables_Doublon : un match_id repete dans la demande ne doit
// produire qu'UNE entree.
func TestTacticalRepo_MatchsOuvrables_Doublon(t *testing.T) {
	pdb := newTacticalTestPlayerDB(t)
	seedTacticalCorpus(t, pdb)

	got, err := NewTacticalRepo(pdb).MatchsOuvrables(context.Background(), tacXUIDMoi, []string{"m1", "m1", "m1"})
	if err != nil {
		t.Fatalf("MatchsOuvrables: %v", err)
	}
	if len(got) != 1 {
		t.Fatalf("map = %+v, want exactement 1 entree malgre le doublon", got)
	}
}
