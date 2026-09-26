package replay

// flag_carries_rounds_test.go — LES FERMOIRS PAR SLOT SONT INDEXES PAR (MANCHE, SLOT) (constat
// RB1-6 de l audit du 2026-09-24, lot J9.3).
//
// LE DEFAUT : la capture et la prise suivante « du meme slot » bornaient un portage sans regarder
// la manche. Or le slot statborg est REATTRIBUE d une manche a l autre (cf. `RoundIdentity`) : la
// capture ou la prise d un AUTRE joueur, en manche suivante, fermait le portage du joueur de la
// manche precedente — et la capture lui etait creditee.
//
// La fixture est celle de l identite par manche (`manches_segments_test.go`) : le slot 22 est
// « 1022 » en manche 0 et « 9022 » en manche 1, qui commence a 298 s.

import (
	"testing"

	"levelup/go-api/internal/games/halo_infinite/film/internal/facts/objectives"
)

// flagRoundsScan : « 1022 » vole a 150 s (manche 0) sur le slot 22 ; `suite` sont les evenements
// du meme slot en manche 1.
func flagRoundsScan(t *testing.T, suite ...objectives.NamedEvent) FlagCarryScan {
	t.Helper()
	recs, deaths := identiteFixture()
	ri := objectives.ResolveRoundIdentity(recs, deathInstantsOf(deaths))
	if ri.At(identiteSlotReattribue, 150_000) != identiteXUIDManche0 ||
		ri.At(identiteSlotReattribue, identiteDebutR1+1_000) != identiteXUIDManche1 {
		t.Fatalf("la fixture ne reattribue plus le slot %d d une manche a l autre", identiteSlotReattribue)
	}
	evs := append([]objectives.NamedEvent{
		{TimeMS: 150_000, Slot: identiteSlotReattribue, Stat: objectives.StatFlagSteals},
	}, suite...)
	return FlagCarryScan{
		Scanned: true, Signals: flagTestSignals(), Events: evs, Identity: ri,
		Spawns: flagInvariantSpawns(),
		TeamOf: map[string]int{identiteXUIDManche0: 0, identiteXUIDManche1: 1},
	}
}

// TestFermoirsDuSlotNeFranchissentPasLaManche — LE POINT DU LOT. Ni la capture ni la prise du
// slot 22 en manche 1 ne sont des faits du portage de « 1022 » en manche 0 : il ne doit etre ni
// capture, ni borne par elles.
//
// MUTATION : indexer les deux fermoirs par le slot seul (retirer la manche de `flagRoundSlot`)
// rougit les deux cas.
func TestFermoirsDuSlotNeFranchissentPasLaManche(t *testing.T) {
	apres := identiteDebutR1 + 1_000
	for _, cas := range []struct {
		nom   string
		suite []objectives.NamedEvent
	}{
		{"capture du slot en manche suivante", []objectives.NamedEvent{
			{TimeMS: apres, Slot: identiteSlotReattribue, Stat: objectives.StatFlagCaptures}}},
		{"prise du slot en manche suivante", []objectives.NamedEvent{
			{TimeMS: apres, Slot: identiteSlotReattribue, Stat: objectives.StatFlagGrabs}}},
	} {
		t.Run(cas.nom, func(t *testing.T) {
			scan := flagRoundsScan(t, cas.suite...)
			ops := flagOpenings(scan.Events, scan.Identity)
			raws := boundFlagCarries(ops, scan, flagTestCtx(nil, nil, 4000))
			if len(raws) == 0 || raws[0].xuid != identiteXUIDManche0 {
				t.Fatalf("portages %+v : le premier doit etre celui de %s", raws, identiteXUIDManche0)
			}
			if r := raws[0]; r.t1 <= int64(apres) || r.captured || r.closed {
				t.Errorf("portage de %s borne a %d (capture=%v, ferme=%v) par un fait de la manche "+
					"suivante — attendu : aucun fermoir de manche 0, fin de l axe", r.xuid, r.t1,
					r.captured, r.closed)
			}
		})
	}
}

// TestFermoirsDuSlotDansLaMemeManche — LE TEMOIN : dans la MEME manche, la capture du slot ferme
// toujours le portage, et le capture.
func TestFermoirsDuSlotDansLaMemeManche(t *testing.T) {
	scan := flagRoundsScan(t, objectives.NamedEvent{
		TimeMS: 160_000, Slot: identiteSlotReattribue, Stat: objectives.StatFlagCaptures})
	raws := boundFlagCarries(flagOpenings(scan.Events, scan.Identity), scan, flagTestCtx(nil, nil, 4000))
	if len(raws) != 1 || raws[0].t1 != 160_000 || !raws[0].captured {
		t.Fatalf("portages %+v : attendu un portage capture a 160 000 ms", raws)
	}
}
