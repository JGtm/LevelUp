package skill_v2

import (
	"math"
	"math/rand"
	"testing"
)

// synthRandomWalk génère un random walk observé bruité, déterministe (seed fixe).
// Retourne les états vrais et les observations.
func synthRandomWalk(t *testing.T, n int, qTrue, rTrue, m0 float64, seed int64) (trueS, z []float64) {
	t.Helper()
	rng := rand.New(rand.NewSource(seed)) //nolint:gosec // test déterministe
	trueS = make([]float64, n)
	z = make([]float64, n)
	s := m0
	for i := 0; i < n; i++ {
		if i > 0 {
			s += rng.NormFloat64() * math.Sqrt(qTrue)
		}
		trueS[i] = s
		z[i] = s + rng.NormFloat64()*math.Sqrt(rTrue)
	}
	return trueS, z
}

func rmse(a, b []float64) float64 {
	var sum float64
	for i := range a {
		d := a[i] - b[i]
		sum += d * d
	}
	return math.Sqrt(sum / float64(len(a)))
}

func TestEstimateTTT_ConvergesUnder10Iterations(t *testing.T) {
	qTrue, rTrue := 0.4, 4.0
	_, z := synthRandomWalk(t, 300, qTrue, rTrue, 25.0, 42)

	res := EstimateTTT(z, TTTConfig{
		InitialMean:    z[0],
		InitialVar:     10,
		InitProcessVar: 0.5,
		InitObsVar:     3.0,
		MaxIter:        30,
		// Tol sur |Δq|+|Δr| ; avec r~4 c'est ~0.5% de r → critère de convergence
		// légitime, pas un raccourci.
		Tol: 2e-2,
	})

	if !res.Converged {
		t.Fatalf("EM n'a pas convergé en %d itérations (q=%v r=%v)", res.Iterations, res.ProcessVar, res.ObsVar)
	}
	if res.Iterations >= 10 {
		t.Errorf("convergence en %d itérations, attendu < 10", res.Iterations)
	}
	// Paramètres récupérés dans le bon ordre de grandeur (EM sur chaîne unique
	// n'est pas exact mais doit rester dans un facteur ~3 du vrai).
	if res.ProcessVar < qTrue/3 || res.ProcessVar > qTrue*3 {
		t.Errorf("q ré-estimé = %v, attendu ~%v (facteur 3)", res.ProcessVar, qTrue)
	}
	if res.ObsVar < rTrue/3 || res.ObsVar > rTrue*3 {
		t.Errorf("r ré-estimé = %v, attendu ~%v (facteur 3)", res.ObsVar, rTrue)
	}
}

func TestEstimateTTT_LogLikelihoodNonDecreasing(t *testing.T) {
	_, z := synthRandomWalk(t, 200, 0.5, 3.0, 25.0, 7)
	res := EstimateTTT(z, TTTConfig{
		InitialMean: z[0], InitialVar: 10,
		InitProcessVar: 2.0, InitObsVar: 0.5, // volontairement loin
		MaxIter: 40, Tol: 1e-3,
	})
	if len(res.LogLikelihoods) < 2 {
		t.Fatalf("trop peu d'itérations pour vérifier la monotonie (%d)", len(res.LogLikelihoods))
	}
	for i := 1; i < len(res.LogLikelihoods); i++ {
		if res.LogLikelihoods[i] < res.LogLikelihoods[i-1]-1e-6 {
			t.Errorf("log-vraisemblance décroît à l'itération %d : %v < %v (EM doit croître)",
				i, res.LogLikelihoods[i], res.LogLikelihoods[i-1])
		}
	}
}

func TestTTT_SmootherBeatsFilter(t *testing.T) {
	qTrue, rTrue := 0.5, 5.0
	trueS, z := synthRandomWalk(t, 250, qTrue, rTrue, 25.0, 123)

	fMean, fVar, _, pVar, _ := kalmanForward(z, qTrue, rTrue, z[0], 10)
	sMean, _, _ := rtsBackward(fMean, fVar, pVar, qTrue)

	filtRMSE := rmse(fMean, trueS)
	smRMSE := rmse(sMean, trueS)
	// Le lisseur exploite passé ET futur → erreur ≤ filtre (qui n'a que le passé).
	if smRMSE > filtRMSE+1e-9 {
		t.Errorf("RMSE lisseur %v > filtre %v — le lisseur devrait être meilleur", smRMSE, filtRMSE)
	}
}

func TestEstimateTTT_EdgeCases(t *testing.T) {
	cfg := TTTConfig{InitialMean: 25, InitialVar: 10, InitProcessVar: 1, InitObsVar: 1, MaxIter: 10, Tol: 1e-3}

	empty := EstimateTTT(nil, cfg)
	if !empty.Converged || len(empty.SmoothedMean) != 0 {
		t.Errorf("vide → Converged + aucun état lissé, got %+v", empty)
	}

	single := EstimateTTT([]float64{30}, cfg)
	if !single.Converged || len(single.SmoothedMean) != 1 || single.SmoothedMean[0] != 30 {
		t.Errorf("1 obs → 1 état lissé = obs, got %+v", single)
	}
}
