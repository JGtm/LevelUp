package skill

// skill_v2_solver_fallback.go : repli du calcul LUSR v2 quand le solveur EP ne converge pas.
//
// Les observations de frags et de morts (un SumFactor et un PriorFactor par joueur contre la
// moyenne adverse) sont la cause des non-convergences mesurées : sans elles, le même match
// converge, poids de temps joué et pénalités d'abandon conservés. Un amortissement ne suffit
// pas, aucun réglage essayé ne rattrape plus de la moitié de ces matchs. Le repli recalcule donc
// le match une fois, sans ces observations ; le match reçoit une note au lieu de laisser un trou
// dans l'historique.
//
// Le repli est une logique de calcul : tout changement de son comportement impose de changer
// LUSRRev (scoring_revision.go), sans quoi un rejeu produirait un historique différent de celui
// déjà stocké.

import (
	"context"
	"errors"
	"expvar"
	"fmt"
	"log/slog"

	skillv2 "levelup/go-api/internal/analysis/skill_v2"
)

// solverCountFallbackTotal compte les matchs dont le solveur n'a pas convergé avec les comptes et
// qui ont été recalculés sans eux. Publié sous /debug/vars ; un second échec du recalcul est
// compté en plus par le classement de noteComputeFailure.
var solverCountFallbackTotal = expvar.NewInt("levelup.lusr_v2.solver_count_fallback_total")

// twoTeamSolver est la signature de skillv2.UpdateTwoTeamWithCountsEP, passée en paramètre pour
// que le repli se teste sans fabriquer un match qui diverge.
type twoTeamSolver func(m skillv2.TwoTeamMatch, counts *skillv2.CountInputs, p skillv2.Priors) (teamA, teamB []skillv2.Gaussian, err error)

// countedMatch regroupe les entrées d'un calcul EP avec comptes.
type countedMatch struct {
	matchID string
	match   skillv2.TwoTeamMatch
	counts  *skillv2.CountInputs
	priors  skillv2.Priors
}

// solveWithCountFallback résout le match avec ses comptes. Sur skillv2.ErrNotConverged seulement,
// et si des observations de comptes existent, il journalise puis recalcule le match sans elles.
// Toute autre erreur est rendue telle quelle ; l'échec du recalcul est rendu enveloppé (son erreur
// reste reconnue par errors.Is, donc classée comme avant par noteComputeFailure) : aucun échec n'est avalé.
func solveWithCountFallback(ctx context.Context, solve twoTeamSolver, in countedMatch) (teamA, teamB []skillv2.Gaussian, err error) {
	teamA, teamB, err = solve(in.match, in.counts, in.priors)
	if err == nil || !errors.Is(err, skillv2.ErrNotConverged) || !hasCountObservations(in.counts) {
		return teamA, teamB, err
	}
	solverCountFallbackTotal.Add(1)
	slog.WarnContext(ctx, "LUSR v2: solveur non convergé avec les comptes, recalcul sans frags ni morts",
		"match_id", in.matchID, "err", err)
	teamA, teamB, err = solve(in.match, withoutCountObservations(in.counts), in.priors)
	if err != nil {
		return nil, nil, fmt.Errorf("recalcul sans comptes: %w", err)
	}
	return teamA, teamB, nil
}

// hasCountObservations dit si un recalcul sans comptes diffère du calcul qui vient d'échouer.
func hasCountObservations(c *skillv2.CountInputs) bool {
	if c == nil {
		return false
	}
	for _, team := range [][]skillv2.PlayerCounts{c.TeamA, c.TeamB} {
		for _, pc := range team {
			if pc.Kills != nil || pc.Deaths != nil {
				return true
			}
		}
	}
	return false
}

// withoutCountObservations rend une copie de c où les frags et les morts de chaque joueur sont
// retirés ; le poids de temps joué et l'abandon sont conservés. c n'est pas modifié.
func withoutCountObservations(c *skillv2.CountInputs) *skillv2.CountInputs {
	strip := func(team []skillv2.PlayerCounts) []skillv2.PlayerCounts {
		out := make([]skillv2.PlayerCounts, len(team))
		for i, pc := range team {
			pc.Kills, pc.Deaths = nil, nil
			out[i] = pc
		}
		return out
	}
	return &skillv2.CountInputs{
		TeamA:       strip(c.TeamA),
		TeamB:       strip(c.TeamB),
		Hyperparams: c.Hyperparams,
	}
}
