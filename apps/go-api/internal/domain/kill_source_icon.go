package domain

// KillSourceIconEntry est une ligne du référentiel « arme -> pictogramme » d'un titre : la clé
// canonique du registre d'armes, son nom dans la langue de la requête, et le pictogramme du fil
// d'éliminations (le MÊME que celui d'une élimination par cette arme).
type KillSourceIconEntry struct {
	// Key est la weapon_key du registre d'armes du titre.
	Key string `json:"key"`
	// Label est le nom de l'arme dans la langue de la requête (repli : langue par défaut, puis nom
	// propre de la source de dégât quand le titre ne nomme pas l'arme).
	Label string `json:"label"`
	// ImageURL est l'URL relative du pictogramme ; jamais vide.
	ImageURL string `json:"image_url"`
	// Tinted : le pictogramme est un MASQUE à teindre, pas une image finie.
	Tinted bool `json:"tinted"`
}
