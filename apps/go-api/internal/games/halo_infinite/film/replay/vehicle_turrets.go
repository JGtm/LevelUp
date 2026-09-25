package replay

// vehicle_turrets.go — LES PIECES MONTEES : une tourelle se dessine SUR son vehicule, jamais seule
// a sa naissance (schema 69, retours du rejeu du 2026-09-23, lot M4a, decision utilisateur Q12).
//
// LE DEFAUT MESURE. Le film cree les tourelles d un vehicule (LAAG et lance-roquettes du Warthog,
// tourelles laterales du Falcon, tourelle plasma et piece du mortier du Wraith, canon du Scorpion)
// comme des objets `ti=40` A PART ENTIERE, avec leur record de creation et leur mot d identite —
// mais SANS AUCUN echantillon de position : c est le chassis qui se deplace. Publiees comme des
// vehicules, elles restaient a leur NAISSANCE, et avec elles l artilleur et ses tirs : mediane
// 44,7 m entre un tir de roquettes et le Warthog qui le portait, 119 m au pire (annexe
// `RAPPORT_tirs_vehicules.md` § 2.3.3).
//
// CE QUE CE FICHIER FAIT, SUR LES VIES DEJA ASSEMBLEES (aucune lecture de film) :
//
//  1. il NOMME chaque piece montee par son chassis (`vehicleTurretByChassis`, table de pieces
//     ecrites, une source chacune) et la publie `part = turret` — le client ne la dessine plus ;
//  2. il cherche son PORTEUR — REPLI NOMME ET COMPTE `repli_tourelle_porteur_voisin_de_slot` :
//     le lien parent n est pas lu dans le film (sonde P1-S2, 2026-09-23) ; la piece est creee
//     juste AVANT son chassis, qui prend le slot suivant (+1) ou le surlendemain quand une autre
//     piece s intercale (+2), et la famille du voisin doit etre celle que la piece attend ; un
//     voisin qui n est pas NE AVEC la piece (meme instant, meme point) est refuse et COMPTE
//     (`turretCarrierBirthMismatch`) : c est le temoin INDEPENDANT du voisinage de slot ;
//  3. il REPORTE les episodes d occupation de la piece sur le porteur (l artilleur est a bord du
//     vehicule), SANS leur siege : le siege lu etait celui de la tourelle, et le publier sur le
//     porteur ferait de l artilleur un conducteur — un episode de REPLI dont la montee a bord ne
//     se voit pas PRES du porteur est ecarte (`vehicle_turrets_boarding.go`, 2026-09-24) ;
//  4. il nomme la VARIANTE du porteur quand la piece la designe (le lance-roquettes fait le
//     Rockethog, le canon Gauss le Warthog Gauss).
//
// PUR : aucune I/O. Appele par `buildVehicleTracks` apres la fusion des relais et AVANT le
// comptage, pour que la couverture decrive ce qui est publie.

import (
	"sort"
	"strconv"

	"levelup/go-api/internal/games/halo_infinite/film/internal/facts/fallback"
)

// VehiclePartTurret : la seule valeur de `VehicleTrack.Part`.
const VehiclePartTurret = "turret"

// vehicleTurret est ce que le depot sait d une piece montee.
type vehicleTurret struct {
	// carrier est la FAMILLE du chassis qui la porte. Un voisin d une autre famille n est pas
	// son porteur, quel que soit son slot.
	carrier string
	// variant est la variante que la piece DONNE a son porteur, vide quand elle n en nomme pas.
	variant string
}

