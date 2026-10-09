package service

import (
	"strings"
	"testing"

	"levelup/go-api/internal/games/halo_infinite/film/decfilm"
	"levelup/go-api/internal/games/halo_infinite/film/replay"
)

// replay_vehicle_scenery_aloft_test.go — LA VIE TENUE EN L AIR A VIDE (chantier « Falcon de
// Behemoth », 2026-10-02), sur des vies RECOPIEES des documents du parc au schema 76 (un
// echantillon sur cinq de la montee, positions et instants exacts ; `End` omis la ou il vaut `unknown`, la regle ne le lit pas).

// falconBorneSud : 1cd3848a 774 (Behemoth, Super Fiesta) — ne a la borne sud, monte seul de 2,2 m,
// derive de 2,4 m, puis plus rien n est replique jusqu a `T1` : il se tient en l air.
func falconBorneSud() replay.VehicleTrack {
	return replay.VehicleTrack{
		Slot: 774, Gen: 1, Family: "falcon", T0: 0, T1: 564, T1Max: 764,
		Spawn: &replay.VehicleSpawn{X: -146.02, Y: 27.43, Z: 8.54},
		Samples: []replay.VehicleSample{
			{T: 0, X: -146.01, Y: 27.43, Z: 8.54}, {T: 5, X: -145.76, Y: 26.57, Z: 8.66},
			{T: 10, X: -145.7, Y: 26.35, Z: 9.15}, {T: 15, X: -145.76, Y: 26.34, Z: 9.59},
			{T: 20, X: -146.18, Y: 26.07, Z: 10.31}, {T: 25, X: -146.84, Y: 25.68, Z: 10.66},
			{T: 30, X: -147.15, Y: 25.54, Z: 10.71}, {T: 35, X: -147.2, Y: 25.5, Z: 10.71},
			{T: 40, X: -147.22, Y: 25.5, Z: 10.74}, {T: 45, X: -147.26, Y: 25.47, Z: 10.75},
			{T: 50, X: -147.28, Y: 25.46, Z: 10.77}, {T: 55, X: -147.29, Y: 25.45, Z: 10.77},
			{T: 60, X: -147.3, Y: 25.44, Z: 10.78}, {T: 75, X: -147.31, Y: 25.45, Z: 10.8},
		},
	}
}

// falconBorneNord : e624c2a4 772 (Behemoth, Super Fiesta) — ne a la borne nord, se pose 8 cm plus
// bas et n en bouge plus. Personne ne le prend dans ce match ; au parc, 9 vies nees aux bornes nord
// sont pilotees.
func falconBorneNord() replay.VehicleTrack {
	return replay.VehicleTrack{
		Slot: 772, Gen: 1, Family: "falcon", T0: 0, T1: 5164, T1Max: 5164, End: replay.VehicleEndFilmEnd,
		Spawn: &replay.VehicleSpawn{X: -146.11, Y: 80.31, Z: 8.28},
		Samples: []replay.VehicleSample{
			{T: 0, X: -146.11, Y: 80.31, Z: 8.27}, {T: 1, X: -146.11, Y: 80.31, Z: 8.25},
			{T: 2, X: -146.1, Y: 80.31, Z: 8.2}, {T: 3, X: -146.1, Y: 80.3, Z: 8.19},
			{T: 731, X: -146.1, Y: 80.3, Z: 8.19},
		},
	}
}

func docAloft(vies ...replay.VehicleTrack) *replay.ReplayDocument {
	return docJoue(3.86, vies...)
}

func TestDecideVehicleScenery_TenuEnLAirAVideMasque(t *testing.T) {
	got := decideVehicleScenery(docAloft(falconBorneSud()), arenaBehemoth)
	if got == nil || len(got.Hidden) != 1 || got.Hidden[0].Slot != 774 ||
		got.Hidden[0].Reason != sceneryReasonAloftUnoccupied {
		t.Fatalf("le Falcon de la borne sud doit etre masque : %+v", got)
	}
	if got.Candidates != 0 {
		t.Errorf("la vie a bouge : elle n est pas une candidate de pose, compte %d", got.Candidates)
	}
}

// La regle ne lit pas la carte : une carte sans zone connue masque quand meme.
func TestDecideVehicleScenery_TenuEnLAirSansZone(t *testing.T) {
	got := decideVehicleScenery(docAloft(falconBorneSud()), nil)
	if got == nil || got.Zone != sceneryZoneUnknown || len(got.Hidden) != 1 || got.ZoneUnknown != 0 {
		t.Fatalf("sans zone, la vie tenue en l air reste masquee : %+v", got)
	}
}

func TestDecideVehicleScenery_FalconPoseAuSolAffiche(t *testing.T) {
	if got := decideVehicleScenery(docAloft(falconBorneNord()), arenaBehemoth); got != nil {
		t.Fatalf("le Falcon pose a sa borne n est d aucune regle : %+v", got)
	}
}

func TestDecideVehicleScenery_TenuEnLAirOccupeAffiche(t *testing.T) {
	v := falconBorneSud()
	v.Rides = []replay.VehicleRide{{T0: 40, T1: 60, Slot: 512}}
	if got := decideVehicleScenery(docAloft(v), arenaBehemoth); got != nil {
		t.Fatalf("une vie occupee n est jamais du decor : %+v", got)
	}
}

