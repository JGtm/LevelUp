// Package scheduler — auto_sync_progress.go : la garde de progrès de la cadence de rattrapage.
//
// La cadence courte (catchUpCycleInterval) n'a de sens que si les cycles avancent. Un joueur en échec
// permanent (jeton mort, base illisible, liste qui échoue, film trop long en tête d'arriéré, match du haut
// qui échoue toujours) laisse sa dette ouverte à chaque cycle : sans garde, il tiendrait TOUS les joueurs à
// la cadence courte pour toujours. Un cycle a progressé quand au moins un de ces signaux est vrai :
//
//   - le marque-page d'un joueur a avancé, ou sa fin d'historique a été rangée (position lue dans sa base
//     avant et après le cycle, port.HistoryMarkerReader) ;
//   - au moins un film a été écrit sur un titre du cycle (cumul port.FilmBacklog.Ecrits, comparé avant et
//     après) ;
//   - un joueur resté en rattrapage a rangé au moins un match (compte du cycle ; couvre une position
//     illisible ou un lecteur non câblé).
//
// Après catchUpStallLimit cycles d'affilée sans progrès, la cadence revient à l'intervalle réglé (journalisé) ;
// un cycle qui progresse, une dette soldée ou un cycle demandé à la main remet le compteur à zéro.
package scheduler

import (
	"context"
	"log/slog"
	"time"

	"levelup/go-api/internal/domain"
	"levelup/go-api/internal/port"
)

// catchUpStallLimit est le nombre de cycles d'affilée sans progrès que la cadence courte tolère : au-delà,
// la boucle revient à l'intervalle réglé jusqu'au prochain progrès.
const catchUpStallLimit = 3

// progressWatch retient ce qui précède un cycle, pour le comparer à ce qui le suit.
type progressWatch struct {
	// markers : position de l'historique, par couple (titre, joueur), des joueurs dont elle a pu être lue
	// avant le cycle.
	markers map[string]port.HistoryMarker
	// films : films écrits (cumul) sur les titres du cycle avant lui.
	films int
}

// WithHistoryMarkerReader branche la lecture de la position de l'historique d'un joueur (marque-page et fin),
// dont la garde de progrès compare deux lectures autour d'un cycle. Nil : le progrès ne se lit que sur les
// films et les matchs rangés par un joueur en rattrapage.
func (s *AutoSyncScheduler) WithHistoryMarkerReader(r port.HistoryMarkerReader) *AutoSyncScheduler {
	s.markerReader = r
	return s
}

// watchProgress relève, avant un cycle, la position de l'historique des joueurs et le cumul de films écrits.
func (s *AutoSyncScheduler) watchProgress(ctx context.Context, players []domain.PlayerSummary) progressWatch {
	w := progressWatch{markers: map[string]port.HistoryMarker{}, films: s.filmsWrittenOf(players)}
	if s.markerReader == nil {
		return w
	}
	for _, p := range players {
		if m, ok := s.readMarker(ctx, p); ok {
			w.markers[passKey(resolveTitleSlug(p), p.Gamertag)] = m
		}
	}
	return w
}

// readMarker lit la position de l'historique d'un joueur ; une lecture en échec est journalisée et rend faux
// (aucun signal de progrès n'en sort).
func (s *AutoSyncScheduler) readMarker(ctx context.Context, p domain.PlayerSummary) (port.HistoryMarker, bool) {
	m, err := s.markerReader.ReadHistoryMarker(ctx, resolveTitleSlug(p), p.Gamertag)
	if err != nil {
		slog.WarnContext(ctx, "auto_sync: position de l'historique illisible — pas de signal de progrès pour ce joueur",
			"gamertag", p.Gamertag, "title_slug", resolveTitleSlug(p), "err", err)
		return port.HistoryMarker{}, false
	}
	return m, true
}

// progressedSince dit si le cycle qui vient de finir a fait avancer le rattrapage (signaux : voir l'en-tête du
// fichier). Seuls les joueurs dont l'historique était incomplet avant le cycle sont relus après lui.
func (s *AutoSyncScheduler) progressedSince(ctx context.Context, w progressWatch, players []domain.PlayerSummary, res *RunOnceResult) bool {
	if s.filmsWrittenOf(players) > w.films {
		return true
	}
	pending := make(map[string]bool, len(res.debt.history))
	for _, gt := range res.debt.history {
		pending[gt] = true
	}
	for _, p := range players {
		if pending[p.Gamertag] && res.inserted[p.Gamertag] > 0 {
			return true
		}
		before, ok := w.markers[passKey(resolveTitleSlug(p), p.Gamertag)]
		if !ok || before.Complete {
			continue
		}
		if after, ok := s.readMarker(ctx, p); ok && after != before {
			return true
		}
	}
	return false
}

// filmsWrittenOf rend le cumul de films écrits sur les titres des joueurs (chaque titre compté une fois).
// Sans lecteur d'arriéré câblé, zéro : les films ne comptent pas comme signal.
func (s *AutoSyncScheduler) filmsWrittenOf(players []domain.PlayerSummary) int {
	if s.filmBacklog == nil {
		return 0
	}
	seen := make(map[string]bool, 2)
	total := 0
	for _, p := range players {
		slug := resolveTitleSlug(p)
		if seen[slug] {
			continue
		}
		seen[slug] = true
		total += s.filmBacklog.FilmBacklog(slug).Ecrits
	}
	return total
}

// advanceStall met à jour le compteur de cycles sans progrès après un cycle et dit si la cadence courte
// tient encore. Une dette soldée ou un progrès remet le compteur à zéro ; la cadence courte tient tant que
// le compteur ne dépasse pas catchUpStallLimit. Le passage de la limite est journalisé une fois.
func (st *tickState) advanceStall(ctx context.Context, debt cycleDebt) (shortCadence bool) {
	if !debt.pending() || debt.progressed {
		st.stalled = 0
		return debt.pending()
	}
	st.stalled++
	if st.stalled > catchUpStallLimit {
		if st.stalled == catchUpStallLimit+1 && catchUpCycleInterval < st.configured {
			slog.WarnContext(ctx, "auto_sync: rattrapage sans progrès — retour à l'intervalle réglé jusqu'au prochain progrès",
				"cycles_sans_progres", st.stalled, "limite", catchUpStallLimit,
				"joueurs_en_rattrapage", len(debt.history), "gamertags", debt.history,
				"intervalle_regle", st.configured)
		}
		return false
	}
	return true
}

// addInserted verse au résultat les matchs insérés par gamertag d'une voie du cycle.
func (res *RunOnceResult) addInserted(counts map[string]int) {
	if len(counts) == 0 {
		return
	}
	if res.inserted == nil {
		res.inserted = make(map[string]int, len(counts))
	}
	for gamertag, n := range counts {
		res.inserted[gamertag] += n
	}
}

// insertedSince rend les matchs insérés par gamertag depuis start par les joueurs passés par syncPlayer
// (live-only et filet de boot, rangés par recordOutcome) ; les joueurs du pipeline V2 ont leur compte dans
// l'issue du cycle.
func (s *AutoSyncScheduler) insertedSince(start time.Time) map[string]int {
	s.snapshotMu.RLock()
	defer s.snapshotMu.RUnlock()
	out := map[string]int{}
	for gamertag, d := range s.playerOutcomes {
		if d.Outcome == outcomeName(outcomeOK) && d.MatchesInserted > 0 && !d.AttemptedAt.Before(start) {
			out[gamertag] = d.MatchesInserted
		}
	}
	return out
}
