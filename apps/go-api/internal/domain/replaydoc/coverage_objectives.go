package replaydoc

// coverage_objectives.go — LES COUVERTURES DES CALQUES D'OBJECTIF (drapeau, couronne, crane,
// bombe, objets, zones). Chacune est publiee MEME VIDE : son absence dit autre chose que
// ses zeros — l'appelant n'a pas reconnu le mode.

// FlagCarriesCoverage porte les denominateurs du calque. Sans eux, « 12 portages » se lirait
// comme une exhaustivite, et un film CTF sans aucun portage publie serait indistinguable d'un
// film qui n'est pas du CTF.
type FlagCarriesCoverage struct {
	FlagFilm              bool `json:"flagFilm"`
	Bursts                int  `json:"bursts"`
	Captures              int  `json:"captures"`
	Steals                int  `json:"steals"`
	Openings              int  `json:"openings"`
	Carries               int  `json:"carries"`
	Closed                int  `json:"closed"`
	Open                  int  `json:"open"`
	NoBridge              int  `json:"noBridge"`
	NoTrack               int  `json:"noTrack"`
	OutOfWindow           int  `json:"outOfWindow"`
	MarkerObserved        int  `json:"markerObserved"`
	MarkerConfirmed       int  `json:"markerConfirmed"`
	OpenObserved          int  `json:"openObserved"`
	OpenConfirmed         int  `json:"openConfirmed"`
	Overlaps              int  `json:"overlaps"`
	ClosedOverlaps        int  `json:"closedOverlaps"`
	AmbiguousCarrierKills int  `json:"ambiguousCarrierKills"`
	AmbiguousReturns      int  `json:"ambiguousReturns"`
	HomeByObject          int  `json:"homeByObject"`
	AmbiguousHomecomings  int  `json:"ambiguousHomecomings"`
	NeutralFlag           bool `json:"neutralFlag"`
	NeutralBirths         int  `json:"neutralBirths"`
	TeamBirths            int  `json:"teamBirths"`
	Spawns                int  `json:"spawns"`
	ObjectLives           int  `json:"objectLives"`
	ClosedByObject        int  `json:"closedByObject"`
	DropsRepositioned     int  `json:"dropsRepositioned"`
}

// VipCrownCoverage porte les denominateurs du calque. Sans eux, « 15 periodes » se lirait comme
// une exhaustivite, et un film VIP sans aucune periode publiee serait indistinguable d'un film
// non-VIP. ELLE EST PUBLIEE MEME QUAND AUCUNE PERIODE NE L'EST, pour la meme raison que les
// couvertures freres ; son ABSENCE dit encore autre chose : l'appelant n'a pas reconnu un film VIP.
type VipCrownCoverage struct {
	VipFilm           bool `json:"vipFilm"`
	Selections        int  `json:"selections"`
	Periods           int  `json:"periods"`
	Closed            int  `json:"closed"`
	Open              int  `json:"open"`
	ClosedByDeath     int  `json:"closedByDeath"`
	ClosedBySelection int  `json:"closedBySelection"`
	NoBridge          int  `json:"noBridge"`
	OutOfWindow       int  `json:"outOfWindow"`
}

// SkullCarriesCoverage porte les denominateurs du calque. Sans eux, « 8 portages » se lirait
// comme une exhaustivite, et un film Oddball sans aucun portage publie serait indistinguable d'un
// film non-Oddball. ELLE EST PUBLIEE MEME QUAND AUCUN PORTAGE NE L'EST, pour la meme raison que
// les couvertures freres ; son ABSENCE dit encore autre chose : l'appelant n'a pas reconnu un
// film Oddball.
type SkullCarriesCoverage struct {
	SkullFilm     bool `json:"skullFilm"`
	Grabs         int  `json:"grabs"`
	Trains        int  `json:"trains"`
	Carries       int  `json:"carries"`
	Closed        int  `json:"closed"`
	Open          int  `json:"open"`
	NoBridge      int  `json:"noBridge"`
	OutOfWindow   int  `json:"outOfWindow"`
	CarrierAbsent int  `json:"carrierAbsent"`
}

