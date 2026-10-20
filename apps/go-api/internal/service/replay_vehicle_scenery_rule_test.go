package service

import (
	"strings"
	"testing"

	"levelup/go-api/internal/games/halo_infinite/film/decfilm"
	"levelup/go-api/internal/games/halo_infinite/film/replay"
)

// replay_vehicle_scenery_rule_test.go — LA REGLE DU DECOR, PURE (lot M7 des retours du rejeu,
// 2026-09-24), sur des vies RECOPIEES des documents reels reconstruits au schema 69 ; le sol est le
// sol FOULE mesure de chaque match temoin. La zone en
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
	got := decideVehicleScenery(docJoue(80.61, scorpionStarboard()), arenaStarboard)
	if got == nil || got.Zone != sceneryZoneMap || got.Floor != sceneryFloorPlayed ||
		len(got.Hidden) != 1 || got.Hidden[0].Reason != sceneryReasonOffPlayArea {
		t.Fatalf("verdict = %+v", got)
	}
}

func TestDecideVehicleScenery_SousLeSolJoueMasque(t *testing.T) {
	got := decideVehicleScenery(docJoue(64.47, waspGoliath()), arenaGoliath)
	if got == nil || len(got.Hidden) != 1 || got.Hidden[0].Reason != sceneryReasonBelowPlayedFloor {
		t.Fatalf("le Wasp de Goliath, 3,03 m sous le sol foule, doit etre masque : %+v", got)
	}
}

// 7b0d89c4 773 (Behemoth) : un Mongoose pose dans l aire de jeu, que personne ne touche.
func TestDecideVehicleScenery_PoseDansLaZoneAffiche(t *testing.T) {
	got := decideVehicleScenery(docJoue(3.86, viePosee(773, "mongoose", -101.61, 27.63, 8.8)), arenaBehemoth)
	if got == nil || got.Candidates != 1 || got.InPlayArea != 1 || len(got.Hidden) != 0 {
		t.Fatalf("un vehicule pose dans l aire de jeu reste affiche : %+v", got)
	}
}

// 8a485699 782 (Launch Site) : le socle de la Wasp est 0,10 m sous le sol foule du match.
// Ramenee a une pose seule, elle n est PAS sous le sol : la tolerance nommee la garde affichee.
func TestDecideVehicleScenery_SocleJusteSousLeSolJoueAffiche(t *testing.T) {
	wasp := viePosee(782, "wasp", 1.86, 7.13, -4.02)
	got := decideVehicleScenery(docJoue(-3.92, wasp), zoneRect{minX: -25, minY: -44, maxX: 37, maxY: 26})
	if got == nil || got.InPlayArea != 1 || len(got.Hidden) != 0 {
		t.Fatalf("socle a 0,10 m sous le sol foule : affiche, rendu %+v", got)
	}
}

func TestDecideVehicleScenery_CarteSansZoneNeMasqueRien(t *testing.T) {
	got := decideVehicleScenery(docJoue(80.61, scorpionStarboard()), nil)
	if got == nil || got.Zone != sceneryZoneUnknown || got.ZoneUnknown != 1 || len(got.Hidden) != 0 {
		t.Fatalf("zone inconnue : rien masque, tout compte : %+v", got)
	}
}

