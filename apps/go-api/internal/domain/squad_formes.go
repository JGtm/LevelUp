package domain

// squad_formes.go — LE BLOC « FORMES RETENUES » de la page Escouade (artefact
// 2ec1b8eb, lot D2 du 2026-09-13).
//
// # CE QUE CE BLOC PUBLIE, ET POURQUOI IL EST « PLAT »
//
// Les dix-neuf cartes de l'artefact posent dix-neuf questions sur LA MÊME
// matière : ce que chaque joueur des DEUX camps a fait de son équipement, des
// socles d'arme et de l'objectif, match par match. Publier dix-neuf agrégats
// aurait figé dix-neuf fois la même jointure et rendu impossible la moindre
// variation de dénominateur (l'artefact en emploie quatre : mon équipe, le
// lobby, l'escouade seule, les occupations de socle). Le bloc publie donc LA
// MATIÈRE — une ligne par joueur et par match — et les parts se calculent là où
// elles s'affichent, dans des modèles purs testés côté web.
//
// # CE QU'IL NE PORTE PAS
//
//   - Aucun libellé FR/EN : les noms d'arme viennent du catalogue du titre
//     (résolus au service, cf. squadagg), les clés de colonne d'objectif se
//     traduisent côté web. Aucune chaîne de langue ne descend d'ici.
//   - Aucune cadence normalisée par la durée : décision utilisateur du
//     2026-09-13, « cadence c'est par match, pas par minutes ». La durée du
//     match reste publiée (elle sert à dire un match, pas à diviser une
//     grandeur).
//   - Les grenades (ce ne sont pas des équipements, décision D5) et le
//     répulseur (aucun canal ne mesure son usage — une ligne dirait « 0 »
//     là où la vérité est « non mesuré »).
//
// # NON MESURÉ N'EST JAMAIS ZÉRO
//
// Un match sans film décodé porte Measured=false et AUCUNE ligne de lobby : les
// formes le rendent en hachure, jamais en zéro. De même un match sans objectif
// n'a pas de bloc Objective — les six matchs Assassin d'une soirée ne sont pas
// un trou de mesure.

// SquadFormesBlock — la matière des dix-neuf cartes, sur le scope filtré de la
// page Escouade (même population que le bloc equipment_usage).
type SquadFormesBlock struct {
	// Available : le titre porte la capability de mesure et les lectures ont
	// abouti. Faux ⇒ UnavailableReason dit laquelle des deux a manqué
	// (constantes SessionUsage* — le même vocabulaire que le bloc d'usage).
	Available         bool   `json:"available"`
	UnavailableReason string `json:"unavailable_reason,omitempty"`
	// MatchesTotal : les matchs du scope. MatchesMeasured : ceux qui portent un
	// film décodé. Le couple s'affiche tel quel (« 8 sur 9 »).
	MatchesTotal    int `json:"matches_total"`
	MatchesMeasured int `json:"matches_measured"`
	// MainXUID : le joueur de la page. Squad : lui EN TÊTE, puis les
	// coéquipiers sélectionnés, dans l'ordre d'affichage.
	MainXUID string                    `json:"main_xuid,omitempty"`
	Squad    []SessionUsageSquadPlayer `json:"squad,omitempty"`
	// Matches : le scope dans l'ordre chronologique de la page.
	Matches []SquadFormesMatch `json:"matches,omitempty"`
	// Weapons : les armes de socle rencontrées sur le scope, nommées et rangées.
	Weapons []SquadFormesWeapon `json:"weapons,omitempty"`
}

// SquadFormesMatch — un match du scope : son identité d'affichage, son camp, et
// ce que le film en dit.
type SquadFormesMatch struct {
	MatchID string `json:"match_id"`
	// StartTime : ISO 8601 UTC (l'heure affichée est locale, côté web).
	StartTime string `json:"start_time,omitempty"`
	// ModeLabel / MapLabel : libellés déjà résolus par les adapters du titre
	// (jamais une chaîne écrite en Go). Vides quand l'historique de la page ne
	// porte pas ce match.
	ModeLabel string `json:"mode_label,omitempty"`
	MapLabel  string `json:"map_label,omitempty"`
	// DurationSeconds : durée mesurée du film. 0 = inconnue.
	DurationSeconds float64 `json:"duration_seconds,omitempty"`
	// Measured : le match porte un film décodé. Faux ⇒ Lobby vide.
	Measured bool `json:"measured"`
	// PlayerTeam : camp du joueur de la page (nil = inconnu, FFA). TeamSize /
	// LobbySize : effectifs présents à la fin — les deux parités du match.
	PlayerTeam *int `json:"player_team,omitempty"`
	TeamSize   int  `json:"team_size,omitempty"`
	LobbySize  int  `json:"lobby_size,omitempty"`
	// Lobby : une ligne par joueur MESURÉ du match, les deux camps.
	Lobby []SquadFormesLobbyPlayer `json:"lobby,omitempty"`
	// PadNamed / PadUnnamed : prises de socle d'ARME avec et sans ramasseur
	// nommé. Les secondes n'entrent dans aucun camp — c'est la réserve que la
	// carte « Les deux frises » affiche en toutes lettres.
	PadNamed   int `json:"pad_named,omitempty"`
	PadUnnamed int `json:"pad_unnamed,omitempty"`
	// WeaponPads : un élément par SOCLE du match (deux socles de la même arme
	// restent deux entrées — l'agrégation appartient au lecteur).
	WeaponPads []SquadFormesWeaponPad `json:"weapon_pads,omitempty"`
	// Objective : absent quand le mode n'a pas d'objectif.
	Objective *SquadFormesObjective `json:"objective,omitempty"`
}

