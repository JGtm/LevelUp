package service

// replay_vehicle_scenery_aloft.go — LA SECONDE REGLE DU DECOR : LE VEHICULE TENU EN L AIR A VIDE
// (chantier « Falcon de Behemoth », 2026-10-02). Pure, comme la regle de pose
// (replay_vehicle_scenery_rule.go) : aucun fichier, aucune base, rien que le document.
//
// # LE FAIT QU ELLE TRANCHE
//
// Un vehicule que personne n occupe obeit a la pesanteur : il repose a l altitude ou il est ne, ou
// plus bas. Une vie qui, SANS OCCUPANT, ne se pose jamais a l altitude de sa naissance mais
// seulement plus haut — et sans s eloigner en plan de son point de naissance — n est pas un
// vehicule libre de la partie : la carte la tient (expulsee par la geometrie a l apparition, puis
// maintenue en vol stationnaire). Personne ne la prend, et le jeu la retire puis la refait naitre
// au meme point, dans le meme etat.
//
// Le film ne dit pas « non pilotable » (sonde C2 du 2026-09-23, cf. la regle de pose) : la regle lit
// le MOUVEMENT publie, jamais la carte ni la famille du chassis. Elle ne se confond pas avec la
// regle de pose : celle-ci exige une pose SEULE (un echantillon), celle-ci une vie qui a bouge.
//
// # LES CONDITIONS, TOUTES NECESSAIRES
//
//  1. un vehicule (pas une piece montee : une tourelle suit son porteur) ;
//  2. aucun episode d occupation publie ;
//  3. au moins une STATION (se tenir au moins [sceneryStandMinMs] dans une bande de
//     [sceneryStandMaxDZ], la notion du sol foule), l echantillon final tenu jusqu a `T1` — un
//     vehicule ne se deplace pas pendant un silence de replication ;
//  4. la plus BASSE station est au moins [aloftMinRiseM] au-dessus de la naissance (`Spawn`, le
//     record de creation ; a defaut le premier echantillon) ;
//  5. aucun echantillon a plus de [aloftMaxDriftM] en plan de la naissance.
//
// La condition 5 protege les vols reels : un episode d occupation que la primitive n attribue pas
// laisse `Rides` vide, mais un vol s eloigne de sa naissance (cf. [aloftMaxDriftM]).
//
// La raison publiee est [sceneryReasonAloftUnoccupied] ; la vie est comptee dans `hidden`. C est
// un repli nomme (`repli_decor_tenu_en_l_air_a_vide`, registre facts/fallback) : il infere le decor
// d un mouvement, faute d un champ du film qui le dise.

import (
	"math"

	"levelup/go-api/internal/games/halo_infinite/film/replay"
)

// sceneryReasonAloftUnoccupied : raison publiee d une vie tenue en l air a vide (cf. en-tete).
const sceneryReasonAloftUnoccupied = "aloft_unoccupied"

// aloftMinRiseM / aloftMaxDriftM : CONSTANTES MESUREES du repli
// `repli_decor_tenu_en_l_air_a_vide` (instrument `TestDecorTenuEnLAirParc`, 1 227 documents au
// schema 76, 6 612 vies de vehicule, 2026-10-02).
//   - Elevation de la plus basse station : 1,88 a 2,27 m pour les 15 vies tenues en l air ; au plus
//     0,58 m pour toute autre vie inoccupee qui reste a moins de 20 m de sa naissance. 1,2 m coupe
//     l ecart en deux.
//   - Derive en plan : 1,77 a 4,86 m pour les vies tenues en l air ; 27 m au moins pour toute vie
//     inoccupee posee plus de 1 m au-dessus de sa naissance sans etre masquee (vols dont
//     l occupation n est pas lue). 8 m, du cote des vies tenues : dans le doute, la vie s affiche.
const (
	aloftMinRiseM  float32 = 1.2
	aloftMaxDriftM float32 = 8.0
)

// aloftVehicles rend les vies du document tenues en l air a vide.
func aloftVehicles(doc *replay.ReplayDocument) []replay.VehicleTrack {
	if doc == nil || doc.FrameIntervalMS <= 0 {
		return nil
	}
	minFrames := standMinFrames(doc.FrameIntervalMS)
	var out []replay.VehicleTrack
	for _, v := range doc.Vehicles {
		if vehicleIsAloftUnoccupied(v, minFrames) {
			out = append(out, v)
		}
	}
	return out
}

// standMinFrames : la duree d une station ([sceneryStandMinMs]) en images du document.
func standMinFrames(frameIntervalMS int) int {
	return int(math.Ceil(sceneryStandMinMs / float64(frameIntervalMS)))
}

// vehicleIsAloftUnoccupied applique les cinq conditions de l en-tete a une vie.
func vehicleIsAloftUnoccupied(v replay.VehicleTrack, minFrames int) bool {
	if v.Part != "" || len(v.Rides) > 0 || len(v.Samples) == 0 {
		return false
	}
	bx, by, bz := vehicleBirth(v)
	maxDrift2 := float64(aloftMaxDriftM) * float64(aloftMaxDriftM)
	for _, s := range v.Samples {
		dx, dy := float64(s.X-bx), float64(s.Y-by)
		if dx*dx+dy*dy > maxDrift2 {
			return false
		}
	}
	rest, ok := lowestStand(vehicleHeldPoints(v), minFrames)
	return ok && rest >= bz+aloftMinRiseM
}

// vehicleBirth rend la naissance de la vie : le record de creation quand il a ete lu, sinon le
// premier echantillon.
func vehicleBirth(v replay.VehicleTrack) (x, y, z float32) {
	if v.Spawn != nil {
		return v.Spawn.X, v.Spawn.Y, v.Spawn.Z
	}
	s := v.Samples[0]
	return s.X, s.Y, s.Z
}

// vehicleHeldPoints rend la trajectoire de la vie sous la forme que lit [lowestStand], le dernier
// echantillon TENU jusqu a `T1` (derniere preuve de presence) : un vehicule immobile n est plus
// replique, et son repos ne se lit qu ainsi.
func vehicleHeldPoints(v replay.VehicleTrack) []replay.Point {
	pts := make([]replay.Point, 0, len(v.Samples)+1)
	for _, s := range v.Samples {
		pts = append(pts, replay.Point{T: s.T, X: s.X, Y: s.Y, Z: s.Z})
	}
	if last := v.Samples[len(v.Samples)-1]; v.T1 > last.T {
		pts = append(pts, replay.Point{T: v.T1, X: last.X, Y: last.Y, Z: last.Z})
	}
	return pts
}
