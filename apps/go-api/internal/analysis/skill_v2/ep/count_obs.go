package ep

// count_obs.go : observation Bayésienne des kills/deaths individuels (TS2 §8).
//
// Modèle génératif (forme MVP Phase 3c) :
//
//	expected_count_i = w_p · perf_i + w_o · avg(perf_opp_j)
//	observed_count_i ~ N(expected_count_i, v)             (Gaussienne observée)
//
// Le `max(0, ...)` du paper est ignoré côté Phase 3c — pour les counts > 0
// (cas le plus fréquent en pratique) le modèle tronqué se réduit à la
// Gaussienne. Phase ultérieure pourra ajouter le facteur tronqué propre
// pour les matchs très courts où count = 0 est fréquent.
//
// Conventions de signe (Halo 5, paper §8) :
//
//	type     w_p    w_o    rationale
//	kill     > 0    < 0    kills monte si TON perf monte, baisse si adversaires forts
//	death    < 0    > 0    deaths baisse si TON perf monte, monte si adversaires forts
//
// Hyperparams par défaut basés sur les figures du paper. À ré-estimer en Phase 5.

// CountType identifie le type d'événement observé.
type CountType int

const (
	CountKill CountType = iota
	CountDeath
)

// CountObservation représente une observation kill_i ou death_i pour un joueur.
type CountObservation struct {
	PlayerIndex int  // index dans TeamA ou TeamB (selon Side)
	Side        Side // équipe d'appartenance du joueur (A ou B)
	Type        CountType
	Value       float64 // valeur observée (typiquement entier ≥ 0)
}

// Side : équipe d'appartenance d'un joueur.
type Side int

const (
	SideA Side = iota
	SideB
)

// CountHyperparams regroupe (b, w_p, w_o, v) pour un type de count donné.
//
// Formule : expected_count = bias + w_p · perf_player + w_o · avg(perf_opp)
//
//	bias        ajuste l'expected vers une magnitude réaliste pour ce type
//	            d'observation (typiquement ~ 12 pour kills/deaths à la fin d'un match Slayer)
//	w_p > 0     pour kill (TON perf ↑ → TES kills ↑)
//	w_p < 0     pour death (TON perf ↑ → TES deaths ↓)
//	w_o         signe opposé à w_p (adversaire fort retient tes kills / inflige des deaths)
//	v           variance de l'observation (plus elle est grande, moins le signal pèse)
//
// Le bias est CRITIQUE pour des priors avec μ_0 large (échelle TrueSkill native
// μ_0 = 25). Le paper Halo 5 utilise μ_0 = 3 et bias = 0 implicite ; sur notre
// échelle 8× plus grande, sans bias l'expected_death devient négatif et le
// modèle pousse tous les joueurs vers le bas (cf. trace Phase 3c initiale).
type CountHyperparams struct {
	Bias           float64 // intercept additif
	WeightPlayer   float64 // w_p
	WeightOpponent float64 // w_o
	ObservationVar float64 // v
}

// DefaultCountHyperparams retourne des paramètres initiaux raisonnables,
// calibrés pour DefaultPriors() (μ_0 = 25, σ_0 ≈ 8.33).
//
// Pour un joueur "typique" (perf ≈ 25) vs des adversaires typiques (avg ≈ 25),
// expected_count_kill ≈ 12.5 et expected_count_death ≈ 12.5 — ordres de grandeur
// cohérents avec les match-stats Halo Slayer (~10-15 kills/deaths par joueur).
//
// Calculs :
//
//	kill  : bias=0, w_p=1, w_o=-0.5 → expected = 0 + 25 - 12.5 = 12.5
//	death : bias=25, w_p=-1, w_o=+0.5 → expected = 25 - 25 + 12.5 = 12.5
//
// À ré-estimer sur les données réelles (Phase 5 batch).
func DefaultCountHyperparams(t CountType) CountHyperparams {
	switch t {
	case CountKill:
		return CountHyperparams{Bias: 0, WeightPlayer: 1.0, WeightOpponent: -0.5, ObservationVar: 25.0}
	case CountDeath:
		return CountHyperparams{Bias: 25.0, WeightPlayer: -1.0, WeightOpponent: 0.5, ObservationVar: 25.0}
	default:
		return CountHyperparams{Bias: 0, WeightPlayer: 1.0, WeightOpponent: 0, ObservationVar: 25.0}
	}
}

