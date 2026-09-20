package replaydoc

// document.go — LA RACINE DU DOCUMENT SERVI et les types qu'un client dessine sans
// contexte : trajectoires, tirs, grenades, projectiles, libelles.
//
// `ReplayDocument` est le corps de `GET /players/{slug}/matches/{id}/replay`. Son champ
// `schemaVersion` porte la version de l'ARTEFACT LU, celle du producteur au moment de la
// cuisson : c'est elle qui dit au parc « a re-cuire », et le client la lit telle quelle. Le
// paquet servi ne porte, lui, aucun numero de version (cf. doc.go).

// ReplayDocument est le rejeu 2D sérialisé d'un match.
type ReplayDocument struct {
	SchemaVersion       int                     `json:"schemaVersion"`
	MatchID             string                  `json:"matchId"`
	TitleSlug           string                  `json:"titleSlug"`
	FrameCount          int                     `json:"frameCount"`
	Bounds              Bounds                  `json:"bounds"`
	Tracks              []Track                 `json:"tracks"`
	FrameIntervalMS     int                     `json:"frameIntervalMs,omitempty"`
	DurationMS          int                     `json:"durationMs,omitempty"`
	OriginMs            *int64                  `json:"originMs,omitempty"`
	T0FilmMs            *int64                  `json:"t0FilmMs,omitempty"`
	Geometry            []MapObject             `json:"geometry,omitempty"`
	GeometryBounds      *Bounds                 `json:"geometryBounds,omitempty"`
	Structure           []Surface               `json:"structure,omitempty"`
	StructureBounds     *Bounds                 `json:"structureBounds,omitempty"`
	Shots               []Shot                  `json:"shots,omitempty"`
	Loadouts            []Loadout               `json:"loadouts,omitempty"`
	Inventory           []Inventory             `json:"inventory,omitempty"`
	GrenadeLabels       []Label                 `json:"grenadeLabels,omitempty"`
	Abilities           []AbilityRead           `json:"abilities,omitempty"`
	GrenadeReads        []GrenadeRead           `json:"grenadeReads,omitempty"`
	AbilityLabels       map[string]Label        `json:"abilityLabels,omitempty"`
	EquipmentEpisodes   []EquipmentEpisode      `json:"equipmentEpisodes,omitempty"`
	GrappleLines        []GrappleLine           `json:"grappleLines,omitempty"`
	EquipmentPlacements []EquipmentPlacement    `json:"equipmentPlacements,omitempty"`
	WeaponChanges       []WeaponChange          `json:"weaponChanges,omitempty"`
	Pickups             []Pickup                `json:"pickups,omitempty"`
	EquipmentChanges    []EquipmentChange       `json:"equipmentChanges,omitempty"`
	Translocations      []Translocation         `json:"translocations,omitempty"`
	AbilityImpulses     []AbilityImpulse        `json:"abilityImpulses,omitempty"`
	AbilityCharges      []AbilityCharge         `json:"abilityCharges,omitempty"`
	GroundWeapons       []GroundWeapon          `json:"groundWeapons,omitempty"`
	Vehicles            []VehicleTrack          `json:"vehicles,omitempty"`
	VehicleLabels       map[string]VehicleLabel `json:"vehicleLabels,omitempty"`
	VehicleCycles       []VehicleCycle          `json:"vehicleCycles,omitempty"`
	WeaponPads          []WeaponPad             `json:"weaponPads,omitempty"`
	PadPickups          []PadPickup             `json:"padPickups,omitempty"`
	Grenades            []Grenade               `json:"grenades,omitempty"`
	Projectiles         []Projectile            `json:"projectiles,omitempty"`
	WeaponLabels        map[string]WeaponLabel  `json:"weaponLabels,omitempty"`
	KillEffects         map[string]string       `json:"killEffects,omitempty"`
	NeutralDeaths       []NeutralDeath          `json:"neutralDeaths,omitempty"`
	Roster              []RosterEntry           `json:"roster,omitempty"`
	MapObjectives       *MapObjectives          `json:"mapObjectives,omitempty"`
	MapWeaponPads       *MapWeaponPads          `json:"mapWeaponPads,omitempty"`
	WeaponTiers         *WeaponTiersInfo        `json:"weaponTiers,omitempty"`
	Objectives          []ObjectiveAction       `json:"objectives,omitempty"`
	ScoreTimeline       *ScoreTimeline          `json:"scoreTimeline,omitempty"`
	FlagCarries         []FlagCarry             `json:"flagCarries,omitempty"`
	FlagReturnZone      *FlagReturnZone         `json:"flagReturnZone,omitempty"`
	ObjectiveObjects    []ObjectiveObjectLife   `json:"objectiveObjects,omitempty"`
	ZoneStates          []ZoneState             `json:"zoneStates,omitempty"`
	VipCrown            []VipPeriod             `json:"vipCrown,omitempty"`
	BombArmings         []BombArming            `json:"bombArmings,omitempty"`
	SkullCarries        []SkullCarry            `json:"skullCarries,omitempty"`
	BombCarries         []BombCarry             `json:"bombCarries,omitempty"`
	BombStats           *BombMatchStats         `json:"bombStats,omitempty"`
	BombEvents          []BombEvent             `json:"bombEvents,omitempty"`
	Coverage            *Coverage               `json:"coverage,omitempty"`
	Identity            *IdentitySection        `json:"identity,omitempty"`
	// Layers dit, CALQUE PAR CALQUE, SOUS QUELLE REVISION DE COUCHE il a ete produit (schema 62).
	// La cle est la balise JSON du calque a cette racine ; la valeur est `source-...`,
	// `profile-...`, `grammar-...`, `killsource-...` ou `publication-<schemaVersion>`.
	//
	// Objet ABSENT = artefact anterieur au schema 62 ; entree ABSENTE dans un objet PRESENT = ce
	// calque n a pas ete produit, et c est une reponse, pas un trou ; entree presente = produit
	// sous la revision nommee. Les calques resolus a la requete (`mapObjectives`, `mapWeaponPads`,
	// `weaponTiers`, `vehicleLabels`) n y figurent jamais : la cuisson ne les ecrit pas.
	Layers map[string]string `json:"layers,omitempty"`
}

