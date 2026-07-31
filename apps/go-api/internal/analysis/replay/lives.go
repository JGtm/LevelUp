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
}

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
				out = append(out, lifeSpan{slot: s, from: start, to: last})
				start = t
			}
			last = t
		}
		out = append(out, lifeSpan{slot: s, from: start, to: last})
	}
	return out
}

// bestDeathOffset résout le décalage entre l'horloge du fil des morts et celle du film.
//
// LE MAXIMUM EST UN PLATEAU : toute la largeur de la fenêtre d'acceptation donne le même
// compte. On retient son CENTRE — au bord, tous les écarts d'appariement vaudraient la
// demi-fenêtre, ce qui ferait passer un mauvais calage pour bon.
func bestDeathOffset(lives []lifeSpan, deaths []Death) (int64, int) {
	ends := lifeEndsMS(lives)
	if len(ends) == 0 || len(deaths) == 0 {
		return 0, 0
	}
	lo, hi := ends[0], ends[0]
	for _, e := range ends {
		lo, hi = minI64(lo, e), maxI64(hi, e)
	}
	bestN := -1
	var plateau []int64
	// La plage balayée est celle des fins de vie : l'origine du fil des morts est le début
	// du match, qui tombe forcément dedans. La marge amont couvre l'avant-match.
	for off := lo - 60_000; off <= hi; off += 10 {
		if n := countDeathMatches(ends, deaths, off); n > bestN {
			bestN, plateau = n, []int64{off}
		} else if n == bestN {
			plateau = append(plateau, off)
		}
	}
	return plateau[len(plateau)/2], bestN
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
func ownersFromLives(
	lives []lifeSpan, xuidToIndex map[uint64]int,
) (map[uint32]int, map[uint32]uint64, int) {
	out := map[uint32]int{}
	byXUID := map[uint32]uint64{}
	collisions := 0
	for _, l := range lives {
		if l.xuid == 0 {
			continue
		}
		idx, ok := xuidToIndex[l.xuid]
		if !ok {
			continue
		}
		if prev, seen := out[l.slot]; seen && prev != idx {
			collisions++
			continue // conflit : on ne tranche pas, on ne publie pas
		}
		out[l.slot] = idx
		byXUID[l.slot] = l.xuid
	}
	return out, byXUID, collisions
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
