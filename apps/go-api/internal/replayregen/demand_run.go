package replayregen

// demand_run.go — le service des demandes explicites : la boucle hors lot, l'entrée de la boucle du lot, le
// traitement d'une demande (téléchargement du film, reconstruction), le classement de l'issue et la reprise après
// arrêt. UN FILM À LA FOIS : le jeton [Service.slot] est tenu pendant toute demande et tout film du lot.

import (
	"context"
	"errors"
	"log/slog"

	"levelup/go-api/internal/domain"
	"levelup/go-api/internal/observability"
	"levelup/go-api/internal/replaybuild"

	"github.com/JGtm/halo-infinite-film-decoder/finalise"
)

// kickDemandLoopLocked lance la boucle des demandes quand aucune ne tourne, qu'aucun lot n'est actif (le lot sert
// lui-même les demandes avant chaque film, [Service.serveDemands]) et qu'une demande attend ; s.mu est tenu.
func (s *Service) kickDemandLoopLocked() {
	if s.demandLooping || s.running || !s.queuedDemandLocked() || s.ctx == nil || s.ctx.Err() != nil {
		return
	}
	s.demandLooping = true
	s.wg.Add(1)
	go s.demandLoop(s.ctx)
}

// demandLoop sert les demandes hors lot, une à la fois ; elle sort quand plus rien n'attend, quand un lot démarre
// (il prend le relais) ou à l'arrêt.
func (s *Service) demandLoop(ctx context.Context) {
	defer s.wg.Done()
	for {
		s.mu.Lock()
		if s.running || ctx.Err() != nil || !s.queuedDemandLocked() {
			s.demandLooping = false
			s.mu.Unlock()
			return
		}
		s.mu.Unlock()
		s.serveNext(ctx)
	}
}

// serveDemands sert toutes les demandes en attente, avant le film suivant du lot. Faux : ctx annulé.
func (s *Service) serveDemands(ctx context.Context) bool {
	for s.serveNext(ctx) {
	}
	return ctx.Err() == nil
}

// serveNext sert UNE demande : jeton du film, attente de son tour (pause, partie, synchronisation), prise de la
// demande, traitement. Faux : plus de demande en attente, ou ctx annulé.
func (s *Service) serveNext(ctx context.Context) bool {
	s.slot.Lock()
	defer s.slot.Unlock()
	s.mu.Lock()
	queued := s.queuedDemandLocked()
	s.mu.Unlock()
	if !queued || !s.waitTurn(ctx) {
		return false
	}
	e := s.claimDemand()
	if e == nil {
		return false
	}
	s.processDemand(ctx, e)
	return ctx.Err() == nil
}

// claimDemand prend la première demande en attente.
func (s *Service) claimDemand() *demandEntry {
	s.mu.Lock()
	defer s.mu.Unlock()
	for _, e := range s.demands {
		if !e.claimed {
			e.claimed, e.phase = true, domain.ReplayGenPhaseDownloading
			return e
		}
	}
	return nil
}

func (s *Service) setDemandPhase(e *demandEntry, phase string) {
	s.mu.Lock()
	e.phase = phase
	s.mu.Unlock()
}

// processDemand traite UNE demande : un rejeu déjà à jour sort en `ready` sans rien lancer ; sinon le film se
// télécharge (phase `downloading`), puis le rejeu se reconstruit (phase `building`) après un nouveau tour de porte.
func (s *Service) processDemand(ctx context.Context, e *demandEntry) {
	d := e.d
	if replaybuild.ArtifactUpToDate(s.deps.Paths.ReplayArtifactPath(d.TitleSlug, d.MatchID)) {
		s.settleDemand(ctx, e, nil, "")
		return
	}
	phase := domain.ReplayGenPhaseDownloading
	var err error
	if s.deps.FetchFilm != nil {
		err = s.deps.FetchFilm(ctx, d)
	}
	if err == nil {
		phase = domain.ReplayGenPhaseBuilding
		s.setDemandPhase(e, phase)
		err = s.buildDemand(ctx, d)
	}
	s.settleDemand(ctx, e, err, phase)
}

// buildDemand reconstruit le rejeu d'une demande dont le film est à l'archive, puis retire le film si
// l'archive est désactivée ([Deps.ReleaseFilm]), après un succès seulement.
func (s *Service) buildDemand(ctx context.Context, d domain.ReplayGenerationDemand) error {
	if !s.waitTurn(ctx) {
		return ctx.Err()
	}
	if err := s.rebuildWithRetry(ctx, Candidate{TitleSlug: d.TitleSlug, MatchID: d.MatchID}); err != nil {
		return err
	}
	if s.deps.ReleaseFilm != nil {
		s.deps.ReleaseFilm(ctx, d.TitleSlug, d.MatchID)
	}
	return nil
}

