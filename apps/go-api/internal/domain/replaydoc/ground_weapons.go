package replaydoc

// ground_weapons.go — LES SOCLES D'ARME ET LES ARMES AU SOL : ou une arme reapparait, quand
// le socle se vide, et la vie des objets armes laisses sur la carte.

// WeaponPad est UN socle d'arme du match : une position où une arme de la même famille
// réapparaît, avec ses apparitions, ses intervalles de présence et son cycle quand il est établi.
type WeaponPad struct {
	X        float32       `json:"x"`
	Y        float32       `json:"y"`
	Z        float32       `json:"z,omitempty"`
	Weapon   string        `json:"weapon"`
	Spawns   []int         `json:"spawns"`
	Presence []PadPresence `json:"presence"`
	Cycle    *PadCycle     `json:"cycle,omitempty"`
}

// PadPresence est UNE occupation du socle : de l'apparition de l'arme à sa disparition, telle
// que le recensement des images-clés la BORNE.
type PadPresence struct {
	T0    int `json:"t0"`
	TLow  int `json:"tLow"`
	THigh int `json:"tHigh"`
}

// PadCycle est le délai de réapparition d'un socle, en secondes, mesuré du moment où il se vide
// à la réapparition suivante.
type PadCycle struct {
	MedianS float32 `json:"medianS"`
	P10S    float32 `json:"p10S"`
	P90S    float32 `json:"p90S"`
	Gaps    int     `json:"gaps"`
	Missing int     `json:"missing"`
}

// PadPickup est une occupation de socle qui S'EST ACHEVÉE : le socle s'est vidé quelque part
// dans [TLow, THigh].
type PadPickup struct {
	Pad   int     `json:"pad"`
	TLow  int     `json:"tLow"`
	THigh int     `json:"tHigh"`
	XUID  *string `json:"xuid"`
	T     *int    `json:"t,omitempty"`
}

// GroundWeapon est UN objet arme au sol, borné par l'observation.
type GroundWeapon struct {
	T0      int     `json:"t0"`
	T1      int     `json:"t1"`
	T1Max   int     `json:"t1max"`
	X       float32 `json:"x"`
	Y       float32 `json:"y"`
	Z       float32 `json:"z,omitempty"`
	W       string  `json:"w"`
	Origin  string  `json:"origin"`
	Dropper int     `json:"dropper"`
	End     string  `json:"end"`
	Picker  int     `json:"picker"`
}
