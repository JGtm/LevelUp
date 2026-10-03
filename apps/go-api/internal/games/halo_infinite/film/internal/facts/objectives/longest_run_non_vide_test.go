package objectives

import (
	"testing"

	"levelup/go-api/internal/games/halo_infinite/film/types"
)

// longest_run_non_vide_test.go — LA PREUVE QUI A RETIRE DEUX BRANCHES MORTES (lot J8.6 du plan de
// suite d audit, constat FO-4).
//
// `cumulateRounds` et `SeriesByRound` sautaient une manche quand la plus longue sous-suite de ses
// emissions etait VIDE — le repli `repli_manche_du_slot_sautee`. Une manche n entre dans leurs
// tables que par une emission, donc sa suite n est jamais vide, et [longestRun] (suivi de
// [boundedSeries] pour un compteur unitaire) rend au moins un point d une suite non vide : la
// branche ne pouvait pas se prendre. Ce test fige la propriete sur laquelle repose son retrait.
func TestLongestRunNeRendJamaisVidePourUneSuiteNonVide(t *testing.T) {
	motifs := [][]int64{
		{0}, {5}, {-3}, {3, 2, 1}, {1, 1, 1}, {1, 50, 2, 3}, {400_000_000, 1},
		{10, 9, 8, 7, 6, 5, 4, 3, 2, 1, 0}, {0, 17, 0, 17}, {2, 2, 1, 1, 0, 0},
	}
	for _, valeurs := range motifs {
		pts := make([]types.ScorePoint, 0, len(valeurs))
		for i, v := range valeurs {
			pts = append(pts, types.ScorePoint{TimeMS: 1000 * i, Slot: 10, Value: v})
		}
		for _, strict := range []bool{false, true} {
			kept := longestRun(pts, strict)
			if len(kept) == 0 {
				t.Fatalf("longestRun(%v, strict=%v) est vide — la branche retiree redevient atteignable",
					valeurs, strict)
			}
			if b := boundedSeries(kept); len(b) != len(kept) {
				t.Fatalf("boundedSeries change la cardinalite de %v : %d -> %d", valeurs, len(kept), len(b))
			}
		}
	}
}
