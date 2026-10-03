package replay

// vehicle_rides_next_life_test.go — UN JOUEUR N EST JAMAIS A DEUX ENDROITS (reprise du lot M7b,
// revue adverse RR-M7b-01). Gabarits construits sur la FORME mesuree au parc — un joueur mort a
// bord qui reapparait sous un autre slot de bipede —, jamais sur un match.

import (
	"testing"

	"levelup/go-api/internal/games/halo_infinite/film/internal/facts/fallback"
)

// nlTracks : trois vies publiees — le joueur A (slot 10 puis 20), un bot (slot 11 puis 21).
func nlTracks() []Track {
	return []Track{
		{Slot: 10, XUID: "A", StartFrame: 0, EndFrame: 50},
		{Slot: 20, XUID: "A", StartFrame: 80, EndFrame: 300},
		{Slot: 11, Bot: "Bot [bot]", StartFrame: 0, EndFrame: 40},
		{Slot: 21, Bot: "Bot [bot]", StartFrame: 90, EndFrame: 300},
	}
}

// TestEpisodeCoupeALaNaissanceDeLaVieSuivante — l episode du joueur A court de 50 a 200 alors que
// sa vie suivante nait a 80 : il est coupe a 79, sa visee avec lui, et la coupe est comptee. Meme
// regle pour un bot (identite = son nom), et pour un episode LU : la naissance ecrite borne tout.
func TestEpisodeCoupeALaNaissanceDeLaVieSuivante(t *testing.T) {
	vehicles := []VehicleTrack{{Slot: 700, Rides: []VehicleRide{
		{T0: 50, T1: 200, Slot: 10, XUID: "A", Src: VehicleRideSrcFilm,
			Aim: []VehicleAim{{T: 60}, {T: 79}, {T: 80}, {T: 150}}},
		{T0: 40, T1: 120, Slot: 11, Src: VehicleRideSrcProximity},
	}}}
	fb := fallback.NouveauCompteur()
	if n := cutRidesAtNextLife(vehicles, nlTracks(), fb); n != 2 {
		t.Fatalf("coupes = %d, attendu 2", n)
	}
	if got := fbCount(fb, fallback.NomEpisodeBorneParLaVieSuivante); got != 2 {
		t.Errorf("repli compte %d fois, attendu 2 (une par coupe)", got)
	}
	a, bot := vehicles[0].Rides[0], vehicles[0].Rides[1]
	if a.T0 != 50 || a.T1 != 79 || len(a.Aim) != 2 {
		t.Errorf("episode de A = %d-%d (%d visees), attendu 50-79 et 2 visees", a.T0, a.T1, len(a.Aim))
	}
	if bot.T1 != 89 {
		t.Errorf("episode du bot = %d-%d, attendu coupe a 89 (sa vie suivante nait a 90)", bot.T0, bot.T1)
	}
}

// TestEpisodeSansVieSuivanteDansSaFenetreIntact — les cas qui ne coupent RIEN : la vie suivante
// nait apres la fin de l episode ; l occupant n a pas d identite publiee ; une autre identite nait
// pendant l episode.
func TestEpisodeSansVieSuivanteDansSaFenetreIntact(t *testing.T) {
	vehicles := []VehicleTrack{{Slot: 700, Rides: []VehicleRide{
		{T0: 50, T1: 79, Slot: 10, XUID: "A", Src: VehicleRideSrcFilm},
		{T0: 10, T1: 250, Slot: 99, Src: VehicleRideSrcProximity},
		{T0: 60, T1: 250, Slot: 12, XUID: "B", Src: VehicleRideSrcProximity},
	}}}
	if n := cutRidesAtNextLife(vehicles, nlTracks(), nil); n != 0 {
		t.Fatalf("coupes = %d, attendu 0 : %+v", n, vehicles[0].Rides)
	}
}

// TestRecompteDesEpisodesApresCoupe — la couverture des episodes decrit ce qui est PUBLIE : apres la
// coupe, la duree couverte et le chevauchement se recomptent (le reste du calque ne bouge pas).
func TestRecompteDesEpisodesApresCoupe(t *testing.T) {
	vehicles := []VehicleTrack{{Slot: 700, Rides: []VehicleRide{
		{T0: 50, T1: 200, Slot: 10, XUID: "A", Src: VehicleRideSrcFilm},
		{T0: 100, T1: 150, Slot: 30, XUID: "C", Src: VehicleRideSrcProximity},
	}}}
	cov := VehicleCoverage{Published: 7}
	tallyVehicleRides(vehicles[0].Rides, &cov)
	if cov.Ambiguous != 1 {
		t.Fatalf("prealable : les deux episodes se chevauchent (%+v)", cov)
	}
	cutRidesAtNextLife(vehicles, nlTracks(), nil)
	recountVehicleRides(vehicles, &cov)
	if cov.Ambiguous != 0 || cov.AimRideFrames != 30+51 || cov.Rides != 2 || cov.Published != 7 {
		t.Errorf("couverture = %+v, attendu ambigu 0, 81 frames, 2 episodes, published intact", cov)
	}
}
