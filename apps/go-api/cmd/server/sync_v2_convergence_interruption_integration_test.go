//go:build integration

// sync_v2_convergence_interruption_integration_test.go : cas difficiles du rattrapage de l'historique : arret du processus en plein cycle puis redemarrage sur les memes bases (pendant la recuperation, pendant la persistance, entre deux passes).
package main

import (
	"context"
	"fmt"
	"levelup/go-api/internal/observability"
	"testing"
	"time"
)

// ---------------------------------------------------------------------------------------------
// 1. Arrêt brutal pendant la récupération.
// ---------------------------------------------------------------------------------------------

// Le contexte du cycle est annulé au milieu de la phase de récupération d'une passe de rattrapage (à la
// 100e des 200 demandes de stats) ; le serveur s'arrête par sa séquence d'arrêt (runShutdown) ; un démarrage
// NEUF sur les mêmes bases reprend : 1 000 matchs rangés, aucun doublon, aucun trou, fin rangée, marque-page
// effacé. Le cycle annulé a déjà soumis à la file les matchs récupérés : ils finissent en arrière-plan ou
// restent au journal d'écriture, rejoué au démarrage.
func TestDifficile_Interruption_ArretPendantRecuperation(t *testing.T) {
	monde := newDifficileMonde()
	fa := monde.ajouter(difficileA, 0, 1000)
	e := newDifficileEnv(t, monde, difficileTous, difficileTous)
	b := e.start(difficileTous, false)

	c1 := b.cycle()
	if out := c1.player(difficileA); out.Status != "ok" || out.MatchesInserted != 200 || !out.HistoryPending {
		t.Fatalf("cycle 1 : %+v, attendu 200 matchs insérés et un rattrapage en cours", out)
	}
	_, cur1 := b.historyState(difficileA)
	if cur1 == nil || cur1.Offset != 200 {
		t.Fatalf("après le cycle 1 : marque-page %v, attendu un décalage de 200", cur1)
	}

	// Cycle 2 : annulé à la 100e demande de stats (sur 200 attendues).
	cctx, annuler := context.WithCancel(b.ctx)
	monde.setOnStats(func(_ string, nth int) {
		if nth == 100 {
			annuler()
		}
	})
	c2 := b.cycleCtx(cctx)
	monde.setOnStats(nil)
	annuler()
	if !c2.ran || c2.err == nil {
		t.Fatalf("cycle 2 : attendu une erreur de cycle (contexte annulé), obtenu ran=%v err=%v", c2.ran, c2.err)
	}
	if c2.statsTotal() != 100 {
		t.Errorf("cycle 2 : %d demandes de stats servies, attendu 100 (les suivantes échouent, contexte annulé)", c2.statsTotal())
	}
	t.Logf("cycle 2 interrompu : %d demandes de stats servies, erreur %q", c2.statsTotal(), c2.err)

	// Arrêt du serveur par sa séquence de production, SANS attendre la file (un vrai arrêt n'attend pas) : le
	// vidage est borné à 5 s, le reste du journal d'écriture est rejoué au démarrage.
	propre := b.shutdownOrderly()
	restants, corrompus := e.walState()
	t.Logf("arrêt en plein cycle : propre=%v, %d journaux d'écriture restent, corrompus=%v", propre, restants, corrompus)

	// Démarrage NEUF sur les mêmes bases ; la file rejoue le journal (RecoverPending du démarrage).
	b2 := e.start(difficileTous, false)
	e.assertWALVide("reprise du journal au démarrage")
	_, cur2 := b2.historyState(difficileA)
	if cur2 == nil || *cur2 != *cur1 {
		t.Fatalf("au redémarrage : marque-page %v, attendu %+v (le cycle annulé n'a rien avancé)", cur2, *cur1)
	}
	avant := b2.assertNoDuplicate("au redémarrage")
	if avant.registryDistinct < 200 || avant.registryDistinct > 300 {
		t.Errorf("au redémarrage : %d matchs rangés, attendu entre 200 (cycle 1) et 300 (cycle 1 + les 100 soumis)", avant.registryDistinct)
	}
	reprise := b2.jusquAuBout("reprise après arrêt en récupération", difficileA)
	b2.assertConverged("reprise après arrêt en récupération", map[difficileJoueur]*fakeHalo{difficileA: fa}, nil)
	e.assertWALVide("reprise après arrêt en récupération")

	// Chaque match est demandé une fois, sauf ceux de la passe annulée dont l'écriture n'a pas abouti.
	cumul := difficileCumul([]difficileCycle{c1, c2}, reprise)
	doubles := 0
	for id, n := range cumul {
		switch {
		case n > 2:
			t.Errorf("match %s demandé %d fois", id, n)
		case n == 2:
			doubles++
		}
	}
	if doubles > c2.statsTotal() {
		t.Errorf("%d matchs demandés deux fois, plus que ceux de la passe annulée (%d)", doubles, c2.statsTotal())
	}
	t.Logf("arrêt en récupération : %d matchs rangés au redémarrage ; reprise en %d cycles, lectures de pages %v ; %d matchs redemandés (passe annulée : %d demandes)",
		avant.registryDistinct, len(reprise), difficileReads(reprise, difficileA), doubles, c2.statsTotal())
}

