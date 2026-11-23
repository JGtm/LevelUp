//go:build integration

// sync_v2_convergence_429_integration_test.go : la lecture de la liste de matchs sous 429 (site hébergé, jetons
// mis en commun). Banc : sync_v2_convergence_difficile_env_integration_test.go.
package main

import (
	"fmt"
	"testing"
	"time"

	"levelup/go-api/internal/observability"
)

// difficileJoueurN fabrique le joueur de rang n (1 à 99) : gamertag et xuid distincts de A et de B.
func difficileJoueurN(n int) difficileJoueur {
	return difficileJoueur{gamertag: fmt.Sprintf("ConvergenceJoueur%02d", n), xuid: fmt.Sprintf("25332748000010%02d", n)}
}

// Six joueurs suivis, dont cinq en rattrapage : l'API refuse (429) toute page de la liste au-delà des deux
// premières pendant trois cycles, puis redevient saine. À chaque cycle, chaque joueur reçoit ses nouveaux matchs
// (le haut de sa liste), aucun n'est en échec, aucun marque-page ne bouge ; ensuite le rattrapage reprend et
// finit, sans trou ni doublon et sans repli « reprise depuis 0 ».
func TestDifficile_SixJoueursSous429_NouveauxMatchsGardesEtRattrapageReprend(t *testing.T) {
	const (
		cyclesRefus = 3
		parCycle    = 2 // nouveaux matchs par joueur et par cycle
	)
	catching := []difficileJoueur{difficileJoueurN(1), difficileJoueurN(2), difficileJoueurN(3), difficileJoueurN(4), difficileJoueurN(5)}
	tous := append([]difficileJoueur{difficileA}, catching...)

	monde := newDifficileMonde()
	listes := map[difficileJoueur]*fakeHalo{difficileA: monde.ajouter(difficileA, 0, 30)}
	for i, j := range catching {
		listes[j] = monde.ajouter(j, (i+1)*4000, 60)
	}
	e := newDifficileEnv(t, monde, tous, []difficileJoueur{difficileA})
	b := e.start(tous, false)
	avantRepli := observability.LoadCounter(difficileCompteurRepriseDepuisZero)

	// A seul rattrape sa liste (30 matchs : une passe), puis les cinq autres rejoignent le suivi.
	b.jusquAuBout("A seul", difficileA)
	b.waitCadence(time.Duration(e.intervalH) * time.Hour)
	e.setActive(tous...)

	// Premier cycle à six, API saine : chaque joueur en rattrapage range sa part du budget (200 / 5) et son marque-page.
	c0 := b.cycle()
	if !c0.ran || c0.err != nil {
		t.Fatalf("cycle initial : %v", c0.err)
	}
	avant := map[difficileJoueur]int{}
	for _, j := range catching {
		out := c0.player(j)
		complete, cur := b.historyState(j)
		if out.Status != "ok" || !out.HistoryPending || complete || cur == nil {
			t.Fatalf("cycle initial, %s : %+v complet=%v marque-page=%v, attendu en rattrapage avec un marque-page", j.gamertag, out, complete, cur)
		}
		avant[j] = cur.Offset
	}
	b.waitCadence(catchUpCadence)

	// L'API refuse les pages profondes : les deux premières pages (le haut de la liste) restent servies.
	monde.setRefuserPage(func(_ string, start int) int {
		if start > 25 {
			return 429
		}
		return 0
	})
	for n := 1; n <= cyclesRefus; n++ {
		fresh := map[difficileJoueur][]string{}
		for _, j := range tous {
			fresh[j] = listes[j].addOnTop(parCycle)
		}
		c := b.cycle()
		if !c.ran || c.err != nil {
			t.Fatalf("cycle %d sous 429 : orchestrateur non joué ou en échec (%v)", n, c.err)
		}
		if got := monde.pagesRefuseesDepuisRaz(); got == 0 {
			t.Fatalf("cycle %d : aucune page refusée, le scénario ne mord pas", n)
		}
		reg := b.registryIDs()
		for _, j := range tous {
			out := c.player(j)
			if out.Status != "ok" {
				t.Errorf("cycle %d, %s : issue %q (%s), attendu un joueur qui n'est pas en échec", n, j.gamertag, out.Status, out.FirstError)
			}
			for _, id := range fresh[j] {
				if !reg[id] {
					t.Errorf("cycle %d, %s : le nouveau match %s n'a pas été récupéré", n, j.gamertag, id)
				}
			}
			if out.MatchesInserted != parCycle {
				t.Errorf("cycle %d, %s : %d matchs insérés, attendu les %d nouveaux", n, j.gamertag, out.MatchesInserted, parCycle)
			}
			if j == difficileA {
				continue
			}
			complete, cur := b.historyState(j)
			if !out.HistoryPending || complete || cur == nil || cur.Offset != avant[j] {
				t.Errorf("cycle %d, %s : en attente=%v complet=%v marque-page=%v, attendu en rattrapage, marque-page inchangé à %d", n, j.gamertag, out.HistoryPending, complete, cur, avant[j])
			}
		}
		b.assertNoDuplicate(fmt.Sprintf("cycle %d sous 429", n))
		t.Logf("cycle %d sous 429 : %d lectures de page refusées, lectures par joueur %v", n, monde.pagesRefuseesDepuisRaz(), difficileReads([]difficileCycle{c}, catching[0]))
		b.waitCadence(catchUpCadence)
	}

	// L'API redevient saine : le rattrapage reprend et finit pour les cinq.
	monde.setRefuserPage(nil)
	b.jusquAuBout("API saine", tous...)
	b.assertConverged("six joueurs sous 429", listes, nil)
	if apres := observability.LoadCounter(difficileCompteurRepriseDepuisZero); apres != avantRepli {
		t.Errorf("repli « reprise depuis 0 » compté %d fois : la reprise n'était pas exacte", apres-avantRepli)
	}
}
