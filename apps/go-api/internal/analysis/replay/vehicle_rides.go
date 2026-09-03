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
	"sort"
	"strconv"

	"levelup/go-api/internal/analysis/filmdec"
)

// CES SEUILS ONT ETE REMIS EN CAUSE LE 2026-09-02, ET LA MESURE LES A CONFIRMES. Le calque
// publiait 12 episodes sur 45 vies (`0d76e8f1`) et 2 sur 21 (`fccc61cd`) ; l hypothese etait que
// les portes de cette page etaient trop serrees. L instrument `vehicules_v4_couverture_test.go`
// a mesure chaque porte contre un ORACLE — un trou CONFIRME par un evenement d embarquement ou
// de sortie de la liste (grammaire portee et validee, V3). VERDICT, sur les deux films :
//
//	LE RAYON N EST PAS LA PORTE. Les 10 trous confirmes de `0d76e8f1` et les 2 de `fccc61cd`
//	sont TOUS sous 1,5 m — d25 0,6 m, d50 0,9 m, d90 1,2 m. Elargir a 3, 5, 8 ou 12 m n ajoute
//	AUCUN trou confirme et fait passer les trous NON confirmes de 2 a 4, 5, 8 puis 8 (a 3 s),
//	pendant que le temoin decale de 60 s en ramasse 1, 2, 4 puis 6 : ce qu on gagnerait est du
//	hasard, et il est deja compte.
//
//	LE SEUIL DE TROU NON PLUS. A 3 s, 1,5 s et 0,8 s le nombre de trous CONFIRMES vaut 10, 10
//	et 10 (`0d76e8f1`) — exactement le meme. Seuls les non confirmes montent (16, 24, 45).
//
//	LA FRAICHEUR NON PLUS. 9 des 10 trous confirmes avaient un echantillon de vehicule de moins
//	d une seconde, et aucun n a eu besoin de la position de naissance.
//
//	ET L OCCUPANT NE REPLIQUE VRAIMENT PLUS. L hypothese inverse — des trajets invisibles parce
//	que le bipede continue d emettre — a ete testee par la CO-MOBILITE
//	(`vehicules_v4_comobilite_test.go`) : sur 1 347 instants de vehicule EN MOUVEMENT, AUCUN
//	couple (vehicule, bipede) ne reste sous 3 m plus de 1,6 s. Il n y a pas de population cachee.
//
// CE QUI RESTE, MESURE ET NON CORRIGE : le SILENCE TERMINAL. Un occupant qui ne re-emet JAMAIS
// (mort a bord, encore a bord a la fin) n ouvre aucun trou — la primitive exige un point APRES le
// silence. L oracle des ARMES DE VEHICULE (cf. `vehicle_shots.go`) le chiffre : 6 tirs sur 23 a
// `0d76e8f1`, 8 sur 16 a `fccc61cd`. Aucun seuil n est justifiable aujourd hui pour les recuperer
// — le vehicule le plus proche du dernier point replique est a 2,6 m sur un film et a 29,3 m sur
// l autre. Ouvrir la porte sur cette base echangerait de la precision contre du rappel sans
// preuve ; la decouverte est ecrite, elle n est pas traitee.
//
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
//
// DEUX SOURCES, DANS CET ORDRE (lot V6). La machine d etats par OCCUPANT
// (`vehicle_rides_events.go`) passe d abord : ses bornes sont des EVENEMENTS de la liste, dates a
// la milliseconde et valides (occupant en bande 100 %). Le TROU du flux de position ne sert plus
// qu en REPLI, pour les episodes qu aucun evenement n atteste — aux memes portes qu avant.
func buildVehicleRides(in vehicleRideInputs) map[filmdec.EquipmentLifeKey][]VehicleRide {
	if in.clock.step == 0 || len(in.lives) == 0 {
		return nil
	}
	boards, exits := vehicleEventsByOccupant(in.events)
	bySlot := vehiclePositionsBySlot(in.bipeds)
	out := map[filmdec.EquipmentLifeKey][]VehicleRide{}
	var kept []vehicleEpisode
	for _, ep := range vehicleEventEpisodes(boards, exits, bySlot) {
		key, r, resolved, ok := vehicleRideFromEpisode(ep, bySlot, in)
		if !ok {
			continue
		}
		out[key] = append(out[key], r)
		kept = append(kept, resolved)
	}
	for _, g := range vehicleGaps(in.bipeds) {
		// SEULS LES EPISODES PUBLIES ferment la porte du repli. Un episode d evenement dont le
		// vehicule n a pas pu etre resolu ne doit RIEN supprimer : sinon la machine d etats
		// retirerait un episode que le trou, lui, savait rattacher (mesure du 2026-09-03 :
		// 12 -> 11 sur `0d76e8f1` avec la regle naive).
		if vehicleEpisodeCovers(kept, g) {
			continue
		}
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
	return vehicleNearestWithin(e, vehBySlot, vehicleBoardRadiusM)
}

// vehicleNearestWithin est la MEME recherche, sous un rayon donne. Une seule implementation :
// le rayon du TROU (`vehicleBoardRadiusM`) et celui de l ANCRE D EVENEMENT
// (`vehicleEventAnchorRadiusM`) ne sont pas le meme chiffre, mais ils lisent le meme nuage avec
// la meme regle de fraicheur.
func vehicleNearestWithin(
	e filmdec.BipedPosition, vehBySlot map[uint32][]filmdec.BipedPosition, radiusM float64,
) (uint32, bool) {
	slots := make([]uint32, 0, len(vehBySlot))
	for s := range vehBySlot {
		slots = append(slots, s)
	}
	sort.Slice(slots, func(i, j int) bool { return slots[i] < slots[j] })
	best, found, bestD := uint32(0), false, 0.0
	for _, s := range slots {
		p, gap, ok := vehicleSampleNear(vehBySlot[s], e.TimestampUS)
		if !ok || gap > vehicleNearestSampleUS {
			continue
		}
		d := planDist(e.X, e.Y, p.X, p.Y)
		if d <= radiusM && (!found || d < bestD) {
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
