package service

// replay_vehicle_scenery_rule.go — LA REGLE DU DECOR DE CARTE (retours du rejeu, lot M7,
// 2026-09-24), pure : aucun fichier, aucune base. Elle remplace la regle cliente du lot L1.3
// (`vehiclesLayer.vehicleIsScenery`), dont c etait le critere de retrait : « quand le producteur
// publie lui-meme le decor de carte, ce predicat lit ce marqueur et ses conditions disparaissent
// d ici ». Le chargement de la zone vit a cote (replay_vehicle_scenery.go).
//
// # CE QUE LE FILM DIT, ET CE QU IL NE DIT PAS (sonde C2 du 2026-09-23)
//
// Un vehicule de decor est un vrai vehicule physique, pose PAR LA CARTE au chargement : le film
// replique sa POSE (7 a 117 records delta, en 3 a 4 s) AVANT L ORIGINE du match, puis plus rien.
// La publication n en garde qu UN echantillon, ramene a la naissance (`t0`, frame 0) : c est ce
// que les conditions de pose lisent. Aucun champ du film ne dit « non jouable » : le champ de
// naissance le plus proche (index de placement Forge, `FUN_14080d524`) dit « pose par la carte »,
// et des tourelles ACTIVES le portent aussi — jamais une preuve de decor a lui seul.
//
// # LA REGLE : POSE SEULE ET HORS DE LA ZONE JOUABLE (decision utilisateur du 2026-09-24)
//
// Une vie est du decor quand elle remplit les CINQ conditions de pose (L1.3 : un seul echantillon,
// a sa naissance, nee a la frame 0, vivante jusqu a la fin du film, aucun occupant) ET qu elle se
// trouve HORS DE LA ZONE JOUABLE. Sur Behemoth (hors Super Fiesta) des Mongoose et des Gungoose
// sont toujours poses au depart : gares dans l aire de jeu, ils restent affiches meme si personne
// n y touche.
//
// # LA ZONE JOUABLE EN PLAN : LA MATIERE DU FOND PUBLIE (parc du 2026-09-24, 107 documents)
//
// La matiere praticable de la carte est le masque du fond publie (alpha > 0) ferme au rayon
// canonique du depot (`mapdecoupe.ToleranceParDefaut`, 4 m) — une reference VERSIONNEE, disponible
// en production sans le jeu, independante du match. Ce que la cuisson y a mis (cmd/mapfond-build) :
// la geometrie de la tranche jouee, ROGNEE aux zones de callout dilatees (`rogneAuxZones`, marge
// 1 m, Starboard et Goliath au reglage du 2026-08-30) ou aux volumes de mort, puis le CADRE de
// l image est la boite de cette matiere elargie de `himap.MargeCadreUtile` (6 m, `himap.CadreUtile`,
// depuis le 2026-08-26). Un point hors du cadre est donc hors de toute matiere publiee, a plus de
// 6 m d elle sur un fond rogne : « hors cadre » n est pas un choix de rendu independant, c est le
// cas limite de « hors matiere », et la fermeture (4 m) ne l atteint jamais.
//
// CE QUI DECIDE AUJOURD HUI, DIT SANS DETOUR (revue RR-M7-05) : au parc, les 12 decors de Starboard
// sont HORS DU CADRE (12 a 16 m au sud du bord de l image) ; aucun decor n est decide par un vide
// DANS le cadre, et la fermeture n en sauve aucun. Les deux cas sont tenus par des tests sur un fond
// synthetique (un vide dans le cadre masque, un trou plus etroit que 8 m ne masque pas).
// Les autres references du depot ne conviennent pas : `map_geometry/` ne porte les props que d une
// carte, `map_positions_jouees.json` qu une carte (Dredge), les paves de callout debordent sur le
// vide et n existent pas pour les cartes Forge. Les bornes jouees du match non plus : un Mongoose
// pose de Behemoth y est a 2,4 m du bord (f2966f08).
//
// # EN HAUTEUR : LE SOL FOULE DU MATCH, REPLI NOMME (registre facts/fallback)
//
// AUCUNE reference ne publie l altitude praticable d une cellule (le sidecar n en porte qu une par
// carte, `playLevelZ` ; constat ecrit dans `mapdecoupe/masque.go`). Le Wasp de Goliath est DANS le
// masque, 3,03 m sous le sol. Le repli `repli_decor_sous_le_sol_foule_du_match` lit donc le sol
// FOULE du match : la plus basse altitude ou un joueur est RESTE (au moins [sceneryStandMinMs] dans
// une bande de [sceneryStandMaxDZ]) — jamais la plus basse position publiee, que la moindre chute
// dans le vide tire vers le bas (revue RR-M7-04 : Streets -33,76 m contre -0,26 m selon le match,
// Launch Site -14,29 contre -3,92, Starboard 70,42 contre 80,63 ; une chute sur Goliath aurait
// rendu le Wasp). MESURE sur les 107 documents : le sol foule d une carte varie de 0,34 m au plus
// d un match a l autre (Cliffhanger), 1,42 m sur deux matchs ecourtes de Lattice (767 et 882
// points) ; le seuil de duree est au milieu d un plateau (0,7 s a 1,5 s donnent les memes sols).
// CE QUI RESTE DEPENDANT DU MATCH, ET C EST ECRIT AU REGISTRE : un match ou des joueurs se TIENNENT
// plus bas que le Wasp de Goliath le rendrait — c est qu il y aurait alors du jeu a sa hauteur. Un
// seul match de Goliath au parc. Critere de retrait : la zone publiee porte l altitude de sa
// matiere, et le test de hauteur la lit au lieu du match.
//
// Une carte SANS zone connue (aucun fond publie, fond illisible) ne masque RIEN : repli
// `repli_decor_carte_sans_zone_affiche`, compte dans `zoneUnknown`. Un document ou personne ne s est
// tenu nulle part ne sait pas son sol : le test de hauteur ne s applique pas ([sceneryFloorUnknown]).

