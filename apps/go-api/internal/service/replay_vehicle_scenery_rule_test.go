package service

import (
	"testing"

	"levelup/go-api/internal/games/halo_infinite/film/replay"
)

// replay_vehicle_scenery_rule_test.go — LA REGLE DU DECOR, PURE (lot M7 des retours du rejeu,
// 2026-09-24), sur des vies RECOPIEES des documents reels reconstruits au schema 69. La zone en
// plan est ici un rectangle de test ; la zone REELLE (masque des fonds publies) est prouvee par
// replay_vehicle_scenery_test.go.

// zoneRect est une zone jouable de test : un rectangle en plan.
type zoneRect struct{ minX, minY, maxX, maxY float64 }

func (z zoneRect) Praticable(x, y float64) bool {
	return x >= z.minX && x <= z.maxX && y >= z.minY && y <= z.maxY
}

// Les arenes jouees (bornes publiees) des trois temoins.
var (
	arenaStarboard = zoneRect{minX: -13.88, minY: -110.34, maxX: 21.61, maxY: -76.61}
	arenaGoliath   = zoneRect{minX: -17.31, minY: -25.49, maxX: 10.98, maxY: 17.68}
	arenaBehemoth  = zoneRect{minX: -153.5, minY: 19.96, maxX: -98.81, maxY: 87.81}
)

// scorpionStarboard : ab526724 771, pose 22 m au sud de l arene.
func scorpionStarboard() replay.VehicleTrack { return viePosee(771, "scorpion", 8.96, -132.79, 81.7) }

func TestDecideVehicleScenery_HorsDuPlanMasque(t *testing.T) {
	got := decideVehicleScenery(docJoue(70.42, scorpionStarboard()), arenaStarboard)
	if got == nil || got.Zone != sceneryZoneMap || got.Floor != sceneryFloorPlayed ||
		len(got.Hidden) != 1 || got.Hidden[0].Reason != sceneryReasonOffPlayArea {
		t.Fatalf("verdict = %+v", got)
	}
}

func TestDecideVehicleScenery_SousLeSolJoueMasque(t *testing.T) {
	got := decideVehicleScenery(docJoue(64.47, viePosee(768, "wasp", -0.44, -8.16, 61.44)), arenaGoliath)
	if got == nil || len(got.Hidden) != 1 || got.Hidden[0].Reason != sceneryReasonBelowPlayedFloor {
		t.Fatalf("le Wasp de Goliath, 3,03 m sous le sol joue, doit etre masque : %+v", got)
	}
}

// 7b0d89c4 773 (Behemoth) : un Mongoose pose dans l aire de jeu, que personne ne touche.
func TestDecideVehicleScenery_PoseDansLaZoneAffiche(t *testing.T) {
	got := decideVehicleScenery(docJoue(-11.19, viePosee(773, "mongoose", -101.61, 27.63, 8.8)), arenaBehemoth)
	if got == nil || got.Candidates != 1 || got.InPlayArea != 1 || len(got.Hidden) != 0 {
		t.Fatalf("un vehicule pose dans l aire de jeu reste affiche : %+v", got)
	}
}

// 8a485699 782 (Launch Site) : le socle de la Wasp est 0,10 m sous tout pas de joueur du match.
// Ramenee a une pose seule, elle n est PAS sous le sol : la tolerance nommee la garde affichee.
func TestDecideVehicleScenery_SocleJusteSousLeSolJoueAffiche(t *testing.T) {
	wasp := viePosee(782, "wasp", 1.86, 7.13, -4.02)
	got := decideVehicleScenery(docJoue(-3.92, wasp), zoneRect{minX: -25, minY: -44, maxX: 37, maxY: 26})
	if got == nil || got.InPlayArea != 1 || len(got.Hidden) != 0 {
		t.Fatalf("socle a 0,10 m sous le sol joue : affiche, rendu %+v", got)
	}
}

func TestDecideVehicleScenery_CarteSansZoneNeMasqueRien(t *testing.T) {
	got := decideVehicleScenery(docJoue(70.42, scorpionStarboard()), nil)
	if got == nil || got.Zone != sceneryZoneUnknown || got.ZoneUnknown != 1 || len(got.Hidden) != 0 {
		t.Fatalf("zone inconnue : rien masque, tout compte : %+v", got)
	}
}

func TestDecideVehicleScenery_SolInconnuNAppliquePasLaHauteur(t *testing.T) {
	doc := &replay.ReplayDocument{Vehicles: []replay.VehicleTrack{viePosee(768, "wasp", -0.44, -8.16, 61.44)}}
	got := decideVehicleScenery(doc, arenaGoliath)
	if got == nil || got.Floor != sceneryFloorUnknown || got.InPlayArea != 1 || len(got.Hidden) != 0 {
		t.Fatalf("sans position de joueur, pas de test de hauteur : %+v", got)
	}
}

func TestVehicleIsPosedOnly_CinqConditions(t *testing.T) {
	base := scorpionStarboard()
	simule := base
	simule.Samples = []replay.VehicleSample{{T: 0, X: 8.96, Y: -132.79}, {T: 10, X: 8.96, Y: -132.79}}
	tardif := base
	tardif.T0, tardif.Samples = 6400, []replay.VehicleSample{{T: 6400, X: 8.96, Y: -132.79}}
	occupe := base
	occupe.Rides = []replay.VehicleRide{{T0: 0, T1: 10}}
	detruit := base
	detruit.End = replay.VehicleEndDestroyed
	horsNaissance := base
	horsNaissance.Samples = []replay.VehicleSample{{T: 3601, X: 8.96, Y: -132.79}}
	for nom, v := range map[string]replay.VehicleTrack{
		"simule": simule, "ne tard": tardif, "occupe": occupe, "detruit": detruit, "echantillon tardif": horsNaissance,
	} {
		if vehicleIsPosedOnly(v) {
			t.Errorf("%s : n est pas une pose seule", nom)
		}
	}
	if !vehicleIsPosedOnly(base) {
		t.Error("la vie posee de Starboard remplit les cinq conditions")
	}
	if got := decideVehicleScenery(docJoue(70.42, simule, occupe), arenaStarboard); got != nil {
		t.Errorf("aucune candidate : verdict absent, rendu %+v", got)
	}
}