// ---------------------------------------------------------------------------------------------
// 2. Arrêt brutal pendant la persistance (file en cours d'écriture).
// ---------------------------------------------------------------------------------------------

// Le worker de la file écrit une partie des 200 matchs de la passe puis la porte le retient : le cycle est
// annulé en plein vidage. Trois fins :
//   - arret_ordonne_petit_reste : peu de journaux restent ; la séquence d'arrêt du serveur (runShutdown) vide
//     la file dans son délai (5 s) : arrêt propre, journal vide ;
//   - arret_ordonne_gros_reste : beaucoup de journaux restent, la file ne se vide pas dans le délai de la
//     séquence d'arrêt (les bases se ferment sous le worker) : le journal d'écriture garde le reste ;
//   - coupure_brutale : la mort du processus ; le journal reste sur disque, dont un match déjà écrit mais non
//     acquitté.
//
// Dans les trois cas le démarrage suivant rejoue le journal (RecoverPending) et le test attend la fin du
// rejeu avant le premier cycle (un cycle lancé pendant le rejeu : défaut gardé à part) : 1 000 matchs, aucun
// doublon, aucun trou, journal vide à la fin.
func TestDifficile_Interruption_ArretPendantPersistance(t *testing.T) {
	cas := []struct {
		nom     string
		laisses int // matchs de la passe écrits avant que la porte retienne le suivant
		brutal  bool
	}{
		{"arret_ordonne_petit_reste", 190, false},
		{"arret_ordonne_gros_reste", 60, false},
		{"coupure_brutale", 60, true},
	}
	for _, c := range cas {
		t.Run(c.nom, func(t *testing.T) {
			monde := newDifficileMonde()
			fa := monde.ajouter(difficileA, 0, 1000)
			e := newDifficileEnv(t, monde, difficileTous, difficileTous)
			b := e.start(difficileTous, true)

			c1 := b.cycle()
			if out := c1.player(difficileA); out.Status != "ok" || out.MatchesInserted != 200 {
				t.Fatalf("cycle 1 : %+v", out)
			}
			_, cur1 := b.historyState(difficileA)

			b.gate.arm(c.laisses)
			cctx, annuler := context.WithCancel(b.ctx)
			fini := make(chan difficileCycle, 1)
			go func() { fini <- b.cycleCtx(cctx) }()
			b.gate.waitBlocked(t, 2*time.Minute)
			enAttente, _ := e.walState()
			annuler()
			var c2 difficileCycle
			select {
			case c2 = <-fini:
			case <-time.After(time.Minute):
				t.Fatal("le cycle annulé ne rend pas la main")
			}
			if !c2.ran || c2.err == nil {
				t.Fatalf("cycle 2 : attendu une erreur (annulé en plein vidage de la file), obtenu ran=%v err=%v", c2.ran, c2.err)
			}

			propre := "sans objet"
			if c.brutal {
				b.kill(true)
				if n, _ := e.walState(); n != enAttente {
					t.Errorf("après la coupure : %d journaux, attendu les %d d'avant (rien n'est acquitté après la mort)", n, enAttente)
				}
			} else {
				b.gate.letGo()
				clean := b.shutdownOrderly()
				propre = fmt.Sprint(clean)
				if n, _ := e.walState(); c.laisses == 190 && (!clean || n != 0) {
					t.Errorf("petit reste : arrêt propre=%v, %d journaux en attente ; attendu propre et vide", clean, n)
				}
			}
			restants, _ := e.walState()

			b2 := e.start(difficileTous, false)
			// Le journal rejoué finit AVANT le premier cycle : lire l'état d'une base pendant qu'un worker
			// l'écrit est le défaut gardé dans sync_v2_convergence_difficile_defaut_integration_test.go.
			e.assertWALVide(c.nom + " : rejeu au démarrage")
			ecrits := len(b2.registryIDs())
			b2.assertNoDuplicate(c.nom + " : après le rejeu")
			b2.assertCursor(c.nom+" : après le rejeu", difficileA, fa, 0, nil)
			reprise := b2.jusquAuBout(c.nom, difficileA)
			e.assertWALVide(c.nom)
			b2.assertConverged(c.nom, map[difficileJoueur]*fakeHalo{difficileA: fa}, nil)
			t.Logf("%s : à l'annulation %d journaux en attente (%d matchs écrits dans la passe) ; arrêt propre : %s ; journaux au redémarrage : %d, matchs au registre : %d ; reprise en %d cycles, lectures de pages %v ; marque-page avant l'arrêt %+v",
				c.nom, enAttente, c.laisses, propre, restants, ecrits, len(reprise), difficileReads(reprise, difficileA), *cur1)
		})
	}
}

