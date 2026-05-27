package ep

import (
	"fmt"
	"math"
)

// match2team.go : assemble le factor graph d'un match 2-équipes et résout EP
// pour produire les posteriors.
//
// Le graph reproduit la structure générative TS classique :
//
//	skill_{A,i} ~ N(prior_{A,i})
//	skill_{B,j} ~ N(prior_{B,j})
//	perf_{A,i}  ~ N(skill_{A,i}, β²)
//	perf_{B,j}  ~ N(skill_{B,j}, β²)
//	team_perf_A = Σ perf_{A,i}
//	team_perf_B = Σ perf_{B,j}
//	diff        = team_perf_A - team_perf_B
//	observed    : diff > ε    (TeamWin)
//	            | diff < -ε   (TeamLoss, encodé via swap)
//	            | |diff| < ε  (TeamDraw, WithinFactor)
//
// Après convergence EP, on lit les marginaux des skill_*, on ajoute τ² aux
// variances (random walk dynamique, identique au closed-form Phase 1a).
//
// Cette implémentation produit numériquement les mêmes résultats que le
// closed-form de skill_v2.UpdateTwoTeam à eps près (régression test).

// TeamResult duplique le type du package parent pour éviter une dépendance
// cyclique. Garde la même sémantique (mêmes valeurs entières).
type TeamResult int

const (
	TeamWin  TeamResult = iota // équipe gagne
	TeamLoss                   // équipe perd
	TeamDraw                   // match nul
)

// Match2TeamInput décrit le match à inférer en EP.
type Match2TeamInput struct {
	TeamA      []Gaussian
	TeamB      []Gaussian
	ResultA    TeamResult
	Beta       float64 // bruit perf (β en notation TS)
	Tau        float64 // évolution dynamique (τ en notation TS)
	DrawMargin float64 // ε de draw, déjà calculé via DrawMargin(prob, n_a, n_b, beta)

	// Counts (TS2 §8, Phase 3c). Optionnels : si vide, le match utilise
	// uniquement le signal win/loss/draw (équivalent TS classique).
	//
	// Chaque CountObservation lie un (playerIdx, side, type) à une valeur
	// observée. Le solver crée alors un sous-graphe SumFactor + PriorFactor
	// pour chaque obs (cf. addCountObservationFactors).
	Counts            []CountObservation
	CountHyperparams  map[CountType]CountHyperparams // optionnel ; défaut = DefaultCountHyperparams
}

// Match2TeamConfig regroupe les paramètres EP — séparés des hyperparams skill
// pour pouvoir tester le solver avec des seuils plus agressifs.
type Match2TeamConfig struct {
	Tolerance float64 // delta max accepté pour convergence (1e-4 défaut)
	MaxIters  int     // itérations max (50 défaut)
}

// DefaultMatch2TeamConfig retourne des paramètres de solver conservateurs,
// adaptés aux factor graphs TS classiques (win/loss seul) qui convergent en
// 5-20 itérations.
//
// Phase 3c (count observations) ajoute des facteurs additionnels — le graph
// peut nécessiter plus d'itérations, surtout sur des scenarios contradictoires
// (ex : équipe gagne mais joueur a des stats faibles, ou tailles d'équipes
// légèrement asymétriques 4v5/5v4 fréquents post-quit). MaxIters bumped à 500
// pour absorber ces cas. La sortie reste correcte pour les cas convergents
// en < 50 itérations.
//
// À monitorer : si les warns "EP n'a pas convergé" deviennent fréquents,
// soit augmenter MaxIters encore, soit introduire du damping EP, soit ajouter
// un skip plus agressif côté caller (cf. isTeamImbalanceTooHigh).
func DefaultMatch2TeamConfig() Match2TeamConfig {
	return Match2TeamConfig{Tolerance: 1e-4, MaxIters: 500}
}

