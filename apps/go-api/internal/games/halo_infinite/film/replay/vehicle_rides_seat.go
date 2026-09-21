package replay

// vehicle_rides_seat.go — LE SIEGE D UN EPISODE EST LU DANS LE FILM (lot 5.10).
//
// CE QUI DECIDAIT AVANT, ET POURQUOI C ETAIT FAUX. Le siege venait du champ `R(6)` de
// l EVENEMENT d embarquement / de sortie (`types.VehicleEvent.Seat`). La mesure du lot 5.5 (D1)
// l a REFUTE sur pieces : sur les 100 tirs de tourelle de `4f77afc1`, la ventilation des
// episodes qui couvrent l instant du tir trouve 153 occurrences de `seat = 0` — donc PLUSIEURS
// episodes revendiquent le siege du conducteur sur le MEME vehicule a la MEME image — 16 sieges
// muets, et JAMAIS de siege 1 ni 2, alors que le tourelleur d un Warthog est un passager. Un
// champ qui ne departage pas les occupants ne publie pas un siege : il publie un zero.
//
// CE QUI DECIDE MAINTENANT : `object-parent-state` (`i10`), le composant par lequel le film
// ATTACHE une unite a une autre. Son ecrivain (`FUN_140c1e4d0`, relu le 2026-09-21) ecrit le
// handle du parent en +0x274 et un entier de SIX bits en +0x3a0, tous deux effaces par une
// SENTINELLE sur la branche libre. Mesure sur `4f77afc1` (carte installee, oracle de contenu
// tenu) : 48 lectures attachees nomment un slot `ti=40`, 43 portent le champ de queue, et 42 de
// ces 43 valent 0, 1 ou 2 — conducteur, passager, tourelleur. Detail :
// `grammar/vehicle_occupancy_march.go`.
//
// CE QUE CE FICHIER NE FAIT PAS : construire des episodes. Le canal est une TRANSITION du chemin
// delta (48 montees nommees quand le document publie 86 episodes) ; les episodes restent bornes
// par le trou de position et les evenements. Un episode qu aucune lecture ne couvre sort donc
// SANS siege (`seat` absent) plutot qu avec un zero par defaut — et la couverture le compte
// (`VehicleCoverage.RidesWithSeat`).
//
// PUR : aucune I/O, aucune lecture de film.

import (
	"levelup/go-api/internal/games/halo_infinite/film/types"
)

// vehicleSeatTolMS est la tolerance appliquee AVANT le debut d un episode pour y rattacher la
// montee a bord qui l ouvre. Meme valeur et meme raison que `vehicleEventTolMS` : deux paquets
// delta valent ~1 s, et le biais joue CONTRE le rattachement quand il manque.
const vehicleSeatTolMS = vehicleEventTolMS

// assignVehicleSeats pose, sur chaque episode publie, le siege que le film ECRIT pour lui, et
// rend le nombre d episodes servis.
//
// LA CLE EST LE COUPLE (OCCUPANT, VEHICULE) : la lecture porte le slot du bipede et le slot de
// son parent ; l episode porte le slot de son occupant et la vie du vehicule auquel il est
// rattache. La fenetre departage les vies successives d un meme slot.
//
// LA GENERATION DU PARENT N EST PAS EXIGEE : le handle en porte deux bits, la vie aussi, mais
// c est la FENETRE qui rattache deja l episode a sa vie — ajouter une egalite de generation ne
// departagerait rien qu elle n ait departage, et perdrait les lectures dont le handle a reboucle.
func assignVehicleSeats(
	rides map[types.EquipmentLifeKey][]VehicleRide, occ []types.VehicleOccupancy,
	clock replayClock,
) int {
	if len(rides) == 0 || len(occ) == 0 || clock.step == 0 {
		return 0
	}
	tol := int(vehicleSeatTolMS / clock.step) //nolint:gosec // pas de grille, quelques frames
	var servis int
	for key, list := range rides {
		for i := range list {
			r := &list[i]
			s, ok := vehicleSeatFor(occ, r.Slot, key.Slot, r.T0-tol, r.T1, clock)
			if !ok {
				continue
			}
			r.Seat = &s
			servis++
		}
	}
	return servis
}

// vehicleSeatFor rend le siege lu pour un occupant dans un vehicule entre deux frames.
//
// LA PREMIERE MONTEE DE LA FENETRE GAGNE : un episode commence par un embarquement, et si le
// film en ecrit deux (l occupant change de siege sans descendre), c est celui qui l a ouvert qui
// decrit l episode publie.
func vehicleSeatFor(
	occ []types.VehicleOccupancy, occupant, vehicule uint32, t0, t1 int, clock replayClock,
) (int, bool) {
	for _, o := range occ {
		if !o.Attached || !o.HasSeat || o.Slot != occupant || o.ParentSlot != vehicule {
			continue
		}
		f := clock.frame(o.TimestampUS)
		if f < t0 || f > t1 {
			continue
		}
		return int(o.Seat), true //nolint:gosec // R(6) : 0..63
	}
	return 0, false
}