// vehicleTurretByChassis nomme les pieces montees par leur mot d identite `MPPWord32`.
//
// CHAQUE ENTREE CITE SA PIECE, et aucune n est devinee :
//   - Warthog : `WARTHOG_FINAL_2026-09-02.md` § 1 (lignes 13, 15, 16 de la table des chassis) —
//     `vehi bcfb852f -> weap c7d50912` (lance-roquettes, banque `veh_un_rockethog`),
//     `vehi dd7f9102 -> weap 0c6fd911` (LAAG, classe `turret`), `vehi 64b925eb -> weap 8647925a`
//     (canon Gauss) ;
//   - Falcon : `labels.tsv` du depot — `vehi 1a043c29` porte la banque
//     `sb_010_veh_un_falcongrenadelauncher` (tourelle laterale, classe `turret`), `vehi f4c45d71`
//     la banque `sb_010_veh_un_falconlmgturret` ;
//   - Wraith : `ASSEMBLAGE_ENFANTS_2026-09-01.md` (`wraith_g` `0x001b33fc`, tourelle plasma) et
//     `labels.tsv` (`vehi 233c877d`, banque `sb_010_veh_cv_wraith`, piece du mortier) ;
//   - Scorpion : `ASSEMBLAGE_ENFANTS_2026-09-01.md` (`scorpion_c` `0x0000d4ff` le canon,
//     `scorpion_g` `0x0000d500` le collier de tourelle).
var vehicleTurretByChassis = map[uint32]vehicleTurret{
	0xbcfb852f: {carrier: familleWarthog, variant: familleRockethog},
	0xdd7f9102: {carrier: familleWarthog},
	0x64b925eb: {carrier: familleWarthog, variant: familleWarthogGauss},
	0x1a043c29: {carrier: familleFalcon},
	0xf4c45d71: {carrier: familleFalcon},
	0x001b33fc: {carrier: familleWraith},
	0x233c877d: {carrier: familleWraith},
	0x0000d4ff: {carrier: familleScorpion},
	0x0000d500: {carrier: familleScorpion},
}

// vehicleTurretSlotReach : les voisins de slot ou chercher le porteur (+1, puis +2). MESURE du
// 2026-09-23 (111 documents) : la LAAG a son Warthog en +1 dans 44 vies sur 45, la tourelle du
// Falcon `1a043c29` a le sien en +2 derriere sa jumelle `f4c45d71`. C est le parametre du repli,
// nomme pour que son elargissement soit une decision datee.
const vehicleTurretSlotReach = 2

// LA NAISSANCE COMMUNE, TEMOIN INDEPENDANT DU VOISINAGE DE SLOT (revue adverse du lot M4a, F5).
// Une piece montee et son chassis sont crees ENSEMBLE : MESURE du 2026-09-23 sur les 111 documents
// du parc, 141 pieces posees sur 141 — naissances au meme point (ecart 0,0 m, maximum 0,0 m) et au
// meme instant (ecart <= 1 frame). Un voisin de slot qui n est pas ne avec la piece n est PAS son
// porteur : le poser la ferait tirer depuis un autre vehicule. Les deux tolerances sont le
// parametre du repli `repli_tourelle_porteur_voisin_de_slot` (meme registre, meme critere de
// retrait) ; la distance ne se juge que si les DEUX naissances sont lues.
const (
	vehicleTurretBirthFrames = 1
	vehicleTurretBirthMeters = 1.0
)

// vehicleTurretOf rend la piece montee d une vie, ou faux si la vie n en est pas une.
func vehicleTurretOf(tr VehicleTrack) (vehicleTurret, bool) {
	if tr.Chassis == "" {
		return vehicleTurret{}, false
	}
	id, err := strconv.ParseUint(tr.Chassis, 16, 32)
	if err != nil {
		return vehicleTurret{}, false
	}
	t, ok := vehicleTurretByChassis[uint32(id)]
	return t, ok
}

// turretTally est le bilan de la pose des pieces montees, verse a la couverture.
type turretTally struct {
	turrets, onCarrier, birthMismatch, rides, variants int
	// kept ventile les episodes GARDES sur la piece par raison ; `dropped()` est leur somme.
	kept turretRidesKept
}

// turretRidesKept : les refus de report qui GARDENT l episode d artilleur sur sa piece (cf.
// `moveTurretRides`). Le troisieme refus publie, `turretRidesNotRideable`, vaut 0 depuis le
// 2026-09-24 et n a plus de branche (cf. `moveTurretRides`).
type turretRidesKept struct {
	outOfWindow, alreadyAboard int
}