// settleDemand clôt une demande : sortie de la file (fichier réécrit), issue gardée, compteur et journal. Un arrêt du
// serveur ne clôt rien : la demande reste au fichier et recommence à l'ouverture suivante.
func (s *Service) settleDemand(ctx context.Context, e *demandEntry, err error, failedIn string) {
	d := e.d
	if err != nil && ctx.Err() != nil {
		s.mu.Lock()
		e.claimed, e.phase = false, ""
		s.mu.Unlock()
		slog.InfoContext(ctx, "replay_generation: demande interrompue par l'arrêt, reprise à l'ouverture suivante",
			"titleSlug", d.TitleSlug, "match_id", d.MatchID, "player", d.PlayerSlug)
		return
	}
	st := classifyDemand(err, failedIn)
	s.mu.Lock()
	s.removeDemandLocked(ctx, e)
	if st.State != domain.ReplayGenerationReady {
		s.rememberResultLocked(demandKey(d.TitleSlug, d.MatchID), st)
	}
	s.mu.Unlock()
	reportDemand(ctx, d, st, err)
}

// reportDemand compte et journalise l'issue d'une demande.
func reportDemand(ctx context.Context, d domain.ReplayGenerationDemand, st domain.ReplayGenerationStatus, err error) {
	attrs := []any{"titleSlug", d.TitleSlug, "match_id", d.MatchID, "player", d.PlayerSlug, "state", st.State, "reason", st.Reason}
	switch st.State {
	case domain.ReplayGenerationReady:
		observability.IncCounter(MetricGenDone)
		slog.InfoContext(ctx, "replay_generation: rejeu généré", attrs...)
	case domain.ReplayGenerationFilmUnavailable:
		observability.IncCounter(MetricGenFilmGone)
		slog.WarnContext(ctx, "replay_generation: film plus disponible chez l'éditeur", append(attrs, "err", err)...)
	case domain.ReplayGenerationNotSupported:
		slog.InfoContext(ctx, "replay_generation: match écarté par la cuisson", append(attrs, "err", err)...)
	default:
		observability.IncCounter(MetricGenFailed)
		slog.WarnContext(ctx, "replay_generation: rejeu non généré", append(attrs, "retryable", st.Retryable, "err", err)...)
	}
}

// classifyDemand classe l'erreur d'une demande en état servi. failedIn : la phase où l'erreur est survenue (elle
// départage un échec de téléchargement d'un échec de construction).
func classifyDemand(err error, failedIn string) domain.ReplayGenerationStatus {
	failed := func(reason string, retry bool) domain.ReplayGenerationStatus {
		return domain.ReplayGenerationStatus{State: domain.ReplayGenerationFailed, Reason: reason, Retryable: retry}
	}
	switch {
	case err == nil:
		return domain.ReplayGenerationStatus{State: domain.ReplayGenerationReady}
	case errors.Is(err, domain.ErrReplayFilmGone):
		return domain.ReplayGenerationStatus{State: domain.ReplayGenerationFilmUnavailable}
	case errors.Is(err, domain.ErrReplayNoToken):
		return failed(domain.ReplayGenReasonTokenUnavailable, true)
	case errors.Is(err, finalise.ErrFilmNonFinalise):
		return failed(domain.ReplayGenReasonFilmNotReady, true)
	case errors.Is(err, replaybuild.ErrMapNotInCatalog):
		return failed(domain.ReplayGenReasonMapNotInCatalog, false)
	case errors.Is(err, replaybuild.ErrUnknownFilmKey):
		return failed(domain.ReplayGenReasonUnknownFilmKey, false)
	case errors.Is(err, replaybuild.ErrModeNonGere):
		return domain.ReplayGenerationStatus{State: domain.ReplayGenerationNotSupported, Reason: domain.ReplayGenReasonModeUnmanaged}
	case failedIn == domain.ReplayGenPhaseDownloading:
		return failed(domain.ReplayGenReasonFetchFailed, true)
	default:
		return failed(domain.ReplayGenReasonBuildFailed, true)
	}
}

// ResumeDemands relit les demandes persistées à l'ouverture, les remet en file dans l'ordre et lance la boucle. Un
// fichier illisible se journalise et laisse la file vide ; une demande incomplète est écartée.
func (s *Service) ResumeDemands() {
	s.mu.Lock()
	bound, store, ctx := s.bound, s.deps.Demands, s.ctx
	s.mu.Unlock()
	if !bound || store == nil {
		return
	}
	loaded, err := store.LoadDemands(ctx)
	if err != nil {
		slog.WarnContext(ctx, "replay_generation: demandes en vol illisibles, file vide", "err", err)
		return
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	kept := 0
	for _, d := range loaded {
		if d.TitleSlug == "" || d.MatchID == "" || d.XUID == "" {
			slog.WarnContext(ctx, "replay_generation: demande persistée incomplète, écartée",
				"titleSlug", d.TitleSlug, "match_id", d.MatchID, "player", d.PlayerSlug)
			continue
		}
		if s.findDemandLocked(demandKey(d.TitleSlug, d.MatchID)) != nil {
			continue
		}
		s.demands = append(s.demands, &demandEntry{d: d})
		kept++
	}
	if kept != len(loaded) {
		s.persistDemandsLocked(ctx)
	}
	if kept > 0 {
		slog.InfoContext(ctx, "replay_generation: reprise des demandes lancées avant l'arrêt", "demands", kept)
	}
	s.kickDemandLoopLocked()
}
