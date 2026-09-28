package objectives

import (
	"sort"
	"testing"

	"levelup/go-api/internal/games/halo_infinite/film/types"
)

// slotidentity_deaths_serie_test.go — LE PONT PAR INSTANTS DE MORT LIT LA SERIE PUBLIEE (lot J8.5
// du plan de suite d audit, constat FO-3).
//
// Le pont deroulait le compteur de morts avec ses PROPRES gardes (slot de joueur, valeur dans
// [0, 1000]) ; la serie que le document publie passe par d autres filtres (manche confrontee au
// temps, manches fantomes, plus longue sous-suite non decroissante, borne par pas). Une emission
// que la serie publiee jette pouvait donc nommer un joueur dans le pont — ou, comme ici, lui
// fabriquer 49 morts au meme instant, qui noient ses coincidences et le font taire.
//
// L INVARIANT TESTE EST L ACCORD, PAS UNE VALEUR : pour chaque slot (et chaque manche), le nombre
// d instants que le pont deroule vaut le dernier point de la serie publiee.

// avecEmissionAberrante ajoute au slot 10 de la manche `round` une emission de morts aberrante
// (50) entre deux emissions vraies : la plus longue sous-suite non decroissante l ecarte, et son
// pas (> maxUnrollPerStep) aussi.
func avecEmissionAberrante(recs []types.StatRecord, round, timeMS int) []types.StatRecord {
	out := append([]types.StatRecord(nil), recs...)
	out = append(out, types.StatRecord{TimeMS: timeMS, Slot: 10, Round: round,
		Comps: map[int]types.StatValue{coreKillsComp: {A: 1, B: 50}}})
	sort.SliceStable(out, func(i, j int) bool { return out[i].TimeMS < out[j].TimeMS })
	return out
}

// TestLePontPlatParMortsLitLaSeriePubliee : le pont PLAT (mono-manche) deroule exactement la
// serie totale publiee.
//
// ROUGE OBSERVE AVANT LE LOT (gardes propres du pont, `v.B < 0 || v.B > 1000`) : 50 instants
// pour 3 morts sur le slot 10.
func TestLePontPlatParMortsLitLaSeriePubliee(t *testing.T) {
	recs := avecEmissionAberrante([]types.StatRecord{
		recKDA(1000, 10, 0, 1, 1, 0), recKDA(2000, 10, 0, 1, 2, 0), recKDA(3000, 10, 0, 1, 3, 0),
		recKDA(1200, 12, 0, 1, 1, 0),
	}, 0, 1500)
	serie := SeriesTotal(recs, DeathsComponent, false, nil)
	got := deathProgressions(recs, nil)
	for slot, pts := range serie {
		if want := int(pts[len(pts)-1].Value); len(got[slot]) != want {
			t.Errorf("slot %d : le pont deroule %d instant(s), la serie publiee en compte %d", slot,
				len(got[slot]), want)
		}
	}
	if want := []int{1000, 2000, 3000}; len(got[10]) != 3 || got[10][0] != want[0] || got[10][2] != want[2] {
		t.Fatalf("slot 10 : %v, attendu %v — l emission aberrante a ete deroulee", got[10], want)
	}
}

// TestLePontParMancheLitLaSeriePubliee : MEME accord, manche par manche, sur un film a deux
// manches reelles — la serie par manche publiee, non cumulee.
func TestLePontParMancheLitLaSeriePubliee(t *testing.T) {
	recs := avecEmissionAberrante(deuxManchesTroisSlots(), 1, 1950)
	if len(realRoundsSorted(recs)) != 2 {
		t.Fatalf("le scenario doit porter deux manches reelles, %v", realRoundsSorted(recs))
	}
	parManche := SeriesByRound(recs, DeathsComponent, false, nil)
	deroule := deathProgressionsByRound(recs, nil)
	for _, round := range realRoundsSorted(recs) {
		got := deroule[round]
		for slot, byRound := range parManche {
			pts := byRound[round]
			if len(pts) == 0 {
				continue
			}
			if want := int(pts[len(pts)-1].Value); len(got[slot]) != want {
				t.Errorf("manche %d slot %d : le pont deroule %d instant(s), la serie publiee en "+
					"compte %d", round, slot, len(got[slot]), want)
			}
		}
	}
}
