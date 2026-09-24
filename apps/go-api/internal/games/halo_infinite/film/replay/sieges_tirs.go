package replay

// sieges_tirs.go — LA PLACE LUE DANS LES TIRS, ET LA MESURE DE L'AFFICHAGE (lot M2.3, 2026-09-23).
//
// # LE FAIT (rapport `RAPPORT_equipes_b1ad85eb.md`, « le fait qui ouvre la solution » ; sonde P4)
//
// L'index de tireur d'un tir n'est pas l'index de participant : c'est la PLACE, et le remplacant
// en herite. Mesure : 3 remplacements sur 3 (b1ad85eb place 5 -> Claudors 84/84, 43716616 place 5
// -> KernelPanic10 65/65, 43e96765 place 0 -> Cmillward21), et chaque autre index de tireur tombe
// a 89-100 % dans les vies du joueur de la table au meme numero. La lecture est donc :
//
//	un tir de la place P a l'instant T que l'occupant d'index P ne couvre pas appartient a un
//	ARRIVANT ; s'il n'y a qu'UN arrivant present a T (d'equipe compatible), le tir vote pour lui ;
//	un arrivant dont TOUS les votes designent la MEME place, libre pendant sa presence, la LIT.
//
// Deux votes pour deux places, ou une place deja prise pendant sa presence : CONTESTE, compte, et
// l'arrivant retombe sur le chainage. Aucun seuil : l'unanimite ou rien.
//
// # CE QUE LA LECTURE NE PEUT PAS DIRE
//
// L'index de tireur PERSISTE dans les faits est l'index sur 4 bits (`grammar.FireEvent.FilmIndex`) :
// au-dela de 15 joueurs (BTB) il confond la place P et la place P+16. Sur un tel film la lecture
// s'ABSTIENT en entier (`tirsIndexTronque`) — le lot M4b lira l'index sur 5 bits. Les bots
// n'ecrivent aucun tir long (sonde P4) : leur place passe par le chainage.

import "sort"

// placesLisiblesParLesTirs : le nombre de places que l'index de tireur persiste distingue — 1 << 4,
// la largeur de `grammar.FireEvent.FilmIndex` (cf. l'en-tete). Ce n'est pas un seuil mesure : c'est
// la largeur du champ, et le jour ou les faits portent l'index sur 5 bits elle devient 1 << 5.
const placesLisiblesParLesTirs = 1 << 4

// lireLesPlacesDansLesTirs : la LECTURE `tirs` (cf. l'en-tete). Rend les places lues, les arrivants
// contestes, et si la lecture s'est abstenue faute d'un index de tireur assez large.
func (pp *poseDesPlaces) lireLesPlacesDansLesTirs(fire []FireEventRef) (lues, contestes int, tronque bool) {
	if pp.indexDeTireurTronque() {
		return 0, 0, true
	}
	candidats := pp.arrivants()
	if len(candidats) == 0 || len(fire) == 0 {
		return 0, 0, false
	}
	votes := map[int]map[int]int{} // entree -> place -> tirs
	for _, f := range fire {
		p := pp.places[f.FilmIndex]
		if p == nil {
			continue
		}
		fr := frameDInstant(pp.in.horloge, f.TimestampUS)
		if pp.tireurDeLaTable(f.FilmIndex, fr) {
			continue
		}
		if c, ok := pp.arrivantUnique(candidats, p, fr); ok {
			if votes[c] == nil {
				votes[c] = map[int]int{}
			}
			votes[c][p.index]++
		}
	}
	for _, c := range candidats {
		v := votes[c]
		if len(v) == 0 {
			continue
		}
		p := pp.places[placeUnique(v)]
		if len(v) > 1 || !pp.libre(p, c) || !memeEquipe(p, pp.occ.parEntree[c].equipe) {
			contestes++
			continue
		}
		pp.asseoir(c, p, SeatSourceTirs)
		lues++
	}
	return lues, contestes, false
}

// placeUnique rend la place d'un vote unanime (la plus petite quand ils ne le sont pas — l'appelant
// les compte alors contestes avant de s'en servir).
func placeUnique(v map[int]int) int {
	idx := make([]int, 0, len(v))
	for p := range v {
		idx = append(idx, p)
	}
	sort.Ints(idx)
	return idx[0]
}

// indexDeTireurTronque dit que le film porte une place ou un index que l'index de tireur persiste
// ne distingue pas.
func (pp *poseDesPlaces) indexDeTireurTronque() bool {
	for _, idx := range pp.ordre {
		if idx >= placesLisiblesParLesTirs {
			return true
		}
	}
	for _, e := range pp.roster {
		if e.FilmIndex >= placesLisiblesParLesTirs {
			return true
		}
	}
	return false
}

