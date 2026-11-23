// Package v2 — budget.go : partage du budget de rattrapage entre les joueurs d'un même cycle.
//
// Le budget d'une passe (historystate.PassBudget) est celui du CYCLE, pas d'un joueur : chaque match
// inconnu listé est tenu en mémoire jusqu'à sa persistance, et N joueurs au rattrapage à 200 chacun
// en tenaient N fois 200. Le budget est réparti également entre les joueurs au rattrapage, au moins
// catchUpMinBudget chacun ; au-delà de PassBudget / catchUpMinBudget joueurs, seuls les premiers par
// ordre stable (identifiant de joueur) rattrapent ce cycle, les autres font un delta ordinaire et
// restent en attente (HistoryPending) ; leur delta est borné à catchUpMinBudget matchs (fetchers.go), les
// plus récents : un joueur sans historique rangé n'y tient pas 500 matchs en mémoire. Le delta d'un joueur
// à l'historique complet n'entre pas dans ce budget : il ne liste que les matchs neufs.
package v2

import (
	"context"
	"log/slog"
	"slices"

	"levelup/go-api/internal/ctxkeys"
	"levelup/go-api/internal/observability"
	"levelup/go-api/internal/sync/historystate"
)

// catchUpMinBudget est le budget minimal d'un joueur au rattrapage : en dessous, la passe avance trop peu
// pour valoir ses appels de liste (une page en compte 25).
const catchUpMinBudget = 25

// counterHistoryDeferred : joueurs au rattrapage dont le tour est reporté au cycle suivant faute de budget.
const counterHistoryDeferred = "sync_v2_history_deferred_total"

// catchUpPlan est la répartition du budget du cycle entre les joueurs au rattrapage.
type catchUpPlan struct {
	budget   int             // budget de chaque joueur qui rattrape
	deferred map[string]bool // joueurs au rattrapage sans place ce cycle (PlayerSlug)
}

// planCatchUp répartit le budget du cycle entre les joueurs dont l'historique est incomplet
// (HistoryState.Incomplete) parmi `states`, indexés par PlayerSlug.
func planCatchUp(states map[string]KnownState) catchUpPlan {
	var catching []string
	for slug, st := range states {
		if st.History.Incomplete {
			catching = append(catching, slug)
		}
	}
	slices.Sort(catching)
	total := historystate.PassBudget()
	plan := catchUpPlan{budget: total, deferred: map[string]bool{}}
	if len(catching) == 0 {
		return plan
	}
	if maxPlayers := total / catchUpMinBudget; len(catching) > maxPlayers {
		for _, slug := range catching[maxPlayers:] {
			plan.deferred[slug] = true
		}
		catching = catching[:maxPlayers]
	}
	plan.budget = total / len(catching)
	return plan
}

// apply rend l'état de l'historique d'un joueur tel que la découverte le voit : son budget, ou le delta
// ordinaire quand il n'a pas de place ce cycle.
func (pl catchUpPlan) apply(slug string, h HistoryState) HistoryState {
	if !h.Incomplete {
		return h
	}
	if pl.deferred[slug] {
		h.Incomplete, h.Deferred = false, true
		h.Cursor = nil
		return h
	}
	h.Budget = pl.budget
	return h
}

// noteDeferred journalise et compte un joueur dont le rattrapage attend le cycle suivant.
func (pl catchUpPlan) noteDeferred(ctx context.Context, slug string) {
	observability.IncCounterT(ctxkeys.TitleSlug(ctx), counterHistoryDeferred)
	slog.InfoContext(ctx, "sync.v2: rattrapage d'historique reporté au cycle suivant — budget du cycle réparti entre les autres joueurs",
		"event", "sync.v2.history.deferred", "player", slug, "budget_per_player", pl.budget)
}
