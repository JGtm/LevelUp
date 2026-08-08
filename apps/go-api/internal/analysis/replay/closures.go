package replay

import "sort"

// closures.go — REFERMER LE PONT SANS JAMAIS DEVINER.
//
// # LE DÉFAUT QUE CE FICHIER RÉPARE
//
// `lives.go` nomme chaque vie par LA MORT QUI LA TERMINE. Une vie que nulle mort ne termine —
// celle qui court de la dernière mort d'un joueur jusqu'au coup de sifflet — reste donc anonyme,
// et tout ce qu'elle porte est rejeté « slot introuvable ». Mesuré le 2026-08-08 sur sept films :
// **63 à 92 % des tirs perdus tombent dans une vie non nommée**, et le dernier décile du film
// perd de 40 à 74 % de ses tirs — dans TOUS les modes, pas seulement en CTF.
//
// # POURQUOI CE N'EST PAS LE VOTE RETIRÉ LE 2026-07-28
//
// La distinction est de nature, pas de degré, et elle est la seule chose qui autorise ce fichier
// à exister après le retrait du repli voté :
//
//	le vote        plusieurs candidats, on garde le mieux placé        -> un CHOIX
//	la fermeture   un seul candidat POSSIBLE, les autres sont exclus   -> une DÉDUCTION
//
// Dès que deux candidats subsistent, RIEN n'est attribué. La règle de l'utilisateur reste
// intacte : « je préfère rien afficher que quelque chose de complètement faux ».
//
// # LES DEUX FERMETURES
//
//	A — par le corps disponible   un joueur tire alors qu'aucune de ses vies nommées ne le
//	                              couvre : son corps est l'une des vies anonymes vivantes à cet
//	                              instant. S'il n'y en a QU'UNE, c'est elle.
//	B — par la réapparition       une vie commence UNE RÉAPPARITION après la mort qui l'a causée,
//	                              et le fil des morts NOMME cette victime. Si UNE SEULE mort tombe
//	                              dans la fenêtre, la vie est la sienne.
//
// # LES DEUX GARDE-FOUS, ET ILS MORDENT
//
//	contestation   deux joueurs revendiquent le même corps -> aucune attribution
//	recouvrement   un joueur n'a qu'un corps : si le corps déduit chevauche dans le temps une vie
//	               DÉJÀ nommée du même joueur, l'attribution est impossible -> rejetée
//
// Mesuré sur les sept films : **33 vies attribuées, 17 refusées** par ces deux contrôles. Un
// contrôle qui ne rejette jamais rien ne prouve rien ; celui-ci rejette. C'est le pendant du
// critère « huit entités distinctes » qui a réfuté la piste i19 le 2026-07-28.
//
// # RÉSULTAT MESURÉ (avant -> après, sept films)
//
//	0edb8512 Team Slayer  93,4 -> 96,4      db7b8c3c CTF   88,5 -> 94,5
//	9aeca4b3 Team Slayer  89,0 -> 95,0      64e8adfa CTF   80,3 -> 92,6  (+12,3)
//	000d5950 Fiesta       91,5 -> 93,3      829abef9 CTF   79,7 -> 88,7  (+9,0)
//	01e1f945 KOTH         86,4 -> 89,7
//
// Détail, méthode et échec de réglage : `.ai/V7.5/RECHERCHE_CTF_TIRS_PERDUS.md` §7.5.

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

// closureReport compte ce que les fermetures ont attribué ET refusé. Publier l'un sans l'autre
// laisserait croire que la déduction ne se trompe jamais.
type closureReport struct {
	byShot, byRespawn  int
	contested, refused int
}

// closeBridge applique les deux fermetures, dans l'ordre mesuré (A puis B), et rend le pont
// augmenté avec son compte rendu. Le pont d'entrée n'est jamais modifié.
func closeBridge(tracks map[uint32]slotTrack, owner map[uint32]int, lives []lifeSpan,
	deaths []Death, off int64, byXUID map[uint64]int, fire []FireEventRef) (map[uint32]int, closureReport) {
	var rep closureReport
	out := copyOwners(owner)
	closeByAvailableBody(tracks, out, lives, fire, &rep)
	closeByRespawn(tracks, out, lives, deaths, off, byXUID, &rep)
	return out, rep
}

