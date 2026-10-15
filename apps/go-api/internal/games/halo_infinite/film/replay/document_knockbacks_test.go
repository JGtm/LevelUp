package replay

// document_knockbacks_test.go — LE CALQUE DES POUSSÉES : ce qui est publié, ce qui est écarté, et
// l'identité de sa couverture.

import (
	"testing"

	"levelup/go-api/internal/games/halo_infinite/film/types"
)

// TestLaCouvertureDesPousseesSeReferme : chaque poussée lue finit dans UNE case — publiée, non
// appariée, sans identité, sans impulsion, avant l'origine — et la somme rend les 104 lus.
// MUTATIONS — la poussée sans impulsion publiée, ou écartée sans être comptée : ROUGES.
func TestLaCouvertureDesPousseesSeReferme(t *testing.T) {
	const origin, step = 1_000_000, 100_000
	tracks := []Track{{Slot: 519}, {Slot: 513}}
	imp := [3]float32{2.95, 4.8, 2.06}
	evts := []types.Knockback{
		{TimestampUS: 500_000, Pushed: types.LifeKey{Slot: 519}, HasImpulse: true, Impulse: imp},
		{TimestampUS: 2_000_000, Pushed: types.LifeKey{Slot: 777}, HasImpulse: true, Impulse: imp},
		{TimestampUS: 3_000_000, Pushed: types.LifeKey{Slot: 519}},
		{TimestampUS: 4_000_000, Pushed: types.LifeKey{Slot: 519}, Author: types.LifeKey{Slot: 513},
			HasAuthor: true, HasImpulse: true, Impulse: imp},
		{TimestampUS: 5_000_000, Pushed: types.LifeKey{Slot: 519}, HasImpulse: true, Impulse: imp},
	}
	st := types.KnockbackStats{Read104: 6, Read105: 6, Paired: 5, Unpaired: 1}
	out, cov := buildKnockbacks(evts, st, tracks, origin, step)
	if cov.Published != 2 || cov.BeforeOrigin != 1 || cov.NoIdentity != 1 || cov.WithoutImpulse != 1 ||
		cov.WithoutAuthor != 1 {
		t.Fatalf("couverture %+v", cov)
	}
	if got := cov.Published + cov.Unpaired + cov.NoIdentity + cov.WithoutImpulse + cov.BeforeOrigin; got != cov.Read104 {
		t.Errorf("identite : %d cases contre %d 104 lus (%+v)", got, cov.Read104, cov)
	}
	if len(out) != 2 || out[0].T != 30 || out[0].By == nil || *out[0].By != 513 || out[0].VX == nil ||
		out[1].By != nil || out[1].VY == nil {
		t.Errorf("poussees publiees %+v", out)
	}
}
