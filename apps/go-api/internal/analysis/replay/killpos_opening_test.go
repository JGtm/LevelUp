package replay

// killpos_opening_test.go — le décalage d'entame : il décale, il ne place pas.
//
// Trois choses à épingler, et une seule est arithmétique : (1) le décalage s'applique à
// TOUS les couples sans muter l'entrée ; (2) composé avec `BuildKillPositions`, il rend bien
// la position d'AVANT le coup fatal ; (3) un instant qui passe avant l'origine du film ne
// rend AUCUNE position — jamais une position de réapparition présentée comme une entame.

import (
	"testing"

	"levelup/go-api/internal/analysis/filmdec"
)

func TestShiftKillRefsDecaleSansMuterLEntree(t *testing.T) {
	kills := []KillRef{
		{KillerXUID: 111, VictimXUID: 222, TimeMS: 10_000},
		{KillerXUID: 222, VictimXUID: 111, TimeMS: 42_500},
	}
	avant := append([]KillRef(nil), kills...)
	got := ShiftKillRefs(kills, -OpeningLeadMS)
	if len(got) != 2 {
		t.Fatalf("2 couples attendus, obtenu %d", len(got))
	}
	for i, want := range []int64{8_500, 41_000} {
		if got[i].TimeMS != want {
			t.Errorf("couple %d : instant %d, attendu %d", i, got[i].TimeMS, want)
		}
		if got[i].KillerXUID != avant[i].KillerXUID || got[i].VictimXUID != avant[i].VictimXUID {
			t.Errorf("couple %d : les identités ne doivent pas bouger : %+v", i, got[i])
		}
	}
	for i := range avant {
		if kills[i] != avant[i] {
			t.Fatalf("entrée MUTÉE au rang %d : %+v -> %+v", i, avant[i], kills[i])
		}
	}
	if ShiftKillRefs(nil, -OpeningLeadMS) != nil {
		t.Error("une entrée vide rend nil, pas une tranche de longueur 0")
	}
}

// TestShiftKillRefsCompositionRendLaPositionDEntame : la composition du plan, sur des
// trajectoires où la réponse est connue d'avance — le tueur avance de 20 m sur son adversaire
// pendant le temps-pour-tuer, l'entame doit rendre la position D'AVANT.
func TestShiftKillRefsCompositionRendLaPositionDEntame(t *testing.T) {
	const mortMS = 30_000
	// posAt(slot, tUS, x, y, cap) : le dernier nombre est un CAP DE VISÉE, pas une altitude —
	// seul l'écart en X porte la mesure ici.
	pos := []filmdec.BipedPosition{
		// à T − 1,5 s
		posAt(1, uint64(mortMS-OpeningLeadMS)*1000, 0, 0, 0),
		posAt(2, uint64(mortMS-OpeningLeadMS)*1000, 20, 0, 0),
		// au coup fatal
		posAt(1, mortMS*1000, 19, 0, 0),
		posAt(2, mortMS*1000, 20, 0, 0),
	}
	slotXUID := map[uint32]uint64{1: 111, 2: 222}
	kills := []KillRef{{KillerXUID: 111, VictimXUID: 222, TimeMS: mortMS}}

	fatal, _ := BuildKillPositions(pos, slotXUID, kills, 0)
	entame, rep := BuildKillPositions(pos, slotXUID, ShiftKillRefs(kills, -OpeningLeadMS), 0)
	if len(fatal) != 1 || len(entame) != 1 || rep.Both != 1 {
		t.Fatalf("les deux placements devaient aboutir : fatal %+v, entame %+v (%+v)",
			fatal, entame, rep)
	}
	if fatal[0].Killer.X != 19 {
		t.Errorf("position fatale du tueur : X = %v, attendu 19", fatal[0].Killer.X)
	}
	if entame[0].Killer.X != 0 {
		t.Errorf("position d'entame du tueur : X = %v, attendu 0", entame[0].Killer.X)
	}
	if entame[0].TimeMS != mortMS-OpeningLeadMS {
		t.Errorf("l'instant porté par l'entame vaut %d, attendu %d",
			entame[0].TimeMS, mortMS-OpeningLeadMS)
	}
}

// TestShiftKillRefsInstantNegatifNeDonneAucunePosition : une mort trop proche de l'origine
// n'a pas d'entame. Ce test EXISTE POUR ÊTRE FRAGILE : si un jour `positionOf` acceptait un
// instant décalé avant l'origine, il tomberait — plutôt que de laisser publier la position de
// réapparition du début de match comme si c'était une entame.
func TestShiftKillRefsInstantNegatifNeDonneAucunePosition(t *testing.T) {
	pos := []filmdec.BipedPosition{
		posAt(1, 200_000, 1, 2, 3),
		posAt(2, 200_000, 4, 5, 6),
	}
	slotXUID := map[uint32]uint64{1: 111, 2: 222}
	kills := []KillRef{{KillerXUID: 111, VictimXUID: 222, TimeMS: 200}} // 200 ms < 1 500 ms
	decales := ShiftKillRefs(kills, -OpeningLeadMS)
	if decales[0].TimeMS != -1_300 {
		t.Fatalf("instant décalé attendu -1 300 ms, obtenu %d", decales[0].TimeMS)
	}
	got, rep := BuildKillPositions(pos, slotXUID, decales, 0)
	if len(got) != 0 || rep.Dropped != 1 {
		t.Fatalf("aucune entame ne devait être écrite : %+v (%+v)", got, rep)
	}
}
