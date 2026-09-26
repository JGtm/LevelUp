package replay

// vehicle_rides_film.go — L EPISODE D OCCUPATION LU DANS LE FILM (lot 5.10, arbitrage du pilote
// du 2026-09-21 : « la source primaire devient la LECTURE »).
//
// CE QUE LE FILM ECRIT, ET C EST UNE TRANSITION. `object-parent-state` (`i10`) porte, sur le
// record du BIPEDE, le handle de son parent et son siege (grammaire relue chez l ecrivain,
// `FUN_140c1e4d0` ; detail dans `grammar/vehicle_occupancy_march.go`). Le chemin delta ne
// transmettant que ce qui CHANGE, une lecture ATTACHEE est une MONTEE A BORD et la lecture
// suivante du meme occupant la FERME — branche libre (il descend) ou autre parent (il change de
// vehicule).
//
// TROIS FERMETURES, DANS CET ORDRE, ET AUCUNE N EST UNE SUPPOSITION :
//
//	1. la lecture d `i10` SUIVANTE du meme occupant — le film dit lui-meme que l attachement a
//	   cesse ;
//	2. la REAPPARITION de l occupant dans le flux de position — un bipede embarque ne replique
//	   plus sa trajectoire (acquis V1/V4 : 1 347 instants, aucun couple sous 3 m plus de 1,6 s),
//	   donc son premier point posterieur BORNE l episode ;
//	3. la fin de la VIE du vehicule (`hiUS`), quand ni l une ni l autre n arrive.
//
// LA PRIMAUTE, ET CE QU ELLE COUTE. Un episode d HEURISTIQUE (trou de position + evenements,
// `vehicle_rides.go` / `vehicle_rides_events.go`) n est publie que s il ne CONTREDIT aucune
// lecture de la meme vie de vehicule : ni chevauchement d un episode lu, ni occupant absent des
// occupants que le film a nommes pour cette vie. Le prix est ecrit : une vie dont le film n a lu
// QU UN siege perd les episodes heuristiques de ses AUTRES sieges. Il est paye sciemment — c est
// le defaut que le verdict Theater du 2026-09-19 a nomme (un occupant FAUX publie dans le
// Razorback `776/1`), et le compte des episodes ecartes voyage au journal.
//
// PUR : aucune I/O, aucune lecture de film.

import (
	"sort"

	"levelup/go-api/internal/games/halo_infinite/film/internal/grammar"
	"levelup/go-api/internal/games/halo_infinite/film/types"
)

// vehicleFilmRides porte ce que la LECTURE rend : les episodes par vie, les occupants que le
// film nomme par vie, et les fenetres publiees (en frames) qui servent au test de contradiction.
type vehicleFilmRides struct {
	rides     map[types.EquipmentLifeKey][]VehicleRide
	occupants map[types.EquipmentLifeKey]map[uint32]bool
	fenetres  map[types.EquipmentLifeKey][][2]int
}

// vehicleFilmTally compte ce que la lecture a rendu et ce qu elle a du ecarter.
type vehicleFilmTally struct {
	// lectures : montees a bord ATTACHEES rendues par la marche ; nommees celles dont le parent
	// tombe sur une vie de vehicule recensee.
	lectures, nommees int
	// horsVie : le parent ne tombe dans la fenetre d AUCUNE vie ; nonDessinable : la vie existe
	// mais le calque ne saura pas la dessiner (meme predicat que `vehicleTrackOf`).
	horsVie, nonDessinable int
	// publies : episodes lus effectivement publies.
	publies int
	// fermeturePar* ventile la borne de fin retenue.
	fermetureParLecture, fermetureParRetour, fermetureParVie int
}

// buildVehicleFilmRides construit les episodes LUS. PUR.
func buildVehicleFilmRides(in vehicleRideInputs) (vehicleFilmRides, vehicleFilmTally) {
	out := vehicleFilmRides{
		rides:     map[types.EquipmentLifeKey][]VehicleRide{},
		occupants: map[types.EquipmentLifeKey]map[uint32]bool{},
		fenetres:  map[types.EquipmentLifeKey][][2]int{},
	}
	var t vehicleFilmTally
	if len(in.occupancy) == 0 || len(in.lives) == 0 || in.clock.step == 0 {
		return out, t
	}
	lectures := vehicleOccupancyBySlot(in.occupancy)
	pts := vehiclePositionsBySlot(in.bipeds)
	for _, o := range in.occupancy {
		if !o.Attached {
			continue
		}
		t.lectures++
		key, ok := vehicleLifeAt(in.lives, o.ParentSlot, o.TimestampUS)
		if !ok {
			t.horsVie++
			continue
		}
		t.nommees++
		if !in.drawable[key] {
			t.nonDessinable++
			continue
		}
		endUS, par := vehicleFilmRideEnd(o, lectures[o.Slot], pts[o.Slot], in.lives, key)
		vehicleFilmTallyEnd(&t, par)
		out.ajouter(key, vehicleFilmRideOf(o, endUS, in))
		t.publies++
	}
	out.trier()
	return out, t
}

// ajouter range un episode lu dans les trois index de la structure.
func (f vehicleFilmRides) ajouter(key types.EquipmentLifeKey, r VehicleRide) {
	f.rides[key] = append(f.rides[key], r)
	if f.occupants[key] == nil {
		f.occupants[key] = map[uint32]bool{}
	}
	f.occupants[key][r.Slot] = true
	f.fenetres[key] = append(f.fenetres[key], [2]int{r.T0, r.T1})
}

