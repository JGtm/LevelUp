package replay

import (
	"sort"
	"testing"

	"levelup/go-api/internal/analysis/objectiveevents"
)

// vip_crown_rounds_test.go — LA PREUVE DE LA CORRECTION MULTI-MANCHE, AU NIVEAU DU CALQUE.
//
// Le pont d'identite est passe PAR MANCHE (lot porteur Oddball). Ce test montre ce que cela
// change pour la COURONNE VIP quand le SLOT est reattribue d'une manche a l'autre : une selection
// de la manche 1 doit nommer le joueur de la MANCHE 1, pas celui de la manche 0. Le pont plat, lui,
// nomme le meme joueur aux deux (il ne voit que la premiere manche).

// twoRoundVipRecs fabrique un film synthetique a deux manches : le slot 22 est le joueur "A" en
// manche 0 puis "B" en manche 1 (le compteur de morts `comp 2 B` repart de zero par manche, et le
// score de mode `comp 0 A` fait reconnaitre les deux manches).
func twoRoundVipRecs() ([]objectiveevents.StatRecord, []objectiveevents.DeathInstant) {
	rec := func(t, slot, round, comp int, a, b int64) objectiveevents.StatRecord {
		return objectiveevents.StatRecord{TimeMS: t, Slot: slot, Round: round,
			Comps: map[int]objectiveevents.StatValue{comp: {A: a, B: b}}}
	}
	recs := []objectiveevents.StatRecord{
		rec(900, 22, 0, 0, 10, 0), rec(1900, 22, 0, 0, 20, 0), rec(2900, 22, 0, 0, 30, 0),
		rec(10900, 22, 1, 0, 10, 0), rec(11900, 22, 1, 0, 20, 0), rec(12900, 22, 1, 0, 30, 0),
		rec(1000, 22, 0, 2, 0, 1), rec(2000, 22, 0, 2, 0, 2), rec(3000, 22, 0, 2, 0, 3),
		rec(11000, 22, 1, 2, 0, 1), rec(12000, 22, 1, 2, 0, 2), rec(13000, 22, 1, 2, 0, 3),
	}
	sort.SliceStable(recs, func(i, j int) bool { return recs[i].TimeMS < recs[j].TimeMS })
	deaths := []objectiveevents.DeathInstant{
		{XUID: "A", TimeMS: 1000}, {XUID: "A", TimeMS: 2000}, {XUID: "A", TimeMS: 3000},
		{XUID: "B", TimeMS: 11000}, {XUID: "B", TimeMS: 12000}, {XUID: "B", TimeMS: 13000},
	}
	return recs, deaths
}

func TestVipCrownRoundReassignedSlot(t *testing.T) {
	recs, deaths := twoRoundVipRecs()
	// Deux selections VIP du meme slot : une par manche.
	events := []objectiveevents.NamedEvent{
		{Slot: 22, TimeMS: 2000, Stat: objectiveevents.StatVipSelected},
		{Slot: 22, TimeMS: 12000, Stat: objectiveevents.StatVipSelected},
	}

	// PAR MANCHE (production) : la selection de la manche 1 nomme B.
	byRound := vipReconstructPeriods(events, objectiveevents.ResolveRoundIdentity(recs, deaths), nil, 20000)
	if len(byRound) != 2 {
		t.Fatalf("periodes par manche : %d, attendu 2", len(byRound))
	}
	if byRound[0].xuid != "A" {
		t.Errorf("periode manche 0 = %q, attendu \"A\"", byRound[0].xuid)
	}
	if byRound[1].xuid != "B" {
		t.Errorf("periode manche 1 = %q, attendu \"B\" (slot reattribue)", byRound[1].xuid)
	}

	// PONT PLAT (avant le lot) : les deux selections nommeraient le meme joueur — la manche 1
	// serait FAUSSE. On le montre pour que la correction soit lisible.
	flat := objectiveevents.FlatRoundIdentity(objectiveevents.SlotIdentityByDeaths(recs, deaths))
	plat := vipReconstructPeriods(events, flat, nil, 20000)
	if plat[1].xuid == "B" {
		t.Errorf("le pont plat aurait deja nomme B en manche 1 — la fixture ne prouve rien")
	}
	if plat[1].xuid != plat[0].xuid {
		t.Errorf("pont plat : manches 0 et 1 devraient nommer le meme joueur (%q vs %q)",
			plat[0].xuid, plat[1].xuid)
	}
}