// addCountObservationFactors construit, pour chaque observation, le sous-graph :
//
//	SumFactor(perf_player, perf_opp_1..M) -> expected_count_var
//	PriorFactor(expected_count_var, N(observed_value, v))
//
// et l'ajoute à la liste de facteurs. perfA/perfB sont les variables de
// performance déjà créées par makeVariables. Retourne les facteurs ajoutés
// (à appender à la liste du runner).
//
// Si une observation référence un PlayerIndex hors bornes, elle est skippée
// silencieusement (résilience aux bugs callers).
func addCountObservationFactors(
	obs []CountObservation,
	perfA, perfB []*Variable,
	hypByType map[CountType]CountHyperparams,
) []Factor {
	out := make([]Factor, 0, len(obs)*2)
	for _, o := range obs {
		playerPerf, oppPerfs := resolvePerfs(o.PlayerIndex, o.Side, perfA, perfB)
		if playerPerf == nil || len(oppPerfs) == 0 {
			continue
		}
		h, ok := hypByType[o.Type]
		if !ok {
			h = DefaultCountHyperparams(o.Type)
		}
		expected := NewVariable(countVarName(o))

		// SumFactor : expected = w_p · perf_player + (w_o / M) · Σ perf_opp
		// (le facteur w_o/M donne la "moyenne" de l'équipe adverse).
		sources := make([]*Variable, 0, 1+len(oppPerfs))
		weights := make([]float64, 0, 1+len(oppPerfs))
		sources = append(sources, playerPerf)
		weights = append(weights, h.WeightPlayer)
		wOpp := h.WeightOpponent / float64(len(oppPerfs))
		for _, p := range oppPerfs {
			sources = append(sources, p)
			weights = append(weights, wOpp)
		}
		out = append(out, NewSumFactor("count_sum_"+countVarName(o), expected, sources, weights))

		// PriorFactor : observation = N(value - bias, v). Joue le rôle de la
		// likelihood Gaussienne pour le count observé. Le bias retire la
		// constante de la formule pour que `expected_count` (sortie du SumFactor)
		// matche la résiduelle linéaire des perfs.
		obsPrior, err := FromMeanVariance(o.Value-h.Bias, h.ObservationVar)
		if err != nil {
			continue
		}
		out = append(out, NewPriorFactor("count_obs_"+countVarName(o), expected, obsPrior))
	}
	return out
}

// resolvePerfs retourne (perf du joueur, perfs des adversaires) selon Side.
func resolvePerfs(playerIdx int, side Side, perfA, perfB []*Variable) (*Variable, []*Variable) {
	switch side {
	case SideA:
		if playerIdx < 0 || playerIdx >= len(perfA) {
			return nil, nil
		}
		return perfA[playerIdx], perfB
	case SideB:
		if playerIdx < 0 || playerIdx >= len(perfB) {
			return nil, nil
		}
		return perfB[playerIdx], perfA
	default:
		return nil, nil
	}
}

func countVarName(o CountObservation) string {
	side := "A"
	if o.Side == SideB {
		side = "B"
	}
	typ := "kill"
	if o.Type == CountDeath {
		typ = "death"
	}
	return side + "_" + fmtInt(o.PlayerIndex) + "_" + typ
}

func fmtInt(i int) string {
	if i < 0 {
		return "neg" + fmtInt(-i)
	}
	if i < 10 {
		return string(rune('0' + i))
	}
	return fmtInt(i/10) + fmtInt(i%10)
}
