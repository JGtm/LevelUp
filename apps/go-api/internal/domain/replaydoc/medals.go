package replaydoc

import "levelup/go-api/internal/i18n"

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

// MedalLabel est ce qu'il faut pour NOMMER et DESSINER une medaille du document, comme sur les
// tuiles de match : son libelle par langue, sa description, son image (ou sa decoupe de feuille de
// sprites) et sa rarete. La table `medalLabels` du document est cle par `Medal.Name`.
//
// RESOLU A LA REQUETE, jamais ecrit dans l'artefact : le referentiel du titre peut s'ameliorer, et
// les artefacts sont cuits hors ligne. Une medaille que le referentiel ne connait pas n'a AUCUNE
// entree : le lecteur garde le nom du film, il n'invente ni libelle ni visuel.
type MedalLabel struct {
	// Text est le libelle de la medaille, une valeur par langue de `config/locales.toml`.
	Text i18n.Text `json:"i18n_text,omitempty"`
	// Description : ce que la medaille recompense, absent quand le referentiel n'en porte aucune.
	Description *MedalDescription `json:"description,omitempty"`
	// Img est l'URL de l'icone ; vide quand le titre sert une decoupe de feuille (`Sprite`).
	Img string `json:"img,omitempty"`
	// Sprite : la decoupe de feuille de sprites du titre qui en sert (l'inverse de `Img`).
	Sprite *MedalSprite `json:"sprite,omitempty"`
	// Difficulty est la rarete du referentiel (`Normal`, `Heroic`, `Legendary`, `Mythic`) ; vide
	// quand le referentiel n'en dit rien.
	Difficulty string `json:"difficulty,omitempty"`
}

// MedalDescription est la description d'une medaille, une valeur par langue.
type MedalDescription struct {
	Text i18n.Text `json:"i18n_text,omitempty"`
}

// MedalSprite est la decoupe d'une feuille de sprites : l'URL de la feuille, le decalage et la taille
// native de la cellule, en pixels.
type MedalSprite struct {
	Sheet  string `json:"sheet"`
	Left   int    `json:"left"`
	Top    int    `json:"top"`
	Width  int    `json:"width"`
	Height int    `json:"height"`
}
