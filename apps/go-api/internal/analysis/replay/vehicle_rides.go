package replay

// vehicle_rides.go — L EPISODE D OCCUPATION : qui est a bord de quel vehicule, de quand a quand.
//
// C EST L OBJET QUE LA MESURE A VALIDE, ET CE N EST PAS CELUI QU ON CHERCHAIT. Le lot V3 visait
// la DESTRUCTION datee par la mort du conducteur : REFUTEE sur 460 vies et 12 films (rapport
// `V3_DESTRUCTION_DATEE_2026-09-02.md`). Ce qui a passe ses gates est l EPISODE D OCCUPATION, et
// le rapport le nomme explicitement comme le livrable a produire (§ 6).
//
// TROIS SIGNAUX, UNE SEULE GRANDEUR — et chacun a ses chiffres :
//
//	LE TROU DE POSITION (le liant, primitive V1a.4 productionisee au lot V1). Un occupant
//	attache cesse de repliquer sa position monde : l embarquement se voit comme le DERNIER point
//	d un flux bipede, a moins de 1,5 m d un vehicule, suivi d un silence. Signal a x20,3 et x30,5
//	le hasard sur les deux films du gate V1, TEMOIN FANTOME NUL (0 contre 12 et 14).
//
//	L EVENEMENT DE SORTIE (`unit_exit_vehicle`, type 22). Sa grammaire est portee et validee :
//	l occupant tombe dans la bande bipede 75/75 = 100 %, et la sortie FERME le trou a +/-2 s dans
//	69/69 = 100 % des cas (V3 gate 6, 12 films). C est la borne de fin la plus sure du calque.
//
//	L EVENEMENT D EMBARQUEMENT (`biped_board_vehicle`, type 8). Domaines lus dans l executable
//	le 2026-09-02 (2/3/7, cf. `filmdec/event_list.go`) : l occupant tombe dans la bande 22/22 =
//	100 % et OUVRE un trou a l instant exact dans 77,3 % des cas (temoin decale 0,0 %), contre
//	90,7 % pour la REFERENCE qu est la sortie. Le gate absolu de 90 % a echoue et c est ecrit ;
//	ce qui manque tient a la primitive du trou (un trajet de moins de 3 s n en ouvre aucun), pas
//	au decodage.
//
// L EVENEMENT NE NOMME PAS LE VEHICULE. Sa reference 2 (domaine 7, celui des objets du monde)
// est gardee-absente dans la quasi-totalite des cas mesures (rapport V3 embarquement § 4.2). Le
// vehicule d un episode vient donc de la GEOMETRIE — le vehicule le plus proche a l ouverture du
// trou —, jamais de l evenement.
//
// PUR : aucune I/O, aucune lecture de film. Les entrees sont deja decodees.

import (
	"math"
	"sort"
	"strconv"

	"levelup/go-api/internal/analysis/filmdec"
)

// Seuils de l episode d occupation. Aucun n est neuf : ce sont ceux sous lesquels le signal a ete
// mesure, et les changer rendrait les chiffres publies incomparables aux rapports.
const (
	// vehicleGapMinMS : duree minimale d une interruption du flux de position d un bipede pour
	// compter comme un embarquement (oracle geometrique du 2026-08-18, repris par V1a.4 et V1c).
	vehicleGapMinMS = 3000
	// vehicleBoardRadiusM : distance EN PLAN sous laquelle un bipede et un vehicule sont « au
	// meme endroit ». EN PLAN, et c est delibere : la hauteur d un occupant et celle du repere du
	// vehicule ne se referent pas au meme point, une distance 3D ecarterait des embarquements
	// reels. Meme choix, meme raison qu aux socles.
	vehicleBoardRadiusM = 1.5
	// vehicleEventTolMS : tolerance d appariement entre un evenement et le bord du trou qu il
	// doit expliquer. Deux paquets delta valent ~1 s (ecart median mesure 0,5 s) ; on prend
	// large, le biais joue CONTRE le signal quand il manque, jamais en sa faveur.
	vehicleEventTolMS = 2000
	// vehicleNearestSampleUS borne l ecart temporel accepte entre l echantillon du bipede et
	// celui du vehicule qu on lui compare. Au-dela, on ne compare pas deux positions simultanees
	// mais deux positions distantes dans le temps.
	vehicleNearestSampleUS = uint64(1_000_000)
)

// Provenance des bornes d un episode. Identifiants STABLES du document.
const (
	// VehicleRideSrcEvent : les DEUX bornes sont datees par un evenement de la liste (a la
	// milliseconde).
	VehicleRideSrcEvent = "event"
	// VehicleRideSrcMixed : une borne par evenement, l autre par le trou de position.
	VehicleRideSrcMixed = "mixed"
	// VehicleRideSrcGap : les deux bornes viennent du trou de position (aucun evenement apparie).
	VehicleRideSrcGap = "gap"
)

