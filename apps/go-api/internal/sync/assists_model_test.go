// Tests pour assists_model.go : F.1 du plan de tests.
//
// Couvre fitOLS — la regression OLS pure. Tests unitaires sans DB :
//   - seuil minAssistsSamples=15 → nil si en-dessous
//   - dataset lineaire parfait → coefs reconstruisent les beta
//   - dataset bruyant → R² > 0.5
//   - input degenere (toutes rows identiques) → nil (singulier)
//   - R² calcule correctement (1.0 sur perfect, 0 sur random)
package sync

import (
	"math"
	"math/rand"
	"testing"
)

// makeLinearSample fabrique un sample avec assists = trueIntercept + sum(beta_i * x_i).
// Utile pour generer des datasets "parfaits" reproductibles.
func makeLinearSample(mode string, kills, deaths, dd, dt, mmrD float64, beta [6]float64) assistsSample {
	assists := beta[0] + beta[1]*kills + beta[2]*deaths + beta[3]*dd + beta[4]*dt + beta[5]*mmrD
	return assistsSample{
		mode:        mode,
		assists:     assists,
		kills:       kills,
		deaths:      deaths,
		damageDealt: dd,
		damageTaken: dt,
		mmrDelta:    mmrD,
	}
}

func TestFitOLS_BelowMinSamples_ReturnsNil(t *testing.T) {
	// 10 samples (< 15) → nil par contrat.
	var rows []assistsSample
	for i := 0; i < 10; i++ {
		rows = append(rows, assistsSample{
			mode: "arena", assists: 5, kills: 8, deaths: 4,
			damageDealt: 1000, damageTaken: 800, mmrDelta: 50,
		})
	}
	coefs := fitOLS(rows)
	if coefs != nil {
		t.Errorf("fitOLS(10 samples) doit retourner nil (seuil minAssistsSamples=15), got %+v", coefs)
	}
}

func TestFitOLS_ExactlyMinSamples_ReturnsCoefs(t *testing.T) {
	// 15 samples avec features RANDOMISEES (independantes) pour eviter
	// la singularite due a la colinearite.
	beta := [6]float64{1.0, 0.3, 0.05, 0.001, 0.0005, 0.01}
	rng := rand.New(rand.NewSource(99))
	var rows []assistsSample
	for i := 0; i < 15; i++ {
		k := float64(rng.Intn(20))
		d := float64(rng.Intn(15))
		dd := float64(rng.Intn(2000) + 200)
		dt := float64(rng.Intn(1500) + 100)
		mmrD := float64(rng.Intn(200) - 100)
		rows = append(rows, makeLinearSample("arena", k, d, dd, dt, mmrD, beta))
	}
	coefs := fitOLS(rows)
	if coefs == nil {
		t.Fatal("fitOLS(15 samples randomises) doit reussir, got nil")
	}
}

func TestFitOLS_PerfectLinearData_RecoversCoefs(t *testing.T) {
	// Dataset parfait avec features INDEPENDANTES (RNG) pour eviter
	// la colinearite qui redistribue les coefs entre features correles.
	beta := [6]float64{0.5, 0.4, 0.1, 0.002, 0.001, 0.005}
	rng := rand.New(rand.NewSource(31))

	var rows []assistsSample
	for i := 0; i < 60; i++ {
		k := float64(rng.Intn(20))
		d := float64(rng.Intn(15))
		dd := float64(rng.Intn(2000) + 200)
		dt := float64(rng.Intn(1500) + 100)
		mmrD := float64(rng.Intn(200) - 100)
		rows = append(rows, makeLinearSample("arena", k, d, dd, dt, mmrD, beta))
	}

	coefs := fitOLS(rows)
	if coefs == nil {
		t.Fatal("fitOLS perfect data : nil inattendu")
	}

	// Tolerance OLS sur dataset parfait (sans bruit) : devrait etre quasi-exact.
	checks := []struct {
		name string
		got  float64
		want float64
		tol  float64
	}{
		{"intercept", coefs.intercept, beta[0], 1e-6},
		{"coefKills", coefs.coefKills, beta[1], 1e-6},
		{"coefDeaths", coefs.coefDeaths, beta[2], 1e-6},
		{"coefDamageDealt", coefs.coefDamageDealt, beta[3], 1e-6},
		{"coefDamageTaken", coefs.coefDamageTaken, beta[4], 1e-6},
		{"coefMMRDelta", coefs.coefMMRDelta, beta[5], 1e-6},
	}
	for _, c := range checks {
		if math.Abs(c.got-c.want) > c.tol {
			t.Errorf("%s : got %v, want %v (tol %v)", c.name, c.got, c.want, c.tol)
		}
	}

	// R² doit etre tres proche de 1.0 sur dataset parfait.
	if coefs.r2 < 0.9999 {
		t.Errorf("R² = %v, want >= 0.9999 (data parfaite)", coefs.r2)
	}
}

