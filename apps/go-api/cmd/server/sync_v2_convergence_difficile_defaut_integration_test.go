//go:build integration

// sync_v2_convergence_difficile_defaut_integration_test.go : défaut constaté par les cas difficiles, test gardé
// sauté tant que le défaut n est pas corrigé (le correctif relève du code de production, hors de ce lot).
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

// Coupure brutale en plein vidage de la file (140 journaux d'écriture restent), redémarrage, cycle lancé
// aussitôt. Deux contrats, deux constats (mêmes causes : le rejeu du journal par le worker et la découverte du
// cycle se disputent la base joueur dans le même processus) :
//   - le cycle rend la main en moins de trois minutes. Constaté : les journaux rejoués au démarrage échouent sur
//     « Can't open a connection to same database file with a different configuration » (la découverte du cycle tient
//     la base joueur en lecture), erreur que le worker ne reprend pas (hors de sa liste d'erreurs transitoires, alors
//     que le message dit « réessayer ») ; le journal reste jusqu'à la reprise périodique (10 minutes) et le vidage du
//     cycle attend le journal ENTIER (soixante secondes plus quatre par match soumis : 14 minutes 20 pour 200 matchs) ;
//   - aucun cycle ne décide « historique complet » sur une lecture en échec. Constaté : après la reprise périodique
//     du journal, la lecture de l'état de la découverte rend « sql: database is closed » (poignée empruntée au cache,
//     fermée par le worker qui rejoue), `historystate.Complete` rend alors « complet » : le cycle ne parcourt rien
//     (0 match inconnu), `HistoryPending` est faux, la cadence repasse à l'intervalle réglé alors que la fin n'est
//     pas vue.
func TestDifficile_Interruption_CycleApresRedemarrageSansBlocage(t *testing.T) {
	t.Skip("défaut: après un redémarrage avec journal d'écriture en attente, un cycle lancé aussitôt (1) reste bloqué jusqu'à 14 minutes " +
		"(rejeu du journal en échec « same database file with a different configuration », non repris par le worker, vidage du cycle qui " +
		"attend tout le journal) et (2) peut lire l'état de l'historique sur une poignée fermée (« sql: database is closed »), que " +
		"historystate.Complete traduit en « complet » : passe de rattrapage sautée, cadence de 10 minutes perdue.")
	monde := newDifficileMonde()
	monde.ajouter(difficileA, 0, 1000)
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
	c1 := b2.cycle() // rend la main après le délai du vidage : on attend la fin pour juger la suite
	if d := time.Since(debut); d > 3*time.Minute {
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
}
