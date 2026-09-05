package replaydoc

// coverage.go — CE QUE CHAQUE CALQUE A LU, ET SUR QUEL DENOMINATEUR. Publier « N faits »
// sans dire combien existaient laisserait croire a l'exhaustivite ; ces compteurs sont la
// pour que le client — et un lecteur humain — puissent juger sans relire le film.

// Coverage porte la couverture de chaque calque du document, et le VERDICT qui en découle.
type Coverage struct {
	Shots             LayerCoverage               `json:"shots"`
	Grenades          LayerCoverage               `json:"grenades"`
	Objectives        LayerCoverage               `json:"objectives"`
	Equipment         *EquipmentCoverage          `json:"equipment,omitempty"`
	Grapple           *GrappleCoverage            `json:"grapple,omitempty"`
	Placements        *EquipmentPlacementCoverage `json:"placements,omitempty"`
	GroundWeapons     *GroundWeaponCoverage       `json:"groundWeapons,omitempty"`
	Score             *ScoreCoverage              `json:"score,omitempty"`
	FlagCarries       *FlagCarriesCoverage        `json:"flagCarries,omitempty"`
	VipCrown          *VipCrownCoverage           `json:"vipCrown,omitempty"`
	SkullCarries      *SkullCarriesCoverage       `json:"skullCarries,omitempty"`
	BombCarries       *BombCarriesCoverage        `json:"bombCarries,omitempty"`
	BombArmings       *BombArmingsCoverage        `json:"bombArmings,omitempty"`
	WeaponChanges     *WeaponChangeCoverage       `json:"weaponChanges,omitempty"`
	Pickups           *PickupCoverage             `json:"pickups,omitempty"`
	PadDating         *PadDatingStats             `json:"padDating,omitempty"`
	EquipmentChanges  *EquipmentChangeCoverage    `json:"equipmentChanges,omitempty"`
	Translocations    *TranslocationCoverage      `json:"translocations,omitempty"`
	AbilityImpulses   *AbilityImpulseCoverage     `json:"abilityImpulses,omitempty"`
	AbilityCharges    *AbilityChargeCoverage      `json:"abilityCharges,omitempty"`
	GroundWeaponItems *GroundWeaponItemsCoverage  `json:"groundWeaponItems,omitempty"`
	Vehicles          *VehicleCoverage            `json:"vehicles,omitempty"`
	ObjectiveObjects  *ObjectiveObjectsCoverage   `json:"objectiveObjects,omitempty"`
	Inventory         *InventoryCoverage          `json:"inventory,omitempty"`
	GrenadeReads      *GrenadeReadCoverage        `json:"grenadeReads,omitempty"`
	Zones             *ZonesCoverage              `json:"zones,omitempty"`
	OriginResolved    bool                        `json:"originResolved"`
	T0Film            *T0FilmCoverage             `json:"t0Film,omitempty"`
	Verdict           map[string]string           `json:"verdict,omitempty"`
	Bridge            BridgeHealth                `json:"bridge"`
}

// LayerCoverage est la couverture d'un calque : combien il a rattaché, sur combien
// existaient, et pourquoi il a écarté le reste.
type LayerCoverage struct {
	Available   int `json:"available"`
	Attached    int `json:"attached"`
	NoSlot      int `json:"noSlot"`
	Ambiguous   int `json:"ambiguous"`
	OutOfWindow int `json:"outOfWindow"`
	Unpublished int `json:"unpublished"`
}

// BridgeHealth résume la santé du pont slot -> joueur.
type BridgeHealth struct {
	Slots              int `json:"slots"`
	FromReading        int `json:"fromReading"`
	LivesNamed         int `json:"livesNamed"`
	LivesTotal         int `json:"livesTotal"`
	IndexReadings      int `json:"indexReadings"`
	IndexDisagreements int `json:"indexDisagreements"`
	SlotCollisions     int `json:"slotCollisions"`
	ClosedByShot       int `json:"closedByShot"`
	ClosedByRespawn    int `json:"closedByRespawn"`
	ClosedContested    int `json:"closedContested"`
	ClosedRefused      int `json:"closedRefused"`
}

// T0FilmCoverage est le VERDICT du detecteur, publie dans l'artefact a cote du champ.
type T0FilmCoverage struct {
	Detected bool   `json:"detected"`
	Reason   string `json:"reason,omitempty"`
	Tracks   int    `json:"tracks"`
	Moving   int    `json:"moving"`
	Burst    int    `json:"burst"`
	MarginMs int64  `json:"marginMs"`
}

// PadDatingStats dit ce que la datation a pu faire, et ce qu'elle n'a pas pu.
type PadDatingStats struct {
	Occupations        int `json:"occupations"`
	Dated              int `json:"dated"`
	Named              int `json:"named"`
	Ambiguous          int `json:"ambiguous"`
	Uncovered          int `json:"uncovered"`
	PowerupOccupations int `json:"powerupOccupations"`
}

// InventoryCoverage est la couverture du calque INVENTAIRE (munitions, grenades, capacité,
// emplacement dégainé) : combien de lectures le décodeur a produites, combien ont été
// écartées parce qu'antérieures à l'origine du rejeu, et combien ont été retirées faute de
// trajectoire publiée — le même entonnoir que `Shots` et `Grenades`.
type InventoryCoverage struct {
	Decoded             int `json:"decoded"`
	DroppedBeforeOrigin int `json:"droppedBeforeOrigin"`
	Unpublished         int `json:"unpublished"`
	Published           int `json:"published"`
}

