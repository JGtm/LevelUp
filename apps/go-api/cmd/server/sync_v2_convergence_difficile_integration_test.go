//go:build integration

// sync_v2_convergence_difficile_integration_test.go : cas difficiles du rattrapage de l'historique : historique deja complet, deux joueurs dans un meme cycle, matchs neufs meles a des introuvables.
package main

import (
	"levelup/go-api/internal/observability"
	duckdbpkg "levelup/go-api/internal/platform/duckdb"
	"levelup/go-api/internal/sync/historystate"
	"levelup/go-api/internal/sync/syncmeta"
	"sort"
	"testing"
	"time"
)

// ---------------------------------------------------------------------------------------------
// 4. Historique déjà complet : le cas normal ne change pas.
// ---------------------------------------------------------------------------------------------

// Historique de 220 matchs (fin rangée à la 2e passe : la fin vient d'une reprise au marque-page, qui s'efface), puis trois cycles : une page lue, ni marque-page ni clé de fin touchés (valeur identique), aucune
// demande de stats, cadence égale à l'intervalle réglé ; un match neuf est pris par le delta d'une page.
func TestDifficile_HistoriqueDejaComplet_CasNormalInchange(t *testing.T) {
	monde := newDifficileMonde()
	fa := monde.ajouter(difficileA, 0, 220)
	e := newDifficileEnv(t, monde, difficileTous, difficileTous)
	b := e.start(difficileTous, false)
	b.jusquAuBout("rattrapage initial", difficileA)
	b.waitCadence(time.Duration(e.intervalH) * time.Hour)
	cle0 := b.cleDeFin(difficileA)
	if cle0 == "" {
		t.Fatal("clé de fin absente alors que la fin est rangée")
	}

	for i := 1; i <= 3; i++ {
		c := b.cycle()
		out := c.player(difficileA)
		if reads := c.reads[difficileA.xuid]; len(reads) != 1 || reads[0] != 0 {
			t.Errorf("cycle complet %d : lectures %v, attendu une seule page (la première)", i, reads)
		}
		if out.Status != "ok" || out.HistoryPending || out.MatchesInserted != 0 || c.statsTotal() != 0 {
			t.Errorf("cycle complet %d : %+v, %d demandes de stats (post-sync : %d) ; attendu rien d'inséré, rien en attente", i, out, c.statsTotal(), monde.postSyncStats())
		}
		if got := b.cleDeFin(difficileA); got != cle0 {
			t.Errorf("cycle complet %d : clé de fin %q, attendu inchangée %q", i, got, cle0)
		}
		if _, cur := b.historyState(difficileA); cur != nil {
			t.Errorf("cycle complet %d : marque-page apparu %+v", i, *cur)
		}
		b.waitCadence(time.Duration(e.intervalH) * time.Hour)
	}
	b.assertConverged("historique complet", map[difficileJoueur]*fakeHalo{difficileA: fa}, nil)

	fresh := fa.addOnTop(1)
	c := b.cycle()
	// D-23 : le delta s'arrête après une page de connus d'affilée ; un match neuf en tête fait lire deux pages.
	if reads, out := c.reads[difficileA.xuid], c.player(difficileA); len(reads) != 2 || reads[0] != 0 || reads[1] != convergencePageSize ||
		out.MatchesInserted != 1 || !b.registryIDs()[fresh[0]] || out.HistoryPending {
		t.Errorf("delta d'un match neuf : lectures %v, issue %+v, attendu deux pages et un match inséré", reads, out)
	}
	if got := b.cleDeFin(difficileA); got != cle0 {
		t.Errorf("après le delta : clé de fin %q, attendu inchangée %q", got, cle0)
	}
	b.waitCadence(time.Duration(e.intervalH) * time.Hour)
}

// cleDeFin rend la valeur brute de la clé de fin d'historique du joueur ("" si absente).
func (b *difficileBoot) cleDeFin(j difficileJoueur) string {
	b.env.t.Helper()
	db, release, err := duckdbpkg.OpenReadForQuery(b.env.playerPath(j))
	if err != nil {
		b.env.t.Fatalf("lecture base joueur: %v", err)
	}
	defer release()
	v, _, err := syncmeta.Get(b.ctx, db, historystate.Key)
	if err != nil {
		b.env.t.Fatalf("lecture de la clé de fin: %v", err)
	}
	return v
}