// tireurDeLaTable dit qu'une entree dont l'index EST cette place est affichee a cette frame : le
// tir est le sien.
func (pp *poseDesPlaces) tireurDeLaTable(place, fr int) bool {
	for i, e := range pp.roster {
		if e.FilmIndex == place && couvre(pp.occ.parEntree[i].presence, fr, true) {
			return true
		}
	}
	return false
}

// arrivantUnique rend l'UNIQUE arrivant present (presence certaine) a cette frame dont l'equipe ne
// contredit pas celle de la place.
func (pp *poseDesPlaces) arrivantUnique(candidats []int, p *placeDeLaTable, fr int) (int, bool) {
	trouve := -1
	for _, c := range candidats {
		if !couvre(pp.occ.parEntree[c].presence, fr, false) || !memeEquipe(p, pp.occ.parEntree[c].equipe) {
			continue
		}
		if trouve >= 0 {
			return 0, false
		}
		trouve = c
	}
	return trouve, trouve >= 0
}

// couvre dit si une presence couvre une frame — jusqu'a `aMax` (l'affichage) ou jusqu'a `a` (le
// certain).
func couvre(ivs []intervalleDePresence, fr int, affichage bool) bool {
	for _, iv := range ivs {
		fin := iv.a
		if affichage {
			fin = iv.aMax
		}
		if fr >= iv.de && fr <= fin {
			return true
		}
	}
	return false
}

// mesureDeLAffichage : ce que [poseDesPlaces.mesurerLAffichage] rend (cf. SeatCoverage).
type mesureDeLAffichage struct {
	occupantsMax, depassements, capacite, placesEnTrop, sansEquipe int
}

// mesurerLAffichage mesure ce que le web va dessiner, CONTRE LA CAPACITE ESTIMEE de chaque equipe
// et non contre les places que la pose a attribuees (revue M2-R6 : ce plafond-la etait circulaire,
// il rendait 0 quelle que soit la pose). Rend le plus grand nombre d'entrees affichees a une meme
// frame ; les couples (frame, equipe) ou une equipe affiche plus d'occupants que sa capacite —
// entrees sans place (`index`) comprises ; la plus grande capacite ; les places affichees au-dela
// de la capacite de leur equipe ; et les entrees presentes sans equipe, que rien ne plafonne.
func (pp *poseDesPlaces) mesurerLAffichage() mesureDeLAffichage {
	var m mesureDeLAffichage
	tous := []borneDAffichage{}
	parEquipe := map[int][]borneDAffichage{}
	placesParEquipe := map[int]map[int]bool{}
	for i := range pp.roster {
		ivs := pp.occ.parEntree[i].presence
		t := pp.occ.parEntree[i].equipe
		if t == nil && len(ivs) > 0 {
			m.sansEquipe++
		}
		for _, iv := range ivs {
			b := []borneDAffichage{{iv.de, +1}, {iv.aMax + 1, -1}}
			tous = append(tous, b...)
			if t != nil {
				parEquipe[*t] = append(parEquipe[*t], b...)
				if placesParEquipe[*t] == nil {
					placesParEquipe[*t] = map[int]bool{}
				}
				placesParEquipe[*t][pp.roster[i].Seat] = true
			}
		}
	}
	m.occupantsMax, _ = balayerLesBornes(tous, sansPlafond)
	if pp.places == nil {
		return m
	}
	for t, bornes := range parEquipe {
		c := pp.capaciteDe(t)
		m.capacite = max(m.capacite, c)
		_, au := balayerLesBornes(bornes, c)
		m.depassements += au
		m.placesEnTrop += max(0, len(placesParEquipe[t])-c)
	}
	return m
}

// borneDAffichage : l'ouverture (+1) ou la fermeture (-1) d'un affichage a une frame.
type borneDAffichage struct {
	f, d int
}

// sansPlafond : un plafond qu aucun compte n atteint, pour ne mesurer que le maximum.
const sansPlafond = 1 << 30

// balayerLesBornes rend le maximum d'affichages simultanes, et le nombre de frames ou il depasse
// `plafond`. LES FERMETURES AVANT LES OUVERTURES a egalite de frame : deux presences qui se
// touchent sans se recouvrir ne comptent pas pour deux — c'est exactement un relais.
func balayerLesBornes(bornes []borneDAffichage, plafond int) (maxi, au int) {
	sort.Slice(bornes, func(a, b int) bool {
		if bornes[a].f != bornes[b].f {
			return bornes[a].f < bornes[b].f
		}
		return bornes[a].d < bornes[b].d
	})
	n := 0
	for k, b := range bornes {
		n += b.d
		maxi = max(maxi, n)
		if n > plafond && k+1 < len(bornes) {
			au += bornes[k+1].f - b.f
		}
	}
	return maxi, au
}
