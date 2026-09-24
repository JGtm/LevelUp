package replayview

// convert_vehicles.go — vies de vehicule, positions, occupants. Jumeau de
// `domain/replaydoc/vehicles.go`.

import (
	"levelup/go-api/internal/domain/replaydoc"
	"levelup/go-api/internal/games/halo_infinite/film/replay"
)

func toVehicleTrack(v replay.VehicleTrack) replaydoc.VehicleTrack {
	return replaydoc.VehicleTrack{
		Slot:    v.Slot,
		Gen:     v.Gen,
		Chassis: v.Chassis,
		Family:  v.Family,
		T0:      v.T0,
		T1:      v.T1,
		T1Max:   v.T1Max,
		End:     v.End,
		TEnd:    v.TEnd,
		Spawn:   ptrOf(v.Spawn, toVehicleSpawn),
		Samples: sliceOf(v.Samples, toVehicleSample),
		Rides:   sliceOf(v.Rides, toVehicleRide),
		Part:    v.Part,
		Carrier: ptrOf(v.Carrier, toVehicleLifeRef),
		Variant: v.Variant,
	}
}

func toVehicleLifeRef(v replay.VehicleLifeRef) replaydoc.VehicleLifeRef {
	return replaydoc.VehicleLifeRef{Slot: v.Slot, Gen: v.Gen}
}

func toVehicleWeapon(v replay.VehicleWeapon) replaydoc.VehicleWeapon {
	return replaydoc.VehicleWeapon{
		Vehicle: v.Vehicle,
		En:      v.En,
		Fr:      v.Fr,
		Fire:    v.Fire,
		Fx:      v.Fx,
		Tint:    v.Tint,
		Sound:   v.Sound,
		Mount:   ptrOf(v.Mount, toVehicleWeaponMount),
	}
}

func toVehicleWeaponMount(v replay.VehicleWeaponMount) replaydoc.VehicleWeaponMount {
	return replaydoc.VehicleWeaponMount{Aim: v.Aim, AX: v.AX, AY: v.AY}
}

func toVehicleSpawn(v replay.VehicleSpawn) replaydoc.VehicleSpawn {
	return replaydoc.VehicleSpawn{
		X: v.X,
		Y: v.Y,
		Z: v.Z,
		H: v.H,
	}
}

func toVehicleSample(v replay.VehicleSample) replaydoc.VehicleSample {
	return replaydoc.VehicleSample{
		T: v.T,
		X: v.X,
		Y: v.Y,
		Z: v.Z,
		G: v.G,
		H: v.H,
	}
}

func toVehicleRide(v replay.VehicleRide) replaydoc.VehicleRide {
	return replaydoc.VehicleRide{
		T0:   v.T0,
		T1:   v.T1,
		Slot: v.Slot,
		XUID: v.XUID,
		Seat: v.Seat,
		Src:  v.Src,
		Aim:  sliceOf(v.Aim, toVehicleAim),
		// Turret : schema 69.
		Turret: ptrOf(v.Turret, toVehicleLifeRef),
	}
}

func toVehicleAim(v replay.VehicleAim) replaydoc.VehicleAim {
	return replaydoc.VehicleAim{
		T: v.T,
		H: v.H,
		P: v.P,
	}
}

func toVehicleCycle(v replay.VehicleCycle) replaydoc.VehicleCycle {
	return replaydoc.VehicleCycle{
		X:       v.X,
		Y:       v.Y,
		Family:  v.Family,
		MedianS: v.MedianS,
		P10S:    v.P10S,
		P90S:    v.P90S,
		Gaps:    v.Gaps,
		Missing: v.Missing,
	}
}
