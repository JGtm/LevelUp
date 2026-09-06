package replaydoc

// inventory.go — CE QUE CHAQUE JOUEUR PORTE ET CE QUI CHANGE DE MAIN : inventaire,
// capacite d'armure, grenades, equipement, ramassages et changements d'arme.

// Inventory est l'inventaire complet d'un slot à un instant d'image-clé.
type Inventory struct {
	T     int        `json:"t"`
	Slot  uint32     `json:"slot"`
	G     []uint32   `json:"g,omitempty"`
	Gs    *int       `json:"gs,omitempty"`
	D     *int       `json:"d,omitempty"`
	Am    []AmmoSlot `json:"am,omitempty"`
	Cand  int        `json:"cand,omitempty"`
	Empty string     `json:"empty,omitempty"`
}

// AmmoSlot est l'état de munitions d'un emplacement d'arme, dans l'ordre de Loadout.W.
type AmmoSlot struct {
	Mag   *uint32  `json:"mag,omitempty"`
	Res   *uint32  `json:"res,omitempty"`
	Gauge *float32 `json:"gauge,omitempty"`
}

// AbilityRead est UNE lecture de la capacité d'armure portée par un slot.
type AbilityRead struct {
	T    int    `json:"t"`
	Slot uint32 `json:"slot"`
	R    int    `json:"r"`
	Src  string `json:"src"`
}

// GrenadeRead est UNE lecture des grenades portées par un slot.
type GrenadeRead struct {
	T    int      `json:"t"`
	Slot uint32   `json:"slot"`
	G    []uint32 `json:"g"`
	Gs   *int     `json:"gs,omitempty"`
	Src  string   `json:"src"`
}

// AbilityImpulse est UNE impulsion de capacité publiée : qui, quand, et de quel équipement.
type AbilityImpulse struct {
	T      int    `json:"t"`
	Slot   uint32 `json:"slot"`
	Family string `json:"family"`
}

// AbilityCharge est UNE lecture de charge publiée : qui, quand, quel équipement, et ce
// qu'il en reste.
type AbilityCharge struct {
	T       int    `json:"t"`
	Slot    uint32 `json:"slot"`
	Family  string `json:"family"`
	Charges int    `json:"charges"`
}

// WeaponChange est UN changement d'arme en main.
type WeaponChange struct {
	T    int              `json:"t"`
	Slot uint32           `json:"slot"`
	Kind WeaponChangeKind `json:"kind"`
	W    string           `json:"w,omitempty"`
	From string           `json:"from,omitempty"`
}

// WeaponChangeKind qualifie un changement d'arme en main, tel que le document le publie.
type WeaponChangeKind string

// EquipmentChange est UN changement d'équipement porté.
type EquipmentChange struct {
	T         int                 `json:"t"`
	Slot      uint32              `json:"slot"`
	Kind      EquipmentChangeKind `json:"kind"`
	R         int                 `json:"r"`
	From      int                 `json:"from"`
	Recovered bool                `json:"recovered,omitempty"`
	Gap       int                 `json:"gap,omitempty"`
}

// EquipmentChangeKind qualifie un changement d'équipement, tel que le document le publie.
type EquipmentChangeKind string

// EquipmentEpisode est un épisode daté d'état ACTIF d'un équipement, porté par une vie.
type EquipmentEpisode struct {
	Slot    uint32 `json:"slot"`
	Fam     string `json:"fam"`
	T0      int    `json:"t0"`
	T1      int    `json:"t1"`
	EndRead bool   `json:"endRead,omitempty"`
	K       int    `json:"k,omitempty"`
	A       int    `json:"a,omitempty"`
}

// EquipmentPlacement est UNE pose d'équipement, datée et située.
type EquipmentPlacement struct {
	T0       int      `json:"t0"`
	T1       int      `json:"t1"`
	X        float32  `json:"x"`
	Y        float32  `json:"y"`
	Z        float32  `json:"z,omitempty"`
	Family   string   `json:"family"`
	ID       string   `json:"id"`
	Owner    int      `json:"owner"`
	H        *float32 `json:"h,omitempty"`
	Origin   string   `json:"origin,omitempty"`
	Until    int      `json:"until,omitempty"`
	UntilMax int      `json:"untilMax,omitempty"`
	End      string   `json:"end,omitempty"`
}

// GrappleLine est UNE traction de grappin : la fenêtre datée et le point d'accroche.
type GrappleLine struct {
	Slot uint32  `json:"slot"`
	T0   int     `json:"t0"`
	T1   int     `json:"t1"`
	AX   float32 `json:"ax"`
	AY   float32 `json:"ay"`
	AZ   float32 `json:"az,omitempty"`
}

// Translocation est UNE téléportation exécutée, sur l'axe de frames du document.
type Translocation struct {
	T    int      `json:"t"`
	Slot uint32   `json:"slot"`
	FX   *float32 `json:"fx,omitempty"`
	FY   *float32 `json:"fy,omitempty"`
	FZ   *float32 `json:"fz,omitempty"`
	TX   *float32 `json:"tx,omitempty"`
	TY   *float32 `json:"ty,omitempty"`
	TZ   *float32 `json:"tz,omitempty"`
}

// Pickup est UN ramassage : quand, qui, quoi.
type Pickup struct {
	T      int        `json:"t"`
	Slot   uint32     `json:"slot"`
	XUID   string     `json:"xuid,omitempty"`
	W      string     `json:"w"`
	Family string     `json:"family,omitempty"`
	Kind   PickupKind `json:"kind"`
	Class  int        `json:"class"`
	Origin string     `json:"origin,omitempty"`
}

// PickupKind qualifie ce qui a été ramassé.
type PickupKind string