// trier ordonne les episodes de chaque vie par instant de debut puis par slot.
func (f vehicleFilmRides) trier() {
	for k := range f.rides {
		v := f.rides[k]
		sort.SliceStable(v, func(i, j int) bool {
			if v[i].T0 != v[j].T0 {
				return v[i].T0 < v[j].T0
			}
			return v[i].Slot < v[j].Slot
		})
	}
}

// contredit dit si un episode d heuristique est en desaccord avec la LECTURE de cette vie.
//
// DEUX FORMES DE DESACCORD, et la seconde est celle que le verdict Theater a nommee :
//   - CHEVAUCHEMENT : la fenetre de l episode recouvre celle d un episode lu ;
//   - OCCUPANT NON NOMME : le film a lu des montees a bord pour cette vie, et cet occupant n en
//     fait pas partie.
//
// Une vie dont le film n a RIEN lu n est jamais contredite : l absence de lecture n est pas une
// absence d occupant.
func (f vehicleFilmRides) contredit(key types.EquipmentLifeKey, r VehicleRide) bool {
	occ := f.occupants[key]
	if len(occ) == 0 {
		return false
	}
	if !occ[r.Slot] {
		return true
	}
	for _, w := range f.fenetres[key] {
		if r.T0 <= w[1] && w[0] <= r.T1 {
			return true
		}
	}
	return false
}

// vehicleFilmRideOf assemble l episode publie d une montee a bord lue.
func vehicleFilmRideOf(o types.VehicleOccupancy, endUS uint64, in vehicleRideInputs) VehicleRide {
	r := VehicleRide{Slot: o.Slot, Src: VehicleRideSrcFilm}
	r.T0, r.T1 = in.clock.frame(o.TimestampUS), in.clock.frame(endUS)
	if r.T1 < r.T0 {
		r.T1 = r.T0
	}
	if o.HasSeat {
		s := int(o.Seat) //nolint:gosec // R(6) : 0..63
		r.Seat = &s
	}
	r.Aim = vehicleRideAimOf(in.aimBySlot[o.Slot], o.TimestampUS, endUS, in.clock)
	r.XUID = in.reg.XUIDAt(o.Slot, o.TimestampUS)
	return r
}

// vehicleFilmRideEnd rend la borne de fin d un episode lu et LA CAUSE qui l a posee.
func vehicleFilmRideEnd(
	o types.VehicleOccupancy, lectures []types.VehicleOccupancy,
	pts []grammar.BipedPosition, lives []vehicleLife, key types.EquipmentLifeKey,
) (uint64, int) {
	fin, par := vehicleLifeEndUS(lives, key), vehicleFilmEndByLife
	if at, ok := vehicleNextOccupancy(lectures, o.TimestampUS); ok && at < fin {
		fin, par = at, vehicleFilmEndByRead
	}
	if at, ok := vehicleFirstPointAfter(pts, o.TimestampUS); ok && at < fin {
		fin, par = at, vehicleFilmEndByReturn
	}
	return fin, par
}

// Les trois causes de fermeture d un episode lu.
const (
	vehicleFilmEndByLife = iota
	vehicleFilmEndByRead
	vehicleFilmEndByReturn
)

// vehicleFilmTallyEnd range la cause de fermeture dans le bilan.
func vehicleFilmTallyEnd(t *vehicleFilmTally, par int) {
	switch par {
	case vehicleFilmEndByRead:
		t.fermetureParLecture++
	case vehicleFilmEndByReturn:
		t.fermetureParRetour++
	default:
		t.fermetureParVie++
	}
}

// vehicleOccupancyBySlot indexe les lectures par slot d OCCUPANT, triees par instant.
func vehicleOccupancyBySlot(
	occ []types.VehicleOccupancy,
) map[uint32][]types.VehicleOccupancy {
	out := map[uint32][]types.VehicleOccupancy{}
	for _, o := range occ {
		out[o.Slot] = append(out[o.Slot], o)
	}
	for s := range out {
		v := out[s]
		sort.SliceStable(v, func(i, j int) bool { return v[i].TimestampUS < v[j].TimestampUS })
	}
	return out
}

// vehicleNextOccupancy rend l instant de la lecture d `i10` SUIVANTE du meme occupant.
func vehicleNextOccupancy(lectures []types.VehicleOccupancy, afterUS uint64) (uint64, bool) {
	i := sort.Search(len(lectures), func(k int) bool { return lectures[k].TimestampUS > afterUS })
	if i >= len(lectures) {
		return 0, false
	}
	return lectures[i].TimestampUS, true
}

// vehicleFirstPointAfter rend l instant du premier echantillon de position POSTERIEUR — la
// reapparition de l occupant, donc sa descente.
func vehicleFirstPointAfter(pts []grammar.BipedPosition, afterUS uint64) (uint64, bool) {
	i := sort.Search(len(pts), func(k int) bool { return pts[k].TimestampUS > afterUS })
	if i >= len(pts) {
		return 0, false
	}
	return pts[i].TimestampUS, true
}

// vehicleLifeEndUS rend la borne haute de la vie demandee.
func vehicleLifeEndUS(lives []vehicleLife, key types.EquipmentLifeKey) uint64 {
	for _, l := range lives {
		if l.key == key {
			return l.hiUS
		}
	}
	return 0
}
