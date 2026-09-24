package service

import (
	"context"
	"testing"

	"levelup/go-api/internal/domain/title"
	"levelup/go-api/internal/games/halo_infinite/film/replay"
	"levelup/go-api/internal/port"
	"levelup/go-api/internal/testutil"
)

// replay_vehicle_scenery_test.go — LE DECOR DE CARTE SUR LES ZONES JOUABLES REELLES (lot M7 des
// retours du rejeu, 2026-09-24). Les fonds et calages sont ceux du DEPOT (references versionnees,
// servies en production) ; les vies et le sol joue sont recopies des documents reels reconstruits
// depuis les faits au schema 69 (tete de la vague C). Les noms de carte sont ceux du registre.

// sceneryService rend un service sur la racine du depot, dont la carte du match porte `noms`.
func sceneryService(t *testing.T, noms ...string) *replayService {
	t.Helper()
	root, err := testutil.RepoRoot()
	if err != nil {
		t.Fatal(err)
	}
	return NewReplayService(title.DefaultSlug, root, &mapNamesStub{names: noms}).(*replayService)
}

// viePosee rend une vie qui remplit les cinq conditions de pose.
func viePosee(slot uint32, family string, x, y, z float32) replay.VehicleTrack {
	return replay.VehicleTrack{
		Slot: slot, Gen: 1, Family: family, T0: 0, T1: 5000, T1Max: 5000, End: replay.VehicleEndFilmEnd,
		Samples: []replay.VehicleSample{{T: 0, X: x, Y: y, Z: z}},
	}
}

// docJoue rend un document dont le sol joue est `minZ`.
func docJoue(minZ float32, vies ...replay.VehicleTrack) *replay.ReplayDocument {
	return &replay.ReplayDocument{
		Bounds:   replay.Bounds{MinZ: minZ},
		Tracks:   []replay.Track{{Points: []replay.Point{{T: 0, Z: minZ}}}},
		Vehicles: vies,
	}
}

func resoudreDecor(t *testing.T, doc *replay.ReplayDocument, noms ...string) *replay.VehicleScenery {
	t.Helper()
	s := sceneryService(t, noms...)
	keys := s.matchMapKeys(context.Background(), "m7")
	s.resolveVehicleScenery(context.Background(), doc, "m7", keys)
	return doc.VehicleScenery
}

func raisons(v *replay.VehicleScenery) map[uint32]string {
	out := map[uint32]string{}
	for _, h := range v.Hidden {
		out[h.Slot] = h.Reason
	}
	return out
}

// Starboard (ab526724 et f0220a96 : memes six vies au centimetre) : hors de la matiere praticable.
func TestVehicleScenery_StarboardReel(t *testing.T) {
	for _, minZ := range []float32{70.42, 80.63} { // ab526724, f0220a96
		v := resoudreDecor(t, docJoue(minZ,
			viePosee(771, "scorpion", 8.96, -132.79, 81.7),
			viePosee(772, "wasp", -1.82, -130.09, 80.8),
			viePosee(773, "wasp", -1.82, -133.38, 80.8),
			viePosee(774, "warthog", 6.42, -129.74, 81.32),
			viePosee(776, "warthog", 1.58, -133.84, 81.33),
			viePosee(778, "warthog", 1.67, -132.04, 81.3),
		), "Starboard")
		if v == nil || v.Zone != sceneryZoneMap || v.Candidates != 6 || len(v.Hidden) != 6 {
			t.Fatalf("Starboard : 6 decors masques attendus, rendu %+v", v)
		}
		for slot, r := range raisons(v) {
			if r != sceneryReasonOffPlayArea {
				t.Errorf("slot %d : raison %q", slot, r)
			}
		}
	}
}

// Goliath (d8b13ec2 768) : DANS la matiere en plan, 3,03 m sous le sol joue.
func TestVehicleScenery_GoliathReel(t *testing.T) {
	v := resoudreDecor(t, docJoue(64.47, viePosee(768, "wasp", -0.44, -8.16, 61.44)), "Goliath")
	if v == nil || v.Zone != sceneryZoneMap || raisons(v)[768] != sceneryReasonBelowPlayedFloor {
		t.Fatalf("Goliath : le Wasp sous le sol doit etre masque, rendu %+v", v)
	}
}

// Behemoth hors Super Fiesta (7b0d89c4 773 et 774, f2966f08 769 et 770 : Mongoose poses au depart
// aux quatre coins des bases, RAMENES A UNE POSE SEULE) : dans l aire de jeu, jamais masques. Le
// Mongoose de f2966f08 770 est a 2,4 m du bord des positions jouees du match — le masque de la
// carte, lui, ne depend pas du match.
func TestVehicleScenery_BehemothPoseDansLAireDeJeuAffiche(t *testing.T) {
	v := resoudreDecor(t, docJoue(-11.19,
		viePosee(773, "mongoose", -101.61, 27.63, 8.8),
		viePosee(774, "mongoose", -101.65, 80.2, 8.31),
		viePosee(768, "mongoose", -146.02, 27.43, 8.59),
		viePosee(769, "mongoose", -146.11, 80.31, 8.33),
	), "Behemoth")
	if v == nil || v.Zone != sceneryZoneMap || v.Candidates != 4 || v.InPlayArea != 4 || len(v.Hidden) != 0 {
		t.Fatalf("Behemoth : 4 vehicules poses dans l aire de jeu, affiches ; rendu %+v", v)
	}
}

// Carte sans fond publie : repli nomme `unknown`, rien n est masque, tout est compte.
func TestVehicleScenery_CarteSansZoneReel(t *testing.T) {
	v := resoudreDecor(t, docJoue(70.42, viePosee(771, "scorpion", 8.96, -132.79, 81.7)),
		"Carte inexistante du test M7")
	if v == nil || v.Zone != sceneryZoneUnknown || v.ZoneUnknown != 1 || len(v.Hidden) != 0 {
		t.Fatalf("carte sans zone : rien masque, rendu %+v", v)
	}
}

// Aucune candidate : aucun chargement de masque, aucun verdict.
func TestVehicleScenery_SansCandidateAucunVerdict(t *testing.T) {
	simule := viePosee(771, "warthog", 1, 1, 1)
	simule.Samples = append(simule.Samples, replay.VehicleSample{T: 5, X: 1, Y: 1, Z: 1})
	doc := docJoue(0, simule)
	s := sceneryService(t, "Starboard")
	s.resolveVehicleScenery(context.Background(), doc, "m7", port.MatchMapKeys{Names: []string{"Starboard"}})
	if doc.VehicleScenery != nil {
		t.Fatalf("aucune candidate : verdict absent, rendu %+v", doc.VehicleScenery)
	}
}
