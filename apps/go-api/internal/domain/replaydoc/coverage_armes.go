package replaydoc

// coverage_armes.go — CE QUE LE FILM DIT DES ARMES A L INSTANT, ET SUR QUEL DENOMINATEUR (schema
// 69, campagne « retours rejeu », lot M3). Jumeau servi des types de
// `film/replay/coverage_keyframes.go` et `film/replay/document_birth_loadouts.go` : memes champs,
// meme ordre, memes etiquettes JSON (`TestDocumentShapeTwinsAgree`).

// KeyframeCoverage dit, pour le film entier, comment le balayeur d'image-clé a atteint chaque
// record, et combien de bipèdes il a manqués alors que les images-clés voisines les portaient.
type KeyframeCoverage struct {
	Keyframes          int `json:"keyframes"`
	Records            int `json:"records"`
	Bipeds             int `json:"bipeds"`
	Neighbors          int `json:"neighbors"`
	Jumps              int `json:"jumps"`
	Resyncs            int `json:"resyncs"`
	Elections          int `json:"elections"`
	Refutations        int `json:"refutations"`
	Slides             int `json:"slides"`
	FramedAbsentBipeds int `json:"framedAbsentBipeds"`
}

// BirthLoadoutCoverage dit ce que la lecture des dotations de naissance a vu, refusé et publié.
type BirthLoadoutCoverage struct {
	Creations         int `json:"creations"`
	Closed            int `json:"closed"`
	Read              int `json:"read"`
	Desync            int `json:"desync"`
	Overflow          int `json:"overflow"`
	Unconfirmed       int `json:"unconfirmed"`
	NoWeaponComponent int `json:"noWeaponComponent"`
	Published         int `json:"published"`
	Snapped           int `json:"snapped"`
	NoLife            int `json:"noLife"`
	BeforeOrigin      int `json:"beforeOrigin"`
	NonWeapon         int `json:"nonWeapon"`
	UnarmedGrants     int `json:"unarmedGrants"`
	NoDisplayable     int `json:"noDisplayable"`
}