// vehicleRideInputs porte ce que l assemblage des episodes consomme. Une structure plutot que
// six parametres — regle des 5 parametres du depot, et ces six-la voyagent toujours ensemble.
type vehicleRideInputs struct {
	// vehBySlot est le nuage des positions de VEHICULE, indexe par slot et trie par instant.
	vehBySlot map[uint32][]filmdec.BipedPosition
	// bipeds est le nuage NON decime des BIPEDES, d ou sortent les trous.
	bipeds []filmdec.BipedPosition
	// events sont les embarquements et les sorties.
	events []filmdec.VehicleEvent
	// own donne le pont slot -> xuid. Vide : les episodes sortent anonymes, pas supprimes.
	own OwnerReport
	// lives sont les vies de vehicule, avec leur fenetre — c est elle qui rattache un episode a
	// une GENERATION, que le nuage de positions ne porte pas.
	lives []vehicleLife
	clock replayClock
}

// vehicleGap est une interruption du flux de position d un bipede.
type vehicleGap struct {
	slot           uint32
	startUS, endUS uint64
	// last est le DERNIER echantillon avant l interruption : c est la position d embarquement.
	last filmdec.BipedPosition
}

// buildVehicleRides rend les episodes d occupation par vie de vehicule. PUR.
func buildVehicleRides(in vehicleRideInputs) map[filmdec.EquipmentLifeKey][]VehicleRide {
	if in.clock.step == 0 || len(in.lives) == 0 {
		return nil
	}
	boards, exits := vehicleEventsByOccupant(in.events)
	out := map[filmdec.EquipmentLifeKey][]VehicleRide{}
	for _, g := range vehicleGaps(in.bipeds) {
		vs, ok := vehicleNearestTo(g.last, in.vehBySlot)
		if !ok {
			continue
		}
		key, ok := vehicleLifeAt(in.lives, vs, g.startUS)
		if !ok {
			continue
		}
		out[key] = append(out[key], vehicleRideOf(g, boards[g.slot], exits[g.slot], in))
	}
	for k := range out {
		v := out[k]
		sort.SliceStable(v, func(i, j int) bool {
			if v[i].T0 != v[j].T0 {
				return v[i].T0 < v[j].T0
			}
			return v[i].Slot < v[j].Slot
		})
	}
	return out
}

// vehicleRideOf assemble UN episode : les bornes du trou, affinees par les evenements quand ils
// tombent dessus, et l identite de l occupant quand le pont la donne.
func vehicleRideOf(
	g vehicleGap, boards, exits []filmdec.VehicleEvent, in vehicleRideInputs,
) VehicleRide {
	r := VehicleRide{Slot: g.slot, Src: VehicleRideSrcGap}
	startUS, endUS := g.startUS, g.endUS
	fromEvent := 0
	if ev, ok := vehicleEventNear(boards, g.startUS); ok {
		startUS, fromEvent = ev.TimestampUS, fromEvent+1
		r.Seat = vehicleSeatOf(ev)
	}
	if ev, ok := vehicleEventNear(exits, g.endUS); ok {
		endUS, fromEvent = ev.TimestampUS, fromEvent+1
		// LE SIEGE DE LA SORTIE PRIME : c est celui dont la mesure est la plus fournie
		// (`siege = 0` sur 93,8 % des sorties, n = 237), et il s accorde a celui de
		// l embarquement apparie dans 5 cas sur 6.
		if s := vehicleSeatOf(ev); s != nil {
			r.Seat = s
		}
	}
	switch fromEvent {
	case 2:
		r.Src = VehicleRideSrcEvent
	case 1:
		r.Src = VehicleRideSrcMixed
	}
	r.T0 = in.clock.frame(startUS)
	r.T1 = in.clock.frame(endUS)
	if r.T1 < r.T0 {
		r.T1 = r.T0
	}
	if x, ok := in.own.SlotXUID[g.slot]; ok {
		r.XUID = strconv.FormatUint(x, 10)
	}
	return r
}

// vehicleSeatOf rend le siege d un evenement, ou nil quand la charge etait trop courte pour le
// porter. POINTEUR, et il le faut : le siege 0 est le CONDUCTEUR, c est-a-dire la valeur la plus
// frequente et la plus utile du champ — `omitempty` sur un entier l effacerait exactement comme
// une absence de lecture.
func vehicleSeatOf(ev filmdec.VehicleEvent) *int {
	if !ev.SeatValid {
		return nil
	}
	s := int(ev.Seat)
	return &s
}

// vehicleGaps releve les interruptions >= vehicleGapMinMS du flux de position de chaque bipede.
// L ordre de sortie est deterministe (slots tries, echantillons tries par instant).
func vehicleGaps(bipeds []filmdec.BipedPosition) []vehicleGap {
	bySlot := map[uint32][]filmdec.BipedPosition{}
	for _, b := range bipeds {
		if b.HasWorld {
			bySlot[b.Slot] = append(bySlot[b.Slot], b)
		}
	}
	slots := make([]uint32, 0, len(bySlot))
	for s := range bySlot {
		slots = append(slots, s)
	}
	sort.Slice(slots, func(i, j int) bool { return slots[i] < slots[j] })
	var out []vehicleGap
	for _, s := range slots {
		ech := bySlot[s]
		sort.SliceStable(ech, func(i, j int) bool { return ech[i].TimestampUS < ech[j].TimestampUS })
		for i := 1; i < len(ech); i++ {
			if (ech[i].TimestampUS-ech[i-1].TimestampUS)/1000 < vehicleGapMinMS {
				continue
			}
			out = append(out, vehicleGap{
				slot: s, startUS: ech[i-1].TimestampUS, endUS: ech[i].TimestampUS, last: ech[i-1],
			})
		}
	}
	return out
}