import (
	"math"

	"levelup/go-api/internal/games/halo_infinite/film/replay"
)

// playArea est la zone jouable EN PLAN d une carte.
type playArea interface {
	// Praticable dit si la position monde (x, y) tombe sur de la matiere praticable.
	Praticable(x, y float64) bool
}

// Etats de la zone et du sol, et raisons d un masquage : valeurs PUBLIEES (cf.
// `replay.VehicleScenery`), listes fermees.
const (
	sceneryZoneMap                = "map"
	sceneryZoneUnknown            = "unknown"
	sceneryFloorPlayed            = "played"
	sceneryFloorUnknown           = "unknown"
	sceneryReasonOffPlayArea      = "off_play_area"
	sceneryReasonBelowPlayedFloor = "below_played_floor"
)

// sceneryFloorToleranceM : de combien de metres une pose doit passer SOUS le sol foule pour etre
// « sous le sol ». CONSTANTE MESUREE du repli `repli_decor_sous_le_sol_foule_du_match` (raison
// [sceneryReasonBelowPlayedFloor], comptee dans `hidden`). Parc du 2026-09-24 (107 documents, 260
// vies de vehicule en jeu ramenees a leur premier echantillon) : la plus basse est 0,10 m SOUS le
// sol foule (Wasp de 8a485699 sur son socle, Launch Site) ; le seul decor sous le sol est 3,03 m
// dessous (Wasp de Goliath). 1,5 m coupe l ecart en deux. Critere de retrait : celui du repli.
const sceneryFloorToleranceM float32 = 1.5

// sceneryStandMinMs / sceneryStandMaxDZ : ce qu est « se tenir » pour le sol foule — rester au
// moins 1 s dans une bande de 0,3 m d altitude. Une chute ne s y tient jamais ; un pas, une course
// a plat, une attente, si. CONSTANTES MESUREES du meme repli : au parc, 0,7 s a 1,5 s rendent les
// memes sols (plateau ; 0,5 s laisse passer une chute de Cliffhanger, 5 s perd des sols joues), et
// la bande est indifferente de 0,1 a 0,5 m.
const (
	sceneryStandMinMs float64 = 1000
	sceneryStandMaxDZ float32 = 0.3
)

// vehicleIsPosedOnly dit si une vie remplit les CINQ conditions de pose : un seul echantillon, a sa
// naissance, nee a la frame 0 (le decor existe des le chargement de la carte), vivante jusqu a la
// fin du film, aucun occupant. Un vehicule SIMULE — meme gare et jamais occupe — est replique
// apres l origine et porte plusieurs echantillons : il n est jamais candidat.
func vehicleIsPosedOnly(v replay.VehicleTrack) bool {
	return len(v.Samples) == 1 &&
		v.T0 <= 0 &&
		v.Samples[0].T == v.T0 &&
		v.End == replay.VehicleEndFilmEnd &&
		len(v.Rides) == 0
}

