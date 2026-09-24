package replaydoc

// coverage.go — CE QUE CHAQUE CALQUE A LU, ET SUR QUEL DENOMINATEUR. Publier « N faits »
// sans dire combien existaient laisserait croire a l'exhaustivite ; ces compteurs sont la
// pour que le client — et un lecteur humain — puissent juger sans relire le film.

// Coverage porte la couverture de chaque calque du document, et le VERDICT qui en découle.
type Coverage struct {
	Shots             LayerCoverage               `json:"shots"`
	Grenades          LayerCoverage               `json:"grenades"`
	Objectives        LayerCoverage               `json:"objectives"`
	Tracks            *TrackCoverage              `json:"tracks,omitempty"`
	Teams             *TeamCoverage               `json:"teams,omitempty"`
	Seats             *SeatCoverage               `json:"seats,omitempty"`
	Projectiles       *ProjectileCoverage         `json:"projectiles,omitempty"`
	Equipment         *EquipmentCoverage          `json:"equipment,omitempty"`
	Stances           *StanceCoverage             `json:"stances,omitempty"`
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
	FilmMajorVersion  *int                        `json:"filmMajorVersion,omitempty"`
	GrenadeReads      *GrenadeReadCoverage        `json:"grenadeReads,omitempty"`
	Abilities         *AbilityCoverage            `json:"abilities,omitempty"`
	Zones             *ZonesCoverage              `json:"zones,omitempty"`
	OriginResolved    bool                        `json:"originResolved"`
	T0Film            *T0FilmCoverage             `json:"t0Film,omitempty"`
	Verdict           map[string]string           `json:"verdict,omitempty"`
	Bridge            BridgeHealth                `json:"bridge"`
	// Fallbacks dit QUELLE PART DE CE DOCUMENT VIENT D'UN REPLI (schéma 58) : les replis
	// déclenchés pendant la cuisson, triés par nom. Absente quand aucun ne s'est déclenché.
	// Le nom est stable et se joint au registre des replis du décodeur.
	Fallbacks []FallbackHit `json:"fallbacks,omitempty"`
	// Decoder dit SOUS QUELLES RÉVISIONS cet artefact a été cuit (schéma 61) : les quatre
	// révisions de calque, la clé du profil, et la classification de l empreinte du registre ECS.
	// Absent = artefact antérieur au schéma 61 — c est une réponse, pas un trou.
	Decoder *DecoderCoverage `json:"decoder,omitempty"`
	// DeathsPaths dit CE QUE CHAQUE VOIE de lecture des morts a propose, apparie et publie
	// (schema 62) : la MARCHE et le SCAN DIRECT lisent le meme champ par deux localisateurs, de
	// precisions differentes, et un consommateur doit pouvoir ponderer ce qu il lit. Absent =
	// artefact anterieur au schema 62, ou morts non lues sur ce match.
	DeathsPaths *DeathsPathsCoverage `json:"deathsPaths,omitempty"`
}

// FallbackHit est un repli du décodeur et son nombre de déclenchements sur la cuisson qui a
// produit ce document.
type FallbackHit struct {
	Name string `json:"name"`
	Hits int    `json:"hits"`
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
	// RefusedByRoster : actions non publiées parce que l'effectif du match dépasse les huit
	// slots d'entité de joueur du statborg (calque des objectifs uniquement).
	//
	// `omitempty` COMME CÔTÉ STOCKÉ, et le garde-rail de parité l'exige : le tag doit être le
	// MÊME des deux côtés, sinon le client lirait un autre nom que celui que la cuisson écrit
	// (`replayview/parity_test.go`). Côté artefact, l'option évite de changer la forme des
	// 65 documents où le compteur vaut zéro — donc de recuire le parc pour un champ vide.
	RefusedByRoster int `json:"refusedByRoster,omitempty"`
}

