package duckdb

// home_repo_assisted_frags_fenetres_test.go — LES FRAGS ASSISTÉS DES TUILES DE L'ACCUEIL SONT
// PAYÉS AU LOT DE MATCHS DEMANDÉ (ADR 0036 I2).
//
// Q26l lit deux fois la vue `_latest` du journal des morts ; chaque lecture doit porter le lot en
// prédicat constant sous sa fenêtre. Une semi-jointure seule (`IN (SELECT …)`) ne change aucun
// nombre et laisse la fenêtre balayer tout le journal : seul le nombre de lignes vues le montre.

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
	exigerFenetresBornees(t, b, "LoadMatchAssistedFrags", 2*mortsParMatchFenetres, 1)
}
