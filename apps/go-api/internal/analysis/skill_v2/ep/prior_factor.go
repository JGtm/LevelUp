package ep

import "fmt"

// PriorFactor ancre une variable à une Gaussienne fixe : X ~ N(μ_0, σ_0²).
// Utilisé en début de match pour injecter le prior (μ, σ) du skill latent
// d'un joueur avant le match courant.
//
// Un PriorFactor n'est jamais "mis à jour" par les autres : il pousse son
// message une fois et reste stable. UpdateMessages reste idempotent — appeler
// 10 fois ou 1 fois donne le même résultat.
type PriorFactor struct {
	variable *Variable
	prior    Gaussian
	name     string
}

// NewPriorFactor attache un PriorFactor à la variable. Le prior est appliqué
// immédiatement à la variable (premier message envoyé).
func NewPriorFactor(name string, v *Variable, prior Gaussian) *PriorFactor {
	pf := &PriorFactor{variable: v, prior: prior, name: name}
	return pf
}

// UpdateMessages envoie le prior à la variable. Idempotent.
func (f *PriorFactor) UpdateMessages() float64 {
	return f.variable.UpdateMessage(f, f.prior)
}

// Name implémente Factor.
func (f *PriorFactor) Name() string {
	if f.name == "" {
		return fmt.Sprintf("Prior(%s)", f.variable.Name)
	}
	return f.name
}
