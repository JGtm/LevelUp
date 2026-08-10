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
//	                              couvre : son corps est l'une des vies anonymes dont l'INTERVALLE
//	                              couvre cet instant. S'il n'y en a QU'UNE, c'est elle.
//	B — par la réapparition       une vie commence UNE RÉAPPARITION après la mort qui l'a causée,
//	                              et le fil des morts NOMME cette victime. Si UNE SEULE mort tombe
//	                              dans la fenêtre, ET qu'aucune autre vie ne revendique cette
//	                              mort, la vie est la sienne.
//
// # LES GARDE-FOUS, ET ILS MORDENT
//
//	contestation   l'unicité manque, dans l'un des quatre sens -> aucune attribution
//	               deux corps possibles pour un même tir (A) · deux joueurs revendiquent le même
//	               corps (A) · deux corps revendiquent le même joueur (A — un joueur n'a qu'un
//	               DERNIER corps) · deux corps revendiquent la même mort (B — une mort ne rend
//	               qu'un corps, symétrique du refus de la vie qui voit deux morts)
//	corroboration  l'unicité du candidat ne prouve RIEN de son appartenance (A) : le corps déduit
//	               n'est attribué que si le tireur peut le PROLONGER — il a un corps connu, et
//	               tous ses corps connus s'achèvent avant que le candidat ne commence -> sinon
//	               rejetée. Ce contrôle absorbe et durcit l'ancien « recouvrement ».
//	recouvrement   un joueur n'a qu'un corps : si le corps déduit chevauche dans le temps une vie
//	               DÉJÀ nommée du même joueur, l'attribution est impossible -> rejetée (B)
//
// Mesuré sur les sept films, APRÈS LA RONDE DE CORROBORATION DU 2026-08-11 : **31 vies attribuées,
// 99 refusées** — 87 contestées, 12 rejetées. Ce compte de refus NE SE COMPARE PAS aux 17 du
// 2026-08-08 : la contestation couvre depuis le 2026-08-09 les vies écartées faute d'unicité
// d'INTERVALLE, qui n'étaient alors comptées nulle part. Un contrôle qui ne rejette jamais rien ne
// prouve rien ; celui-ci refuse plus de trois fois pour une attribution. C'est le pendant du
// critère « huit entités distinctes » qui a réfuté la piste i19 le 2026-07-28.
//
// # RÉSULTAT MESURÉ (lecture seule -> fermetures, sept films, 2026-08-11)
//
//	0edb8512 Team Slayer  93,4 -> 96,4      db7b8c3c CTF   88,5 -> 94,5
//	9aeca4b3 Team Slayer  89,0 -> 91,3      64e8adfa CTF   80,3 -> 87,4
//	000d5950 Fiesta       91,5 -> 93,1      829abef9 CTF   79,7 -> 88,7
//	01e1f945 KOTH         86,4 -> 89,2
//
// LE PLANCHER DU CORPUS EST PASSÉ SOUS LE CRITÈRE DU GARDE LOCAL, ET CE N'EST PAS UN DÉFAUT DE CE
// FICHIER : 87,4 % sur `64e8adfa` contre 88 % exigés. La corroboration a retiré 3 fermetures A sur
// ce film (98 tirs), toutes attribuées à des tireurs que rien ne situait — exactement le « corps
// d'autrui » que le chantier interdit. Le seuil n'a PAS été touché et la règle n'a PAS été
// relâchée pour faire remonter le chiffre : l'arbitrage entre la justesse et le seuil
// d'activation appartient à l'utilisateur (cf. `api/handlers/replay_local_gate.go`).
//
// LES CHIFFRES BAISSENT, ET C'EST LE SENS ATTENDU d'un correctif de justesse : quatre films sont
// inchangés, trois reculent (90,8 -> 87,4 · 89,7 -> 89,2 · 91,6 -> 91,3). Un correctif de ce genre
// qui ferait MONTER le compte serait le signal qu'il a élargi au lieu de resserrer.
//
// Détail, méthode et échec de réglage : `.ai/V7.5/RECHERCHE_CTF_TIRS_PERDUS.md` §7.5, §7.5bis
// et §7.5ter.

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
// désigne l'unique vie anonyme QUI COUVRE cet instant, s'il n'y en a qu'une.
//
// L'UNICITÉ SE JUGE SUR L'INTERVALLE DE LA VIE, PAS SUR SES ÉCHANTILLONS, et ce point a été payé.
// Le premier jet exigeait du candidat une position répliquée à moins de `shotPosToleranceUS`
// (120 ms) de l'instant du tir. Or une vie survit à un trou de réplication de `lifeGapUS` (5 s) :
// deux vies anonymes pouvaient couvrir l'instant, celle du VRAI tireur être dans son trou, et
// l'autre — la seule échantillonnée — passer pour l'unique candidate. Le corps d'autrui était
// alors attribué sans qu'aucun garde-fou ne s'en aperçoive, puisque du point de vue du code il
// n'y avait qu'un candidat.
//
// L'échantillon ne sert donc qu'au RATTACHEMENT (`slotFor`, qui refuse toujours de poser un
// événement sans position proche), jamais à trancher l'unicité.
func closeByAvailableBody(tracks map[uint32]slotTrack, owner map[uint32]int,
	lives []lifeSpan, fire []FireEventRef, rep *closureReport) {
	free := freeLives(owner, lives)
	if len(free) == 0 {
		return
	}
	claims, blocked := claimsFromShots(tracks, owner, free, fire)
	attributeClaimedBodies(tracks, owner, claims, rep)
	// Une vie contestée à un instant peut être seule candidate à un autre : seules celles qui
	// n'ont JAMAIS été revendiquées sont des déductions abandonnées. Le comptage ne dépend pas
	// de l'ordre d'itération de la map, donc rien à trier ici.
	for slot := range blocked {
		if _, claimed := claims[slot]; !claimed {
			rep.contested++
		}
	}
}

