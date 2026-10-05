package replay

import (
	"testing"

	"levelup/go-api/internal/games/halo_infinite/film/types"
)

// filmRideFixture : une vie de vehicule (slot 900, frames 0 a 600 sous `vehClock`) et la montee a
// bord LUE du bipede 600 de generation 1 a la frame 100, conducteur.
func filmRideFixture(suite types.VehicleOccupancy) vehicleRideInputs {
	key := types.LifeKey{Slot: 900, Gen: 1}
	return vehicleRideInputs{
		lives:    []vehicleLife{{key: key, loUS: 1_000_000, hiUS: 61_000_000}},
		drawable: map[types.LifeKey]bool{key: true},
		occupancy: []types.VehicleOccupancy{
			{TimestampUS: 11_000_000, Slot: 600, Gen: 1, Attached: true, ParentSlot: 900, ParentGen: 1,
				HasSeat: true},
			suite,
		},
		clock: vehClock(),
	}
}

// TestUneLectureDUneAutreGenerationNeFermePasLEpisode : la fermeture par la lecture suivante est
// celle du MEME OBJET. Une lecture du slot 600 en generation 3 — un autre objet, ou une lecture
// fausse, ici attachee a un bipede — ne ferme pas l episode de la generation 1, qui court jusqu a
// la fin de la vie du vehicule ; la descente lue du meme objet, elle, le ferme.
func TestUneLectureDUneAutreGenerationNeFermePasLEpisode(t *testing.T) {
	cas := []struct {
		nom      string
		suite    types.VehicleOccupancy
		finFrame int
	}{
		{"autre generation", types.VehicleOccupancy{TimestampUS: 21_000_000, Slot: 600, Gen: 3,
			Attached: true, ParentSlot: 618, ParentGen: 3}, 600},
		{"meme objet, descente", types.VehicleOccupancy{TimestampUS: 21_000_000, Slot: 600, Gen: 1}, 200},
	}
	for _, c := range cas {
		lus, _ := buildVehicleFilmRides(filmRideFixture(c.suite))
		rides := lus.rides[types.LifeKey{Slot: 900, Gen: 1}]
		if len(rides) != 1 {
			t.Fatalf("%s : %d episodes, veut 1", c.nom, len(rides))
		}
		if rides[0].T0 != 100 || rides[0].T1 != c.finFrame {
			t.Errorf("%s : episode [%d, %d], veut [100, %d]", c.nom, rides[0].T0, rides[0].T1, c.finFrame)
		}
	}
}

// TestLaPrimauteNommeLeJoueurPasLeCorps : le film lit le joueur 111 (slot 574) dans la vie de
// vehicule [800, 900]. Son episode de repli d une vie precedente (slot 745, [600, 700]) n est pas
// contredit ; celui d un autre joueur l est, comme celui du meme joueur qui chevauche la lecture,
// et comme un corps sans identite qui n est pas celui que le film a lu.
func TestLaPrimauteNommeLeJoueurPasLeCorps(t *testing.T) {
	key := types.LifeKey{Slot: 900, Gen: 1}
	f := vehicleFilmRides{
		rides:     map[types.LifeKey][]VehicleRide{},
		occupants: map[types.LifeKey]map[identiteOccupant]bool{},
		fenetres:  map[types.LifeKey][][2]int{},
	}
	f.ajouter(key, VehicleRide{T0: 800, T1: 900, Slot: 574, XUID: "111", Src: VehicleRideSrcFilm})
	f.ajouter(key, VehicleRide{T0: 950, T1: 990, Slot: 590, Src: VehicleRideSrcFilm})
	cas := []struct {
		nom       string
		r         VehicleRide
		contredit bool
	}{
		{"meme joueur, autre vie", VehicleRide{T0: 600, T1: 700, Slot: 745, XUID: "111"}, false},
		{"autre joueur", VehicleRide{T0: 600, T1: 700, Slot: 746, XUID: "222"}, true},
		{"meme joueur, chevauchement", VehicleRide{T0: 850, T1: 870, Slot: 745, XUID: "111"}, true},
		{"corps sans identite, autre slot", VehicleRide{T0: 600, T1: 700, Slot: 591}, true},
		{"corps sans identite, slot lu", VehicleRide{T0: 600, T1: 700, Slot: 590}, false},
	}
	for _, c := range cas {
		if got := f.contredit(key, c.r); got != c.contredit {
			t.Errorf("%s : contredit = %v, veut %v", c.nom, got, c.contredit)
		}
	}
}