func TestFitOLS_NoisyData_R2Reasonable(t *testing.T) {
	// Dataset bruyant : assists = lineaire + N(0, sigma=1.0). R² attendu > 0.5.
	beta := [6]float64{1.0, 0.5, -0.1, 0.001, 0.0, 0.0}
	rng := rand.New(rand.NewSource(42))

	var rows []assistsSample
	for i := 0; i < 60; i++ {
		k := float64(rng.Intn(20))
		d := float64(rng.Intn(15))
		dd := float64(rng.Intn(2000) + 200)
		dt := float64(rng.Intn(1500) + 100)
		mmrD := float64(rng.Intn(200) - 100)

		s := makeLinearSample("arena", k, d, dd, dt, mmrD, beta)
		s.assists += rng.NormFloat64() // bruit gaussien
		rows = append(rows, s)
	}

	coefs := fitOLS(rows)
	if coefs == nil {
		t.Fatal("fitOLS noisy data : nil inattendu")
	}

	if coefs.r2 < 0.5 {
		t.Errorf("R² = %v, want >= 0.5 sur dataset bruyant", coefs.r2)
	}
	t.Logf("noisy data R² = %.4f, beta intercept = %.3f (vrai = %.3f)", coefs.r2, coefs.intercept, beta[0])
}

func TestFitOLS_DegenerateData_AllIdentical_ReturnsNil(t *testing.T) {
	// 30 rows toutes identiques → matrice X'X singuliere → fitOLS retourne nil.
	var rows []assistsSample
	for i := 0; i < 30; i++ {
		rows = append(rows, assistsSample{
			mode: "arena", assists: 5, kills: 8, deaths: 4,
			damageDealt: 1000, damageTaken: 800, mmrDelta: 50,
		})
	}
	coefs := fitOLS(rows)
	if coefs != nil {
		t.Errorf("fitOLS sur dataset degenere doit retourner nil (matrice singuliere), got %+v", coefs)
	}
}

func TestFitOLS_NSamples_RecordedInCoefs(t *testing.T) {
	beta := [6]float64{1, 0.3, 0.05, 0.001, 0.0005, 0.01}
	rng := rand.New(rand.NewSource(123))
	var rows []assistsSample
	const N = 20
	for i := 0; i < N; i++ {
		rows = append(rows, makeLinearSample("arena",
			float64(rng.Intn(20)), float64(rng.Intn(15)),
			float64(rng.Intn(2000)+200), float64(rng.Intn(1500)+100),
			float64(rng.Intn(200)-100), beta))
	}
	coefs := fitOLS(rows)
	if coefs == nil {
		t.Fatal("fitOLS: nil inattendu")
	}
	if coefs.n != N {
		t.Errorf("n = %d, want %d", coefs.n, N)
	}
}

func TestFitOLS_R2NeverNegative(t *testing.T) {
	// R² peut etre teoriquement negatif (model pire que la moyenne) mais
	// le code clamp a 0. Verifier sur dataset bizarre.
	rng := rand.New(rand.NewSource(7))
	var rows []assistsSample
	for i := 0; i < 30; i++ {
		// Random assists totalement decorrele des features.
		rows = append(rows, assistsSample{
			mode:        "arena",
			assists:     rng.Float64() * 20,
			kills:       float64(rng.Intn(20)),
			deaths:      float64(rng.Intn(15)),
			damageDealt: float64(rng.Intn(2000)),
			damageTaken: float64(rng.Intn(1500)),
			mmrDelta:    float64(rng.Intn(200) - 100),
		})
	}
	coefs := fitOLS(rows)
	if coefs == nil {
		t.Fatal("fitOLS: nil sur random data inattendu")
	}
	if coefs.r2 < 0 {
		t.Errorf("R² = %v, doit etre clamp a >= 0", coefs.r2)
	}
}
