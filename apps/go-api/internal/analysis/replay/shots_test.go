package replay

import (
	"math"
	"testing"

	"levelup/go-api/internal/analysis/filmdec"
)

// posAt fabrique un échantillon de position portant un cap de visée i21.
func posAt(slot uint32, tUS uint64, x, y float32, headingDeg float64) filmdec.BipedPosition {
	p := filmdec.BipedPosition{Slot: slot, TimestampUS: tUS, X: x, Y: y, HasWorld: true}
	p.HasYaw = true
	// AimHeadingDeg = 360*(q+0.5)/4096 : on inverse pour poser le cap voulu.
	p.YawRaw = uint32(headingDeg/360*4096 - 0.5)
	return p
}

func fireAt(tUS uint64, playerIdx int, headingDeg float64) filmdec.FireEvent {
	e := filmdec.FireEvent{TimestampUS: tUS, FilmIndex: playerIdx, WeaponID: 0x6ACDC44D42C9679F}
	code, ok := filmdec.EncodeAimVector(unitFromHeading(headingDeg), filmdec.FireAimBits)
	if !ok {
		panic("largeur de visée refusée")
	}
	v, ok := filmdec.DecodeAimVectorChecked(code, filmdec.FireAimBits)
	if !ok {
		panic("visée non décodable")
	}
	e.HasAim, e.Aim = true, v
	return e
}

func unitFromHeading(deg float64) [3]float32 {
	r := deg * math.Pi / 180
	return [3]float32{float32(math.Cos(r)), float32(math.Sin(r)), 0}
}

// TestBuildShots_PlacesShotOnItsOwnerSlot : le tir est posé à la position du slot que le pont
// désigne pour son auteur.
//
// LE PONT EST FOURNI EXPLICITEMENT, et c'est plus honnête qu'avant : ce test portait sur
// `voteSlotOwners`, qui construisait le pont en faisant élire un propriétaire par la visée.
// Cette fonction est supprimée. La construction du pont est testée là où elle vit désormais
// (lives_test.go, à partir du fil des morts) ; ici on teste ce que buildShots en fait.
func TestBuildShots_PlacesShotOnItsOwnerSlot(t *testing.T) {
	var pos []filmdec.BipedPosition
	for i := uint64(0); i < 20; i++ {
		ts := 1_000_000 + i*50_000
		pos = append(pos, posAt(10, ts, 1, 1, 90))  // slot 10 regarde vers +Y
		pos = append(pos, posAt(11, ts, 5, 5, 270)) // slot 11 regarde vers -Y
	}
	events := []filmdec.FireEvent{
		fireAt(1_200_000, 3, 90),
		fireAt(1_300_000, 3, 90),
		{TimestampUS: 1_400_000, FilmIndex: 3, WeaponID: 1}, // sans visée : le pont suffit
	}
	shots, cov := buildShots(pos, events, 1_000_000, 100_000, map[uint32]int{10: 3})
	// L'invariant de couverture vaut AUSSI sur le cas nominal : tout ce qui existait est
	// soit rattache, soit rejete sous une cause nommee.
	if !cov.Balanced() {
		t.Errorf("fuite dans la couverture : %+v", cov)
	}
	if len(shots) != 3 {
		t.Fatalf("tirs publiés = %d, attendu 3 : %+v", len(shots), shots)
	}
	for _, s := range shots {
		if s.Slot != 10 {
			t.Errorf("tir rattaché au slot %d, attendu 10", s.Slot)
		}
		if s.X != 1 || s.Y != 1 {
			t.Errorf("origine = (%v, %v), attendu la position du tireur (1, 1)", s.X, s.Y)
		}
	}
	if shots[0].Weapon != "0x6ACDC44D42C9679F" {
		t.Errorf("arme = %q, attendu l'identifiant 64 bits en hexadécimal", shots[0].Weapon)
	}
	if shots[2].H != 0 {
		t.Errorf("un tir sans visée décodée ne doit porter aucun cap, reçu %v", shots[2].H)
	}
}

// TestBuildShots_RejectsAmbiguous : deux bipeds au même cap -> aucune désignation, donc
// aucun tir publié. Un tir placé au mauvais endroit serait pire que pas de tir.
func TestBuildShots_RejectsAmbiguous(t *testing.T) {
	var pos []filmdec.BipedPosition
	for i := uint64(0); i < 20; i++ {
		ts := 1_000_000 + i*50_000
		pos = append(pos, posAt(10, ts, 1, 1, 90))
		pos = append(pos, posAt(11, ts, 5, 5, 90)) // même cap : ambigu
	}
	events := []filmdec.FireEvent{fireAt(1_200_000, 3, 90), fireAt(1_300_000, 3, 90)}
	// DEUX slots pour le MEME joueur au meme instant : le rattachement est ambigu, et rien ne
	// doit etre publie. C'est le cas que la categorie `Ambiguous` existe pour nommer.
	shots, cov := buildShots(pos, events, 1_000_000, 100_000, map[uint32]int{10: 3, 11: 3})
	if len(shots) != 0 {
		t.Fatalf("tirs publiés malgré l'ambiguïté : %+v", shots)
	}
	if cov.Ambiguous != len(events) {
		t.Errorf("attendu %d rejets pour ambiguite, obtenu %+v", len(events), cov)
	}
	if !cov.Balanced() {
		t.Errorf("fuite dans la couverture : %+v", cov)
	}
}
