package replayview

// convert_coverage_world.go — couvertures des calques du monde. Jumeau de
// `domain/replaydoc/coverage_world.go`.

import (
	"levelup/go-api/internal/domain/replaydoc"
	"levelup/go-api/internal/games/halo_infinite/film/replay"
)

func toEquipmentPlacementCoverage(v replay.EquipmentPlacementCoverage) replaydoc.EquipmentPlacementCoverage {
	return replaydoc.EquipmentPlacementCoverage{
		Scanned:        v.Scanned,
		Widths:         v.Widths,
		Calibrated:     v.Calibrated,
		Lives:          v.Lives,
		Anchors:        v.Anchors,
		Confirmed:      v.Confirmed,
		Placements:     v.Placements,
		Named:          v.Named,
		Other:          v.Other,
		WithOwner:      v.WithOwner,
		WithHeading:    v.WithHeading,
		ByFamily:       v.ByFamily,
		Deployed:       v.Deployed,
		Dropped:        v.Dropped,
		Unknown:        v.Unknown,
		EndSeen:        v.EndSeen,
		EndOpen:        v.EndOpen,
		ByFamilyOrigin: v.ByFamilyOrigin,
		SpawnEvents:    v.SpawnEvents,
		SpawnLists:     v.SpawnLists,
		ByCause:        v.ByCause,
	}
}

func toGroundWeaponCoverage(v replay.GroundWeaponCoverage) replaydoc.GroundWeaponCoverage {
	return replaydoc.GroundWeaponCoverage{
		Scanned:         v.Scanned,
		Slots:           v.Slots,
		Anchors:         v.Anchors,
		Accepted:        v.Accepted,
		Kept:            v.Kept,
		Rejected:        v.Rejected,
		Objectives:      v.Objectives,
		Dropped:         v.Dropped,
		Spawned:         v.Spawned,
		AtRest:          v.AtRest,
		Clusters:        v.Clusters,
		Pads:            v.Pads,
		Occupancies:     v.Occupancies,
		Dated:           v.Dated,
		Unknown:         v.Unknown,
		Never:           v.Never,
		Cycles:          v.Cycles,
		PowerupScanned:  v.PowerupScanned,
		PowerupAccepted: v.PowerupAccepted,
		PowerupKept:     v.PowerupKept,
		PowerupPads:     v.PowerupPads,
	}
}

func toGroundWeaponItemsCoverage(v replay.GroundWeaponItemsCoverage) replaydoc.GroundWeaponItemsCoverage {
	return replaydoc.GroundWeaponItemsCoverage{
		Objects:      v.Objects,
		Published:    v.Published,
		AtRest:       v.AtRest,
		DropperNamed: v.DropperNamed,
		TakesTotal:   v.TakesTotal,
		PickupLinked: v.PickupLinked,
		EndPickup:    v.EndPickup,
		EndSeen:      v.EndSeen,
		EndOpen:      v.EndOpen,
		AmmoRead:     v.AmmoRead,
	}
}

func toPickupCoverage(v replay.PickupCoverage) replaydoc.PickupCoverage {
	return replaydoc.PickupCoverage{
		Decoded:            v.Decoded,
		Published:          v.Published,
		Named:              v.Named,
		Weapons:            v.Weapons,
		Items:              v.Items,
		UnknownFamilies:    v.UnknownFamilies,
		UnarmedGrants:      v.UnarmedGrants,
		BeforeOrigin:       v.BeforeOrigin,
		MultiEvent:         v.MultiEvent,
		Refused:            v.Refused,
		OriginSpawner:      v.OriginSpawner,
		OriginGround:       v.OriginGround,
		OriginUnknown:      v.OriginUnknown,
		SpawnPointsState:   v.SpawnPointsState,
		MapCatalogPoints:   v.MapCatalogPoints,
		SpawnerByPointKind: v.SpawnerByPointKind,
	}
}

func toVehicleCoverage(v replay.VehicleCoverage) replaydoc.VehicleCoverage {
	return replaydoc.VehicleCoverage{
		Scanned:    v.Scanned,
		Lives:      v.Lives,
		Published:  v.Published,
		NoPosition: v.NoPosition,
		// La porte des positions (schema 69, lot M1 des retours du rejeu).
		EchantillonsHorsEmprise:         v.EchantillonsHorsEmprise,
		SpawnsHorsEmprise:               v.SpawnsHorsEmprise,
		EchantillonsAuTraversDUnSilence: v.EchantillonsAuTraversDUnSilence,
		SilencesNonTranches:             v.SilencesNonTranches,
		Merged:                          v.Merged,
		WithSpawn:                       v.WithSpawn,
		WithChassis:                     v.WithChassis,
		FamilyResolved:                  v.FamilyResolved,
		FamilyUnknown:                   v.FamilyUnknown,
		UnknownChassis:                  v.UnknownChassis,
		Samples:                         v.Samples,
		WithHeading:                     v.WithHeading,
		DeathsRead:                      v.DeathsRead,
		DeathsMatched:                   v.DeathsMatched,
		DeathsUnmatched:                 v.DeathsUnmatched,
		DeathsTailDesync:                v.DeathsTailDesync,
		EndDestroyed:                    v.EndDestroyed,
		EndFilmEnd:                      v.EndFilmEnd,
		EndUnknown:                      v.EndUnknown,
		SamplesAfterEnd:                 v.SamplesAfterEnd,
		Rides:                           v.Rides,
		VehiclesRidden:                  v.VehiclesRidden,
		RidesNamed:                      v.RidesNamed,
		RidesRead:                       v.RidesRead,
		RidesProximity:                  v.RidesProximity,
		RidesWithSeat:                   v.RidesWithSeat,
		AimReads:                        v.AimReads,
		RidesWithAim:                    v.RidesWithAim,
		AimSamples:                      v.AimSamples,
		AimRideFrames:                   v.AimRideFrames,
		Ambiguous:                       v.Ambiguous,
		Shots:                           v.Shots,
		ShotsAmbiguous:                  v.ShotsAmbiguous,
		ShotsUnplaced:                   v.ShotsUnplaced,
		ShotsNoRide:                     v.ShotsNoRide,
		ShotsVehicleWeapon:              v.ShotsVehicleWeapon,
		Turrets:                         v.Turrets,
		TurretsOnCarrier:                v.TurretsOnCarrier,
		TurretRides:                     v.TurretRides,
		TurretRidesDropped:              v.TurretRidesDropped,
		ShotsOnCarrier:                  v.ShotsOnCarrier,
		Variants:                        v.Variants,

		// Les trois refus ventiles et le temoin de naissance (revue adverse du lot M4a).
		TurretCarrierBirthMismatch: v.TurretCarrierBirthMismatch,
		TurretRidesNotRideable:     v.TurretRidesNotRideable,
		TurretRidesOutOfWindow:     v.TurretRidesOutOfWindow,
		TurretRidesAlreadyAboard:   v.TurretRidesAlreadyAboard,

		CycleLocations: v.CycleLocations,
		Cycles:         v.Cycles,
		CycleGaps:      v.CycleGaps,
		CycleMissing:   v.CycleMissing,
	}
}
