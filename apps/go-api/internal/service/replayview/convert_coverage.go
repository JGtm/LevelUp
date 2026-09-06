package replayview

// convert_coverage.go — couvertures des calques generaux. Jumeau de
// `domain/replaydoc/coverage.go`.

import (
	"levelup/go-api/internal/analysis/replay"
	"levelup/go-api/internal/domain/replaydoc"
)

func toCoverage(v replay.Coverage) replaydoc.Coverage {
	return replaydoc.Coverage{
		Shots:             toLayerCoverage(v.Shots),
		Grenades:          toLayerCoverage(v.Grenades),
		Objectives:        toLayerCoverage(v.Objectives),
		Equipment:         ptrOf(v.Equipment, toEquipmentCoverage),
		Grapple:           ptrOf(v.Grapple, toGrappleCoverage),
		Placements:        ptrOf(v.Placements, toEquipmentPlacementCoverage),
		GroundWeapons:     ptrOf(v.GroundWeapons, toGroundWeaponCoverage),
		Score:             ptrOf(v.Score, toScoreCoverage),
		FlagCarries:       ptrOf(v.FlagCarries, toFlagCarriesCoverage),
		VipCrown:          ptrOf(v.VipCrown, toVipCrownCoverage),
		SkullCarries:      ptrOf(v.SkullCarries, toSkullCarriesCoverage),
		BombCarries:       ptrOf(v.BombCarries, toBombCarriesCoverage),
		BombArmings:       ptrOf(v.BombArmings, toBombArmingsCoverage),
		WeaponChanges:     ptrOf(v.WeaponChanges, toWeaponChangeCoverage),
		Pickups:           ptrOf(v.Pickups, toPickupCoverage),
		PadDating:         ptrOf(v.PadDating, toPadDatingStats),
		EquipmentChanges:  ptrOf(v.EquipmentChanges, toEquipmentChangeCoverage),
		Translocations:    ptrOf(v.Translocations, toTranslocationCoverage),
		AbilityImpulses:   ptrOf(v.AbilityImpulses, toAbilityImpulseCoverage),
		AbilityCharges:    ptrOf(v.AbilityCharges, toAbilityChargeCoverage),
		GroundWeaponItems: ptrOf(v.GroundWeaponItems, toGroundWeaponItemsCoverage),
		Vehicles:          ptrOf(v.Vehicles, toVehicleCoverage),
		ObjectiveObjects:  ptrOf(v.ObjectiveObjects, toObjectiveObjectsCoverage),
		Inventory:         ptrOf(v.Inventory, toInventoryCoverage),
		GrenadeReads:      ptrOf(v.GrenadeReads, toGrenadeReadCoverage),
		Zones:             ptrOf(v.Zones, toZonesCoverage),
		OriginResolved:    v.OriginResolved,
		T0Film:            ptrOf(v.T0Film, toT0FilmCoverage),
		Verdict:           v.Verdict,
		Bridge:            toBridgeHealth(v.Bridge),
	}
}

func toLayerCoverage(v replay.LayerCoverage) replaydoc.LayerCoverage {
	return replaydoc.LayerCoverage{
		Available:   v.Available,
		Attached:    v.Attached,
		NoSlot:      v.NoSlot,
		Ambiguous:   v.Ambiguous,
		OutOfWindow: v.OutOfWindow,
		Unpublished: v.Unpublished,
	}
}

func toBridgeHealth(v replay.BridgeHealth) replaydoc.BridgeHealth {
	return replaydoc.BridgeHealth{
		Slots:              v.Slots,
		FromReading:        v.FromReading,
		LivesNamed:         v.LivesNamed,
		LivesTotal:         v.LivesTotal,
		IndexReadings:      v.IndexReadings,
		IndexDisagreements: v.IndexDisagreements,
		SlotCollisions:     v.SlotCollisions,
		ClosedByShot:       v.ClosedByShot,
		ClosedByRespawn:    v.ClosedByRespawn,
		ClosedContested:    v.ClosedContested,
		ClosedRefused:      v.ClosedRefused,
	}
}

func toT0FilmCoverage(v replay.T0FilmCoverage) replaydoc.T0FilmCoverage {
	return replaydoc.T0FilmCoverage{
		Detected: v.Detected,
		Reason:   v.Reason,
		Tracks:   v.Tracks,
		Moving:   v.Moving,
		Burst:    v.Burst,
		MarginMs: v.MarginMs,
	}
}

func toPadDatingStats(v replay.PadDatingStats) replaydoc.PadDatingStats {
	return replaydoc.PadDatingStats{
		Occupations:        v.Occupations,
		Dated:              v.Dated,
		Named:              v.Named,
		Ambiguous:          v.Ambiguous,
		Uncovered:          v.Uncovered,
		PowerupOccupations: v.PowerupOccupations,
	}
}