// GrenadeReadCoverage dit ce que chaque canal a apporté. Sans ces dénominateurs, un axe
// clairsemé ne se diagnostique pas : rien ne distinguerait « le film ne transmet pas i22 » de
// « le balayage a échoué ».
type GrenadeReadCoverage struct {
	FromKeyframe int  `json:"fromKeyframe"`
	FromDelta    int  `json:"fromDelta"`
	Unpublished  int  `json:"unpublished"`
	AmmoRefused  bool `json:"ammoRefused,omitempty"`
}

// EquipmentCoverage dit combien de vies publiées portent au moins un épisode, par
// famille — le dénominateur sans lequel « N épisodes » ne se juge pas. Une couverture
// partielle est un résultat, pas un échec : la plupart des vies ne portent NI camouflage
// NI surbouclier, et zéro épisode sur un film sans porteur est la valeur juste.
type EquipmentCoverage struct {
	TracksTotal        int  `json:"tracksTotal"`
	CamoLives          int  `json:"camoLives"`
	CamoEpisodes       int  `json:"camoEpisodes"`
	OvershieldLives    int  `json:"overshieldLives"`
	OvershieldEpisodes int  `json:"overshieldEpisodes"`
	KillsRead          bool `json:"killsRead"`
}

// GrappleCoverage dit ce que le calque a lu et ce qu'il en a publié — le dénominateur
// sans lequel « N tractions » ne se juge pas.
type GrappleCoverage struct {
	LightReads    int `json:"lightReads"`
	HeavyReads    int `json:"heavyReads"`
	Pulls         int `json:"pulls"`
	PullLives     int `json:"pullLives"`
	UnpairedFires int `json:"unpairedFires"`
	BrokenBodies  int `json:"brokenBodies"`
}

// ScoreCoverage dit ce que vaut le calque du score — et ce qu'il ne vaut pas.
type ScoreCoverage struct {
	TeamIdentity  string `json:"teamIdentity"`
	Rounds        int    `json:"rounds"`
	ModeSupported bool   `json:"modeSupported"`
	Truncated     bool   `json:"truncated"`
	Oracle        string `json:"oracle"`
	Points        int    `json:"points"`
}

// WeaponChangeCoverage dit ce que le calque a vu et ce qu'il a écarté, pour qu'un lecteur
// puisse juger sans relire le film.
type WeaponChangeCoverage struct {
	Decoded      int `json:"decoded"`
	Published    int `json:"published"`
	Restated     int `json:"restated"`
	BeforeOrigin int `json:"beforeOrigin"`
	Taken        int `json:"taken"`
	Dropped      int `json:"dropped"`
	Swapped      int `json:"swapped"`
}

// TranslocationCoverage dit ce que le calque a vu et ce qu'il a écarté — le patron des
// autres canaux d'événements.
type TranslocationCoverage struct {
	Events       int `json:"events"`
	Published    int `json:"published"`
	BeforeOrigin int `json:"beforeOrigin"`
	Unpublished  int `json:"unpublished"`
	Positioned   int `json:"positioned"`
}

// AbilityImpulseCoverage dit ce que le calque a lu et ce qu'il a écarté — l'entonnoir
// complet, sans lequel « N impulsions » ne se juge pas.
type AbilityImpulseCoverage struct {
	Reads           int  `json:"reads"`
	Episodes        int  `json:"episodes"`
	Published       int  `json:"published"`
	BeforeOrigin    int  `json:"beforeOrigin"`
	Unpublished     int  `json:"unpublished"`
	NoIdentity      int  `json:"noIdentity"`
	OtherFamily     int  `json:"otherFamily"`
	NoResolver      int  `json:"noResolver"`
	ComponentAbsent bool `json:"componentAbsent,omitempty"`
}

// AbilityChargeCoverage dit ce que le calque a lu et ce qu'il a écarté — l'entonnoir
// complet, sans lequel « N lectures » ne se juge pas. La somme des six cases
// (published + beforeOrigin + unpublished + noIdentity + otherFamily + noResolver) vaut
// EXACTEMENT reads — l'invariant est testé.
type AbilityChargeCoverage struct {
	Reads           int  `json:"reads"`
	Published       int  `json:"published"`
	BeforeOrigin    int  `json:"beforeOrigin"`
	Unpublished     int  `json:"unpublished"`
	NoIdentity      int  `json:"noIdentity"`
	OtherFamily     int  `json:"otherFamily"`
	NoResolver      int  `json:"noResolver"`
	ComponentAbsent bool `json:"componentAbsent,omitempty"`
}

// EquipmentChangeCoverage dit ce que le calque a vu, ce qu'il a écarté, et — seul de tous les
// calques du rejeu — ce qu'il a MANQUÉ.
type EquipmentChangeCoverage struct {
	Decoded           int `json:"decoded"`
	Published         int `json:"published"`
	Taken             int `json:"taken"`
	Spent             int `json:"spent"`
	Spawned           int `json:"spawned"`
	BeforeOrigin      int `json:"beforeOrigin"`
	Lives             int `json:"lives"`
	MissedEstimate    int `json:"missedEstimate"`
	CounterJumps      int `json:"counterJumps"`
	LivesFirstOffSpec int `json:"livesFirstOffSpec"`
	Repeats           int `json:"repeats"`
	Recovered         int `json:"recovered"`
}
