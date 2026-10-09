package replayview

// convert_tir_continu.go — les rafales de tir continu et leur couverture (schema 71, lot M4b).
// Jumeau de `domain/replaydoc/bursts.go` et `coverage_tir_continu.go`.

import (
	"levelup/go-api/internal/domain/replaydoc"
	"levelup/go-api/internal/games/halo_infinite/film/replay"
)

func toFireBurst(v replay.FireBurst) replaydoc.FireBurst {
	return replaydoc.FireBurst{
		T0: v.T0, T1: v.T1, Slot: v.Slot, Weapon: v.Weapon, Vehicle: v.Vehicle, Rate: v.Rate,
		Rate0: v.Rate0, Ramp: v.Ramp, Holes: sliceOf(v.Holes, toFireBurstHole),
		StartBound: v.StartBound, EndBound: v.EndBound,
	}
}

func toFireBurstHole(v replay.FireBurstHole) replaydoc.FireBurstHole {
	return replaydoc.FireBurstHole{T0: v.T0, T1: v.T1}
}

// toContinuousFireCoverage : les champs portent les memes noms des deux cotes.
func toContinuousFireCoverage(v replay.ContinuousFireCoverage) replaydoc.ContinuousFireCoverage {
	return replaydoc.ContinuousFireCoverage(v)
}
