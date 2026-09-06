package replay

import "sort"

// closures_respawn.go — LA FERMETURE B, ET ELLE SEULE.
//
// Sortie de closures.go le 2026-09-06, quand ce fichier a franchi les 500 lignes en gagnant la
// DÉSIGNATION DE LA VIE (cf. `closureReport.closedLife`). La doctrine et les garde-fous des deux
// fermetures restent écrits en tête de closures.go, qui porte la fermeture A et le rapport commun.
//
// CE N'EST PAS UN DÉPLACEMENT PUR, et le dire faussement ferait sauter au relecteur le cœur du
// correctif (constat C3 de la revue REG-R1). Ce qui a changé en même temps que le déplacement,
// et rien d'autre :
//
//	closeByRespawn   `claims` passe de `map[uint64][]uint32` (des SLOTS) à `map[uint64][]int`
//	                 (des INDICES DE VIE) ; la boucle parcourt `free` en indices et lit
//	                 `lives[i].from` ; le slot se relit par `lives[vies[0]].slot` ; et surtout
//	                 **`rep.noteLife(slot, vies[0])` est AJOUTÉ** — la ligne qui fait fonctionner
//	                 le correctif pour cette fermeture.
//	sortedVictims    signature accordée au nouveau type de `claims`.
//	respawnWindow, victimsInWindow, overlapsNamedLife, containsXUID : identiques à l'octet près.
//
// La logique de DÉCISION, elle, est inchangée : mêmes gardes, mêmes refus, mêmes compteurs —
// vérifié sur données, `coverage.bridge` est identique avant et après sur les films témoins.

// respawnHalfWidthUS : demi-largeur de la fenêtre de réapparition, en microsecondes.
//
// LA RÉAPPARITION EST DÉTERMINISTE, et c'est une constante DU MATCH, pas du jeu : mesurée sur les
// vies déjà nommées, l'écart entre le centile 5 et la médiane vaut de 2 ms à 202 ms sur les sept
// films (8,09 s sur trois d'entre eux, 10,18 s sur quatre). 750 ms couvre donc largement la
// dispersion réelle tout en restant vingt fois plus étroit que l'intervalle entre deux morts.
//
// LE RÉGLAGE PRÉCÉDENT ÉTAIT [p05, p95] ET IL A ÉCHOUÉ, ce qui vaut d'être conservé : le
// centile 95 monte à 51,7 s et 67,7 s sur deux films (les vies dont la mort précédente du même
// joueur n'est PAS celle qui les a fait réapparaître), rendant 13 vies sur 13 contestées et le
// gain NUL là où il était le plus nécessaire.
const respawnHalfWidthUS = 750_000

// closeByRespawn — FERMETURE B. Une vie commence une réapparition après la mort qui l'a causée ;
// si une seule mort du fil tombe dans la fenêtre, la vie est celle de sa victime.
//
// L'EXCLUSION JOUE DANS LES DEUX SENS, et il a manqué le second. Refuser la vie qui voit deux
// morts ne suffit pas : il faut aussi refuser la MORT que deux vies revendiquent. Une mort ne
// rend qu'un corps ; deux vies qui la réclament chacune sans concurrente ne sont pas deux
// déductions, c'est une déduction impossible — et sans ce contrôle, l'ordre de parcours des slots
// décidait laquelle des deux gagnait, l'autre héritant du même joueur quelques instants plus tard.
// Le comptage des revendications est donc symétrique de la map `claims` de la fermeture A.
func closeByRespawn(tracks map[uint32]slotTrack, owner map[uint32]int, lives []lifeSpan,
	deaths []Death, off int64, byXUID map[uint64]int, rep *closureReport) {
	free := freeLives(owner, lives)
	if len(free) == 0 || len(deaths) == 0 {
		return
	}
	lo, hi := respawnWindow(lives, deaths, off)
	if lo == 0 && hi == 0 {
		return // aucune vie nommée : rien à calibrer, donc rien à déduire
	}
	claims := map[uint64][]int{} // victime -> vies qui la revendiquent
	for _, i := range free {
		cand := victimsInWindow(deaths, off, lives[i].from, lo, hi)
		if len(cand) == 0 {
			continue
		}
		if len(cand) > 1 {
			rep.contested++
			continue
		}
		claims[cand[0]] = append(claims[cand[0]], i)
	}
	for _, xuid := range sortedVictims(claims) {
		vies := claims[xuid]
		if len(vies) != 1 { // une mort, deux corps : les deux déductions tombent
			rep.contested += len(vies)
			continue
		}
		pi, ok := byXUID[xuid]
		if !ok {
			// L'identité n'est dans aucune table d'index : la déduction est REJETÉE, pas
			// silencieusement perdue. Un rejet non compté ferait mentir la somme publiée.
			rep.refused++
			continue
		}
		slot := lives[vies[0]].slot
		if overlapsNamedLife(tracks, owner, slot, pi) {
			rep.refused++
			continue
		}
		owner[slot] = pi
		rep.noteLife(slot, vies[0])
		rep.byRespawn++
	}
}

