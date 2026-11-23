// Package scheduler — auto_sync_catchup_budget.go : le budget de rattrapage d'un titre live-only, réparti entre
// ses joueurs.
//
// Un joueur d'un titre live-only (Halo 5) passe par syncPlayer, dont le runner rattrape l'historique par
// passes bornées par opts.MaxMatches (200 par défaut, historystate.PassBudget). Ce budget était donc par
// JOUEUR : sept joueurs en rattrapage sur le titre en rangeaient 1 400 par cycle, l'un après l'autre. Il est
// désormais par CYCLE ET PAR TITRE : les joueurs du titre dont l'historique est incomplet se partagent
// PassBudget, avec au moins minCatchUpShare matchs chacun ; au-delà de PassBudget / minCatchUpShare joueurs
// (8), les autres attendent le cycle suivant, les moins récemment passés d'abord, pour que chacun finisse par
// passer. Un joueur dont l'historique est complet garde ses options par défaut.
package scheduler

import (
	"context"
	"log/slog"
	"sort"
	"time"

	"levelup/go-api/internal/domain"
	"levelup/go-api/internal/games/halo_5/livesync"
	"levelup/go-api/internal/sync/historystate"
)

// minCatchUpShare est la part minimale du budget d'un titre pour un joueur en rattrapage : en dessous, une
// passe ne ferait plus avancer assez l'historique pour valoir un cycle.
const minCatchUpShare = 25

// catchUpPlan est la répartition du budget de rattrapage d'un cycle : la part de chaque joueur en rattrapage
// et ceux qui attendent le cycle suivant, par couple (titre, joueur).
type catchUpPlan struct {
	maxMatches map[string]int
	wait       map[string]bool
}

// waits dit si le budget du titre ne couvre pas ce joueur ce cycle.
func (pl catchUpPlan) waits(p domain.PlayerSummary) bool {
	return pl.wait[passKey(resolveTitleSlug(p), p.Gamertag)]
}

// optionsFor rend les options de passe du joueur : les options par défaut, avec sa part du budget quand il
// est en rattrapage.
func (pl catchUpPlan) optionsFor(p domain.PlayerSummary) domain.SyncOptions {
	opts := domain.DefaultSyncOptions()
	if n, ok := pl.maxMatches[passKey(resolveTitleSlug(p), p.Gamertag)]; ok {
		opts.MaxMatches = n
	}
	return opts
}

// planLiveCatchUp répartit, titre live-only par titre live-only, le budget de rattrapage du cycle entre les
// joueurs dont l'historique est incomplet (lu par historyIncomplete : une lecture en échec compte comme en
// rattrapage). Sans lecteur d'historique câblé, personne n'est en rattrapage : le plan est vide.
func (s *AutoSyncScheduler) planLiveCatchUp(ctx context.Context, players []domain.PlayerSummary) catchUpPlan {
	plan := catchUpPlan{maxMatches: map[string]int{}, wait: map[string]bool{}}
	if s.historyReader == nil {
		return plan
	}
	catching := map[string][]domain.PlayerSummary{}
	for _, p := range players {
		slug := resolveTitleSlug(p)
		if livesync.HandlesTitle(slug) && s.historyIncomplete(ctx, p) {
			catching[slug] = append(catching[slug], p)
		}
	}
	budget := historystate.PassBudget()
	maxPlayers := max(1, budget/minCatchUpShare)
	for slug, group := range catching {
		selected := group
		if len(group) > maxPlayers {
			s.sortStalestFirst(slug, group)
			selected = group[:maxPlayers]
			for _, p := range group[maxPlayers:] {
				plan.wait[passKey(slug, p.Gamertag)] = true
			}
			slog.InfoContext(ctx, "auto_sync: budget de rattrapage du titre réparti — des joueurs attendent le cycle suivant",
				"title_slug", slug, "en_rattrapage", len(group), "servis", len(selected),
				"attendent", len(group)-len(selected), "budget", budget)
		}
		share := budget / len(selected)
		for _, p := range selected {
			plan.maxMatches[passKey(slug, p.Gamertag)] = share
		}
	}
	return plan
}

// sortStalestFirst trie les joueurs du titre par passe la moins récente d'abord (un joueur sans passe connue
// en premier, puis par gamertag) : ceux qui ont attendu passent avant ceux qui viennent de passer.
func (s *AutoSyncScheduler) sortStalestFirst(slug string, group []domain.PlayerSummary) {
	last := make(map[string]time.Time, len(group))
	for _, p := range group {
		if pass, ok := s.LastPass(slug, p.Gamertag); ok {
			last[p.Gamertag] = pass.At
		}
	}
	sort.SliceStable(group, func(i, j int) bool {
		ti, tj := last[group[i].Gamertag], last[group[j].Gamertag]
		if !ti.Equal(tj) {
			return ti.Before(tj)
		}
		return group[i].Gamertag < group[j].Gamertag
	})
}

// deferCatchUp enregistre, pour le diagnostic, qu'un joueur en rattrapage n'a pas eu sa part du budget de son
// titre ce cycle. Sa dernière passe connue reste celle qu'elle était : rien n'a tourné.
func (s *AutoSyncScheduler) deferCatchUp(ctx context.Context, p domain.PlayerSummary) {
	slog.InfoContext(ctx, "auto_sync: rattrapage différé — le budget du titre est réparti entre les autres joueurs",
		"gamertag", p.Gamertag, "title_slug", resolveTitleSlug(p))
	s.recordOutcome(PlayerOutcomeDetail{
		Gamertag:               p.Gamertag,
		XUID:                   p.XUID,
		Outcome:                outcomeName(outcomeSkipped),
		Reason:                 "différé: budget de rattrapage du titre réparti, passe au cycle suivant",
		AttemptedAt:            s.clock(),
		ConsecutiveZeroInserts: s.previousZeroInsertCount(p.Gamertag),
	})
}
