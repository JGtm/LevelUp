package replaydoc

// coverage_seats.go — la couverture des PLACES, sortie de coverage.go (lot M4b : place pour les
// compteurs du tir continu). Jumeau de `replay.SeatCoverage`.

// SeatCoverage est ce que la pose des PLACES a lu, deduit, borne et laisse sans place (lot
// 1.9.14 ; refondu au lot M2.3, schema 69, sur la regle des places de l utilisateur).
//
// SES CHIFFRES CENTRAUX SONT `depassements` et `placesEnTrop` (0 attendus : les couples frame x
// equipe ou une equipe affiche plus d occupants que sa CAPACITE estimee, et les places affichees
// au-dela de cette capacite — revue M2-R6, le plafond n est plus les places posees). `lus` /
// `placesTirs` disent la part LUE des places, `apparies` celle du chainage et `placesOuvertes` celle des places ouvertes sous la capacite
// estimee (deux replis), `sansPlace` ce qui n en a trouve aucune ; `presences`
// vaut `film` (entites ti=9, BOT_METADATA) ou `vies` (repli sans entite).
type SeatCoverage struct {
	Entrees         int `json:"entrees"`
	Sieges          int `json:"sieges"`
	Lus             int `json:"lus"`
	PlacesTirs      int `json:"placesTirs"`
	Apparies        int `json:"apparies"`
	PlacesOuvertes  int `json:"placesOuvertes"`
	SansPlace       int `json:"sansPlace"`
	ReprisesEcrites int `json:"reprisesEcrites"`
	Arrivants       int `json:"arrivants"`
	PresencesCloses int `json:"presencesCloses"`
	SansPresence    int `json:"sansPresence"`
	OccupantsMax    int `json:"occupantsMax"`
	Capacite        int `json:"capacite"`
	Depassements    int `json:"depassements"`
	PlacesEnTrop    int `json:"placesEnTrop"`
	SansEquipe      int `json:"sansEquipe"`
	// IdentitesHorsRoster / BotsSuccesseurs / PresencesParLesVies : revue du lot M2 (2026-09-24),
	// cf. `replay.SeatCoverage`.
	IdentitesHorsRoster int  `json:"identitesHorsRoster"`
	BotsSuccesseurs     int  `json:"botsSuccesseurs"`
	PresencesParLesVies int  `json:"presencesParLesVies"`
	RelaisBornes        int  `json:"relaisBornes"`
	Chevauchements      int  `json:"chevauchements"`
	TirsContestes       int  `json:"tirsContestes"`
	TirsIndexTronque    bool `json:"tirsIndexTronque,omitempty"`
	// TirsParPlace : les tirs rendus a l occupant de leur place (lot M4b.4, cf.
	// `replay.SeatCoverage`).
	TirsParPlace        int    `json:"tirsParPlace"`
	Presences           string `json:"presences"`
	EntitesNonLiees     int    `json:"entitesNonLiees"`
	EntitesContestees   int    `json:"entitesContestees"`
	TrousDEntite        int    `json:"trousDEntite"`
	ImagesClesDouteuses int    `json:"imagesClesDouteuses"`
	BornesDifferees     int    `json:"bornesDifferees"`
	SansTableDuFilm     bool   `json:"sansTableDuFilm,omitempty"`
}