// 9b4dba45 783 : nee a la borne sud a t0=1124, son premier echantillon (1348) est deja en vol. La
// hauteur se mesure depuis la NAISSANCE (`spawn`), pas depuis le premier echantillon.
func TestDecideVehicleScenery_NeeEnBasPremierEchantillonEnVol(t *testing.T) {
	v := replay.VehicleTrack{
		Slot: 783, Gen: 1, Family: "falcon", T0: 1124, T1: 1730, T1Max: 1930,
		Spawn: &replay.VehicleSpawn{X: -146.02, Y: 27.43, Z: 8.54},
		Samples: []replay.VehicleSample{
			{T: 1348, X: -147.32, Y: 25.45, Z: 10.81}, {T: 1352, X: -147.33, Y: 25.44, Z: 10.83},
			{T: 1471, X: -147.35, Y: 25.42, Z: 10.83}, {T: 1647, X: -147.35, Y: 25.43, Z: 10.83},
		},
	}
	got := decideVehicleScenery(docAloft(v), arenaBehemoth)
	if got == nil || len(got.Hidden) != 1 {
		t.Fatalf("mesuree depuis sa naissance, la vie est tenue en l air : %+v", got)
	}
	v.Spawn = nil
	if got := decideVehicleScenery(docAloft(v), arenaBehemoth); got != nil {
		t.Fatalf("sans naissance lue, le premier echantillon fait foi (aucune elevation) : %+v", got)
	}
}

// Un vol dont l occupation n est pas lue s eloigne de sa naissance : la derive le garde affiche.
func TestDecideVehicleScenery_VolSansOccupantLuAffiche(t *testing.T) {
	v := falconBorneSud()
	v.Samples = append(v.Samples, replay.VehicleSample{T: 200, X: -120, Y: 40, Z: 25},
		replay.VehicleSample{T: 260, X: -120.1, Y: 40, Z: 25.1})
	if got := decideVehicleScenery(docAloft(v), arenaBehemoth); got != nil {
		t.Fatalf("une vie qui s eloigne de plus de %.0f m reste affichee : %+v", aloftMaxDriftM, got)
	}
}

func TestVehicleIsAloftUnoccupied_Conditions(t *testing.T) {
	minFrames := standMinFrames(100)
	piece := falconBorneSud()
	piece.Part = "turret"
	bref := falconBorneSud()
	bref.Samples, bref.T1 = bref.Samples[:6], 25 // jamais une seconde dans une bande de 0,3 m
	basse := falconBorneSud()
	for i := range basse.Samples {
		basse.Samples[i].Z = min(basse.Samples[i].Z, 8.54+aloftMinRiseM-0.05)
	}
	for nom, v := range map[string]replay.VehicleTrack{
		"piece montee": piece, "sans station": bref, "sous le seuil": basse,
	} {
		if vehicleIsAloftUnoccupied(v, minFrames) {
			t.Errorf("%s : n est pas tenue en l air", nom)
		}
	}
	if !vehicleIsAloftUnoccupied(falconBorneSud(), minFrames) {
		t.Error("le Falcon de la borne sud remplit les cinq conditions")
	}
}

func TestDecor_TenuEnLAirInscritAuRegistre(t *testing.T) {
	r, ok := decfilm.Lire(decfilm.Nom("repli_decor_tenu_en_l_air_a_vide"))
	if !ok {
		t.Fatal("repli_decor_tenu_en_l_air_a_vide : absent du registre facts/fallback")
	}
	if len(r.Sites) != 1 || !strings.HasSuffix(r.Sites[0].Fichier, "service/replay_vehicle_scenery_rule.go") {
		t.Errorf("site %+v, attendu la regle du decor", r.Sites)
	}
	if !r.CompteurBranche || r.CritereRetrait == "" || r.DatePose != "2026-10-02" {
		t.Errorf("compteur %v, critere %q, date %q", r.CompteurBranche, r.CritereRetrait, r.DatePose)
	}
}

// L echantillon final est TENU jusqu a `T1` : un Falcon dont le film ne replique que la montee se
// tient en l air jusqu a sa derniere preuve de presence.
func TestVehicleIsAloftUnoccupied_DernierEchantillonTenuJusquaT1(t *testing.T) {
	minFrames := standMinFrames(100)
	v := falconBorneSud()
	v.Samples = v.Samples[:7] // la montee seule, jusqu a 10,71 m a T=30
	if !vehicleIsAloftUnoccupied(v, minFrames) {
		t.Fatal("tenu jusqu a T1=564, le dernier echantillon fait une station en l air")
	}
	v.T1 = 30
	if vehicleIsAloftUnoccupied(v, minFrames) {
		t.Fatal("sans tenue, la montee seule n a aucune station")
	}
}

// Par le chemin de production, sur le fond REEL de Behemoth : seule la vie tenue en l air est
// masquee, le cadre dit la zone sans decoder le masque (aucune candidate de pose).
func TestVehicleScenery_BehemothTenuEnLAirReel(t *testing.T) {
	v := resoudreDecor(t, docAloft(falconBorneSud(), falconBorneNord()), "Behemoth")
	if v == nil || v.Zone != sceneryZoneMap || v.Candidates != 0 || len(v.Hidden) != 1 ||
		v.Hidden[0].Slot != 774 || v.Hidden[0].Reason != sceneryReasonAloftUnoccupied {
		t.Fatalf("Behemoth : le Falcon de la borne sud masque, celui de la borne nord affiche ; rendu %+v", v)
	}
}
