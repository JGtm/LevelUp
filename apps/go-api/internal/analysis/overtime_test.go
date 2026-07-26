package analysis

import "testing"

// Valeurs RÉELLES relevées pendant l'investigation (1793 matchs) sur les
// variantes à 720 s de temps réglementaire.
func TestComputeOvertime_RealValues(t *testing.T) {
	const reg = 720
	cases := []struct {
		name        string
		elapsed     int
		wantFlag    bool
		wantSeconds int
	}{
		// Terminés dans le temps (+ bruit d'horloge de fin de match).
		{"718s — juste sous le règlement", 718, false, 0},
		{"721s — bruit de fin de match", 721, false, 0},
		{"728s — bruit de fin de match", 728, false, 0},
		// Prolongations réelles.
		{"763s — prolongation courte", 763, true, 43},
		{"774s — prolongation", 774, true, 54},
		{"797s — prolongation", 797, true, 77},
		{"990s — prolongation longue", 990, true, 270},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			flag, secs := ComputeOvertime(c.elapsed, reg)
			if flag != c.wantFlag || secs != c.wantSeconds {
				t.Errorf("ComputeOvertime(%d, %d) = (%v, %d), want (%v, %d)",
					c.elapsed, reg, flag, secs, c.wantFlag, c.wantSeconds)
			}
		})
	}
}

// Bornes exactes autour du seuil 720 + OvertimeMarginSeconds (= 760).
// La marge est INCLUSE : 760 s reste du temps réglementaire.
func TestComputeOvertime_Boundaries(t *testing.T) {
	const reg = 720
	cases := []struct {
		elapsed     int
		wantFlag    bool
		wantSeconds int
	}{
		{759, false, 0},
		{760, false, 0},
		{761, true, 41},
	}
	for _, c := range cases {
		flag, secs := ComputeOvertime(c.elapsed, reg)
		if flag != c.wantFlag || secs != c.wantSeconds {
			t.Errorf("ComputeOvertime(%d, %d) = (%v, %d), want (%v, %d)",
				c.elapsed, reg, flag, secs, c.wantFlag, c.wantSeconds)
		}
	}
}

// Prolongations COURTES assumées perdues (coût documenté du seuil +40 s).
func TestComputeOvertime_ShortOvertimesAreLost(t *testing.T) {
	const reg = 720
	for _, elapsed := range []int{reg + 19, reg + 24} {
		if flag, _ := ComputeOvertime(elapsed, reg); flag {
			t.Errorf("ComputeOvertime(%d, %d) : flagué alors que la prolongation courte est assumée perdue", elapsed, reg)
		}
	}
}

// Variante inconnue / titre sans table / durée absente → jamais de flag.
func TestComputeOvertime_NoRegulationNeverFlags(t *testing.T) {
	cases := []struct{ elapsed, reg int }{
		{990, 0},  // variante inconnue
		{990, -1}, // valeur aberrante
		{0, 720},  // durée indisponible
		{-5, 720}, // durée aberrante
	}
	for _, c := range cases {
		if flag, secs := ComputeOvertime(c.elapsed, c.reg); flag || secs != 0 {
			t.Errorf("ComputeOvertime(%d, %d) = (%v, %d), want (false, 0)", c.elapsed, c.reg, flag, secs)
		}
	}
}

// Le seuil documenté vaut bien 40 s (garde la constante alignée avec la mesure).
func TestOvertimeMarginSeconds(t *testing.T) {
	if OvertimeMarginSeconds != 40 {
		t.Errorf("OvertimeMarginSeconds = %d, want 40 (mesure : 0 faux positif sur 724 Slayer de contrôle)", OvertimeMarginSeconds)
	}
}
