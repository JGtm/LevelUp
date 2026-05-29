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

// PlayerCounts capture les counts individuels d'un joueur (kills/deaths)
// observés pendant un match + son statut "quit" (TS2 §9).
// Utilisé par UpdateTwoTeamWithCountsEP pour injecter le signal individuel.
type PlayerCounts struct {
	Kills  *float64 // nil = pas d'obs (count ignoré)
	Deaths *float64

	// Quit indique que le joueur a quitté en cours de match (TS2 §9).
	// QuitPenaltyDelta = magnitude de la pénalité (par défaut DefaultQuitDelta).
	// Si Quit=false, QuitPenaltyDelta est ignoré.
	Quit             bool
	QuitPenaltyDelta float64

	// Weight est le poids du joueur dans la somme team_perf = Σ wᵢ·perfᵢ
	// (TS2 : wᵢ = time_played_i / match_length, cf. ep/sum_factor.go). Permet de
	// down-weighter quitters/latecomers proportionnellement à leur temps de jeu
	// RÉEL (countdown retranché, via MatchTimeline.GameplayDuration). 0 (ou non
	// renseigné) → 1.0 (TS classique, participation pleine).
	Weight float64
}

// CountInputs structure les counts par équipe pour rester aligné avec
// TeamA/TeamB. Doit avoir len(TeamA) == len(TeamA) etc.
type CountInputs struct {
	TeamA []PlayerCounts // len doit matcher m.TeamA
	TeamB []PlayerCounts // len doit matcher m.TeamB
}

// DefaultQuitDeltaRelated : pénalité quit "related" (team perdait au moment
// du quit — quitter aurait perdu de toute façon, pénalité modérée).
// Calibré pour ≈ β/2 en unités v2 (β ≈ 4.17 → δ ≈ 2.0).
const DefaultQuitDeltaRelated = 1.0

// DefaultQuitDeltaUnrelated : pénalité quit "unrelated" (team gagnait/égalisait
// au moment du quit — quitter a abandonné une situation favorable, pénalité forte).
// Calibré pour ≈ β en unités v2.
const DefaultQuitDeltaUnrelated = 2.5

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

// UpdateTwoTeamWithCountsEP : variante TS2 §8 — intègre les counts (kills,
// deaths) par joueur comme observations Bayésiennes. C'est le signal qui
// permet de DÉPARTAGER les coéquipiers d'un même squad (cf. verdict Phase 1d
// où Madina/Choco/JGtm bougeaient ensemble parce que win/loss seul ne
// discrimine pas dans un squad).
//
// counts.TeamA[i] = obs du joueur i de TeamA. Si counts.TeamA[i] est nul
// (Kills et Deaths nil), aucune obs n'est posée pour ce joueur. Cela permet
// de mixer joueurs trackés (obs complète) et adversaires (pas d'obs).
//
// Si counts est nil → équivalent à UpdateTwoTeamEP (pas d'obs).
func UpdateTwoTeamWithCountsEP(m TwoTeamMatch, counts *CountInputs, p Priors) (teamA, teamB []Gaussian, err error) {
	if p.Beta <= 0 {
		return nil, nil, fmt.Errorf("skill_v2: Beta doit être > 0 (reçu %v)", p.Beta)
	}
	if counts != nil {
		if len(counts.TeamA) != len(m.TeamA) {
			return nil, nil, fmt.Errorf("skill_v2: counts.TeamA len %d != m.TeamA len %d", len(counts.TeamA), len(m.TeamA))
		}
		if len(counts.TeamB) != len(m.TeamB) {
			return nil, nil, fmt.Errorf("skill_v2: counts.TeamB len %d != m.TeamB len %d", len(counts.TeamB), len(m.TeamB))
		}
	}

	teamAIn, err := toEpGaussians(m.TeamA)
	if err != nil {
		return nil, nil, err
	}
	teamBIn, err := toEpGaussians(m.TeamB)
	if err != nil {
		return nil, nil, err
	}

	result, err := toEpResult(m.ResultA)
	if err != nil {
		return nil, nil, err
	}

	drawMargin := DrawMargin(p.DrawProbability, len(m.TeamA), len(m.TeamB), p.Beta)

	input := ep.Match2TeamInput{
		TeamA:      teamAIn,
		TeamB:      teamBIn,
		ResultA:    result,
		Beta:       p.Beta,
		Tau:        p.Tau,
		DrawMargin: drawMargin,
	}
	if counts != nil {
		input.Counts = flattenCountInputs(counts)
		input.WeightsA = extractTeamWeights(counts.TeamA)
		input.WeightsB = extractTeamWeights(counts.TeamB)
	}

	postA, postB, err := ep.UpdateMatch2Team(input, ep.DefaultMatch2TeamConfig())
	if err != nil {
		return nil, nil, err
	}
	finalA := fromEpGaussians(postA)
	finalB := fromEpGaussians(postB)
	if counts != nil {
		applyQuitPenaltyPost(finalA, counts.TeamA)
		applyQuitPenaltyPost(finalB, counts.TeamB)
	}
	return finalA, finalB, nil
}

