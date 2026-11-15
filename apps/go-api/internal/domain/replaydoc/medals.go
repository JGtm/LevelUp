package replaydoc

// medals.go — LES MEDAILLES GAGNEES, DATEES SUR L'AXE DU REJEU, ET LA COUVERTURE DU CALQUE.

// Medal est une medaille gagnee par un joueur : la frame, l'instant sur l'horloge des temps
// forts (millisecondes), le joueur (xuid decimal), l'indice de medaille du jeu et son nom
// anglais (cle de referentiel ; absent quand l'indice n'est pas dans la table du decodeur).
type Medal struct {
	T    int    `json:"t"`
	MS   int64  `json:"ms"`
	XUID string `json:"xuid"`
	ID   int    `json:"id"`
	Name string `json:"name,omitempty"`
}

// MedalCoverage dit ce que le calque des medailles a pose et ce qu'il a ecarte.
type MedalCoverage struct {
	Decoded      int `json:"decoded"`
	Published    int `json:"published"`
	NoOrigin     int `json:"noOrigin"`
	BeforeOrigin int `json:"beforeOrigin"`
	AfterEnd     int `json:"afterEnd"`
	UnknownName  int `json:"unknownName"`
}
