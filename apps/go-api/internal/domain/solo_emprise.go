package domain

// solo_emprise.go — L'EMPRISE DU PÉRIMÈTRE SOLO, onglet « Usages » des Séries temporelles (plan
// `.ai/PLAN_TIMESERIES_USAGES_EMPRISE_2026-10-05.md`, décisions D2-D4).
//
// Le bloc EMBARQUE SquadEmpriseBlock : mêmes ressources, mêmes objets, même grille match par
// match, mêmes règles de mesure que l'onglet Emprise de l'Escouade — calculés par le même
// `analysis/squademprise` sur les matchs solo de la page, avec un seul joueur des fiches (le joueur
// de la page) et sans habitude. Il y ajoute ce que seule la page solo affiche : la grille par
// CARTE (`Maps`) et la carte « Équipement » (`Equipment`).

// EmpriseGridMaxMaps — les cartes nommées de la grille ; au-delà de EmpriseGridMaxMaps + 1 cartes,
// les suivantes sont sommées dans une colonne « Autres cartes ».
const EmpriseGridMaxMaps = 12

// SoloEmpriseBlock — le bloc publié sous `emprise` sur /pages/timeseries. Nil quand le périmètre
// n'a aucun match.
type SoloEmpriseBlock struct {
	SquadEmpriseBlock
	// Maps : une colonne par carte jouée, la plus jouée d'abord (libellé en départage), puis au
	// besoin la colonne de repli (OtherMaps > 0).
	Maps []EmpriseMapColumn `json:"maps"`
	// Equipment : ce que sont devenus les équipements tenus, moi et le reste de mon camp. Nil sans
	// film (titre sans résumé d'usage ou lecture en échec).
	Equipment *EmpriseEquipment `json:"equipment,omitempty"`
}

// EmpriseMapColumn — une carte (ou la colonne de repli) : ses matchs, leurs états de mesure, leurs
// résultats, et les prises de chaque ressource sommées sur ses matchs lisibles.
type EmpriseMapColumn struct {
	// MapKey / MapLabel : l'identifiant de la carte et son nom dans la langue de la requête ; vides
	// pour la colonne de repli.
	MapKey   string `json:"map_key,omitempty"`
	MapLabel string `json:"map_label,omitempty"`
	// OtherMaps : > 0 pour la colonne de repli — le nombre de cartes qu'elle somme.
	OtherMaps int `json:"other_maps,omitempty"`
	// Matches : les matchs du périmètre sur cette carte. MatchesFilmed : ceux dont le film est
	// résumé ; MatchesMeasured : filmés ET camp connu (les seuls qui comptent des prises) ;
	// MatchesTiers : mesurés ET niveaux de socle établis (armes spéciales, râteliers) ;
	// VehiclesMeasured : ceux dont la ressource véhicules est mesurée.
	Matches          int `json:"matches"`
	MatchesFilmed    int `json:"matches_filmed"`
	MatchesMeasured  int `json:"matches_measured"`
	MatchesTiers     int `json:"matches_tiers"`
	VehiclesMeasured int `json:"vehicles_measured"`
	// Wins / Losses / Others : résultats du joueur sur ces matchs (Others : égalité, abandon,
	// inconnu).
	Wins   int `json:"wins"`
	Losses int `json:"losses"`
	Others int `json:"others"`
	// Resources : ressource puis chaque objet, comme un match (SquadEmpriseMatchResource), sommés
	// sur les matchs de la carte où la ressource se lit.
	Resources []SquadEmpriseMatchResource `json:"resources"`
	// PowerWeaponKills : frags aux armes spéciales de chaque camp (feuille de match), sur les matchs
	// qui les portent. Nil quand aucun ne les porte.
	PowerWeaponKills *SquadEmpriseCount `json:"power_weapon_kills,omitempty"`
}

// EmpriseEquipment — la carte « Équipement pris, et ce que j'en ai fait ».
type EmpriseEquipment struct {
	// MatchesMeasured : les matchs dont les comptes sont lus (filmés à camp connu) — le même
	// périmètre pour moi et pour le reste de mon camp.
	MatchesMeasured int `json:"matches_measured"`
	// Families : grappin (non mesuré), les familles du bilan hors bonus, propulseur (non mesuré).
	Families []EmpriseEquipmentFamily `json:"families"`
}

// EmpriseEquipmentFamily — une famille d'équipement.
type EmpriseEquipmentFamily struct {
	// Family : la clé de famille du résumé (« wall », « sensor », « grapple »…), nommée côté web.
	Family string `json:"family"`
	// Measured : la famille porte une ligne d'issue (servi / gardé / lâché). Faux : seuls mes
	// lâchers sont connus (DroppedMe), ni prise ni usage ne sont publiés.
	Measured bool `json:"measured"`
	// Me / Rest : mes comptes et ceux du reste de mon camp (mesurées seulement).
	Me   *EmpriseEquipmentOutcomes `json:"me,omitempty"`
	Rest *EmpriseEquipmentOutcomes `json:"rest,omitempty"`
	// Lobby : les comptes de TOUS les joueurs des matchs mesurés — mon camp, l'adversaire et les
	// joueurs sans camp connu (mesurées seulement). Dit si la famille a été tenue par quelqu'un.
	Lobby *EmpriseEquipmentOutcomes `json:"lobby,omitempty"`
	// DroppedMe : mes lâchers (non mesurées seulement).
	DroppedMe int `json:"dropped_me,omitempty"`
	// DroppedLobby : les lâchers de tous les joueurs des matchs mesurés (non mesurées seulement).
	DroppedLobby int `json:"dropped_lobby,omitempty"`
}

// EmpriseEquipmentOutcomes — pris sur la carte, servi (posé pour le mur, charge consommée pour les
// autres), gardé sans servir, lâché ; servi + gardé + lâché = les objets tenus (équipement de
// réapparition compris).
type EmpriseEquipmentOutcomes struct {
	Taken   int `json:"taken"`
	Used    int `json:"used"`
	Kept    int `json:"kept"`
	Dropped int `json:"dropped"`
}
