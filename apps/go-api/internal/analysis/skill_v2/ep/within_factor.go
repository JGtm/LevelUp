package ep

import "math"

// WithinFactor encode l'observation |X| < ε avec X gaussienne en entrée.
// Sémantique TrueSkill draw : X = team_perf_winner - team_perf_loser, ε = draw margin,
// et l'observation |X| < ε signifie que les performances étaient assez proches
// pour déclarer un match nul.
//
// Comme GreaterThanFactor, WithinFactor est uni-directionnel (1 variable connectée).
// Le facteur tronque la distribution de X au domaine ]-ε, +ε[, EP la remplace par
// la Gaussienne moment-matchée et envoie back `moment_matched_marginal / incoming`.
//
// Math (Moserware/Skills/TruncatedGaussianCorrectionFunctions.cs `VWithinMargin`/`WWithinMargin`) :
//
//	d_abs = |μ| / σ          (centré absolu, σ-normalisé)
//	eps_n = ε / σ            (ε σ-normalisé)
//	denom = Φ(eps_n - d_abs) - Φ(-eps_n - d_abs)
//	num   = pdf(-eps_n - d_abs) - pdf(eps_n - d_abs)
//	v     = sign(μ) · num/denom            (note : signe inversé par rapport à win)
//	w     = v² + [(eps_n - d_abs)·pdf(eps_n - d_abs) - (-eps_n - d_abs)·pdf(-eps_n - d_abs)] / denom
//	μ_new = μ - σ · v
//	σ²_new= σ² · (1 - w)
//
// Les limites numériques (denom < ε_machine) renvoient un message uniform pour
// éviter NaN.
type WithinFactor struct {
	x       *Variable
	epsilon float64
	name    string
}

// NewWithinFactor encode l'observation : la variable x est dans (-epsilon, +epsilon).
func NewWithinFactor(name string, x *Variable, epsilon float64) *WithinFactor {
	return &WithinFactor{x: x, epsilon: epsilon, name: name}
}

// UpdateMessages applique la troncature draw à la variable.
func (f *WithinFactor) UpdateMessages() float64 {
	incoming := f.x.MessageTo(f)
	if incoming.IsUniform() {
		return f.x.UpdateMessage(f, UniformGaussian())
	}
	mu := incoming.Mu()
	sigma := incoming.Sigma()
	if sigma <= 0 {
		return f.x.UpdateMessage(f, UniformGaussian())
	}
	v := vTruncatedDraw(mu, sigma, f.epsilon)
	w := wTruncatedDraw(mu, sigma, f.epsilon)

	// E[Y | |Y| < ε] = μ + σ · v où v est la moyenne de la Gaussienne
	// tronquée standard. Signe + cohérent avec la dérivation Herbrich Table 1
	// pour des bornes (-ε, +ε) ; le signe est porté par v lui-même selon le
	// signe de μ.
	newMu := mu + sigma*v
	newVariance := sigma * sigma * (1.0 - w)
	if newVariance <= 0 {
		return f.x.UpdateMessage(f, UniformGaussian())
	}
	newMarginal, err := FromMeanVariance(newMu, newVariance)
	if err != nil {
		return f.x.UpdateMessage(f, UniformGaussian())
	}
	return f.x.UpdateMessage(f, newMarginal.Div(incoming))
}

// Name implémente Factor.
func (f *WithinFactor) Name() string {
	if f.name == "" {
		return "Within"
	}
	return f.name
}

// vTruncatedDraw : moment de la Gaussienne tronquée sur (-ε, +ε).
// Le signe est inversé par rapport à win : si μ > 0, v est négatif (pull μ down
// vers 0) ; si μ < 0, v est positif (pull μ up vers 0). C'est cohérent avec
// l'interprétation "draw rapproche les performances".
func vTruncatedDraw(mu, sigma, epsilon float64) float64 {
	absMu := math.Abs(mu)
	dAbs := absMu / sigma
	epsN := epsilon / sigma
	denom := stdNormalCDF(epsN-dAbs) - stdNormalCDF(-epsN-dAbs)
	if denom < 1e-12 {
		// Limite : draw quasi-impossible numériquement. On retourne le pull
		// vers la borne la plus proche (cohérent avec Moserware).
		if mu < 0 {
			return -mu - epsilon
		}
		return -mu + epsilon
	}
	num := stdNormalPDF(-epsN-dAbs) - stdNormalPDF(epsN-dAbs)
	if mu < 0 {
		return -num / denom
	}
	return num / denom
}

// wTruncatedDraw : facteur de réduction de variance pour le draw.
func wTruncatedDraw(mu, sigma, epsilon float64) float64 {
	absMu := math.Abs(mu)
	dAbs := absMu / sigma
	epsN := epsilon / sigma
	denom := stdNormalCDF(epsN-dAbs) - stdNormalCDF(-epsN-dAbs)
	if denom < 1e-12 {
		return 1.0
	}
	v := vTruncatedDraw(absMu, sigma, epsilon) // |μ| pour la symétrie
	w := v*v + ((epsN-dAbs)*stdNormalPDF(epsN-dAbs)-(-epsN-dAbs)*stdNormalPDF(-epsN-dAbs))/denom
	if w < 0 {
		return 0
	}
	if w > 1 {
		return 1
	}
	return w
}
