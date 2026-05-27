package ep

import "fmt"

// SumFactor encode la contrainte linéaire :
//
//	Y = Σ w_i · X_i        (sum = Σ weights[i] · inputs[i])
//
// Utilisé en TrueSkill pour Y = team_performance et X_i = player_performance,
// avec w_i = 1 (TS classique) ou w_i = time_played_i / match_length (TS2).
//
// Messages :
//
//   - Pour la sortie Y, le message reçu (depuis le sum factor) est une combinaison
//     linéaire de gaussiennes :
//     μ_Y = Σ w_i · μ_{X_i}
//     σ_Y² = Σ w_i² · σ_{X_i}²
//
//   - Pour une entrée X_k, on inverse la contrainte :
//     X_k = (Y - Σ_{i≠k} w_i · X_i) / w_k
//     μ_{X_k} = (μ_Y - Σ_{i≠k} w_i · μ_{X_i}) / w_k
//     σ_{X_k}² = (σ_Y² + Σ_{i≠k} w_i² · σ_{X_i}²) / w_k²
//
// Les coefficients utilisés dans la formule sortante vers X_k sont :
//   - pour Y : 1/w_k
//   - pour chaque X_i (i≠k) : -w_i/w_k
//
// updateMessageOnVariable factorise cette logique pour réutilisation.
type SumFactor struct {
	sum     *Variable
	inputs  []*Variable
	weights []float64
	name    string
}

// NewSumFactor : sum = Σ weights[i] · inputs[i]. Panic si len(weights) != len(inputs).
func NewSumFactor(name string, sum *Variable, inputs []*Variable, weights []float64) *SumFactor {
	if len(weights) != len(inputs) {
		panic(fmt.Sprintf("ep: NewSumFactor: len(weights)=%d != len(inputs)=%d", len(weights), len(inputs)))
	}
	dup := make([]float64, len(weights))
	copy(dup, weights)
	return &SumFactor{sum: sum, inputs: inputs, weights: dup, name: name}
}

// UpdateMessages : envoie un message vers chaque variable (sum + chaque input).
// Pour chaque destination on construit les coefficients adaptés et on délègue
// à updateOneVariable.
func (f *SumFactor) UpdateMessages() float64 {
	total := 0.0
	// Vers sum : coeffs = weights, sources = inputs.
	total += f.updateOneVariable(f.sum, f.inputs, f.weights)
	// Vers chaque input k : coeffs derivés, sources = sum + autres inputs.
	for k := range f.inputs {
		dest := f.inputs[k]
		wk := f.weights[k]
		if wk == 0 {
			continue // input avec poids 0 = absent, pas de message à envoyer
		}
		coeffs := make([]float64, len(f.inputs))
		sources := make([]*Variable, len(f.inputs))
		coeffs[0] = 1.0 / wk
		sources[0] = f.sum
		idx := 1
		for i := range f.inputs {
			if i == k {
				continue
			}
			coeffs[idx] = -f.weights[i] / wk
			sources[idx] = f.inputs[i]
			idx++
		}
		total += f.updateOneVariable(dest, sources, coeffs)
	}
	return total
}

// updateOneVariable calcule le message à destination de dest en additionnant
// les Gaussiennes coeffs[i] · msg(sources[i]→f).
//
// L'addition d'une variable mise à l'échelle a · X (en canonique) :
//   - si π_X = 0, le résultat reste uniforme
//   - sinon : π = π_X / a², τ = τ_X / a (μ et σ² mis à l'échelle de façon cohérente)
//
// Sommer N gaussiennes indépendantes : addition simple de (μ, σ²), donc
// en canonique : on convertit chaque (π_i, τ_i) en (μ_i, σ_i²), on additionne,
// on reconvertit en canonique. Plus simple à écrire que d'additionner les
// inverses précisions.
func (f *SumFactor) updateOneVariable(dest *Variable, sources []*Variable, coeffs []float64) float64 {
	var muSum, varSum float64
	for i, src := range sources {
		c := coeffs[i]
		msg := src.MessageTo(f)
		if msg.IsUniform() {
			// L'une des sources est uniforme → la somme l'est aussi. Pas
			// d'info à propager vers dest sur cette passe.
			return dest.UpdateMessage(f, UniformGaussian())
		}
		mu := msg.Mu()
		variance := msg.Variance()
		muSum += c * mu
		varSum += c * c * variance
	}
	// (muSum, varSum) → canonical. varSum > 0 garanti tant que toutes les
	// sources sont propres (vérifié plus haut).
	g, err := FromMeanVariance(muSum, varSum)
	if err != nil {
		// Cas pathologique : varSum a flotté à 0 ou négatif (sources extrêmes).
		// On dégrade en uniform plutôt que de paniquer.
		return dest.UpdateMessage(f, UniformGaussian())
	}
	return dest.UpdateMessage(f, g)
}

// Name implémente Factor.
func (f *SumFactor) Name() string {
	if f.name == "" {
		return "Sum"
	}
	return f.name
}
