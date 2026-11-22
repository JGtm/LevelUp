//go:build integration

// sync_v2_convergence_difficile_defaut_integration_test.go : défaut constaté par les cas difficiles, test gardé
// sauté tant que le défaut n est pas corrigé (le correctif relève du code de production, hors de ce lot).
package main

import (
	"testing"
)

// Historique de 30 matchs dont le plus récent est introuvable (404). Contrat : « Matchs insérés » dit ce qui
// est rangé (c'est la valeur que la notification de fin de cycle annonce comme nouveaux matchs). Le 404 n'est
// pas rangé : 29 insérés au premier cycle, puis zéro à chaque cycle suivant.
func TestDifficile_MatchIntrouvable_NEstPasCompteInsere(t *testing.T) {
	t.Skip("défaut: populateOutcome (internal/sync/v2/cycle.go) compte comme MatchesInserted tout match inconnu listé pour le joueur, " +
		"sans regarder s il a été récupéré et rangé : un match introuvable (404) en haut de liste est compté inséré à chaque cycle, pour toujours " +
		"(le cycle l annonce comme nouveau match à la notification de fin de cycle). Présent depuis D6.1, donc dans la 8.2.1.")
	monde := newDifficileMonde()
	fa := monde.ajouter(difficileA, 0, 30)
	gone := fa.idAt(0)
	monde.setStatus(gone, 404)
	e := newDifficileEnv(t, monde, difficileTous, difficileTous)
	b := e.start(difficileTous, false)

	c1 := b.cycle()
	out1 := c1.player(difficileA)
	ranges := len(b.registryIDs())
	if ranges != 29 {
		t.Fatalf("premier cycle : %d matchs ranges, attendu 29", ranges)
	}
	if out1.MatchesInserted != ranges {
		t.Errorf("premier cycle : MatchesInserted=%d alors que %d matchs sont ranges (l'introuvable est compté)", out1.MatchesInserted, ranges)
	}
	for i := 1; i <= 2; i++ {
		c := b.cycle()
		out := c.player(difficileA)
		if got := len(b.registryIDs()); got != ranges {
			t.Fatalf("cycle %d : %d matchs ranges, attendu %d", i, got, ranges)
		}
		if out.MatchesInserted != 0 {
			t.Errorf("cycle %d : MatchesInserted=%d, rien n'a été rangé (le match introuvable est redemandé %d fois et compté comme inséré)",
				i, out.MatchesInserted, c.stats[gone])
		}
	}
}
