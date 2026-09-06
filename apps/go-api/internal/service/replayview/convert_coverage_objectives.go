package replayview

// convert_coverage_objectives.go — couvertures des calques d'objectif. Jumeau de
// `domain/replaydoc/coverage_objectives.go`.

import (
	"levelup/go-api/internal/analysis/replay"
	"levelup/go-api/internal/domain/replaydoc"
)

func toFlagCarriesCoverage(v replay.FlagCarriesCoverage) replaydoc.FlagCarriesCoverage {
	return replaydoc.FlagCarriesCoverage{
		FlagFilm:              v.FlagFilm,
		Bursts:                v.Bursts,
		Captures:              v.Captures,
		Steals:                v.Steals,
		Openings:              v.Openings,
		Carries:               v.Carries,
		Closed:                v.Closed,
		Open:                  v.Open,
		NoBridge:              v.NoBridge,
		NoTrack:               v.NoTrack,
		OutOfWindow:           v.OutOfWindow,
		MarkerObserved:        v.MarkerObserved,
		MarkerConfirmed:       v.MarkerConfirmed,
		OpenObserved:          v.OpenObserved,
		OpenConfirmed:         v.OpenConfirmed,
		Overlaps:              v.Overlaps,
		ClosedOverlaps:        v.ClosedOverlaps,
		AmbiguousCarrierKills: v.AmbiguousCarrierKills,
		AmbiguousReturns:      v.AmbiguousReturns,
		HomeByObject:          v.HomeByObject,
		AmbiguousHomecomings:  v.AmbiguousHomecomings,
		NeutralFlag:           v.NeutralFlag,
		NeutralBirths:         v.NeutralBirths,
		TeamBirths:            v.TeamBirths,
		Spawns:                v.Spawns,
		ObjectLives:           v.ObjectLives,
		ClosedByObject:        v.ClosedByObject,
		DropsRepositioned:     v.DropsRepositioned,
	}
}

func toVipCrownCoverage(v replay.VipCrownCoverage) replaydoc.VipCrownCoverage {
	return replaydoc.VipCrownCoverage{
		VipFilm:           v.VipFilm,
		Selections:        v.Selections,
		Periods:           v.Periods,
		Closed:            v.Closed,
		Open:              v.Open,
		ClosedByDeath:     v.ClosedByDeath,
		ClosedBySelection: v.ClosedBySelection,
		NoBridge:          v.NoBridge,
		OutOfWindow:       v.OutOfWindow,
	}
}

func toSkullCarriesCoverage(v replay.SkullCarriesCoverage) replaydoc.SkullCarriesCoverage {
	return replaydoc.SkullCarriesCoverage{
		SkullFilm:     v.SkullFilm,
		Grabs:         v.Grabs,
		Trains:        v.Trains,
		Carries:       v.Carries,
		Closed:        v.Closed,
		Open:          v.Open,
		NoBridge:      v.NoBridge,
		OutOfWindow:   v.OutOfWindow,
		CarrierAbsent: v.CarrierAbsent,
	}
}

func toBombCarriesCoverage(v replay.BombCarriesCoverage) replaydoc.BombCarriesCoverage {
	return replaydoc.BombCarriesCoverage{
		BombFilm:      v.BombFilm,
		Events:        v.Events,
		Periods:       v.Periods,
		Carries:       v.Carries,
		Closed:        v.Closed,
		Open:          v.Open,
		ByDeath:       v.ByDeath,
		NoBridge:      v.NoBridge,
		OutOfWindow:   v.OutOfWindow,
		CarrierAbsent: v.CarrierAbsent,
	}
}

func toBombArmingsCoverage(v replay.BombArmingsCoverage) replaydoc.BombArmingsCoverage {
	return replaydoc.BombArmingsCoverage{
		Scanned:            v.Scanned,
		Reads:              v.Reads,
		Rises:              v.Rises,
		BelowFull:          v.BelowFull,
		Armed:              v.Armed,
		PairMerged:         v.PairMerged,
		Published:          v.Published,
		OutOfWindow:        v.OutOfWindow,
		Detonations:        v.Detonations,
		DetonationsCovered: v.DetonationsCovered,
		Suppressed:         v.Suppressed,
	}
}

func toBombStatsCoverage(v replay.BombStatsCoverage) replaydoc.BombStatsCoverage {
	return replaydoc.BombStatsCoverage{
		DetonationsRead:      v.DetonationsRead,
		CarryRead:            v.CarryRead,
		KillsRead:            v.KillsRead,
		ArmingsRead:          v.ArmingsRead,
		Detonations:          v.Detonations,
		Armings:              v.Armings,
		ArmingsAttributed:    v.ArmingsAttributed,
		ArmingsByDrop:        v.ArmingsByDrop,
		ArmingsByActiveCarry: v.ArmingsByActiveCarry,
		ArmingsNoCarrier:     v.ArmingsNoCarrier,
		ArmingsNoBridge:      v.ArmingsNoBridge,
		ArmingsAmbiguous:     v.ArmingsAmbiguous,
		Periods:              v.Periods,
		PeriodsNoBridge:      v.PeriodsNoBridge,
		PeriodsOpen:          v.PeriodsOpen,
		PeriodsByDeath:       v.PeriodsByDeath,
		Kills:                v.Kills,
		KillsOnCarrier:       v.KillsOnCarrier,
		Players:              v.Players,
	}
}

func toObjectiveObjectsCoverage(v replay.ObjectiveObjectsCoverage) replaydoc.ObjectiveObjectsCoverage {
	return replaydoc.ObjectiveObjectsCoverage{
		Scanned:    v.Scanned,
		Declared:   v.Declared,
		Lives:      v.Lives,
		Points:     v.Points,
		Motionless: v.Motionless,
		OutOfAxis:  v.OutOfAxis,
	}
}

func toZonesCoverage(v replay.ZonesCoverage) replaydoc.ZonesCoverage {
	return replaydoc.ZonesCoverage{
		Method:        v.Method,
		Roles:         v.Roles,
		Catalog:       v.Catalog,
		Slots:         v.Slots,
		Paired:        v.Paired,
		Unpaired:      v.Unpaired,
		Captures:      v.Captures,
		Attributed:    v.Attributed,
		OwnerChecked:  v.OwnerChecked,
		OwnerAgreed:   v.OwnerAgreed,
		OwnerUnpaired: v.OwnerUnpaired,
		Spans:         v.Spans,
		HillPeriods:   v.HillPeriods,
		UnknownOwner:  v.UnknownOwner,
		Letters:       v.Letters,
		GaugePoints:   v.GaugePoints,
	}
}
