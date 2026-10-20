package replay

// vehicle_shots_unit.go — UN TIR DONT L UNITE TIREUSE EST UN VEHICULE SE POSE SUR LUI (lot M4b.4 de
// la campagne « retours rejeu », 2026-09-24).
//
// LE FAIT. Le record 36 lu par la grammaire (lot M4b.2, `internal/grammar/fire_events.go`) porte sa
// REFERENCE 0 : l unite qui tire — le bipede, le vehicule ou la piece montee —, au meme gabarit de
// slot que les vies du document (`0x200 + index`). Tant qu elle n etait pas lue, un tir en vehicule
// ne trouvait son vehicule que par l EPISODE d occupation de son tireur : sans episode, il tombait.
//
// LA REGLE. Quand la reference 0 designe une vie de vehicule qui couvre l instant du tir, le tir se
// pose sur ELLE (sur son porteur pour une piece montee) ; l episode ne sert plus qu a nommer
// l occupant — le tireur du record quand l un de ses episodes est a bord, l occupant unique de la
// vie quand le record ne nomme pas de tireur. Sans occupant nommable, la porte d avant (l episode
// seul) reprend la main. Les tirs poses ainsi sont comptes (`shotsByUnit`), et parmi eux ceux
// qu AUCUN episode du tireur ne couvrait (`shotsByUnitNoRide`) : c est ce que la reference apporte.
//
// PUR : aucune I/O.

import (
	"levelup/go-api/internal/games/halo_infinite/film/internal/grammar"
)

// vehicleLivesBySlot indexe les vies de vehicule par slot.
func vehicleLivesBySlot(tracks []VehicleTrack) map[uint32][]int {
	out := map[uint32][]int{}
	for i, tr := range tracks {
		out[tr.Slot] = append(out[tr.Slot], i)
	}
	return out
}

// vieDeLUnite rend la vie de vehicule que la reference 0 designe a cette frame : le slot de
// l unite, une vie publiee qui couvre la frame — UNIQUE.
func (b vehicleShotBoard) vieDeLUnite(u grammar.UnitRef, fr int) (int, bool) {
	if !u.Present {
		return 0, false
	}
	trouve, n := -1, 0
	for _, i := range b.parSlot[u.Slot] {
		tr := b.tracks[i]
		if fr >= tr.T0 && fr <= max(tr.T1, tr.T1Max) {
			trouve = i
			n++
		}
	}
	return trouve, n == 1
}

// memeVehicule dit si l episode `rr` est a bord de la vie `at` : sur elle, sur son porteur, ou
// sur une piece qu elle porte.
func (b vehicleShotBoard) memeVehicule(rr vehicleShotRide, at int) bool {
	if rr.track == at {
		return true
	}
	ref := VehicleLifeRef{Slot: b.tracks[at].Slot, Gen: b.tracks[at].Gen}
	if rr.ride.Turret != nil && *rr.ride.Turret == ref {
		return true
	}
	if c := b.tracks[rr.track].Carrier; c != nil && *c == ref {
		return true
	}
	c := b.tracks[at].Carrier
	return c != nil && *c == VehicleLifeRef{Slot: b.tracks[rr.track].Slot, Gen: b.tracks[rr.track].Gen}
}

// occupantUnique rend l occupant unique de la vie `at` a la frame (episodes de la vie elle-meme).
func (b vehicleShotBoard) occupantUnique(at, fr int) (vehicleShotRide, bool) {
	var pick vehicleShotRide
	n := 0
	for _, r := range b.tracks[at].Rides {
		if fr >= r.T0 && fr <= r.T1 {
			pick, n = vehicleShotRide{track: at, ride: r}, n+1
		}
	}
	return pick, n == 1
}

// shotOfUnit pose un orphelin sur le vehicule de sa reference 0, quand elle en designe un et que
// l occupant se nomme (cf. l en-tete). Le quatrieme retour dit que la porte s est appliquee ;
// le cinquieme, qu aucun episode du tireur ne couvrait le tir.
func (b vehicleShotBoard) shotOfUnit(
	o orphanShot, cand []vehicleShotRide, fr int,
) (Shot, vehicleShotVerdict, bool, bool, bool) {
	at, ok := b.vieDeLUnite(o.ev.Unit, fr)
	if !ok {
		return Shot{}, vehicleShotNoRide, false, false, false
	}
	var pick vehicleShotRide
	trouve, sansEpisode := false, false
	for _, c := range cand {
		if b.memeVehicule(c, at) {
			pick, trouve = c, true
			break
		}
	}
	if !trouve && !o.ev.HasShooter {
		pick, trouve = b.occupantUnique(at, fr)
		sansEpisode = trouve
	}
	if !trouve {
		return Shot{}, vehicleShotNoRide, false, false, false
	}
	pick.track = at
	s, v, onCarrier := b.poser(o, pick, fr)
	return s, v, onCarrier, true, sansEpisode
}