func (k turretRidesKept) total() int { return k.outOfWindow + k.alreadyAboard }

func (k *turretRidesKept) add(o turretRidesKept) {
	k.outOfWindow += o.outOfWindow
	k.alreadyAboard += o.alreadyAboard
}

// applyTo verse le bilan a la couverture du calque (`coverage.vehicles`).
func (t turretTally) applyTo(cov *VehicleCoverage) {
	cov.Turrets, cov.TurretsOnCarrier = t.turrets, t.onCarrier
	cov.TurretCarrierBirthMismatch = t.birthMismatch
	cov.TurretRides, cov.Variants = t.rides, t.variants
	cov.TurretRidesDropped = t.kept.total()
	cov.TurretRidesOutOfWindow = t.kept.outOfWindow
	cov.TurretRidesAlreadyAboard = t.kept.alreadyAboard
}

// poseTurretsOnCarriers nomme les pieces montees, trouve leur porteur, y reporte leurs occupants
// et nomme la variante du porteur. `tracks` est modifiee en place ; l ordre ne change pas.
// `anchors` porte le nuage des bipedes qui juge la montee a bord d un episode de repli.
func poseTurretsOnCarriers(
	tracks []VehicleTrack, anchors vehicleBoardingAnchors, fb *fallback.Compteur,
) turretTally {
	var tally turretTally
	bySlot := vehicleTracksBySlot(tracks)
	for i := range tracks {
		spec, ok := vehicleTurretOf(tracks[i])
		if !ok {
			continue
		}
		tally.turrets++
		tracks[i].Part = VehiclePartTurret
		c, ok, mismatch := carrierOfTurret(tracks, bySlot, i, spec, fb)
		if mismatch {
			tally.birthMismatch++
		}
		if !ok {
			continue
		}
		tally.onCarrier++
		carrier := &tracks[c]
		tracks[i].Carrier = &VehicleLifeRef{Slot: carrier.Slot, Gen: carrier.Gen}
		if spec.variant != "" && carrier.Variant == "" {
			carrier.Variant = spec.variant
			tally.variants++
		}
		moved, kept := moveTurretRides(&tracks[i], carrier, anchors, fb)
		tally.rides += moved
		tally.kept.add(kept)
	}
	return tally
}

// vehicleTracksBySlot indexe les vies par slot (rangs dans `tracks`).
func vehicleTracksBySlot(tracks []VehicleTrack) map[uint32][]int {
	out := map[uint32][]int{}
	for i, tr := range tracks {
		out[tr.Slot] = append(out[tr.Slot], i)
	}
	return out
}

// carrierOfTurret rend le rang du porteur de la piece `i`, par le REPLI du voisin de slot. Le
// troisieme retour dit qu un voisin de la bonne famille, present au meme moment, a ete REFUSE
// parce qu il n est pas ne avec la piece (`vehicleBornTogether`).
//
// LE REPLI SE COMPTE A CHAQUE PORTEUR QU IL DECIDE : c est lui, et non une lecture, qui dit
// « ce vehicule porte cette tourelle ». Une piece sans candidat n est pas un declenchement : le
// repli n a rien decide, et la couverture la compte a part (`turrets - turretsOnCarrier`).
func carrierOfTurret(
	tracks []VehicleTrack, bySlot map[uint32][]int, i int, spec vehicleTurret, fb *fallback.Compteur,
) (carrier int, found, birthMismatch bool) {
	t := tracks[i]
	for d := uint32(1); d <= vehicleTurretSlotReach; d++ {
		for _, c := range bySlot[t.Slot+d] {
			cand := tracks[c]
			if cand.Part != "" || cand.Family != spec.carrier || !vehicleWindowsOverlap(t, cand) {
				continue
			}
			if !vehicleBornTogether(t, cand) {
				birthMismatch = true
				continue
			}
			fb.Declenche(fallback.NomTourellePorteurVoisinDeSlot)
			return c, true, birthMismatch
		}
	}
	return 0, false, birthMismatch
}