// respawnWindow calibre la fenêtre de réapparition SUR LE FILM TRAITÉ : la médiane de l'écart
// entre le début d'une vie nommée et la mort précédente de son propre joueur, plus ou moins
// respawnHalfWidthUS. Une constante importée d'un autre film serait une supposition.
func respawnWindow(lives []lifeSpan, deaths []Death, off int64) (int64, int64) {
	var d []int64
	for _, l := range lives {
		if l.xuid == 0 {
			continue
		}
		best := int64(-1)
		for _, dd := range deaths {
			if dd.XUID != l.xuid {
				continue
			}
			if t := dd.TimeMS + off; t < l.from/1000 && (best < 0 || t > best) {
				best = t
			}
		}
		if best >= 0 {
			d = append(d, l.from/1000-best)
		}
	}
	if len(d) == 0 {
		return 0, 0
	}
	sort.Slice(d, func(i, j int) bool { return d[i] < d[j] })
	med := d[len(d)/2] * 1000 // en microsecondes
	return med - respawnHalfWidthUS, med + respawnHalfWidthUS
}

// victimsInWindow rend les xuids DISTINCTS des morts dont la réapparition tomberait dans la
// fenêtre du début de vie fromUS.
func victimsInWindow(deaths []Death, off int64, fromUS, lo, hi int64) []uint64 {
	var out []uint64
	for _, d := range deaths {
		delta := fromUS - (d.TimeMS+off)*1000
		if delta < lo || delta > hi {
			continue
		}
		if !containsXUID(out, d.XUID) {
			out = append(out, d.XUID)
		}
	}
	return out
}

// overlapsNamedLife — LE GARDE-FOU QUI PEUT RÉFUTER UNE DÉDUCTION DE LA FERMETURE B. Un joueur
// n'a qu'un corps : si le slot déduit porte des positions en même temps qu'un slot déjà attribué
// au même joueur, l'attribution est IMPOSSIBLE, et on la rejette plutôt que de la publier.
//
// LA FERMETURE A NE L'APPELLE PLUS : sa corroboration (`bodyExtendsShooter`) exige davantage — non
// pas l'absence de contradiction, mais la preuve que le corps PROLONGE le tireur. B n'en a pas
// besoin : son identité vient du fil des morts, pas de l'unicité d'un candidat.
func overlapsNamedLife(tracks map[uint32]slotTrack, owner map[uint32]int, slot uint32, pi int) bool {
	cand := tracks[slot].pts
	if len(cand) == 0 {
		return false
	}
	from, to := cand[0].TimestampUS, cand[len(cand)-1].TimestampUS
	for s, p := range owner {
		if p != pi || s == slot {
			continue
		}
		pts := tracks[s].pts
		if len(pts) == 0 {
			continue
		}
		if pts[0].TimestampUS <= to && from <= pts[len(pts)-1].TimestampUS {
			return true
		}
	}
	return false
}

// sortedVictims rend les victimes revendiquées dans un ordre STABLE. La raison est la même que
// pour `sortedClaimSlots` : le garde-fou de recouvrement lit le pont EN COURS de construction,
// donc l'ordre d'attribution est observable dans le résultat.
func sortedVictims(m map[uint64][]int) []uint64 {
	out := make([]uint64, 0, len(m))
	for x := range m {
		out = append(out, x)
	}
	sort.Slice(out, func(i, j int) bool { return out[i] < out[j] })
	return out
}

func onlyPlayerIndex(m map[int]int) int {
	for k := range m {
		return k
	}
	return -1
}

func containsXUID(xs []uint64, x uint64) bool {
	for _, v := range xs {
		if v == x {
			return true
		}
	}
	return false
}
