package replayartifacts

// padtiers_prises.go — LES PRISES D ARME EN COURS DE VIE, LE NEGATIF DES EQUIPEMENTS DE DEPART.
//
// FICHIER A PART, ET PAS PAR GOUT DU DECOUPAGE : `padtiers.go` tenait deja 533 lignes (dette
// gelee par la baseline lint) et le seuil du depot est 500. Ce qui vit ici est une seule
// question, posee au film : « cette vie avait-elle deja ramasse une arme quand la grille
// d images-cles a publie son equipement ? »

import (
	"levelup/go-api/internal/analysis/weapontier"
	"levelup/go-api/internal/games/halo_infinite/film/replay"
)

// prisesPour rassemble les PRISES D ARME EN COURS DE VIE des trois canaux qui en portent une,
// et n en garde que ce qui est POSTERIEUR au debut de la vie.
//
// POURQUOI TROIS CANAUX, ET POURQUOI LEUR UNION. Aucun ne couvre seul le match : sur le temoin
// `b1ad85eb`, `pickups` ecrit 118 prises d arme, `weaponChanges` 29 prises et 6 echanges, et la
// chaine des objets au sol nomme 13 preneurs. Les trois disent la meme chose — « cette vie a
// acquis une arme a cet instant » — et l union est le negatif le plus large que le film serve.
// Elle ne sert qu a DISQUALIFIER une emission d equipement de depart : une prise de trop retire
// une vie de la mesure, elle n invente jamais une arme de base.
//
// LA DOTATION DE REAPPARITION EST ECARTEE ICI, ET C EST LA RAISON D ETRE DE CE FILTRE : le jeu
// remet les armes de depart en main a l apparition, et `pickups` l ecrit comme une prise datee
// du debut de la vie (temoin b1ad85eb : huit prises a t=0 pour les huit premieres vies). Sans
// ce filtre, les vies qui partent le plus proprement seraient les premieres ecartees.
func prisesPour(doc *replay.ReplayDocument) []weapontier.Take {
	debut := make(map[uint32]int, len(doc.Tracks))
	for i := range doc.Tracks {
		t := &doc.Tracks[i]
		if len(t.Points) == 0 {
			continue
		}
		if vu, ok := debut[t.Slot]; !ok || t.Points[0].T < vu {
			debut[t.Slot] = t.Points[0].T
		}
	}
	out := make([]weapontier.Take, 0, len(doc.Pickups)+len(doc.WeaponChanges))
	ajoute := func(slot uint32, instant int) {
		if d, ok := debut[slot]; ok && instant <= d {
			return
		}
		out = append(out, weapontier.Take{Slot: slot, T: instant})
	}
	for i := range doc.Pickups {
		if doc.Pickups[i].Kind == replay.PickupWeapon {
			ajoute(doc.Pickups[i].Slot, doc.Pickups[i].T)
		}
	}
	for i := range doc.WeaponChanges {
		if k := doc.WeaponChanges[i].Kind; k == replay.WeaponTaken || k == replay.WeaponSwapped {
			ajoute(doc.WeaponChanges[i].Slot, doc.WeaponChanges[i].T)
		}
	}
	for i := range doc.GroundWeapons {
		if g := &doc.GroundWeapons[i]; g.Picker >= 0 {
			ajoute(uint32(g.Picker), g.T1)
		}
	}
	return out
}
