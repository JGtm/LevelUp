package killsource

import "levelup/go-api/internal/games/halo_infinite/film/types"

// feed_couples.go — LE COUPLE (TUEUR, VICTIME) EST LU AU KILL-EVENT 85, PLUS RECOLLE SUR LE
// VOISIN (lot 1.9.3 du plan `.ai/PLAN_DECODEUR_FILM_2026-09-13.md`, D13).
//
// # CE QUE LE FILM ECRIT, ET QUI N ETAIT PAS LU
//
// Le kill-feed groupe ses evenements par INSTANT : la plupart du temps un instant porte le kill
// ET la mort, et le couple est alors ECRIT par le feed lui-meme. Quand il ne porte que le kill,
// l ancienne lecture RECOLLAIT la victime du voisin immediat (deux instants au plus) — une
// heuristique de voisinage qui decide un fait que le film ECRIT AILLEURS : le kill-event de code
// 85 porte VICTIME ET TUEUR dans le meme enregistrement
// (`victime(E5) tueur(E5) [% TUEUR] R1 assistant(E5) [% ASSISTANT]`, [readKillEvent]). Ce lecteur
// existait ; il n etait employe que pour l ASSISTANT.
//
// # POURQUOI LA LECTURE PEUT DECIDER SANS SE MORDRE LA QUEUE
//
// Un kill-event porte des INDICES de replication, pas des noms. Les traduire par la bijection
// resolue serait CIRCULAIRE : [solveBijection] s ajuste sur [killFeed.pairs], donc sur le
// recollage que l on veut remplacer. On n emploie donc QUE les indices EPINGLES — la table des
// joueurs de `chunk_00` (lots 1.5 et 1.8) et BOT_METADATA —, poses par [buildRoster] AVANT toute
// inference et independants de tout couple. Un film dont la table est refusee ne dispose d aucun
// epinglage : la lecture s y tait, et le repli entier reprend la main, compte.
//
// # DEUX TEMPS, ET L ORDRE EST LE RESULTAT
//
// La fenetre d appariement vaut 2,5 s de part et d autre ([tolMS]) : un joueur qui tue deux fois
// en 2,5 s y produit DEUX kill-events au meme nom de tueur, et une lecture qui ne regarderait que
// le tueur ne trancherait pas. On donne donc d abord a chaque couple ECRIT AU MEME INSTANT le
// kill-event dont LE COUPLE ENTIER correspond — ce n est pas une decision, c est une
// CONSOMMATION —, puis on lit le couple des kills orphelins dans ce qui RESTE. Mesure du lot
// (21 films, 8 builds, tableaux en §5 du plan) : sans ce premier temps, 13 kills orphelins sur
// 271 sont AMBIGUS et 3 paraissent en desaccord ; avec lui, 1 seul reste ambigu et le desaccord
// tombe a ZERO.
//
// # LE RESULTAT MESURE, ET IL FAUT LE DIRE EN ENTIER
//
// Sur les 21 films entiers (8 builds, 14 temoins du corpus gate) : 281 kills sans mort en face,
// dont 198 dont le film ECRIT le couple — et il ecrit EXACTEMENT celui que le recollage rendait,
// 198 fois sur 198. Le gain n est donc pas metrique, il est de NATURE : 198 couples cessent
// d etre une reconstruction. Une seule ligne change sur les 21 films, et c est celle qui compte :
// sur `4f77afc1`, le recollage FABRIQUAIT un couple dont le film dit que la victime est un BOT.
// Les 81 autres kills orphelins sont MUETS (aucun kill-event epingle dans la fenetre) et le repli
// les sert, compte.

// killDeBot : un kill dont le film NOMME un bot en victime.
//
// IL N ENTRE PAS DANS [killFeed.pairs], ET C EST TOUT L INTERET DU LOT : le kill-feed est
// HUMAIN-SEUL (un bot n a pas de XUID, sa mort ne produit aucun evenement `death`), donc un
// couple dont la victime est un bot n est pas un couple du feed. L ancienne reconstruction en
// FABRIQUAIT un en prenant la mort du voisin ; ici la victime est NOMMEE a la source (« les vies
// anonymes n existent pas », decision utilisateur du 2026-09-06) et l instant part directement
// vers la population des morts de bot ([decodeCtx.resolveBotDeaths]).
type killDeBot struct {
	// ev : l instant du feed — `timeMS` et `killer` viennent du kill-feed, `victim` porte le nom
	// du bot tel que BOT_METADATA l ecrit (suffixe [BotSuffix] compris).
	ev feedEvent
	// victime : l indice de replication du bot, LU au kill-event. Il contraint l appariement au
	// dead-state au lieu de laisser passer n importe quel indice de bot.
	victime int
}

