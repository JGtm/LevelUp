package skill_v2

// hyperparams_load.go : conversion des hyperparamètres empiriques (écrits en
// base par cmd/lusr_v2_ttt_batch dans lusr_hyperparams_v2) vers les structures
// que consomme le modèle au runtime — Priors (draw probability) et
// CountHyperparams (biais des observations kills/deaths).
//
// Sprint 1.B : avant ce wiring, le batch écrivait ces valeurs mais le sync
// utilisait toujours les défauts hardcodés → ré-estimation = écriture morte.
//
// Toutes les fonctions sont pures (map d'entrée → struct de sortie), testables
// sans DB.

import "levelup/go-api/internal/analysis/skill_v2/ep"

// Ré-export des types de counts du sous-package ep, pour que les callers
// (internal/sync) manipulent les hyperparams de count sans importer ep
// directement.
type (
	CountType        = ep.CountType
	CountHyperparams = ep.CountHyperparams
)

const (
	CountKill  = ep.CountKill
	CountDeath = ep.CountDeath
)

// Noms des hyperparams empiriques écrits par cmd/lusr_v2_ttt_batch.
const (
	hyperparamDrawProbability = "draw_probability_empirical"
	hyperparamKillMean        = "kill_mean_empirical"
	hyperparamDeathMean       = "death_mean_empirical"
)

// DefaultCountHyperparamsMap retourne la map kill+death des CountHyperparams par
// défaut. Sert de base à LoadCountHyperparamsFromDB et de fallback quand aucun
// hyperparam empirique n'est disponible.
func DefaultCountHyperparamsMap() map[CountType]CountHyperparams {
	return map[CountType]CountHyperparams{
		CountKill:  ep.DefaultCountHyperparams(CountKill),
		CountDeath: ep.DefaultCountHyperparams(CountDeath),
	}
}

// LoadPriorsFromHyperparams part de defaultP et n'override que DrawProbability si
// `draw_probability_empirical` est présent et dans [0, 1[. Les autres scalaires
// (Mu0, Sigma0, Beta, Tau) ne sont pas ré-estimés par le batch actuel — ils
// restent ceux de defaultP.
func LoadPriorsFromHyperparams(params map[string]float64, defaultP Priors) Priors {
	p := defaultP
	if dp, ok := params[hyperparamDrawProbability]; ok && dp >= 0 && dp < 1 {
		p.DrawProbability = dp
	}
	return p
}

// LoadCountHyperparamsFromDB recalibre le Bias des CountHyperparams kill/death à
// partir des moyennes empiriques du batch, en gardant poids (w_p, w_o) et
// variance d'observation aux valeurs par défaut.
//
// Modèle (cf. ep/count_obs.go) : expected_count = bias + w_p·perf + w_o·avg_opp.
// À skill moyen (perf = avg_opp = mu0) on veut expected_count = moyenne
// empirique, d'où :
//
//	bias = mean - (w_p + w_o)·mu0
//
// ⚠️ Le plan d'origine (.ai/LUSR_V2_ROADMAP_SPRINTS.md, étape 1.B.2) prescrivait
// "bias = kill_mean_empirical" en direct — c'est dimensionnellement faux pour ce
// modèle (poserait expected ≈ 2× la moyenne à skill moyen). On applique la
// formule correcte, qui se réduit exactement aux défauts pour une moyenne
// typique de ~12.5 (kill → bias 0, death → bias 25).
func LoadCountHyperparamsFromDB(params map[string]float64, mu0 float64) map[CountType]CountHyperparams {
	out := DefaultCountHyperparamsMap()
	if mean, ok := params[hyperparamKillMean]; ok {
		h := out[CountKill]
		h.Bias = mean - (h.WeightPlayer+h.WeightOpponent)*mu0
		out[CountKill] = h
	}
	if mean, ok := params[hyperparamDeathMean]; ok {
		h := out[CountDeath]
		h.Bias = mean - (h.WeightPlayer+h.WeightOpponent)*mu0
		out[CountDeath] = h
	}
	return out
}

// AppliedHyperparamCount retourne le nombre d'hyperparams empiriques reconnus
// présents dans params. Sert au log de wiring ("hyperparams ré-estimés
// appliqués", count > 0).
func AppliedHyperparamCount(params map[string]float64) int {
	n := 0
	for _, k := range []string{hyperparamDrawProbability, hyperparamKillMean, hyperparamDeathMean} {
		if _, ok := params[k]; ok {
			n++
		}
	}
	return n
}
