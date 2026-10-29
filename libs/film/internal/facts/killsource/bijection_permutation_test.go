package killsource

// bijection_permutation_test.go — LA PERMUTATION A GRAINE FIXE FAIT PARTIE DE LA SORTIE (lot J12.1,
// 2026-09-30, PLAN_SUITE_AUDIT_DECODEUR_FILM_2026-09-25, DT-10).
//
// [solveBijection] tire ses redemarrages par `newRNG().Perm(n)` : la suite des permutations decide
// de la bijection publiee quand deux optimums locaux ont le meme score (le departage `permLess` ne
// voit que les permutations VISITEES). Changer de generateur (`math/rand` -> `math/rand/v2`, piege
// n°4 de l audit) ou de graine change donc la sortie sans qu aucun autre test ne le voie. Ce test
// fige, pour plusieurs tailles, la premiere permutation et l empreinte FNV-64a des
// `BijectionRestarts` (40) premieres, valeurs relevees sur l etat du 2026-09-30 (tete `33fa8abeb`).

import (
	"fmt"
	"hash/fnv"
	"slices"
	"testing"
)

func TestBijectionPermutationGraineFixe(t *testing.T) {
	restarts := DefaultOptions().BijectionRestarts
	if restarts != 40 {
		t.Fatalf("BijectionRestarts = %d, 40 attendu : les empreintes ci-dessous valent pour 40 tirages", restarts)
	}
	attendus := []struct {
		n         int
		premiere  []int
		empreinte uint64
	}{
		{1, []int{0}, 0xd6b1f06696a4a5fd},
		{2, []int{1, 0}, 0x4ce39aee2a7f9bed},
		{5, []int{1, 0, 3, 4, 2}, 0x8a7121b69cd53f2d},
		{8, []int{1, 5, 3, 4, 7, 0, 2, 6}, 0x8fb281631e78438d},
		{13, []int{1, 9, 3, 4, 7, 0, 11, 8, 6, 5, 12, 10, 2}, 0x8067ac27e0cd14ad},
		{24, []int{1, 9, 3, 14, 7, 18, 11, 23, 17, 15, 12, 10, 19, 5, 4, 13, 2, 6, 0, 20, 16, 8, 21, 22}, 0xb4c2a00e32067ac3},
	}
	for _, a := range attendus {
		rng := newRNG()
		h := fnv.New64a()
		var premiere []int
		for i := range restarts {
			p := rng.Perm(a.n)
			if i == 0 {
				premiere = p
			}
			for _, v := range p {
				fmt.Fprintf(h, "%d,", v)
			}
			fmt.Fprint(h, ";")
		}
		if !slices.Equal(premiere, a.premiere) {
			t.Errorf("n=%d : premiere permutation %v, figee %v — generateur ou graine change (DT-10)", a.n, premiere, a.premiere)
		}
		if got := h.Sum64(); got != a.empreinte {
			t.Errorf("n=%d : empreinte des %d permutations %#x, figee %#x — generateur ou graine change (DT-10)",
				a.n, restarts, got, a.empreinte)
		}
	}
}