// vehicleBornTogether dit si deux vies sont nees ensemble : au meme instant, et au meme point
// quand les deux naissances sont lues (cf. `vehicleTurretBirthFrames`).
func vehicleBornTogether(a, b VehicleTrack) bool {
	dt := a.T0 - b.T0
	if dt < -vehicleTurretBirthFrames || dt > vehicleTurretBirthFrames {
		return false
	}
	if a.Spawn == nil || b.Spawn == nil {
		return true
	}
	dx, dy := float64(a.Spawn.X-b.Spawn.X), float64(a.Spawn.Y-b.Spawn.Y)
	return dx*dx+dy*dy <= vehicleTurretBirthMeters*vehicleTurretBirthMeters
}

// vehicleWindowsOverlap dit si deux vies coexistent a au moins une frame d affichage.
func vehicleWindowsOverlap(a, b VehicleTrack) bool {
	return a.T0 <= b.T1Max && b.T0 <= a.T1Max
}

// moveTurretRides reporte les occupants de la piece sur son porteur. Rend le nombre d episodes
// reportes et celui des episodes GARDES sur la piece, ventile par raison.
//
// DEUX REFUS GARDENT L EPISODE SUR LA PIECE (publiee, et que le client ne dessine pas) — il est
// cru VRAI, seulement pas reportable :
//   - un episode HORS DE LA FENETRE du porteur : le reporter l amputerait ou l effacerait. Cause
//     amont mesuree (revue adverse du lot M4a, F4) : la vie publiee d un chassis peut s arreter
//     (`end = unknown`) bien avant celle de sa tourelle — 13 des 23 episodes gardes hors Falcon au
//     parc du 2026-09-23 ;
//   - un occupant DEJA a bord du porteur au meme instant (le conducteur entre a la naissance, le
//     trou de position le designe aussi pour la piece nee au meme point) n y monte pas deux fois.
//
// UNE GARDE L ECARTE, parce qu il est cru FAUX : un episode de REPLI dont la montee a bord ne se
// voit pas PRES du porteur (dernier point de l occupant perime, ou a plus de 3 m) n a pas de
// montee a bord a affirmer, ni sur le porteur ni sur la piece — laisse sur la piece, il poserait
// encore les tirs de ce joueur sur le porteur (`shotHolder`). `turretRideBoardsCarrier`, repli
// nomme `repli_tourelle_montee_loin_du_porteur`, compte au registre (2026-09-24, revue adverse
// du lot M7b, RR-M7b-01).
//
// LE REFUS « PORTEUR NON PILOTABLE » A ETE RETIRE LE 2026-09-24 (revue adverse du lot M7b,
// RR-M7b-04) : depuis que le Falcon est pilotable, aucune piece de `vehicleTurretByChassis` n a
// de porteur non pilotable, et `carrierOfTurret` n elit qu un porteur de la famille que la table
// attend — la branche ne pouvait plus s ouvrir. L invariant vit dans
// `TestPiecesMonteesOntUnPorteurPilotable` : une piece ajoutee a un Pelican, un Phantom ou un Skiff
// le fait rougir, et c est une DECISION a ecrire. Son compteur publie
// `coverage.vehicles.turretRidesNotRideable` reste dans la forme 69 a 0 : il sortira a la prochaine
// montee de schema (cf. `VehicleCoverage`).
func moveTurretRides(
	turret, carrier *VehicleTrack, anchors vehicleBoardingAnchors, fb *fallback.Compteur,
) (moved int, kept turretRidesKept) {
	if len(turret.Rides) == 0 {
		return 0, kept
	}
	ref := &VehicleLifeRef{Slot: turret.Slot, Gen: turret.Gen}
	var restent []VehicleRide
	var ajout []VehicleRide
	for _, r := range turret.Rides {
		bornes := clampVehicleRides([]VehicleRide{r}, carrier.T0, carrier.T1Max)
		// LA GARDE DE LA MONTEE A BORD PASSE EN DERNIER : elle ne juge que ce qui serait reporte,
		// et les deux refus plus anciens gardent leur compte. Elle ECARTE l episode (cf. en-tete).
		switch {
		case len(bornes) == 0:
			kept.outOfWindow++
			restent = append(restent, r)
			continue
		case occupantAlreadyAboard(carrier.Rides, r):
			kept.alreadyAboard++
			restent = append(restent, r)
			continue
		case !turretRideBoardsCarrier(r, *carrier, anchors, fb):
			continue
		}
		b := bornes[0]
		b.Seat = nil
		b.Turret = ref
		ajout = append(ajout, b)
	}
	turret.Rides = restent
	if len(ajout) == 0 {
		return 0, kept
	}
	carrier.Rides = append(carrier.Rides, ajout...)
	sort.SliceStable(carrier.Rides, func(a, b int) bool { return carrier.Rides[a].T0 < carrier.Rides[b].T0 })
	return len(ajout), kept
}

