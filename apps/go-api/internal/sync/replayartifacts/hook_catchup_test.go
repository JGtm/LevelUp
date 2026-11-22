package replayartifacts

import (
	"testing"
)

// Les bornes du cycle suivent la capacité FilmCatchUp du hook : garde-fou de 200 rejeux et lecture du retard
// sur 2000 matchs sous la capacité ; 5 rejeux et 64 matchs hors d'elle (hook nil compris).
func TestHook_BornesSuiventLaCapaciteFilmCatchUp(t *testing.T) {
	cas := []struct {
		nom     string
		hook    *Hook
		perCyc  int
		horizon int
	}{
		{"sous FilmCatchUp", &Hook{FilmCatchUp: true}, 200, 2000},
		{"hors FilmCatchUp", &Hook{}, 5, 64},
		{"hook nil", nil, 5, 64},
	}
	for _, c := range cas {
		t.Run(c.nom, func(t *testing.T) {
			if got := c.hook.PerCycle(); got != c.perCyc {
				t.Errorf("PerCycle = %d, attendu %d", got, c.perCyc)
			}
			if got := c.hook.Horizon(); got != c.horizon {
				t.Errorf("Horizon = %d, attendu %d", got, c.horizon)
			}
		})
	}
	if RejeuxParCycleHeberge != 5 || BacklogHorizonHeberge != 64 || maxPerCycle != 200 || BacklogHorizon != 2000 {
		t.Error("valeurs nommées des bornes changées : D-22 fixe 5 rejeux et 64 matchs hors capacité, 200 et 2000 dessous")
	}
}

// Des Deps sans bornes (tests) prennent celles de la capacité ; des Deps du câblage prennent celles du hook.
func TestDeps_BornesParDefautEtDuCablage(t *testing.T) {
	if got := perCycle(Deps{}); got != maxPerCycle {
		t.Errorf("perCycle(zéro) = %d, attendu %d", got, maxPerCycle)
	}
	if got := horizonDuRetard(Deps{}); got != BacklogHorizon {
		t.Errorf("horizonDuRetard(zéro) = %d, attendu %d", got, BacklogHorizon)
	}
	h := &Hook{}
	d := Deps{PerCycle: h.PerCycle(), Horizon: h.Horizon()}
	if perCycle(d) != RejeuxParCycleHeberge || horizonDuRetard(d) != BacklogHorizonHeberge {
		t.Errorf("Deps du câblage hébergé : plafond %d et horizon %d", perCycle(d), horizonDuRetard(d))
	}
	// Une place de rattrapage de 5 hors capacité, en placement ouvrier sans match inséré.
	if got := placeDuRattrapage(d, 0); got != RejeuxParCycleHeberge {
		t.Errorf("placeDuRattrapage hébergé = %d, attendu %d", got, RejeuxParCycleHeberge)
	}
}
