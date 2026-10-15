package service

import (
	"context"
	"errors"
	"log/slog"
	"sync"

	"levelup/go-api/internal/domain"

	"golang.org/x/sync/errgroup"
)

// matchViewSource nomme une source de la vue match qui porte DEUX raisons partielles
// distinctes : `empty` (la lecture a réussi et ne rend rien) et `unavailable` (la
// lecture a échoué). Une lecture en échec ne doit jamais se lire comme « pas de
// données » (E4, décision D-8).
type matchViewSource struct {
	name        string
	empty       string
	unavailable string
}

// Les quatre sources dont la vue signale déjà le vide (detectPartialMatchData). Les
// autres chargeurs de goLoad restent best-effort sans raison dédiée.
var (
	srcScoreboard    = matchViewSource{"scoreboard", "scoreboard_empty", domain.PartialReasonScoreboardUnavailable}
	srcEvents        = matchViewSource{"events", "events_empty", domain.PartialReasonEventsUnavailable}
	srcPlayerStats   = matchViewSource{"player_stats", "player_stats_empty", domain.PartialReasonPlayerStatsUnavailable}
	srcMedals        = matchViewSource{"medals", "medals_empty", domain.PartialReasonMedalsUnavailable}
	matchViewSources = []matchViewSource{srcScoreboard, srcEvents, srcPlayerStats, srcMedals}
)

// sourceFailures collecte, de façon sûre entre goroutines, les sources dont la lecture a
// échoué pendant un chargement parallèle. Le zéro (nil) est valide : aucune panne.
type sourceFailures struct {
	mu     sync.Mutex
	failed map[string]struct{}
}

func newSourceFailures() *sourceFailures {
	return &sourceFailures{failed: make(map[string]struct{})}
}

func (f *sourceFailures) mark(src matchViewSource) {
	f.mu.Lock()
	f.failed[src.name] = struct{}{}
	f.mu.Unlock()
}

func (f *sourceFailures) has(src matchViewSource) bool {
	if f == nil {
		return false
	}
	f.mu.Lock()
	defer f.mu.Unlock()
	_, ok := f.failed[src.name]
	return ok
}

// goLoad lance load() dans l'errgroup, comme le goLoad best-effort, mais l'échec est
// journalisé en ERREUR (source, match_id, err) AVANT la dégradation et retenu : la vue
// dira `<source>_unavailable` au lieu de laisser croire à une source vide. Jamais fatal.
// Un chargement coupé parce que la requête a été abandonnée n'est pas une panne de la
// source : il est retenu (la réponse n'est de toute façon pas servie) mais journalisé en
// WARN, pour ne pas polluer les journaux d'erreur à chaque navigation interrompue.
func (f *sourceFailures) goLoad(gctx context.Context, g *errgroup.Group, matchID string, src matchViewSource, load func() error) {
	g.Go(func() error {
		err := load()
		if err == nil {
			return nil
		}
		if errors.Is(err, context.Canceled) {
			slog.WarnContext(gctx, "match_view: chargement interrompu", "source", src.name, "match_id", matchID, "err", err)
		} else {
			slog.ErrorContext(gctx, "match_view: source indisponible", "source", src.name, "match_id", matchID, "err", err)
		}
		f.mark(src)
		return nil
	})
}

// matchPartialReasons compose les raisons partielles de la vue : pour chacune des
// quatre sources, `<source>_unavailable` si sa lecture a échoué, sinon l'éventuelle
// `<source>_empty` de detectPartialMatchData. Une source en échec n'émet JAMAIS aussi
// son `_empty` : la raison dit ce qui s'est passé, une seule fois.
func matchPartialReasons(d matchViewData) []string {
	empties := detectPartialMatchData(d.stats, d.scoreboard, d.events, d.medals)
	var reasons []string
	for _, src := range matchViewSources {
		if d.failures.has(src) {
			reasons = append(reasons, src.unavailable)
			continue
		}
		for _, e := range empties {
			if e == src.empty {
				reasons = append(reasons, e)
			}
		}
	}
	return reasons
}