// Bounds est l'étendue alignée sur les axes de tous les points de trajectoire, dans le
// repère monde partagé. Permet au client d'ajuster la scène au viewport (le range monde
// absolu est inutile au rendu — seule la disposition relative importe).
type Bounds struct {
	MinX float32 `json:"minX"`
	MinY float32 `json:"minY"`
	MaxX float32 `json:"maxX"`
	MaxY float32 `json:"maxY"`
	MinZ float32 `json:"minZ,omitempty"`
	MaxZ float32 `json:"maxZ,omitempty"`
}

// Track est la trajectoire d'une entité (slot biped) sur la timeline du rejeu.
type Track struct {
	Slot       uint32  `json:"slot"`
	Team       int     `json:"team"`
	Name       string  `json:"name,omitempty"`
	XUID       string  `json:"xuid,omitempty"`
	Bot        string  `json:"bot,omitempty"`
	Points     []Point `json:"points"`
	StartFrame int     `json:"startFrame,omitempty"`
	EndFrame   int     `json:"endFrame,omitempty"`
}

// Point est une position echantillonnee au pas de temps T. X/Y = plan horizontal de la
// carte ; Z (optionnel) = altitude, pour l'indication d'etage — non critique au rendu 2D.
type Point struct {
	T int     `json:"t"`
	X float32 `json:"x"`
	Y float32 `json:"y"`
	Z float32 `json:"z,omitempty"`
	// G (optionnel) est la DUREE DE LA LACUNE qui precede ce point, en millisecondes ; absent
	// ou 0 = le point suit le precedent sans interruption. La piste ne s'interpole PAS au
	// travers. Cf. `replay.Point.G` pour la decision complete (lot 1.9.13).
	G  int      `json:"g,omitempty"`
	H  float32  `json:"h,omitempty"`
	P  float32  `json:"p,omitempty"`
	Sh *float32 `json:"sh,omitempty"`
	Hp *float32 `json:"hp,omitempty"`
	S  int      `json:"s,omitempty"`
}

// RosterEntry est un joueur du film : son identité, et l'index sous lequel le film le désigne.
type RosterEntry struct {
	XUID      string `json:"xuid"`
	FilmIndex int    `json:"filmIndex"`
	Name      string `json:"name,omitempty"`
	// Team est le DESIGNATEUR D'EQUIPE que le film ecrit, par index de joueur (schema 57) :
	// `0..8` pour les huit camps de `mp_team_designator`, `-1` pour « aucune equipe ». Il vaut
	// aussi pour un BOT, que le film assoit au meme index.
	//
	// POINTEUR, parce que TROIS etats existent : absent (le film n'a pas nomme ce joueur, ou
	// l'artefact precede le schema 57), `-1` (aucune equipe), `0..8` (le camp). Un entier nu
	// ferait dire `0` — le camp 0 — a tout artefact ancien.
	Team *int `json:"team,omitempty"`
	Bot  bool `json:"bot,omitempty"`
	// Bid est l identifiant STABLE d un bot, forme `bid(N.0)` — la meme que la base emploie
	// (schema 50). Vide pour un humain, et vide pour un bot dont la declaration ne portait pas
	// d identifiant : un `bid(0.0)` invente joindrait deux bots distincts.
	Bid string `json:"bid,omitempty"`
	// Seat est LE SIEGE : la fiche que cette entree occupe a l ecran (lot 1.9.14). Il vaut
	// `filmIndex` sauf quand l entree CONTINUE le siege d un partant ; `seatSource` dit alors si
	// le film a ECRIT la reprise (`lu`) ou si un appariement ordinal l a deduite (`apparie`).
	// Deux entrees de meme `seat` sont deux occupants SUCCESSIFS d une meme fiche, et leurs
	// presences — les vies de `tracks[]` — ne se recouvrent pas.
	//
	// TOUJOURS EMIS : le siege 0 est un siege comme un autre, et `omitempty` l effacerait.
	Seat int `json:"seat"`
	// SeatSource : `lu` (l index que le film ecrit) ou `apparie` (l appariement ordinal par
	// camp, un repli nomme et compte). Vide sur un artefact anterieur au lot 1.9.14.
	SeatSource string `json:"seatSource,omitempty"`
}

