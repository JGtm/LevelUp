package ep

// LikelihoodFactor relie deux variables X et Y par la relation Y ~ N(X, σ²).
// Sémantique TrueSkill : X = skill latent, Y = performance dans le match,
// σ² = β² (le bruit du jeu, paramètre du modèle).
//
// La transformation centrale est "ajouter une variance" à une Gaussienne en
// forme canonique. Si X a (π_X, τ_X), alors Y vue à travers ce link a :
//
//	π_Y = π_X / (1 + σ²·π_X)
//	τ_Y = τ_X / (1 + σ²·π_X)
//
// Et symétriquement de Y vers X. Si X est uniforme (π_X = 0), Y le reste aussi
// (aucune info à propager).
type LikelihoodFactor struct {
	x        *Variable
	y        *Variable
	addedVar float64 // σ² (β² en sémantique TS)
	name     string
}

// NewLikelihoodFactor : X "produit" Y avec un bruit gaussien de variance σ².
func NewLikelihoodFactor(name string, x, y *Variable, addedVariance float64) *LikelihoodFactor {
	return &LikelihoodFactor{x: x, y: y, addedVar: addedVariance, name: name}
}

// UpdateMessages propage les messages dans les deux sens. La somme des deltas
// permet au runner de détecter la convergence.
func (f *LikelihoodFactor) UpdateMessages() float64 {
	// X → Y : prends le message que X envoie au facteur, ajoute σ², envoie à Y.
	msgFromX := f.x.MessageTo(f)
	deltaY := f.y.UpdateMessage(f, gaussianAddVariance(msgFromX, f.addedVar))

	// Y → X : symétrique.
	msgFromY := f.y.MessageTo(f)
	deltaX := f.x.UpdateMessage(f, gaussianAddVariance(msgFromY, f.addedVar))

	return deltaX + deltaY
}

// Name implémente Factor.
func (f *LikelihoodFactor) Name() string {
	if f.name == "" {
		return "Likelihood(" + f.x.Name + " → " + f.y.Name + ")"
	}
	return f.name
}

// gaussianAddVariance retourne la Gaussienne avec variance ajoutée. Utilitaire
// fondamental pour LikelihoodFactor et tous les facteurs qui propagent à
// travers un canal bruité.
//
// Si la Gaussienne d'entrée est uniforme, la sortie l'est aussi.
func gaussianAddVariance(in Gaussian, addedVar float64) Gaussian {
	if in.IsUniform() {
		return UniformGaussian()
	}
	factor := 1.0 / (1.0 + addedVar*in.Pi)
	return Gaussian{Pi: in.Pi * factor, Tau: in.Tau * factor}
}
