package replay

// vehicle_rides_events.go — LA MACHINE D ETATS D OCCUPATION, PILOTEE PAR L OCCUPANT.
//
// CE QUI A CHANGE, ET POURQUOI. Jusqu au lot V6 l episode d occupation etait pilote par le TROU
// du flux de position : un trou de plus de 3 s pres d un vehicule OUVRAIT un episode, et les
// evenements de la liste ne servaient qu a en affiner les bornes. Deux consequences mesurees :
//
//	un occupant qui NE RE-EMET JAMAIS (mort a bord, encore a bord a la fin du film) n ouvre
//	AUCUN trou — la primitive exige un echantillon APRES le silence. Le lot V4 l avait chiffre
//	(6 tirs de vehicule sur 23 hors episode a `0d76e8f1`, 8 sur 16 a `fccc61cd`) et laisse
//	ouvert faute de seuil justifiable ;
//
//	une SORTIE dont le trou n est pas rattachable (le dernier point replique de l occupant est
//	a plus de 1,5 m de tout vehicule) etait perdue, alors que l evenement, lui, est sur :
//	occupant en bande 100 %, et il FERME un trou dans 90,7 % des cas (V3, 12 films).
//
// LA MACHINE EST DONC PAR OCCUPANT, et les evenements en sont les BORDS :
//
//	EMBARQUEMENT -> ouvre un episode. Un second embarquement sans sortie ferme le precedent.
//	SORTIE       -> ferme l episode ouvert. S il n y en a pas, elle en ouvre un dont le debut
//	                est le DERNIER point replique par l occupant avant elle (c est exactement le
//	                debut du trou quand il y en a un — la meme grandeur, lue par l autre bout).
//	FIN DE LISTE avec un episode OUVERT -> l episode se ferme a la REAPPARITION de l occupant
//	                (il est descendu, ou mort a bord puis respawne) et, s il ne reapparait
//	                JAMAIS, a la fin de vie du vehicule : c est le SILENCE TERMINAL, ce que
//	                l ancien pilotage ne savait pas attribuer.
//
// LE VEHICULE RESTE RESOLU PAR LA POSITION — l evenement ne le nomme pas. Le lot V6 a re-teste
// l hypothese « la reference 2 de l embarquement est le vehicule » sur les 22 embarquements du
// corpus : les trois references sont bien PRESENTES (37 bits de refs = 11 + 10 + 16, confirme
// par le debut de trame a 53 bits sur 14/14 paquets), mais la valeur de la reference 2 ne tombe
// dans la bande de slots `ti=40` dans AUCUN cas (0/22), ni dans celle des armes au sol (0/22),
// et elle ne prend que QUATRE valeurs distinctes sur 22 instances (180, 116, 244, 208) — la
// signature d un identifiant de DEFINITION, pas d une instance. La geometrie reste donc la
// methode, avec DEUX ancres au lieu d une : le dernier point avant le debut, et le premier point
// apres la fin.
//
// LE TROU RESTE, EN REPLI : un trou qu aucun episode d evenement ne recouvre produit toujours un
// episode, aux memes portes qu avant (3 s, 1,5 m en plan, fraicheur 1 s). Aucun seuil n a bouge.
//
// PUR : aucune I/O.

import (
	"sort"
	"strconv"

	"levelup/go-api/internal/analysis/filmdec"
)

