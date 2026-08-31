package replay

// pad_pickup_dating.go — DATER LES OCCUPATIONS DE SOCLE AVEC L'ÉVÉNEMENT NATIF.
//
// LE PROBLÈME QU'IL RÈGLE. `padPickups` publiait « ce socle s'est vidé quelque part entre
// `tLow` et `tHigh` », un intervalle de vingt secondes, et `xuid` valait `null` PARTOUT. Ce
// n'était pas un oubli : le contrat de `PadPickup.XUID` porte la mesure qui l'a refusé
// (88,1 % en suivant le slot de vie, 79,7 % en suivant le joueur, contre >= 90 % exigé) et il
// nomme ce qui le lèverait — « un oracle plus RAPPROCHÉ que 20 s ».
//
// L'ÉVÉNEMENT NATIF EST CET ORACLE, et il ne fait aucune inférence : il est daté à la
// milliseconde et il porte son ramasseur (`512 + référence` = le slot, exact sur 32/32 paires
// de vérité terrain, deux films). Quand un ramassage natif de la MÊME FAMILLE tombe dans la
// fenêtre d'une occupation, on publie l'instant exact et le joueur.
//
// CE QU'ON NE FAIT PAS, ET C'EST LA RÈGLE : ON N'EFFACE RIEN. L'intervalle `[tLow, tHigh]`
// reste publié dans tous les cas — daté ou non. Une occupation que l'événement natif ne
// couvre pas garde exactement ce qu'elle avait avant ce lot ; le canal AJOUTE, il ne remplace
// pas. Le rappel du canal natif est une borne inférieure (il ne voit que les événements en
// tête de liste) : substituer serait échanger une donnée sûre contre une donnée partielle.
//
// AMBIGUÏTÉ : si PLUSIEURS ramassages natifs de la même famille tombent dans la même fenêtre,
// on ne date PAS. Deux joueurs ont pu prendre la même arme ailleurs sur la carte pendant ces
// vingt secondes, et rien dans l'événement ne dit de quel socle il vient (l'instance de
// l'objet n'est pas dans l'événement — hypothèse mesurée et réfutée). Choisir au hasard
// nommerait un ramasseur faux ; on s'abstient et on le compte.

// PadDatingStats dit ce que la datation a pu faire, et ce qu'elle n'a pas pu.
type PadDatingStats struct {
	// Occupations est le nombre d'occupations achevées examinées.
	Occupations int `json:"occupations"`
	// Dated est le nombre d'occupations dont l'instant exact a été publié.
	Dated int `json:"dated"`
	// Named est le nombre dont le RAMASSEUR a pu être nommé (sous-ensemble de Dated : un
	// ramassage natif peut être daté sans que le pont slot -> joueur nomme sa vie).
	Named int `json:"named"`
	// Ambiguous compte les fenêtres où PLUSIEURS ramassages natifs de la même famille
	// tombaient : on s'abstient plutôt que de nommer un ramasseur au hasard.
	Ambiguous int `json:"ambiguous"`
	// Uncovered compte les fenêtres qu'aucun ramassage natif ne couvre — elles gardent leur
	// intervalle, intact.
	Uncovered int `json:"uncovered"`
}

// datePadPickups pose l'instant exact et le ramasseur sur les occupations que l'événement
// natif couvre. Modifie `picks` en place et rend les compteurs.
//
// `pads` sert à retrouver la FAMILLE d'arme du socle : `PadPickup.Pad` est un index dans
// `pads`, et c'est la famille qui apparie une occupation à un ramassage natif.
func datePadPickups(pads []WeaponPad, picks []PadPickup, pickups []Pickup) PadDatingStats {
	st := PadDatingStats{Occupations: len(picks)}
	if len(picks) == 0 || len(pickups) == 0 || len(pads) == 0 {
		st.Uncovered = len(picks)
		return st
	}
	// Index par famille : une occupation ne s'apparie qu'à un ramassage de LA MÊME arme.
	byFamily := map[string][]Pickup{}
	for _, p := range pickups {
		if p.Kind != PickupWeapon {
			continue // un socle d'arme ne rend pas de l'équipement
		}
		byFamily[p.W] = append(byFamily[p.W], p)
	}
	for i := range picks {
		k := &picks[i]
		if k.Pad < 0 || k.Pad >= len(pads) {
			st.Uncovered++
			continue
		}
		var hits []Pickup
		for _, p := range byFamily[pads[k.Pad].Weapon] {
			if p.T >= k.TLow && p.T <= k.THigh {
				hits = append(hits, p)
			}
		}
		switch len(hits) {
		case 0:
			st.Uncovered++
		case 1:
			t := hits[0].T
			k.T = &t
			st.Dated++
			if hits[0].XUID != "" {
				x := hits[0].XUID
				k.XUID = &x
				st.Named++
			}
		default:
			// Plusieurs candidats : on s'abstient. Voir l'en-tête de ce fichier.
			st.Ambiguous++
		}
	}
	return st
}
