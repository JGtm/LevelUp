package replay

// vehicle_rides_aim.go — LA VISEE DE L OCCUPANT PENDANT SON EPISODE, projetee sur la grille du
// document.
//
// # CE QUE LE LOT V11 A TROUVE, ET POURQUOI CE FICHIER EXISTE
//
// Un occupant de vehicule cesse de repliquer sa POSITION monde : c est la primitive du « trou »
// sur laquelle repose toute l attribution des episodes (`vehicle_rides.go`). Le depot en avait
// conclu qu il ne repliquait plus RIEN, et le cone de visee du conducteur retombait donc sur le
// cap du CHASSIS (`vehiclesLayer.vehicleAimAngle`, approximation assumee).
//
// C etait faux, et le point aveugle etait dans le DETECTEUR, pas dans le format :
// `ScanBipedRecords` exige un `i0` ABSOLU et un masque commencant par 0, alors qu un record delta
// n a aucune obligation de declarer `i0`. La forme de masque la plus frequente de la bande bipede
// est `i21,i25` — un record de VISEE SANS POSITION. 22 963 lectures sur `0d76e8f1`, 222,9 par
// slot, contre 0,9 par slot sur une bande FANTOME de meme cardinalite (x261).
// `filmdec.ScanFilmBipedAimOnly` les lit ; ce fichier les publie.
//
// # LES TROIS CHIFFRES QUI FONDENT LA PUBLICATION (V11_ORIENTATION_TOURELLE_2026-09-03)
//
//	JUSTESSE    appariee a la lecture `i21` AVEC position du meme slot a moins de 200 ms, l ecart
//	            median de cap vaut 0,2 a 0,5 deg sur 5 films (R 0,979 a 0,989). TEMOIN par
//	            melange deterministe : 75,7 a 93,7 deg, R 0,011 a 0,134.
//	COUVERTURE  sur les 35 episodes d occupation ATTESTES par la sortie (5 films), 35 / 35 (100 %)
//	            portent au moins une visee A BORD, a 5 a 46 lectures par seconde — quand le meme
//	            episode porte 0 ou 1 lecture `i21` AVEC position.
//	UTILITE     cette visee N EST PAS le cap du chassis : ecart median 15,7 a 21,8 deg, quartile
//	            superieur 39,6 a 52,9 deg, un tiers des instants au-dela de 30 deg. C est
//	            exactement l erreur que faisait le cone du conducteur.
//
// # POURQUOI SUR L EPISODE, ET NULLE PART AILLEURS
//
// `Point.H` ne convient pas : par definition l occupant n a PAS de position pendant l episode, il
// n y a donc aucun point ou accrocher l angle. `VehicleTrack.Samples` non plus : la visee est
// celle de l OCCUPANT et non du vehicule — un vehicule porte plusieurs occupants, donc plusieurs
// visees simultanees et distinctes. L EPISODE D OCCUPATION est le seul objet du document qui
// connaisse a la fois l occupant, son vehicule et sa fenetre.
//
// PUR : aucune I/O. Les lectures entrent deja decodees (cf. `build_vehicles.go`).

import (
	"sort"

	"levelup/go-api/internal/analysis/filmdec"
)

// vehicleAimBySlot indexe les lectures de visee par slot d occupant et TRIE chaque liste par
// instant. Le tri n est pas cosmetique : l echantillonnage ci-dessous applique « le premier
// observe gagne », qui n est deterministe que sur une entree ordonnee.
func vehicleAimBySlot(aims []filmdec.BipedAim) map[uint32][]filmdec.BipedAim {
	if len(aims) == 0 {
		return nil
	}
	out := make(map[uint32][]filmdec.BipedAim, 16)
	for _, a := range aims {
		out[a.Slot] = append(out[a.Slot], a)
	}
	for s := range out {
		v := out[s]
		sort.SliceStable(v, func(i, j int) bool { return v[i].TimestampUS < v[j].TimestampUS })
	}
	return out
}

// vehicleRideAimOf projette les lectures de visee d UN occupant sur la grille de frames du
// document, DANS la fenetre [startUS, endUS] de son episode.
//
// MEME PAS ET MEME REGLE QUE LES AUTRES SERIES DU CALQUE (`vehicleSamplesOf`, `decimateTracks`) :
// un point par frame, LE PREMIER OBSERVE GAGNE. Le film replique la visee a 5 a 46 lectures par
// seconde ; la grille du document en compte 10 par seconde (`FrameIntervalMS` = 100 ms), publier
// le flux brut multiplierait le poids du calque pour un rendu que le client n affiche pas plus
// finement.
//
// AUCUNE INTERPOLATION, ni ici ni au client : interpoler deux caps ferait tourner le cone par le
// chemin le plus court a travers 0/360 deg, un artefact que le film ne montre pas. Une frame sans
// lecture n en porte pas — le client maintient la derniere, ou retombe sur le cap du chassis.
func vehicleRideAimOf(
	aims []filmdec.BipedAim, startUS, endUS uint64, clock replayClock,
) []VehicleAim {
	if len(aims) == 0 || clock.step == 0 || endUS < startUS {
		return nil
	}
	lo := sort.Search(len(aims), func(i int) bool { return aims[i].TimestampUS >= startUS })
	var out []VehicleAim
	lastFr := -1
	for _, a := range aims[lo:] {
		if a.TimestampUS > endUS {
			break
		}
		fr := clock.frame(a.TimestampUS)
		if fr <= lastFr {
			continue
		}
		lastFr = fr
		out = append(out, VehicleAim{
			T: fr, H: headingForJSON(a.AimHeadingDeg()), P: pitchForJSON(a.AimPitchDeg()),
		})
	}
	return out
}

// clampVehicleRideAim ramene la serie de visee dans la fenetre d affichage [t0, t1] de son
// episode deja clampe (`clampVehicleRides`).
//
// SANS LUI, LA SERIE SURVIVRAIT A SON EPISODE : un episode date avant `T0` de la vie est ramene a
// `T0`, mais ses lectures de visee, elles, garderaient leurs frames d origine — le client
// dessinerait un cone a une image ou le document affirme que l episode n a pas encore commence.
func clampVehicleRideAim(aim []VehicleAim, t0, t1 int) []VehicleAim {
	if len(aim) == 0 {
		return nil
	}
	if aim[0].T >= t0 && aim[len(aim)-1].T <= t1 {
		return aim // cas nominal : rien a couper, aucune allocation
	}
	out := make([]VehicleAim, 0, len(aim))
	for _, a := range aim {
		if a.T < t0 || a.T > t1 {
			continue
		}
		out = append(out, a)
	}
	if len(out) == 0 {
		return nil
	}
	return out
}
