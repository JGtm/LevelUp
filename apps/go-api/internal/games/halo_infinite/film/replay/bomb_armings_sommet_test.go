package replay

// bomb_armings_sommet_test.go — LE PREDICAT D ARMEMENT, SUR TAMPON SYNTHETIQUE (lot 5.1.8).
//
// UN ARMEMENT EST UNE MONTEE QUI ATTEINT LE PLEIN, QUOI QU IL ARRIVE APRES. Ce test fige les
// trois formes qui decident, et il MORD : avant le 2026-09-19 le predicat exigeait que le segment
// FINISSE a son sommet, et le premier cas — celui que le portage de `ti=12` a rendu majoritaire —
// rendait ZERO armement sur `c75f33b8` la ou le film en porte quatre.

import (
	"testing"

	"levelup/go-api/internal/games/halo_infinite/film/internal/grammar"
	"levelup/go-api/internal/games/halo_infinite/film/types"
)

// sommetLectures fabrique une serie de lectures d un slot, un echantillon toutes les 100 ms.
func sommetLectures(slot uint32, depart int32, quanta ...uint8) []types.NavpointRadialRead {
	out := make([]types.NavpointRadialRead, 0, len(quanta))
	for i, q := range quanta {
		out = append(out, types.NavpointRadialRead{
			Slot: slot, TMS: depart + int32(i)*100, Q: q, Chained: true,
		})
	}
	return out
}

// sommetArmements rend le nombre d armements que le classement retient.
func sommetArmements(t *testing.T, reads []types.NavpointRadialRead) int {
	t.Helper()
	cov := &BombArmingsCoverage{}
	armed, _ := classifyBombSegments(grammar.NavpointSegments(reads), bombReadsBySlot(reads), cov)
	return len(armed)
}

// TestArmementEstUneMonteeQuiAtteintLePlein : les trois formes, et ce qu elles rendent.
func TestArmementEstUneMonteeQuiAtteintLePlein(t *testing.T) {
	// LA FORME DU FILM (`c75f33b8`) : la montee atteint 254, PUIS l anneau redescend a son
	// plancher de cycle et s y tient. L armement compte quand meme, et c est tout l objet du lot.
	monteePuisRecharge := sommetLectures(1459, 1000,
		127, 160, 200, 240, 254, 254, 200, 160, 127, 127, 127)
	// La meme montee, arretee SOUS le plein : ce n est pas un armement.
	monteeSousLePlein := sommetLectures(1459, 1000, 127, 160, 200, 230, 240, 200, 127)
	// Deux montees separees par un trou STRICTEMENT plus long que NavpointRiseMaxGapMS : la
	// premiere finit a 2 000 ms, la seconde part a 3 000 ms.
	deuxMontees := make([]types.NavpointRadialRead, 0, 19)
	deuxMontees = append(deuxMontees, monteePuisRecharge...)
	deuxMontees = append(deuxMontees,
		sommetLectures(1459, 3000, 127, 160, 200, 240, 254, 254, 200, 127)...)

	for _, c := range []struct {
		nom   string
		reads []types.NavpointRadialRead
		want  int
	}{
		{"montee qui atteint le plein puis continue", monteePuisRecharge, 1},
		{"montee qui s arrete sous le plein", monteeSousLePlein, 0},
		{"deux montees", deuxMontees, 2},
	} {
		t.Run(c.nom, func(t *testing.T) {
			if got := sommetArmements(t, c.reads); got != c.want {
				t.Errorf("%s : %d armement(s), %d attendu(s).\n"+
					"Un armement est une montee qui ATTEINT le plein (%d), quoi qu il arrive "+
					"apres ; l instant retenu est le PREMIER echantillon au plein.",
					c.nom, got, c.want, bombArmedFullQuantum)
			}
		})
	}
}

// TestArmementDateAuPremierPlein : l instant publie est le PREMIER echantillon au plein, pas le
// dernier echantillon du segment. C est ce qui rend a `c75f33b8` les instants d avant le portage
// de `ti=12`, a la milliseconde.
func TestArmementDateAuPremierPlein(t *testing.T) {
	reads := sommetLectures(1459, 1000, 127, 160, 200, 240, 254, 254, 200, 127)
	cov := &BombArmingsCoverage{}
	armed, _ := classifyBombSegments(grammar.NavpointSegments(reads), bombReadsBySlot(reads), cov)
	if len(armed) != 1 {
		t.Fatalf("%d armement(s), 1 attendu", len(armed))
	}
	// Le cinquieme echantillon (index 4) est le premier a 254 : 1000 + 4 x 100.
	if armed[0].EndMS != 1400 {
		t.Errorf("instant d armement = %d ms, 1400 attendu (le PREMIER echantillon au plein).\n"+
			"Le dernier echantillon du segment est a 1700 ms : le retenir daterait l armement sur "+
			"la recharge qui le suit.", armed[0].EndMS)
	}
}
