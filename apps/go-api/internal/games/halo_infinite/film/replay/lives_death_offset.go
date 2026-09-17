package replay

// lives_death_offset.go — LE CALAGE DE L HORLOGE DU FIL DES MORTS SUR CELLE DU FILM, et rien
// d autre : le vote qui localise les calages candidats, l affinage au pas de 10 ms qui tranche,
// et l appariement mort <-> fin de vie qui en decoule.
//
// DEPLACEMENT PUR depuis `lives.go` (lot 2.7 volet publication, 2026-09-16) : le fichier passait
// 500 lignes en portant DEUX sujets — ce QU EST une vie (le type `Death`, `lifeSpan`, les causes
// de fin et les provenances d identite, qui restent dans `lives.go` avec leur garde-rail
// `archlint/no_life_cause_divergence_test.go`) et COMMENT on cale les deux horloges. Le second
// est ici. Aucune ligne n a change : memes constantes, memes fonctions, meme ordre.

import "sort"

// deathOffsetStepMS est le pas du balayage fin du calage, et la grille reste ancrée sur la
// PREMIÈRE FIN DE VIE — celle sur laquelle le balayage linéaire d'avant 2026-09-07 était déjà
// ancré (il partait de `min(fins) − 60 000`, et 60 000 est multiple de 10). Un film déjà bien
// calé retient donc le MÊME entier qu'avant, et sa re-cuisson ne déplace rien.
const deathOffsetStepMS = 10

// deathOffsetCandidats : combien de paniers du vote sont affinés avant de trancher.
//
// TROIS, ET PAS UN, parce que le vote est une HEURISTIQUE de localisation quand l'affinage est
// la mesure. Le balayage exhaustif d'avant ne pouvait pas se tromper de pic — il les évaluait
// tous ; une heuristique qui n'en affine qu'un rendrait un calage faux SANS SIGNAL si elle se
// trompait de panier (constat PONT-R1/C1). En affiner plusieurs et garder le meilleur compte
// RÉEL rend le verdict insensible à une erreur de localisation, et donne au passage le
// deuxième meilleur compte : la marge, qui se publie et se surveille.
const deathOffsetCandidats = 3

// deathOffsetMargeMin : sous ce rapport entre le calage retenu et le meilleur des AUTRES
// candidats, le calage n'est plus franchement meilleur que le bruit, et la cuisson le
// journalise.
//
// DEUX, parce que c'est le premier seuil au-dessus duquel « deux fois mieux que tout le reste »
// cesse d'être vrai. Mesuré sur le parc : x8,9 sur `51ebbc0f` (71 appariements contre 8) et
// x10,5 sur `d9781168` (157 contre 15) — un ordre de grandeur, très au-dessus. Le seuil n'est
// donc pas un réglage : c'est le plancher sous lequel il faut aller REGARDER.
const deathOffsetMargeMin = 2