// resoudreCouples : LA DECOMPOSITION DU KILL-FEED, sous la decision du kill-event 85.
//
// Elle remplit `pairs`, `real`, `lus`, `fab`, `botLus`, `orphK` et `orphD`, et rend ses
// compteurs. `recs` nil ou `r` sans epinglage = la lecture se tait partout, et le comportement
// est EXACTEMENT celui d avant le lot : c est ce que les tests purs exercent.
func (kf *killFeed) resoudreCouples(recs []killEventRec, r *roster) types.CoupleStats {
	res := &resolveurDeCouples{kf: kf, recs: recs, r: r,
		pris: make([]bool, len(recs)), prisMort: make([]bool, len(kf.events))}
	res.consommerLesCouplesDuMemeInstant()
	res.resoudre()
	res.isolerLesMortsSansTueur()
	return res.stats
}

// resolveurDeCouples : l etat d une resolution. Une structure plutot que six parametres.
type resolveurDeCouples struct {
	kf   *killFeed
	recs []killEventRec
	r    *roster
	// pris : les kill-events deja consommes. Un enregistrement ne parle que d UNE mort.
	pris []bool
	// prisMort : les instants du feed dont la MORT a ete consommee par un couple.
	prisMort []bool
	stats    types.CoupleStats
}

// consommerLesCouplesDuMemeInstant : temps 1. Chaque couple que le FEED ecrit prend le
// kill-event dont les deux indices epingles nomment exactement ce couple.
//
// LE KILL-EVENT CONSOMME N EST PLUS JETE (lot 1.9.7) : son IDENTITE DE PAQUET reste sur
// l instant, et c est elle qui appariera le dead-state. `resoudre` recopie l instant tel quel
// dans `pairs` et `real`, donc l identite voyage sans autre geste.
func (res *resolveurDeCouples) consommerLesCouplesDuMemeInstant() {
	for i := range res.kf.events {
		e := res.kf.events[i]
		if e.killer == "" || e.victim == "" {
			continue
		}
		if j := res.chercherCoupleEcrit(e.timeMS, e.killer, e.victim); j >= 0 {
			res.pris[j] = true
			res.kf.events[i].paquet = res.paquetDe(j)
		}
	}
}

// paquetDe : l identite de paquet d un kill-event, par son indice.
func (res *resolveurDeCouples) paquetDe(rec int) paquetID {
	return paquetID{chunk: res.recs[rec].chunk, pidx: res.recs[rec].pidx, ok: true}
}

// chercherCoupleEcrit : le premier kill-event NON CONSOMME de la fenetre dont les deux indices
// epingles nomment exactement ce couple. -1 quand il n y en a pas.
func (res *resolveurDeCouples) chercherCoupleEcrit(ms int, tueur, victime string) int {
	for i := range res.recs {
		if res.pris[i] || !dansLaFenetre(res.recs[i].ms, ms) {
			continue
		}
		k, okK := res.r.nomEpingle(res.recs[i].fields.killer)
		v, okV := res.r.nomEpingle(res.recs[i].fields.victim)
		if okK && okV && k == tueur && v == victime {
			return i
		}
	}
	return -1
}

// resoudre : temps 2. Un instant a la fois, dans l ordre du feed.
func (res *resolveurDeCouples) resoudre() {
	for i := range res.kf.events {
		e := res.kf.events[i]
		if e.killer == "" {
			continue
		}
		if e.victim != "" {
			res.kf.pairs = append(res.kf.pairs, e)
			res.kf.real = append(res.kf.real, e)
			res.prisMort[i] = true
			res.stats.MemeInstant++
			continue
		}
		res.resoudreUnKillSansMort(i)
	}
}

// resoudreUnKillSansMort : LA DECISION. On LIT d abord ; on ne se replie que sur un silence.
func (res *resolveurDeCouples) resoudreUnKillSansMort(i int) {
	idx, rec, ambigu := res.lireVictime(res.kf.events[i].timeMS, res.kf.events[i].killer)
	switch {
	case ambigu:
		res.stats.Ambigu++
		res.repliRecollageSurLeVoisin(i)
	case rec < 0:
		res.stats.Muet++
		res.repliRecollageSurLeVoisin(i)
	default:
		res.pris[rec] = true
		res.publierLeCoupleLu(i, idx, rec)
	}
}

// lireVictime : la victime que le film ECRIT pour ce tueur a cet instant, prise dans les
// kill-events NON CONSOMMES. `ambigu` quand deux enregistrements nomment des victimes
// differentes : la lecture ne tranche pas, donc elle ne decide pas.
func (res *resolveurDeCouples) lireVictime(ms int, tueur string) (victime, rec int, ambigu bool) {
	victime, rec = -1, -1
	for i := range res.recs {
		if res.pris[i] || !dansLaFenetre(res.recs[i].ms, ms) {
			continue
		}
		k, okK := res.r.nomEpingle(res.recs[i].fields.killer)
		if !okK || k != tueur {
			continue
		}
		v := res.recs[i].fields.victim
		if _, okV := res.r.nomEpingle(v); !okV {
			continue
		}
		if rec >= 0 && v != victime {
			return -1, -1, true
		}
		victime, rec = v, i
	}
	return victime, rec, false
}

