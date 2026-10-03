package replaydoc

// bursts.go — LES RAFALES DE TIR CONTINU (schema 71, lot M4b). Jumeau de `replay.FireBurst`.

// FireBurst est une rafale de tir continu publiee (cf. `replay.FireBurst`).
type FireBurst struct {
	T0         int             `json:"t0"`
	T1         int             `json:"t1"`
	Slot       uint32          `json:"slot"`
	Weapon     string          `json:"w"`
	Vehicle    *uint32         `json:"v,omitempty"`
	Rate       float64         `json:"rate"`
	Rate0      float64         `json:"rate0,omitempty"`
	Ramp       float64         `json:"ramp,omitempty"`
	Holes      []FireBurstHole `json:"holes,omitempty"`
	StartBound string          `json:"b0"`
	EndBound   string          `json:"b1"`
}

// FireBurstHole est un passage muet d une rafale, en frames [T0, T1).
type FireBurstHole struct {
	T0 int `json:"t0"`
	T1 int `json:"t1"`
}
