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
//     piece s intercale (+2), et la famille du voisin doit etre celle que la piece attend ;
//  3. il REPORTE les episodes d occupation de la piece sur le porteur (l artilleur est a bord du
//     vehicule), SANS leur siege : le siege lu etait celui de la tourelle, et le publier sur le
//     porteur ferait de l artilleur un conducteur ;
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
	turrets, onCarrier, rides, dropped, variants int
}

// poseTurretsOnCarriers nomme les pieces montees, trouve leur porteur, y reporte leurs occupants
// et nomme la variante du porteur. `tracks` est modifiee en place ; l ordre ne change pas.
func poseTurretsOnCarriers(tracks []VehicleTrack, fb *fallback.Compteur) turretTally {
	var tally turretTally
	bySlot := vehicleTracksBySlot(tracks)
	for i := range tracks {
		spec, ok := vehicleTurretOf(tracks[i])
		if !ok {
			continue
		}
		tally.turrets++
		tracks[i].Part = VehiclePartTurret
		c, ok := carrierOfTurret(tracks, bySlot, i, spec, fb)
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
		moved, dropped := moveTurretRides(&tracks[i], carrier)
		tally.rides += moved
		tally.dropped += dropped
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

// carrierOfTurret rend le rang du porteur de la piece `i`, par le REPLI du voisin de slot.
//
// LE REPLI SE COMPTE A CHAQUE PORTEUR QU IL DECIDE : c est lui, et non une lecture, qui dit
// « ce vehicule porte cette tourelle ». Une piece sans candidat n est pas un declenchement : le
// repli n a rien decide, et la couverture la compte a part (`turrets - turretsOnCarrier`).
func carrierOfTurret(
	tracks []VehicleTrack, bySlot map[uint32][]int, i int, spec vehicleTurret, fb *fallback.Compteur,
) (int, bool) {
	t := tracks[i]
	for d := uint32(1); d <= vehicleTurretSlotReach; d++ {
		for _, c := range bySlot[t.Slot+d] {
			cand := tracks[c]
			if cand.Part != "" || cand.Family != spec.carrier || !vehicleWindowsOverlap(t, cand) {
				continue
			}
			fb.Declenche(fallback.NomTourellePorteurVoisinDeSlot)
			return c, true
		}
	}
	return 0, false
}

// vehicleWindowsOverlap dit si deux vies coexistent a au moins une frame d affichage.
func vehicleWindowsOverlap(a, b VehicleTrack) bool {
	return a.T0 <= b.T1Max && b.T0 <= a.T1Max
}

// moveTurretRides reporte les occupants de la piece sur son porteur. Rend le nombre d episodes
// reportes et celui des episodes GARDES sur la piece.
//
// DEUX REFUS, ET AUCUN N EFFACE L EPISODE (il reste sur la piece, qui est publiee) :
//   - un porteur NON PILOTABLE ne porte aucun occupant (`vehicleFamilyIsRideable`, decision du
//     2026-09-02 sur le Falcon, le Pelican, le Phantom et le Skiff) — le document ne l affirmera
//     pas plus par la tourelle que par le chassis ;
//   - un occupant DEJA a bord du porteur au meme instant (le conducteur entre a la naissance, le
//     trou de position le designe aussi pour la piece nee au meme point) n y monte pas deux fois.
func moveTurretRides(turret, carrier *VehicleTrack) (moved, kept int) {
	if len(turret.Rides) == 0 {
		return 0, 0
	}
	if !vehicleFamilyIsRideable(carrier.Family) {
		return 0, len(turret.Rides)
	}
	ref := &VehicleLifeRef{Slot: turret.Slot, Gen: turret.Gen}
	var restent []VehicleRide
	var ajout []VehicleRide
	for _, r := range turret.Rides {
		// UN EPISODE HORS DE LA FENETRE DU PORTEUR reste sur la piece : le reporter l amputerait
		// ou l effacerait, et le compte des episodes publies ne doit rien perdre en route.
		bornes := clampVehicleRides([]VehicleRide{r}, carrier.T0, carrier.T1Max)
		if len(bornes) == 0 || occupantAlreadyAboard(carrier.Rides, r) {
			restent = append(restent, r)
			continue
		}
		b := bornes[0]
		b.Seat = nil
		b.Turret = ref
		ajout = append(ajout, b)
	}
	turret.Rides = restent
	if len(ajout) == 0 {
		return 0, len(restent)
	}
	carrier.Rides = append(carrier.Rides, ajout...)
	sort.SliceStable(carrier.Rides, func(a, b int) bool { return carrier.Rides[a].T0 < carrier.Rides[b].T0 })
	return len(ajout), len(restent)
}

// occupantAlreadyAboard dit si l occupant de `r` a deja un episode du porteur qui recouvre le sien.
func occupantAlreadyAboard(rides []VehicleRide, r VehicleRide) bool {
	for _, o := range rides {
		if o.Slot == r.Slot && o.T0 <= r.T1 && r.T0 <= o.T1 {
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
