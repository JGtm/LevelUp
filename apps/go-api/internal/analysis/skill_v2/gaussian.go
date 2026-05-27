package skill_v2

import (
	"fmt"
	"math"
)

// Gaussian représente une distribution normale N(mu, sigma²) sur les skills latents.
// Toujours stockée sous forme (mu, sigma) plutôt que (mu, sigma²) — sigma est ce
// qui s'interprète directement comme "incertitude" et c'est la quantité qu'on
// rapproche d'un seuil minimal de fiabilité.
type Gaussian struct {
	Mu    float64
	Sigma float64
}

// NewGaussian construit une Gaussian validée (sigma ≥ 0, valeurs finies).
// Retourne une erreur explicite plutôt que de paniquer — les callers viennent
// souvent de la DB où des NaN/Inf peuvent traîner après corruption.
func NewGaussian(mu, sigma float64) (Gaussian, error) {
	if math.IsNaN(mu) || math.IsInf(mu, 0) {
		return Gaussian{}, fmt.Errorf("skill_v2: mu invalide (%v)", mu)
	}
	if math.IsNaN(sigma) || math.IsInf(sigma, 0) {
		return Gaussian{}, fmt.Errorf("skill_v2: sigma invalide (%v)", sigma)
	}
	if sigma < 0 {
		return Gaussian{}, fmt.Errorf("skill_v2: sigma négatif (%v)", sigma)
	}
	return Gaussian{Mu: mu, Sigma: sigma}, nil
}

// Variance retourne sigma².
func (g Gaussian) Variance() float64 { return g.Sigma * g.Sigma }

// ConservativeRating retourne mu - k*sigma. C'est l'estimation pessimiste
// classique TrueSkill : "le skill du joueur est sûrement ≥ ConservativeRating
// (avec une confiance contrôlée par k)". Standard : k=3 (≈ 99.7% de confiance).
func (g Gaussian) ConservativeRating(k float64) float64 {
	return g.Mu - k*g.Sigma
}

// String pour logs/debug.
func (g Gaussian) String() string {
	return fmt.Sprintf("N(μ=%.3f, σ=%.3f)", g.Mu, g.Sigma)
}

// DefaultPriors retourne les paramètres standards TrueSkill, mis à l'échelle
// pour rester compatibles avec les conventions historiques du domaine (μ ≈ 25,
// σ_0 ≈ 25/3, β = σ_0/2, τ = σ_0/100).
//
// Ces valeurs sont reprises de Herbrich et al. 2007 ; TS2 (Halo 5) utilise
// (μ_0=3, σ_0²=1.6) à l'échelle natif Bayes (cf. paper §4). On reste sur
// l'échelle classique pour faciliter la comparaison avec les implémentations
// existantes (Moserware/Skills C#) ; un mapping de display vers [0..50] ou
// vers la grille LUSR v1 [1000..2000] est appliqué côté UI, pas ici.
func DefaultPriors() Priors {
	mu0 := 25.0
	sigma0 := 25.0 / 3.0
	return Priors{
		Mu0:           mu0,
		Sigma0:        sigma0,
		Beta:          sigma0 / 2.0,
		Tau:           sigma0 / 100.0,
		DrawProbability: 0.10, // 10 % — à recalibrer par mode si nécessaire
	}
}

// Priors regroupe les hyperparamètres scalaires du modèle TrueSkill classique.
// Toutes les fonctions de mise à jour les reçoivent par valeur — ils n'évoluent
// pas pendant un match. La ré-estimation batch (Phase 5) produira de nouveaux
// Priors à appliquer à chaud sans modifier l'historique.
type Priors struct {
	// Mu0 est la moyenne initiale du skill d'un nouveau joueur.
	Mu0 float64
	// Sigma0 est l'écart-type initial du skill d'un nouveau joueur.
	Sigma0 float64
	// Beta contrôle la variance de la performance autour du skill (le "bruit du
	// jeu"). Plus β est grand, plus deux joueurs proches en skill peuvent
	// permuter leur classement par chance.
	Beta float64
	// Tau est l'évolution dynamique du skill entre deux matchs (random walk).
	// σ_i² += τ² à chaque match — empêche σ de converger trop vite.
	Tau float64
	// DrawProbability est la fréquence attendue de draws sur l'ensemble des
	// matchs. Sert à calculer la marge ε de draw via DrawMargin.
	DrawProbability float64
}

// NewPlayerState retourne l'état d'un joueur encore non vu.
func (p Priors) NewPlayerState() Gaussian {
	return Gaussian{Mu: p.Mu0, Sigma: p.Sigma0}
}