// FireEventRef est ce que les fermetures ont besoin de savoir d'un tir : QUI et QUAND. Le type
// est réduit à dessein — la fermeture ne doit pas pouvoir dépendre de l'arme ni de la visée, qui
// n'ont aucun pouvoir de désignation.
type FireEventRef struct {
	FilmIndex   int
	TimestampUS uint64
}

// closeByAvailableBody — FERMETURE A. Un tir dont l'auteur n'a aucune vie nommée à cet instant
// désigne l'unique vie anonyme vivante, s'il n'y en a qu'une.
func closeByAvailableBody(tracks map[uint32]slotTrack, owner map[uint32]int,
	lives []lifeSpan, fire []FireEventRef, rep *closureReport) {
	free := freeLives(owner, lives)
	if len(free) == 0 {
		return
	}
	claims := map[uint32]map[int]int{}
	for _, e := range fire {
		if _, r := slotFor(tracks, owner, e.FilmIndex, e.TimestampUS); r == reasonAttached {
			continue
		}
		alive := livesAliveAt(free, tracks, e.TimestampUS)
		if len(alive) != 1 { // deux corps possibles : on ne tranche pas
			continue
		}
		if claims[alive[0].slot] == nil {
			claims[alive[0].slot] = map[int]int{}
		}
		claims[alive[0].slot][e.FilmIndex]++
	}
	for _, slot := range sortedClaimSlots(claims) {
		if len(claims[slot]) != 1 {
			rep.contested++
			continue
		}
		pi := onlyPlayerIndex(claims[slot])
		if overlapsNamedLife(tracks, owner, slot, pi) {
			rep.refused++
			continue
		}
		owner[slot] = pi
		rep.byShot++
	}
}

// closeByRespawn — FERMETURE B. Une vie commence une réapparition après la mort qui l'a causée ;
// si une seule mort du fil tombe dans la fenêtre, la vie est celle de sa victime.
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
	for _, l := range free {
		cand := victimsInWindow(deaths, off, l.from, lo, hi)
		if len(cand) == 0 {
			continue
		}
		if len(cand) > 1 {
			rep.contested++
			continue
		}
		pi, ok := byXUID[cand[0]]
		if !ok {
			continue
		}
		if overlapsNamedLife(tracks, owner, l.slot, pi) {
			rep.refused++
			continue
		}
		owner[l.slot] = pi
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

// freeLives rend les vies sans identité dont le slot n'est pas DÉJÀ au pont : un slot nommé par
// une autre de ses vies n'a rien à déduire.
func freeLives(owner map[uint32]int, lives []lifeSpan) []lifeSpan {
	var out []lifeSpan
	for _, l := range lives {
		if l.xuid != 0 {
			continue
		}
		if _, known := owner[l.slot]; known {
			continue
		}
		out = append(out, l)
	}
	return out
}

// livesAliveAt rend les vies libres qui portent une position à moins de la tolérance de
// rattachement de tUS — « vivante » au sens du rattachement, pas au sens du jeu.
func livesAliveAt(free []lifeSpan, tracks map[uint32]slotTrack, tUS uint64) []lifeSpan {
	var out []lifeSpan
	t := int64(tUS)
	for _, l := range free {
		if t < l.from || t > l.to {
			continue
		}
		if _, d := tracks[l.slot].at(tUS); d <= shotPosToleranceUS {
			out = append(out, l)
		}
	}
	return out
}

// overlapsNamedLife — LE GARDE-FOU QUI PEUT RÉFUTER UNE DÉDUCTION. Un joueur n'a qu'un corps :
// si le slot déduit porte des positions en même temps qu'un slot déjà attribué au même joueur,
// l'attribution est IMPOSSIBLE, et on la rejette plutôt que de la publier.
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

func copyOwners(in map[uint32]int) map[uint32]int {
	out := make(map[uint32]int, len(in))
	for k, v := range in {
		out[k] = v
	}
	return out
}

// sortedClaimSlots rend les slots revendiqués dans un ordre STABLE : itérer une map donnerait un
// pont différent d'une exécution à l'autre, donc un artefact non reproductible.
func sortedClaimSlots(m map[uint32]map[int]int) []uint32 {
	out := make([]uint32, 0, len(m))
	for s := range m {
		out = append(out, s)
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
