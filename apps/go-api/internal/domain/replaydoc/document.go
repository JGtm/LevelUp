package replaydoc

// document.go — LA RACINE DU DOCUMENT SERVI et les types qu'un client dessine sans
// contexte : trajectoires, tirs, grenades, projectiles, libelles.
//
// `ReplayDocument` est le corps de `GET /players/{slug}/matches/{id}/replay`. Son champ
// `schemaVersion` porte la version STOCKEE de l'artefact lu (pas ContractVersion) : c'est
// elle qui dit au parc « a re-cuire », et le client la lit telle quelle.

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
	T  int      `json:"t"`
	X  float32  `json:"x"`
	Y  float32  `json:"y"`
	Z  float32  `json:"z,omitempty"`
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
	Bot       bool   `json:"bot,omitempty"`
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
}

// WeaponLabel est le libellé d'une arme, plus l'EFFET de rendu de ses tirs.
type WeaponLabel struct {
	En     string `json:"en"`
	Fr     string `json:"fr"`
	Fx     string `json:"fx,omitempty"`
	Key    string `json:"key,omitempty"`
	Tint   string `json:"tint,omitempty"`
	Img    string `json:"img,omitempty"`
	Tinted bool   `json:"tinted,omitempty"`
}

// VehicleLabel est ce qu il faut pour DESSINER une famille de chassis : sa vignette, et le fait
// qu elle se teigne.
type VehicleLabel struct {
	Img    string `json:"img,omitempty"`
	Tinted bool   `json:"tinted,omitempty"`
}
