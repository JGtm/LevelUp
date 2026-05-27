package skill_v2

// trueskill_ep.go : variante EP de UpdateTwoTeam.
//
// Wrap UpdateMatch2Team du sous-package ep en convertissant les types
// (μ,σ) ↔ canonique. Sémantique identique à UpdateTwoTeam : mêmes
// inputs, mêmes outputs, mêmes priors. Sert de référence pour les
// extensions TS2 §8/§11 qui nécessiteront EP.

import (
	"fmt"

	"levelup/go-api/internal/analysis/skill_v2/ep"
)

// UpdateTwoTeamEP : équivalent EP de UpdateTwoTeam.
//
// Garantit numériquement le même résultat (au tolérance EP près, typiquement
// 1e-3 sur μ et 1e-3 sur σ). À utiliser en remplacement du closed-form pour
// préparer Phase 3c (kills/deaths comme observations) qui ajoutera des facteurs
// supplémentaires au graph — le closed-form ne pouvant pas exprimer ces facteurs.
func UpdateTwoTeamEP(m TwoTeamMatch, p Priors) (teamA, teamB []Gaussian, err error) {
	if p.Beta <= 0 {
		return nil, nil, fmt.Errorf("skill_v2: Beta doit être > 0 (reçu %v)", p.Beta)
	}

	teamAIn, err := toEpGaussians(m.TeamA)
	if err != nil {
		return nil, nil, fmt.Errorf("convert teamA: %w", err)
	}
	teamBIn, err := toEpGaussians(m.TeamB)
	if err != nil {
		return nil, nil, fmt.Errorf("convert teamB: %w", err)
	}

	result, err := toEpResult(m.ResultA)
	if err != nil {
		return nil, nil, err
	}

	// DrawMargin est déjà défini dans le package parent (math.go Phase 1a).
	drawMargin := DrawMargin(p.DrawProbability, len(m.TeamA), len(m.TeamB), p.Beta)

	postA, postB, err := ep.UpdateMatch2Team(ep.Match2TeamInput{
		TeamA:      teamAIn,
		TeamB:      teamBIn,
		ResultA:    result,
		Beta:       p.Beta,
		Tau:        p.Tau,
		DrawMargin: drawMargin,
	}, ep.DefaultMatch2TeamConfig())
	if err != nil {
		return nil, nil, err
	}

	return fromEpGaussians(postA), fromEpGaussians(postB), nil
}

func toEpGaussians(team []Gaussian) ([]ep.Gaussian, error) {
	out := make([]ep.Gaussian, len(team))
	for i, g := range team {
		if g.Sigma <= 0 {
			return nil, fmt.Errorf("skill_v2: sigma <= 0 pour joueur %d (%v)", i, g.Sigma)
		}
		eg, err := ep.FromMeanSigma(g.Mu, g.Sigma)
		if err != nil {
			return nil, fmt.Errorf("skill_v2: convert joueur %d: %w", i, err)
		}
		out[i] = eg
	}
	return out, nil
}

func fromEpGaussians(team []ep.Gaussian) []Gaussian {
	out := make([]Gaussian, len(team))
	for i, g := range team {
		out[i] = Gaussian{Mu: g.Mu(), Sigma: g.Sigma()}
	}
	return out
}

func toEpResult(r TeamResult) (ep.TeamResult, error) {
	switch r {
	case TeamWin:
		return ep.TeamWin, nil
	case TeamLoss:
		return ep.TeamLoss, nil
	case TeamDraw:
		return ep.TeamDraw, nil
	default:
		return 0, fmt.Errorf("skill_v2: TeamResult invalide (%d)", r)
	}
}