func toInventoryCoverage(v replay.InventoryCoverage) replaydoc.InventoryCoverage {
	return replaydoc.InventoryCoverage{
		Decoded:             v.Decoded,
		DroppedBeforeOrigin: v.DroppedBeforeOrigin,
		Unpublished:         v.Unpublished,
		Published:           v.Published,
	}
}

func toGrenadeReadCoverage(v replay.GrenadeReadCoverage) replaydoc.GrenadeReadCoverage {
	return replaydoc.GrenadeReadCoverage{
		FromKeyframe: v.FromKeyframe,
		FromDelta:    v.FromDelta,
		Unpublished:  v.Unpublished,
		AmmoRefused:  v.AmmoRefused,
	}
}

func toEquipmentCoverage(v replay.EquipmentCoverage) replaydoc.EquipmentCoverage {
	return replaydoc.EquipmentCoverage{
		TracksTotal:        v.TracksTotal,
		CamoLives:          v.CamoLives,
		CamoEpisodes:       v.CamoEpisodes,
		OvershieldLives:    v.OvershieldLives,
		OvershieldEpisodes: v.OvershieldEpisodes,
		KillsRead:          v.KillsRead,
	}
}

func toGrappleCoverage(v replay.GrappleCoverage) replaydoc.GrappleCoverage {
	return replaydoc.GrappleCoverage{
		LightReads:    v.LightReads,
		HeavyReads:    v.HeavyReads,
		Pulls:         v.Pulls,
		PullLives:     v.PullLives,
		UnpairedFires: v.UnpairedFires,
		BrokenBodies:  v.BrokenBodies,
	}
}

func toScoreCoverage(v replay.ScoreCoverage) replaydoc.ScoreCoverage {
	return replaydoc.ScoreCoverage{
		TeamIdentity:  v.TeamIdentity,
		Rounds:        v.Rounds,
		ModeSupported: v.ModeSupported,
		Truncated:     v.Truncated,
		Oracle:        v.Oracle,
		Points:        v.Points,
	}
}

func toWeaponChangeCoverage(v replay.WeaponChangeCoverage) replaydoc.WeaponChangeCoverage {
	return replaydoc.WeaponChangeCoverage{
		Decoded:      v.Decoded,
		Published:    v.Published,
		Restated:     v.Restated,
		BeforeOrigin: v.BeforeOrigin,
		Taken:        v.Taken,
		Dropped:      v.Dropped,
		Swapped:      v.Swapped,
	}
}

func toTranslocationCoverage(v replay.TranslocationCoverage) replaydoc.TranslocationCoverage {
	return replaydoc.TranslocationCoverage{
		Events:       v.Events,
		Published:    v.Published,
		BeforeOrigin: v.BeforeOrigin,
		Unpublished:  v.Unpublished,
		Positioned:   v.Positioned,
	}
}

func toAbilityImpulseCoverage(v replay.AbilityImpulseCoverage) replaydoc.AbilityImpulseCoverage {
	return replaydoc.AbilityImpulseCoverage{
		Reads:           v.Reads,
		Episodes:        v.Episodes,
		Published:       v.Published,
		BeforeOrigin:    v.BeforeOrigin,
		Unpublished:     v.Unpublished,
		NoIdentity:      v.NoIdentity,
		OtherFamily:     v.OtherFamily,
		NoResolver:      v.NoResolver,
		ComponentAbsent: v.ComponentAbsent,
	}
}

func toAbilityChargeCoverage(v replay.AbilityChargeCoverage) replaydoc.AbilityChargeCoverage {
	return replaydoc.AbilityChargeCoverage{
		Reads:           v.Reads,
		Published:       v.Published,
		BeforeOrigin:    v.BeforeOrigin,
		Unpublished:     v.Unpublished,
		NoIdentity:      v.NoIdentity,
		OtherFamily:     v.OtherFamily,
		NoResolver:      v.NoResolver,
		ComponentAbsent: v.ComponentAbsent,
	}
}

func toEquipmentChangeCoverage(v replay.EquipmentChangeCoverage) replaydoc.EquipmentChangeCoverage {
	return replaydoc.EquipmentChangeCoverage{
		Decoded:           v.Decoded,
		Published:         v.Published,
		Taken:             v.Taken,
		Spent:             v.Spent,
		Spawned:           v.Spawned,
		BeforeOrigin:      v.BeforeOrigin,
		Lives:             v.Lives,
		MissedEstimate:    v.MissedEstimate,
		CounterJumps:      v.CounterJumps,
		LivesFirstOffSpec: v.LivesFirstOffSpec,
		Repeats:           v.Repeats,
		Recovered:         v.Recovered,
	}
}