// vehicleEventAnchorRadiusM est la distance EN PLAN sous laquelle l ancre d un episode
// D EVENEMENT designe SON vehicule. Elle est plus large que celle du trou
// (`vehicleBoardRadiusM` = 1,5 m), et la raison n est pas un assouplissement : LES DEUX
// POPULATIONS NE SONT PAS LES MEMES.
//
// LE RAYON DU TROU DECIDE S IL Y A EU EMBARQUEMENT. Il doit donc etre serre, et le lot V4 a
// REFUTE de l ouvrir : a 3, 5, 8 et 12 m il n ajoutait AUCUN trou confirme et faisait passer les
// trous non confirmes de 2 a 8, pendant que le temoin decale de 60 s en ramassait 1 puis 6.
//
// LE RAYON DE L ANCRE D EVENEMENT NE DECIDE PAS DE L EMBARQUEMENT — l evenement l a deja prouve
// (occupant en bande 100 %, grammaire lue dans l executable). Il ne choisit que QUEL vehicule.
// La mesure est donc differente, et elle a ete refaite (`vehicules_v6_etats_test.go`, 2026-09-03,
// 5 films, 49 episodes attestes) : la distance de l ancre a son vehicule vaut 0,5 a 2,2 m dans la
// quasi-totalite des cas (un 3,0 m et un aberrant a 53,7 m), tandis que le temoin decale de 60 s
// se tient a 4,7 a 75 m. TABLE COMPLETE, episodes attestes seulement :
//
//	rayon    rattaches   AMBIGUS (un 2e vehicule sous le rayon)   TEMOIN +60 s
//	1,5 m    39 / 49                 1                                 1
//	2,0 m    46 / 49                 1                                 1
//	3,0 m    48 / 49                 1                                 1
//	5,0 m    48 / 49                 2                                 2
//	8,0 m    48 / 49                 3                                 3
//	12,0 m   49 / 49                 4                                 3
//
// 3 m EST LE DERNIER RAYON QUI NE COUTE RIEN : +9 episodes rattaches sur 49, l ambiguite et le
// temoin STRICTEMENT INCHANGES par rapport a 1,5 m. Des 5 m, chaque episode gagne (zero) se paie
// en ambiguite et en temoin. Le critere est le meme que celui du lot V4 pour le trou — on s
// arrete ou le temoin commence a bouger —, seule la population mesuree differe.
const vehicleEventAnchorRadiusM = 3.0

// vehicleEpisode est un episode d occupation AVANT rattachement a un vehicule : l occupant, ses
// deux bornes, et d ou elles viennent.
type vehicleEpisode struct {
	slot           uint32
	startUS, endUS uint64
	seat           *int
	// borders compte les bornes datees par un EVENEMENT (0, 1 ou 2).
	borders int
	// openEnd : aucune sortie n a ferme l episode — il se ferme a la REAPPARITION de l occupant
	// si elle existe, sinon a la fin de vie du vehicule (SILENCE TERMINAL vrai).
	openEnd bool
	// vehSlot / vehValid / vehAtUS : LE VEHICULE NOMME PAR L EVENEMENT, et l instant ou il le
	// nomme. Ils viennent de la SORTIE (`filmdec.VehicleEvent.VehicleSlot`, reference 1 de
	// domaine 1) ; un EMBARQUEMENT n en porte pas — ses trois references sont en domaines 2/3/7
	// et AUCUNE ne resout un slot `ti=40` (0/15 sur 12 films, rapport V8 § 2). Un episode ferme
	// par un second embarquement, ou un SILENCE TERMINAL, sort donc sans nom : c est exactement
	// la population que la geometrie continue de rattacher.
	vehSlot  uint32
	vehValid bool
	vehAtUS  uint64
	// resolvedBy dit PAR QUELLE VOIE le vehicule a ete trouve. Rempli par
	// `vehicleRideFromEpisode` sur l episode qu il rend ; nul sur un episode non rattache.
	resolvedBy vehicleResolvedBy
	// reappearUS : premier instant ou l occupant re-emet une position APRES le debut. Zero =
	// jamais. UN OCCUPANT QUI MEURT A BORD REAPPARAIT (il respawne et re-replique) : sans cette
	// borne, son episode courrait jusqu a la fin de vie du vehicule et le montrerait au volant
	// bien apres sa mort.
	//
	// HONNETETE DE MESURE : ce cas NE S EST PAS PRODUIT sur les deux films de demonstration du
	// 2026-09-03 (l unique silence terminal de `0d76e8f1`, slot 561, ne reapparait vraiment
	// jamais — son episode de 906 frames est la bonne reponse, pas un defaut). La borne est
	// posee parce que la SEMANTIQUE l exige, pas parce qu un chiffre l a exigee ; elle est
	// couverte par `TestVehicleEpisodeReappearanceClosesOpenEnd`, sur fixtures.
	reappearUS uint64
}

