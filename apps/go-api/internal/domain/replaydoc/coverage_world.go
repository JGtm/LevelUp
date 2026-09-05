package replaydoc

// coverage_world.go — LES COUVERTURES DES CALQUES DU MONDE : poses d'equipement, socles et
// armes au sol, ramassages, vehicules.

// EquipmentPlacementCoverage dit ce que le calque a lu et ce qu'il en a publié.
type EquipmentPlacementCoverage struct {
	Scanned        bool           `json:"scanned"`
	Widths         string         `json:"widths,omitempty"`
	Calibrated     bool           `json:"calibrated"`
	Lives          int            `json:"lives"`
	Anchors        int            `json:"anchors"`
	Confirmed      int            `json:"confirmed"`
	Placements     int            `json:"placements"`
	Named          int            `json:"named"`
	Other          int            `json:"other"`
	WithOwner      int            `json:"withOwner"`
	WithHeading    int            `json:"withHeading"`
	ByFamily       map[string]int `json:"byFamily,omitempty"`
	Deployed       int            `json:"deployed"`
	Dropped        int            `json:"dropped"`
	Unknown        int            `json:"unknown"`
	EndSeen        int            `json:"endSeen"`
	EndOpen        int            `json:"endOpen"`
	ByFamilyOrigin map[string]int `json:"byFamilyOrigin,omitempty"`
}

// GroundWeaponCoverage dit ce que le calque a lu, ce qu'il a retenu, et ce qu'il a écarté.
type GroundWeaponCoverage struct {
	Scanned         bool `json:"scanned"`
	Slots           int  `json:"slots"`
	Anchors         int  `json:"anchors"`
	Accepted        int  `json:"accepted"`
	Kept            int  `json:"kept"`
	Rejected        int  `json:"rejected"`
	Objectives      int  `json:"objectives"`
	Dropped         int  `json:"dropped"`
	Spawned         int  `json:"spawned"`
	AtRest          int  `json:"atRest"`
	Clusters        int  `json:"clusters"`
	Pads            int  `json:"pads"`
	Occupancies     int  `json:"occupancies"`
	Dated           int  `json:"dated"`
	Unknown         int  `json:"unknown"`
	Never           int  `json:"never"`
	Cycles          int  `json:"cycles"`
	PowerupScanned  bool `json:"powerupScanned"`
	PowerupAccepted int  `json:"powerupAccepted"`
	PowerupKept     int  `json:"powerupKept"`
	PowerupPads     int  `json:"powerupPads"`
}

// GroundWeaponItemsCoverage dit ce que le calque a vu, lié, et refusé de dire.
type GroundWeaponItemsCoverage struct {
	Objects      int `json:"objects"`
	Published    int `json:"published"`
	AtRest       int `json:"atRest"`
	DropperNamed int `json:"dropperNamed"`
	TakesTotal   int `json:"takesTotal"`
	PickupLinked int `json:"pickupLinked"`
	EndPickup    int `json:"endPickup"`
	EndSeen      int `json:"endSeen"`
	EndOpen      int `json:"endOpen"`
}

// PickupCoverage dit ce que le canal a vu, ce qu'il a écarté et ce qu'il ne PEUT PAS voir.
type PickupCoverage struct {
	Decoded            int            `json:"decoded"`
	Published          int            `json:"published"`
	Named              int            `json:"named"`
	Weapons            int            `json:"weapons"`
	Items              int            `json:"items"`
	UnknownFamilies    int            `json:"unknownFamilies"`
	BeforeOrigin       int            `json:"beforeOrigin"`
	MultiEvent         int            `json:"multiEvent"`
	Refused            int            `json:"refused"`
	OriginSpawner      int            `json:"originSpawner"`
	OriginGround       int            `json:"originGround"`
	OriginUnknown      int            `json:"originUnknown"`
	SpawnPointsState   string         `json:"spawnPointsState"`
	MapCatalogPoints   int            `json:"mapCatalogPoints"`
	SpawnerByPointKind map[string]int `json:"spawnerByPointKind,omitempty"`
}

// VehicleCoverage dit ce que le calque a vu, resolu, et refuse de dire.
type VehicleCoverage struct {
	Scanned            bool           `json:"scanned"`
	Lives              int            `json:"lives"`
	Published          int            `json:"published"`
	NoPosition         int            `json:"noPosition"`
	Merged             int            `json:"merged"`
	WithSpawn          int            `json:"withSpawn"`
	WithChassis        int            `json:"withChassis"`
	FamilyResolved     int            `json:"familyResolved"`
	FamilyUnknown      int            `json:"familyUnknown"`
	UnknownChassis     map[string]int `json:"unknownChassis,omitempty"`
	Samples            int            `json:"samples"`
	WithHeading        int            `json:"withHeading"`
	Rides              int            `json:"rides"`
	VehiclesRidden     int            `json:"vehiclesRidden"`
	RidesNamed         int            `json:"ridesNamed"`
	RidesFromEvent     int            `json:"ridesFromEvent"`
	RidesMixed         int            `json:"ridesMixed"`
	RidesFromGap       int            `json:"ridesFromGap"`
	RidesWithSeat      int            `json:"ridesWithSeat"`
	AimReads           int            `json:"aimReads"`
	RidesWithAim       int            `json:"ridesWithAim"`
	AimSamples         int            `json:"aimSamples"`
	AimRideFrames      int            `json:"aimRideFrames"`
	Ambiguous          int            `json:"ambiguous"`
	Shots              int            `json:"shots"`
	ShotsAmbiguous     int            `json:"shotsAmbiguous"`
	ShotsUnplaced      int            `json:"shotsUnplaced"`
	ShotsNoRide        int            `json:"shotsNoRide"`
	ShotsVehicleWeapon int            `json:"shotsVehicleWeapon"`
}