// bestDeathOffset résout le décalage entre l'horloge du fil des morts et celle du film. Il rend
// le calage, le nombre de morts qu'il apparie, et le meilleur compte des AUTRES candidats — la
// MARGE, dont `BridgeHealth` publie les deux termes.
//
// LE MAXIMUM EST UN PLATEAU : toute la largeur de la fenêtre d'acceptation donne le même
// compte. On retient son CENTRE — au bord, tous les écarts d'appariement vaudraient la
// demi-fenêtre, ce qui ferait passer un mauvais calage pour bon.
//
// # LA MARGE AMONT DE 60 s A ÉTÉ SUPPRIMÉE, ET C'EST LE DÉFAUT QUE CE FICHIER FERMAIT MAL
//
// Le balayage partait de `min(fins de vie) − 60 000`. La grandeur que cette borne suppose
// petite est donc l'instant de match auquel correspond LA PLUS PRÉCOCE DES FINS DE VIE DU FILM
// — pas la première mort du match, qui lui est seulement corrélée (`d9781168` : première fin de
// vie à 18,4 s, première mort à 53,6 s ; `43716616` : première mort à 60,4 s et pourtant sain,
// parce que sa première fin de vie tombe à 28,9 s). Une vie se termine aussi SANS mort — fin de
// film, fin de manche, trou de réplication —, et c'est cette fin-là qui fixe la borne.
//
// Le fil des morts est daté depuis le début du MATCH, or la partie ne commence pas à t = 0 :
// les joueurs rejoignent après la mise en place. Quand la première fin de vie du film tombe
// au-delà de 60 s de match, le vrai calage passe SOUS la borne basse et l'optimiseur se rabat
// sur un pic de bruit.
//
// MESURE DU 2026-09-07, cinq films du parc sur 106 — et ce sont EXACTEMENT les cinq dont
// l'origine du fil n'était pas publiée (`resolveOriginMs` prend ce calage pour témoin) :
//
//	film        1re fin de vie   morts appariées      vies nommées
//	51ebbc0f          71 393 ms      9  ->  71  / 71      9 ->  71 / 87
//	fb1a1a72          63 402 ms     17  -> 140  / 141    17 -> 140 / 147
//	4f77afc1          87 480 ms     44  -> 192  / 300    44 -> 192 / 375
//	11de8353          87 061 ms     31  -> 155  / 166    31 -> 150 / 246
//	06dfe6d9         109 834 ms     37  -> 225  / 261    37 -> 225 / 291
//
// Les quatre témoins de contrôle (première fin de vie à 18,4 / 28,9 / 46,7 / 52,3 s) rendent le
// même nombre de vies nommées avant et après : la frontière est bien cette grandeur-là, et le
// critère ne souffre aucune exception dans les deux sens.
//
// # LA PLAGE EST CELLE DES DONNÉES, ET LE BALAYAGE DEVIENT UN VOTE PUIS UN AFFINAGE
//
// Aucune constante ne remplace 60 s : le support complet est
// `[min(fins) − max(morts), max(fins) − min(morts)]` — hors de là, aucune mort ne peut tomber
// sur aucune fin de vie. Le balayer au pas de 10 ms coûterait vingt fois plus cher sur un film
// BTB, d'où le vote de [voteDeathOffsets], qui localise les calages candidats en UN parcours,
// et [refineDeathOffset], qui garde la règle historique (pas de 10 ms, plateau centré) sur la
// seule fenêtre utile autour de chacun.
func bestDeathOffset(lives []lifeSpan, deaths []Death) (int64, int, int) {
	ends := lifeEndsMS(lives)
	if len(ends) == 0 || len(deaths) == 0 {
		return 0, 0, 0
	}
	var meilleurOff int64
	meilleurN, secondN := -1, 0
	for _, panier := range voteDeathOffsets(ends, deaths) {
		off, n := refineDeathOffset(ends, deaths, panier)
		// Deux paniers voisins peuvent affiner vers le MÊME plateau : le compter comme
		// « deuxième candidat » ferait croire à une marge nulle là où il n'y a qu'un calage.
		if meilleurN >= 0 && absI64(off-meilleurOff) <= 2*deathMatchWindowMS {
			if n > meilleurN {
				meilleurOff, meilleurN = off, n
			}
			continue
		}
		switch {
		case n > meilleurN:
			meilleurOff, meilleurN, secondN = off, n, meilleurN
		case n > secondN:
			secondN = n
		}
	}
	if meilleurN < 0 {
		return 0, 0, 0
	}
	if secondN < 0 {
		secondN = 0
	}
	return meilleurOff, meilleurN, secondN
}

