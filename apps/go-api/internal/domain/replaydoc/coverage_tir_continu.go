package replaydoc

// coverage_tir_continu.go — la couverture du TIR CONTINU (schema 71, lot M4b). Jumeau de
// `replay.ContinuousFireCoverage`.

// ContinuousFireCoverage est la couverture du tir continu (cf. `replay.ContinuousFireCoverage`).
type ContinuousFireCoverage struct {
	Packets         int   `json:"packets"`
	Reached         int   `json:"reached"`
	Closed          int   `json:"closed"`
	Holes           int   `json:"holes"`
	HoleRuns        int   `json:"holeRuns"`
	HolesUnlocated  int   `json:"holesUnlocated"`
	HolesOpenViewB  int   `json:"holesOpenViewB"`
	HolesOverflow   int   `json:"holesOverflow"`
	HolesKind       int   `json:"holesKind"`
	HolesBlockBC    int   `json:"holesBlockBC"`
	HolesCap        int   `json:"holesCap"`
	HolesNotClosing int   `json:"holesNotClosing"`
	Entries         int   `json:"entries"`
	WithAction      int   `json:"withAction"`
	Firing          int   `json:"firing"`
	BurstsRead      int   `json:"burstsRead"`
	BurstsWithHole  int   `json:"burstsWithHole"`
	InnerHoles      int   `json:"innerHoles"`
	HeldHoleMS      int64 `json:"heldHoleMs"`
	Published       int   `json:"published"`
	OnVehicle       int   `json:"onVehicle"`
	OnFoot          int   `json:"onFoot"`
	OtherInput      int   `json:"otherInput"`
	NoPlayer        int   `json:"noPlayer"`
	Ambiguous       int   `json:"ambiguous"`
	VehicleNoWeapon int   `json:"vehicleNoWeapon"`
	NoTrack         int   `json:"noTrack"`
	WeaponUnknown   int   `json:"weaponUnknown"`
	NotContinuous   int   `json:"notContinuous"`
	Empty           int   `json:"empty"`
	ByPlace         int   `json:"byPlace"`
	ClippedToMount  int   `json:"clippedToMount"`
	Shots           int   `json:"shots"`
}
