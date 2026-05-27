// Package ep implémente un factor graph + Expectation Propagation pour le LUSR v2.
//
// Construit indépendamment du closed-form du package parent (skill_v2) : ce
// dernier reste comme fast-path pour TS classique 2-équipes ; ep est la voie
// principale pour TS2 §8 (kills/deaths comme observations) et §11 (mode
// correlation) où le closed-form ne tient plus.
//
// # Forme canonique
//
// Toutes les Gaussiennes utilisées en EP vivent en représentation canonique
// (precision π, precision-adjusted-mean τ) plutôt que (μ, σ) :
//
//	π = 1/σ²        (precision = inverse de la variance)
//	τ = μ/σ²        (precision-adjusted-mean)
//
// Avantages :
//   - Multiplication de Gaussiennes = addition en (π, τ) : trivial à coder.
//   - Division = soustraction : essentielle pour calculer le message sortant
//     d'une variable vers un facteur (marginal / dernier message reçu).
//   - π = 0 représente la Gaussienne uniforme N(?, ∞), prior par défaut d'une
//     variable jamais informée — pas de cas spécial nécessaire pour l'init.
//
// # Pipeline EP
//
//  1. Construire le factor graph du match (priors → likelihoods → sums → contraintes).
//  2. Itérer les message-updates jusqu'à convergence (typiquement 5-20 passes
//     pour TS).
//  3. Lire les marginaux finaux sur les variables skill_i comme nouveau posterior.
//
// # Phases d'implémentation
//
//   - Phase 3a (ce package) : foundation Gaussian + Variable + Factor + factors
//     basiques (Prior, Likelihood, Sum, GreaterThan).
//   - Phase 3b : reconstruction du match 2-équipes sur EP, régression test
//     vs closed-form (mêmes résultats numériques au eps près).
//   - Phase 3c : ajout du TruncatedGaussianCountFactor pour kills/deaths
//     comme observations Bayésiennes (TS2 §8).
//
// # Référence
//
// Moserware/Skills (C#, BSD) implémente exactement ce pattern ; le port Go
// suit la même structure de classes mais utilise les idiomes Go (slices au
// lieu d'IList<T>, interface au lieu d'abstract base class).
package ep
