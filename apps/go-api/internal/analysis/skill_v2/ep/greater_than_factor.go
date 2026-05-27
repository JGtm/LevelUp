package ep

import "math"

// GreaterThanFactor encode l'observation X > ε avec X gaussienne en entrée.
// Sémantique TrueSkill : X = team_perf_winner - team_perf_loser, ε = draw margin.
//
// Le facteur est UNI-DIRECTIONNEL : il a une seule variable (la différence), pas
// deux. La contrainte (X > ε) tronque la distribution de X au-dessus de ε ;
// EP la remplace par la Gaussienne moment-matchée, et envoie back le message
// `moment_matched_marginal / incoming` à la variable.
//
// Math (Herbrich 2007 §4.1, Moserware/Skills/TruncatedGaussianCorrectionFunctions) :
//
//	d = (μ - ε) / σ
//	v = pdf(d) / cdf(d)        (winner correction)
//	w = v · (v + d)
//	μ_new = μ + σ · v
//	σ²_new = σ² · (1 - w)
//
// Le message back est alors N(μ_new, σ²_new) / N(μ, σ²), calculé en canonique.
type GreaterThanFactor struct {
	x       *Variable
	epsilon float64
	name    string
}

// NewGreaterThanFactor encode l'observation : la variable x est strictement > epsilon.
func NewGreaterThanFactor(name string, x *Variable, epsilon float64) *GreaterThanFactor {
	return &GreaterThanFactor{x: x, epsilon: epsilon, name: name}
}

// UpdateMessages applique la correction tronquée à la variable.
func (f *GreaterThanFactor) UpdateMessages() float64 {
	incoming := f.x.MessageTo(f)
	if incoming.IsUniform() {
		// Sans info en entrée, le facteur ne peut rien envoyer back (le calcul
		// μ/σ est indéfini). Reste uniforme.
		return f.x.UpdateMessage(f, UniformGaussian())
	}
	mu := incoming.Mu()
	sigma := incoming.Sigma()
	d := (mu - f.epsilon) / sigma
	v := vTruncatedWin(d)
	w := wTruncatedWin(d)

	newMu := mu + sigma*v
	newVariance := sigma * sigma * (1.0 - w)

	if newVariance <= 0 {
		// Cas dégénéré : la troncature dégrade tellement la variance qu'elle
		// passe ≤ 0 numériquement. Renvoie uniform plutôt que NaN.
		return f.x.UpdateMessage(f, UniformGaussian())
	}
	newMarginal, err := FromMeanVariance(newMu, newVariance)
	if err != nil {
		return f.x.UpdateMessage(f, UniformGaussian())
	}
	// message_back = new_marginal / incoming
	msg := newMarginal.Div(incoming)
	return f.x.UpdateMessage(f, msg)
}

// Name implémente Factor.
func (f *GreaterThanFactor) Name() string {
	if f.name == "" {
		return "GreaterThan"
	}
	return f.name
}

// vTruncatedWin et wTruncatedWin sont les corrections moment-matching pour
// une Gaussienne tronquée en-dessous de t. Variantes pour win / draw / loss
// vivront dans des facteurs dédiés ; ici on n'expose que win pour Phase 3a.
func vTruncatedWin(t float64) float64 {
	denom := stdNormalCDF(t)
	if denom < 1e-12 {
		// Limite : t très négatif, v tend vers -t.
		return -t
	}
	return stdNormalPDF(t) / denom
}

func wTruncatedWin(t float64) float64 {
	v := vTruncatedWin(t)
	w := v * (v + t)
	if w < 0 {
		return 0
	}
	if w > 1 {
		return 1
	}
	return w
}

// stdNormalPDF / stdNormalCDF : copies locales pour éviter une dépendance au
// package parent. Implémentations identiques (utiliseraient le même math.Erf).
func stdNormalPDF(x float64) float64 {
	return math.Exp(-0.5*x*x) / math.Sqrt(2.0*math.Pi)
}

func stdNormalCDF(x float64) float64 {
	return 0.5 * (1.0 + math.Erf(x/math.Sqrt(2.0)))
}
