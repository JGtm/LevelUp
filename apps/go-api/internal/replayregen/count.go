package replayregen

// count.go — le décompte des rejeux en retard (sous la porte du mode jeu) et l'ensemble des matchs écartés dans ce
// processus. Aucune logique propre : le découpage de service.go, que le seuil de taille de fichier imposait.

import (
	"context"
	"log/slog"
	"time"

	"levelup/go-api/internal/observability"
)

func (s *Service) backgroundScan(seq int64) {
	defer s.wg.Done()
	cands, err := s.scanUnderGate(s.ctx)
	s.mu.Lock()
	defer s.mu.Unlock()
	s.scanning = false
	if err != nil {
		slog.WarnContext(s.ctx, "replay_regen: décompte des rejeux en retard interrompu", "err", err)
		return
	}
	s.publishCountLocked(seq, len(cands))
}

// scanUnderGate : le décompte sous la porte du mode jeu.
func (s *Service) scanUnderGate(ctx context.Context) ([]Candidate, error) {
	if !s.waitHeavy(ctx) {
		return nil, ctx.Err()
	}
	return s.scan(ctx)
}

// publishCountLocked publie un décompte, sauf s'il est plus ancien que le dernier publié.
func (s *Service) publishCountLocked(seq int64, n int) {
	if seq < s.countApplied {
		slog.Debug("replay_regen: décompte périmé jeté", "seq", seq, "publie", s.countApplied)
		return
	}
	s.countApplied = seq
	s.checked, s.stale = true, n
	observability.SetInt(MetricStale, int64(n))
}

// waitHeavy attend la porte du mode jeu (ni la pause ni la synchronisation : un décompte ne décode rien). Rend
// faux si ctx est annulé en attendant.
func (s *Service) waitHeavy(ctx context.Context) bool {
	for {
		if ctx.Err() != nil {
			return false
		}
		if s.deps.HeavyAllowed == nil || s.deps.HeavyAllowed() {
			return true
		}
		select {
		case <-ctx.Done():
			return false
		case <-time.After(s.deps.gateEvery()):
		}
	}
}

// scan liste les rejeux régénérables de tous les titres, le plus récent d'abord.
func (s *Service) scan(ctx context.Context) ([]Candidate, error) {
	s.mu.Lock()
	d := s.deps
	s.mu.Unlock()
	var all []Candidate
	for _, slug := range d.Titles() {
		skip := s.skipFor(ctx, d, slug)
		c, err := scanTitle(ctx, d.Paths.ReplayArtifactsDir(slug), slug, d.FilmPresent, skip)
		if err != nil {
			return nil, err
		}
		all = append(all, c...)
	}
	sortRecentFirst(all)
	return all, nil
}

// skipFor rend le test « ce match n'est pas à régénérer » d'un titre : refusé dans ce processus (isBlocked) ou refusé par
// règle de cuisson (Deps.RefusedByRule, lu UNE fois par décompte et par titre).
func (s *Service) skipFor(ctx context.Context, d Deps, slug string) func(slug, matchID string) bool {
	var refused map[string]struct{}
	if d.RefusedByRule != nil {
		var err error
		if refused, err = d.RefusedByRule(ctx, slug); err != nil {
			slog.WarnContext(ctx, "replay_regen: refus par règle illisibles, ils seront écartés au premier essai",
				"titleSlug", slug, "err", err)
			refused = nil
		}
	}
	return func(slug, matchID string) bool {
		if _, no := refused[matchID]; no {
			return true
		}
		return s.isBlocked(slug, matchID)
	}
}

func blockKey(slug, matchID string) string { return slug + "/" + matchID }

func (s *Service) isBlocked(slug, matchID string) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	_, ok := s.blocked[blockKey(slug, matchID)]
	return ok
}

func (s *Service) block(slug, matchID string) {
	s.mu.Lock()
	s.blocked[blockKey(slug, matchID)] = struct{}{}
	s.mu.Unlock()
}
