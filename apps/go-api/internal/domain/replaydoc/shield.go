package replaydoc

// shield.go — LES BRIS ET LES RECHARGES DE BOUCLIER D'UN JOUEUR, DATES SUR L'AXE DU DOCUMENT.

// ShieldCycle donne, pour un joueur dont le film mesure le bouclier, l'instant de chacun de ses bris
// de bouclier et de ses recharges completes, en millisecondes sur l'axe du document (image fois
// `frameIntervalMs`, l'horloge de la tete de lecture, origine du document non retranchee), tries du
// plus ancien au plus recent.
//
// Les listes sont toujours presentes : un joueur mesure sans evenement a deux listes vides, et c'est
// une mesure (zero bris). Un joueur que le film ne mesure pas n'a AUCUNE entree dans
// `ReplayDocument.ShieldCycles`.
//
// CALCULE A LA REQUETE par `analysis/shieldcycle` (la seule definition des bris et des recharges), jamais
// ecrit dans l'artefact : le client compte les instants <= tete de lecture et la totalite pour le match.
type ShieldCycle struct {
	Breaks    []int `json:"breaks"`
	Recharges []int `json:"recharges"`
}
