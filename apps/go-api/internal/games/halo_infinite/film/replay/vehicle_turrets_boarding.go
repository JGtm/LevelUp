package replay

// vehicle_turrets_boarding.go — LA MONTEE A BORD D UN ARTILLEUR REPORTE SE VOIT PRES DU PORTEUR
// (reprise du lot M7b des retours du rejeu, revue adverse RR-M7b-01, 2026-09-24).
//
// LE DEFAUT. Une piece montee (tourelle du Warthog, du Falcon, du Wraith, du Scorpion) n a AUCUN
// echantillon de position : un episode de REPLI (`src = proximity`) ne s y accroche donc jamais par
// la geometrie, seulement par le NOM que porte un evenement de sortie. Quand aucun embarquement ne
// date son debut, la machine d etats le fait commencer au DERNIER point replique par l occupant
// avant la sortie (`vehicleLastPointBefore`) — et ce point peut etre celui d une vie finie, loin du
// vehicule. Tant que la piece restait cachee, l episode ne se voyait pas. Reporte sur un porteur
// dessine, il ecrit le nom d un joueur mort sur le vehicule et le montre deux fois sur la carte.
// MESURE du 2026-09-24 (107 documents rejoues des faits) : sur les 18 episodes de repli reportes,
// 12 ont leur dernier point a 0,9-2,2 m du porteur (age 0 frame), 5 a 21,6-90,9 m, 1 n en a aucun
// (l occupant n est pas encore ne) — tous les six sur des Falcon, rendus visibles par le lot M7b.
//
// LA GARDE : un episode de repli n est reporte que si le dernier point replique par l occupant
// avant son debut est FRAIS (au plus `vehicleEventTolMS` avant) et PROCHE du porteur (au plus
// `vehicleEventAnchorRadiusM` en plan, a la position que le client dessine a cet instant). Sinon il
// est ECARTE — ni reporte, ni garde sur sa piece, ou il poserait encore les tirs de ce joueur sur
// le porteur — et le repli nomme se declenche : son compte est publie par `coverage.fallbacks`,
// sans champ neuf (la forme 69 ne bouge pas). Un episode LU (`src = film`) n est pas juge : le
// film ecrit sa montee a bord.
//
// LES DEUX SEUILS NE SONT PAS NEUFS : ce sont ceux de l ancre d evenement (rayon 3 m, table de
// `vehicleEventAnchorRadiusM`) et de la tolerance d appariement (2 s). L ecart mesure (2,2 m contre
// 21,6 m) les laisse loin de toute bordure.

import (
	"sort"

	"levelup/go-api/internal/games/halo_infinite/film/internal/facts/fallback"
	"levelup/go-api/internal/games/halo_infinite/film/internal/grammar"
)

// vehicleBoardingAnchors est ce que la garde consulte : le nuage des BIPEDES par slot (trie par
// instant) et l horloge du document. Vide, aucun episode de repli n est reporte — c est le sens sur
// d une ignorance.
type vehicleBoardingAnchors struct {
	bySlot map[uint32][]grammar.BipedPosition
	clock  replayClock
}

// boardedNear dit si l occupant de `r` a ete vu pres de `carrier` a sa montee a bord.
func (a vehicleBoardingAnchors) boardedNear(r VehicleRide, carrier VehicleTrack) bool {
	if a.clock.step == 0 {
		return false
	}
	startUS := a.clock.origin + uint64(r.T0)*a.clock.step
	endUS := startUS + a.clock.step // exclusif : tout point de la frame `T0` compte
	pts := a.bySlot[r.Slot]
	i := sort.Search(len(pts), func(k int) bool { return pts[k].TimestampUS >= endUS })
	if i == 0 {
		return false
	}
	p := pts[i-1]
	if p.TimestampUS+uint64(vehicleEventTolMS)*1000 < startUS {
		return false
	}
	x, y, ok := vehiclePosAt(carrier, a.clock.frame(p.TimestampUS))
	return ok && planDist(p.X, p.Y, x, y) <= vehicleEventAnchorRadiusM
}

// turretRideBoardsCarrier applique la garde a UN episode de la piece : vrai s il peut etre
// reporte. Le refus declenche le repli nomme (compteur de cuisson, `coverage.fallbacks`).
func turretRideBoardsCarrier(
	r VehicleRide, carrier VehicleTrack, a vehicleBoardingAnchors, fb *fallback.Compteur,
) bool {
	if r.Src != VehicleRideSrcProximity || a.boardedNear(r, carrier) {
		return true
	}
	fb.Declenche(fallback.NomTourelleMonteeLoinDuPorteur)
	return false
}
