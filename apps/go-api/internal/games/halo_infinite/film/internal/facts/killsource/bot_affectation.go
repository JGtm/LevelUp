package killsource

import (
	"cmp"
	"slices"
)

// bot_affectation.go — QUEL DEAD-STATE DECRIT LA MORT D UN BOT (temps 4 de l hybride).
//
// # UN DEAD-STATE NE DECRIT QU UNE MORT
//
// Le kill-feed est humain-seul : un kill dont la victime est un bot n a presque jamais de kill-event
// 85 en face, donc presque jamais d identite de paquet, et la fenetre de 2,5 s ([tolMS]) est sa voie
// normale (`repli_mort_de_bot_premier_candidat`). Deux kills voisins d un meme tueur sur deux bots
// ont chacun leur dead-state, a quelques millisecondes de LEUR instant, et chacun tombe dans la
// fenetre de l autre. Laisser chaque kill prendre « le premier candidat » donnait au second kill le
// dead-state du premier : deja servi, la ligne du second se perdait, et le dead-state du second
// restait libre.
//
// L AFFECTATION EST DONC GLOBALE, et elle ne depend pas de l ordre des kills :
//
//	LECTURE   un kill dont le paquet est identifie prend le premier dead-state libre de CE paquet
//	          qui satisfait la contrainte de couple ;
//	REPLI     les couples (kill, dead-state) restants de la fenetre qui satisfont la contrainte se
//	          servent du plus PROCHE en temps au plus lointain ; un kill et un dead-state ne
//	          servent qu une fois. A ecart egal, l ordre des kills puis celui des dead-states.
//
// Le tout se joue deux fois : avec le couple ECRIT (le tueur du feed), puis, sur ce qui reste, avec
// la source qui appartient au bot lui-meme ([botMatch.sourceDeLaVictime]) — le temps 3 de
// l hybride, pour la population des bots.

// affecterLesMortsDeBot pose, sur chaque kill de `ms`, le dead-state qui decrit sa mort de bot.
func (c *decodeCtx) affecterLesMortsDeBot(ms []botMatch) {
	servis := map[[3]int]bool{}
	for _, deLaVictime := range []bool{false, true} {
		if deLaVictime && !c.opts.SelfSource {
			return
		}
		couple := func(m *botMatch, i int) bool {
			return !m.found && !servis[positionDuCandidat(c.scanCands[i])] &&
				c.coupleDeMortDeBot(m, c.scanCands[i], deLaVictime)
		}
		c.affecterParLePaquet(ms, servis, couple, deLaVictime)
		c.affecterParLaFenetre(ms, servis, couple, deLaVictime)
	}
}

// affecterParLePaquet : la LECTURE — le paquet que le kill-event 85 du kill designe.
func (c *decodeCtx) affecterParLePaquet(ms []botMatch, servis map[[3]int]bool,
	couple func(*botMatch, int) bool, deLaVictime bool,
) {
	for k := range ms {
		m := &ms[k]
		for i := range c.scanCands {
			cd := c.scanCands[i]
			if m.event.paquet.memeQue(cd.chunk, cd.pidx) && couple(m, i) {
				c.poserMortDeBot(m, i, false, deLaVictime)
				servis[positionDuCandidat(cd)] = true
				break
			}
		}
	}
}

// paireDeFenetre : un kill et un dead-state de sa fenetre, et leur ecart en millisecondes.
type paireDeFenetre struct{ kill, cand, ecart int }

// affecterParLaFenetre : le REPLI — les paires de la fenetre, de la plus proche a la plus lointaine.
func (c *decodeCtx) affecterParLaFenetre(ms []botMatch, servis map[[3]int]bool,
	couple func(*botMatch, int) bool, deLaVictime bool,
) {
	var paires []paireDeFenetre
	for k := range ms {
		for i, cd := range c.scanCands {
			if e := absMS(cd.ms - ms[k].event.timeMS); e <= tolMS && couple(&ms[k], i) {
				paires = append(paires, paireDeFenetre{kill: k, cand: i, ecart: e})
			}
		}
	}
	slices.SortStableFunc(paires, func(a, b paireDeFenetre) int {
		return cmp.Or(cmp.Compare(a.ecart, b.ecart), cmp.Compare(a.kill, b.kill), cmp.Compare(a.cand, b.cand))
	})
	for _, p := range paires {
		m, cd := &ms[p.kill], c.scanCands[p.cand]
		if m.found || servis[positionDuCandidat(cd)] {
			continue
		}
		c.poserMortDeBot(m, p.cand, true, deLaVictime)
		servis[positionDuCandidat(cd)] = true
	}
}

// poserMortDeBot : le dead-state `i` decrit la mort de bot du kill `m`.
func (c *decodeCtx) poserMortDeBot(m *botMatch, i int, parLaFenetre, deLaVictime bool) {
	m.found, m.cand, m.parLaFenetre, m.sourceDeLaVictime = true, c.scanCands[i], parLaFenetre, deLaVictime
}