// vehicleNearestTo rend le slot du vehicule sous le rayon a cet instant, s il y en a un. Predicat
// EN PLAN, tolerance temporelle bornee : c est exactement l oracle geometrique du 2026-08-18,
// celui sous lequel le signal a ete mesure.
func vehicleNearestTo(
	e filmdec.BipedPosition, vehBySlot map[uint32][]filmdec.BipedPosition,
) (uint32, bool) {
	slots := make([]uint32, 0, len(vehBySlot))
	for s := range vehBySlot {
		slots = append(slots, s)
	}
	sort.Slice(slots, func(i, j int) bool { return slots[i] < slots[j] })
	best, found, bestD := uint32(0), false, math.MaxFloat64
	for _, s := range slots {
		p, gap, ok := vehicleSampleNear(vehBySlot[s], e.TimestampUS)
		if !ok || gap > vehicleNearestSampleUS {
			continue
		}
		d := math.Hypot(float64(e.X-p.X), float64(e.Y-p.Y))
		if d <= vehicleBoardRadiusM && d < bestD {
			best, bestD, found = s, d, true
		}
	}
	return best, found
}

// vehicleSampleNear rend l echantillon le plus proche d un instant dans une liste TRIEE, et
// l ecart.
func vehicleSampleNear(
	pts []filmdec.BipedPosition, atUS uint64,
) (filmdec.BipedPosition, uint64, bool) {
	if len(pts) == 0 {
		return filmdec.BipedPosition{}, 0, false
	}
	i := sort.Search(len(pts), func(k int) bool { return pts[k].TimestampUS >= atUS })
	best := i
	switch {
	case i >= len(pts):
		best = len(pts) - 1
	case i > 0 && gapUS(pts[i-1].TimestampUS, atUS) < gapUS(pts[i].TimestampUS, atUS):
		best = i - 1
	}
	return pts[best], gapUS(pts[best].TimestampUS, atUS), true
}

// vehicleLifeAt rattache un slot de vehicule et un instant a UNE vie : celle dont la fenetre le
// contient. Les fenetres d un meme slot ne se recouvrent pas (cf. assignVehicleWindows), la
// reponse est donc unique.
func vehicleLifeAt(
	lives []vehicleLife, slot uint32, atUS uint64,
) (filmdec.EquipmentLifeKey, bool) {
	for _, l := range lives {
		if l.key.Slot == slot && atUS >= l.loUS && atUS <= l.hiUS {
			return l.key, true
		}
	}
	return filmdec.EquipmentLifeKey{}, false
}

// vehicleEventsByOccupant range les evenements par slot d occupant, embarquements d un cote et
// sorties de l autre, chacun TRIE par instant.
//
// LES OCCUPANTS HORS BANDE SONT ECARTES : un slot qui ne tombe pas dans la bande bipede du film
// n est pas un joueur, c est une lecture qui a rate. La mesure les compte a zero sur le corpus
// (68/68 en bande apres le portage du 2026-09-02) ; le filtre est la pour que ca reste vrai.
func vehicleEventsByOccupant(
	events []filmdec.VehicleEvent,
) (boards, exits map[uint32][]filmdec.VehicleEvent) {
	boards, exits = map[uint32][]filmdec.VehicleEvent{}, map[uint32][]filmdec.VehicleEvent{}
	for _, ev := range events {
		if !ev.OccupantPresent || !ev.OccupantInBand {
			continue
		}
		switch ev.Kind {
		case filmdec.EventBipedBoardVehicle:
			boards[ev.OccupantSlot] = append(boards[ev.OccupantSlot], ev)
		case filmdec.EventUnitExitVehicle:
			exits[ev.OccupantSlot] = append(exits[ev.OccupantSlot], ev)
		}
	}
	for _, m := range []map[uint32][]filmdec.VehicleEvent{boards, exits} {
		for s := range m {
			v := m[s]
			sort.SliceStable(v, func(i, j int) bool { return v[i].TimestampUS < v[j].TimestampUS })
		}
	}
	return boards, exits
}

// vehicleEventNear rend l evenement le plus proche d un instant dans la tolerance, s il y en a un.
func vehicleEventNear(
	evs []filmdec.VehicleEvent, atUS uint64,
) (filmdec.VehicleEvent, bool) {
	best, found, bestGap := filmdec.VehicleEvent{}, false, uint64(0)
	for _, ev := range evs {
		g := gapUS(ev.TimestampUS, atUS)
		if g/1000 > vehicleEventTolMS {
			continue
		}
		if !found || g < bestGap {
			best, bestGap, found = ev, g, true
		}
	}
	return best, found
}

// gapUS rend l ecart absolu entre deux instants microseconde.
func gapUS(a, b uint64) uint64 {
	if a > b {
		return a - b
	}
	return b - a
}
