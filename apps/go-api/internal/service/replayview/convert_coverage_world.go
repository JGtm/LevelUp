package replayview

// convert_coverage_world.go — couvertures des calques du monde. Jumeau de
// `domain/replaydoc/coverage_world.go`.

import (
	"levelup/go-api/internal/analysis/replay"
	"levelup/go-api/internal/domain/replaydoc"
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
		Scanned:            v.Scanned,
		Lives:              v.Lives,
		Published:          v.Published,
		NoPosition:         v.NoPosition,
		Merged:             v.Merged,
		WithSpawn:          v.WithSpawn,
		WithChassis:        v.WithChassis,
		FamilyResolved:     v.FamilyResolved,
		FamilyUnknown:      v.FamilyUnknown,
		UnknownChassis:     v.UnknownChassis,
		Samples:            v.Samples,
		WithHeading:        v.WithHeading,
		Rides:              v.Rides,
		VehiclesRidden:     v.VehiclesRidden,
		RidesNamed:         v.RidesNamed,
		RidesFromEvent:     v.RidesFromEvent,
		RidesMixed:         v.RidesMixed,
		RidesFromGap:       v.RidesFromGap,
		RidesWithSeat:      v.RidesWithSeat,
		AimReads:           v.AimReads,
		RidesWithAim:       v.RidesWithAim,
		AimSamples:         v.AimSamples,
		AimRideFrames:      v.AimRideFrames,
		Ambiguous:          v.Ambiguous,
		Shots:              v.Shots,
		ShotsAmbiguous:     v.ShotsAmbiguous,
		ShotsUnplaced:      v.ShotsUnplaced,
		ShotsNoRide:        v.ShotsNoRide,
		ShotsVehicleWeapon: v.ShotsVehicleWeapon,
	}
}