// UpdateMatch2Team construit le graph EP, le résout, applique τ² aux variances
// et retourne les posteriors (mêmes shapes que TeamA, TeamB).
func UpdateMatch2Team(m Match2TeamInput, cfg Match2TeamConfig) (teamA, teamB []Gaussian, err error) {
	if len(m.TeamA) == 0 || len(m.TeamB) == 0 {
		return nil, nil, fmt.Errorf("ep: équipe vide (nA=%d, nB=%d)", len(m.TeamA), len(m.TeamB))
	}
	if m.Beta <= 0 {
		return nil, nil, fmt.Errorf("ep: Beta doit être > 0 (reçu %v)", m.Beta)
	}

	// Pour TeamLoss : on swap les équipes pour traiter comme TeamWin de B.
	// On démêle à la fin avant de renvoyer.
	swapped := false
	teamAIn, teamBIn := m.TeamA, m.TeamB
	if m.ResultA == TeamLoss {
		teamAIn, teamBIn = m.TeamB, m.TeamA
		swapped = true
	}

	nA, nB := len(teamAIn), len(teamBIn)
	skillA, skillB, perfA, perfB := makeVariables(nA, nB)
	diff := NewVariable("diff")

	factors := buildFactors(buildFactorsArgs{
		teamA: teamAIn, teamB: teamBIn,
		skillA: skillA, skillB: skillB,
		perfA: perfA, perfB: perfB,
		teamPerfA: NewVariable("team_perf_A"),
		teamPerfB: NewVariable("team_perf_B"),
		diff:      diff,
		betaVar:   m.Beta * m.Beta,
		result:    m.ResultA,
		swapped:   swapped,
		drawMargin: m.DrawMargin,
	})

	// Counts (Phase 3c) — append après les facteurs du match principal.
	// On adapte le PlayerIndex/Side si on a swap pour TeamLoss : SideA dans
	// l'input devient SideB en interne et vice versa.
	if len(m.Counts) > 0 {
		hyp := m.CountHyperparams
		if hyp == nil {
			hyp = map[CountType]CountHyperparams{
				CountKill:  DefaultCountHyperparams(CountKill),
				CountDeath: DefaultCountHyperparams(CountDeath),
			}
		}
		adjustedCounts := m.Counts
		if swapped {
			adjustedCounts = make([]CountObservation, len(m.Counts))
			for i, o := range m.Counts {
				o.Side = flipSide(o.Side)
				adjustedCounts[i] = o
			}
		}
		factors = append(factors, addCountObservationFactors(adjustedCounts, perfA, perfB, hyp)...)
	}

	runner := NewRunner(factors)
	runner.Tolerance = cfg.Tolerance
	runner.MaxIters = cfg.MaxIters
	_, _, converged := runner.Run()
	if !converged {
		return nil, nil, fmt.Errorf("ep: EP n'a pas convergé en %d itérations", cfg.MaxIters)
	}

	outA := extractPosteriors(skillA, m.Tau)
	outB := extractPosteriors(skillB, m.Tau)
	if swapped {
		outA, outB = outB, outA
	}
	return outA, outB, nil
}

func makeVariables(nA, nB int) (skillA, skillB, perfA, perfB []*Variable) {
	skillA = make([]*Variable, nA)
	perfA = make([]*Variable, nA)
	for i := 0; i < nA; i++ {
		skillA[i] = NewVariable(fmt.Sprintf("skill_A_%d", i))
		perfA[i] = NewVariable(fmt.Sprintf("perf_A_%d", i))
	}
	skillB = make([]*Variable, nB)
	perfB = make([]*Variable, nB)
	for j := 0; j < nB; j++ {
		skillB[j] = NewVariable(fmt.Sprintf("skill_B_%d", j))
		perfB[j] = NewVariable(fmt.Sprintf("perf_B_%d", j))
	}
	return
}

type buildFactorsArgs struct {
	teamA, teamB                 []Gaussian
	skillA, skillB, perfA, perfB []*Variable
	teamPerfA, teamPerfB, diff   *Variable
	betaVar                      float64
	result                       TeamResult
	swapped                      bool
	drawMargin                   float64
}