// publierLeCoupleLu : le film a nomme la victime. Reste a la ranger.
//
// UNE VICTIME BOT N EST PAS UN COUPLE DU FEED : elle part vers la population des morts de bot,
// avec son indice, et AUCUNE mort de voisin n est consommee — celle-ci reste disponible pour le
// temps qui cherche un TUEUR bot ou une mort que personne ne revendique.
func (res *resolveurDeCouples) publierLeCoupleLu(i, victime, rec int) {
	e := res.kf.events[i]
	nom, _ := res.r.nomEpingle(victime)
	paq := res.paquetDe(rec)
	if res.r.isBotIndex(victime) {
		res.stats.VictimesBotLues++
		res.kf.botLus = append(res.kf.botLus,
			killDeBot{ev: feedEvent{timeMS: e.timeMS, killer: e.killer, victim: nom, paquet: paq},
				victime: victime})
		return
	}
	res.stats.Lus++
	couple := feedEvent{timeMS: e.timeMS, killer: e.killer, victim: nom, paquet: paq}
	if v := res.voisinPortantLaMort(i, nom); v >= 0 {
		res.prisMort[v] = true
		couple.victimXUID = res.kf.events[v].victimXUID
		res.stats.Accord++
	} else {
		// Le film nomme un joueur dont aucun voisin immediat ne porte la mort. La lecture prime
		// (D13) ; le xuid vient alors de la table du feed, et la CONTRADICTION se compte.
		couple.victimXUID = res.kf.xuidDe[nom]
		res.stats.Contradiction++
	}
	res.kf.pairs = append(res.kf.pairs, couple)
	res.kf.lus = append(res.kf.lus, couple)
}

// voisinPortantLaMort : le voisin IMMEDIAT (deux instants au plus, le MEME voisinage que le
// repli) qui porte la mort de ce joueur, et qu aucun couple n a deja consomme.
//
// LA LECTURE NE CHANGE PAS QUI EST CONSOMME, ELLE CHANGE QUI EST PUBLIE : sur les 198 couples ou
// la lecture et le repli se prononcent tous les deux, c est le MEME instant et le meme xuid — ce
// qui rend la sortie identique a l octet partout ou la lecture confirme (mesure du lot, §5).
func (res *resolveurDeCouples) voisinPortantLaMort(i int, victime string) int {
	for d := 1; d <= porteeDuRecollage && i+d < len(res.kf.events); d++ {
		o := res.kf.events[i+d]
		if o.victim == victime && o.killer == "" && !res.prisMort[i+d] {
			return i + d
		}
	}
	return -1
}

// porteeDuRecollage : la fenetre du REPLI, en nombre d instants du feed. Valeur historique du
// chantier (`reconstructPairs`, avant le lot 1.9.3) : deux instants. Elle ne se regle pas — elle
// n existe que pour les instants ou le film se tait, et elle est destinee a disparaitre.
const porteeDuRecollage = 2

// repliRecollageSurLeVoisin : LE REPLI NOMME `repli_couple_recolle_sur_le_voisin` (registre des replis,
// D14). Il ne se declenche QUE sur un silence ou une ambiguite de la lecture, jamais sur un
// desaccord avec elle — l ordre est fixe : lire d abord, se replier ensuite.
//
// MECANISME INCHANGE depuis l origine du chantier, et c est voulu : la conversion change QUI
// DECIDE, pas ce que le repli fait quand il reprend la main. Deux morts a la meme seconde
// n arrivent pas toujours dans le meme instant du feed ; le kill orphelin prend alors la mort du
// voisin immediat.
func (res *resolveurDeCouples) repliRecollageSurLeVoisin(i int) {
	e := res.kf.events[i]
	for d := 1; d <= porteeDuRecollage && i+d < len(res.kf.events); d++ {
		o := res.kf.events[i+d]
		if o.victim == "" || o.killer != "" {
			continue
		}
		couple := feedEvent{timeMS: e.timeMS, killer: e.killer, victim: o.victim, victimXUID: o.victimXUID}
		res.kf.pairs = append(res.kf.pairs, couple)
		res.kf.fab = append(res.kf.fab, couple)
		res.prisMort[i+d] = true
		res.stats.Recolles++
		return
	}
	res.kf.orphK = append(res.kf.orphK, e)
	res.stats.Perdus++
}

// isolerLesMortsSansTueur : les morts que AUCUN couple n a consommees. Le kill-feed porte la
// MORT et aucun kill en face : LE TUEUR N EST PAS HUMAIN, ou personne ne revendique cette mort.
func (res *resolveurDeCouples) isolerLesMortsSansTueur() {
	for i := range res.kf.events {
		if res.kf.events[i].victim != "" && !res.prisMort[i] {
			res.kf.orphD = append(res.kf.orphD, res.kf.events[i])
		}
	}
}

// dansLaFenetre : l instant d un kill-event tombe-t-il dans la demi-fenetre d appariement d un
// instant du feed ?
func dansLaFenetre(msRec, msFeed int) bool {
	dt := msRec - msFeed
	return dt >= -tolMS && dt <= tolMS
}