// vehicleEventEpisodes deroule la machine d etats, occupant par occupant. Les episodes sortent
// tries (occupant, instant de debut) : la sortie est deterministe.
func vehicleEventEpisodes(
	boards, exits map[uint32][]filmdec.VehicleEvent,
	bySlot map[uint32][]filmdec.BipedPosition,
) []vehicleEpisode {
	slots := make([]uint32, 0, len(boards)+len(exits))
	seen := map[uint32]bool{}
	for _, m := range []map[uint32][]filmdec.VehicleEvent{boards, exits} {
		for s := range m {
			if !seen[s] {
				seen[s], slots = true, append(slots, s)
			}
		}
	}
	sort.Slice(slots, func(i, j int) bool { return slots[i] < slots[j] })
	var out []vehicleEpisode
	for _, s := range slots {
		out = append(out, vehicleEpisodesOfOccupant(s, mergeVehicleEvents(boards[s], exits[s]),
			bySlot[s])...)
	}
	return out
}

// mergeVehicleEvents fusionne deux listes DEJA triees par instant en une seule, stable. A instant
// egal la SORTIE passe avant l EMBARQUEMENT : descendre puis remonter est le seul ordre qui ait
// un sens physique, et l inverse fabriquerait un episode de duree nulle.
func mergeVehicleEvents(boards, exits []filmdec.VehicleEvent) []filmdec.VehicleEvent {
	out := make([]filmdec.VehicleEvent, 0, len(boards)+len(exits))
	i, j := 0, 0
	for i < len(boards) && j < len(exits) {
		if exits[j].TimestampUS <= boards[i].TimestampUS {
			out, j = append(out, exits[j]), j+1
			continue
		}
		out, i = append(out, boards[i]), i+1
	}
	out = append(out, boards[i:]...)
	return append(out, exits[j:]...)
}

// vehicleEpisodesOfOccupant deroule la machine pour UN occupant.
func vehicleEpisodesOfOccupant(
	slot uint32, evs []filmdec.VehicleEvent, pts []filmdec.BipedPosition,
) []vehicleEpisode {
	var out []vehicleEpisode
	open, hasOpen := vehicleEpisode{}, false
	for _, ev := range evs {
		if ev.Kind == filmdec.EventBipedBoardVehicle {
			if hasOpen {
				// Deux embarquements sans sortie entre eux : le premier episode se termine ou le
				// second commence. Le film n en dit pas plus, et supposer autre chose serait
				// inventer une duree.
				open.endUS, open.openEnd = ev.TimestampUS, false
				out = append(out, open)
			}
			open = vehicleEpisode{slot: slot, startUS: ev.TimestampUS,
				seat: vehicleSeatOf(ev), borders: 1, openEnd: true}
			hasOpen = true
			continue
		}
		if !hasOpen {
			open = vehicleEpisode{slot: slot, startUS: vehicleLastPointBefore(pts, ev.TimestampUS)}
		}
		open.endUS, open.openEnd = ev.TimestampUS, false
		open.borders++
		// LA SORTIE NOMME LE VEHICULE — c est elle, et elle seule, qui le fait (105/105 en bande
		// `ti=40`, zero bipede, zero hors bande sur 12 films).
		if ev.VehicleSlotValid {
			open.vehSlot, open.vehValid, open.vehAtUS = ev.VehicleSlot, true, ev.TimestampUS
		}
		// LE SIEGE DE LA SORTIE PRIME : c est celui dont la mesure est la plus fournie
		// (siege 0 sur 93,8 % des sorties, n = 237) et il s accorde a celui de l embarquement
		// apparie dans 5 cas sur 6 (V3).
		if s := vehicleSeatOf(ev); s != nil {
			open.seat = s
		}
		out, hasOpen = append(out, open), false
	}
	if hasOpen {
		// SILENCE TERMINAL — mais seulement si l occupant ne re-emet JAMAIS. S il reapparait, il
		// est descendu (ou mort a bord et respawne) : l episode se ferme la, comme le ferait le
		// trou du flux de position.
		if p, ok := vehicleAnchorAt(pts, open.startUS+1, true); ok {
			open.reappearUS = p.TimestampUS
		}
		out = append(out, open)
	}
	return out
}

