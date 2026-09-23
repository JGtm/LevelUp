package replay

// vehicle_weapons.go — LE REGISTRE DES ARMES DE VEHICULE, tel que le document le publie (schema 69,
// retours du rejeu du 2026-09-23, lot M4a).
//
// D OU IL VIENT. Trois tables CLIENT (style, son, montage : `vehicleShotFx.ts`,
// `vehicleShotSound.ts`, `vehicleWeaponMounts.ts`) etaient clees par des tags `weap` lus dans le
// module du jeu (lot V3F), jamais confrontes a un film : 7 des 11 cles n apparaissaient dans aucun
// document, 3 tags publies en etaient absents. Le registre du TITRE les remplace
// (`config/titles/{slug}/mappings/vehicle_weapons.toml`) : une entree par tag OBSERVE dans un
// film, avec sa PREUVE, et un garde-rail Go qui tient la clef sur une fixture datee du parc.
//
// A LA REQUETE, JAMAIS DANS L ARTEFACT, et c est la regle de `vehicleLabels` : la table est une
// resolution du TITRE qui s ameliore (une teinte tranchee, un son reconstruit), et la cuire
// laisserait les artefacts deja publies sur l ancienne valeur jusqu a une recuisson complete. Le
// service la pose (`internal/service/replay_vehicle_weapons.go`) pour les seules armes que les
// tirs du document emploient.

// VehicleWeapon est ce que le registre dit d UNE arme de vehicule.
type VehicleWeapon struct {
	// Vehicle est la famille (ou la variante) du vehicule qui porte l arme — `wraith`,
	// `rockethog`, `gungoose`... : le meme vocabulaire que `VehicleTrack.Family` / `Variant`.
	Vehicle string `json:"vehicle"`
	// En / Fr : le nom de l arme, dans les deux langues du produit (jamais ecrit en Go).
	En string `json:"en"`
	Fr string `json:"fr"`
	// Fire : `single` (un evenement par coup) ou `continuous` (tir tenu). Liste fermee.
	Fire string `json:"fire"`
	// Fx / Tint : la FORME de l eclair et la NATURE de la decharge, dans les memes listes fermees
	// que `WeaponLabel.Fx` / `.Tint` (`[shot_effects]` / `[shot_tints]` du titre).
	Fx   string `json:"fx"`
	Tint string `json:"tint"`
	// Sound est le stem de la PREMIERE variante du son de tir. VIDE = SILENCE DECIDE : le registre
	// porte la raison (`silence`), le document ne publie que le silence.
	Sound string `json:"sound,omitempty"`
	// Mount est l ANCRE de l arme sur le sprite de son vehicule. Absent = arme sans montage
	// documente : l eclair part du centre du vehicule.
	Mount *VehicleWeaponMount `json:"mount,omitempty"`
}

// VehicleWeaponMount est l ancre d une arme en FRACTIONS DU SPRITE (repere nez en haut : `ax` de
// -0,5 a +0,5 de gauche a droite, `ay` de -0,5 au nez a +0,5 a l arriere) et sa classe de visee.
type VehicleWeaponMount struct {
	// Aim : `fixed` (solidaire du nez, vise ou pointe le vehicule) ou `turret` (visee
	// independante, celle du tireur).
	Aim string  `json:"aim"`
	AX  float64 `json:"ax"`
	AY  float64 `json:"ay"`
}

// VehicleWeaponKey rend la cle de `Shot.Weapon` d une arme de vehicule : le tag `weap` de 32 bits
// dans la moitie HAUTE, la moitie basse nulle. C EST LE SEUL ENDROIT QUI ECRIT CE GABARIT : le
// client n a plus a le connaitre, il lit la table par la cle meme du tir.
func VehicleWeaponKey(tag uint32) string {
	return formatWeaponID(uint64(tag) << 32)
}
