package replaydoc

// vehicles.go — LES VEHICULES : la vie d'un chassis, ses positions, et qui l'occupe.

// VehicleTrack est LA VIE D UN VEHICULE, de sa naissance a la derniere preuve de sa presence.
type VehicleTrack struct {
	Slot    uint32 `json:"slot"`
	Gen     uint32 `json:"gen"`
	Chassis string `json:"chassis,omitempty"`
	Family  string `json:"family,omitempty"`
	T0      int    `json:"t0"`
	T1      int    `json:"t1"`
	T1Max   int    `json:"t1max"`
	End     string `json:"end"`
	// TEnd est la frame de la fin ECRITE (composant dead-state), presente pour le seul
	// `end == "destroyed"`. Absente = le film n ecrit pas la mort de cette vie.
	TEnd    *int            `json:"tEnd,omitempty"`
	Spawn   *VehicleSpawn   `json:"spawn,omitempty"`
	Samples []VehicleSample `json:"samples,omitempty"`
	Rides   []VehicleRide   `json:"rides,omitempty"`
	// Part / Carrier / Variant : la piece montee, son porteur et la variante du chassis (schema
	// 69, cf. `replay.VehicleTrack`).
	Part    string          `json:"part,omitempty"`
	Carrier *VehicleLifeRef `json:"carrier,omitempty"`
	Variant string          `json:"variant,omitempty"`
}

// VehicleLifeRef designe une vie de vehicule par `(slot, gen)` (schema 69).
type VehicleLifeRef struct {
	Slot uint32 `json:"slot"`
	Gen  uint32 `json:"gen"`
}

// VehicleWeapon est l entree du registre des armes de vehicule (schema 69, cf.
// `replay.VehicleWeapon`), resolue a la requete.
type VehicleWeapon struct {
	Vehicle string              `json:"vehicle"`
	En      string              `json:"en"`
	Fr      string              `json:"fr"`
	Fire    string              `json:"fire"`
	Fx      string              `json:"fx"`
	Tint    string              `json:"tint"`
	Sound   string              `json:"sound,omitempty"`
	Loop    string              `json:"loop,omitempty"`
	Mount   *VehicleWeaponMount `json:"mount,omitempty"`
}

// VehicleWeaponMount est l ancre d une arme sur le sprite de son vehicule (schema 69).
type VehicleWeaponMount struct {
	Aim string  `json:"aim"`
	AX  float64 `json:"ax"`
	AY  float64 `json:"ay"`
}

// VehicleSpawn est la naissance d un vehicule : ou, et sous quel cap.
type VehicleSpawn struct {
	X float32  `json:"x"`
	Y float32  `json:"y"`
	Z float32  `json:"z,omitempty"`
	H *float32 `json:"h,omitempty"`
}

// VehicleSample est une position du vehicule sur l axe de frames.
type VehicleSample struct {
	T int     `json:"t"`
	X float32 `json:"x"`
	Y float32 `json:"y"`
	Z float32 `json:"z,omitempty"`
	// G : la lacune de replication qui precede cet echantillon, en ms (schema 69) — meme
	// semantique que `Point.G` : le client TIENT la derniere position au travers.
	G int     `json:"g,omitempty"`
	H float32 `json:"h,omitempty"`
}

// VehicleRide est un EPISODE D OCCUPATION : un joueur a bord de ce vehicule, de `T0` a `T1`.
type VehicleRide struct {
	T0   int          `json:"t0"`
	T1   int          `json:"t1"`
	Slot uint32       `json:"slot"`
	XUID string       `json:"xuid,omitempty"`
	Seat *int         `json:"seat,omitempty"`
	Src  string       `json:"src"`
	Aim  []VehicleAim `json:"aim,omitempty"`
	// Turret : la tourelle d ou l episode a ete reporte (schema 69, cf. `replay.VehicleRide`).
	Turret *VehicleLifeRef `json:"turret,omitempty"`
}

// VehicleAim est UNE lecture de visee d occupant, posee sur l axe de frames.
type VehicleAim struct {
	T int     `json:"t"`
	H float32 `json:"h,omitempty"`
	P float32 `json:"p,omitempty"`
}

// VehicleCycle est LE CYCLE DE REAPPARITION d un emplacement de naissance de vehicule (schema
// 63) : le delai, en secondes, entre la destruction d un vehicule et la naissance du suivant au
// meme endroit. Meme forme et meme juge que `PadCycle` — cle absente quand le cycle n est pas
// ETABLI.
type VehicleCycle struct {
	X       float32 `json:"x"`
	Y       float32 `json:"y"`
	Family  string  `json:"family,omitempty"`
	MedianS float32 `json:"medianS"`
	P10S    float32 `json:"p10S"`
	P90S    float32 `json:"p90S"`
	Gaps    int     `json:"gaps"`
	Missing int     `json:"missing"`
}

// VehicleScenery est le verdict de decor des vies de vehicule posees par la carte hors de la
// zone jouable (lot M7, cf. `replay.VehicleScenery`), resolu a la requete.
type VehicleScenery struct {
	Zone        string               `json:"zone"`
	Floor       string               `json:"floor"`
	Candidates  int                  `json:"candidates"`
	InPlayArea  int                  `json:"inPlayArea"`
	ZoneUnknown int                  `json:"zoneUnknown"`
	Hidden      []VehicleSceneryLife `json:"hidden,omitempty"`
}

// VehicleSceneryLife designe une vie de decor et la raison de son masquage.
type VehicleSceneryLife struct {
	Slot   uint32 `json:"slot"`
	Gen    uint32 `json:"gen"`
	Reason string `json:"reason"`
}
