package service

import (
	"context"
	"errors"
	"log/slog"
	"sync"

	"levelup/go-api/internal/domain"
	"levelup/go-api/internal/games"
	"levelup/go-api/internal/observability"
	"levelup/go-api/internal/port"

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
	// section : au moins un chargeur SANS raison dédiée a échoué (code générique).
	section bool
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

// reportLoadFailure journalise l'échec d'un chargeur AVANT la dégradation. Un chargement
// coupé parce que la requête a été abandonnée n'est pas une panne de la source : WARN (règle
// observability.LevelUnlessCanceled : seul l'abandon de la requête fait foi, un délai interne expiré reste une panne),
// pour ne pas polluer les journaux d'erreur à chaque navigation interrompue.
func reportLoadFailure(gctx context.Context, matchID, source string, err error) {
	if observability.LevelUnlessCanceled(gctx, err, slog.LevelError) != slog.LevelError {
		slog.WarnContext(gctx, "match_view: chargement interrompu", "source", source, "match_id", matchID, "err", err)
		return
	}
	slog.ErrorContext(gctx, "match_view: source indisponible", "source", source, "match_id", matchID, "err", err)
}

// goLoad lance load() dans l'errgroup pour une source qui porte sa propre raison
// (`<source>_unavailable`) : l'échec est journalisé en ERREUR (source, match_id, err) et
// retenu, la vue dira `<source>_unavailable` au lieu de laisser croire à une source vide.
// Une *domain.SectionDegradedError (donnée principale servie, section secondaire en panne)
// ne dit PAS que la source est en panne : elle rejoint le code générique. Jamais fatal.
func (f *sourceFailures) goLoad(gctx context.Context, g *errgroup.Group, matchID string, src matchViewSource, load func() error) {
	g.Go(func() error {
		err := load()
		if err == nil {
			return nil
		}
		if domain.IsSectionDegraded(err) {
			f.failSection(gctx, matchID, src.name, err)
			return nil
		}
		reportLoadFailure(gctx, matchID, src.name, err)
		f.mark(src)
		return nil
	})
}

// goLoadSection est le goLoad des chargeurs SANS raison dédiée : voir failSection.
func (f *sourceFailures) goLoadSection(gctx context.Context, g *errgroup.Group, matchID, label string, load func() error) {
	g.Go(func() error {
		f.failSection(gctx, matchID, label, load())
		return nil
	})
}

// failSection traite l'échec err d'un chargeur SANS raison dédiée (nil : rien) : journal ERREUR
// (la source en cause est dans le journal, pas dans le contrat) puis vue « en panne partielle »
// par le code générique `section_unavailable`. Une capability non supportée par le titre n'est
// pas une panne : absence attendue, la section reste vide.
func (f *sourceFailures) failSection(gctx context.Context, matchID, label string, err error) {
	if err == nil {
		return
	}
	if errors.Is(err, games.ErrCapabilityNotSupported) {
		slog.DebugContext(gctx, "match_view: capability non supportée", "source", label, "match_id", matchID)
		return
	}
	reportLoadFailure(gctx, matchID, label, err)
	f.mu.Lock()
	f.section = true
	f.mu.Unlock()
}

// loadCanonicalEvents charge les highlight events canoniques (loader unifié MV4.A) dans
// l'errgroup. Sans eventsRepo, canonicalEvents reste nil et les builders narrative retombent
// sur la conversion à la volée des events bruts. Un échec n'est plus tu : il est dit
// (`section_unavailable`) même si le repli sert la vue.
func (s *MatchViewService) loadCanonicalEvents(gctx context.Context, g *errgroup.Group, matchID string, d *matchViewData) {
	if s.eventsRepo == nil {
		return
	}
	g.Go(func() error {
		filters := port.HighlightEventFilters{MatchIDs: []string{matchID}}
		if err := filters.Validate(); err != nil {
			d.failures.failSection(gctx, matchID, "canonical_events", err)
			return nil
		}
		ev, err := s.eventsRepo.Load(gctx, filters)
		if err != nil {
			d.failures.failSection(gctx, matchID, "canonical_events", err)
			return nil
		}
		d.canonicalEvents = ev
		return nil
	})
}

func (f *sourceFailures) hasSection() bool {
	if f == nil {
		return false
	}
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.section
}

// matchPartialReasons compose les raisons partielles de la vue : pour chacune des
// quatre sources, `<source>_unavailable` si sa lecture a échoué, sinon l'éventuelle
// `<source>_empty` de detectPartialMatchData, puis `section_unavailable` si un autre
// chargeur a échoué. Une source en échec n'émet JAMAIS aussi son `_empty` : la raison dit
// ce qui s'est passé, une seule fois.
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
	if d.failures.hasSection() {
		reasons = append(reasons, domain.PartialReasonSectionUnavailable)
	}
	return reasons
}