func TestDecideVehicleScenery_SolInconnuNAppliquePasLaHauteur(t *testing.T) {
	doc := &replay.ReplayDocument{Vehicles: []replay.VehicleTrack{viePosee(768, "wasp", -0.44, -8.16, 61.44)}}
	got := decideVehicleScenery(doc, arenaGoliath)
	if got == nil || got.Floor != sceneryFloorUnknown || got.InPlayArea != 1 || len(got.Hidden) != 0 {
		t.Fatalf("personne ne s est tenu nulle part : pas de test de hauteur : %+v", got)
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
	if got := decideVehicleScenery(docJoue(80.61, simule, occupe), arenaStarboard); got != nil {
		t.Errorf("aucune candidate : verdict absent, rendu %+v", got)
	}
}

// waspGoliath : d8b13ec2 768, pose 3,03 m sous le sol foule du match.
func waspGoliath() replay.VehicleTrack { return viePosee(768, "wasp", -0.44, -8.16, 61.44) }

// chute : un joueur qui tombe de `depuis` pendant 1,2 s puis meurt — sa plus basse position
// publiee est 24,5 m plus bas, et la garde des aberrations du lot M1 la garde (une chute dans un
// vide REEL n est pas un artefact de decodage).
func chute(t0 int, depuis float32) replay.Track {
	var pts []replay.Point
	for k := 0; k <= 12; k++ {
		pts = append(pts, replay.Point{T: t0 + k, Z: depuis - float32(k*k)*0.17})
	}
	return replay.Track{Points: pts}
}

// LA CHUTE N ABAISSE PAS LE SOL (revue RR-M7-04). Le sol lu etait `Bounds.MinZ` : la plus basse
// position publiee. Une seule chute sur Goliath l aurait tire 24 m plus bas et rendu le Wasp de
// decor. Le sol FOULE ne retient que les stations (1 s dans 0,3 m) : la chute n en est pas une.
func TestDecideVehicleScenery_UneChuteNAbaissePasLeSol(t *testing.T) {
	doc := docJoue(64.47, waspGoliath())
	c := chute(100, 64.47)
	doc.Tracks = append(doc.Tracks, c)
	doc.Bounds.MinZ = c.Points[len(c.Points)-1].Z
	got := decideVehicleScenery(doc, arenaGoliath)
	if got == nil || raisons(got)[768] != sceneryReasonBelowPlayedFloor {
		t.Fatalf("une chute a %.2f m ne doit pas rendre le Wasp de decor : %+v", doc.Bounds.MinZ, got)
	}
}

// Des joueurs qui se TIENNENT plus bas que le Wasp : il y a du jeu a sa hauteur, il est affiche.
// C est la dependance au match qui reste, et elle est voulue (registre facts/fallback).
func TestDecideVehicleScenery_UneStationPlusBasseRendLeVehicule(t *testing.T) {
	doc := docJoue(64.47, waspGoliath())
	doc.Tracks = append(doc.Tracks, replay.Track{Points: []replay.Point{
		{T: 200, Z: 58.10}, {T: 204, Z: 58.02}, {T: 210, Z: 58.12},
	}})
	got := decideVehicleScenery(doc, arenaGoliath)
	if got == nil || got.InPlayArea != 1 || len(got.Hidden) != 0 {
		t.Fatalf("station d 1 s a 58 m : le Wasp a 61,44 m est a hauteur de jeu, rendu %+v", got)
	}
}

// Une station trop breve (0,5 s : l apogee d un saut, un rebond) ne fait pas un sol.
func TestPlayedFloor_StationTropBreveIgnoree(t *testing.T) {
	doc := docJoue(64.47)
	doc.Tracks = append(doc.Tracks, replay.Track{Points: []replay.Point{{T: 300, Z: 50}, {T: 305, Z: 50.1}}})
	if sol, ok := playedFloor(doc); !ok || sol != 64.47 {
		t.Fatalf("sol = %.2f (%v), attendu 64,47 : une station de 0,5 s n est pas un sol", sol, ok)
	}
}

// Sans echelle de temps, aucune duree ne se mesure : le sol est inconnu, le test de hauteur ne
// s applique pas.
func TestPlayedFloor_SansEchelleDeTempsInconnu(t *testing.T) {
	doc := docJoue(64.47)
	doc.FrameIntervalMS = 0
	if _, ok := playedFloor(doc); ok {
		t.Fatal("sans frameIntervalMs, le sol foule doit etre inconnu")
	}
}

// La fenetre glisse : une station au MILIEU d une piste qui monte et descend est trouvee, et c est
// son point le plus bas qui fait le sol.
func TestLowestStand_FenetreGlissante(t *testing.T) {
	pts := []replay.Point{
		{T: 0, Z: 80}, {T: 1, Z: 75}, {T: 2, Z: 70.2}, {T: 5, Z: 70.0}, {T: 9, Z: 70.25},
		{T: 12, Z: 70.1}, {T: 13, Z: 76}, {T: 14, Z: 60}, {T: 15, Z: 90},
	}
	sol, ok := lowestStand(pts, 10)
	if !ok || sol != 70.0 {
		t.Fatalf("station 70,0 a 70,25 sur 10 images : sol = %.2f (%v)", sol, ok)
	}
}

// TestDecor_ReplisInscritsAuRegistre (revue RR-M7-03) : les deux replis de la regle sont au registre
// facts/fallback — nommes, dates, avec leur critere de retrait — et leur site est CETTE regle. §4.0
// du plan : « un repli est NOMME, COMPTE en couverture, inscrit au registre avec date et critere de
// retrait ». Le compte est publie par le calque lui-meme (`hidden[].reason`, `zoneUnknown`).
func TestDecor_ReplisInscritsAuRegistre(t *testing.T) {
	for _, nom := range []string{"repli_decor_sous_le_sol_foule_du_match", "repli_decor_carte_sans_zone_affiche"} {
		r, ok := decfilm.Lire(decfilm.Nom(nom))
		if !ok {
			t.Errorf("%s : absent du registre facts/fallback", nom)
			continue
		}
		if len(r.Sites) != 1 || !strings.HasSuffix(r.Sites[0].Fichier, "service/replay_vehicle_scenery_rule.go") {
			t.Errorf("%s : site %+v, attendu la regle du decor", nom, r.Sites)
		}
		if !r.CompteurBranche || r.CritereRetrait == "" || r.DatePose != "2026-09-24" {
			t.Errorf("%s : compteur %v, critere %q, date %q", nom, r.CompteurBranche, r.CritereRetrait, r.DatePose)
		}
	}
}
