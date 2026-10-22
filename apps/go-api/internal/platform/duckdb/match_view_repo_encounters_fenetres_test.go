package duckdb

// match_view_repo_encounters_fenetres_test.go — LES STATISTIQUES DE RENCONTRE DE LA VUE DE MATCH
// SONT PAYÉES AUX MATCHS DU JOUEUR (ADR 0036 I2).
//
// Q23b compte les frags échangés entre le joueur et chaque rencontre sur tout l'historique : sa
// lecture du kill-feed doit porter la liste des matchs du joueur en prédicat constant sous la
// fenêtre `_latest`. Sans elle les nombres sont les mêmes, mais la fenêtre balaie tout le journal
// des morts (plusieurs secondes sur la base de production) : seul le nombre de lignes vues le montre.

import (
	"context"
	"testing"
)

// TestMatchViewRepo_EncounterStats_FenetreBornee : quatre matchs à moi sur dix ; la fenêtre du
// kill-feed ne voit que les lignes de ces quatre-là, et les frags échangés sont ceux du corpus.
func TestMatchViewRepo_EncounterStats_FenetreBornee(t *testing.T) {
	b := newBaseNotee(t)
	ids := seedFenetresTactiques(t, b.pdb, matchsDuJoueurFenetres)
	seedMatchsDesAutres(t, b.pdb, matchsDesAutresFenetres)
	b.carnet.vider()

	got, err := NewMatchViewRepo(b.pdb, tacXUIDMoi).GetMatchEncounterStats(context.Background(), ids[0], tacXUIDMoi)
	if err != nil {
		t.Fatalf("GetMatchEncounterStats: %v", err)
	}
	// Par match : moi → adv aux morts d'indice pair (2), adv → moi à l'indice impair (1).
	if len(got) != 1 || got[0].XUID != tacXUIDAdv ||
		got[0].KillsDealt != 2*matchsDuJoueurFenetres || got[0].DeathsSuffered != matchsDuJoueurFenetres {
		t.Fatalf("rencontres = %+v, want l'adversaire seul, %d infligés / %d subis",
			got, 2*matchsDuJoueurFenetres, matchsDuJoueurFenetres)
	}
	exigerFenetresBornees(t, b, "GetMatchEncounterStats", matchsDuJoueurFenetres*mortsParMatchFenetres, 1)
}
