package killsource

// feed_couples_fiches.go — UNE FICHE DE KILL APPARTIENT AU KILL LE PLUS PROCHE DE SON TUEUR.
//
// La fenetre de lecture d un kill sans mort vaut 2,5 s de part et d autre ([tolMS]). Un joueur qui
// tue deux fois en 2 s voit donc, depuis son PREMIER kill, la fiche (kill-event 85) de son SECOND —
// et quand le premier kill est celui d un bot, dont le kill-event peut manquer, la seule fiche de la
// fenetre est celle de l autre kill. La prendre echangeait les deux instants : le premier kill
// publiait la victime humaine du second, le second retombait sur une mort de bot, et la mort de la
// victime humaine, restee sans couple, revenait en orpheline (une mort de trop).
//
// LA REGLE : une fiche se dispute entre les kills du MEME tueur qui attendent encore la leur (kills
// sans mort au meme instant, sans fiche consommee) et dont la fenetre la contient. Parmi eux,
//
//  1. si la mort de la victime qu elle nomme est au voisin immediat de certains de ces kills, seuls
//     ceux-la la disputent encore : une lecture qui serait une contradiction pour un kill (aucun
//     voisin ne porte cette mort) alors que la mort est voisine d un autre kill du meme tueur
//     appartient a cet autre kill ;
//  2. elle va ensuite au kill dont l instant est le plus proche du sien ; a egalite, elle reste
//     disputable.

// ficheDUnAutreKill : la fiche `rec`, qui nomme `nom` en victime, appartient-elle a un autre kill du
// tueur du kill `e` (cf. l en-tete) ?
func (res *resolveurDeCouples) ficheDUnAutreKill(e, rec int, nom string) bool {
	msFiche := res.recs[rec].ms
	tueur := res.kf.events[e].killer
	var prets, voisins []int
	for o := range res.kf.events {
		k := res.kf.events[o]
		if o != e && (res.aSaFiche[o] || k.killer != tueur || k.victim != "" || !dansLaFenetre(msFiche, k.timeMS)) {
			continue
		}
		prets = append(prets, o)
		if res.voisinPortantLaMort(o, nom) >= 0 {
			voisins = append(voisins, o)
		}
	}
	if len(voisins) > 0 {
		prets = voisins
	}
	plusProche := -1
	for _, o := range prets {
		if d := absInt(msFiche - res.kf.events[o].timeMS); plusProche < 0 || d < plusProche {
			plusProche = d
		}
	}
	for _, o := range prets {
		if o == e && absInt(msFiche-res.kf.events[o].timeMS) == plusProche {
			return false
		}
	}
	return true
}

// absInt : la valeur absolue d un ecart en millisecondes.
func absInt(x int) int {
	if x < 0 {
		return -x
	}
	return x
}