// voteDeathOffsets localise les `k` meilleurs calages candidats : chaque couple (fin de vie,
// mort) désigne l'écart qui les apparierait, et l'on compte ces écarts par paniers de la
// largeur de la fenêtre d'appariement. Au vrai calage, toutes les morts désignent le même
// panier.
//
// UNE VOIX PAR MORT ET PAR PANIER, JAMAIS UNE PAR COUPLE, et c'est la correction du constat
// PONT-R1/C1 : `buildLifeSpans` termine la vie de chaque slot à son dernier point répliqué,
// donc une fin de manche ou une fin de film arrête toutes les pistes dans un même cycle de
// réplication (~16 ms, très en deçà d'un panier). Croisées avec un multi-kill, ces `k` fins
// SIMULTANÉES et ces `m` morts SIMULTANÉES déposaient `k × m` voix dans un seul panier de
// bruit — 96 voix pour 24 fins et 4 morts, de quoi passer devant le vrai calage. Dédoublonnées
// par mort, elles n'en pèsent plus que `m`.
//
// ET AUTANT PAR FIN DE VIE, ce qui ferme le constat PONT-R2/D3. Dédoublonner d'un seul côté
// laissait le vote compter une grandeur que l'affinage ne mesure PAS : le vote comptait les
// morts appariables, quand [countDeathMatches] apparie 1:1 (chaque fin de vie sert une seule
// fois). Un amas de `M` morts distinctes vise donc CHAQUE fin de vie isolée du film et y dépose
// `M` voix pour UNE SEULE paire réalisable — 20 voix pour 1 appariement, mesuré sur la fixture
// adversariale de `pont_marge_test.go`, assez pour remplir le budget de [deathOffsetCandidats]
// et faire rendre 2 appariements là où le vrai calage en apparie 15.
//
// La voix d'un panier est donc `min(morts distinctes, fins de vie distinctes)` : c'est la borne
// de Hall/König de l'appariement 1:1 maximal du panier, donc un MAJORANT EXACT de ce que
// l'affinage y mesurera. Le vote et l'affinage comptent enfin la même grandeur, et aucun seuil
// n'est introduit.
//
// LE BUDGET N'EST PLUS UN PARAMÈTRE : les quatre appelants passaient tous
// [deathOffsetCandidats], et un levier que personne ne bouge est du code mort déguisé en
// souplesse (`unparam` le signalait, lot R7).
func voteDeathOffsets(ends []int64, deaths []Death) []int64 {
	const k = deathOffsetCandidats
	const w = deathMatchWindowMS
	instantsMorts := make([]int64, len(deaths))
	for i, d := range deaths {
		instantsMorts[i] = d.TimeMS
	}
	parMort := paniersParPivot(instantsMorts, ends, false) // pivot = les morts
	parFin := paniersParPivot(ends, instantsMorts, true)   // pivot = les fins de vie
	type panier struct {
		centre int64
		voix   int
	}
	var tous []panier
	for g := range parMort {
		shift := int64(g) * (w / 2)
		for b, n := range parMort[g] {
			// Le centre du panier, ramené sur l'axe des écarts.
			tous = append(tous, panier{centre: b*w + w/2 - shift, voix: min(n, parFin[g][b])})
		}
	}
	// L'ordre d'itération d'une map n'est pas garanti : à égalité de voix, le plus petit centre
	// tranche, pour que deux exécutions rendent la même liste.
	sort.Slice(tous, func(i, j int) bool {
		if tous[i].voix != tous[j].voix {
			return tous[i].voix > tous[j].voix
		}
		return tous[i].centre < tous[j].centre
	})
	out := make([]int64, 0, k)
	for _, p := range tous {
		proche := false
		for _, c := range out {
			if absI64(p.centre-c) <= w {
				proche = true
				break
			}
		}
		if proche {
			continue
		}
		out = append(out, p.centre)
		if len(out) == k {
			break
		}
	}
	return out
}

// paniersParPivot compte, pour chacun des paniers des DEUX grilles décalées d'une demi-largeur,
// le nombre de PIVOTS DISTINCTS qui y déposent au moins un écart. Le pivot est le côté
// dédoublonné : une fin de vie qui vise dix morts d'un même panier n'y pèse qu'une voix.
//
// L'écart d'un couple est toujours `fin de vie − instant de mort` ; `pivotEstFin` dit de quel
// côté vient le pivot, pour que les deux passes indexent LE MÊME panier.
func paniersParPivot(pivots, autres []int64, pivotEstFin bool) [2]map[int64]int {
	const w = deathMatchWindowMS
	grids := [2]map[int64]int{{}, {}}
	vus := [2]map[int64]bool{{}, {}}
	for _, p := range pivots {
		clear(vus[0])
		clear(vus[1])
		for _, a := range autres {
			diff := a - p
			if pivotEstFin {
				diff = p - a
			}
			for g, b := range [2]int64{floorDivI64(diff, w), floorDivI64(diff+w/2, w)} {
				if !vus[g][b] {
					vus[g][b] = true
					grids[g][b]++
				}
			}
		}
	}
	return grids
}

// refineDeathOffset garde la règle historique — pas de [deathOffsetStepMS], plateau centré —
// appliquée à la seule fenêtre utile autour d'un candidat.
//
// LA FENÊTRE FAIT DEUX FOIS LA LARGEUR D'APPARIEMENT DE CHAQUE CÔTÉ, et c'est une borne, pas
// un réglage : un plateau ne peut pas dépasser `2 × deathMatchWindowMS` (au-delà, une mort au
// moins sort de la fenêtre), et le centre du panier voté est à au plus une demi-largeur du
// vrai calage. Tout plateau du maximum tient donc dedans.
func refineDeathOffset(ends []int64, deaths []Death, around int64) (int64, int) {
	anchor := ends[0]
	for _, e := range ends {
		anchor = minI64(anchor, e)
	}
	start := alignOnGridI64(around-2*deathMatchWindowMS, anchor, deathOffsetStepMS)
	bestN := -1
	var plateau []int64
	for off := start; off <= around+2*deathMatchWindowMS; off += deathOffsetStepMS {
		if n := countDeathMatches(ends, deaths, off); n > bestN {
			bestN, plateau = n, []int64{off}
		} else if n == bestN {
			plateau = append(plateau, off)
		}
	}
	return plateau[len(plateau)/2], bestN
}