// claimsFromShots dépouille les tirs ORPHELINS — ceux dont l'auteur n'a aucun corps rattachable
// à cet instant — et rend qui revendique quel corps libre. Le second retour retient les corps
// écartés faute d'unicité : sans lui, l'abstention la plus fréquente de la fermeture A ne serait
// comptée nulle part.
func claimsFromShots(tracks map[uint32]slotTrack, owner map[uint32]int, free []lifeSpan,
	fire []FireEventRef) (map[uint32]map[int]int, map[uint32]bool) {
	claims := map[uint32]map[int]int{}
	blocked := map[uint32]bool{}
	for _, e := range fire {
		if _, r := slotFor(tracks, owner, e.FilmIndex, e.TimestampUS); r == reasonAttached {
			continue
		}
		cand := livesCoveringAt(free, e.TimestampUS)
		if len(cand) != 1 { // plusieurs corps possibles : on ne tranche pas
			for _, l := range cand {
				blocked[l.slot] = true
			}
			continue
		}
		if claims[cand[0].slot] == nil {
			claims[cand[0].slot] = map[int]int{}
		}
		claims[cand[0].slot][e.FilmIndex]++
	}
	return claims, blocked
}

// attributeClaimedBodies pose au pont les corps qu'UN SEUL tireur revendique, que ce tireur ne
// revendique QU'UNE FOIS, et qu'il peut PROLONGER. Chacun des trois refus est compté.
func attributeClaimedBodies(tracks map[uint32]slotTrack, owner map[uint32]int,
	claims map[uint32]map[int]int, rep *closureReport) {
	twice := shootersClaimingTwoBodies(claims)
	for _, slot := range sortedClaimSlots(claims) {
		if len(claims[slot]) != 1 { // deux joueurs pour un même corps
			rep.contested++
			continue
		}
		pi := onlyPlayerIndex(claims[slot])
		if twice[pi] { // deux corps pour un même joueur
			rep.contested++
			continue
		}
		if !bodyExtendsShooter(tracks, owner, slot, pi) {
			rep.refused++
			continue
		}
		owner[slot] = pi
		rep.byShot++
	}
}

// shootersClaimingTwoBodies rend les tireurs que DEUX corps libres revendiquent.
//
// UN JOUEUR N'A QU'UN DERNIER CORPS, et sans ce contrôle l'ORDRE DES SLOTS déciderait laquelle
// des deux déductions passe : la première attribuée devient un corps connu du tireur, ce qui fait
// tomber la seconde — ou pas, selon que son numéro de slot est plus petit ou plus grand. Deux
// déductions qui s'excluent ne sont pas deux déductions ; les deux tombent. C'est le symétrique
// exact du refus « deux joueurs revendiquent le même corps » ici, et de « deux corps revendiquent
// la même mort » à la fermeture B.
func shootersClaimingTwoBodies(claims map[uint32]map[int]int) map[int]bool {
	n := map[int]int{}
	for _, m := range claims {
		if len(m) != 1 {
			continue
		}
		n[onlyPlayerIndex(m)]++
	}
	out := map[int]bool{}
	for pi, c := range n {
		if c > 1 {
			out[pi] = true
		}
	}
	return out
}

