// Package domain — squad_weapon_tools.go : « Outils de destruction » de l'Escouade
// (décision D8 du plan PLAN_EMPRISE_ET_CARTES_DEPLACEES_2026-09-26) — chaque frag nommé.
package domain

// Nature d'une ligne des « Outils de destruction » de l'Escouade (décision D8 du plan
// PLAN_EMPRISE_ET_CARTES_DEPLACEES_2026-09-26). Des CLÉS, jamais des libellés : le web
// nomme chaque nature dans sa langue ; seule la nature `weapon` porte un nom, celui du
// registre d'armes du titre (weapon_names.toml).
const (
	// SquadToolKindWeapon : une clé d'arme (film pour un titre qui décode son film, table
	// native sinon), nommée par le registre.
	SquadToolKindWeapon = "weapon"
	// SquadToolKindMelee : la mêlée, lue sur la feuille de match (le film n'a pas de clé
	// de mêlée). Sur un titre aux mécaniques natives, hors assassinats.
	SquadToolKindMelee = "melee"
	// SquadToolKindGrenade : les grenades de la feuille de match, en une ligne, pour un
	// joueur dont le film ne donne AUCUN détail typé (titre sans film, film sans grenade).
	// Quand le film les type, les lignes par type la remplacent.
	SquadToolKindGrenade = "grenade"
	// SquadToolKindAssassination / GroundPound / ShoulderBash : mécaniques natives de la
	// feuille de match, servies seulement si le titre les déclare (capability).
	SquadToolKindAssassination = "assassination"
	SquadToolKindGroundPound   = "ground_pound"
	SquadToolKindShoulderBash  = "shoulder_bash"
	// SquadToolKindExplosiveObject : objet de décor qui blesse en explosant (bidon),
	// d'après la catégorie de la source de dégât du film.
	SquadToolKindExplosiveObject = "explosive_object"
	// SquadToolKindEnvironment : chute et environnement, d'après la catégorie de la
	// source de dégât du film.
	SquadToolKindEnvironment = "environment"
	// SquadToolKindUnattributed : le reliquat feuille de match − lignes ci-dessus, s'il
	// est positif. Toujours la dernière ligne.
	SquadToolKindUnattributed = "unattributed"
)

// Catégories canoniques d'une SOURCE DE DÉGÂT du film (port.KillSourceCategorizer). Le
// titre traduit ses propres classes de source vers ces clés ; aucun code partagé ne
// connaît les noms de classe d'un titre.
const (
	KillSourceCategoryExplosiveObject = SquadToolKindExplosiveObject
	KillSourceCategoryEnvironment     = SquadToolKindEnvironment
)

// SquadWeaponToolLine est une ligne des « Outils de destruction » : un outil avec les
// frags de chaque joueur de l'escouade et leur total.
type SquadWeaponToolLine struct {
	// Kind : nature de la ligne (SquadToolKind*).
	Kind string `json:"kind"`
	// WeaponKey : clé de registre (nature `weapon` seule).
	WeaponKey string `json:"weapon_key,omitempty"`
	// Label / LabelEN : nom de l'arme au registre, FR d'abord / EN d'abord (nature
	// `weapon` seule ; les autres natures sont nommées par le web).
	Label   string `json:"label,omitempty"`
	LabelEN string `json:"label_en,omitempty"`
	// Class : classe de frag (FragClass*) dont la ligne prend la couleur — la pastille
	// devant le nom relie la ligne à la « Répartition des frags ».
	Class         string         `json:"class"`
	KillsByPlayer map[string]int `json:"kills_by_player"` // gamertag → frags
	TotalSquad    int            `json:"total_squad"`
}

// SquadWeaponTools alimente « Outils de destruction » (Escouade, D8) : chaque frag
// nommé, sans plafond ni regroupement « Autres ». Players est l'ordre canonique (main puis
// coéquipiers) ; Lines est trié par total décroissant, « Non attribué » en dernier.
type SquadWeaponTools struct {
	Players []string              `json:"players"`
	Lines   []SquadWeaponToolLine `json:"lines"`
}
