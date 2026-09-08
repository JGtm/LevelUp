package replay

import (
	"sort"
)

// lives.go — LE PONT SLOT -> JOUEUR, LU AU LIEU D'ÊTRE VOTÉ.
//
// POURQUOI CE FICHIER REMPLACE UN VOTE. Le pont vivait dans owners.go, où il était élu :
// les lancers de grenade votaient pour désigner le propriétaire d'un slot. Un vote a
// besoin d'électeurs, et il y avait 70 lancers pour 99 vies — dont le premier à 73,1 s.
// Aucune vie antérieure ne pouvait donc être nommée, quelle que soit la qualité du
// décodage : **le défaut était dans le choix de la méthode, pas dans les données**.
// Résultat mesuré de ce vote : 26 slots couverts sur 99, et 147 tirs publiés sur 519.
//
// CE QU'ON FAIT À LA PLACE. Chaque vie de biped se termine par une mort, et le film porte
// le fil des morts : une victime, datée, nommée par son XUID. On nomme donc chaque vie par
// LA MORT QUI LA TERMINE. C'est une jointure sur un fait, pas une élection.
//
// MESURES sur 000d5950 (cmd/tmp_deathnaming), toutes avec leur témoin :
//
//	vies nommées                    90 / 105        témoin (morts replacées au hasard) : 10
//	écart d'appariement             médiane 34 ms, maximum 36 ms
//	slots changeant de porteur      0 / 90          — la table slot -> joueur est licite
//	tirs rattachés                  475 / 519 = 91,5 %   contre 398 par le vote supprimé
//	arme du tir dans le loadout     405 / 418 = 96,9 %   témoin (autre slot vivant) : 3,7 %
//
// LE DERNIER CONTRÔLE EST LE PLUS IMPORTANT : il ne partage AUCUNE pièce avec ce fichier.
// L'arme vient des records de dégât du flux de trames, le loadout du balayage des familles
// dans les records de biped des images-clés. Un rapport de 26x entre le rattachement et
// son témoin ne s'obtient pas par construction.
//
// L'INDEX DE JOUEUR N'EST PLUS RÉSOLU, IL EST LU. Il fut un temps calculé par affectation de
// coût minimal sur les 8! permutations ; le film l'écrit, et `player_index.go` le lit (26
// chunks concordants sur 000d5950, table identique à celle que le calcul produisait). Le pont
// n'a donc plus aucune part de choix : deux lectures composées, et rien d'autre.

// deathMatchWindowMS est l'écart maximal accepté entre la fin d'une vie et une mort du
// fil. La médiane mesurée étant de 34 ms et le maximum de 36, cette fenêtre borne le bruit
// d'horloge, pas le signal.
const deathMatchWindowMS = 150

// lifeGapUS : au-delà de ce trou dans un même slot, on ouvre une nouvelle vie. 5 s est
// très au-dessus du pas de réplication (~16 ms) et bien en deçà du temps de réapparition
// mesuré (médiane 8,0 s).
const lifeGapUS = 5_000_000

// Death est une mort du fil, telle que le film la porte : une identité et un instant.
// L'identité est le XUID — jamais un index (cf. la règle « un ordre n'est pas une
// identité », qui a déjà produit une fausse découverte dans ce chantier).
type Death struct {
	// XUID identifie la victime. Stable, global, indépendant de tout tri.
	XUID uint64
	// Gamertag est le nom porté PAR LE FILM lui-même, dans le même enregistrement que le xuid
	// (32 octets UTF-16LE). Il n'est pas obligatoire au rattachement — celui-ci ne travaille
	// que sur le xuid — mais il rend le rejeu lisible SANS base de données, ce qui est la
	// propriété que tout ce pipeline cherche à préserver. Vide si l'enregistrement ne le porte
	// pas ; l'identité reste alors le xuid.
	Gamertag string
	// TimeMS est l'instant de la mort sur l'horloge du MATCH (origine = début du match),
	// qui n'est pas celle du film. Le décalage entre les deux est résolu par mesure.
	TimeMS int64
}

