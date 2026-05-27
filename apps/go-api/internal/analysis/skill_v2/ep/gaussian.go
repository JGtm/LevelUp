package ep

import (
	"fmt"
	"math"
)

// Gaussian représente une distribution N(μ, σ²) en forme canonique (precision,
// precision-adjusted-mean). Voir godoc package pour le rationnel.
//
// Invariants :
//   - Pi >= 0 (precision non-négative).
//   - Si Pi == 0 : Tau doit être 0. La Gaussienne est alors "uniforme" — un
//     prior infinitement large utilisé comme état d'initialisation neutre.
type Gaussian struct {
	Pi  float64 // precision = 1/σ²
	Tau float64 // precision-adjusted mean = μ/σ²
}

// UniformGaussian : Gaussienne neutre (variance infinie). État par défaut d'une
// variable jamais informée — agit comme l'élément neutre de la multiplication.
func UniformGaussian() Gaussian { return Gaussian{Pi: 0, Tau: 0} }

// FromMeanVariance construit une Gaussian à partir des paramètres (μ, σ²).
// Retourne une erreur si σ² ≤ 0 ou si les valeurs ne sont pas finies.
func FromMeanVariance(mu, variance float64) (Gaussian, error) {
	if math.IsNaN(mu) || math.IsInf(mu, 0) {
		return Gaussian{}, fmt.Errorf("ep: mu invalide (%v)", mu)
	}
	if math.IsNaN(variance) || math.IsInf(variance, 0) {
		return Gaussian{}, fmt.Errorf("ep: variance invalide (%v)", variance)
	}
	if variance <= 0 {
		return Gaussian{}, fmt.Errorf("ep: variance non strictement positive (%v)", variance)
	}
	pi := 1.0 / variance
	return Gaussian{Pi: pi, Tau: mu * pi}, nil
}

// FromMeanSigma construit une Gaussian depuis (μ, σ).
func FromMeanSigma(mu, sigma float64) (Gaussian, error) {
	if sigma <= 0 {
		return Gaussian{}, fmt.Errorf("ep: sigma non strictement positif (%v)", sigma)
	}
	return FromMeanVariance(mu, sigma*sigma)
}

// Mu retourne la moyenne μ = Tau / Pi. Indéfinie pour une Gaussienne uniforme
// (Pi=0) ; retourne 0 dans ce cas (par convention).
func (g Gaussian) Mu() float64 {
	if g.Pi <= 0 {
		return 0
	}
	return g.Tau / g.Pi
}

// Variance retourne σ² = 1 / Pi. Infinie pour une Gaussienne uniforme ; on
// retourne math.Inf(1) explicitement plutôt que de paniquer.
func (g Gaussian) Variance() float64 {
	if g.Pi <= 0 {
		return math.Inf(1)
	}
	return 1.0 / g.Pi
}

// Sigma retourne l'écart-type σ. Voir Variance pour le cas uniforme.
func (g Gaussian) Sigma() float64 {
	v := g.Variance()
	if math.IsInf(v, 0) {
		return math.Inf(1)
	}
	return math.Sqrt(v)
}

// IsUniform retourne true si la Gaussienne est neutre (Pi = 0).
func (g Gaussian) IsUniform() bool { return g.Pi == 0 }

// Mul multiplie deux Gaussiennes (= addition en canonique). Conserve la
// commutativité, l'associativité et l'élément neutre Uniform.
func (g Gaussian) Mul(other Gaussian) Gaussian {
	return Gaussian{Pi: g.Pi + other.Pi, Tau: g.Tau + other.Tau}
}

// Div divise deux Gaussiennes (= soustraction en canonique). Essentielle pour
// les message-passes EP : message_out = marginal / message_in.
//
// Si Pi devient < 0 numériquement (l'ancien message contenait plus d'info que
// le marginal courant — cas pathologique en cas de divergence EP), on clamp à 0
// et on retourne uniform plutôt que de paniquer. Ce cas signale typiquement
// un problème de convergence dans le caller.
func (g Gaussian) Div(other Gaussian) Gaussian {
	pi := g.Pi - other.Pi
	tau := g.Tau - other.Tau
	if pi <= 0 {
		return UniformGaussian()
	}
	return Gaussian{Pi: pi, Tau: tau}
}

// AbsoluteDifference retourne une borne sur l'écart entre deux Gaussiennes,
// utile pour détecter la convergence EP. Renvoie max(|Δπ|, sqrt(|Δτ|)).
//
// Convention d'usage : si AbsoluteDifference(prev, new) < eps, considérer
// que la passe n'a pas changé la variable de façon significative.
func (g Gaussian) AbsoluteDifference(other Gaussian) float64 {
	dPi := math.Abs(g.Pi - other.Pi)
	dTau := math.Sqrt(math.Abs(g.Tau - other.Tau))
	if dPi > dTau {
		return dPi
	}
	return dTau
}

// String pour debug/logs.
func (g Gaussian) String() string {
	if g.IsUniform() {
		return "Gaussian(uniform)"
	}
	return fmt.Sprintf("Gaussian(μ=%.4f, σ=%.4f, π=%.4f, τ=%.4f)",
		g.Mu(), g.Sigma(), g.Pi, g.Tau)
}
