package ep

import (
	"math"
	"testing"
)

// TestFromMeanSigma couvre le chemin d'erreur (sigma ≤ 0) et la délégation
// nominale à FromMeanVariance (σ → σ²).
func TestFromMeanSigma(t *testing.T) {
	t.Run("sigma <= 0 erreur", func(t *testing.T) {
		for _, s := range []float64{0, -1, -8.33} {
			if _, err := FromMeanSigma(0, s); err == nil {
				t.Errorf("FromMeanSigma(0, %v) devrait échouer", s)
			}
		}
	})

	t.Run("sigma > 0 délègue (σ² = variance)", func(t *testing.T) {
		g, err := FromMeanSigma(25, 2)
		if err != nil {
			t.Fatalf("FromMeanSigma(25, 2): %v", err)
		}
		if math.Abs(g.Mu()-25) > tol {
			t.Errorf("Mu = %v, want 25", g.Mu())
		}
		if math.Abs(g.Variance()-4) > tol {
			t.Errorf("Variance = %v, want 4 (σ=2)", g.Variance())
		}
		if math.Abs(g.Sigma()-2) > tol {
			t.Errorf("Sigma = %v, want 2", g.Sigma())
		}
	})
}