// Shot est un tir décodé, placé à la position de son tireur.
type Shot struct {
	T       int     `json:"t"`
	Slot    uint32  `json:"slot"`
	X       float32 `json:"x"`
	Y       float32 `json:"y"`
	H       float32 `json:"h,omitempty"`
	Weapon  string  `json:"w,omitempty"`
	Vehicle *uint32 `json:"v,omitempty"`
}

// Loadout est l'ensemble des armes PORTÉES par un slot à un instant de référence.
type Loadout struct {
	T    int      `json:"t"`
	Slot uint32   `json:"slot"`
	W    []string `json:"w"`
}

// Grenade est un lancer de grenade, situé dans le temps et l'espace.
type Grenade struct {
	T    int     `json:"t"`
	Slot uint32  `json:"slot"`
	Idx  int     `json:"i"`
	X    float32 `json:"x"`
	Y    float32 `json:"y"`
	Rank int     `json:"rank"`
	Src  string  `json:"s"`
	Proj *int    `json:"proj,omitempty"`
}

// Projectile est une trajectoire de projectile, échantillonnée sur la grille du rejeu.
type Projectile struct {
	T0   int          `json:"t0"`
	P    [][3]float32 `json:"p"`
	Rest bool         `json:"rest,omitempty"`
}

// NeutralDeath est une mort que PERSONNE ne revendique — et de quoi le joueur est mort.
type NeutralDeath struct {
	XUID   string `json:"xuid"`
	FeedMs int    `json:"feedMs"`
	Kind   string `json:"kind"`
	Img    string `json:"img,omitempty"`
	Tinted bool   `json:"tinted,omitempty"`
}

// Label est un libellé affichable dans les deux langues du produit.
type Label struct {
	En     string `json:"en"`
	Fr     string `json:"fr"`
	Img    string `json:"img,omitempty"`
	Tinted bool   `json:"tinted,omitempty"`
	// Family est l'identite STABLE de la chose nommee, dans le vocabulaire des familles
	// d'equipement du titre (`wall`, `sensor`, `powerup_camo`...) — schema 51, lot 4.3. Le rang
	// d'une capacite n'est PAS une identite (le propulseur vaut 5 en famille A et 21 en famille
	// B) : c'est ce champ qui permet a un lecteur de designer une capacite sans ecrire son rang.
	// Vide = le manifeste ne classe pas ce rang.
	Family string `json:"family,omitempty"`
}

// WeaponLabel est le libellé d'une arme, plus l'EFFET de rendu de ses tirs.
type WeaponLabel struct {
	En  string `json:"en"`
	Fr  string `json:"fr"`
	Fx  string `json:"fx,omitempty"`
	Key string `json:"key,omitempty"`
	// Role est la fonction de combat (sniper, power, special...) postée à la requête depuis
	// le registre canonique — filtre « armes spéciales » du calque des armes au sol (lot
	// 2026-09-10). Cf. replay.WeaponLabel.Role pour la décision complète.
	Role   string `json:"role,omitempty"`
	Tint   string `json:"tint,omitempty"`
	Img    string `json:"img,omitempty"`
	Tinted bool   `json:"tinted,omitempty"`
}

// VehicleLabel est ce qu il faut pour DESSINER une famille de chassis : sa vignette, le fait
// qu elle se teigne, et — depuis le lot 1.9.9 — ce qu elle EST quand ce n est pas un vehicule.
//
// `Kind`, `En` et `Fr` sont OPTIONNELS et presque toujours vides : le nom d une famille de
// vehicule est un nom propre du jeu, qui ne se traduit pas, et la cle de la table EST ce nom. Ils
// ne se remplissent que pour les familles que le titre QUALIFIE dans son manifeste — la tourelle
// automatique bannie (`kind = "map_element"`), aujourd hui la seule.
type VehicleLabel struct {
	Img    string `json:"img,omitempty"`
	Tinted bool   `json:"tinted,omitempty"`
	// Kind : la NATURE de la famille quand elle n est pas un vehicule de la partie. Vide = un
	// vehicule. Le client s en sert pour lui reserver un pictogramme dedie plutot que le
	// marqueur neutre des chassis non resolus.
	Kind string `json:"kind,omitempty"`
	// En / Fr : le libelle de la famille, vide pour un nom propre du jeu.
	En string `json:"en,omitempty"`
	Fr string `json:"fr,omitempty"`
}