// lifeSpan est une vie de biped : les positions d'un même slot sans trou majeur.
type lifeSpan struct {
	slot     uint32
	from, to int64  // microsecondes, horloge du film
	xuid     uint64 // identité lue dans le fil des morts ; 0 = non nommée
	// cause dit COMMENT la vie s'est terminée. Posée à la découpe (structure), écrasée par
	// [CauseVieMort] si le fil des morts apparie sa fin.
	//
	// LA MORT PRIME SUR LA STRUCTURE, jamais l'inverse : le trou de 5 s qui suit une mort est
	// le temps de réapparition, pas une coupure de réplication. Sans cette priorité, toute
	// mort suivie d'un respawn sortirait « coupure ».
	cause string
	// nomPar dit COMMENT ON SAIT À QUI la vie appartient — une question ORTHOGONALE à la
	// précédente, et les confondre est exactement ce qui a coûté la lecture d'isolement.
	//
	// UNE FERMETURE N'EST PAS UNE FIN. `nameClosedLives` déduit une identité par élimination
	// (un autre corps est réapparu, donc ce corps-ci était celui-là) ; elle ne dit RIEN sur
	// la façon dont la vie s'est terminée. Un survivant nommé par fermeture porte donc
	// `nomPar = closure` ET `cause = film_end` : il est identifié, et il n'est pas mort.
	nomPar string
}

// Les QUATRE causes de fin d'une vie. Elles sont toutes DÉTERMINABLES sans seuil arbitraire,
// et c'est la condition pour qu'elles existent : une cause qu'on devinerait ne serait qu'un
// avis présenté comme un fait.
//
//	CauseVieMort       le fil des morts apparie la fin de la vie (à deathMatchWindowMS,
//	                   médiane mesurée 34 ms). LA SEULE QUI DISE « CE JOUEUR EST MORT ».
//	CauseVieFinFilm    la réplication du slot s'arrête et ne reprend jamais : la vie court
//	                   jusqu'au bout de ce que le film montre. C'est le cas du SURVIVANT.
//	CauseVieCoupure    un trou de plus de lifeGapUS (5 s) a fermé la vie et aucune mort ne
//	                   l'apparie. Le cas typique est l'embarquement en véhicule : le biped
//	                   cesse d'être répliqué, le joueur est bien vivant.
const (
	CauseVieMort    = "death"
	CauseVieFinFilm = "film_end"
	CauseVieCoupure = "cut"
)

// Les DEUX provenances d'identité d'une vie. Elles répondent à « comment sait-on à qui elle
// appartient », jamais à « comment s'est-elle terminée ».
//
//	NomParMort        le fil des morts a nommé la vie par sa victime — une LECTURE.
//	NomParFermeture   une fermeture de slot l'a nommée par élimination (closures.go) — une
//	                  DÉDUCTION, et surtout PAS UNE MORT. Confondre les deux fabrique une
//	                  mort pour un survivant qui a tiré : c'est le P0 de la ronde 2
//	                  (2026-09-07), qui a coûté toute une lecture d'isolement.
const (
	NomParMort      = "death"
	NomParFermeture = "closure"
)

// buildLifeSpans découpe les trajectoires en vies. Un slot qui disparaît plus de lifeGapUS
// puis revient est une NOUVELLE vie : le slot migre aux réapparitions.
func buildLifeSpans(tracks map[uint32]slotTrack) []lifeSpan {
	slots := make([]uint32, 0, len(tracks))
	for s := range tracks {
		slots = append(slots, s)
	}
	sort.Slice(slots, func(i, j int) bool { return slots[i] < slots[j] })
	var out []lifeSpan
	for _, s := range slots {
		pts := tracks[s].pts
		if len(pts) == 0 {
			continue
		}
		start, last := int64(pts[0].TimestampUS), int64(pts[0].TimestampUS)
		for _, p := range pts[1:] {
			t := int64(p.TimestampUS)
			if t-last > lifeGapUS {
				// TROU AU-DELÀ DU SEUIL : la vie se ferme ici. C'est la cause STRUCTURELLE,
				// que le nommage écrasera s'il sait mieux (une mort, une fermeture).
				out = append(out, lifeSpan{slot: s, from: start, to: last, cause: CauseVieCoupure})
				start = t
			}
			last = t
		}
		// LA DERNIÈRE VIE DU SLOT N'EST FERMÉE PAR AUCUN TROU : ses points sont simplement
		// épuisés. C'est la fin de ce que le film montre de ce slot — pas une coupure.
		out = append(out, lifeSpan{slot: s, from: start, to: last, cause: CauseVieFinFilm})
	}
	return out
}

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