// bodyExtendsShooter — LA CORROBORATION, ET ELLE EST POSITIVE.
//
// L'UNICITÉ DU CANDIDAT NE PROUVE RIEN DE SON APPARTENANCE, et c'est le défaut que cette fonction
// ferme. `livesCoveringAt` rend les corps libres dont l'intervalle couvre l'instant du tir ; qu'il
// n'y en ait qu'un dit seulement qu'UN SEUL CORPS LIBRE EST LÀ — jamais qu'il est celui du tireur.
// Le tireur, lui, peut n'être nulle part dans les positions à cet instant : l'événement de tir est
// un record INDÉPENDANT de la réplication des bipeds, un trou de plus de `lifeGapUS` scinde une vie
// en deux dont aucune ne couvre l'instant, et une vie peut s'achever avant lui. Sans corroboration,
// le corps d'AUTRUI était alors attribué au tireur — silencieusement, puisque du point de vue du
// code il ne restait qu'un candidat. C'est exactement ce que le chantier interdit.
//
// CE QUE LE MODÈLE AUTORISE, ET RIEN D'AUTRE. Une vie anonyme est une vie QUE NULLE MORT NE TERMINE
// (`lives.go`) : dans la vie d'un joueur, c'est donc sa DERNIÈRE — celle qui court de sa dernière
// mort au coup de sifflet, précisément la zone où les tirs se perdent. Le corps déduit n'est donc
// attribuable au tireur que sous DEUX conditions, toutes deux positives :
//
//	ancrage        le tireur possède AU MOINS UN corps déjà connu. Sans ancre, rien ne le situe
//	               dans le film, et l'unicité du candidat ne dit rien de lui.
//	terminalité    TOUS ses corps connus s'achèvent AVANT que le corps candidat ne commence. Un
//	               corps connu postérieur ferait du candidat une vie INTERMÉDIAIRE du tireur —
//	               donc une vie terminée par une mort, qui l'aurait NOMMÉE. Elle ne l'a pas été :
//	               le candidat n'est pas de lui.
//
// LA SECONDE CONDITION ABSORBE LE CONTRÔLE DE RECOUVREMENT et le durcit. « S'achever avant que le
// candidat ne commence » interdit à la fois le chevauchement (un joueur n'a qu'un corps) et la
// postériorité. C'est le même invariant, énoncé dans le sens qui PROUVE au lieu de celui qui ne
// réfute pas — un test de non-contradiction laisse passer tout ce qu'il ne voit pas.
//
// CE QU'ELLE NE COUVRE PAS, ET IL FAUT LE DIRE : un tireur dont TOUS les corps connus précèdent le
// candidat reste attribuable même s'il était en réalité invisible à cet instant. Ce cas-là ne se
// distingue pas du cas nominal avec les pièces disponibles — c'est la limite de la fermeture A,
// pas un contrôle oublié.
func bodyExtendsShooter(tracks map[uint32]slotTrack, owner map[uint32]int, slot uint32, pi int) bool {
	cand := tracks[slot].pts
	if len(cand) == 0 {
		return false
	}
	from := cand[0].TimestampUS
	anchored := false
	for s, p := range owner {
		if p != pi || s == slot {
			continue
		}
		pts := tracks[s].pts
		if len(pts) == 0 {
			continue
		}
		if pts[len(pts)-1].TimestampUS >= from {
			return false // chevauchement, ou corps postérieur : le candidat n'est pas terminal
		}
		anchored = true
	}
	return anchored
}

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
	claims := map[uint64][]uint32{} // victime -> slots des vies qui la revendiquent
	for _, l := range free {
		cand := victimsInWindow(deaths, off, l.from, lo, hi)
		if len(cand) == 0 {
			continue
		}
		if len(cand) > 1 {
			rep.contested++
			continue
		}
		claims[cand[0]] = append(claims[cand[0]], l.slot)
	}
	for _, xuid := range sortedVictims(claims) {
		slots := claims[xuid]
		if len(slots) != 1 { // une mort, deux corps : les deux déductions tombent
			rep.contested += len(slots)
			continue
		}
		pi, ok := byXUID[xuid]
		if !ok {
			// L'identité n'est dans aucune table d'index : la déduction est REJETÉE, pas
			// silencieusement perdue. Un rejet non compté ferait mentir la somme publiée.
			rep.refused++
			continue
		}
		if overlapsNamedLife(tracks, owner, slots[0], pi) {
			rep.refused++
			continue
		}
		owner[slots[0]] = pi
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

// livesCoveringAt rend les vies libres dont l'INTERVALLE contient tUS — c'est-à-dire tous les
// corps qui pouvaient être là, y compris ceux dont la réplication a un trou à cet instant.
//
// C'EST VOLONTAIREMENT PLUS LARGE QUE LE RATTACHEMENT. Filtrer ici sur la présence d'un
// échantillon rendrait INVISIBLES les candidats en trou de réplication, et une candidature
// invisible ne conteste rien : la fermeture croirait n'avoir qu'un candidat là où elle en a deux.
func livesCoveringAt(free []lifeSpan, tUS uint64) []lifeSpan {
	var out []lifeSpan
	t := int64(tUS)
	for _, l := range free {
		if t < l.from || t > l.to {
			continue
		}
		out = append(out, l)
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

// sortedVictims rend les victimes revendiquées dans un ordre STABLE. La raison est la même que
// pour `sortedClaimSlots` : le garde-fou de recouvrement lit le pont EN COURS de construction,
// donc l'ordre d'attribution est observable dans le résultat.
func sortedVictims(m map[uint64][]uint32) []uint64 {
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
