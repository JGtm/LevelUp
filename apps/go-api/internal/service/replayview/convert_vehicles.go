package replayview

// convert_vehicles.go — vies de vehicule, positions, occupants. Jumeau de
// `domain/replaydoc/vehicles.go`.

import (
	"levelup/go-api/internal/analysis/replay"
	"levelup/go-api/internal/domain/replaydoc"
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
		Spawn:   ptrOf(v.Spawn, toVehicleSpawn),
		Samples: sliceOf(v.Samples, toVehicleSample),
		Rides:   sliceOf(v.Rides, toVehicleRide),
	}
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
	}
}

func toVehicleAim(v replay.VehicleAim) replaydoc.VehicleAim {
	return replaydoc.VehicleAim{
		T: v.T,
		H: v.H,
		P: v.P,
	}
}
