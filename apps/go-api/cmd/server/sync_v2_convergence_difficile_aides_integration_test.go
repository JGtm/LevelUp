//go:build integration

// sync_v2_convergence_difficile_aides_integration_test.go : aides communes des cas difficiles du rattrapage (preuves
// d absence de doublon, de trou, marque-page coherent ; boucle de cycles jusqu a la fin rangee).
// Le banc : sync_v2_convergence_difficile_env_integration_test.go.
package main

import "time"

const (
	difficileCompteurRepriseDepuisZero = "sync_v2_history_cursor_fallback_total"
	difficileMaxCycles                 = 12
)

var difficileTous = []difficileJoueur{difficileA}

func difficileIDs(f *fakeHalo) []string {
	f.mu.Lock()
	defer f.mu.Unlock()
	return append([]string(nil), f.ids...)
}

// difficileAttendus rend l'ensemble des matchs listés par les joueurs, moins les introuvables.
func difficileAttendus(absent map[string]bool, lists ...*fakeHalo) map[string]bool {
	out := map[string]bool{}
	for _, f := range lists {
		for _, id := range difficileIDs(f) {
			if !absent[id] {
				out[id] = true
			}
		}
	}
	return out
}

// assertNoDuplicate : le registre et les participants n'ont aucune ligne en double.
func (b *difficileBoot) assertNoDuplicate(label string) difficileCounts {
	b.env.t.Helper()
	c := b.counts()
	if c.registryRows != c.registryDistinct {
		b.env.t.Fatalf("%s : %d lignes au registre pour %d matchs distincts (doublon)", label, c.registryRows, c.registryDistinct)
	}
	if c.partRows != c.partDistinct {
		b.env.t.Fatalf("%s : %d lignes de participants pour %d couples (match, joueur) distincts (doublon)", label, c.partRows, c.partDistinct)
	}
	return c
}

// assertCursor : le marque-page ne dépasse jamais ce qui est rangé. L'entrée à Offset-1 est son match, et
// toutes celles de la position `depuis` à Offset-1 sont au registre ou introuvables.
func (b *difficileBoot) assertCursor(label string, j difficileJoueur, f *fakeHalo, depuis int, absent map[string]bool) {
	b.env.t.Helper()
	_, cur := b.historyState(j)
	if cur == nil {
		return
	}
	ids := difficileIDs(f)
	if cur.Offset < 1 || cur.Offset > len(ids) || ids[cur.Offset-1] != cur.LastMatchID {
		b.env.t.Fatalf("%s : marque-page %+v incohérent avec la liste (entrée %d = %q)", label, *cur, cur.Offset-1, func() string {
			if cur.Offset >= 1 && cur.Offset <= len(ids) {
				return ids[cur.Offset-1]
			}
			return "hors liste"
		}())
	}
	reg := b.registryIDs()
	for i := depuis; i < cur.Offset; i++ {
		if !reg[ids[i]] && !absent[ids[i]] {
			b.env.t.Fatalf("%s : le marque-page (%d) dépasse le match %s (position %d), ni rangé ni introuvable : trou", label, cur.Offset, ids[i], i)
		}
	}
}

