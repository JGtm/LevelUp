//go:build integration

// sync_v2_convergence_difficile_defaut_integration_test.go : défauts constatés par les cas difficiles. Un test
// reste sauté tant que son défaut n'est pas corrigé ; celui du cycle après redémarrage (D-25) est corrigé.
package main

import (
	"context"
	"testing"
	"time"
)

// Historique de 30 matchs dont le plus récent est introuvable (404). Contrat : « Matchs insérés » dit ce qui
// est rangé (c'est la valeur que la notification de fin de cycle annonce comme nouveaux matchs). Le 404 n'est
// pas rangé : 29 insérés au premier cycle, puis zéro à chaque cycle suivant.
func TestDifficile_MatchIntrouvable_NEstPasCompteInsere(t *testing.T) {
	t.Skip("défaut: populateOutcome (internal/sync/v2/cycle.go) compte comme MatchesInserted tout match inconnu listé pour le joueur, " +
		"sans regarder s il a été récupéré et rangé : un match introuvable (404) en haut de liste est compté inséré à chaque cycle, pour toujours " +
		"(le cycle l annonce comme nouveau match à la notification de fin de cycle). Présent depuis D6.1, donc dans la 8.2.1 ; depuis D-23 les introuvables des 25 premières entrées sont recomptés à chaque cycle (21 constatés après une fin de rattrapage).")
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

// Coupure brutale en plein vidage de la file (140 journaux d'écriture restent), redémarrage, cycle lancé
// aussitôt. Deux contrats (D-25 ; avant le correctif : 14 minutes 28 de blocage, puis, une fois sur deux, un
// état « complet » lu sur une poignée fermée) :
//   - le cycle rend la main en moins de trois minutes : la découverte lit la base joueur par le handle que
//     partage le worker qui rejoue le journal (même clé de cache, même configuration), le rejeu ne bute plus
//     sur « different configuration », et une attente sans acquittement rendrait la main après 60 s ;
//   - aucun cycle ne décide « historique complet » sur une lecture en échec : après la reprise périodique du
//     journal, le cycle suivant fait une passe de rattrapage tant que la fin n'est pas rangée.
//
// À la fin : le journal est vide, aucun match n'est en double ni perdu, et la fin finit rangée.
func TestDifficile_Interruption_CycleApresRedemarrageSansBlocage(t *testing.T) {
	monde := newDifficileMonde()
	fa := monde.ajouter(difficileA, 0, 1000)
	e := newDifficileEnv(t, monde, difficileTous, difficileTous)
	b := e.start(difficileTous, true)
	b.cycle()

	b.gate.arm(60)
	cctx, annuler := context.WithCancel(b.ctx)
	fini := make(chan difficileCycle, 1)
	go func() { fini <- b.cycleCtx(cctx) }()
	b.gate.waitBlocked(t, 2*time.Minute)
	annuler()
	<-fini
	b.kill(true)
	if n, _ := e.walState(); n < 100 {
		t.Fatalf("scénario : %d journaux restent, attendu une centaine", n)
	}

	b2 := e.start(difficileTous, false)
	debut := time.Now()
	c1 := b2.cycle()
	d := time.Since(debut)
	t.Logf("cycle lancé juste après le redémarrage : rendu après %v", d.Round(time.Millisecond))
	if d > 3*time.Minute {
		t.Errorf("le cycle lancé juste après le redémarrage a rendu la main après %v, attendu moins de 3 minutes", d.Round(time.Second))
	}
	if !c1.ran || c1.err != nil {
		t.Fatalf("premier cycle après le redémarrage : %v", c1.err)
	}

	// La reprise périodique du journal passe entre deux cycles (toutes les 10 minutes en production) ; le cycle
	// suivant ne doit pas lire un état d'historique incohérent.
	b2.recoveryTick()
	c2 := b2.cycle()
	complete, _ := b2.historyState(difficileA)
	if out := c2.player(difficileA); out.HistoryPending == complete || (!complete && out.MatchesUnknown == 0) {
		t.Errorf("cycle 2 : fin rangée=%v, HistoryPending=%v, %d matchs inconnus listés ; attendu une passe de rattrapage tant que la fin n'est pas rangée",
			complete, out.HistoryPending, out.MatchesUnknown)
	}

	// Le journal est rejoué, rien n'est en double, et la fin finit rangée.
	e.assertWALVide("après le redémarrage")
	b2.assertNoDuplicate("après le redémarrage")
	if !complete {
		b2.jusquAuBout("reprise après le redémarrage", difficileA)
	}
	b2.assertConverged("reprise après le redémarrage", map[difficileJoueur]*fakeHalo{difficileA: fa}, nil)
	e.assertWALVide("fin du rattrapage")
}
