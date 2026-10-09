package replay

import (
	"testing"

	"levelup/go-api/internal/games/halo_infinite/film/types"
)

// TestVehicleSeatToleranceEnFrames : la tolerance AVANT le debut d un episode vaut 2 s DE FILM,
// donc 20 frames sur une grille de 100 ms (J10.3, RB2-6 de l audit du 2026-09-24).
//
// LE DEFAUT : `vehicleSeatTolMS / clock.step` divisait 2 000 (des millisecondes) par 100 000 (des
// microsecondes) et rendait 0 frame — une montee lue 1,5 s avant le debut de l episode, que la
// tolerance existe pour rattacher, n etait jamais servie.
func TestVehicleSeatToleranceEnFrames(t *testing.T) {
	const occupant, vehicule = uint32(500), uint32(700)
	clock := vehClock() // origine 1 s, pas 100 ms
	key := types.LifeKey{Slot: vehicule, Gen: 1}
	lecture := func(tUS uint64) []types.VehicleOccupancy {
		return []types.VehicleOccupancy{{TimestampUS: tUS, Slot: occupant, Attached: true,
			ParentSlot: vehicule, HasSeat: true, Seat: 1}}
	}
	// L episode commence a la frame 40 (5 s de film).
	debut := clock.origin + 40*clock.step

	cas := []struct {
		nom    string
		tUS    uint64
		servie bool
	}{
		{"1,5 s avant le debut : dans la tolerance", debut - 1_500_000, true},
		{"2 s avant le debut : au bord, incluse", debut - 2_000_000, true},
		{"2,5 s avant le debut : hors tolerance", debut - 2_500_000, false},
	}
	for _, c := range cas {
		t.Run(c.nom, func(t *testing.T) {
			rides := map[types.LifeKey][]VehicleRide{key: {{Slot: occupant, T0: 40, T1: 160}}}
			n := assignVehicleSeats(rides, lecture(c.tUS), clock)
			if got := n == 1; got != c.servie {
				t.Fatalf("episode servi = %v, attendu %v (lecture a la frame %d, episode a la frame 40)",
					got, c.servie, clock.frame(c.tUS))
			}
		})
	}
}
