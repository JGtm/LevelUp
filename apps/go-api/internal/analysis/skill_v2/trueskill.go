package skill_v2

import (
	"fmt"
	"math"
)

// trueskill.go : closed-form TrueSkill classique pour matchs à 2 équipes.
//
// L'algorithme implémente la "online update" Bayésienne, suffisante pour
// Phase 1 (skill seul) et Phase 2 (squad offset additif). Phase 3 (kills/deaths
// comme observations) demandera un factor graph + EP, qui remplacera ce module.

// TeamResult décrit l'issue d'une équipe dans un match.
type TeamResult int

const (
	// TeamWin indique que l'équipe a gagné.
	TeamWin TeamResult = iota
	// TeamLoss indique que l'équipe a perdu.
	TeamLoss
	// TeamDraw indique que l'équipe a fait match nul avec l'adverse.
	TeamDraw
)

// TwoTeamMatch décrit un match 2 équipes prêt à être inféré.
// Les slices teamA et teamB contiennent les états AVANT-match de chaque joueur ;
// ResultA décrit l'issue de teamA. ResultB en est l'opposé exact (déduit, pas requis).
type TwoTeamMatch struct {
	TeamA   []Gaussian
	TeamB   []Gaussian
	ResultA TeamResult
}

// UpdateTwoTeam calcule les nouveaux skills posterior après un match 2 équipes.
// Retourne deux slices alignées sur TeamA/TeamB en entrée.
//
// La formule est dérivée de Herbrich 2007 §4 pour le cas 2 équipes :
//
//	c² = (n_A + n_B) β² + Σ σ_i²       (variance combinée)
//	t  = (Σ μ_A - Σ μ_B) / c           (skill gap normalisé du POV de A)
//	v, w = vWin(t, ε/c), wWin(t, ε/c)  (ou v/wDraw pour un draw)
//
//	Pour un i ∈ A (winner) :  μ_i ← μ_i + (σ_i²/c) v ; σ_i² ← σ_i² · (1 - (σ_i²/c²) w)
//	Pour un i ∈ B (loser)  :  μ_i ← μ_i - (σ_i²/c) v ; σ_i² ← σ_i² · (1 - (σ_i²/c²) w)
//	Pour un draw           :  signes selon le signe de t (qui est devant).
//
// Après la mise à jour, on ajoute τ² à σ_i² pour modéliser l'évolution dynamique
// (random walk dans Herbrich 2007 §3.1).
func UpdateTwoTeam(m TwoTeamMatch, p Priors) (teamA, teamB []Gaussian, err error) {
	nA, nB := len(m.TeamA), len(m.TeamB)
	if nA == 0 || nB == 0 {
		return nil, nil, fmt.Errorf("skill_v2: match avec une équipe vide (nA=%d, nB=%d)", nA, nB)
	}
	if p.Beta <= 0 {
		return nil, nil, fmt.Errorf("skill_v2: Beta doit être > 0 (reçu %v)", p.Beta)
	}

	muA, varA := sumMuVar(m.TeamA)
	muB, varB := sumMuVar(m.TeamB)

	c2 := float64(nA+nB)*p.Beta*p.Beta + varA + varB
	c := math.Sqrt(c2)
	eps := DrawMargin(p.DrawProbability, nA, nB, p.Beta)

	// Calcule t depuis le POV "winner": si A a perdu, on inverse le signe pour
	// que le facteur de correction agisse dans la bonne direction.
	var v, w, signA float64
	switch m.ResultA {
	case TeamWin:
		t := (muA - muB) / c
		v = vWin(t, eps/c)
		w = wWin(t, eps/c)
		signA = +1
	case TeamLoss:
		t := (muB - muA) / c
		v = vWin(t, eps/c)
		w = wWin(t, eps/c)
		signA = -1
	case TeamDraw:
		t := (muA - muB) / c
		v = vDraw(t, eps/c)
		w = wDraw(t, eps/c)
		// Pour un draw, le signe est dans v (dépend du signe de t).
		// On encode signA = +1 et on laisse v porter l'orientation.
		signA = +1
	default:
		return nil, nil, fmt.Errorf("skill_v2: ResultA invalide (%d)", m.ResultA)
	}

	teamA = applyTeamUpdate(m.TeamA, +signA, v, w, c, c2, p.Tau)
	teamB = applyTeamUpdate(m.TeamB, -signA, v, w, c, c2, p.Tau)
	return teamA, teamB, nil
}

// applyTeamUpdate met à jour chaque joueur d'une équipe via la formule closed-form.
// Le sign détermine la direction de la mise à jour de μ (winners → +v, losers → -v).
// w est partagé entre les deux équipes (réduction de variance symétrique).
//
// τ² est ajouté APRÈS la réduction par w — séquencement standard TrueSkill :
// (1) on met à jour le skill à partir du résultat, (2) on ajoute le bruit dynamique.
func applyTeamUpdate(team []Gaussian, sign, v, w, c, c2, tau float64) []Gaussian {
	out := make([]Gaussian, len(team))
	tau2 := tau * tau
	for i, g := range team {
		sigma2 := g.Variance()
		newMu := g.Mu + sign*(sigma2/c)*v
		newVar := sigma2 * (1.0 - (sigma2/c2)*w)
		// Garde-fou numérique : avec v/w bornés et c2 > 0, newVar > 0
		// analytiquement, mais l'arithmétique float64 peut le pousser au négatif
		// dans les queues extrêmes — on clamp à zéro avant d'ajouter τ².
		if newVar < 0 {
			newVar = 0
		}
		newVar += tau2
		out[i] = Gaussian{Mu: newMu, Sigma: math.Sqrt(newVar)}
	}
	return out
}

func sumMuVar(team []Gaussian) (mu, variance float64) {
	for _, g := range team {
		mu += g.Mu
		variance += g.Variance()
	}
	return mu, variance
}

// PredictWinProbability / PredictDrawProbability / PredictTwoTeamWinProb vivent
// désormais dans predict.go (partagent le helper matchSpread).