// vehicleLastPointBefore rend l instant du DERNIER echantillon de `pts` (TRIE) strictement
// anterieur a `atUS`, ou `atUS` lui-meme quand il n y en a aucun.
func vehicleLastPointBefore(pts []filmdec.BipedPosition, atUS uint64) uint64 {
	i := sort.Search(len(pts), func(k int) bool { return pts[k].TimestampUS >= atUS })
	if i == 0 {
		return atUS
	}
	return pts[i-1].TimestampUS
}

// vehicleEpisodeCovers dit si un episode PUBLIE recouvre deja ce trou, pour le MEME occupant.
// C est la regle anti-doublon : le trou est un REPLI, pas une seconde source.
func vehicleEpisodeCovers(eps []vehicleEpisode, g vehicleGap) bool {
	for _, e := range eps {
		if e.slot == g.slot && g.startUS <= e.endUS && e.startUS <= g.endUS {
			return true
		}
	}
	return false
}

// vehicleRideFromEpisode rattache un episode a une VIE de vehicule et rend l episode publiable.
//
// L EVENEMENT PASSE AVANT LA GEOMETRIE (lot V8, 2026-09-03). La SORTIE NOMME son vehicule : sa
// reference 1 est de domaine 1 — le domaine des UNITES, bipedes ET vehicules — et elle tombe dans
// la bande `ti=40` sur 105 / 105 sorties de 12 films, zero bipede, zero hors bande, quand le
// hasard en mettrait 3 a 16 %. Un nom exact ne se remplace pas par une distance : la geometrie
// (rayon `vehicleEventAnchorRadiusM`) devient le REPLI, pour les episodes qu aucune sortie ne
// nomme — ceux que ferme un second embarquement, et les SILENCES TERMINAUX.
//
// LE REPLI GEOMETRIQUE, INCHANGE, GARDE SES DEUX ANCRES : le dernier point replique par
// l occupant AVANT le debut (la position d embarquement), puis le premier point APRES la fin (la
// position de debarquement). La seconde n existe pas pour un silence terminal, et c est justement
// pour cela que la premiere passe d abord.
func vehicleRideFromEpisode(
	ep vehicleEpisode, bySlot map[uint32][]filmdec.BipedPosition, in vehicleRideInputs,
) (filmdec.EquipmentLifeKey, VehicleRide, vehicleEpisode, bool) {
	pts := bySlot[ep.slot]
	life, src := vehicleLifeFromEvent(ep, in)
	if src == vehicleResolvedNone {
		var ok bool
		if life, ok = vehicleLifeFromGeometry(ep, pts, in); ok {
			src = vehicleResolvedByGeometry
		}
	}
	if src == vehicleResolvedNone {
		return filmdec.EquipmentLifeKey{}, VehicleRide{}, vehicleEpisode{}, false
	}
	ep.resolvedBy = src
	if ep.openEnd {
		ep.endUS = life.hiUS
		if ep.reappearUS > ep.startUS && ep.reappearUS < ep.endUS {
			ep.endUS = ep.reappearUS
		}
	}
	r := VehicleRide{Slot: ep.slot, Seat: ep.seat, Src: vehicleRideSrcOf(ep.borders)}
	r.T0, r.T1 = in.clock.frame(ep.startUS), in.clock.frame(ep.endUS)
	if r.T1 < r.T0 {
		r.T1 = r.T0
	}
	if x, ok := in.own.SlotXUID[ep.slot]; ok {
		r.XUID = strconv.FormatUint(x, 10)
	}
	return life.key, r, ep, true
}

// Comment le vehicule d un episode a ete resolu. INTERNE — le document ne publie pas ce champ
// (le contrat ne bouge pas), mais le journal et les instruments en vivent : sans lui, « 49
// episodes publies » ne dirait pas si c est l evenement ou la distance qui les a rattaches.
type vehicleResolvedBy int