// nameLivesByDeaths pose l'identité de la victime sur la vie que sa mort termine.
//
// L'APPARIEMENT EST GLOUTON PAR ÉCART CROISSANT, ce qui rend le résultat indépendant de
// l'ordre d'itération — une boucle naïve donnerait un résultat différent selon l'ordre des
// slots, donc non reproductible.
func nameLivesByDeaths(lives []lifeSpan, deaths []Death, off int64) int {
	ends := lifeEndsMS(lives)
	type pair struct {
		di, li int
		d      int64
	}
	var ps []pair
	for di, d := range deaths {
		target := d.TimeMS + off
		for li, e := range ends {
			if delta := absI64(e - target); delta <= deathMatchWindowMS {
				ps = append(ps, pair{di, li, delta})
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
	n := 0
	for _, p := range ps {
		if usedD[p.di] || usedL[p.li] {
			continue
		}
		usedD[p.di], usedL[p.li] = true, true
		lives[p.li].xuid = deaths[p.di].XUID
		lives[p.li].cause = CauseVieMort
		lives[p.li].nomPar = NomParMort
		n++
	}
	return n
}

// lifeEndsMS rend la fin de chaque vie en millisecondes.
func lifeEndsMS(lives []lifeSpan) []int64 {
	out := make([]int64, len(lives))
	for i, l := range lives {
		out[i] = l.to / 1000
	}
	return out
}

// ownersFromLives compose la table slot -> index de joueur à partir des vies nommées et du
// pont index -> XUID.
//
// LA TABLE EST LICITE PARCE QU'UN SLOT NE CHANGE PAS DE PORTEUR : mesuré 0 slot sur 90 sur
// 000d5950. Si un film violait cette propriété, la table serait fausse par construction —
// d'où le compteur de collisions rendu au rapport plutôt que masqué.
// Le second retour donne slot -> XUID, c'est-à-dire l'IDENTITÉ du porteur et non son rang.
// Les deux sortent du même parcours et de la même règle de collision : les séparer ferait
// diverger deux tables censées dire la même chose.
//
// LE SLOT EN COLLISION EST DESORMAIS MARQUE, PAS SEULEMENT COMPTE (2026-09-07). La boucle garde
// le PREMIER occupant nomme et compte les suivants, mais `SlotCollisions` est un TOTAL de match :
// aucun consommateur ne pouvait savoir QUEL slot etait concerne, et tous — les marques de
// portage, les ramassages, les frags sous equipement actif, les calques d'objectif — heritaient
// donc d'un nom ARBITRAIRE (celui du premier occupant, par ordre des vies) sur ces slots-la. Le
// troisieme retour rend l'ensemble des slots ambigus, pour que « ce slot a eu deux occupants »
// cesse d'etre indiscernable de « ce slot appartient a ce joueur ».
func ownersFromLives(
	lives []lifeSpan, xuidToIndex map[uint64]int,
) (map[uint32]int, map[uint32]uint64, map[uint32]bool) {
	out := map[uint32]int{}
	byXUID := map[uint32]uint64{}
	ambigus := map[uint32]bool{}
	for _, l := range lives {
		if l.xuid == 0 {
			continue
		}
		idx, ok := xuidToIndex[l.xuid]
		if !ok {
			continue
		}
		if prev, seen := out[l.slot]; seen && prev != idx {
			ambigus[l.slot] = true
			continue // conflit : on ne tranche pas, on ne publie pas
		}
		out[l.slot] = idx
		byXUID[l.slot] = l.xuid
	}
	return out, byXUID, ambigus
}

func absI64(v int64) int64 {
	if v < 0 {
		return -v
	}
	return v
}

func minI64(a, b int64) int64 {
	if a < b {
		return a
	}
	return b
}

func maxI64(a, b int64) int64 {
	if a > b {
		return a
	}
	return b
}