// floorDivI64 est la division entière vers le BAS, y compris sur un dividende négatif — la
// division de Go tronque vers zéro, ce qui ferait partager un panier aux écarts de part et
// d'autre de l'origine.
func floorDivI64(a, b int64) int64 {
	q := a / b
	if a%b != 0 && (a < 0) != (b < 0) {
		q--
	}
	return q
}

// alignOnGridI64 rend le plus petit multiple de `step` décalé de `anchor` qui atteint `v`.
func alignOnGridI64(v, anchor, step int64) int64 {
	if r := ((v-anchor)%step + step) % step; r != 0 {
		v += step - r
	}
	return v
}

// countDeathMatches compte les morts appariables à une fin de vie, chaque vie servant une
// seule fois.
func countDeathMatches(ends []int64, deaths []Death, off int64) int {
	used := make([]bool, len(ends))
	n := 0
	for _, d := range deaths {
		if i := nearestFreeEnd(ends, used, d.TimeMS+off); i >= 0 {
			used[i] = true
			n++
		}
	}
	return n
}

// nearestFreeEnd rend l'index de la fin de vie libre la plus proche de target, dans la
// fenêtre ; -1 si aucune.
func nearestFreeEnd(ends []int64, used []bool, target int64) int {
	bi, bd := -1, int64(deathMatchWindowMS+1)
	for i, e := range ends {
		if used[i] {
			continue
		}
		if d := absI64(e - target); d < bd {
			bd, bi = d, i
		}
	}
	return bi
}

// deathPair apparie UNE mort du fil a LA vie qu'elle termine.
type deathPair struct {
	// li est l'indice de la vie, di celui de la mort.
	li, di int
}

// apparierMortsEtVies apparie chaque mort du fil a la vie qu'elle termine, et ne fait QUE cela.
//
// # ELLE NE NOMME PLUS (lot E2, 2026-09-08)
//
// Elle posait l'identite de la victime sur la vie. Le film ECRIT cette identite dans le record
// de creation du corps (cf. identity_registry_creation.go) : le pont par morts est donc devenu
// une VERIFICATION, et l'appariement qu'il produit sert deux choses seulement — la CAUSE de fin
// (`CauseVieMort` : la seule qui dise « ce joueur est mort ») et la confrontation du nom direct
// a la victime. Ce qui nomme, ou verifie, est le RESSORT DE L'APPELANT ; cette fonction rend une
// mesure, pas une decision.
//
// L'APPARIEMENT EST GLOUTON PAR ECART CROISSANT, ce qui rend le resultat independant de l'ordre
// d'iteration — une boucle naive donnerait un resultat different selon l'ordre des slots, donc
// non reproductible. Le resultat sort TRIE PAR VIE, pour la meme raison : un journal dont les
// lignes changent d'ordre d'un run a l'autre ne se compare pas.
func apparierMortsEtVies(lives []lifeSpan, deaths []Death, off int64) []deathPair {
	ends := lifeEndsMS(lives)
	type candidat struct {
		di, li int
		d      int64
	}
	var ps []candidat
	for di, d := range deaths {
		target := d.TimeMS + off
		for li, e := range ends {
			if delta := absI64(e - target); delta <= deathMatchWindowMS {
				ps = append(ps, candidat{di, li, delta})
			}
		}
	}
	sort.Slice(ps, func(i, j int) bool {
		if ps[i].d != ps[j].d {
			return ps[i].d < ps[j].d
		}
		return ps[i].li < ps[j].li // départage stable : jamais l'ordre de la map
	})
	usedD := make([]bool, len(deaths))
	usedL := make([]bool, len(lives))
	var out []deathPair
	for _, p := range ps {
		if usedD[p.di] || usedL[p.li] {
			continue
		}
		usedD[p.di], usedL[p.li] = true, true
		out = append(out, deathPair{li: p.li, di: p.di})
	}
	sort.Slice(out, func(i, j int) bool { return out[i].li < out[j].li })
	return out
}

// lifeEndsMS rend la fin de chaque vie en millisecondes.
func lifeEndsMS(lives []lifeSpan) []int64 {
	out := make([]int64, len(lives))
	for i, l := range lives {
		out[i] = l.to / 1000
	}
	return out
}
