package killsource

import (
	"cmp"
	"slices"
)

// bot_affectation.go — LES KILLS SUR UN BOT QUE LA PREMIERE PASSE LAISSE SANS LIGNE (temps 4 de
// l hybride).
//
// # CE QUE LA PREMIERE PASSE MANQUE
//
// Le kill-feed est humain-seul : un kill dont la victime est un bot n a presque jamais de kill-event
// 85 en face, donc presque jamais d identite de paquet, et la fenetre de 2,5 s ([tolMS]) est sa voie
// normale (`repli_mort_de_bot_premier_candidat`). Chaque kill y prend le PREMIER candidat de sa
// fenetre ([decodeCtx.apparierMortDeBot]). Deux kills voisins d un meme tueur sur deux bots ont
// chacun leur dead-state, et chacun tombe dans la fenetre de l autre : le second reprend le
// dead-state du premier, deja servi, et perd sa ligne pendant que le sien reste libre. Et une mort de
// bot dont le dead-state designe le bot lui-meme (chute, source globale) n a aucun candidat.
//
// # LE COMPLEMENT
//
// Il ne touche QUE les kills que la premiere passe laisse sans ligne faute de dead-state libre (aucun
// candidat, ou un candidat deja servi), et QUE les dead-states qu elle n a pas servis : aucune ligne
// de la premiere passe ne change, il en ajoute.
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

// completerLesMortsDeBot rend les kills de `ms` (la premiere passe) que `aCompleter` designe par
// leur instant — sans ligne, et sans dead-state ou avec un dead-state deja servi —, chacun avec le
// dead-state que le complement lui donne parmi ceux que `servis` ne porte pas. `servis` recoit les
// dead-states retenus.
func (c *decodeCtx) completerLesMortsDeBot(ms []botMatch, aCompleter func(int) bool,
	servis map[[3]int]bool,
) []botMatch {
	var reste []botMatch
	for _, m := range ms {
		if !aCompleter(m.event.timeMS) {
			continue
		}
		m.found, m.cand, m.parLaFenetre, m.sourceDeLaVictime = false, candidate{}, false, false
		reste = append(reste, m)
	}
	for _, deLaVictime := range []bool{false, true} {
		if deLaVictime && !c.opts.SelfSource {
			break
		}
		couple := func(m *botMatch, i int) bool {
			return !m.found && !servis[positionDuCandidat(c.scanCands[i])] &&
				c.coupleDeMortDeBot(m, c.scanCands[i], deLaVictime)
		}
		c.affecterParLePaquet(reste, servis, couple, deLaVictime)
		c.affecterParLaFenetre(reste, servis, couple, deLaVictime)
	}
	return reste
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

// copieDesServis rend une copie de la table des dead-states servis : le complement y ajoute ce qu il
// retient sans toucher a celle de la passe.
func copieDesServis(servis map[[3]int]bool) map[[3]int]bool {
	out := make(map[[3]int]bool, len(servis))
	for k := range servis {
		out[k] = true
	}
	return out
}
