package replayview

// convert_ground_weapons.go — socles d'arme et armes au sol. Jumeau de
// `domain/replaydoc/ground_weapons.go`.

import (
	"levelup/go-api/internal/analysis/replay"
	"levelup/go-api/internal/domain/replaydoc"
)

func toWeaponPad(v replay.WeaponPad) replaydoc.WeaponPad {
	return replaydoc.WeaponPad{
		X:        v.X,
		Y:        v.Y,
		Z:        v.Z,
		Weapon:   v.Weapon,
		Spawns:   v.Spawns,
		Presence: sliceOf(v.Presence, toPadPresence),
		Cycle:    ptrOf(v.Cycle, toPadCycle),
	}
}

func toPadPresence(v replay.PadPresence) replaydoc.PadPresence {
	return replaydoc.PadPresence{
		T0:    v.T0,
		TLow:  v.TLow,
		THigh: v.THigh,
	}
}

func toPadCycle(v replay.PadCycle) replaydoc.PadCycle {
	return replaydoc.PadCycle{
		MedianS: v.MedianS,
		P10S:    v.P10S,
		P90S:    v.P90S,
		Gaps:    v.Gaps,
		Missing: v.Missing,
	}
}

func toPadPickup(v replay.PadPickup) replaydoc.PadPickup {
	return replaydoc.PadPickup{
		Pad:   v.Pad,
		TLow:  v.TLow,
		THigh: v.THigh,
		XUID:  v.XUID,
		T:     v.T,
	}
}

func toGroundWeapon(v replay.GroundWeapon) replaydoc.GroundWeapon {
	return replaydoc.GroundWeapon{
		T0:      v.T0,
		T1:      v.T1,
		T1Max:   v.T1Max,
		X:       v.X,
		Y:       v.Y,
		Z:       v.Z,
		W:       v.W,
		Origin:  v.Origin,
		Dropper: v.Dropper,
		End:     v.End,
		Picker:  v.Picker,
	}
}
