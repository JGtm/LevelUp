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
	"sync"

	skillv2 "levelup/go-api/internal/analysis/skill_v2"
)

// solverCountFallbackTotal compte les tentatives de repli : chaque calcul dont le solveur n'a pas
// convergé avec les comptes et qui est recalculé sans eux. Un même match peut être retenté (nouveau
// cycle, rejeu), le compteur dépasse alors le nombre de matchs distincts. Publié sous /debug/vars ;
// un second échec du recalcul est compté en plus par le classement de noteComputeFailure.
var solverCountFallbackTotal = expvar.NewInt("levelup.lusr_v2.solver_count_fallback_total")

// repliExemplesMax borne les identifiants de match cités par la ligne de synthèse d'un rejeu.
const repliExemplesMax = 3

// repliTally compte, pour UN rejeu d'un joueur, les matchs recalculés sans frags ni morts. Posé
// sur le ctx du rejeu (withRepliTally) : le repli d'un match s'y inscrit au lieu d'écrire une ligne
// par match, et la fin du rejeu écrit une seule synthèse (logSummary). Sans tally sur le ctx
// (appel direct, outil d'étude), le repli journalise lui-même chaque match en WARN.
type repliTally struct {
	mu       sync.Mutex
	matchs   int
	exemples []string
}

type repliTallyKey struct{}

func withRepliTally(ctx context.Context) (context.Context, *repliTally) {
	t := &repliTally{}
	return context.WithValue(ctx, repliTallyKey{}, t), t
}

func repliTallyFrom(ctx context.Context) *repliTally {
	t, _ := ctx.Value(repliTallyKey{}).(*repliTally)
	return t
}

func (t *repliTally) noter(matchID string) {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.matchs++
	if len(t.exemples) < repliExemplesMax {
		t.exemples = append(t.exemples, matchID)
	}
}

// logSummary écrit la synthèse du rejeu s'il y a eu au moins un repli : matchs rejoués, matchs en
// repli, quelques identifiants d'exemple. Le détail par match est en DEBUG (solveWithCountFallback).
func (t *repliTally) logSummary(ctx context.Context, xuid string, rejoues int) {
	t.mu.Lock()
	defer t.mu.Unlock()
	if t.matchs == 0 {
		return
	}
	slog.WarnContext(ctx, "LUSR v2: matchs recalculés sans frags ni morts, solveur non convergé avec les comptes",
		"xuid", xuid, "matchs_rejoues", rejoues, "matchs_en_repli", t.matchs,
		"exemples_match_id", t.exemples)
}

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

// solveMatchWithCounts est l'unique point d'appel de skillv2.UpdateTwoTeamWithCountsEP hors du
// moteur : tout calcul LUSR d'un match passe par le repli (garde-rail
// TestUpdateTwoTeamWithCountsEPAppeleSeulementParLeRepli).
func solveMatchWithCounts(ctx context.Context, in countedMatch) (teamA, teamB []skillv2.Gaussian, err error) {
	return solveWithCountFallback(ctx, skillv2.UpdateTwoTeamWithCountsEP, in)
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
	if t := repliTallyFrom(ctx); t != nil {
		t.noter(in.matchID)
		slog.DebugContext(ctx, "LUSR v2: solveur non convergé avec les comptes, recalcul sans frags ni morts",
			"match_id", in.matchID, "err", err)
	} else {
		slog.WarnContext(ctx, "LUSR v2: solveur non convergé avec les comptes, recalcul sans frags ni morts",
			"match_id", in.matchID, "err", err)
	}
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
