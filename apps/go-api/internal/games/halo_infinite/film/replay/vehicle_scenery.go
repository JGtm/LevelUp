package replay

// vehicle_scenery.go — LE VERDICT DE DECOR DE CARTE, tel que le document le publie (retours du
// rejeu, lot M7, 2026-09-24).
//
// A LA REQUETE, JAMAIS DANS L ARTEFACT, et c est la regle de `vehicleWeapons` : la zone jouable
// d une carte est une REFERENCE du titre (le fond publie et son calage), que l artefact — decode
// des seuls chunks d un film qui ne nomme pas sa carte — ne connait pas. Le service decide et pose
// ce verdict (`internal/service/replay_vehicle_scenery*.go`, ou vivent la regle et sa preuve) ; ce
// fichier n en porte que la FORME.

// VehicleScenery est le verdict de decor du document. ABSENT quand aucune vie ne remplit les
// conditions de pose : il n y a alors rien a decider.
type VehicleScenery struct {
	// Zone : `map` (la zone jouable de la carte a ete lue) ou `unknown` (aucune zone connue — rien
	// n est masque, repli nomme et compte dans `ZoneUnknown`).
	Zone string `json:"zone"`
	// Floor : `played` (le sol FOULE du match est connu : la plus basse altitude ou un joueur est
	// reste au moins 1 s) ou `unknown` (personne ne s est tenu nulle part — le test de hauteur ne
	// s applique pas).
	Floor string `json:"floor"`
	// Candidates : vies qui remplissent les cinq conditions de pose.
	Candidates int `json:"candidates"`
	// InPlayArea : candidates posees DANS la zone jouable — affichees.
	InPlayArea int `json:"inPlayArea"`
	// ZoneUnknown : candidates affichees faute de zone connue.
	ZoneUnknown int `json:"zoneUnknown"`
	// Hidden : les vies de decor, avec la raison. Le client ne les dessine pas.
	Hidden []VehicleSceneryLife `json:"hidden,omitempty"`
}

// VehicleSceneryLife designe une vie de decor et dit pourquoi elle l est.
type VehicleSceneryLife struct {
	Slot uint32 `json:"slot"`
	Gen  uint32 `json:"gen"`
	// Reason : `off_play_area` (hors de la matiere praticable de la carte) ou `below_played_floor`
	// (sous le sol foule du match, repli nomme au registre facts/fallback).
	Reason string `json:"reason"`
}