// ---------------------------------------------------------------------------------------------
// 5. Deux joueurs suivis dans le même cycle.
// ---------------------------------------------------------------------------------------------

// A (120 matchs) est complet ; B (500 matchs : trois passes) arrive en rattrapage, 10 matchs communs déjà rangés par A et
// 3 matchs communs neufs en haut des deux listes. Dans le même cycle : A ne lit qu'une page, B rattrape ; la
// cadence reste à 10 minutes jusqu'à la fin de B ; aucun match de l'un n'est attribué à l'autre ; un match
// commun neuf n'est demandé qu'une fois et compte pour les deux.
func TestDifficile_DeuxJoueurs_UnCompletUnEnRattrapage(t *testing.T) {
	tous := []difficileJoueur{difficileA, difficileB}
	monde := newDifficileMonde()
	fa := monde.ajouter(difficileA, 0, 120)
	fb := monde.ajouter(difficileB, difficileFirstIDB, 500)
	monde.partagerAu(difficileA, difficileB, 50, 300, 10)
	e := newDifficileEnv(t, monde, tous, []difficileJoueur{difficileA})
	b := e.start(tous, false)

	// A seul : fin d'historique rangée dès la première passe (120 < 200).
	c1 := b.jusquAuBout("A seul", difficileA)
	if len(c1) != 1 {
		t.Fatalf("A (120 matchs) : fin rangée en %d cycles, attendu 1", len(c1))
	}
	b.waitCadence(time.Duration(e.intervalH) * time.Hour)
	if n := len(b.registryIDs()); n != 120 {
		t.Fatalf("après A seul : %d matchs au registre, attendu 120", n)
	}

	// B rejoint le suivi ; trois matchs communs neufs apparaissent en haut des deux listes.
	e.setActive(difficileA, difficileB)
	communs := monde.partagerEnHaut(difficileA, difficileB, 3)
	c2 := b.cycle()
	outA, outB := c2.player(difficileA), c2.player(difficileB)
	// D-23 : les trois matchs communs neufs en tête de A font lire deux pages (la seconde ne porte que des connus).
	if reads := c2.reads[difficileA.xuid]; len(reads) != 2 || reads[0] != 0 || reads[1] != convergencePageSize {
		t.Errorf("cycle partagé : A a lu les pages %v, attendu les deux premières", reads)
	}
	if outA.Status != "ok" || outA.HistoryPending {
		t.Errorf("cycle partagé : A %+v, attendu complet", outA)
	}
	if outB.Status != "ok" || !outB.HistoryPending {
		t.Errorf("cycle partagé : B %+v, attendu en rattrapage", outB)
	}
	if got := len(b.registryIDs()); got != 120+200 {
		t.Errorf("cycle partagé : %d matchs au registre, attendu %d (budget de 200 de B, communs neufs compris)", got, 120+200)
	}
	for _, id := range communs {
		if n := c2.stats[id]; n != 1 {
			t.Errorf("match commun %s demandé %d fois dans le cycle, attendu une", id, n)
		}
	}
	if n := c2.statsTotal(); n != 200 {
		t.Errorf("cycle partagé : %d demandes de stats, attendu 200 (aucune pour A seul, communs une fois)", n)
	}
	parts := b.participantsOf()
	for _, id := range communs {
		if !parts[id][difficileA.xuid] || !parts[id][difficileB.xuid] {
			t.Errorf("match commun %s : participants %v, attendu A et B", id, parts[id])
		}
	}
	if got := outA.MatchesInserted + outB.MatchesInserted; got < 200 {
		t.Errorf("matchs insérés déclarés %d (A %d, B %d), attendu au moins 200", got, outA.MatchesInserted, outB.MatchesInserted)
	}
	t.Logf("cycle partagé : A lu %v, B lu %d pages ; insérés A=%d B=%d ; stats %d", c2.reads[difficileA.xuid], len(c2.reads[difficileB.xuid]), outA.MatchesInserted, outB.MatchesInserted, c2.statsTotal())
	b.waitCadence(catchUpCadence)
	if complete, cur := b.historyState(difficileA); !complete || cur != nil {
		t.Errorf("A : complet=%v marque-page=%v", complete, cur)
	}
	if _, cur := b.historyState(difficileB); cur == nil {
		t.Error("B : aucun marque-page rangé après sa première passe")
	}

	// Cycles suivants : B rattrape, A ne lit toujours qu'une page ; la cadence reste à 10 minutes.
	cycles := []difficileCycle{c2}
	for len(cycles) < difficileMaxCycles {
		c := b.cycle()
		cycles = append(cycles, c)
		if !c.ran || c.err != nil {
			t.Fatalf("cycle %d : %v", len(cycles), c.err)
		}
		if reads := c.reads[difficileA.xuid]; len(reads) != 1 || reads[0] != 0 {
			t.Errorf("cycle %d : A a lu %v, attendu une page", len(cycles), reads)
		}
		if out := c.player(difficileA); out.HistoryPending || out.MatchesInserted != 0 {
			t.Errorf("cycle %d : A %+v, attendu complet et rien d'inséré", len(cycles), out)
		}
		b.assertNoDuplicate("deux joueurs")
		completeB, _ := b.historyState(difficileB)
		out := c.player(difficileB)
		if completeB == out.HistoryPending {
			t.Fatalf("cycle %d : fin de B rangée=%v mais HistoryPending=%v", len(cycles), completeB, out.HistoryPending)
		}
		if completeB {
			b.waitCadence(time.Duration(e.intervalH) * time.Hour)
			break
		}
		b.waitCadence(catchUpCadence)
	}

	listes := map[difficileJoueur]*fakeHalo{difficileA: fa, difficileB: fb}
	b.assertConverged("deux joueurs", listes, nil)
	// Aucun match de l'un n'est attribué à l'autre : A figure aux participants des seuls matchs de sa liste.
	idsA, idsB := map[string]bool{}, map[string]bool{}
	for _, id := range difficileIDs(fa) {
		idsA[id] = true
	}
	for _, id := range difficileIDs(fb) {
		idsB[id] = true
	}
	for id, xs := range b.participantsOf() {
		if xs[difficileA.xuid] != idsA[id] || xs[difficileB.xuid] != idsB[id] {
			t.Errorf("match %s : participants %v, attendu A=%v B=%v", id, xs, idsA[id], idsB[id])
		}
	}
	for id, n := range difficileCumul(cycles) {
		if n != 1 {
			t.Errorf("match %s demandé %d fois, attendu une", id, n)
		}
	}
	t.Logf("deux joueurs : B rattrapé en %d cycles, lectures de pages de B par cycle %v, de A %v", len(cycles), difficileReads(cycles, difficileB), difficileReads(cycles, difficileA))
	_ = c1
}