const (
	// vehicleResolvedNone : aucune des deux voies n a rendu de vie — l episode n est pas publie.
	vehicleResolvedNone vehicleResolvedBy = iota
	// vehicleResolvedByEvent : la sortie nomme le vehicule, et l instant de la sortie tombe DANS
	// la fenetre d une vie de ce slot. C est le cas nominal.
	vehicleResolvedByEvent
	// vehicleResolvedByEventNearest : la sortie nomme le vehicule, mais son instant ne tombe dans
	// AUCUNE fenetre de vie de ce slot — la vie la plus proche dans le temps est retenue.
	//
	// POURQUOI CE CAS EXISTE, MESURE : la fenetre d une vie s arrete a la premiere image-cle qui
	// ne la recense plus, et les images-cles sont espacees de ~20 s. Une sortie peut donc tomber
	// APRES cette borne (5 sorties sur 105, ecart maximal 41,9 s — rapport V8 § 2). Le nom, lui,
	// reste exact : c est la FENETRE qui est trop courte, pas la reference qui est fausse. La vie
	// retenue est ensuite ramenee dans sa fenetre d affichage par `clampVehicleRides`, qui ecarte
	// l episode s il lui est entierement exterieur — le cas se solde donc au pire par un episode
	// non publie, jamais par un episode attribue au mauvais vehicule.
	vehicleResolvedByEventNearest
	// vehicleResolvedByGeometry : aucun nom — le vehicule le plus proche de l ancre, sous
	// `vehicleEventAnchorRadiusM`.
	vehicleResolvedByGeometry
)

// vehicleLifeFromEvent resout la vie du vehicule NOMME par l evenement, ET REFUSE une vie que le
// calque ne publiera pas.
//
// POURQUOI CE REFUS — C EST LA MESURE DU LOT V8, ET ELLE EST INSTRUCTIVE. Sur 41 episodes ou les
// deux voies repondent (5 films), 35 designent la MEME vie et 6 divergent. Les 6 ont exactement la
// meme forme : l evenement nomme une vie MUETTE (aucun echantillon de position, aucun record de
// creation, donc ni chassis ni sprite) dont le voisin immediat `slot + 1`, RECENSE A LA MEME
// FENETRE, porte le chassis et la trajectoire. Cinq de ces voisins sont des `warthog`, un est un
// `falcon` — les deux seules familles du corpus a tourelle d artilleur —, et les six episodes
// portent le siege 0. Sur les 50 vies muettes du corpus, le voisin `slot + 1` a la MEME fenetre
// est un warthog (23) ou un falcon (10) ou un chassis non resolu (3), JAMAIS un ghost, un
// mongoose, un chopper ni une banshee.
//
// L EXPLICATION LA PLUS ECONOMIQUE, et elle recoupe un rapport anterieur : le Warthog est un
// ASSEMBLAGE — sa tourelle (`warthog_g`) est un tag `vehi` a part entiere
// (`ASSEMBLAGE_ENFANTS_2026-09-01.md`), donc une entite `ti=40` de plus, ATTACHEE au chassis et
// qui ne replique donc jamais sa position. La tourelle est elle-meme une UNITE qui porte un
// siege : l ARTILLEUR y monte, et son siege y vaut 0 comme celui du conducteur vaut 0 sur le
// chassis. L evenement de sortie nommerait alors l unite REELLEMENT quittee — chassis pour le
// conducteur, tourelle pour l artilleur —, ce que le calque ne sait pas distinguer aujourd hui.
// Les deux voies auraient RAISON ; elles ne repondraient pas a la meme question. CE DERNIER PAS
// N EST PAS PROUVE (aucune verite terrain ne dit qui, du conducteur ou de l artilleur, sortait) ;
// ce qui est mesure, c est la forme des six cas et la famille des voisins.
//
// CE QUE LE CALQUE PEUT EN FAIRE AUJOURD HUI : rien de plus, parce qu il ne publie qu un sprite
// par vehicule et que la tourelle n en a pas. Un episode accroche a une vie muette serait ECARTE a
// la publication (`vehicleTrackOf` refuse une vie sans naissance ni echantillon) — c est-a-dire un
// occupant PERDU. La regle est donc : le nom prime, SAUF quand il designe une vie que le calque ne
// dessinera pas ; la geometrie, qui designe le chassis porteur, reprend alors la main. Mesure
// avant / apres a l appui : sans ce garde-fou, l artefact perdait 4 episodes sur 41.
func vehicleLifeFromEvent(ep vehicleEpisode, in vehicleRideInputs) (vehicleLife, vehicleResolvedBy) {
	l, src := vehicleLifeNamedByEvent(ep, in.lives)
	if src != vehicleResolvedNone && !in.drawable[l.key] {
		return vehicleLife{}, vehicleResolvedNone
	}
	return l, src
}

