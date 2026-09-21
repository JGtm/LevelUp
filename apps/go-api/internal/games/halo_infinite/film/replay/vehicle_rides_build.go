package replay

// vehicle_rides_build.go — L ASSEMBLAGE DES EPISODES D OCCUPATION : la LECTURE d abord, les deux
// replis ensuite, et la regle qui les departage.
//
// SORTI DE `vehicle_rides.go` par DEPLACEMENT PUR le 2026-09-21 (lot 5.10) : le fichier passait
// le seuil de 500 lignes en accueillant la primaute de la lecture, et la construction y melait
// deja trois sujets. Le corps est DECOUPE en trois etapes nommees, une par source, pour tenir le
// seuil de 80 lignes par fonction — aucune regle n a change dans le deplacement.
//
// L ORDRE EST LE RESULTAT DU LOT, PAS UNE COMMODITE :
//
//	1. LA LECTURE (`i10 object-parent-state`) — le film ECRIT la montee a bord et son siege ;
//	2. LA MACHINE D ETATS PAR OCCUPANT (evenements d embarquement / de sortie) ;
//	3. LE TROU du flux de position, pour ce qu aucun evenement n atteste.
//
// Les etapes 2 et 3 sont des REPLIS : un episode qu elles produisent n est publie que s il ne
// CONTREDIT aucune lecture de la meme vie (cf. `vehicleFilmRides.contredit`).

import (
	"sort"

	"levelup/go-api/internal/games/halo_infinite/film/internal/grammar"
	"levelup/go-api/internal/games/halo_infinite/film/types"
)

// vehicleRideBuild porte l etat de l assemblage — les entrees, la LECTURE qui fait autorite, la
// sortie et le bilan. Une structure plutot que sept parametres promenes d une etape a l autre.
type vehicleRideBuild struct {
	in  vehicleRideInputs
	lus vehicleFilmRides
	out map[types.EquipmentLifeKey][]VehicleRide
	st  vehicleRideStats
	// boards / exits : les evenements ranges par occupant ; bySlot le nuage des bipedes.
	boards, exits map[uint32][]types.VehicleEvent
	bySlot        map[uint32][]grammar.BipedPosition
}

// buildVehicleRides rend les episodes d occupation par vie de vehicule, et le bilan de leur
// rattachement. PUR.
func buildVehicleRides(
	in vehicleRideInputs,
) (map[types.EquipmentLifeKey][]VehicleRide, vehicleRideStats) {
	if in.clock.step == 0 || len(in.lives) == 0 {
		return nil, vehicleRideStats{}
	}
	b := &vehicleRideBuild{in: in, out: map[types.EquipmentLifeKey][]VehicleRide{}}
	b.boards, b.exits = vehicleEventsByOccupant(in.events)
	b.bySlot = vehiclePositionsBySlot(in.bipeds)
	b.parLaLecture()
	b.parLesTrous(b.parLesEvenements())
	// LE SIEGE D UN EPISODE DE REPLI SE POSE ICI, ET EN UN SEUL ENDROIT : les episodes LUS
	// portent deja le leur (il sort de la meme lecture que leur montee a bord).
	b.st.sieges = assignVehicleSeats(b.out, in.occupancy, in.clock)
	b.trier()
	return b.out, b.st
}

// parLaLecture pose les episodes que le film ECRIT. Ils ne sont jamais ecartes.
func (b *vehicleRideBuild) parLaLecture() {
	lus, tally := buildVehicleFilmRides(b.in)
	b.lus, b.st.film = lus, tally
	for k, v := range lus.rides {
		b.out[k] = append(b.out[k], v...)
	}
}

// parLesEvenements deroule la machine d etats par occupant et rend les episodes RETENUS — ceux
// qui ont trouve leur vehicule, publies ou non : ce sont eux qui ferment la porte du trou.
func (b *vehicleRideBuild) parLesEvenements() []vehicleEpisode {
	var kept []vehicleEpisode
	for _, ep := range vehicleEventEpisodes(b.boards, b.exits, b.bySlot) {
		b.st.episodes++
		if ep.vehValid {
			b.st.nommes++
		}
		key, r, resolved, ok := vehicleRideFromEpisode(ep, b.bySlot, b.in)
		if !ok {
			b.st.perdus++
			continue
		}
		b.tallyResolution(resolved.resolvedBy)
		kept = append(kept, resolved)
		if b.lus.contredit(key, r) {
			b.st.ecartes++
			continue
		}
		vehicleTallyBorders(&b.st, resolved.borders)
		b.out[key] = append(b.out[key], r)
	}
	return kept
}

// parLesTrous ajoute les episodes qu aucun evenement n atteste.
//
// SEULS LES EPISODES RETENUS ferment la porte du repli. Un episode d evenement dont le vehicule
// n a pas pu etre resolu ne doit RIEN supprimer : sinon la machine d etats retirerait un episode
// que le trou, lui, savait rattacher (mesure du 2026-09-03 : 12 -> 11 sur `0d76e8f1` avec la
// regle naive).
func (b *vehicleRideBuild) parLesTrous(kept []vehicleEpisode) {
	for _, g := range vehicleGaps(b.in.bipeds) {
		if vehicleEpisodeCovers(kept, g) {
			continue
		}
		vs, ok := vehicleNearestTo(g.last, b.in.vehBySlot)
		if !ok {
			continue
		}
		key, ok := vehicleLifeAt(b.in.lives, vs, g.startUS)
		if !ok {
			continue
		}
		r := vehicleRideOf(g, b.boards[g.slot], b.exits[g.slot], b.in)
		if b.lus.contredit(key, r) {
			b.st.ecartes++
			continue
		}
		vehicleTallyBorders(&b.st, vehicleRideBorders(g, b.boards[g.slot], b.exits[g.slot]))
		b.out[key] = append(b.out[key], r)
		b.st.repli++
	}
}

// tallyResolution range la voie par laquelle un episode d evenement a trouve son vehicule.
func (b *vehicleRideBuild) tallyResolution(par vehicleResolvedBy) {
	switch par {
	case vehicleResolvedByEvent:
		b.st.parEvenement++
	case vehicleResolvedByEventNearest:
		b.st.parEvenementProche++
	default:
		b.st.parGeometrie++
	}
}

// trier ordonne les episodes de chaque vie par instant de debut puis par slot d occupant.
func (b *vehicleRideBuild) trier() {
	for k := range b.out {
		v := b.out[k]
		sort.SliceStable(v, func(i, j int) bool {
			if v[i].T0 != v[j].T0 {
				return v[i].T0 < v[j].T0
			}
			return v[i].Slot < v[j].Slot
		})
	}
}