// ---------------------------------------------------------------------------------------------
// 6. Matchs neufs du dessus, mêlés à des introuvables.
// ---------------------------------------------------------------------------------------------

// 500 matchs dont 10 introuvables (404) dispersés, dont trois autour de la limite de la 1re passe et trois
// autour de celle de la 2e ; 10 matchs neufs en haut avant chacune des passes 2 et 3 (trois passes en tout),
// dont 5 introuvables chacun : 20 introuvables au total. Rien n'est perdu (tout ce qui existe est rangé),
// la reprise est exacte (page du haut, puis la position du marque-page décalée du nombre exact de neufs,
// jamais un repli « depuis 0 »).
func TestDifficile_MatchsNeufsEtIntrouvables_RepriseExacte(t *testing.T) {
	monde := newDifficileMonde()
	fa := monde.ajouter(difficileA, 0, 500)
	absent := map[string]bool{}
	for _, pos := range []int{3, 77, 199, 200, 201, 333, 399, 400, 401, 470} {
		id := fa.idAt(pos)
		monde.setStatus(id, 404)
		absent[id] = true
	}
	e := newDifficileEnv(t, monde, difficileTous, difficileTous)
	b := e.start(difficileTous, false)
	avantRepli := observability.LoadCounter(difficileCompteurRepriseDepuisZero)
	listes := map[difficileJoueur]*fakeHalo{difficileA: fa}

	var cycles []difficileCycle
	var complete bool
	for n := 1; n <= difficileMaxCycles && !complete; n++ {
		var fresh []string
		if n >= 2 && n <= 3 {
			fresh = fa.addOnTop(10)
			for i := 0; i < 10; i += 2 { // le plus récent d'abord : le haut de liste est introuvable
				monde.setStatus(fresh[i], 404)
				absent[fresh[i]] = true
			}
		}
		_, curAvant := b.historyState(difficileA)
		var avant difficileCounts = b.assertNoDuplicate("avant le cycle")

		c := b.cycle()
		cycles = append(cycles, c)
		out := c.player(difficileA)
		if !c.ran || c.err != nil || out.Status != "ok" {
			t.Fatalf("cycle %d : %v %+v", n, c.err, out)
		}
		apres := b.assertNoDuplicate("après le cycle")
		reg := b.registryIDs()
		for _, id := range fresh {
			if !absent[id] && !reg[id] {
				t.Fatalf("cycle %d : le match neuf %s n'a pas été pris", n, id)
			}
		}
		reads := c.reads[difficileA.xuid]
		if curAvant != nil {
			resume := curAvant.Offset + len(fresh) - 1
			if len(reads) < 2 || reads[0] != 0 || reads[1] != resume {
				t.Fatalf("cycle %d : lectures %v, attendu la page 0 puis la reprise exacte à %d (marque-page %+v, %d neufs)", n, reads, resume, *curAvant, len(fresh))
			}
			for _, start := range reads[2:] {
				if start < resume {
					t.Fatalf("cycle %d : lecture à %d, en deçà de la reprise %d : rebalayage", n, start, resume)
				}
			}
		}
		var cc bool
		cc, _ = b.historyState(difficileA)
		complete = cc
		if cc == out.HistoryPending {
			t.Fatalf("cycle %d : fin rangée=%v mais HistoryPending=%v", n, cc, out.HistoryPending)
		}
		b.assertCursor("cycle", difficileA, fa, 0, absent)
		t.Logf("cycle %d : %d neufs (dont %d introuvables) ; +%d au registre ; %d pages lues %v ; %d demandes de stats", n, len(fresh), (len(fresh)+1)/2, apres.registryDistinct-avant.registryDistinct, len(reads), reads, c.statsTotal())
	}
	if !complete {
		t.Fatalf("fin non rangée en %d cycles", difficileMaxCycles)
	}
	if len(absent) != 20 {
		t.Fatalf("scénario : %d introuvables, attendu 20", len(absent))
	}
	if len(cycles) != 3 {
		t.Fatalf("fin rangée en %d passes, attendu 3 (500 + 20 neufs, budget de 200)", len(cycles))
	}
	b.assertConverged("matchs neufs et introuvables", listes, absent)
	if apres := observability.LoadCounter(difficileCompteurRepriseDepuisZero); apres != avantRepli {
		t.Errorf("repli « reprise depuis 0 » compté %d fois : la reprise n'était pas exacte", apres-avantRepli)
	}
	demandes := difficileCumul(cycles)
	var repetes []string
	for id, n := range demandes {
		if !absent[id] && n != 1 {
			t.Errorf("match %s existant demandé %d fois, attendu une", id, n)
		}
		if absent[id] && n != 1 {
			repetes = append(repetes, id)
		}
	}
	sort.Strings(repetes)
	t.Logf("introuvables demandés plus d'une fois pendant le rattrapage : %d sur %d %v", len(repetes), len(absent), repetes)

	// Après la fin (D-23, D-24) : les introuvables des premières entrées sont à ne plus demander, donc neutres pour
	// le delta (ni comptés dans les 25 connus d'affilée, ni remise à zéro) : ici 11 introuvables dans les 25 premières
	// entrées (dix des matchs neufs, un de l'historique d'origine), 36 entrées pour atteindre 25 connus, soit deux pages, sans aucune demande de stats ni match annoncé inséré.
	b.waitCadence(time.Duration(e.intervalH) * time.Hour)
	for i := 1; i <= 2; i++ {
		c := b.cycle()
		out := c.player(difficileA)
		t.Logf("après la fin, cycle %d : lectures %v, %d demandes de stats, issue %+v", i, c.reads[difficileA.xuid], c.statsTotal(), out)
		if n := c.statsTotal(); n != 0 {
			t.Errorf("après la fin, cycle %d : %d demandes de stats, attendu aucune (les introuvables ne sont plus demandés)", i, n)
		}
		if out.MatchesInserted != 0 {
			t.Errorf("après la fin, cycle %d : %d matchs annoncés insérés, attendu 0", i, out.MatchesInserted)
		}
		if reads := c.reads[difficileA.xuid]; len(reads) != 2 {
			t.Errorf("après la fin, cycle %d : %d pages lues %v, attendu deux (jamais un rebalayage)", i, len(reads), reads)
		}
	}
}