// buildFactors assemble la liste de facteurs dans l'ordre où ils doivent être
// mis à jour par le runner — priors d'abord (pour qu'ils se propagent dans
// le graph), puis liens, sommes, et enfin la contrainte d'outcome.
func buildFactors(a buildFactorsArgs) []Factor {
	factors := []Factor{}
	// Priors sur skills.
	for i, prior := range a.teamA {
		factors = append(factors, NewPriorFactor(fmt.Sprintf("prior_A_%d", i), a.skillA[i], gaussianFromPrior(prior)))
	}
	for j, prior := range a.teamB {
		factors = append(factors, NewPriorFactor(fmt.Sprintf("prior_B_%d", j), a.skillB[j], gaussianFromPrior(prior)))
	}
	// Liens skill → perf.
	for i := range a.teamA {
		factors = append(factors, NewLikelihoodFactor(fmt.Sprintf("link_A_%d", i), a.skillA[i], a.perfA[i], a.betaVar))
	}
	for j := range a.teamB {
		factors = append(factors, NewLikelihoodFactor(fmt.Sprintf("link_B_%d", j), a.skillB[j], a.perfB[j], a.betaVar))
	}
	// Sommes team_perf = Σ perf.
	weightsA := make([]float64, len(a.teamA))
	for i := range weightsA {
		weightsA[i] = 1
	}
	weightsB := make([]float64, len(a.teamB))
	for j := range weightsB {
		weightsB[j] = 1
	}
	factors = append(factors,
		NewSumFactor("team_sum_A", a.teamPerfA, a.perfA, weightsA),
		NewSumFactor("team_sum_B", a.teamPerfB, a.perfB, weightsB),
	)
	// diff = team_perf_A - team_perf_B.
	factors = append(factors, NewSumFactor("diff_sum", a.diff,
		[]*Variable{a.teamPerfA, a.teamPerfB}, []float64{1, -1}))
	// Contrainte d'outcome. Pour Win/Loss, swapped déjà inversé en amont
	// donc le facteur observé est toujours GreaterThan(diff > ε).
	if a.result == TeamDraw {
		factors = append(factors, NewWithinFactor("obs_draw", a.diff, a.drawMargin))
	} else {
		factors = append(factors, NewGreaterThanFactor("obs_win", a.diff, a.drawMargin))
	}
	return factors
}

func flipSide(s Side) Side {
	if s == SideA {
		return SideB
	}
	return SideA
}

func gaussianFromPrior(p Gaussian) Gaussian {
	// Si l'entrée vient du package parent (skill_v2.Gaussian), elle pourrait
	// être en (μ, σ) ; ici on attend déjà ep.Gaussian (canonique). Le caller
	// (typiquement skill_v2.UpdateTwoTeamEP en Phase 3b) convertit avant.
	if p.IsUniform() {
		// Garde-fou : un prior uniforme n'apporte pas d'info, mais on tolère
		// (utile pour les tests qui veulent simuler "premier match").
		return UniformGaussian()
	}
	return p
}

// extractPosteriors lit le marginal de chaque skill variable et ajoute τ² à
// la variance pour modéliser le random walk dynamique (équivalent au "+τ²"
// du closed-form Phase 1a après l'update).
func extractPosteriors(skills []*Variable, tau float64) []Gaussian {
	tau2 := tau * tau
	out := make([]Gaussian, len(skills))
	for i, s := range skills {
		marg := s.Marginal
		if marg.IsUniform() {
			out[i] = UniformGaussian()
			continue
		}
		// Variance augmentée → π diminue.
		v := marg.Variance() + tau2
		mu := marg.Mu()
		g, err := FromMeanVariance(mu, math.Max(v, 1e-12))
		if err != nil {
			out[i] = UniformGaussian()
			continue
		}
		out[i] = g
	}
	return out
}
