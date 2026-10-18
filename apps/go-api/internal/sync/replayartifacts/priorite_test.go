package replayartifacts

import (
	"testing"

	"levelup/go-api/internal/replaybuild"
)

func TestPlaceDuRattrapage(t *testing.T) {
	cas := []struct {
		nom       string
		placement replaybuild.Placement
		inseres   int
		attendu   int
	}{
		{"local avec insérés : aucune place", replaybuild.PlacementLocal, 2, 0},
		{"local sans insertion : lot plein", replaybuild.PlacementLocal, 0, maxPerCycle},
		{"ouvrier avec insérés : le solde du lot", replaybuild.PlacementWorker, 2, maxPerCycle - 2},
	}
	for _, c := range cas {
		if got := placeDuRattrapage(Deps{Placement: c.placement}, c.inseres); got != c.attendu {
			t.Errorf("%s : place = %d, attendu %d", c.nom, got, c.attendu)
		}
	}
}