// SquadFormesLobbyPlayer — ce qu'un joueur du lobby a fait dans ce match. Les
// cinq familles de geste sont celles que la session mesure ; les grenades en
// sont exclues (ce ne sont pas des équipements).
type SquadFormesLobbyPlayer struct {
	XUID     string `json:"xuid"`
	Gamertag string `json:"gamertag,omitempty"`
	TeamID   *int   `json:"team_id,omitempty"`
	// Camo / Overshield : ÉPISODES actifs. Wall : murs de protection posés.
	// Grapple : tractions de grappin. Dropped : objets lâchés au sol.
	Camo       int `json:"camo"`
	Overshield int `json:"overshield"`
	Wall       int `json:"wall"`
	Grapple    int `json:"grapple"`
	Dropped    int `json:"dropped"`
	// PadPickups : prises de socle d'arme nommées à ce joueur.
	PadPickups int `json:"pad_pickups"`
	// PadsByWeapon : la même grandeur ventilée par clé de famille d'arme (la
	// clé de SquadFormesWeapon).
	PadsByWeapon map[string]int `json:"pads_by_weapon,omitempty"`
}

// SquadFormesWeaponPad — un socle d'arme du match : son arme, ses occupations
// achevées, et celles dont le ramasseur est nommé.
type SquadFormesWeaponPad struct {
	Weapon      string `json:"weapon"`
	Occupations int    `json:"occupations"`
	Named       int    `json:"named"`
}

// Les trois familles d'arme de socle du bloc 2. Elles se DÉRIVENT du registre
// canonique d'armes (class = manipulation, role = fonction), jamais d'une table
// écrite pour l'occasion.
const (
	// SquadFormesWeaponHeavy : classe `heavy` du registre (lance-roquettes,
	// empaleur, épée, marteau, fusil de précision...).
	SquadFormesWeaponHeavy = "heavy"
	// SquadFormesWeaponPrecision : rôle `precision` ou `sniper` hors classe
	// lourde (carabines, pistolets de précision...).
	SquadFormesWeaponPrecision = "precision"
	// SquadFormesWeaponOther : tout le reste des socles.
	SquadFormesWeaponOther = "other"
)

// SquadFormesWeapon — une arme de socle rencontrée sur le scope.
type SquadFormesWeapon struct {
	// Key : la clé de famille du film (huit hexadécimaux minuscules), celle de
	// PadsByWeapon et de WeaponPad.Weapon.
	Key string `json:"key"`
	// Label : le nom du catalogue du titre, dans la langue de la requête. VIDE
	// quand le catalogue ne connaît pas la famille — le web affiche alors la
	// réserve « arme non cataloguée », jamais un nom approchant.
	Label string `json:"label,omitempty"`
	// WeaponKey : la clé canonique du registre (`hinf_m41_spnkr`). Vide =
	// famille hors registre.
	WeaponKey string `json:"weapon_key,omitempty"`
	// Class : l'une des trois constantes SquadFormesWeapon*.
	Class string `json:"class"`
}

// SquadFormesObjective — l'objectif d'un match : sa famille, les colonnes que ce
// mode publie VRAIMENT sur ce scope, et les valeurs de chaque joueur des deux
// camps.
type SquadFormesObjective struct {
	// Family : clé stable de famille de mode (`ctf`, `zones_koth`, ...).
	Family  string                       `json:"family"`
	Columns []SquadFormesObjectiveColumn `json:"columns,omitempty"`
	Players []SquadFormesObjectivePlayer `json:"players,omitempty"`
}

// SquadFormesObjectiveColumn — une grandeur du mode et son rôle.
type SquadFormesObjectiveColumn struct {
	// Key : nom de colonne de match_objective_stats (`flag_returns`). Le web le
	// traduit ; aucun libellé ne descend d'ici.
	Key string `json:"key"`
	// Role : `take`, `defend` ou `hold` (narrative.ObjectiveRole).
	Role string `json:"role"`
	// Duration : la colonne porte des SECONDES (elle s'affiche en mm:ss et ne
	// se compare jamais à un compte d'actions).
	Duration bool `json:"duration,omitempty"`
}

// SquadFormesObjectivePlayer — les valeurs d'un joueur sur les colonnes du match.
type SquadFormesObjectivePlayer struct {
	XUID   string             `json:"xuid"`
	TeamID *int               `json:"team_id,omitempty"`
	Values map[string]float64 `json:"values,omitempty"`
}
