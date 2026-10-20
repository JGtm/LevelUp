package replay

import (
	"context"
	"testing"

	"levelup/go-api/internal/games/halo_infinite/film/internal/grammar"
)

// tirs_par_vie_test.go — UN TIR, UN LANCER, UN RELEVE D'UNITE VONT AU CORPS QUI TIENT LE SLOT A
// L'INSTANT (RA2-3, lot J5.4 du plan de suite de l'audit du decodeur, 2026-09-27).

// entreeDeuxJoueursUnSlot : le slot 100 porte le joueur A (index 0, xuid 111) en generation 1
// [1 s..4 s], puis le joueur B (index 1, xuid 222) en generation 2, ne a 12 s, [14 s..18 s].
func entreeDeuxJoueursUnSlot() IdentityInput {
	in := entreeSlotRecycle(1)
	in.PlayerIndices.ByXUID = map[uint64]int{111: 0, 222: 1, 333: 2}
	return in
}

// TestTirsEtLancers_AttribuesALaVieALInstant : a 16 s, le slot 100 est le corps de B.
//
// ROUGE AVANT : le pont aplati donnait le slot 100 a son PREMIER occupant (A) pour tout le film —
// le tir de B restait sans slot, celui de A (qui n'a plus de corps) etait pose sur la position de
// B, le lancer de B n'avait pas d'auteur, et le releve de l'unite 100 comptait un desaccord.
//
// MUTATION : faire lire a `slotFor` le pont par slot (`occupantsPlats(o.plat)`) -> rouge.
func TestTirsEtLancers_AttribuesALaVieALInstant(t *testing.T) {
	in := entreeDeuxJoueursUnSlot()
	reg := BuildIdentityRegistry(context.Background(), in)
	occ := reg.Occupants()
	const t16 = 16_000_000
	origin, step := in.Clock.OriginUS, in.Clock.StepUS

	tirB := fireAt(t16, 1, 90)
	tirB.HasShooter = true
	shots, _, _ := buildShots(in.Positions, []grammar.FireEvent{tirB}, origin, step, occ)
	if len(shots) != 1 || shots[0].Slot != 100 {
		t.Fatalf("tir de B a 16 s : %+v, attendu pose sur le slot 100 (son corps a l'instant)", shots)
	}
	tirA := fireAt(t16, 0, 90)
	tirA.HasShooter = true
	shots, _, cov := buildShots(in.Positions, []grammar.FireEvent{tirA}, origin, step, occ)
	if len(shots) != 0 || cov.NoSlot != 1 {
		t.Fatalf("tir de A a 16 s : %+v (couverture %+v), attendu SANS slot — le slot 100 est le "+
			"corps de B a cet instant", shots, cov)
	}

	lancer := grammar.GrenadeThrow{TimestampUS: t16, FilmIndex: 1, TypeID: grammar.GrenadeFragmentation}
	gren, _ := buildGrenades(in.Positions, []grammar.GrenadeThrow{lancer}, origin, step, occ, nil, nil)
	if len(gren) != 1 || gren[0].Slot != 100 || gren[0].Src != GrenadeSrcBiped {
		t.Fatalf("lancer de B a 16 s : %+v, attendu pose sur le corps de B (slot 100, src biped)", gren)
	}

	releve := grammar.FireEvent{TimestampUS: t16, FilmIndex: 1, HasShooter: true,
		Unit: grammar.UnitRef{Present: true, Slot: 100, Gen: 2}}
	if m := mesurerIndexDeTireur([]grammar.FireEvent{releve}, occ); m.total != 1 || m.accord != 1 {
		t.Fatalf("releve de l'unite 100 a 16 s : accord %d sur %d, attendu 1 sur 1", m.accord, m.total)
	}

	if s := occ.slotsDe(1, t16); len(s) != 1 || s[0] != 100 {
		t.Fatalf("slots de B a 16 s : %v, attendu [100]", s)
	}
	if s := occ.slotsDe(0, t16); len(s) != 0 {
		t.Fatalf("slots de A a 16 s : %v, attendu aucun", s)
	}
	if s := occ.slotsDe(0, 2_000_000); len(s) != 1 || s[0] != 100 {
		t.Fatalf("slots de A a 2 s : %v, attendu [100]", s)
	}
}

// TestOccupants_SAbstiennentSurUnSlotAmbigu : sans records de creation qui departagent, un slot que
// deux joueurs nommes se partagent ne rend AUCUN index — la meme abstention que les equipes
// (`TeamCoverage.TracksSlotAmbiguous`) et que [IdentityRegistry.PontDeSlot].
func TestOccupants_SAbstiennentSurUnSlotAmbigu(t *testing.T) {
	occ := occupantsDesSlots{plat: map[uint32]int{100: 0, 200: 2}, ambigus: map[uint32]bool{100: true}}
	if _, ok := occ.indexA(100, 1); ok {
		t.Fatal("slot ambigu : un index a ete servi")
	}
	if pi, ok := occ.indexA(200, 1); !ok || pi != 2 {
		t.Fatalf("slot 200 : %d %v, attendu 2", pi, ok)
	}
}
