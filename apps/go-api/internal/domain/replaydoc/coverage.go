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
	Abilities         *AbilityCoverage            `json:"abilities,omitempty"`
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
	// Concordant / Discordant : LE PONT PAR MORTS EN TÉMOIN (lot E2, 2026-09-08). Le record de
	// création du bipède écrit le propriétaire du corps ; le pont ne nomme plus, il confronte.
	// Parmi les vies qu'une mort termine ET que la lecture directe nomme, `Concordant` compte
	// celles dont la victime est le joueur lu. Le direct l'emporte toujours : sans ces deux
	// compteurs, la correction serait muette.
	Concordant int `json:"concordant"`
	Discordant int `json:"discordant"`
	// BridgeNamedLives : les vies que le pont a NOMMÉES. Zéro sur tout film dont les créations
	// sont lues — non nul, le registre n'a reçu aucune lecture directe et a dégradé en entier.
	BridgeNamedLives int `json:"bridgeNamedLives"`
	// DirectByCreation / DirectByCreationPropagated : les vies que le RECORD DE CRÉATION nomme,
	// selon qu'il OUVRE cette vie ou un AUTRE séjour du même corps.
	// BodiesWithCreation : les corps dont un record a été lu — le dénominateur.
	DirectByCreation           int `json:"directByCreation"`
	DirectByCreationPropagated int `json:"directByCreationPropagated"`
	BodiesWithCreation         int `json:"bodiesWithCreation"`
	// NamedByPreviousLife / NamedByNextLife / NamedBySlotBridge : ce que le NOMMAGE FINAL a
	// réparé, par voie. UnnamedLives : ce qui a résisté — un DÉFAUT à instruire, jamais une
	// population « inconnue » à afficher (décision produit du 2026-09-07).
	NamedByPreviousLife int `json:"namedByPreviousLife"`
	NamedByNextLife     int `json:"namedByNextLife"`
	NamedBySlotBridge   int `json:"namedBySlotBridge"`
	UnnamedLives        int `json:"unnamedLives"`
	// UnnamedLivesContested : la part du résidu qui tombe sur une frontière entre deux occupants
	// nommés du même slot, que rien ne date — indécidable, pas absente.
	UnnamedLivesContested int `json:"unnamedLivesContested"`
	// DeathOffsetMatched / DeathOffsetRunnerUp : la MARGE du calage du fil des morts — ce que le
	// calage retenu apparie, et ce que le meilleur des AUTRES candidats aurait apparié. Le
	// calage est localisé par un vote puis mesuré par affinage : publier le second compte est ce
	// qui empêche un vote trompé de rendre un calage faux en silence. Un calage vrai écrase ses
	// concurrents (71 contre 8, 157 contre 15 sur les témoins du parc).
	DeathOffsetMatched  int `json:"deathOffsetMatched"`
	DeathOffsetRunnerUp int `json:"deathOffsetRunnerUp"`
	// DeathOffsetMs est LE CALAGE LUI-MEME (lot M1b, 2026-09-08) : `horlogeFilm = horlogeMatch +
	// DeathOffsetMs`. Pointeur, meme piege omitempty que `OriginMs`/`T0FilmMs` (document.go) :
	// absent (nil) veut dire, et seulement, que le pont n'a apparie aucune mort — un calage
	// mesure a zero (horloges deja alignees) est une valeur, pas une absence.
	DeathOffsetMs   *int64 `json:"deathOffsetMs,omitempty"`
	ClosedByShot    int    `json:"closedByShot"`
	ClosedByRespawn int    `json:"closedByRespawn"`
	ClosedContested int    `json:"closedContested"`
	ClosedRefused   int    `json:"closedRefused"`
}

// T0FilmCoverage est le VERDICT du detecteur de coup d'envoi, servi a cote du champ
// `t0FilmMs` : dit-il avoir trouve, sinon pourquoi, et sur quels denominateurs.
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

// AbilityCoverage est la couverture du calque IDENTITÉ DE CAPACITÉ PORTÉE (jumeau de
// `replay.AbilityCoverage`) : lectures i48/image-clé disponibles, celles écartées comme
// BRUIT DE BALAYAGE (rang hors domaine plausible, RAPPORT_E0_2026-09-10 §3), celles sans
// trajectoire publiée, et celles publiées.
type AbilityCoverage struct {
	Reads       int `json:"reads"`
	ScanNoise   int `json:"scanNoise"`
	Unpublished int `json:"unpublished"`
	Published   int `json:"published"`
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
// complet, sans lequel « N lectures » ne se juge pas. Le producteur maintient la somme des
// six cases (published + beforeOrigin + unpublished + noIdentity + otherFamily + noResolver)
// EGALE a reads, et c'est de son cote que l'invariant est teste : ce type ne fait que
// republier les compteurs, il n'en verifie aucun.
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
