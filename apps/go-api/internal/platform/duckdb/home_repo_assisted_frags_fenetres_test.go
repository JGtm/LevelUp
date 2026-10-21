package duckdb

// home_repo_assisted_frags_fenetres_test.go — LES FRAGS ASSISTÉS DES TUILES DE L'ACCUEIL SONT
// PAYÉS AU LOT DE MATCHS DEMANDÉ (ADR 0036 I2).
//
// Q26l lit les résumés du journal (tête S0, frags S1, appuis S2) ; chaque lecture porte le lot en
// prédicat constant sur son `match_id`. Les vues filles (S1, S2) lues sous une liste font voir à la
// fenêtre de leur tête toutes les têtes (une ligne par match, ADR 0039 décision 7), jamais les lignes
// des filles : seul le nombre de lignes vues le montre.

import (
	"context"
	"testing"
)

// TestHomeRepo_LoadMatchAssistedFrags_FenetresBornees : deux matchs demandés sur dix ; chaque
// fenêtre des requêtes de Q26l ne voit que les lignes de ces deux-là.
func TestHomeRepo_LoadMatchAssistedFrags_FenetresBornees(t *testing.T) {
	b := newBaseNotee(t)
	ids := seedFenetresTactiques(t, b.pdb, 10)
	// Films porteurs de l'assistance : sans quoi `porteuses` est vide et le test ne compte rien.
	tacExec(t, b.pdb, `UPDATE match_kill_events SET assist_known = TRUE`)
	ecrireResumes(t, b.pdb.Shared.SQLDb()) // les résumés suivent le journal modifié
	b.carnet.vider()

	got, err := NewHomeRepo(b.pdb).LoadMatchAssistedFrags(context.Background(), ids[:2])
	if err != nil {
		t.Fatalf("LoadMatchAssistedFrags: %v", err)
	}
	if len(got) != 2 {
		t.Fatalf("matchs servis = %d (%+v), want 2", len(got), got)
	}
	for _, id := range ids[:2] {
		// Par match, MOI tue aux morts d'indice pair : 2 frags sur mortsParMatchFenetres = 3.
		if got[id].FragsFilm != 2 {
			t.Fatalf("%s : frags lus = %d, want 2", id, got[id].FragsFilm)
		}
	}
	// Vues filles (S1, S2) sous une liste : la fenêtre de la tête voit toutes les têtes (une par match),
	// jamais les lignes des filles (ADR 0039 décision 7).
	exigerFenetresBornees(t, b, "LoadMatchAssistedFrags", tetesDuJournal(t, b.brute), 1)
}
