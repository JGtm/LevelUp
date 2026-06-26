package ep

import "testing"

// TestFlipSide vérifie l'involution A↔B utilisée pour réindexer les counts
// quand UpdateMatch2Team swap les équipes (cas TeamLoss).
func TestFlipSide(t *testing.T) {
	if flipSide(SideA) != SideB {
		t.Error("flipSide(SideA) doit donner SideB")
	}
	if flipSide(SideB) != SideA {
		t.Error("flipSide(SideB) doit donner SideA")
	}
	// Involution : double flip = identité.
	if flipSide(flipSide(SideA)) != SideA {
		t.Error("flipSide doit être une involution")
	}
}

// TestGaussianFromPrior couvre les deux branches : prior uniforme → UniformGaussian
// (garde-fou "premier match"), prior informé → renvoyé tel quel (pass-through).
func TestGaussianFromPrior(t *testing.T) {
	t.Run("prior uniforme → uniforme", func(t *testing.T) {
		out := gaussianFromPrior(UniformGaussian())
		if !out.IsUniform() {
			t.Errorf("prior uniforme doit rester uniforme, got %v", out)
		}
	})

	t.Run("prior informé → pass-through", func(t *testing.T) {
		in, _ := FromMeanVariance(25, 70)
		out := gaussianFromPrior(in)
		if out.Pi != in.Pi || out.Tau != in.Tau {
			t.Errorf("prior informé modifié : got %v, want %v", out, in)
		}
	})
}

// TestExtractPosteriors_UniformMarginalStaysUniform : une variable jamais informée
// (marginal uniforme) doit ressortir uniforme, sans tenter d'ajouter τ² à une
// variance infinie.
func TestExtractPosteriors_UniformMarginalStaysUniform(t *testing.T) {
	v := NewVariable("skill_0") // marginal initial = uniform
	out := extractPosteriors([]*Variable{v}, 0.5)
	if len(out) != 1 {
		t.Fatalf("len(out) = %d, want 1", len(out))
	}
	if !out[0].IsUniform() {
		t.Errorf("marginal uniforme → posterior uniforme, got %v", out[0])
	}
}

// TestExtractPosteriors_AddsTauSquared : sur un marginal informé, la variance de
// sortie = variance marginale + τ² (random walk dynamique).
func TestExtractPosteriors_AddsTauSquared(t *testing.T) {
	v := NewVariable("skill_0")
	informed, _ := FromMeanVariance(25, 50)
	v.Marginal = informed
	tau := 2.0
	out := extractPosteriors([]*Variable{v}, tau)
	wantVar := 50.0 + tau*tau // 54
	if got := out[0].Variance(); got < wantVar-tol || got > wantVar+tol {
		t.Errorf("Variance posterior = %v, want %v (var + τ²)", got, wantVar)
	}
	if got := out[0].Mu(); got < 25-tol || got > 25+tol {
		t.Errorf("Mu posterior = %v, want 25 (inchangé)", got)
	}
}

// TestUpdateMatch2Team_InputValidation : équipes vides ou Beta ≤ 0 doivent
// produire une erreur explicite (pas un panic ni un résultat silencieux).
func TestUpdateMatch2Team_InputValidation(t *testing.T) {
	prior, _ := FromMeanVariance(25, 70)
	cfg := DefaultMatch2TeamConfig()

	cases := []struct {
		name string
		in   Match2TeamInput
	}{
		{"TeamA vide", Match2TeamInput{TeamA: nil, TeamB: []Gaussian{prior}, Beta: 4}},
		{"TeamB vide", Match2TeamInput{TeamA: []Gaussian{prior}, TeamB: nil, Beta: 4}},
		{"Beta = 0", Match2TeamInput{TeamA: []Gaussian{prior}, TeamB: []Gaussian{prior}, Beta: 0}},
		{"Beta < 0", Match2TeamInput{TeamA: []Gaussian{prior}, TeamB: []Gaussian{prior}, Beta: -1}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if _, _, err := UpdateMatch2Team(tc.in, cfg); err == nil {
				t.Errorf("attendu une erreur pour %s", tc.name)
			}
		})
	}
}
