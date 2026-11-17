package replayregen

// demand.go — les demandes EXPLICITES de génération du rejeu d'un match qui n'en a pas (seconde entrée du paquet,
// après le lot de régénération). Une demande est ajoutée à la file par [Service.Enqueue], persistée à chaque
// changement, servie un film à la fois par demand_run.go.

import (
	"context"
	"log/slog"

	"levelup/go-api/internal/domain"
	"levelup/go-api/internal/domain/title"
	"levelup/go-api/internal/observability"
)

// Compteurs expvar des demandes explicites.
const (
	MetricGenQueued   = "replay_generation_queued_total"
	MetricGenDone     = "replay_generation_done_total"
	MetricGenFailed   = "replay_generation_failed_total"
	MetricGenFilmGone = "replay_generation_film_gone_total"
)

// demandEntry : une demande en file ou en cours.
type demandEntry struct {
	d domain.ReplayGenerationDemand
	// claimed : la demande a été prise par un travailleur (état `running`).
	claimed bool
	// phase : domain.ReplayGenPhaseDownloading puis ReplayGenPhaseBuilding ; vide tant que la demande attend.
	phase string
}

// demandKey : la clé d'une demande, indépendante de la forme (courte ou complète) de l'identifiant du match.
func demandKey(titleSlug, matchID string) string {
	return titleSlug + "/" + title.FilmShortMatchID(matchID)
}

// Enqueue met la demande en file. Les demandes que le fichier portait avant l'arrêt passent d'abord dans la file :
// l'écriture qui suit ne les écrase jamais, même si elle précède [Service.ResumeDemands]. created=false : une demande du même match est déjà en file ou en cours (rend son
// état). Un échec récent du même match est remplacé par la nouvelle demande. Le plafond est vérifié sous le verrou,
// dans le même geste que l'ajout.
func (s *Service) Enqueue(d domain.ReplayGenerationDemand) (domain.ReplayGenerationStatus, bool, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if !s.bound {
		return domain.ReplayGenerationStatus{}, false, domain.ErrReplayGenerationUnbound
	}
	s.adoptPersistedDemandsLocked(s.ctx)
	key := demandKey(d.TitleSlug, d.MatchID)
	if e := s.findDemandLocked(key); e != nil {
		return s.demandStatusLocked(e), false, nil
	}
	if len(s.demands) >= s.deps.maxDemands() {
		slog.WarnContext(s.ctx, "replay_generation: file pleine, demande refusée",
			"titleSlug", d.TitleSlug, "match_id", d.MatchID, "player", d.PlayerSlug, "max", s.deps.maxDemands())
		return domain.ReplayGenerationStatus{}, false, domain.ErrReplayGenerationQueueFull
	}
	s.forgetResultLocked(key)
	s.demands = append(s.demands, &demandEntry{d: d})
	s.persistDemandsLocked(s.ctx)
	observability.IncCounter(MetricGenQueued)
	slog.InfoContext(s.ctx, "replay_generation: demande en file",
		"titleSlug", d.TitleSlug, "match_id", d.MatchID, "player", d.PlayerSlug, "position", len(s.demands))
	st := s.demandStatusLocked(s.demands[len(s.demands)-1])
	s.kickDemandLoopLocked()
	return st, true, nil
}

// DemandStatus rend l'état connu de la file pour ce match : en file, en cours, ou l'issue terminale (échec, film
// indisponible, non pris en charge) gardée dans ce processus. ok=false : la file n'en sait rien.
func (s *Service) DemandStatus(titleSlug, matchID string) (domain.ReplayGenerationStatus, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	key := demandKey(titleSlug, matchID)
	if e := s.findDemandLocked(key); e != nil {
		return s.demandStatusLocked(e), true
	}
	if st, ok := s.results[key]; ok {
		return st, true
	}
	return domain.ReplayGenerationStatus{}, false
}

func (s *Service) findDemandLocked(key string) *demandEntry {
	for _, e := range s.demands {
		if demandKey(e.d.TitleSlug, e.d.MatchID) == key {
			return e
		}
	}
	return nil
}

// demandStatusLocked : l'état servi d'une demande en vol. La pause et l'attente de la partie sont celles de la file
// (le lot en pause retient aussi les demandes).
func (s *Service) demandStatusLocked(e *demandEntry) domain.ReplayGenerationStatus {
	st := domain.ReplayGenerationStatus{
		State:          domain.ReplayGenerationQueued,
		WaitingForGame: s.waitingGame,
		Paused:         s.running && s.userPaused,
	}
	if e.claimed {
		st.State, st.Phase = domain.ReplayGenerationRunning, e.phase
		return st
	}
	for _, o := range s.demands {
		if o.claimed {
			continue
		}
		st.Position++
		if o == e {
			break
		}
	}
	return st
}

// queuedDemandLocked : une demande attend un travailleur.
func (s *Service) queuedDemandLocked() bool {
	for _, e := range s.demands {
		if !e.claimed {
			return true
		}
	}
	return false
}

// persistDemandsLocked écrit les demandes en vol ; s.mu est tenu (petit fichier, comme [Service.persistPendingLocked]).
// Un échec d'écriture se journalise : la file mémoire reste juste, seule la reprise après arrêt en pâtit.
func (s *Service) persistDemandsLocked(ctx context.Context) {
	if s.deps.Demands == nil {
		return
	}
	out := make([]domain.ReplayGenerationDemand, len(s.demands))
	for i, e := range s.demands {
		out[i] = e.d
	}
	if err := s.deps.Demands.SaveDemands(ctx, out); err != nil {
		slog.WarnContext(ctx, "replay_generation: demandes en vol non écrites, reprise après arrêt compromise",
			"demands", len(out), "err", err)
	}
}

// removeDemandLocked sort une demande de la file et réécrit le fichier.
func (s *Service) removeDemandLocked(ctx context.Context, e *demandEntry) {
	for i, o := range s.demands {
		if o == e {
			s.demands = append(s.demands[:i], s.demands[i+1:]...)
			break
		}
	}
	s.persistDemandsLocked(ctx)
}

// rememberResultLocked garde l'issue terminale d'une demande ; au-delà de [maxResults], la plus ancienne sort.
func (s *Service) rememberResultLocked(key string, st domain.ReplayGenerationStatus) {
	if _, ok := s.results[key]; !ok {
		s.resultOrder = append(s.resultOrder, key)
	}
	s.results[key] = st
	for len(s.resultOrder) > maxResults {
		delete(s.results, s.resultOrder[0])
		s.resultOrder = s.resultOrder[1:]
	}
}

// forgetResultLocked oublie l'issue gardée d'un match (une nouvelle demande la remplace).
func (s *Service) forgetResultLocked(key string) {
	if _, ok := s.results[key]; !ok {
		return
	}
	delete(s.results, key)
	for i, k := range s.resultOrder {
		if k == key {
			s.resultOrder = append(s.resultOrder[:i], s.resultOrder[i+1:]...)
			break
		}
	}
}