// extractTeamWeights projette les poids team-sum (PlayerCounts.Weight) en
// []float64 aligné sur l'équipe. Retourne nil si aucun poids non nul (→ l'EP
// retombe sur wᵢ=1 partout, TS classique).
func extractTeamWeights(pcs []PlayerCounts) []float64 {
	any := false
	out := make([]float64, len(pcs))
	for i, pc := range pcs {
		out[i] = pc.Weight
		if pc.Weight > 0 {
			any = true
		}
	}
	if !any {
		return nil
	}
	return out
}

// applyQuitPenaltyPost applique le quit penalty en POST-EP : pour chaque
// joueur marqué Quit, on retire QuitPenaltyDelta (par défaut
// DefaultQuitDeltaRelated) à μ. Cette implémentation pragmatique encode le
// quit penalty "communauté gaming" (= punir le quitter avec une baisse de
// rating), distinct du modèle TS2 §9 intrinsèque (= absorber la baisse de
// perf dans une variable under_i pour épargner le skill — qui aurait l'effet
// inverse, REMONTER le skill du quitter).
//
// Choix Phase 3-quit : on prend l'interprétation "punir" parce que c'est ce
// que la communauté Halo / le système CSR existant suggère, et c'est aussi
// l'interprétation la plus utile UX (un joueur ne devrait pas pouvoir éviter
// la perte de rating en quittant).
//
// σ n'est pas modifié : on garde la confiance EP. Un raffinement futur
// pourrait widen σ légèrement (e.g., +β/4) pour signaler que ce match est
// "moins informatif".
func applyQuitPenaltyPost(team []Gaussian, counts []PlayerCounts) {
	for i, pc := range counts {
		if !pc.Quit {
			continue
		}
		d := pc.QuitPenaltyDelta
		if d <= 0 {
			d = DefaultQuitDeltaRelated
		}
		team[i].Mu -= d
	}
}

// flattenCountInputs convertit la représentation par-équipe en liste plate
// de CountObservation que le builder ep consomme.
func flattenCountInputs(c *CountInputs) []ep.CountObservation {
	out := make([]ep.CountObservation, 0, 2*(len(c.TeamA)+len(c.TeamB)))
	for i, pc := range c.TeamA {
		if pc.Kills != nil {
			out = append(out, ep.CountObservation{
				PlayerIndex: i, Side: ep.SideA, Type: ep.CountKill, Value: *pc.Kills,
			})
		}
		if pc.Deaths != nil {
			out = append(out, ep.CountObservation{
				PlayerIndex: i, Side: ep.SideA, Type: ep.CountDeath, Value: *pc.Deaths,
			})
		}
	}
	for j, pc := range c.TeamB {
		if pc.Kills != nil {
			out = append(out, ep.CountObservation{
				PlayerIndex: j, Side: ep.SideB, Type: ep.CountKill, Value: *pc.Kills,
			})
		}
		if pc.Deaths != nil {
			out = append(out, ep.CountObservation{
				PlayerIndex: j, Side: ep.SideB, Type: ep.CountDeath, Value: *pc.Deaths,
			})
		}
	}
	return out
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
