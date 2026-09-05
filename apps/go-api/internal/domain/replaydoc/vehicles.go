package replaydoc

// vehicles.go — LES VEHICULES : la vie d'un chassis, ses positions, et qui l'occupe.

// VehicleTrack est LA VIE D UN VEHICULE, de sa naissance a la derniere preuve de sa presence.
type VehicleTrack struct {
	Slot    uint32          `json:"slot"`
	Gen     uint32          `json:"gen"`
	Chassis string          `json:"chassis,omitempty"`
	Family  string          `json:"family,omitempty"`
	T0      int             `json:"t0"`
	T1      int             `json:"t1"`
	T1Max   int             `json:"t1max"`
	End     string          `json:"end"`
	Spawn   *VehicleSpawn   `json:"spawn,omitempty"`
	Samples []VehicleSample `json:"samples,omitempty"`
	Rides   []VehicleRide   `json:"rides,omitempty"`
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
}

// VehicleAim est UNE lecture de visee d occupant, posee sur l axe de frames.
type VehicleAim struct {
	T int     `json:"t"`
	H float32 `json:"h,omitempty"`
	P float32 `json:"p,omitempty"`
}