// BombCarriesCoverage porte les denominateurs du calque. Sans eux, « 5 portages » se lirait
// comme une exhaustivite, et un film d'Assaut sans aucun portage publie serait indistinguable
// d'un film d'un autre mode. ELLE EST PUBLIEE MEME QUAND AUCUN PORTAGE NE L'EST ; son ABSENCE
// dit encore autre chose : l'appelant n'a pas reconnu un film d'Assaut.
type BombCarriesCoverage struct {
	BombFilm      bool `json:"bombFilm"`
	Events        int  `json:"events"`
	Periods       int  `json:"periods"`
	Carries       int  `json:"carries"`
	Closed        int  `json:"closed"`
	Open          int  `json:"open"`
	ByDeath       int  `json:"byDeath"`
	NoBridge      int  `json:"noBridge"`
	OutOfWindow   int  `json:"outOfWindow"`
	CarrierAbsent int  `json:"carrierAbsent"`
}

// BombArmingsCoverage dit ce que la lecture de l'anneau a rendu — et pourquoi le calque
// peut être vide alors que le film a été lu (même doctrine que les autres couvertures).
type BombArmingsCoverage struct {
	Scanned            bool `json:"scanned"`
	Reads              int  `json:"reads"`
	Rises              int  `json:"rises"`
	BelowFull          int  `json:"belowFull"`
	Armed              int  `json:"armed"`
	PairMerged         int  `json:"pairMerged"`
	Published          int  `json:"published"`
	OutOfWindow        int  `json:"outOfWindow"`
	Detonations        int  `json:"detonations"`
	DetonationsCovered int  `json:"detonationsCovered"`
	Suppressed         bool `json:"suppressed,omitempty"`
}

// BombStatsCoverage dit ce qui a été lu et ce qui a été écarté : publier des chiffres sans
// dire sur quel dénominateur ils portent laisserait croire à l'exhaustivité.
type BombStatsCoverage struct {
	DetonationsRead      bool `json:"detonationsRead"`
	CarryRead            bool `json:"carryRead"`
	KillsRead            bool `json:"killsRead"`
	ArmingsRead          bool `json:"armingsRead"`
	Detonations          int  `json:"detonations"`
	Armings              int  `json:"armings"`
	ArmingsAttributed    int  `json:"armingsAttributed"`
	ArmingsByDrop        int  `json:"armingsByDrop"`
	ArmingsByActiveCarry int  `json:"armingsByActiveCarry"`
	ArmingsNoCarrier     int  `json:"armingsNoCarrier"`
	ArmingsNoBridge      int  `json:"armingsNoBridge"`
	ArmingsAmbiguous     int  `json:"armingsAmbiguous"`
	Periods              int  `json:"periods"`
	PeriodsNoBridge      int  `json:"periodsNoBridge"`
	PeriodsOpen          int  `json:"periodsOpen"`
	PeriodsByDeath       int  `json:"periodsByDeath"`
	Kills                int  `json:"kills"`
	KillsOnCarrier       int  `json:"killsOnCarrier"`
	Players              int  `json:"players"`
}

// ObjectiveObjectsCoverage porte les dénominateurs du calque. Sans eux, « 16 vies » se lirait
// comme une exhaustivité, et un film sans aucune vie publiée serait indistinguable d'un film
// qu'on n'a pas su lire.
type ObjectiveObjectsCoverage struct {
	Scanned    bool `json:"scanned"`
	Declared   int  `json:"declared"`
	Lives      int  `json:"lives"`
	Points     int  `json:"points"`
	Motionless int  `json:"motionless"`
	OutOfAxis  int  `json:"outOfAxis"`
}

// ZonesCoverage porte les denominateurs du calque. Sans eux, « 3 zones » se lirait comme une
// exhaustivite, et un film d'un autre mode serait indistinguable d'un film dont l'appariement a
// echoue.
type ZonesCoverage struct {
	Method        string `json:"method"`
	Roles         string `json:"roles,omitempty"`
	Catalog       int    `json:"catalog"`
	Slots         int    `json:"slots"`
	Paired        int    `json:"paired"`
	Unpaired      int    `json:"unpaired"`
	Captures      int    `json:"captures"`
	Attributed    int    `json:"attributed"`
	OwnerChecked  int    `json:"ownerChecked"`
	OwnerAgreed   int    `json:"ownerAgreed"`
	OwnerUnpaired int    `json:"ownerUnpaired"`
	Spans         int    `json:"spans"`
	HillPeriods   int    `json:"hillPeriods"`
	UnknownOwner  int    `json:"unknownOwner"`
	Letters       int    `json:"letters"`
	GaugePoints   int    `json:"gaugePoints"`
}