// BridgeHealth résume la santé du pont slot -> joueur.
type BridgeHealth struct {
	Slots       int `json:"slots"`
	FromReading int `json:"fromReading"`
	LivesNamed  int `json:"livesNamed"`
	LivesTotal  int `json:"livesTotal"`
	// DeathsFeed : le VERDICT de la lecture du fil des morts (schéma 69) — `read`, `empty` (le
	// morceau des temps forts est lu et ne porte aucune mort) ou `unreadable` (panne). Absent =
	// non mesuré. `omitempty` comme côté stocké (garde-rail de parité).
	DeathsFeed         string `json:"deathsFeed,omitempty"`
	IndexReadings      int    `json:"indexReadings"`
	IndexDisagreements int    `json:"indexDisagreements"`
	SlotCollisions     int    `json:"slotCollisions"`
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

// TeamCoverage est ce que la lecture de l'EQUIPE a couvert, et ce que la feuille de match en
// pense (schema 57). Elle publie les deux moities separement : `film` dit ce que l'artefact
// tient du FILM, `accord` / `contradiction` / `silence` disent ce qu'une source EXTERIEURE en
// pense — la base ne pose aucune equipe, elle controle.
//
// C'EST LA SEULE FACON DE LIRE UN `team: -1` : `noTeam` (le mode n'a pas de camps) et `unread`
// (le film n'a pas nomme ce joueur) distinguent ce que le champ ne distingue pas.
type TeamCoverage struct {
	Read          bool   `json:"read"`
	Refusal       string `json:"refusal,omitempty"`
	Records       int    `json:"records"`
	Rejected      int    `json:"rejected"`
	Divergences   int    `json:"divergences"`
	Film          int    `json:"film"`
	NoTeam        int    `json:"noTeam"`
	Unread        int    `json:"unread"`
	Accord        int    `json:"accord"`
	Contradiction int    `json:"contradiction"`
	Silence       int    `json:"silence"`
	Tracks        int    `json:"tracks"`
	TracksNamed   int    `json:"tracksNamed"`
	// TracksSlotAmbiguous : les vies sans xuid dont le SLOT a porte deux joueurs nommes
	// d'equipes differentes. Le pont slot -> index s'y ABSTIENT (revue de jalon M1, lentille
	// L4) plutot que de publier l'equipe du PREMIER occupant sur la vie du SECOND. Absent du
	// document quand il vaut zero.
	TracksSlotAmbiguous int `json:"tracksSlotAmbiguous,omitempty"`
}

// SeatCoverage est ce que la pose des SIEGES a lu et ce qu elle a APPARIE (lot 1.9.14).
//
// SON COUPLE CENTRAL EST `entrees` / `occupantsMax` : leur ECART est le nombre de fiches qu un
// client retire de l ecran en n affichant que les occupants PRESENTS a l instant lu. `lus` et
// `apparies` disent, eux, quelle part des sieges vient du film et quelle part d un repli.
type SeatCoverage struct {
	Entrees         int  `json:"entrees"`
	Sieges          int  `json:"sieges"`
	Lus             int  `json:"lus"`
	Apparies        int  `json:"apparies"`
	ReprisesEcrites int  `json:"reprisesEcrites"`
	Arrivants       int  `json:"arrivants"`
	PresencesCloses int  `json:"presencesCloses"`
	SansPresence    int  `json:"sansPresence"`
	OccupantsMax    int  `json:"occupantsMax"`
	SansTableDuFilm bool `json:"sansTableDuFilm,omitempty"`
}

// TrackCoverage est ce que le SEUIL DE PUBLICATION des traces retient et refuse. Le refus était
// MUET avant le schéma 55 : un document publiant 90 traces là où le film en porte 95 était
// indistinguable d'un film à 90 vies. `minPoints` voyage avec ses conséquences — un compte de
// refus ne se relit pas sans savoir contre quoi il a été mesuré.
type TrackCoverage struct {
	Published        int `json:"published"`
	PublishedPoints  int `json:"publishedPoints"`
	RefusedMinPoints int `json:"refusedMinPoints"`
	RefusedPoints    int `json:"refusedPoints"`
	MinPoints        int `json:"minPoints"`
	// Gaps est le nombre de LACUNES des traces publiees, GapMS leur duree totale en
	// millisecondes : un silence de replication de plus de 5 s A L INTERIEUR d une vie, que le
	// film ne ferme pas. Cf. `replay.TrackCoverage` (lot 1.9.13).
	Gaps  int `json:"gaps"`
	GapMS int `json:"gapMs"`
	// Ce que la PORTE DES POSITIONS a ecarte avant toute publication, en positions brutes du
	// film (schema 69, lot M1 des retours du rejeu) : anterieures a la creation de leur corps,
	// dont les vies ecartees entieres, et hors de l emprise jouee ; les slots ou la regle de
	// creation s arme ou se desarme. Cf. `replay.TrackCoverage`.
	AvantCreation             int `json:"avantCreation"`
	ViesAvantPremiereCreation int `json:"viesAvantPremiereCreation"`
	HorsEmprise               int `json:"horsEmprise"`
	SlotsArmes                int `json:"slotsArmes"`
	SlotsDesarmes             int `json:"slotsDesarmes"`
}

// ProjectileCoverage est la couverture des TRAJECTOIRES DE PROJECTILE : pistes décodées,
// trajectoires publiées, et celles qu'un PAS IMPOSSIBLE a coupées. Tant que `truncated` n'est
// pas nul, l'artefact porte des vols dont la fin est INCONNUE — la coupure protège le rendu,
// elle ne répare pas la déquantification qui la cause.
type ProjectileCoverage struct {
	Tracks    int `json:"tracks"`
	Published int `json:"published"`
	Truncated int `json:"truncated"`
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

// StanceCoverage dit ce que la marche des ETATS DE MOUVEMENT a lu et ce qu'elle a jeté — les
// dénominateurs sans lesquels « N intervalles » ne se juge pas. Une couverture partielle est un
// RESULTAT : la plupart des vies ne s'accroupissent ni ne glissent.
//
// `JumpEpisodes` / `JumpsDerived` (schéma 66) sont le dénominateur et le numérateur du SAUT,
// qui est DÉRIVÉ et non lu : montées fermées examinées, puis celles dont la hauteur intégrée
// tombe dans la fenêtre du saut du Spartan. Le rapport des deux est la sélectivité.
type StanceCoverage struct {
	Scanned               bool           `json:"scanned"`
	Absent                bool           `json:"absent,omitempty"`
	Records               int            `json:"records"`
	Desyncs               int            `json:"desyncs"`
	Reads                 int            `json:"reads"`
	Intervals             int            `json:"intervals"`
	JumpEpisodes          int            `json:"jumpEpisodes,omitempty"`
	JumpsDerived          int            `json:"jumpsDerived,omitempty"`
	ByKind                map[string]int `json:"byKind,omitempty"`
	Lives                 int            `json:"lives"`
	TracksTotal           int            `json:"tracksTotal"`
	Dropped               int            `json:"dropped,omitempty"`
	EventPacketsUnlocated int            `json:"eventPacketsUnlocated,omitempty"`
	MapWidths             [3]uint        `json:"mapWidths,omitempty"`
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
//
// LES QUATRE CHAMPS DE MANCHE (lot 1.9.11, 2026-09-16) DISENT LE FAIT « quelles manches sont
// RÉELLES » sous les trois formes que la décision D14 (c) du PLAN_DECODEUR_FILM exige :
// la GRAMMAIRE (`roundsWritten`, ce que le film écrit), la CONTRADICTION
// (`roundsContradicted` / `roundsContradictedRecords`, un désignateur matériel que l'ordre des
// manches refuse) et le REPLI (`roundsDecreed`, la manche 0 décrétée quand le film est muet).
// `rounds` reste le COMPTE des manches retenues — la grandeur que le rejeu consomme.
//
// TOUS OPTIONNELS : un film mono-manche sans contradiction ni repli ne les porte pas, et le
// document garde la forme qu'il avait.
type ScoreCoverage struct {
	TeamIdentity  string `json:"teamIdentity"`
	Rounds        int    `json:"rounds"`
	ModeSupported bool   `json:"modeSupported"`
	Truncated     bool   `json:"truncated"`
	Oracle        string `json:"oracle"`
	Points        int    `json:"points"`
	// RoundsWritten : les désignateurs de manche que le film ÉCRIT, triés — le dénominateur
	// sans lequel « N manches » ne se juge pas. Publié dès qu'il y en a plus d'un (un film
	// mono-manche n'apprendrait rien).
	RoundsWritten []int `json:"roundsWritten,omitempty"`
	// RoundsContradicted : les désignateurs MATÉRIELS que l'ordre des manches refuse, triés.
	// Un désignateur y figure parce que le film ne déclare aucune des manches qui le précèdent
	// — mesuré sur 24 films du cache, dont 23 ont fini dans leur temps réglementaire sur un
	// mode sans manche (lot 1.9.11).
	RoundsContradicted []int `json:"roundsContradicted,omitempty"`
	// RoundsContradictedRecords : les enregistrements que ces désignateurs portent.
	RoundsContradictedRecords int `json:"roundsContradictedRecords,omitempty"`
	// RoundsDecreed : aucune manche n'a été admise et la manche 0 a été DÉCRÉTÉE pour que le
	// film reste lisible (repli `repli_manche_zero_decretee`).
	RoundsDecreed bool `json:"roundsDecreed,omitempty"`
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
	Reads           int                         `json:"reads"`
	Episodes        int                         `json:"episodes"`
	Published       int                         `json:"published"`
	BeforeOrigin    int                         `json:"beforeOrigin"`
	Unpublished     int                         `json:"unpublished"`
	NoIdentity      int                         `json:"noIdentity"`
	OtherFamily     int                         `json:"otherFamily"`
	NoResolver      int                         `json:"noResolver"`
	ComponentAbsent bool                        `json:"componentAbsent,omitempty"`
	Scan            *AbilityImpulseScanCoverage `json:"scan,omitempty"`
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

// DecoderCoverage dit SOUS QUELLES RÉVISIONS cet artefact a été cuit (schéma 61).
//
// Télémétrie pure : aucun rendu n'en dépend. Les quatre révisions sont celles des calques du
// décodeur, dans l'ordre du sens unique ; `build` est la clé du profil, lue en clair dans la
// section 2 de `chunk_00`, et elle est VIDE — le bloc restant présent — quand le film n'écrit
// pas de build ou quand le profil ne le connaît pas.
type DecoderCoverage struct {
	SourceRev  string            `json:"sourceRev"`
	ProfileRev string            `json:"profileRev"`
	GrammarRev string            `json:"grammarRev"`
	FactsRev   string            `json:"factsRev"`
	Build      string            `json:"build"`
	Registry   *RegistryCoverage `json:"registry,omitempty"`
}

// RegistryCoverage CLASSE l'empreinte du registre ECS du film — la grammaire de ses composants.
// Absent quand le registre n'a pas été lu. `status` vaut `connue` ou `inconnue` ; un troisième
// état `presumee` est prévu et c'est pourquoi le champ est une chaîne.
type RegistryCoverage struct {
	Fingerprint string `json:"fingerprint"`
	Status      string `json:"status"`
	Blocks      int    `json:"blocks"`
	NamedSlots  int    `json:"namedSlots"`
}

// AbilityImpulseScanCoverage porte les dénominateurs du BALAYAGE du canal d'impulsion — ce que
// la marche a rencontré, avant tout repliement en épisodes. Absent quand le balayage n'a pas
// tourné : un bloc de zéros affirmerait qu'il a tourné sans rien rencontrer.
type AbilityImpulseScanCoverage struct {
	Records int `json:"records"`
	WithI57 int `json:"withI57"`
	WithI59 int `json:"withI59"`
	Read    int `json:"read"`
	Unread  int `json:"unread"`
	Tag1    int `json:"tag1"`
}

// DeathsPathsCoverage dit CE QUE CHAQUE VOIE DE LECTURE DES MORTS A PROPOSE, APPARIE ET PUBLIE
// (schéma 62). Les deux voies lisent le MÊME champ par deux localisateurs, et quand les deux
// répondent elles répondent au même bit : ce n'est pas un arbitrage, c'est une préférence — mais
// leurs précisions diffèrent (98,2 % contre 78,4 % au gate d'appariement sur la série de
// référence), d'où la publication du compte par voie.
type DeathsPathsCoverage struct {
	// Walk : la MARCHE, qui déroule les records depuis le début du paquet.
	Walk DeathsPathTally `json:"walk"`
	// Scan : le SCAN DIRECT, qui balaie les positions de bit — la voie de rattrapage.
	// LA CLE N EST PAS `scan`, ET C EST UN RATCHET DU DEPOT QUI LE DECIDE : `"scan"` en litteral
	// brut est interdit hors de `domain/killscope` et de `killsource` (J4R-3,
	// `archlint/no_raw_kill_scope_literal_test.go`) — c est sur cette valeur que se decide la
	// PRESEANCE des ecrivains de `shared.match_kill_events`, et une seconde copie libre de deriver
	// y rendrait la preseance du film aveugle SANS erreur ni compteur. `directScan` est le nom que
	// le code donne deja a cette voie (« le SCAN DIRECT »).
	Scan DeathsPathTally `json:"directScan"`
}

// DeathsPathTally : les trois dénominateurs d'une voie de lecture des morts. La somme des deux
// `published` ne se lit pas comme un total de morts — une mort couverte par la voie la plus
// contrainte ne l'est pas deux fois.
type DeathsPathTally struct {
	Population int `json:"population"`
	Matched    int `json:"matched"`
	Published  int `json:"published"`
}