// jusquAuBout lance des cycles jusqu'à la fin rangée de tous les joueurs de js, au plus difficileMaxCycles ;
// chaque cycle est vérifié (réussi, aucun doublon, fin rangée <=> rien en attente).
func (b *difficileBoot) jusquAuBout(label string, js ...difficileJoueur) []difficileCycle {
	b.env.t.Helper()
	var cycles []difficileCycle
	for len(cycles) < difficileMaxCycles {
		c := b.cycle()
		cycles = append(cycles, c)
		n := len(cycles)
		if !c.ran || c.err != nil {
			b.env.t.Fatalf("%s, cycle %d : orchestrateur non joué ou en échec (%v)", label, n, c.err)
		}
		b.assertNoDuplicate(label)
		// Un cycle de rattrapage est espacé de 10 minutes : la reprise périodique du journal d'écriture
		// (walRecoveryTask, toutes les 10 minutes) passe entre deux cycles.
		b.recoveryTick()
		fini := true
		for _, j := range js {
			out := c.player(j)
			if out.Status != "ok" {
				b.env.t.Fatalf("%s, cycle %d, %s : issue %q (%s)", label, n, j.gamertag, out.Status, out.FirstError)
			}
			complete, _ := b.historyState(j)
			if complete == out.HistoryPending {
				b.env.t.Fatalf("%s, cycle %d, %s : fin rangée=%v mais HistoryPending=%v", label, n, j.gamertag, complete, out.HistoryPending)
			}
			fini = fini && complete
		}
		if fini {
			return cycles
		}
	}
	b.env.t.Fatalf("%s : fin non rangée en %d cycles", label, difficileMaxCycles)
	return nil
}

// assertConverged : à la fin, tout est rangé, rien en double, rien en trop, fin rangée sans marque-page.
func (b *difficileBoot) assertConverged(label string, listes map[difficileJoueur]*fakeHalo, absent map[string]bool) {
	b.env.t.Helper()
	if u := b.env.monde.unexpectedRequests(); len(u) > 0 {
		b.env.t.Logf("%s : requêtes hors du faux Halo (réponse 404, sans conséquence sur le contrat) : %d, par exemple %q", label, len(u), u[0])
	}
	fs := make([]*fakeHalo, 0, len(listes))
	for _, f := range listes {
		fs = append(fs, f)
	}
	want := difficileAttendus(absent, fs...)
	got := b.registryIDs()
	for id := range want {
		if !got[id] {
			b.env.t.Errorf("%s : match %s listé et jamais rangé (trou)", label, id)
		}
	}
	for id := range got {
		if !want[id] {
			b.env.t.Errorf("%s : match %s au registre alors qu'aucune liste ne le porte, ou introuvable", label, id)
		}
	}
	c := b.assertNoDuplicate(label)
	if c.registryDistinct != len(want) {
		b.env.t.Errorf("%s : %d matchs au registre, attendu %d", label, c.registryDistinct, len(want))
	}
	for j, f := range listes {
		wantJ := 0
		for _, id := range difficileIDs(f) {
			if !absent[id] {
				wantJ++
			}
		}
		if c.perXUID[j.xuid] != wantJ {
			b.env.t.Errorf("%s : %d lignes de participants pour %s, attendu %d", label, c.perXUID[j.xuid], j.gamertag, wantJ)
		}
		complete, cur := b.historyState(j)
		if !complete {
			b.env.t.Errorf("%s : fin d'historique de %s non rangée", label, j.gamertag)
		}
		if cur != nil {
			b.env.t.Errorf("%s : marque-page de %s à effacer à la fin : %+v", label, j.gamertag, *cur)
		}
	}
}

// assertWALVide : le journal d'écriture de la file finit vide, sans journal corrompu.
func (e *difficileEnv) assertWALVide(label string) {
	e.t.Helper()
	if n := e.waitWALEmpty(2 * time.Minute); n != 0 {
		e.t.Errorf("%s : %d journaux d'écriture restent après la reprise", label, n)
	}
	if _, corrupted := e.walState(); corrupted {
		e.t.Errorf("%s : un journal d'écriture a été déplacé en corrompu", label)
	}
}

func difficileCumul(cycles ...[]difficileCycle) map[string]int {
	out := map[string]int{}
	for _, cs := range cycles {
		for _, c := range cs {
			for id, n := range c.stats {
				out[id] += n
			}
		}
	}
	return out
}

func difficileReads(cycles []difficileCycle, j difficileJoueur) []int {
	out := make([]int, 0, len(cycles))
	for _, c := range cycles {
		out = append(out, len(c.reads[j.xuid]))
	}
	return out
}