// occupantAlreadyAboard dit si l occupant de `r` a deja un episode du porteur qui RECOUVRE le sien.
//
// BORNES STRICTES (revue adverse du lot M4a, F2) : un CHANGEMENT DE SIEGE — conducteur puis
// artilleur, ou l inverse — laisse deux episodes qui se TOUCHENT a une seule frame (la sortie de
// l un est l entree de l autre). Ce n est pas un doublon : l occupant passe d un poste a l autre du
// meme vehicule. Mesure du 2026-09-23 : 7 des 10 refus « deja a bord » hors Falcon ne faisaient
// que se toucher (l artilleur restait sur la piece cachee, 18 a 21 s sans pion ni cone).
func occupantAlreadyAboard(rides []VehicleRide, r VehicleRide) bool {
	for _, o := range rides {
		if o.Slot == r.Slot && o.T0 < r.T1 && r.T0 < o.T1 {
			return true
		}
	}
	return false
}

// vehicleVariantByWeapon nomme la VARIANTE d un chassis par l ARME DE VEHICULE qu il tire : une
// arme qui n equipe qu une variante la designe sans ambiguite.
//
//   - `0042678E` : les mitrailleuses avant du GUNGOOSE (`CONTACT_ARMES_GUNGOOSE_2026-09-02.md`,
//     l objet `scen` 0x004164ea en zone avant ; mesure du parc 2026-09-23 : 15 tirs en vehicule,
//     tous portes par le chassis `af31ab1a`, famille `mongoose`). Le Gungoose partage le modele du
//     Mongoose (`vehicle_families.go`) : sans son arme, rien ne le distingue.
var vehicleVariantByWeapon = map[uint32]vehicleTurret{
	0x0042678E: {carrier: familleMongoose, variant: familleGungoose},
}

// nameVariantsByWeapon nomme la variante des vies qui ont tire une arme qui la designe. Le tir
// porte la vie par son slot (`Shot.Vehicle`) et son instant ; la famille de la vie doit etre celle
// que l arme attend — une arme lue sur une autre famille ne nomme rien.
func nameVariantsByWeapon(tracks []VehicleTrack, shots []Shot, cov *VehicleCoverage) {
	for _, s := range shots {
		if s.Vehicle == nil {
			continue
		}
		spec, ok := variantOfWeapon(s.Weapon)
		if !ok {
			continue
		}
		for i := range tracks {
			tr := &tracks[i]
			if tr.Slot != *s.Vehicle || s.T < tr.T0 || s.T > tr.T1Max {
				continue
			}
			if tr.Family == spec.carrier && tr.Variant == "" {
				tr.Variant = spec.variant
				if cov != nil {
					cov.Variants++
				}
			}
			break
		}
	}
}

// variantOfWeapon rend la variante designee par une cle d arme de vehicule (`Shot.Weapon`).
func variantOfWeapon(key string) (vehicleTurret, bool) {
	for tag, spec := range vehicleVariantByWeapon {
		if VehicleWeaponKey(tag) == key {
			return spec, true
		}
	}
	return vehicleTurret{}, false
}