// vehicleLifeNamedByEvent est la resolution NUE : la vie du slot que l evenement nomme, sans le
// garde-fou de publiabilite. Separee pour que les instruments puissent mesurer ce que l evenement
// DIT, et pas seulement ce que le calque en RETIENT.
func vehicleLifeNamedByEvent(
	ep vehicleEpisode, lives []vehicleLife,
) (vehicleLife, vehicleResolvedBy) {
	if !ep.vehValid {
		return vehicleLife{}, vehicleResolvedNone
	}
	best, found, bestGap := vehicleLife{}, false, uint64(0)
	for _, l := range lives {
		if l.key.Slot != ep.vehSlot {
			continue
		}
		if ep.vehAtUS >= l.loUS && ep.vehAtUS <= l.hiUS {
			// LES FENETRES D UN MEME SLOT NE SE RECOUVRENT PAS (`assignVehicleWindows` les
			// decoupe) : la reponse exacte est unique, on peut rendre la premiere.
			return l, vehicleResolvedByEvent
		}
		g := ep.vehAtUS - l.hiUS
		if ep.vehAtUS < l.loUS {
			g = l.loUS - ep.vehAtUS
		}
		if !found || g < bestGap {
			best, bestGap, found = l, g, true
		}
	}
	if !found {
		return vehicleLife{}, vehicleResolvedNone
	}
	return best, vehicleResolvedByEventNearest
}

// vehicleLifeFromGeometry est le REPLI : le vehicule le plus proche d une des deux ancres.
func vehicleLifeFromGeometry(
	ep vehicleEpisode, pts []filmdec.BipedPosition, in vehicleRideInputs,
) (vehicleLife, bool) {
	a0, ok0 := vehicleAnchorAt(pts, ep.startUS, false)
	life, ok := vehicleLifeForAnchor(a0, ok0, ep.startUS, in)
	if ok || ep.openEnd {
		return life, ok
	}
	a1, ok1 := vehicleAnchorAt(pts, ep.endUS, true)
	return vehicleLifeForAnchor(a1, ok1, ep.endUS, in)
}

// vehicleRideSrcOf traduit le nombre de bornes datees par un evenement en provenance publiee.
func vehicleRideSrcOf(borders int) string {
	switch {
	case borders >= 2:
		return VehicleRideSrcEvent
	case borders == 1:
		return VehicleRideSrcMixed
	default:
		return VehicleRideSrcGap
	}
}

// vehicleAnchorAt rend l echantillon d ancrage : le dernier AVANT `atUS` (after=false) ou le
// premier APRES (after=true).
func vehicleAnchorAt(
	pts []filmdec.BipedPosition, atUS uint64, after bool,
) (filmdec.BipedPosition, bool) {
	i := sort.Search(len(pts), func(k int) bool { return pts[k].TimestampUS >= atUS })
	if after {
		if i >= len(pts) {
			return filmdec.BipedPosition{}, false
		}
		return pts[i], true
	}
	if i == 0 {
		return filmdec.BipedPosition{}, false
	}
	return pts[i-1], true
}

// vehicleLifeForAnchor resout le vehicule le plus proche d une ancre, puis la vie qui porte cet
// instant. Rayon `vehicleEventAnchorRadiusM` en plan, fraicheur 1 s (celle de la production).
func vehicleLifeForAnchor(
	anchor filmdec.BipedPosition, has bool, atUS uint64, in vehicleRideInputs,
) (vehicleLife, bool) {
	if !has {
		return vehicleLife{}, false
	}
	vs, ok := vehicleNearestWithin(anchor, in.vehBySlot, vehicleEventAnchorRadiusM)
	if !ok {
		return vehicleLife{}, false
	}
	for _, l := range in.lives {
		if l.key.Slot == vs && atUS >= l.loUS && atUS <= l.hiUS {
			return l, true
		}
	}
	return vehicleLife{}, false
}
