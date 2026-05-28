package skill_v2

// mode_correlation_matrix.go — Sprint 2.B : matrice de couplage cross-mode
// calibrée empiriquement, remplaçant le scalaire DefaultModeCouplingWeight.
//
// Idée : le leak Phase 4 (mode_correlation.go) propage une fraction du delta de
// μ d'un mode vers les autres. Cette fraction était un scalaire unique (0.3).
// En pratique slayer↔objectif (gunplay proche) sont plus corrélés que
// slayer↔chaos. On estime donc un poids par paire de modes = corrélation de
// Pearson des μ entre modes sur l'ensemble des joueurs ayant joué les deux,
// capée à Phase4ModeCouplingMaxWeight (0.4, contrainte produit).
//
// Pure (0 accès DB). L'estimation tourne dans cmd/lusr_v2_ttt_batch ; les poids
// sont stockés dans lusr_hyperparams_v2 et relus par propagateCrossModeLeak.

import (
	"math"
	"strings"
)

// minPlayersForCoupling : nombre minimum de joueurs ayant joué les deux modes
// pour qu'une corrélation soit jugée fiable. En dessous, aucune entrée n'est
// produite (le runtime retombe sur le scalaire par défaut).
const minPlayersForCoupling = 3

// GroupState : un μ courant d'un joueur sur un groupe de modes (entrée de
// l'estimation de matrice).
type GroupState struct {
	Group string
	Mu    float64
}

// ModeCouplingHyperparamName retourne le nom de hyperparam pour le couplage
// source→target : "mode_coupling_<source>_<target>". Stocké avec
// playlist_group = source dans lusr_hyperparams_v2.
func ModeCouplingHyperparamName(source, target string) string {
	return "mode_coupling_" + source + "_" + target
}

// CouplingWeightFor retourne le poids de couplage source→target depuis les
// hyperparams chargés, ou `fallback` si aucune entrée (matrice pas encore
// calculée pour cette paire). Le clamp final [0, 0.4] est garanti par
// ApplyCrossModeLeak côté application.
func CouplingWeightFor(hyperparams map[string]float64, source, target string, fallback float64) float64 {
	if v, ok := hyperparams[ModeCouplingHyperparamName(source, target)]; ok {
		return v
	}
	return fallback
}

// EstimateCouplingMatrix calcule, pour chaque paire de modes, le poids de
// couplage = corrélation de Pearson des μ sur les joueurs ayant joué les deux,
// clampée à [0, Phase4ModeCouplingMaxWeight]. Symétrique : matrix[A][B] == matrix[B][A].
//
// Une corrélation négative → 0 (le leak additif ne modélise pas l'anti-corrélation).
// Variance nulle ou < minPlayersForCoupling échantillons → pas d'entrée pour la paire.
func EstimateCouplingMatrix(playerStates map[string][]GroupState) map[string]map[string]float64 {
	// Indexe μ par (xuid → group → mu) et collecte l'ensemble des groupes.
	byPlayer := make(map[string]map[string]float64, len(playerStates))
	groupSet := make(map[string]struct{})
	for xuid, states := range playerStates {
		m := make(map[string]float64, len(states))
		for _, s := range states {
			m[s.Group] = s.Mu
			groupSet[s.Group] = struct{}{}
		}
		byPlayer[xuid] = m
	}
	groups := sortedKeys(groupSet)

	out := make(map[string]map[string]float64)
	for i := 0; i < len(groups); i++ {
		for j := i + 1; j < len(groups); j++ {
			a, b := groups[i], groups[j]
			var xs, ys []float64
			for _, pm := range byPlayer {
				muA, okA := pm[a]
				muB, okB := pm[b]
				if okA && okB {
					xs = append(xs, muA)
					ys = append(ys, muB)
				}
			}
			if len(xs) < minPlayersForCoupling {
				continue
			}
			w := clampCoupling(pearson(xs, ys))
			setSym(out, a, b, w)
		}
	}
	return out
}

// clampCoupling borne une corrélation à [0, Phase4ModeCouplingMaxWeight].
func clampCoupling(r float64) float64 {
	if math.IsNaN(r) || r < 0 {
		return 0
	}
	if r > Phase4ModeCouplingMaxWeight {
		return Phase4ModeCouplingMaxWeight
	}
	return r
}

// pearson retourne le coefficient de corrélation de Pearson. Variance nulle
// (dénominateur 0) → 0 (corrélation indéterminée).
func pearson(xs, ys []float64) float64 {
	n := float64(len(xs))
	var sx, sy float64
	for i := range xs {
		sx += xs[i]
		sy += ys[i]
	}
	mx, my := sx/n, sy/n
	var cov, vx, vy float64
	for i := range xs {
		dx, dy := xs[i]-mx, ys[i]-my
		cov += dx * dy
		vx += dx * dx
		vy += dy * dy
	}
	denom := math.Sqrt(vx * vy)
	if denom == 0 {
		return 0
	}
	return cov / denom
}

func setSym(m map[string]map[string]float64, a, b string, v float64) {
	if m[a] == nil {
		m[a] = make(map[string]float64)
	}
	if m[b] == nil {
		m[b] = make(map[string]float64)
	}
	m[a][b] = v
	m[b][a] = v
}

// sortedKeys retourne les clés d'un set, triées (déterminisme des sorties).
func sortedKeys(set map[string]struct{}) []string {
	out := make([]string, 0, len(set))
	for k := range set {
		out = append(out, k)
	}
	// Tri par insertion simple (peu de groupes ; évite d'importer sort ici).
	for i := 1; i < len(out); i++ {
		for j := i; j > 0 && strings.Compare(out[j-1], out[j]) > 0; j-- {
			out[j-1], out[j] = out[j], out[j-1]
		}
	}
	return out
}
