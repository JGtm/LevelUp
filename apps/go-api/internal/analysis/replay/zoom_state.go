package replay

// zoom_state.go — LA RECONSTRUCTION DE L'ETAT DE LUNETTE, ET POURQUOI ELLE NE REPOSE PAS SUR
// LES SEULS EVENEMENTS DE SORTIE.
//
// # L'OBJECTION QUI A FAIT CE FICHIER
//
// Reconstruire « a la lunette » en appariant chaque entree a sa sortie suppose qu'on lit TOUTES
// les sorties. Or on dezoome de beaucoup de facons : en mourant, en subissant des degats, en
// changeant d'arme, en lancant une grenade. Faire dependre l'etat de la capture exhaustive de
// tous ces evenements est fragile par construction — un decodeur qui n'a qu'une seule source de
// fermeture se trompe des qu'elle manque.
//
// MESURE QUI LA FONDE (film de reference, instrument `TestViseeZoomEntreesOrphelines`) :
// 180 entrees, 144 sorties lues, 139 periodes completes. Sur les 41 entrees orphelines, celles
// dont le joueur MEURT vite sont trois a cinq fois surrepresentees (32 % a moins de 5 s contre
// 10 % pour les entrees normalement fermees) : la mort explique environ un tiers des sorties
// manquantes, et les deux tiers restants vivent ailleurs dans le flux — vraisemblablement en
// deuxieme position d'une liste d'evenements, dans le paquet du degat qui force le dezoom.
//
// # LE MODELE RETENU : PLUSIEURS CAUSES DE FERMETURE, PAR ORDRE DE FIABILITE
//
//  1. L'EVENEMENT DE SORTIE, quand il est lu. Seule source qui donne l'instant EXACT.
//  2. LA FIN DE VIE DU SLOT. Un joueur mort n'est plus a la lunette, et sa trajectoire s'arrete :
//     l'information est deja dans le document, elle ne coute rien et elle est certaine.
//  3. UNE NOUVELLE ENTREE du meme slot. Le moteur ne peut pas entrer deux fois sans etre sorti :
//     la periode precedente est donc close, meme si on n'a pas lu quand.
//  4. LE PLAFOND DE MAINTIEN, en dernier recours. Il ne ferme rien : il fait CESSER D'AFFIRMER.
//
// Aucune de ces causes n'invente d'etat. Les trois premieres sont des faits ; la quatrieme est
// un aveu d'ignorance borne dans le temps, ce qui est le bon defaut quand la donnee manque.
//
// # CE QUE CE MODELE NE PRETEND PAS
//
// Il ne simule pas la logique de jeu. Si le moteur dezoome un joueur pour une raison qu'aucun
// evenement lu ne porte, et qui ne coincide ni avec sa mort ni avec une entree suivante, la
// periode restera ouverte jusqu'au plafond — trop longue de quelques secondes, jamais de
// plusieurs vies. La correction definitive n'est pas ici : c'est la marche de la liste
// d'evenements ENTIERE, qui rendra le plafond inutile (cf. PLAN_PERCER_TRAME_FILM_2026-08-30.md).

import (
	"sort"

	"levelup/go-api/internal/analysis/filmdec"
)

// zoomPeriode est un intervalle « a la lunette » reconstruit pour un slot, en microsecondes.
type zoomPeriode struct {
	debut, fin uint64
	niveau     int
}

// buildScopedLookup reconstruit les periodes de lunette et rend la fonction de consultation
// posee dans `Options.Scoped` : pour un slot et un instant, le palier en vigueur (0 = pas a la
// lunette).
//
// `lives` fournit la cause de fermeture n°2 (la fin de vie). Il peut etre vide : le modele
// degrade alors proprement sur les trois autres causes.
func buildScopedLookup(
	evts []filmdec.ZoomEvent, lives []lifeSpan, holdUS uint64,
) func(uint32, uint64) int {
	fins := finsDeVieParSlot(lives)
	parSlot := map[uint32][]zoomPeriode{}
	ouvertes := map[uint32]zoomPeriode{}

	clore := func(slot uint32, p zoomPeriode, fin uint64) {
		if fin > p.debut {
			p.fin = fin
			parSlot[slot] = append(parSlot[slot], p)
		}
	}
	for _, e := range evts {
		encours, ouverte := ouvertes[e.Slot]
		if e.Scoped() {
			if ouverte { // cause 3 : une nouvelle entree clot la precedente
				fin := borneFin(encours.debut, e.TimestampUS, finApres(fins[e.Slot], encours.debut), holdUS)
				clore(e.Slot, encours, fin)
			}
			ouvertes[e.Slot] = zoomPeriode{debut: e.TimestampUS, niveau: e.Level}
			continue
		}
		if ouverte { // cause 1 : l'evenement de sortie, l'instant exact
			clore(e.Slot, encours, e.TimestampUS)
			delete(ouvertes, e.Slot)
		}
	}
	for slot, p := range ouvertes { // causes 2 et 4 pour ce qui reste ouvert
		clore(slot, p, borneFin(p.debut, 0, finApres(fins[slot], p.debut), holdUS))
	}
	for slot, ps := range parSlot {
		sort.Slice(ps, func(i, j int) bool { return ps[i].debut < ps[j].debut })
		parSlot[slot] = ps
	}
	return func(slot uint32, ts uint64) int {
		ps := parSlot[slot]
		i := sort.Search(len(ps), func(i int) bool { return ps[i].fin >= ts })
		if i < len(ps) && ts >= ps[i].debut {
			return ps[i].niveau
		}
		return 0
	}
}

// borneFin choisit la fin d'une periode restee ouverte : la plus PROCHE des bornes disponibles —
// la cause suivante lue (`suivant`, 0 si aucune), la fin de vie du slot (`finDeVie`, 0 si
// inconnue), et le plafond. Prendre la plus proche, c'est refuser d'affirmer au-dela de ce que
// la premiere cause connue autorise.
func borneFin(debut, suivant, finDeVie, holdUS uint64) uint64 {
	fin := debut + holdUS
	if suivant > debut && suivant < fin {
		fin = suivant
	}
	if finDeVie > debut && finDeVie < fin {
		fin = finDeVie
	}
	return fin
}

// finsDeVieParSlot rend, par slot, les fins de vie TRIEES. Un slot est reutilise apres
// reapparition : garder une seule fin melangerait deux vies, d'ou la liste.
func finsDeVieParSlot(lives []lifeSpan) map[uint32][]uint64 {
	out := map[uint32][]uint64{}
	for _, l := range lives {
		out[l.slot] = append(out[l.slot], uint64(l.to))
	}
	for s, fs := range out {
		sort.Slice(fs, func(i, j int) bool { return fs[i] < fs[j] })
		out[s] = fs
	}
	return out
}

// finApres rend la premiere fin de vie a partir de `ts`, ou 0 si aucune — c'est la fin de la vie
// EN COURS a cet instant, la seule qui puisse fermer une periode ouverte alors.
func finApres(fins []uint64, ts uint64) uint64 {
	i := sort.Search(len(fins), func(i int) bool { return fins[i] >= ts })
	if i < len(fins) {
		return fins[i]
	}
	return 0
}