// ---------------------------------------------------------------------------------------------
// 3. Arrêt entre deux passes.
// ---------------------------------------------------------------------------------------------

// Après deux passes (marque-page à 400), le serveur s'arrête puis redémarre : la passe suivante reprend AU
// marque-page (une page du haut, puis la lecture à la position rangée), sans relire l'historique depuis 0.
func TestDifficile_Interruption_ArretEntreDeuxPasses(t *testing.T) {
	monde := newDifficileMonde()
	fa := monde.ajouter(difficileA, 0, 1000)
	e := newDifficileEnv(t, monde, difficileTous, difficileTous)
	b := e.start(difficileTous, false)
	c1, c2 := b.cycle(), b.cycle()
	_, cur := b.historyState(difficileA)
	if cur == nil || cur.Offset != 400 {
		t.Fatalf("après deux passes : marque-page %v, attendu un décalage de 400", cur)
	}
	if !b.shutdownOrderly() {
		t.Error("arrêt entre deux passes non propre")
	}

	b2 := e.start(difficileTous, false)
	if _, cur2 := b2.historyState(difficileA); cur2 == nil || *cur2 != *cur {
		t.Fatalf("au redémarrage : marque-page %v, attendu %+v", cur2, *cur)
	}
	avant := observability.LoadCounter(difficileCompteurRepriseDepuisZero)
	c3 := b2.cycle()
	reads := c3.reads[difficileA.xuid]
	reprise := cur.Offset - 1 // aucun match neuf : la page lue commence par le dernier match rangé
	if len(reads) < 2 || reads[0] != 0 || reads[1] != reprise {
		t.Fatalf("passe 3 : lectures %v, attendu la page 0 puis la reprise à %d (marque-page %+v)", reads, reprise, *cur)
	}
	for _, start := range reads[2:] {
		if start <= reprise {
			t.Errorf("passe 3 : lecture à %d, en deçà de la reprise %d : relecture depuis le début", start, reprise)
		}
	}
	if apres := observability.LoadCounter(difficileCompteurRepriseDepuisZero); apres != avant {
		t.Errorf("repli « reprise depuis 0 » compté %d fois au redémarrage", apres-avant)
	}
	if out := c3.player(difficileA); out.Status != "ok" || out.MatchesInserted != 200 || !out.HistoryPending {
		t.Errorf("passe 3 : %+v, attendu 200 matchs insérés et un rattrapage en cours", out)
	}
	b2.assertCursor("passe 3", difficileA, fa, 0, nil)

	reste := b2.jusquAuBout("reprise entre deux passes", difficileA)
	b2.assertConverged("reprise entre deux passes", map[difficileJoueur]*fakeHalo{difficileA: fa}, nil)
	cumul := difficileCumul([]difficileCycle{c1, c2, c3}, reste)
	for id, n := range cumul {
		if n != 1 {
			t.Errorf("match %s demandé %d fois, attendu une", id, n)
		}
	}
	t.Logf("arrêt entre deux passes : lectures de pages passe 3 %v (reprise à %d) ; passes suivantes %v (total %d passes après le redémarrage)",
		reads, reprise, difficileReads(reste, difficileA), 1+len(reste))
}
