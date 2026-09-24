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
// # LA ZONE JOUABLE, ETABLIE SUR PIECES (parc du 2026-09-24, 107 documents au schema 69)
//
//   - EN PLAN : la MATIERE PRATICABLE de la carte, c est-a-dire le masque du fond de carte publie
//     (alpha > 0) ferme au rayon canonique du depot (`mapdecoupe.ToleranceParDefaut`, 4 m) — une
//     reference VERSIONNEE, disponible en production sans le jeu, independante du match. Mesure :
//     12/12 decors de Starboard hors du masque (ils sont hors du cadre du fond), 260/260 vies de
//     vehicule en jeu dessus. Les autres references du depot ne conviennent pas : `map_geometry/`
//     ne porte les props que d une carte, `map_positions_jouees.json` qu une carte (Dredge), les
//     paves de callout debordent sur le vide et n existent pas pour Goliath. Les bornes jouees du
//     match non plus : un Mongoose pose de Behemoth y est a 2,4 m du bord (f2966f08).
//   - EN HAUTEUR : AUCUNE reference ne publie l altitude praticable d une cellule (le sidecar n en
//     porte qu une par carte, `playLevelZ` ; constat ecrit dans `mapdecoupe/masque.go`). Le Wasp de
//     Goliath est DANS le masque, 3,03 m sous le sol. Le repli [sceneryReasonBelowPlayedFloor] lit
//     donc le sol JOUE du match : plus de [sceneryFloorToleranceM] sous la plus basse position de
//     joueur publiee (`Bounds.MinZ`, garde des aberrations du lot M1 comprise). Critere de retrait :
//     la zone jouable publiee porte l altitude de sa matiere, et le test de hauteur la lit au lieu
//     du match.
//
// Une carte SANS zone connue (aucun fond publie, fond illisible) ne masque RIEN : repli nomme
// [sceneryZoneUnknown], compte. Un document sans position de joueur ne sait pas son sol : le test
// de hauteur ne s applique pas ([sceneryFloorUnknown]).

import "levelup/go-api/internal/games/halo_infinite/film/replay"

// playArea est la zone jouable EN PLAN d une carte (le masque ferme du fond publie).
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

// sceneryFloorToleranceM : de combien de metres une pose doit passer SOUS le sol joue pour etre
// « sous le sol ». CONSTANTE MESUREE, donc repli nomme (raison [sceneryReasonBelowPlayedFloor],
// comptee dans `hidden`). Parc du 2026-09-24 (107 documents, 260 vies de vehicule en jeu ramenees a
// leur premier echantillon) : la plus basse est 0,10 m SOUS le sol joue (Wasp de 8a485699 sur son
// socle, Launch Site : le socle est plus bas que tout pas de joueur du match) ; le seul decor sous
// le sol est 3,03 m dessous (Wasp de Goliath). 1,5 m coupe l ecart en deux. Critere de retrait :
// celui du repli (la zone publiee porte l altitude de sa matiere).
const sceneryFloorToleranceM float32 = 1.5

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

// hasPosedOnlyVehicle dit si le document porte au moins une candidate — la zone jouable n est
// chargee que dans ce cas.
func hasPosedOnlyVehicle(doc *replay.ReplayDocument) bool {
	if doc == nil {
		return false
	}
	for _, v := range doc.Vehicles {
		if vehicleIsPosedOnly(v) {
			return true
		}
	}
	return false
}

// decideVehicleScenery rend le verdict de decor du document. `zone` nil = carte sans zone connue :
// rien n est masque. Rend nil quand aucune vie n est candidate.
func decideVehicleScenery(doc *replay.ReplayDocument, zone playArea) *replay.VehicleScenery {
	if !hasPosedOnlyVehicle(doc) {
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
	for _, v := range doc.Vehicles {
		if !vehicleIsPosedOnly(v) {
			continue
		}
		out.Candidates++
		if zone == nil {
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

// playedFloor rend le sol JOUE du match : la plus basse position de joueur publiee, telle que les
// bornes du document la gardent (aberrations ecartees). Faux quand aucune piste n a de point.
func playedFloor(doc *replay.ReplayDocument) (float32, bool) {
	for _, tr := range doc.Tracks {
		if len(tr.Points) > 0 {
			return doc.Bounds.MinZ, true
		}
	}
	return 0, false
}