// posedOnlyVehicles rend les candidates du document — la zone jouable n est chargee que s il y en
// a, et son masque que si l une d elles tombe dans le cadre du fond.
func posedOnlyVehicles(doc *replay.ReplayDocument) []replay.VehicleTrack {
	if doc == nil {
		return nil
	}
	var out []replay.VehicleTrack
	for _, v := range doc.Vehicles {
		if vehicleIsPosedOnly(v) {
			out = append(out, v)
		}
	}
	return out
}

// decideVehicleScenery rend le verdict de decor du document. `zone` nil = carte sans zone connue :
// rien n est masque. Rend nil quand aucune vie n est candidate.
func decideVehicleScenery(doc *replay.ReplayDocument, zone playArea) *replay.VehicleScenery {
	candidates := posedOnlyVehicles(doc)
	if len(candidates) == 0 {
		return nil
	}
	floor, floorKnown := playedFloor(doc)
	out := &replay.VehicleScenery{Zone: sceneryZoneMap, Floor: sceneryFloorPlayed}
	if zone == nil {
		out.Zone = sceneryZoneUnknown
	}
	if !floorKnown {
		out.Floor = sceneryFloorUnknown
	}
	for _, v := range candidates {
		out.Candidates++
		if zone == nil {
			// repli_decor_carte_sans_zone_affiche : aucune zone, rien de masque, compte.
			out.ZoneUnknown++
			continue
		}
		reason := sceneryReasonOf(v.Samples[0], zone, floor, floorKnown)
		if reason == "" {
			out.InPlayArea++
			continue
		}
		out.Hidden = append(out.Hidden, replay.VehicleSceneryLife{Slot: v.Slot, Gen: v.Gen, Reason: reason})
	}
	return out
}

// sceneryReasonOf rend la raison pour laquelle une pose est HORS de la zone jouable, ou "" quand
// elle est dedans.
func sceneryReasonOf(s replay.VehicleSample, zone playArea, floor float32, floorKnown bool) string {
	switch {
	case !zone.Praticable(float64(s.X), float64(s.Y)):
		return sceneryReasonOffPlayArea
	case floorKnown && s.Z < floor-sceneryFloorToleranceM:
		return sceneryReasonBelowPlayedFloor
	}
	return ""
}

// playedFloor rend le SOL FOULE du match : la plus basse altitude ou une piste de joueur est restee
// au moins [sceneryStandMinMs] dans une bande de [sceneryStandMaxDZ]. Faux quand personne ne s est
// tenu nulle part, ou que le document n a pas d echelle de temps.
func playedFloor(doc *replay.ReplayDocument) (float32, bool) {
	if doc.FrameIntervalMS <= 0 {
		return 0, false
	}
	minFrames := int(math.Ceil(sceneryStandMinMs / float64(doc.FrameIntervalMS)))
	var floor float32
	found := false
	for _, tr := range doc.Tracks {
		if z, ok := lowestStand(tr.Points, minFrames); ok && (!found || z < floor) {
			floor, found = z, true
		}
	}
	return floor, found
}

// lowestStand rend la plus basse altitude d une STATION de la piste : une suite de points consecutifs
// couvrant au moins `minFrames` images, dont l altitude tient dans [sceneryStandMaxDZ]. Fenetre
// glissante a deux files monotones (minimum et maximum de Z) : lineaire en nombre de points.
func lowestStand(pts []replay.Point, minFrames int) (float32, bool) {
	var lo, hi []int
	var best float32
	found, i := false, 0
	for j := range pts {
		for len(lo) > 0 && pts[lo[len(lo)-1]].Z >= pts[j].Z {
			lo = lo[:len(lo)-1]
		}
		lo = append(lo, j)
		for len(hi) > 0 && pts[hi[len(hi)-1]].Z <= pts[j].Z {
			hi = hi[:len(hi)-1]
		}
		hi = append(hi, j)
		for pts[hi[0]].Z-pts[lo[0]].Z > sceneryStandMaxDZ {
			i++
			if lo[0] < i {
				lo = lo[1:]
			}
			if hi[0] < i {
				hi = hi[1:]
			}
		}
		if pts[j].T-pts[i].T >= minFrames && (!found || pts[lo[0]].Z < best) {
			best, found = pts[lo[0]].Z, true
		}
	}
	return best, found
}
